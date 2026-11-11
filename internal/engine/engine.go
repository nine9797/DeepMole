package engine

import (
	"context"
	"deepmole/internal/i18n"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"deepmole/internal/netid"
	"deepmole/internal/probe"
	"deepmole/internal/runner"
	"deepmole/internal/sdns"
	"deepmole/internal/state"
	"deepmole/internal/strategy"
)

type Config struct {
	ZapretDir        string
	Game             string
	ProbeTimeout     time.Duration
	WatchdogInterval time.Duration
	MaxCandidates    int
	StatePath        string
	SecureDNS        string
	DNSBackup        string
	Log              *log.Logger
}

type Phase string

const (
	PhaseIdle      Phase = "idle"
	PhaseSearching Phase = "searching"
	PhaseRunning   Phase = "running"
	PhasePartial   Phase = "partial"
	PhaseFailed    Phase = "failed"
	PhaseOffline   Phase = "offline"
	PhaseDisabled  Phase = "disabled"
)

type Status struct {
	Phase     Phase
	Network   string
	Plan      strategy.Plan
	Blocks    map[string]bool
	Progress  string
	Message   string
	Enabled   bool
	DNS       string
	SecureDNS bool
}

const DNSBlocked = "blocked"

func (s Status) Text() string {
	switch s.Phase {
	case PhaseSearching:
		return i18n.T("status.searching", s.Progress)
	case PhaseRunning:
		return i18n.T("status.running")
	case PhasePartial:
		return i18n.T("status.partial")
	case PhaseFailed:
		return i18n.T("status.failed")
	case PhaseOffline:
		return i18n.T("status.offline")
	case PhaseDisabled:
		return i18n.T("status.disabled")
	}
	return i18n.T("status.starting")
}

type Engine struct {
	cfg     Config
	paths   strategy.Paths
	run     *runner.Runner
	st      *state.State
	targets []probe.Target
	log     *log.Logger

	mu       sync.Mutex
	status   Status
	net      netid.Network
	applied  *strategy.Plan
	failures int
	dns      string
	dnsOff   bool
	cmds     chan func(context.Context)

	OnChange func(Status)
	OnEvent  func(string)
}

var blocks = []string{probe.BlockGoogle, probe.BlockGeneral}

var blockTitle = map[string]string{probe.BlockGoogle: "YouTube", probe.BlockGeneral: "Discord"}

func New(cfg Config) (*Engine, error) {
	bin := filepath.Join(cfg.ZapretDir, "bin") + string(filepath.Separator)
	lists := filepath.Join(cfg.ZapretDir, "lists") + string(filepath.Separator)
	exe := bin + "winws.exe"
	if _, err := os.Stat(exe); err != nil {
		return nil, fmt.Errorf("winws.exe not found: %s", exe)
	}
	for _, f := range []string{"list-general.txt", "list-google.txt", "list-exclude.txt", "ipset-all.txt", "ipset-exclude.txt"} {
		if _, err := os.Stat(lists + f); err != nil {
			return nil, fmt.Errorf("list missing: %s", lists+f)
		}
	}
	ensureUserLists(lists)
	if cfg.Log == nil {
		cfg.Log = log.Default()
	}
	if cfg.ProbeTimeout == 0 {
		cfg.ProbeTimeout = 3 * time.Second
	}
	if cfg.WatchdogInterval == 0 {
		cfg.WatchdogInterval = 60 * time.Second
	}
	if cfg.MaxCandidates == 0 {
		cfg.MaxCandidates = 30
	}
	st := state.Load(cfg.StatePath)
	mode := cfg.SecureDNS
	if v := st.GetSecureDNS(); v != "" {
		mode = v
	}
	dnsOff := strings.EqualFold(mode, "off") || cfg.DNSBackup == ""
	return &Engine{
		cfg:     cfg,
		paths:   strategy.NewPaths(bin, lists),
		run:     &runner.Runner{Exe: exe, Dir: bin},
		st:      st,
		targets: probe.Default(),
		log:     cfg.Log,
		status:  Status{Phase: PhaseIdle, Blocks: map[string]bool{}, Enabled: true, SecureDNS: !dnsOff},
		dnsOff:  dnsOff,
		cmds:    make(chan func(context.Context), 4),
	}, nil
}

