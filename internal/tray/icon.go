package tray

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
)

const iconSize = 32

var zoneColors = map[string]color.NRGBA{
	"silent": {0x9c, 0xa3, 0xaf, 0xff},
	"safe":   {0x22, 0xc5, 0x5e, 0xff},
	"warn":   {0xf5, 0x9e, 0x0b, 0xff},
	"danger": {0xef, 0x44, 0x44, 0xff},
}

// makeIcon menggambar ikon tray: lingkaran berwarna sesuai zona dengan tiga
// bar meter putih. filled=false (limiter mati) → hanya cincin.
// Hasilnya file .ico berisi satu PNG 32×32 (didukung Windows Vista+).
func makeIcon(zone string, filled bool) []byte {
	c, ok := zoneColors[zone]
	if !ok {
		c = zoneColors["silent"]
	}
	white := color.NRGBA{0xff, 0xff, 0xff, 0xff}

	img := image.NewNRGBA(image.Rect(0, 0, iconSize, iconSize))
	const ss = 4 // supersampling untuk tepi halus
	center := float64(iconSize) / 2
	radius := center - 1
	ring := 3.5

	bars := []struct{ x0, x1, y0 float64 }{
		{9.5, 12.5, 17}, {14.5, 17.5, 12}, {19.5, 22.5, 8},
	}
	const barBottom = 23.0

	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			var cov, barCov float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := float64(x) + (float64(sx)+0.5)/ss
					py := float64(y) + (float64(sy)+0.5)/ss
					d := math.Hypot(px-center, py-center)
					if d > radius {
						continue
					}
					inBar := false
					for _, b := range bars {
						if px >= b.x0 && px <= b.x1 && py >= b.y0 && py <= barBottom {
							inBar = true
						}
					}
					switch {
					case filled && inBar:
						barCov++
					case filled || d >= radius-ring:
						cov++
					case inBar:
						barCov++
					}
				}
			}
			n := float64(ss * ss)
			if cov == 0 && barCov == 0 {
				continue
			}
			// Bar putih di atas lingkaran berwarna (atau langsung di atas transparan kalau cincin).
			src := c
			if barCov > cov {
				src = white
				if !filled {
					src = c
				}
			}
			a := (cov + barCov) / n
			img.SetNRGBA(x, y, color.NRGBA{src.R, src.G, src.B, uint8(math.Round(a * 255))})
		}
	}

	var pngBuf bytes.Buffer
	_ = png.Encode(&pngBuf, img)
	return wrapICO(pngBuf.Bytes())
}

// wrapICO membungkus satu PNG menjadi file .ico.
func wrapICO(pngData []byte) []byte {
	var b bytes.Buffer
	// ICONDIR
	_ = binary.Write(&b, binary.LittleEndian, struct{ Reserved, Type, Count uint16 }{0, 1, 1})
	// ICONDIRENTRY
	_ = binary.Write(&b, binary.LittleEndian, struct {
		Width, Height, Colors, Reserved uint8
		Planes, BitCount                uint16
		Size, Offset                    uint32
	}{iconSize, iconSize, 0, 0, 1, 32, uint32(len(pngData)), 22})
	b.Write(pngData)
	return b.Bytes()
}
