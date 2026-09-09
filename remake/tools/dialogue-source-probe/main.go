package main

import (
	"encoding/json"
	"fmt"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"image"
	_ "image/png"
	"os"
)

func read(p string) image.Image {
	f, e := os.Open(p)
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
}) uint32 {
	r, g, b, _ := c.RGBA()
	return r>>8<<16 | g>>8<<8 | b>>8
}
func sample(im image.Image, x, y int) uint32 {
	w, h := im.Bounds().Dx(), im.Bounds().Dy()
	s, top := 0, 0
	if w == 640 && h == 417 {
		s, top = 2, 17
	} else if w%320 == 0 && h%200 == 0 && w/320 == h/200 {
		s = w / 320
	} else {
		panic("unsupported geometry")
	}
	return rgb(im.At(x*s, y*s+top))
}
func main() {
	bank, e := fdicon.DecodeFile(os.Args[1])
	if e != nil {
		panic(e)
	}
	a := read(os.Args[2])
	b := read(os.Args[3])
	pal := b.(*image.Paletted).Palette
	colors := make([]uint32, 256)
	for i, c := range pal {
		colors[i] = rgb(c)
	}
	result := map[string]any{"sprite_count": len(bank.Sprites)}
	for _, pos := range [][2]int{{52, 22}, {124, 22}, {220, 22}, {76, 94}, {196, 94}, {28, -2}, {244, -2}} {
		matches := map[string]any{}
		for k, im := range []image.Image{a, b} {
			exact := []map[string]int{}
			best := map[string]int{"mismatches": 99999}
			for si, s := range bank.Sprites {
				for dy := -2; dy <= 2; dy++ {
					for dx := -2; dx <= 2; dx++ {
						n, bad := 0, 0
						for y := 0; y < 24; y++ {
							for x := 0; x < 24; x++ {
								i := y*24 + x
								if pos[1]+dy+y < 0 || pos[1]+dy+y >= 112 || s.Mask[i] == 0 {
									continue
								}
								n++
								px, py := pos[0]+dx+x, pos[1]+dy+y
								if sample(im, px, py) != colors[s.Pixels[i]] {
									bad++
								}
							}
						}
						if n < 100 && pos[1] >= 0 || n == 0 {
							continue
						}
						v := map[string]int{"sprite": si, "group": si / 12, "pose": si % 12 / 3, "cycle": si % 3, "x": pos[0] + dx, "y": pos[1] + dy, "opaque_pixels": n, "mismatches": bad}
						if bad == 0 {
							exact = append(exact, v)
						}
						if bad < best["mismatches"] {
							best = v
						}
					}
				}
			}
			matches[fmt.Sprint(k)] = map[string]any{"exact": exact, "best": best}
		}
		result[fmt.Sprint(pos)] = matches
	}

	frames, e := dato.DecodeResource(os.Args[4], 0)
	if e != nil {
		panic(e)
	}
	portraits := []map[string]int{}
	for fi, f := range frames {
		for k, im := range []image.Image{a, b} {
			for flip := 0; flip < 2; flip++ {
				bad := 0
				for y := 0; y < f.Height; y++ {
					for x := 0; x < f.Width; x++ {
						px := 8 + x
						if flip == 1 {
							px = 8 + f.Width - 1 - x
						}
						py := 115 + y
						if sample(im, px, py) != colors[f.Pixels[y*f.Width+x]] {
							bad++
						}
					}
				}
				portraits = append(portraits, map[string]int{"image": k, "frame": fi, "flip": flip, "width": f.Width, "height": f.Height, "mismatches": bad})
			}
		}
	}
	result["portraits"] = portraits
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}