func (e *Engine) SetSecureDNS(on bool) {
	e.post(func(ctx context.Context) {
		if on {
			e.st.SetSecureDNS("auto")
			e.dnsOff = e.cfg.DNSBackup == ""
			e.set(func(s *Status) { s.SecureDNS = true; s.DNS = "" })
			e.emit(i18n.T("ev.dns_auto"))
			if st := e.Status(); st.Enabled && (st.Phase == PhaseRunning || st.Phase == PhasePartial || st.Phase == PhaseFailed) {
				e.probe(ctx, e.targets)
			}
			return
		}
		e.st.SetSecureDNS("off")
		e.dnsOff = true
		e.dnsRestore()
		e.set(func(s *Status) { s.SecureDNS = false; s.DNS = "" })
		e.emit(i18n.T("ev.dns_manual"))
	})
}

func ensureUserLists(lists string) {
	defaults := map[string]string{
		"list-general-user.txt":  "domain.example.abc\n",
		"list-exclude-user.txt":  "domain.example.abc\n",
		"ipset-exclude-user.txt": "203.0.113.113/32\n",
	}
	for name, body := range defaults {
		p := lists + name
		if _, err := os.Stat(p); err != nil {
			_ = os.WriteFile(p, []byte(body), 0o644)
		}
	}
}

func (e *Engine) Status() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.status
}

func (e *Engine) set(mut func(s *Status)) {
	e.mu.Lock()
	mut(&e.status)
	s := e.status
	cb := e.OnChange
	e.mu.Unlock()
	if cb != nil {
		cb(s)
	}
}

func (e *Engine) emit(msg string) {
	e.log.Printf("%s", msg)
	e.mu.Lock()
	cb := e.OnEvent
	e.mu.Unlock()
	if cb != nil {
		cb(msg)
	}
}

func (e *Engine) post(fn func(context.Context)) {
	select {
	case e.cmds <- fn:
	default:
	}
}

func (e *Engine) Research() {
	e.post(func(ctx context.Context) {
		if !e.Status().Enabled {
			return
		}
		e.emit(i18n.T("ev.research"))
		e.search(ctx, e.applied)
	})
}

func (e *Engine) SetEnabled(on bool) {
	e.post(func(ctx context.Context) {
		if on {
			e.set(func(s *Status) { s.Enabled = true })
			e.emit(i18n.T("ev.enabled"))
			e.setup(ctx)
			return
		}
		e.run.Stop()
		e.applied = nil
		e.dnsRestore()
		e.set(func(s *Status) {
			s.Enabled = false
			s.Phase = PhaseDisabled
			s.Message = ""
			s.Blocks = map[string]bool{}
		})
		e.emit(i18n.T("ev.disabled"))
	})
}

func (e *Engine) Stop() {
	e.run.Stop()
	runner.KillStray()
	e.dnsRestore()
}

func (e *Engine) dnsFix(host string) {
	if e.dns != "" {
		return
	}
	if e.dnsOff {
		if e.Status().DNS != DNSBlocked {
			e.set(func(s *Status) { s.DNS = DNSBlocked })
			e.emit(i18n.T("ev.dns_blocked_off", host))
		}
		return
	}
	name, secure := probe.Fallback()
	p := sdns.Find(name)
	label := p.Name
	if secure {
		label += " DoH"
	}
	e.emit(i18n.T("ev.dns_blocked", host, label))
	if err := sdns.Apply(p, e.cfg.DNSBackup, secure); err != nil {
		e.dnsOff = true
		e.set(func(s *Status) { s.DNS = DNSBlocked })
		e.emit(i18n.T("ev.dns_fail", errLine(err.Error())))
		return
	}
	e.dns = label
	probe.ResetCache()
	e.set(func(s *Status) { s.DNS = label })
	e.emit(i18n.T("ev.dns_on", label))
}

func (e *Engine) dnsRestore() {
	if e.dns == "" {
		return
	}
	e.dns = ""
	if err := sdns.Restore(e.cfg.DNSBackup); err != nil {
		e.emit(i18n.T("ev.dns_restore_fail", errLine(err.Error())))
	} else {
		e.emit(i18n.T("ev.dns_off"))
	}
	probe.ResetCache()
	e.set(func(s *Status) { s.DNS = "" })
}

