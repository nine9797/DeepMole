package main

import (
	"image"
	"image/color"
	"unsafe"

	"github.com/lxn/walk"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"deepmole/internal/engine"
	"deepmole/internal/i18n"
	"deepmole/internal/probe"
	"deepmole/internal/raster"
)

type theme struct {
	dark                                 bool
	bg, fg, muted, line, field, hover    uint32
	pillOK, pillWarn, pillErr, pillMuted uint32
	accent, ink, paper, ok, warn, err    uint32
	dis, press                           uint32
}

func wc(v uint32) walk.Color { return walk.RGB(byte(v>>16), byte(v>>8), byte(v)) }

func nc(v uint32) color.NRGBA { return raster.Hex(v) }

func detectTheme() theme {
	dark := false
	if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE); err == nil {
		if v, _, err := k.GetIntegerValue("AppsUseLightTheme"); err == nil && v == 0 {
			dark = true
		}
		k.Close()
	}
	t := theme{dark: dark, accent: 0xF0A030, ink: 0x141414, paper: 0xFAFAF7, ok: 0x2E9E5B, warn: 0xD98A14, err: 0xD64545, press: 0xD98A14}
	if dark {
		t.bg, t.fg, t.muted, t.line, t.field, t.hover = 0x202020, 0xECE9E2, 0x9B968B, 0x3A3833, 0x2B2A27, 0x33312D
		t.pillOK, t.pillWarn, t.pillErr, t.pillMuted = 0x223228, 0x3E311E, 0x392525, 0x2B2A27
		t.dis = 0x6B675E
	} else {
		t.bg, t.fg, t.muted, t.line, t.field, t.hover = 0xF9F9F9, 0x1A1916, 0x6B675E, 0xE1DED6, 0xFFFFFF, 0xEFEDE7
		t.pillOK, t.pillWarn, t.pillErr, t.pillMuted = 0xDCECE3, 0xF4E7D4, 0xF4E0E0, 0xEFEDE7
		t.dis = 0x9B968B
	}
	return t
}

func applyDarkTitleBar(hwnd uintptr, dark bool) {
	v := int32(0)
	if dark {
		v = 1
	}
	dwm := windows.NewLazySystemDLL("dwmapi.dll")
	proc := dwm.NewProc("DwmSetWindowAttribute")
	if proc.Find() != nil {
		return
	}
	_, _, _ = proc.Call(hwnd, 20, uintptr(unsafe.Pointer(&v)), 4)
}

type fonts struct {
	ui, bold, title, mono, small *walk.Font
}

func loadFonts() fonts {
	mk := func(family string, size int, style walk.FontStyle) *walk.Font {
		f, err := walk.NewFont(family, size, style)
		if err != nil {
			f, _ = walk.NewFont("Segoe UI", 10, 0)
		}
		return f
	}
	return fonts{
		ui:    mk("Segoe UI", 10, 0),
		bold:  mk("Segoe UI", 10, walk.FontBold),
		title: mk("Segoe UI", 11, walk.FontBold),
		mono:  mk("Consolas", 9, 0),
		small: mk("Segoe UI", 9, 0),
	}
}

func blit(c *walk.Canvas, img image.Image, x, y int) {
	bm, err := walk.NewBitmapFromImageForDPI(img, 96)
	if err != nil {
		return
	}
	defer bm.Dispose()
	_ = c.DrawImagePixels(bm, walk.Point{X: x, Y: y})
}

func fillRect(c *walk.Canvas, r walk.Rectangle, col uint32) {
	b, err := walk.NewSolidColorBrush(wc(col))
	if err != nil {
		return
	}
	defer b.Dispose()
	_ = c.FillRectanglePixels(b, r)
}

type button struct {
	cw      *walk.CustomWidget
	th      *theme
	fonts   *fonts
	text    string
	primary bool
	enabled bool
	pressed bool
	onClick func()
}

