package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type hdQualityBattleProbe struct {
	gpu      modernHDBattleGPU
	plan     modernHDBattlePlan
	index    int
	err      error
	pixels   []byte
	mapGame  *Game
	mapFrame *modernHDMapFrame
}

func (p *hdQualityBattleProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *hdQualityBattleProbe) Update() error {
	if p.err != nil || p.index == 4 {
		return ebiten.Termination
	}
	return nil
}
func (p *hdQualityBattleProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == 4 {
		return
	}
	quality := []hdTextureQuality{hdTextureFull, hdTextureBalanced, hdTextureLight, hdTextureFull}[p.index]
	previous := p.gpu.World
	p.gpu.setQuality(quality)
	if previous != nil {
		bad := p.plan
		bad.Base = nil
		if p.gpu.draw(bad) == nil || p.gpu.World != previous {
			p.err = fmt.Errorf("pending quality discarded canvas before plan validation")
			return
		}
	}
	before := p.gpu.rendered
	if p.err = p.gpu.draw(p.plan); p.err != nil {
		return
	}
	if p.gpu.rendered != before+1 {
		p.err = fmt.Errorf("quality switch reused prior frame")
		return
	}
	size := []image.Point{{36, 24}, {24, 16}, {18, 12}, {36, 24}}[p.index]
	if got := p.gpu.sources["hd/quality-source"].Bounds().Size(); got != size {
		p.err = fmt.Errorf("quality %s texture %v want %v", quality.name(), got, size)
		return
	}
	readHDQualityWorldPixels(p.gpu.World, p.pixels)
	if p.gpu.World.Bounds() != quality.rasterBounds() {
		p.err = fmt.Errorf("battle raster density differs")
		return
	}
	for _, tc := range []struct {
		at  image.Point
		red bool
	}{{image.Pt(150, 210), true}, {image.Pt(440, 390), true}, {image.Pt(130, 210), false}, {image.Pt(452, 390), false}, {image.Pt(300, 402), false}, {image.Pt(300, 190), false}} {
		i := (tc.at.Y*quality.numerator()/6*p.gpu.World.Bounds().Dx() + tc.at.X*quality.numerator()/6) * 4
		want := byte(0)
		if tc.red {
			want = 255
		}
		if p.pixels[i] != want || p.pixels[i+1] != 0 || p.pixels[i+2] != 0 || p.pixels[i+3] != 255 {
			p.err = fmt.Errorf("quality %s changed placement/clip at %v: %v", quality.name(), tc.at, p.pixels[i:i+4])
			return
		}
	}
	p.mapGame.playerHDQuality = quality
	p.mapFrame.TransientTerrain = p.index%2 == 1
	if !p.mapGame.drawPreparedModernHDMapFrame(screen, p.mapFrame, false) {
		p.err = fmt.Errorf("map quality frame rejected")
		return
	}
	mapSize := []int{144, 96, 72, 144}[p.index]
	if got := p.mapGame.modernHDMapGPU.Units[0][0].Bounds().Size(); got != image.Pt(mapSize, mapSize) {
		p.err = fmt.Errorf("map texture size %v", got)
		return
	}
	readHDQualityWorldPixels(p.mapGame.modernHDMapGPU.World, p.pixels)
	if p.mapGame.modernHDMapGPU.World.Bounds() != quality.rasterBounds() {
		p.err = fmt.Errorf("map raster density differs")
		return
	}
	for _, at := range []image.Point{{190, 190}, {290, 290}, {179, 200}, {301, 200}, {200, 179}, {200, 301}} {
		i := (at.Y*quality.numerator()/6*p.mapGame.modernHDMapGPU.World.Bounds().Dx() + at.X*quality.numerator()/6) * 4
		r, b := byte(0), byte(255)
		if at.In(p.mapFrame.Viewport) {
			r, b = 255, 0
		}
		if p.pixels[i] != r || p.pixels[i+1] != 0 || p.pixels[i+2] != b || p.pixels[i+3] != 255 {
			p.err = fmt.Errorf("map %s changed known placement/clip at %v: %v", quality.name(), at, p.pixels[i:i+4])
			return
		}
	}
	p.index++
}

func TestHDTextureQualityBattleGeometryGPU(t *testing.T) {
	if os.Getenv("FD2_HD_QUALITY_BATTLE_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{0, 0, 0, 255}
	}
	clip := image.Rect(132, 192, 450, 400)
	plan := modernHDBattlePlan{Base: make([]byte, 64000), Palette: palette, Layers: []modernHDBattleLayer{{
		Native: figani.Frame{Width: 6, Height: 4}, HD: mapReusePicture(36, 24, color.RGBA{255, 0, 0, 255}), HDKey: "quality-source",
		Rect: image.Rect(120, 180, 480, 420), Clip: &clip, Fill: -1,
	}}}
	p := &hdQualityBattleProbe{plan: plan, pixels: make([]byte, 1920*1200*4)}
	p.mapGame = &Game{modernStoryPortraits: &modernStoryPortraitSet{mapSpritesHD: map[int][]image.Image{
		0: {mapReusePicture(144, 144, color.RGBA{255, 0, 0, 255})},
	}}}
	p.mapFrame = &modernHDMapFrame{Background: mapReusePicture(320, 200, color.RGBA{0, 0, 255, 255}), UI: image.NewRGBA(image.Rect(0, 0, 320, 200)),
		Units: []modernHDMapUnitDraw{{Group: 0, Rect: image.Rect(168, 168, 312, 312)}}, Viewport: image.Rect(180, 180, 300, 300)}
	defer p.gpu.close()
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 4 {
		t.Fatal(p.index, p.err)
	}
}