func (e *Engine) Run(ctx context.Context) {
	if e.cfg.DNSBackup != "" && sdns.Pending(e.cfg.DNSBackup) {
		if err := sdns.Restore(e.cfg.DNSBackup); err != nil {
			e.emit(i18n.T("ev.dns_restore_fail", errLine(err.Error())))
		} else {
			e.emit(i18n.T("ev.dns_restored"))
		}
	}
	e.net = netid.Current()
	e.setup(ctx)

	t := time.NewTicker(e.cfg.WatchdogInterval)
	defer t.Stop()
	netTick := time.NewTicker(10 * time.Second)
	defer netTick.Stop()
	retryAt := time.Time{}

	for {
		select {
		case <-ctx.Done():
			e.run.Stop()
			return
		case fn := <-e.cmds:
			fn(ctx)
		case <-netTick.C:
			n := netid.Current()
			if n.ID != e.net.ID {
				e.emit(i18n.T("ev.net_changed", n.Name))
				e.net = n
				e.failures = 0
				if e.Status().Enabled {
					e.setup(ctx)
				} else {
					e.set(func(s *Status) { s.Network = n.Name })
				}
			}
		case <-t.C:
			st := e.Status()
			if !st.Enabled {
				continue
			}
			switch st.Phase {
			case PhaseRunning, PhasePartial:
				e.watch(ctx)
			case PhaseFailed, PhaseOffline:
				if time.Now().After(retryAt) {
					retryAt = time.Now().Add(5 * time.Minute)
					e.setup(ctx)
				}
			}
		}
	}
}

func (e *Engine) setup(ctx context.Context) {
	e.set(func(s *Status) { s.Network = e.net.Name })
	if e.net.ID == "offline" {
		e.run.Stop()
		e.applied = nil
		e.set(func(s *Status) { s.Phase = PhaseOffline })
		e.emit(i18n.T("ev.net_missing"))
		return
	}
	if entry, ok := e.st.Get(e.net.ID); ok {
		e.emit(i18n.T("ev.cached_check", e.net.Name))
		if err := e.apply(entry.Plan); err == nil {
			sc := e.probe(ctx, e.targets)
			if sc[probe.BlockControl].OK == 0 {
				e.set(func(s *Status) { s.Phase = PhaseOffline; s.Message = i18n.T("msg.internet_down") })
				e.emit(i18n.T("ev.internet"))
				return
			}
			if sc[probe.BlockGoogle].Full() && sc[probe.BlockGeneral].Full() {
				e.st.Touch(e.net.ID)
				e.finish(entry.Plan, sc)
				e.emit(i18n.T("ev.cached_ok"))
				return
			}
			e.emit(i18n.T("ev.cached_fail", scoreText(sc)))
		} else {
			e.emit(i18n.T("ev.start_fail", errLine(err.Error())))
		}
		p := entry.Plan
		e.search(ctx, &p)
		return
	}
	e.emit(i18n.T("ev.net_new", e.net.Name))
	e.search(ctx, nil)
}

func (e *Engine) apply(plan strategy.Plan) error {
	args := strategy.BuildArgs(e.paths, plan, e.cfg.Game)
	if err := e.run.Start(args); err != nil {
		return err
	}
	e.applied = &plan
	return nil
}

func (e *Engine) probe(ctx context.Context, targets []probe.Target) map[string]probe.Score {
	res := probe.Run(ctx, targets, e.cfg.ProbeTimeout)
	sc := probe.Summarize(res)
	var retry []probe.Target
	for _, r := range res {
		s := sc[r.Target.Block]
		if !r.OK && s.OK > 0 && s.OK < s.Total {
			retry = append(retry, r.Target)
		}
	}
	if len(retry) > 0 {
		for _, r := range probe.Run(ctx, retry, e.cfg.ProbeTimeout) {
			if r.OK {
				s := sc[r.Target.Block]
				s.OK++
				sc[r.Target.Block] = s
			}
		}
	}
	dohHost := ""
	for _, r := range res {
		mark := "ok"
		if !r.OK {
			mark = "fail " + r.Err
		}
		if r.DoH {
			mark += " (doh)"
			if dohHost == "" && r.Target.Block != probe.BlockControl {
				dohHost = r.Host()
			}
		}
		e.log.Printf("  %-8s %-55s %6dB %5dms %s", r.Target.Block, r.Target.URL, r.Bytes, r.Millis, mark)
	}
	if dohHost != "" {
		e.dnsFix(dohHost)
	}
	return sc
}

func scoreText(sc map[string]probe.Score) string {
	var parts []string
	for _, b := range blocks {
		s := sc[b]
		mark := "✗"
		if s.Full() {
			mark = "✓"
		}
		parts = append(parts, fmt.Sprintf("%s %d/%d %s", blockTitle[b], s.OK, s.Total, mark))
	}
	return strings.Join(parts, ", ")
}

