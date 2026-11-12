package main

import (
	"context"
	"deepmole/internal/i18n"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"deepmole/assets"
	"deepmole/internal/engine"
	"deepmole/internal/probe"
	"deepmole/internal/strategy"
	"deepmole/internal/update"
)

const taskName = "DeepMole"

func loadIcon() *walk.Icon {
	p := filepath.Join(os.TempDir(), "deepmole.ico")
	_ = os.WriteFile(p, assets.Icon, 0o644)
	ic, err := walk.NewIconFromFile(p)
	if err != nil {
		return walk.IconApplication()
	}
	return ic
}

func runTray(ctx context.Context, cancel context.CancelFunc, eng *engine.Engine, logPath, exe, autoUpdate string) {
	th := detectTheme()
	fs := loadFonts()
	icon := loadIcon()
	enabled := true

	var (
		mw         *walk.MainWindow
		lNet       *walk.Label
		events     *walk.TextEdit
		st         = &pill{th: &th, fonts: &fs, text: i18n.T("status.starting"), color: th.warn, back: th.pillWarn}
		svc        = &services{th: &th, fonts: &fs}
		lg         = &logo{th: &th}
		btnRe      = &button{th: &th, fonts: &fs, text: i18n.T("ui.research"), primary: true, enabled: true}
		btnToggle  = &button{th: &th, fonts: &fs, text: i18n.T("ui.disable"), enabled: true}
		btnLog     = &button{th: &th, fonts: &fs, text: i18n.T("ui.log"), enabled: true}
		btnUp      = &button{th: &th, fonts: &fs, text: "", primary: true, enabled: true}
		autoToggle = &toggle{th: &th, on: autostartEnabled()}
		dnsToggle  = &toggle{th: &th, on: true}
	)
	if eng != nil {
		dnsToggle.on = eng.Status().SecureDNS
	}
	svc.rows = serviceRows(&th, engine.Status{Enabled: true})

	demo := eng == nil
	if demo {
		eng = &engine.Engine{}
	}
	searching := false
	btnRe.onClick = func() {
		if demo {
			return
		}
		if searching {
			eng.StopSearch()
		} else {
			eng.Research()
		}
	}
	btnToggle.onClick = func() {
		enabled = !enabled
		if !demo {
			eng.SetEnabled(enabled)
		}
	}
	btnLog.onClick = func() { openPath(logPath) }

	bgBrush := SolidColorBrush{Color: wc(th.bg)}
	btn := func(b *button, w int) Widget {
		return CustomWidget{AssignTo: &b.cw, Paint: b.paint, MinSize: Size{Width: w, Height: 32}, MaxSize: Size{Width: w, Height: 32}}
	}

	err := MainWindow{
		AssignTo:   &mw,
		Title:      "DeepMole",
		Icon:       icon,
		Background: bgBrush,
		Size:       Size{Width: 600, Height: 540},
		MinSize:    Size{Width: 580, Height: 460},
		Font:       Font{Family: "Segoe UI", PointSize: 10},
		Layout:     VBox{Margins: Margins{Left: 20, Top: 16, Right: 20, Bottom: 16}, Spacing: 10},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true, Spacing: 8},
				Children: []Widget{
					CustomWidget{AssignTo: &lg.cw, Paint: lg.paint, MinSize: Size{Width: 22, Height: 22}, MaxSize: Size{Width: 22, Height: 22}},
					Label{Text: "DeepMole", Font: Font{Family: "Segoe UI", PointSize: 11, Bold: true}, TextColor: wc(th.fg)},
					HSpacer{},
					CustomWidget{AssignTo: &btnUp.cw, Paint: btnUp.paint, Visible: false, MinSize: Size{Width: 150, Height: 28}, MaxSize: Size{Width: 150, Height: 28}},
					Label{Text: version, Font: Font{Family: "Segoe UI", PointSize: 9}, TextColor: wc(th.muted)},
				},
			},
			CustomWidget{AssignTo: &st.cw, Paint: st.paint, MinSize: Size{Height: 30}, MaxSize: Size{Height: 30}},
			Label{AssignTo: &lNet, Text: i18n.T("ui.network", "…"), TextColor: wc(th.muted)},
			CustomWidget{AssignTo: &svc.cw, Paint: svc.paint, MinSize: Size{Height: 150}, MaxSize: Size{Height: 150}},
			TextEdit{AssignTo: &events, ReadOnly: true, VScroll: true, StretchFactor: 1, Background: SolidColorBrush{Color: wc(th.field)}, TextColor: wc(th.muted), Font: Font{Family: "Consolas", PointSize: 9}},
			Composite{
				Layout:  HBox{MarginsZero: true, Spacing: 8},
				MinSize: Size{Height: 32},
				MaxSize: Size{Height: 32},
				Children: []Widget{
					btn(btnRe, 124),
					btn(btnToggle, 96),
					btn(btnLog, 52),
					HSpacer{},
					Label{Text: i18n.T("ui.secure_dns"), TextColor: wc(th.muted), Font: Font{Family: "Segoe UI", PointSize: 9}},
					CustomWidget{AssignTo: &dnsToggle.cw, Paint: dnsToggle.paint, MinSize: Size{Width: 44, Height: 24}, MaxSize: Size{Width: 44, Height: 24}},
					Label{Text: i18n.T("ui.autostart"), TextColor: wc(th.muted), Font: Font{Family: "Segoe UI", PointSize: 9}},
					CustomWidget{AssignTo: &autoToggle.cw, Paint: autoToggle.paint, MinSize: Size{Width: 44, Height: 24}, MaxSize: Size{Width: 44, Height: 24}},
				},
			},
		},
	}.Create()
	if err != nil {
		fatal(false, err.Error())
	}
	applyDarkTitleBar(uintptr(mw.Handle()), th.dark)
	btnRe.attach()
	btnToggle.attach()
	btnLog.attach()
	autoToggle.attach()
	dnsToggle.attach()
	btnUp.attach()

	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		fatal(false, err.Error())
	}
	_ = ni.SetIcon(icon)
	_ = ni.SetToolTip("DeepMole")

	show := func() {
		mw.Show()
		win.ShowWindow(mw.Handle(), win.SW_RESTORE)
		win.SetForegroundWindow(mw.Handle())
	}
	quit := func() {
		cancel()
		if !demo {
			eng.Stop()
		}
		_ = ni.Dispose()
		walk.App().Exit(0)
		os.Exit(0)
	}

	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			show()
		}
	})

	newAction := func(text string, fn func()) *walk.Action {
		a := walk.NewAction()
		_ = a.SetText(text)
		if fn != nil {
			a.Triggered().Attach(fn)
		}
		_ = ni.ContextMenu().Actions().Add(a)
		return a
	}
	aStatus := newAction(i18n.T("status.starting"), nil)
	aStatus.SetEnabled(false)
	_ = ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())
	newAction(i18n.T("ui.open"), show)
	aToggle := newAction(i18n.T("ui.disable_bypass"), btnToggle.onClick)
	newAction(i18n.T("ui.research"), btnRe.onClick)
	aDNS := newAction(i18n.T("ui.secure_dns"), nil)
	aDNS.SetCheckable(true)
	_ = aDNS.SetChecked(dnsToggle.on)
	aAuto := newAction(i18n.T("ui.autostart"), nil)
	aAuto.SetCheckable(true)
	_ = aAuto.SetChecked(autoToggle.on)
	newAction(i18n.T("ui.open_log"), btnLog.onClick)
	aUpdate := newAction(i18n.T("ui.update_check"), nil)
	_ = ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())
	newAction(i18n.T("ui.quit"), quit)
	_ = ni.SetVisible(true)

	up := &updater{mode: autoUpdate, current: version, exe: exe, mw: mw, quit: quit}
	up.event = func(msg string) { events.AppendText(time.Now().Format("15:04:05") + "  " + msg + "\r\n") }
	up.found = func(r *update.Release) {
		btnUp.SetText(i18n.T("ui.update_btn", r.Version))
		btnUp.cw.SetVisible(true)
		_ = aUpdate.SetText(i18n.T("ui.update_btn", r.Version))
	}
	btnUp.onClick = up.install
	aUpdate.Triggered().Attach(func() {
		if up.rel != nil {
			up.install()
			return
		}
		go up.check(true)
	})
	if demo {
		aUpdate.SetEnabled(false)
	}
	up.start()

	setAuto := func(on bool) {
		if err := setAutostart(exe, on); err != nil {
			walk.MsgBox(mw, "DeepMole", i18n.T("ui.autostart_err", err.Error()), walk.MsgBoxIconError)
			on = autostartEnabled()
		}
		autoToggle.SetOn(on)
		_ = aAuto.SetChecked(on)
	}
	autoToggle.onChange = setAuto
	aAuto.Triggered().Attach(func() { setAuto(!autoToggle.on) })

	setDNS := func(on bool) {
		dnsToggle.SetOn(on)
		_ = aDNS.SetChecked(on)
		if !demo {
			eng.SetSecureDNS(on)
		}
	}
	dnsToggle.onChange = setDNS
	aDNS.Triggered().Attach(func() { setDNS(!dnsToggle.on) })

	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		mw.Hide()
	})

	eng.OnChange = func(s engine.Status) {
		mw.Synchronize(func() {
			col, back := statusColors(&th, s)
			st.Set(s.Text(), col, back)
			_ = lNet.SetText(i18n.T("ui.network", s.Network))
			svc.Set(serviceRows(&th, s))
			tip := "DeepMole: " + s.Text()
			if s.Message != "" {
				tip += " (" + s.Message + ")"
			}
			_ = aStatus.SetText(s.Text())
			_ = ni.SetToolTip(tip)
			if s.Enabled {
				btnToggle.SetText(i18n.T("ui.disable"))
				_ = aToggle.SetText(i18n.T("ui.disable_bypass"))
			} else {
				btnToggle.SetText(i18n.T("ui.enable"))
				_ = aToggle.SetText(i18n.T("ui.enable_bypass"))
			}
			searching = s.Phase == engine.PhaseSearching
			if searching {
				btnRe.SetText(i18n.T("ui.stop"))
			} else {
				btnRe.SetText(i18n.T("ui.research"))
			}
			btnRe.SetEnabled(s.Enabled)
			if dnsToggle.on != s.SecureDNS {
				dnsToggle.SetOn(s.SecureDNS)
				_ = aDNS.SetChecked(s.SecureDNS)
			}
		})
	}
	eng.OnEvent = func(msg string) {
		mw.Synchronize(func() {
			events.AppendText(time.Now().Format("15:04:05") + "  " + msg + "\r\n")
		})
	}

	if demo {
		eng.OnChange(engine.Status{
			Phase: engine.PhaseRunning, Network: "Home_5G", Enabled: true, DNS: "Google DoH", SecureDNS: true,
			Plan:   strategy.Plan{Google: strategy.Proven[3], General: strategy.Proven[1], Telegram: strategy.ProvenTelegram[0]},
			Blocks: map[string]bool{probe.BlockGoogle: true, probe.BlockGeneral: true, probe.BlockTelegram: true},
		})
		eng.OnEvent(i18n.T("ev.net_new", "Home_5G"))
		eng.OnEvent(i18n.T("ev.cand", 1, "YouTube 2/3 ✗ [multisplit pos=1 ovl=568/4pda]; Discord 3/3 ✓ [multisplit pos=1 ovl=568/4pda]; Telegram 2/2 ✓ [fake raw badseq r6]"))
		eng.OnEvent(i18n.T("ev.cand", 2, "YouTube 3/3 ✓ [fake+multidisorder badseq pos=1,midsld tls=zero+mod r11]"))
		eng.OnEvent(i18n.T("ev.done", "6s", 2, "YouTube: "+i18n.T("works")+", Discord: "+i18n.T("works")+", Telegram: "+i18n.T("works")))
	} else {
		go eng.Run(ctx)
	}
	mw.Run()
}

func schtasks(cmdLine string) (string, error) {
	cmd := exec.Command("schtasks")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CmdLine: "schtasks " + cmdLine}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func autostartEnabled() bool {
	_, err := schtasks(`/Query /TN "` + taskName + `"`)
	return err == nil
}

func setAutostart(exe string, on bool) error {
	var out string
	var err error
	if on {
		out, err = schtasks(`/Create /F /TN "` + taskName + `" /SC ONLOGON /RL HIGHEST /TR "\"` + exe + `\""`)
	} else {
		out, err = schtasks(`/Delete /F /TN "` + taskName + `"`)
	}
	if err != nil && logger != nil {
		logger.Printf("schtasks: %v %s", err, strings.TrimSpace(out))
	}
	return err
}
