package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
)

func read(p string) (image.Image, int, int, string) {
	bytes, e := os.ReadFile(p)
	if e != nil {
		panic(e)
	}
	f, e := os.Open(p)
	if e != nil {
		panic(e)
	}
	defer f.Close()
	im, _, e := image.Decode(f)
	if e != nil {
		panic(e)
	}
	w, h := im.Bounds().Dx(), im.Bounds().Dy()
	scale, top := 0, 0
	switch {
	case w == 640 && h == 417:
		scale, top = 2, 17
	case w%320 == 0 && h%200 == 0 && w/320 == h/200 && w/320 >= 1 && w/320 <= 4:
		scale = w / 320
	default:
		panic(fmt.Sprintf("拒絕未驗證尺寸%d×%d", w, h))
	}
	return im, scale, top, fmt.Sprintf("%x", sha256.Sum256(bytes))
}
func rgb(im image.Image, x, y int) [3]uint32 {
	r, g, b, _ := im.At(x, y).RGBA()
	return [3]uint32{r >> 8, g >> 8, b >> 8}
}
func main() {
	if len(os.Args) != 4 {
		panic("兩張原始PNG及upper/lower")
	}
	a, as, at, ah := read(os.Args[1])
	b, bs, bt, bh := read(os.Args[2])
	by := 112
	if os.Args[3] == "upper" {
		by = 2
	} else if os.Args[3] != "lower" {
		panic("upper/lower")
	}
	n, maxd := 0, 0
	uniform := [2]int{}
	bounds := [4]int{320, 200, -1, -1}
	regions := map[string]int{}
	locations := [][2]int{}
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			ac, bc := rgb(a, x*as, y*as+at), rgb(b, x*bs, y*bs+bt)
			for k, im := range []image.Image{a, b} {
				sc, tp := as, at
				if k == 1 {
					sc, tp = bs, bt
				}
				base := ac
				if k == 1 {
					base = bc
				}
				for dy := 0; dy < sc; dy++ {
					for dx := 0; dx < sc; dx++ {
						if rgb(im, x*sc+dx, y*sc+tp+dy) != base {
							uniform[k]++
						}
					}
				}
			}
			if ac != bc {
				n++
				locations = append(locations, [2]int{x, y})
				if x < bounds[0] {
					bounds[0] = x
				}
				if y < bounds[1] {
					bounds[1] = y
				}
				if x > bounds[2] {
					bounds[2] = x
				}
				if y > bounds[3] {
					bounds[3] = y
				}
				region := "outside_dialogue"
				if x >= 5 && x < 315 && y >= by && y < by+86 {
					region = "dialogue"
					if (by == 112 && x < 88) || (by == 2 && x >= 232) {
						region = "portrait_zone"
					}
				}
				regions[region]++
			}
			for j := 0; j < 3; j++ {
				v := int(ac[j]) - int(bc[j])
				if v < 0 {
					v = -v
				}
				if v > maxd {
					maxd = v
				}
			}
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(map[string]any{"a": os.Args[1], "a_sha256": ah, "a_scale": as, "a_external_top": at, "b": os.Args[2], "b_sha256": bh, "b_scale": bs, "b_external_top": bt, "pixels": 64000, "different_pixels": n, "different_coordinates": locations, "maximum_channel_delta": maxd, "nonuniform_scaled_pixels": uniform, "bounds": bounds, "regions": regions, "classification": "完整遊戲內容比較，區域只作差異歸因；非完整機器同狀態"})
}