type cursor struct {
	queue []strategy.Params
	seeds []strategy.Params
	tried map[string]bool
	skip  func(strategy.Params) bool
	cur   strategy.Params
	best  strategy.Params
	score float64
	done  bool
	deep  bool
}

const fallbackSeeds = 3

func newCursor(seed *strategy.Params, skip func(strategy.Params) bool) *cursor {
	c := &cursor{tried: map[string]bool{}, skip: skip}
	if seed != nil && !seed.IsZero() {
		c.queue = append(c.queue, *seed)
	}
	c.queue = append(c.queue, strategy.Proven...)
	c.seeds = append(c.seeds, strategy.Proven[:fallbackSeeds]...)
	return c
}

func (c *cursor) next() bool {
	for {
		for len(c.queue) > 0 {
			p := c.queue[0]
			c.queue = c.queue[1:]
			if c.tried[p.Key()] {
				continue
			}
			c.tried[p.Key()] = true
			if c.skip != nil && c.skip(p) {
				continue
			}
			c.cur = p
			return true
		}
		c.deep = true
		src := c.best
		if src.IsZero() {
			if len(c.seeds) == 0 {
				return false
			}
			src, c.seeds = c.seeds[0], c.seeds[1:]
		}
		fresh := false
		for _, p := range strategy.Neighbors(src) {
			if !c.tried[p.Key()] {
				fresh = true
				c.queue = append(c.queue, p)
			}
		}
		if !fresh && !c.best.IsZero() {
			return false
		}
	}
}

func (c *cursor) pick() strategy.Params {
	if c.done {
		if c.best.IsZero() {
			return strategy.Proven[0]
		}
		return c.best
	}
	return c.cur
}

func (e *Engine) search(ctx context.Context, seed *strategy.Plan) {
	e.failures = 0
	e.set(func(s *Status) {
		s.Phase = PhaseSearching
		s.Progress = ""
		s.Message = ""
		s.Blocks = map[string]bool{}
	})
	started := time.Now()

	cs := map[string]*cursor{}
	var sg, sn *strategy.Params
	if seed != nil {
		sg, sn = &seed.Google, &seed.General
	}
	skip := func(p strategy.Params) bool { return !p.Available(e.paths) }
	cs[probe.BlockGoogle] = newCursor(sg, skip)
	cs[probe.BlockGeneral] = newCursor(sn, skip)

	limit := len(strategy.Proven) + e.cfg.MaxCandidates
	if seed != nil {
		limit++
	}
	attempts, deep, startFails := 0, 0, 0
	for {
		if ctx.Err() != nil {
			return
		}
		pending, isDeep := 0, false
		for _, b := range blocks {
			c := cs[b]
			if c.done {
				continue
			}
			if c.next() {
				pending++
				isDeep = isDeep || c.deep
			} else {
				c.done = true
			}
		}
		if pending == 0 {
			break
		}
		if isDeep {
			deep++
			if deep > e.cfg.MaxCandidates {
				break
			}
		}
		plan := strategy.Plan{Google: cs[probe.BlockGoogle].pick(), General: cs[probe.BlockGeneral].pick()}
		attempts++
		e.set(func(s *Status) { s.Progress = fmt.Sprintf("%d/%d", attempts, limit) })
		e.log.Printf("[%d] google=[%s] general=[%s]", attempts, plan.Google.Name(), plan.General.Name())
		if err := e.apply(plan); err != nil {
			startFails++
			e.emit(i18n.T("ev.cand_fail", attempts, errLine(err.Error())))
			if startFails >= 3 {
				e.emit(i18n.T("ev.start_abort"))
				e.set(func(s *Status) {
					s.Phase = PhaseFailed
					s.Progress = ""
					s.Message = errLine(err.Error())
				})
				return
			}
			continue
		}
		startFails = 0
		sc := e.probe(ctx, e.targets)
		if sc[probe.BlockControl].OK == 0 {
			e.set(func(s *Status) { s.Phase = PhaseOffline; s.Message = i18n.T("msg.internet_down") })
			e.emit(i18n.T("ev.control_down"))
			return
		}
		var parts []string
		for _, b := range blocks {
			c := cs[b]
			if c.done {
				continue
			}
			s := sc[b]
			mark := "✗"
			if s.Full() {
				mark = "✓"
				c.best, c.score, c.done = c.cur, 1, true
			} else if s.Frac() > c.score {
				c.best, c.score = c.cur, s.Frac()
			}
			parts = append(parts, fmt.Sprintf("%s %d/%d %s [%s]", blockTitle[b], s.OK, s.Total, mark, c.cur.Name()))
		}
		e.emit(i18n.T("ev.cand", attempts, strings.Join(parts, "; ")))
		if cs[probe.BlockGoogle].done && cs[probe.BlockGeneral].done {
			break
		}
	}

	final := strategy.Plan{}
	ok := map[string]bool{}
	for _, b := range blocks {
		c := cs[b]
		p := c.best
		if p.IsZero() {
			p = strategy.Proven[0]
		}
		ok[b] = c.score >= 1
		if b == probe.BlockGoogle {
			final.Google = p
		} else {
			final.General = p
		}
	}
	if e.applied == nil || e.applied.Google.Key() != final.Google.Key() || e.applied.General.Key() != final.General.Key() {
		if err := e.apply(final); err != nil {
			e.emit(i18n.T("ev.final_fail", errLine(err.Error())))
		}
	}
	if ok[probe.BlockGoogle] || ok[probe.BlockGeneral] {
		e.st.Put(e.net.ID, state.Entry{Name: e.net.Name, Plan: final, Saved: time.Now(), LastOK: time.Now()})
	}
	e.set(func(s *Status) {
		s.Plan = final
		s.Blocks = ok
		s.Progress = ""
		switch {
		case ok[probe.BlockGoogle] && ok[probe.BlockGeneral]:
			s.Phase = PhaseRunning
		case ok[probe.BlockGoogle] || ok[probe.BlockGeneral]:
			s.Phase = PhasePartial
			s.Message = partialText(ok)
		default:
			s.Phase = PhaseFailed
			s.Message = i18n.T("msg.not_found")
		}
	})
	e.emit(i18n.T("ev.done", time.Since(started).Round(time.Second), attempts, yesNo(ok[probe.BlockGoogle]), yesNo(ok[probe.BlockGeneral])))
}

