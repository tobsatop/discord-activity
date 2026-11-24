// Package appicon draws the installer's app icon in code (no image tools
// needed in CI): a red-to-pink tile with a white pair of music notes and a
// small blurple "status" badge. It uses neither the YouTube Music nor the
// Discord logo (both are trademarks and the repo is public).
package appicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Shape gives the colour at a point; x and y run from 0 to 1.
type Shape func(x, y float64) color.NRGBA

// Render samples shape 4x4 times per pixel for smooth edges.
func Render(size int, f Shape) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	const ss = 4
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					c := f((float64(px)+(float64(sx)+0.5)/ss)/float64(size), (float64(py)+(float64(sy)+0.5)/ss)/float64(size))
					ca := float64(c.A) / 255
					r += float64(c.R) * ca
					g += float64(c.G) * ca
					b += float64(c.B) * ca
					a += ca
				}
			}
			if a == 0 {
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{
				R: uint8(r / a), G: uint8(g / a), B: uint8(b / a),
				A: uint8(math.Round(a / (ss * ss) * 255)),
			})
		}
	}
	return img
}

// PNG encodes an image.
func PNG(img image.Image) []byte {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func roundRect(x, y, x0, y0, x1, y1, r float64) bool {
	if x < x0 || x > x1 || y < y0 || y > y1 {
		return false
	}
	cx := math.Max(x0+r, math.Min(x, x1-r))
	cy := math.Max(y0+r, math.Min(y, y1-r))
	return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r
}

// ellipse with radii rx, ry, tilted by deg degrees.
func ellipse(x, y, cx, cy, rx, ry, deg float64) bool {
	t := deg * math.Pi / 180
	dx, dy := x-cx, y-cy
	u := dx*math.Cos(t) + dy*math.Sin(t)
	v := -dx*math.Sin(t) + dy*math.Cos(t)
	return (u*u)/(rx*rx)+(v*v)/(ry*ry) <= 1
}

// quad reports whether (x, y) is inside the convex polygon p (clockwise or not).
func quad(x, y float64, p [4][2]float64) bool {
	pos, neg := false, false
	for i := range p {
		a, b := p[i], p[(i+1)%4]
		c := (b[0]-a[0])*(y-a[1]) - (b[1]-a[1])*(x-a[0])
		pos = pos || c > 0
		neg = neg || c < 0
	}
	return !(pos && neg)
}

func mix(a, b color.NRGBA, t float64) color.NRGBA {
	t = math.Max(0, math.Min(1, t))
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), m(a.A, b.A)}
}

var transparent = color.NRGBA{}

// notes is a beamed pair of eighth notes in tile coordinates.
func notes(x, y float64) bool {
	const stemW = 0.054
	// note heads (tilted ellipses)
	if ellipse(x, y, 0.315, 0.665, 0.1, 0.074, -22) || ellipse(x, y, 0.595, 0.595, 0.1, 0.074, -22) {
		return true
	}
	// stems, on the right side of each head
	if x >= 0.39 && x <= 0.39+stemW && y >= 0.255 && y <= 0.65 {
		return true
	}
	if x >= 0.67 && x <= 0.67+stemW && y >= 0.185 && y <= 0.58 {
		return true
	}
	// beam joining the stem tops, slanted like the heads
	return quad(x, y, [4][2]float64{{0.39, 0.255}, {0.724, 0.185}, {0.724, 0.305}, {0.39, 0.375}})
}

// App is the installer icon. margin is the empty border around the tile as a
// fraction of the size (macOS icons leave about 10%, Windows icons none).
func App(margin float64) Shape {
	top := color.NRGBA{0xFF, 0x5A, 0x7A, 255}
	bottom := color.NRGBA{0xE0, 0x10, 0x3C, 255}
	white := color.NRGBA{255, 255, 255, 255}
	blurple := color.NRGBA{0x58, 0x65, 0xF2, 255}
	lo, hi := margin, 1-margin
	return func(x, y float64) color.NRGBA {
		if !roundRect(x, y, lo, lo, hi, hi, 0.225*(hi-lo)) {
			return transparent
		}
		tx, ty := (x-lo)/(hi-lo), (y-lo)/(hi-lo)
		c := mix(top, bottom, ty*0.9+tx*0.1)
		// a soft highlight across the top
		if ty < 0.5 {
			c = mix(c, white, (0.5-ty)*0.18)
		}
		// soft shadow under the notes
		if notes(tx-0.008, ty-0.014) && !notes(tx, ty) {
			c = mix(c, color.NRGBA{A: 255}, 0.14)
		}
		if notes(tx, ty) {
			c = white
		}
		// status badge: blurple circle with a white ring and three dots
		bx, by := 0.8, 0.8
		d := math.Hypot(tx-bx, ty-by)
		switch {
		case d <= 0.118:
			c = blurple
			for _, dx := range []float64{-0.046, 0, 0.046} {
				if math.Hypot(tx-(bx+dx), ty-by) <= 0.021 {
					c = white
				}
			}
		case d <= 0.148:
			c = white
		}
		return c
	}
}
