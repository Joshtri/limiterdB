package tray

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestMakeIconIsValidICO(t *testing.T) {
	for _, zone := range []string{"silent", "safe", "warn", "danger", "unknown"} {
		for _, filled := range []bool{true, false} {
			ico := makeIcon(zone, filled)
			if binary.LittleEndian.Uint16(ico[2:]) != 1 || binary.LittleEndian.Uint16(ico[4:]) != 1 {
				t.Fatalf("%s: bad ICONDIR", zone)
			}
			size := binary.LittleEndian.Uint32(ico[14:])
			offset := binary.LittleEndian.Uint32(ico[18:])
			if int(offset+size) != len(ico) {
				t.Fatalf("%s: entry size/offset mismatch", zone)
			}
			img, err := png.Decode(bytes.NewReader(ico[offset:]))
			if err != nil {
				t.Fatalf("%s: %v", zone, err)
			}
			if b := img.Bounds(); b.Dx() != iconSize || b.Dy() != iconSize {
				t.Fatalf("%s: size %v", zone, b)
			}

			// Opsional: simpan PNG buat dilihat manual.
			if dir := os.Getenv("LIMITERDB_ICON_DIR"); dir != "" {
				name := zone
				if !filled {
					name += "-off"
				}
				_ = os.WriteFile(filepath.Join(dir, name+".png"), ico[offset:], 0o644)
			}
		}
	}
}
