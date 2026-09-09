// 對正常玩家截圖的完整頭像區逐像素辨識來源畫格，不改寫圖片。
package main

import (
	"encoding/json"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

func read(path string) image.Image {
	f, e := os.Open(path)
	if e != nil {
		panic(e)
	}
	defer f.Close()
	im, _, e := image.Decode(f)
	if e != nil {
		panic(e)
	}
	return im
}
func rgb(c interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) uint32 { r, g, b, _ := c.RGBA(); return r>>8<<16 | g>>8<<8 | b>>8 }
func main() {
	palette := read(os.Args[1]).(*image.Paletted).Palette
	rows := []map[string]any{}
	paths, e := filepath.Glob(filepath.Join(os.Args[3], "*-flow-*.png"))
	if e != nil {
		panic(e)
	}
	for _, path := range paths {
		im := read(path)
		if im.Bounds().Dx() != 1280 || im.Bounds().Dy() != 800 {
			panic("unexpected screenshot size")
		}
		resource, x0, y0, flip := 0, 8, 115, true
		if strings.HasPrefix(filepath.Base(path), "second-") {
			resource, x0, y0, flip = 48, 232, 5, false
		}
		frames, e := dato.DecodeResource(os.Args[2], resource)
		if e != nil {
			panic(e)
		}
		counts := []int{}
		for _, f := range frames {
			bad := 0
			for y := 0; y < 80; y++ {
				for x := 0; x < 80; x++ {
					px := x
					if flip {
						px = 79 - x
					}
					if rgb(im.At((x0+px)*4, (y0+y)*4)) != rgb(palette[f.Pixels[y*80+x]]) {
						bad++
					}
				}
			}
			counts = append(counts, bad)
		}
		rows = append(rows, map[string]any{"file": filepath.Base(path), "resource": resource, "different_pixels_by_frame": counts})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(rows)
}
