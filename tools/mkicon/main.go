package main

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"deepmole/internal/raster"
)

func main() {
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	var blobs [][]byte
	for _, s := range sizes {
		var buf bytes.Buffer
		_ = png.Encode(&buf, raster.Logo(s, color.NRGBA{}))
		blobs = append(blobs, buf.Bytes())
	}
	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, []uint16{0, 1, uint16(len(sizes))})
	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		dim := uint8(s)
		if s >= 256 {
			dim = 0
		}
		out.Write([]byte{dim, dim, 0, 0})
		_ = binary.Write(&out, binary.LittleEndian, []uint16{1, 32})
		_ = binary.Write(&out, binary.LittleEndian, []uint32{uint32(len(blobs[i])), uint32(offset)})
		offset += len(blobs[i])
	}
	for _, b := range blobs {
		out.Write(b)
	}
	dst := filepath.Join("assets", "icon.ico")
	if len(os.Args) > 1 {
		dst = os.Args[1]
	}
	if err := os.WriteFile(dst, out.Bytes(), 0o644); err != nil {
		panic(err)
	}
	_ = os.WriteFile(filepath.Join(filepath.Dir(dst), "icon-256.png"), blobs[len(blobs)-1], 0o644)
	_ = os.WriteFile(filepath.Join(filepath.Dir(dst), "icon-32.png"), blobs[2], 0o644)
}
