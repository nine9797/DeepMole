package strategy

import (
	"sort"
	"strings"
	"testing"
)

func same(t *testing.T, got []string, want string) {
	t.Helper()
	w := strings.Fields(want)
	g := append([]string(nil), got...)
	sort.Strings(g)
	sort.Strings(w)
	if strings.Join(g, " ") != strings.Join(w, " ") {
		t.Errorf("\n got: %s\nwant: %s", strings.Join(g, " "), strings.Join(w, " "))
	}
}

var testPaths = Paths{Bin: `B\`, Lists: `L\`}

func TestRenderMatchesPresets(t *testing.T) {
	cases := []struct {
		p    Params
		http bool
		want string
	}{
		{Params{Method: "multisplit", SplitPos: "1", Pattern: "4pda", SeqOvl: 568}, true,
			`--dpi-desync=multisplit --dpi-desync-split-seqovl=568 --dpi-desync-split-pos=1 --dpi-desync-split-seqovl-pattern=B\tls_clienthello_4pda_to.bin`},
		{Params{Method: "fake,multidisorder", SplitPos: "1,midsld", Repeats: 11, Fooling: "badseq", FakeTLS: "zero+mod"}, true,
			`--dpi-desync=fake,multidisorder --dpi-desync-split-pos=1,midsld --dpi-desync-repeats=11 --dpi-desync-fooling=badseq --dpi-desync-fake-tls=0x00000000 --dpi-desync-fake-tls=! --dpi-desync-fake-tls-mod=rnd,dupsid,sni=www.google.com --dpi-desync-fake-http=B\tls_clienthello_www_sferum_ru.bin`},
		{Params{Method: "fake", Repeats: 6, Fooling: "ts", FakeTLS: "stun+google"}, true,
			`--dpi-desync=fake --dpi-desync-repeats=6 --dpi-desync-fooling=ts --dpi-desync-fake-tls=B\stun.bin --dpi-desync-fake-tls=B\tls_clienthello_www_google_com.bin --dpi-desync-fake-http=B\tls_clienthello_www_sferum_ru.bin`},
		{Params{Method: "fake", Repeats: 6, Fooling: "ts", FakeTLS: "stun+google"}, false,
			`--dpi-desync=fake --dpi-desync-repeats=6 --dpi-desync-fooling=ts --dpi-desync-fake-tls=B\stun.bin --dpi-desync-fake-tls=B\tls_clienthello_www_google_com.bin`},
		{Params{Method: "fake,multisplit", SeqOvl: 664, SplitPos: "1", Fooling: "ts", Repeats: 8, Pattern: "sferum", FakeTLS: "stun2+sferum"}, true,
			`--dpi-desync=fake,multisplit --dpi-desync-split-seqovl=664 --dpi-desync-split-pos=1 --dpi-desync-fooling=ts --dpi-desync-repeats=8 --dpi-desync-split-seqovl-pattern=B\tls_clienthello_www_sferum_ru.bin --dpi-desync-fake-tls=B\stun2.bin --dpi-desync-fake-tls=B\tls_clienthello_www_sferum_ru.bin --dpi-desync-fake-http=B\tls_clienthello_www_sferum_ru.bin`},
		{Params{Method: "fake,multisplit", SeqOvl: 480, SplitPos: "1", Fooling: "ts", Repeats: 4, Pattern: "stun2", FakeTLS: "sferum"}, true,
			`--dpi-desync=fake,multisplit --dpi-desync-split-seqovl=480 --dpi-desync-split-pos=1 --dpi-desync-fooling=ts --dpi-desync-repeats=4 --dpi-desync-split-seqovl-pattern=B\stun2.bin --dpi-desync-fake-tls=B\tls_clienthello_www_sferum_ru.bin --dpi-desync-fake-http=B\tls_clienthello_www_sferum_ru.bin`},
		{Params{Method: "multisplit", SplitPos: "2,sniext+1", SeqOvl: 679, Pattern: "google"}, true,
			`--dpi-desync=multisplit --dpi-desync-split-pos=2,sniext+1 --dpi-desync-split-seqovl=679 --dpi-desync-split-seqovl-pattern=B\tls_clienthello_www_google_com.bin`},
		{Params{Method: "fake,hostfakesplit", Fooling: "ts", FakeTLS: "mod:ya", HostFake: "ya.ru"}, true,
			`--dpi-desync=fake,hostfakesplit --dpi-desync-fake-tls-mod=rnd,dupsid,sni=ya.ru --dpi-desync-hostfakesplit-mod=host=ya.ru,altorder=1 --dpi-desync-fooling=ts --dpi-desync-fake-http=B\tls_clienthello_www_sferum_ru.bin`},
		{Params{Method: "fake,hostfakesplit", Fooling: "ts", Repeats: 5, FakeTLS: "sochi+stun2", HostFake: "mail.ru"}, true,
			`--dpi-desync=fake,hostfakesplit --dpi-desync-fooling=ts --dpi-desync-hostfakesplit-mod=host=mail.ru,altorder=1 --dpi-desync-repeats=5 --dpi-desync-fake-tls=B\tls_clienthello_sochi_park.bin --dpi-desync-fake-tls=B\stun2.bin --dpi-desync-fake-http=B\tls_clienthello_www_sferum_ru.bin`},
		{Params{Method: "hostfakesplit", Repeats: 4, Fooling: "ts,md5sig", HostFake: "ozon.ru"}, true,
			`--dpi-desync=hostfakesplit --dpi-desync-repeats=4 --dpi-desync-fooling=ts,md5sig --dpi-desync-hostfakesplit-mod=host=ozon.ru`},
		{Params{Method: "fake", Repeats: 6, Fooling: "badseq", BadseqInc: 2, FakeTLS: "none"}, true,
			`--dpi-desync=fake --dpi-desync-fake-tls-mod=none --dpi-desync-repeats=6 --dpi-desync-fooling=badseq --dpi-desync-badseq-increment=2 --dpi-desync-fake-http=B\tls_clienthello_www_sferum_ru.bin`},
	}
	for _, c := range cases {
		got := c.p.Render(testPaths, c.http)
		t.Run(c.p.Name(), func(t *testing.T) { same(t, got, c.want) })
	}
}

func TestRenderExtraKnobs(t *testing.T) {
	p := Params{Method: "fake,multisplit", Fooling: "md5sig", SplitPos: "midsld", Pattern: "google", SeqOvl: 681, FakeTLS: "mod:padencap", Repeats: 6, TTL: "autottl=2", WSSize: "1:6"}
	same(t, p.Render(testPaths, false), `--dpi-desync=fake,multisplit --wssize=1:6 --dpi-desync-autottl=2 --dpi-desync-split-seqovl=681 --dpi-desync-split-pos=midsld --dpi-desync-fooling=md5sig --dpi-desync-repeats=6 --dpi-desync-split-seqovl-pattern=B\tls_clienthello_www_google_com.bin --dpi-desync-fake-tls-mod=rnd,dupsid,padencap,sni=www.google.com`)
	p = Params{Method: "fake,fakeddisorder", Fooling: "ts", SplitPos: "1", FakeTLS: "stun+google", Repeats: 6, TTL: "ttl=4"}
	same(t, p.Render(testPaths, true), `--dpi-desync=fake,fakeddisorder --dpi-desync-ttl=4 --dpi-desync-split-pos=1 --dpi-desync-fooling=ts --dpi-desync-repeats=6 --dpi-desync-fake-tls=B\stun.bin --dpi-desync-fake-tls=B\tls_clienthello_www_google_com.bin --dpi-desync-fake-http=B\tls_clienthello_www_sferum_ru.bin`)
	q := Params{Method: "multisplit", TTL: "ttl=3"}
	if q.Render(testPaths, true)[1] != "--dpi-desync-ttl=3" {
		t.Errorf("ttl must render for any method")
	}
}

func TestFileFallback(t *testing.T) {
	old := map[string]bool{
		`B\quic_initial_dbankcloud_ru.bin`:     true,
		`B\quic_initial_www_google_com.bin`:    true,
		`B\tls_clienthello_max_ru.bin`:         true,
		`B\tls_clienthello_www_google_com.bin`: true,
		`B\stun.bin`:                           true,
	}
	paths := Paths{Bin: `B\`, Lists: `L\`, Exists: func(p string) bool { return old[p] }}
	args := strings.Join(BuildArgs(paths, Plan{Google: Proven[0], General: Proven[3]}, "all"), " ")
	for _, want := range []string{
		`--dpi-desync-fake-discord=B\quic_initial_dbankcloud_ru.bin`,
		`--dpi-desync-fake-unknown-udp=B\quic_initial_dbankcloud_ru.bin`,
		`--dpi-desync-fake-http=B\tls_clienthello_max_ru.bin`,
	} {
		if !strings.Contains(args, want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(args, "sferum") || strings.Contains(args, "ACTIVE_") {
		t.Errorf("new files must not be used when absent")
	}
	if !Proven[3].Available(paths) {
		t.Errorf("stun+google must be available")
	}
	if Proven[5].Available(paths) {
		t.Errorf("sferum preset must be unavailable")
	}
	fresh := map[string]bool{`B\ACTIVE_DISCORD_UDP.bin`: true, `B\ACTIVE_GAME_UDP.bin`: true, `B\tls_clienthello_www_sferum_ru.bin`: true}
	paths.Exists = func(p string) bool { return fresh[p] }
	args = strings.Join(BuildArgs(paths, Plan{Google: Proven[0], General: Proven[3]}, "all"), " ")
	if !strings.Contains(args, `--dpi-desync-fake-stun=B\ACTIVE_DISCORD_UDP.bin`) || !strings.Contains(args, `--dpi-desync-fake-unknown-udp=B\ACTIVE_GAME_UDP.bin`) {
		t.Errorf("active fakes must be preferred: %s", args)
	}
}

func TestProvenUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Proven {
		if seen[p.Key()] {
			t.Errorf("duplicate preset: %s", p.Name())
		}
		seen[p.Key()] = true
		for _, f := range p.Files() {
			if f == "" {
				t.Errorf("unknown file in preset: %s", p.Name())
			}
		}
	}
}

func TestNeighborsValid(t *testing.T) {
	for _, p := range Proven {
		for _, n := range Neighbors(p) {
			if n.Key() == p.Key() {
				t.Errorf("neighbor equals origin: %s", p.Name())
			}
			if n.SeqOvl > 0 && n.Pattern == "" {
				t.Errorf("seqovl without pattern: %s", n.Name())
			}
			if !n.hasFake() && n.FakeTLS != "" {
				t.Errorf("fake tls without fake: %s", n.Name())
			}
			if !n.hasFake() && !strings.Contains(n.Method, "hostfakesplit") && n.Fooling != "" {
				t.Errorf("fooling without fake: %s", n.Name())
			}
			if !n.hasFake() && n.TTL != "" {
				t.Errorf("ttl without fake: %s", n.Name())
			}
			if !n.hasSplit() && n.SplitPos != "" {
				t.Errorf("split-pos without split: %s", n.Name())
			}
			for _, f := range n.Files() {
				if f == "" {
					t.Errorf("unknown file in neighbor: %s", n.Name())
				}
			}
			args := n.Render(testPaths, true)
			if len(args) == 0 || !strings.HasPrefix(args[0], "--dpi-desync=") || args[0] == "--dpi-desync=" {
				t.Errorf("bad render: %v", args)
			}
		}
	}
}

func TestBuildArgsBlocks(t *testing.T) {
	args := BuildArgs(testPaths, Plan{Google: Proven[0], General: Proven[3]}, "")
	n := 0
	for _, a := range args {
		if a == "--new" {
			n++
		}
	}
	if n != 6 {
		t.Errorf("expected 7 profiles, got %d", n+1)
	}
	if !strings.HasPrefix(args[0], "--wf-tcp=") || !strings.HasPrefix(args[1], "--wf-udp=") {
		t.Errorf("filters must come first: %v", args[:2])
	}
	all := BuildArgs(testPaths, Plan{Google: Proven[0], General: Proven[3]}, "all")
	if !strings.Contains(strings.Join(all, " "), "--filter-tcp=1024-65535") || !strings.Contains(strings.Join(all, " "), "--filter-udp=1024-65535") {
		t.Errorf("game filter blocks missing")
	}
}