func yesNo(ok bool) string {
	if ok {
		return i18n.T("works")
	}
	return i18n.T("not_works")
}

func (e *Engine) finish(plan strategy.Plan, sc map[string]probe.Score) {
	ok := map[string]bool{}
	for _, b := range blocks {
		ok[b] = sc[b].Full()
	}
	e.set(func(s *Status) {
		s.Plan = plan
		s.Blocks = ok
		s.Phase = PhaseRunning
		s.Message = ""
	})
}

func (e *Engine) watch(ctx context.Context) {
	if !e.run.Running() {
		e.emit(i18n.T("ev.winws_down"))
		if e.applied != nil {
			if err := e.apply(*e.applied); err != nil {
				e.emit(i18n.T("ev.restart_fail", errLine(err.Error())))
			}
		}
		return
	}
	var quick []probe.Target
	seen := map[string]bool{}
	for _, t := range e.targets {
		if !seen[t.Block] {
			seen[t.Block] = true
			quick = append(quick, t)
		}
	}
	sc := e.probe(ctx, quick)
	if sc[probe.BlockControl].OK == 0 {
		e.emit(i18n.T("ev.check_down"))
		return
	}
	st := e.Status()
	bad := false
	for _, b := range blocks {
		if st.Blocks[b] && !sc[b].Full() {
			bad = true
		}
	}
	if !bad {
		e.failures = 0
		e.st.Touch(e.net.ID)
		return
	}
	e.failures++
	e.emit(i18n.T("ev.check_fail", scoreText(sc), e.failures))
	if e.failures >= 2 {
		e.search(ctx, e.applied)
	}
}

func partialText(ok map[string]bool) string {
	var bad []string
	for _, b := range blocks {
		if !ok[b] {
			bad = append(bad, blockTitle[b])
		}
	}
	return i18n.T("msg.not_working", strings.Join(bad, ", "))
}

func errLine(s string) string {
	last := ""
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "github version") {
			continue
		}
		last = l
	}
	if last == "" {
		return strings.TrimSpace(firstLine(s))
	}
	return last
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

var ErrNoZapret = errors.New("zapret directory not found")

func FindZapret(exeDir string) (string, error) {
	candidates := []string{
		filepath.Join(exeDir, "zapret"),
		exeDir,
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "bin", "winws.exe")); err == nil {
			return c, nil
		}
	}
	return "", ErrNoZapret
}
