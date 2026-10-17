// 可丟棄的 Ebiten 呈現原型。背景沿用已驗證場景，頭像直接讀母稿。
// 不啟動遊戲狀態，不修改正式 renderer、catalog 或存檔。
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

type prototype struct {
	background, portrait  *ebiten.Image
	width, height, frames int
	output                string
	done                  bool
}

func readPNG(path string) *ebiten.Image {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	im, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	return ebiten.NewImageFromImage(im)
}

func (p *prototype) Update() error {
	if p.done {
		return ebiten.Termination
	}
	return nil
}

func (p *prototype) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(p.width)/float64(p.background.Bounds().Dx()), float64(p.height)/float64(p.background.Bounds().Dy()))
	screen.DrawImage(p.background, op)
	// 原生320×200座標維持。頭像右上錨點(232,5)，區域80×80。
	scale := float64(p.width) / 320
	op = &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(80*scale/float64(p.portrait.Bounds().Dx()), 80*scale/float64(p.portrait.Bounds().Dy()))
	op.GeoM.Translate(232*scale, 5*scale)
	screen.DrawImage(p.portrait, op)
	p.frames++
	if p.frames == 3 {
		f, err := os.OpenFile(p.output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		if err := png.Encode(f, screen); err != nil {
			panic(err)
		}
		if err := f.Close(); err != nil {
			panic(err)
		}
		p.done = true
		fmt.Printf("prototype canvas=%dx%d portrait=%dx%d\n", p.width, p.height, int(80*scale), int(80*scale))
	}
}

func (p *prototype) Layout(int, int) (int, int) { return p.width, p.height }

func main() {
	w, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	if w != 1280 && w != 1920 {
		panic("unsupported prototype canvas")
	}
	p := &prototype{width: w, height: w * 5 / 8, output: os.Args[4], background: readPNG(os.Args[2]), portrait: readPNG(os.Args[3])}
	ebiten.SetWindowSize(960, 600)
	ebiten.SetWindowTitle("FD2 高解析原型")
	if err := ebiten.RunGame(p); err != nil {
		panic(err)
	}
}