func (b *button) paint(c *walk.Canvas, _ walk.Rectangle) error {
	r := b.cw.ClientBoundsPixels()
	rad := b.cw.IntFrom96DPI(6)
	fillRect(c, r, b.th.bg)
	bg, fg := b.th.field, b.th.fg
	switch {
	case !b.enabled:
		bg, fg = b.th.bg, b.th.dis
	case b.primary && b.pressed:
		bg, fg = b.th.press, b.th.ink
	case b.primary:
		bg, fg = b.th.accent, b.th.ink
	case b.pressed:
		bg = b.th.hover
	}
	if b.primary && b.enabled {
		blit(c, raster.RoundRect(r.Width, r.Height, rad, nc(bg), nc(b.th.bg)), r.X, r.Y)
	} else {
		blit(c, raster.RoundRectBorder(r.Width, r.Height, rad, 1, nc(bg), nc(b.th.line), nc(b.th.bg)), r.X, r.Y)
	}
	return c.DrawTextPixels(b.text, b.fonts.bold, wc(fg), r, walk.TextCenter|walk.TextVCenter|walk.TextSingleLine)
}

func (b *button) SetText(t string) {
	b.text = t
	b.cw.Invalidate()
}

func (b *button) SetEnabled(on bool) {
	b.enabled = on
	b.cw.Invalidate()
}

func (b *button) attach() {
	b.cw.MouseDown().Attach(func(x, y int, btn walk.MouseButton) {
		if btn == walk.LeftButton && b.enabled {
			b.pressed = true
			b.cw.Invalidate()
		}
	})
	b.cw.MouseUp().Attach(func(x, y int, btn walk.MouseButton) {
		if btn != walk.LeftButton {
			return
		}
		was := b.pressed
		b.pressed = false
		b.cw.Invalidate()
		r := b.cw.ClientBoundsPixels()
		if was && b.enabled && x >= 0 && y >= 0 && x < r.Width && y < r.Height && b.onClick != nil {
			b.onClick()
		}
	})
	b.cw.SetCursor(walk.CursorHand())
}

type toggle struct {
	cw       *walk.CustomWidget
	th       *theme
	on       bool
	onChange func(bool)
}

func (t *toggle) paint(c *walk.Canvas, _ walk.Rectangle) error {
	r := t.cw.ClientBoundsPixels()
	fillRect(c, r, t.th.bg)
	track, knob, line := t.th.line, uint32(0xFFFFFF), t.th.line
	if t.th.dark {
		knob = t.th.paper
	}
	if t.on {
		track, knob, line = t.th.accent, t.th.ink, t.th.accent
	}
	blit(c, raster.Toggle(r.Width, r.Height, t.cw.IntFrom96DPI(3), t.on, nc(track), nc(knob), nc(line), nc(t.th.bg)), r.X, r.Y)
	return nil
}

func (t *toggle) SetOn(on bool) {
	t.on = on
	t.cw.Invalidate()
}

func (t *toggle) attach() {
	t.cw.MouseUp().Attach(func(x, y int, btn walk.MouseButton) {
		if btn != walk.LeftButton {
			return
		}
		t.on = !t.on
		t.cw.Invalidate()
		if t.onChange != nil {
			t.onChange(t.on)
		}
	})
	t.cw.SetCursor(walk.CursorHand())
}

type pill struct {
	cw    *walk.CustomWidget
	th    *theme
	fonts *fonts
	text  string
	color uint32
	back  uint32
}

func (p *pill) paint(c *walk.Canvas, _ walk.Rectangle) error {
	r := p.cw.ClientBoundsPixels()
	fillRect(c, r, p.th.bg)
	h := r.Height
	bounds, _, err := c.MeasureTextPixels(p.text, p.fonts.bold, walk.Rectangle{Width: 4096, Height: h}, walk.TextSingleLine|walk.TextCalcRect)
	if err != nil {
		return err
	}
	padX := p.cw.IntFrom96DPI(12)
	dot := p.cw.IntFrom96DPI(8)
	gap := p.cw.IntFrom96DPI(8)
	w := padX + dot + gap + bounds.Width + padX
	if w > r.Width {
		w = r.Width
	}
	blit(c, raster.RoundRect(w, h, h/2, nc(p.back), nc(p.th.bg)), r.X, r.Y)
	blit(c, raster.Circle(dot, nc(p.color), nc(p.back)), r.X+padX, r.Y+(h-dot)/2)
	text := walk.Rectangle{X: r.X + padX + dot + gap, Y: r.Y, Width: w - padX - dot - gap, Height: h}
	return c.DrawTextPixels(p.text, p.fonts.bold, wc(p.color), text, walk.TextLeft|walk.TextVCenter|walk.TextSingleLine)
}

func (p *pill) Set(text string, color, back uint32) {
	p.text, p.color, p.back = text, color, back
	p.cw.Invalidate()
}

