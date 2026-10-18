package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// This test-only executable compares the actual default Ebiten screen with our
// adapter in two separate GPU runs. It adds no player or state-injection flag.
func TestMain(m *testing.M) {
	mode := os.Getenv("FD2_TEST_SCREEN_FILTER_GPU")
	if mode == "" {
		os.Exit(m.Run())
	}
	w, err := strconv.Atoi(os.Getenv("FD2_TEST_SCREEN_WIDTH"))
	if err != nil {
		panic(err)
	}
	h, err := strconv.Atoi(os.Getenv("FD2_TEST_SCREEN_HEIGHT"))
	if err != nil {
		panic(err)
	}
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowTitle("FD2 screen filter GPU probe")
	ebiten.SetScreenFilterEnabled(os.Getenv("FD2_TEST_SCREEN_FILTER_OFF") != "1")
	pattern := image.NewRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			pattern.SetRGBA(x, y, color.RGBA{uint8(x*37 + y*11), uint8(x ^ y), uint8(x*13 - y*29), 255})
		}
	}
	g := &screenFilterGPUProbe{picture: ebiten.NewImageFromImage(pattern)}
	var game ebiten.Game = g
	if mode == "adapter" {
		game = &screenFilterGPUAdapter{screenFilterGPUProbe: g}
	}
	if mode != "adapter" && mode != "default" {
		panic("invalid GPU mode")
	}
	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

type screenFilterGPUProbe struct {
	picture *ebiten.Image
	frames  int
}

func (g *screenFilterGPUProbe) Update() error              { return nil }
func (g *screenFilterGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (g *screenFilterGPUProbe) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.picture, nil)
	g.frames++
	if g.frames == 30 {
		fmt.Println("GPU_READY")
	}
}

type screenFilterGPUAdapter struct {
	*screenFilterGPUProbe
	presentation modernHDPresentation
}

func (g *screenFilterGPUAdapter) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	g.presentation.drawDefaultFinalScreen(screen, offscreen, geometry)
}
