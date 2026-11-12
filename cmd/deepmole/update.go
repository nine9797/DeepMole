package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lxn/walk"

	"deepmole/internal/i18n"
	"deepmole/internal/update"
)

type updater struct {
	mode    string
	current string
	exe     string
	mw      *walk.MainWindow
	event   func(string)
	found   func(*update.Release)
	quit    func()

	mu   sync.Mutex
	rel  *update.Release
	busy bool
}

func (u *updater) enabled() bool {
	return u.mode != "off" && u.current != "dev"
}

func (u *updater) start() {
	if !u.enabled() {
		return
	}
	go func() {
		u.check(false)
		for range time.Tick(24 * time.Hour) {
			u.check(false)
		}
	}()
}

func (u *updater) check(manual bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rel, err := update.Check(ctx, u.current)
	if err != nil {
		if logger != nil {
			logger.Printf("update check: %v", err)
		}
		if manual {
			u.mw.Synchronize(func() { walk.MsgBox(u.mw, "DeepMole", i18n.T("ui.update_err", err.Error()), walk.MsgBoxIconWarning) })
		}
		return
	}
	if !update.Newer(u.current, rel.Version) {
		if manual {
			u.mw.Synchronize(func() { walk.MsgBox(u.mw, "DeepMole", i18n.T("ui.update_none", u.current), walk.MsgBoxIconInformation) })
		}
		return
	}
	u.mu.Lock()
	first := u.rel == nil || u.rel.Version != rel.Version
	u.rel = rel
	u.mu.Unlock()
	u.mw.Synchronize(func() {
		if first {
			u.event(i18n.T("ev.update_found", rel.Version, rel.Page))
		}
		u.found(rel)
		switch {
		case u.mode == "auto" && first:
			u.install()
		case manual || first:
			if walk.MsgBox(u.mw, "DeepMole", i18n.T("ui.update_ask", rel.Version, u.current), walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes {
				u.install()
			}
		}
	})
}

func (u *updater) install() {
	u.mu.Lock()
	rel := u.rel
	if rel == nil || u.busy {
		u.mu.Unlock()
		return
	}
	u.busy = true
	u.mu.Unlock()
	go func() {
		err := u.run(rel)
		u.mu.Lock()
		u.busy = false
		u.mu.Unlock()
		if err != nil {
			u.mw.Synchronize(func() { u.event(i18n.T("ev.update_fail", err.Error())) })
		}
	}()
}

func (u *updater) run(rel *update.Release) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	dir := filepath.Join(os.TempDir(), "deepmole-update")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	installed := update.Installed(filepath.Dir(u.exe))
	last := -10
	file, err := update.Download(ctx, rel, u.current, dir, installed, func(pct int) {
		if pct-last >= 10 || pct == 100 {
			last = pct
			u.mw.Synchronize(func() { u.event(i18n.T("ev.update_dl", rel.Version, pct)) })
		}
	})
	if err != nil {
		return err
	}
	u.mw.Synchronize(func() { u.event(i18n.T("ev.update_apply")) })
	if err := update.Apply(u.exe, file, installed); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	u.mw.Synchronize(u.quit)
	return nil
}

func updateMode(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "auto", "off":
		return strings.ToLower(strings.TrimSpace(s))
	}
	return "ask"
}
