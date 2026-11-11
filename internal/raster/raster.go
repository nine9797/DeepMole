package raster

import (
	"image"
	"image/color"
	"math"
)

type Sampler func(x, y float64) (color.NRGBA, bool)

func Render(w, h, ss int, bg color.NRGBA, s Sampler) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	n := float64(ss * ss)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := float64(x) + (float64(sx)+0.5)/float64(ss)
					py := float64(y) + (float64(sy)+0.5)/float64(ss)
					c, ok := s(px, py)
					if !ok {
						c = bg
					}
					al := float64(c.A) / 255
					r += float64(c.R) * al
					g += float64(c.G) * al
					b += float64(c.B) * al
					a += al
				}
			}
			if a > 0 {
				img.SetNRGBA(x, y, color.NRGBA{uint8(r / a), uint8(g / a), uint8(b / a), uint8(a / n * 255)})
			}
		}
	}
	return img
}

func InEllipse(x, y, cx, cy, rx, ry float64) bool {
	dx, dy := (x-cx)/rx, (y-cy)/ry
	return dx*dx+dy*dy <= 1
}

func InRoundRect(x, y, x0, y0, w, h, r float64) bool {
	x -= x0
	y -= y0
	if x < 0 || y < 0 || x > w || y > h {
		return false
	}
	cx := math.Max(r-x, math.Max(0, x-(w-r)))
	cy := math.Max(r-y, math.Max(0, y-(h-r)))
	return cx*cx+cy*cy <= r*r
}

func Hex(v uint32) color.NRGBA {
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

var (
	Accent = Hex(0xF0A030)
	Ink    = Hex(0x141414)
	Paper  = Hex(0xFAFAF7)
)

func LogoSampler(size float64) Sampler {
	k := size / 256
	return func(x, y float64) (color.NRGBA, bool) {
		u, v := x/k, y/k
		if !InRoundRect(u, v, 0, 0, 256, 256, 56) {
			return color.NRGBA{}, false
		}
		for _, ex := range [2]float64{110, 146} {
			if InEllipse(u, v, ex, 160, 9, 9) {
				return Ink, true
			}
			if InEllipse(u, v, ex, 160, 15, 15) {
				return Paper, true
			}
		}
		if InEllipse(u, v, 128, 150, 88, 36) && !InEllipse(u, v, 128, 165, 92, 38) {
			return Paper, true
		}
		if InEllipse(u, v, 128, 150, 100, 46) {
			return Ink, true
		}
		return Accent, true
	}
}

func Logo(size int, bg color.NRGBA) *image.NRGBA {
	return Render(size, size, 4, bg, LogoSampler(float64(size)))
}

func RoundRect(w, h, radius int, fill, bg color.NRGBA) *image.NRGBA {
	r := float64(radius)
	return Render(w, h, 4, bg, func(x, y float64) (color.NRGBA, bool) {
		return fill, InRoundRect(x, y, 0, 0, float64(w), float64(h), r)
	})
}

func RoundRectBorder(w, h, radius, border int, fill, line, bg color.NRGBA) *image.NRGBA {
	r := float64(radius)
	b := float64(border)
	return Render(w, h, 4, bg, func(x, y float64) (color.NRGBA, bool) {
		if InRoundRect(x, y, b, b, float64(w)-2*b, float64(h)-2*b, r-b) {
			return fill, true
		}
		return line, InRoundRect(x, y, 0, 0, float64(w), float64(h), r)
	})
}

func Circle(d int, fill, bg color.NRGBA) *image.NRGBA {
	r := float64(d) / 2
	return Render(d, d, 4, bg, func(x, y float64) (color.NRGBA, bool) {
		return fill, InEllipse(x, y, r, r, r, r)
	})
}

func Toggle(w, h, pad int, on bool, track, knob, line, bg color.NRGBA) *image.NRGBA {
	fw, fh, fp := float64(w), float64(h), float64(pad)
	kr := (fh - 2*fp) / 2
	kx := fp + kr
	if on {
		kx = fw - fp - kr
	}
	return Render(w, h, 4, bg, func(x, y float64) (color.NRGBA, bool) {
		if InEllipse(x, y, kx, fh/2, kr, kr) {
			return knob, true
		}
		if InRoundRect(x, y, 1.5, 1.5, fw-3, fh-3, (fh-3)/2) {
			return track, true
		}
		return line, InRoundRect(x, y, 0, 0, fw, fh, fh/2)
	})
}