type serviceRow struct {
	name, strat string
	color       uint32
}

type services struct {
	cw    *walk.CustomWidget
	th    *theme
	fonts *fonts
	rows  []serviceRow
}

func (s *services) paint(c *walk.Canvas, _ walk.Rectangle) error {
	r := s.cw.ClientBoundsPixels()
	fillRect(c, r, s.th.bg)
	if len(s.rows) == 0 {
		return nil
	}
	rowH := r.Height / len(s.rows)
	dot := s.cw.IntFrom96DPI(8)
	gap := s.cw.IntFrom96DPI(10)
	fillRect(c, walk.Rectangle{X: r.X, Y: r.Y, Width: r.Width, Height: 1}, s.th.line)
	for i, row := range s.rows {
		y := r.Y + i*rowH
		blit(c, raster.Circle(dot, nc(row.color), nc(s.th.bg)), r.X, y+(rowH-dot)/2)
		name := walk.Rectangle{X: r.X + dot + gap, Y: y, Width: r.Width / 2, Height: rowH}
		if err := c.DrawTextPixels(row.name, s.fonts.bold, wc(s.th.fg), name, walk.TextLeft|walk.TextVCenter|walk.TextSingleLine); err != nil {
			return err
		}
		strat := walk.Rectangle{X: r.X + r.Width/3, Y: y, Width: r.Width - r.Width/3, Height: rowH}
		if err := c.DrawTextPixels(row.strat, s.fonts.mono, wc(s.th.muted), strat, walk.TextRight|walk.TextVCenter|walk.TextSingleLine|walk.TextEndEllipsis); err != nil {
			return err
		}
		fillRect(c, walk.Rectangle{X: r.X, Y: y + rowH - 1, Width: r.Width, Height: 1}, s.th.line)
	}
	return nil
}

func (s *services) Set(rows []serviceRow) {
	s.rows = rows
	s.cw.Invalidate()
}

type logo struct {
	cw *walk.CustomWidget
	th *theme
}

func (l *logo) paint(c *walk.Canvas, _ walk.Rectangle) error {
	r := l.cw.ClientBoundsPixels()
	fillRect(c, r, l.th.bg)
	s := r.Height
	if r.Width < s {
		s = r.Width
	}
	blit(c, raster.Logo(s, nc(l.th.bg)), r.X, r.Y+(r.Height-s)/2)
	return nil
}

func statusColors(th *theme, s engine.Status) (uint32, uint32) {
	switch s.Phase {
	case engine.PhaseRunning:
		return th.ok, th.pillOK
	case engine.PhaseSearching, engine.PhasePartial, engine.PhaseIdle:
		return th.warn, th.pillWarn
	case engine.PhaseDisabled:
		return th.muted, th.pillMuted
	}
	return th.err, th.pillErr
}

func serviceRows(th *theme, s engine.Status) []serviceRow {
	mk := func(name, block, strat string) serviceRow {
		var col uint32
		switch {
		case !s.Enabled:
			col = th.dis
		case s.Phase == engine.PhaseSearching || s.Phase == engine.PhaseIdle:
			if s.Blocks[block] {
				col = th.ok
			} else {
				col, strat = th.warn, "…"
			}
		case s.Blocks[block]:
			col = th.ok
		default:
			col = th.err
		}
		if strat == "" || !s.Enabled {
			strat = "—"
		}
		return serviceRow{name: name, strat: strat, color: col}
	}
	dns := serviceRow{name: "DNS", strat: i18n.T("dns.system"), color: th.muted}
	switch {
	case !s.Enabled:
		dns.strat, dns.color = "—", th.dis
	case s.DNS == engine.DNSBlocked:
		dns.strat, dns.color = i18n.T("dns.blocked"), th.err
	case s.DNS != "":
		dns.strat, dns.color = i18n.T("dns.secure", s.DNS), th.ok
	}
	rows := []serviceRow{
		mk("YouTube", probe.BlockGoogle, s.Plan.Google.Name()),
		mk("Discord", probe.BlockGeneral, s.Plan.General.Name()),
	}
	if _, ok := s.Blocks[probe.BlockTelegram]; ok || !s.Plan.Telegram.IsZero() {
		rows = append(rows, mk("Telegram", probe.BlockTelegram, s.Plan.Telegram.Name()))
	}
	return append(rows, dns)
}
