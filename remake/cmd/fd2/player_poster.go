package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type playerPosterPresentation struct {
	catalogPath    string
	packRoot       string
	settings       playerPosterSettings
	pair           *playerPosterPair
	masked         [2]*ebiten.Image
	maskSet        string
	maskEdge       int
	started        time.Time
	clock          func() time.Time
	menuOpen       bool
	consumeInput   bool
	sliderMouse    bool
	sliderTouch    bool
	touchID        ebiten.TouchID
	clean          *ebiten.Image
	terrainBase    *ebiten.Image
	native         *ebiten.Image
	cleanFrame     *modernHDMapFrame
	nativeSource   *indexedmap.NativeFrameInput
	nativeDAC      [32]byte
	backdropSource string
	backdropError  string
}

func (p *modernHDPresentation) posterNow() time.Time {
	if p.poster.clock != nil {
		return p.poster.clock()
	}
	return time.Now()
}

func (p *modernHDPresentation) posterOffset() int {
	if p.poster.settings.Set != "" && p.poster.settings.Set != "off" && p.poster.pair != nil {
		return playerPosterWidth
	}
	return 0
}

func (p *modernHDPresentation) layoutWidth() int {
	return logicalW + 2*p.posterOffset()
}

func (p *modernHDPresentation) initPosters() {
	configured := normalizePlayerPosters(loadSettings().Poster)
	mode := configured.Set
	configured.Set = "off"
	p.poster.settings = configured
	if err := p.applyPosterMode(mode, false); err != nil {
		p.Game.msg = p.posterLabels().unavailable + err.Error()
	}
}

func (p *modernHDPresentation) savePosterSettings() {
	s := loadSettings()
	copy := p.poster.settings
	s.Poster = &copy
	saveSettings(s)
}

func (p *modernHDPresentation) applyPosterMode(mode string, persist bool) error {
	if !validPosterMode(mode) {
		return fmt.Errorf("posters: unsupported set %q", mode)
	}
	if mode == p.poster.settings.Set {
		return nil
	}
	pair := p.poster.pair
	if mode != "off" {
		root := p.poster.packRoot
		if root == "" {
			root = os.Getenv("FD2_MODERN_THEME_PACK")
		}
		if root == "" {
			root = assetPath("assets/themes/modern")
		}
		catalog := p.poster.catalogPath
		if catalog == "" {
			catalog = assetPath("assets/themes/modern/catalog.json")
		}
		var err error
		pair, err = loadPlayerPosterPair(catalog, root)
		if err != nil {
			return err
		}
	}
	p.poster.pair, p.poster.settings.Set = pair, mode
	p.poster.started = p.posterNow()
	if persist {
		p.savePosterSettings()
	}
	return nil
}

func (p *modernHDPresentation) cyclePosters() {
	next := "hd"
	switch p.poster.settings.Set {
	case "hd":
		next = "classic"
	case "classic":
		next = "off"
	}
	if err := p.applyPosterMode(next, true); err != nil {
		p.Game.msg = p.posterLabels().unavailable + err.Error()
	}
}

func (p *modernHDPresentation) posterAlpha() float64 {
	if p.poster.settings.FadeMS <= 0 || p.poster.started.IsZero() {
		return 1
	}
	t := float64(p.posterNow().Sub(p.poster.started)) / float64(time.Duration(p.poster.settings.FadeMS)*time.Millisecond)
	t = math.Max(0, math.Min(1, t))
	return t * t * (3 - 2*t)
}

func (p *modernHDPresentation) preparePosterMasks() {
	if p.poster.masked[0] != nil && p.poster.maskSet == p.poster.settings.Set && p.poster.maskEdge == p.poster.settings.Edge {
		return
	}
	index := 0
	if p.poster.settings.Set == "classic" {
		index = 1
	}
	for side := range p.poster.masked {
		if p.poster.masked[side] != nil {
			p.poster.masked[side].Dispose()
		}
		p.poster.masked[side] = ebiten.NewImageFromImage(maskPlayerPoster(p.poster.pair.Sides[index][side], side, p.poster.settings.Edge))
	}
	p.poster.maskSet, p.poster.maskEdge = p.poster.settings.Set, p.poster.settings.Edge
}

// Only established scenery layers feed the seam. The final game image is
// deliberately unavailable here: it contains text, people and HUD panels.
func (p *modernHDPresentation) cleanPosterBackground() (*ebiten.Image, image.Rectangle) {
	g := p.Game
	if o := g.modernHDScene; o != nil && o.scene != nil && g.modernStoryPortraits != nil {
		if o.scene.gpu == nil {
			o.scene.gpu = ebiten.NewImageFromImage(o.scene.image)
		}
		p.poster.backdropSource = "facility"
		return o.scene.gpu, o.scene.gpu.Bounds()
	}
	if g.modernHDWorldVisible && g.modernHDMapFrame != nil && g.modernHDMapGPU != nil {
		frame, gpu := g.modernHDMapFrame, g.modernHDMapGPU
		if p.poster.clean == nil {
			p.poster.clean = ebiten.NewImage(modernHDWidth, modernHDHeight)
		}
		if p.poster.cleanFrame != frame {
			base := frame.TerrainViewBase
			if base < 0 || base+191*fdicon.NativeMapStride+312 > len(frame.TerrainSeed) {
				p.poster.backdropSource = "neutral"
				return nil, image.Rectangle{}
			}
			picture := image.NewRGBA(image.Rect(0, 0, 320, 200))
			for y := 0; y < 192; y++ {
				for x := 0; x < 312; x++ {
					c := frame.TerrainPalette[frame.TerrainSeed[base+y*fdicon.NativeMapStride+x]]
					if c.A != 255 {
						p.poster.backdropSource = "neutral"
						return nil, image.Rectangle{}
					}
					picture.SetRGBA(x+4, y+4, c)
				}
			}
			if p.poster.terrainBase == nil {
				p.poster.terrainBase = ebiten.NewImage(320, 200)
			}
			p.poster.terrainBase.WritePixels(picture.Pix)
			p.poster.clean.Clear()
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(6, 6)
			p.poster.clean.DrawImage(p.poster.terrainBase, op)
			clip := modernHDMapViewport()
			for _, terrain := range frame.Terrain {
				op := &ebiten.DrawImageOptions{}
				if terrain.ScaleToRect {
					op.GeoM.Scale(float64(terrain.Rect.Dx())/float64(terrain.Image.Bounds().Dx()), float64(terrain.Rect.Dy())/float64(terrain.Image.Bounds().Dy()))
				}
				op.GeoM.Translate(float64(terrain.Rect.Min.X), float64(terrain.Rect.Min.Y))
				p.poster.clean.SubImage(clip).(*ebiten.Image).DrawImage(gpu.terrainImage(terrain.Image), op)
			}
			p.poster.cleanFrame = frame
		}
		p.poster.backdropSource = "hd-terrain"
		return p.poster.clean, modernHDMapViewport()
	}
	if g.nativeMapPresented && g.nativeMapFrameSource != nil {
		dac := sha256.Sum256(g.nativeMapDAC)
		if p.poster.nativeSource != g.nativeMapFrameSource || p.poster.nativeDAC != dac {
			if err := p.prepareNativePosterTerrain(); err != nil {
				p.poster.backdropError = err.Error()
				p.poster.backdropSource = "neutral"
				return nil, image.Rectangle{}
			}
			p.poster.nativeSource, p.poster.nativeDAC = g.nativeMapFrameSource, dac
		}
		p.poster.backdropSource = "native-terrain"
		return p.poster.native, image.Rect(4, 4, 316, 196)
	}
	p.poster.backdropSource = "neutral"
	return nil, image.Rectangle{}
}

func (p *modernHDPresentation) prepareNativePosterTerrain() error {
	g := p.Game
	if len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize {
		return fmt.Errorf("poster terrain: missing native work")
	}
	in := g.nativeMapFrameSource.Frame
	in.Cells = append([]fdicon.NativeTerrainCell(nil), in.Cells...)
	in.Units, in.ForegroundUnits, in.SelectionOverlay = nil, nil, nil
	in.RangeMode = 0
	work, vga := append([]byte(nil), g.nativeMapWork...), make([]byte, indexedmap.NativeMapVGASize)
	if err := indexedmap.ComposeFrameObserved(work, vga, in, func([]byte) error { return nil },
		func(indexedmap.FrameStage, []byte, []byte) error { return nil }); err != nil {
		return err
	}
	palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
		return err
	}
	picture := image.NewRGBA(image.Rect(0, 0, 320, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			picture.Set(x, y, palette[vga[y*320+x]])
		}
	}
	if p.poster.native == nil {
		p.poster.native = ebiten.NewImage(320, 200)
	}
	p.poster.native.WritePixels(picture.Pix)
	return nil
}

func (p *modernHDPresentation) drawPosters(screen ebiten.FinalScreen, geometry ebiten.GeoM) {
	left := p.posterOffset()
	if left == 0 {
		return
	}
	p.preparePosterMasks()
	background, bounds := p.cleanPosterBackground()
	top := p.toolbarOffset()
	alpha := float32(p.posterAlpha())
	for side, x := range []int{0, left + logicalW} {
		if background != nil {
			width := bounds.Dx() * left / logicalW
			x0 := bounds.Min.X
			if side == 1 {
				x0 = bounds.Max.X - width
			}
			part := background.SubImage(image.Rect(x0, bounds.Min.Y, x0+width, bounds.Max.Y)).(*ebiten.Image)
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Scale(-float64(left)/float64(width), float64(logicalH)/float64(bounds.Dy()))
			op.GeoM.Translate(float64(x+left), float64(top))
			op.GeoM.Concat(geometry)
			screen.DrawImage(part, op)
		} else {
			// No text-bearing framebuffer is ever used as a fallback.
			p.drawPosterNeutral(screen, geometry, image.Rect(x, top, x+left, top+logicalH))
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(float64(left)/296, float64(logicalH)/900)
		op.GeoM.Translate(float64(x), float64(top))
		op.GeoM.Concat(geometry)
		op.ColorScale.Scale(alpha, alpha, alpha, alpha)
		screen.DrawImage(p.poster.masked[side], op)
	}
}

func (p *modernHDPresentation) drawPosterNeutral(screen ebiten.FinalScreen, geometry ebiten.GeoM, rect image.Rectangle) {
	if p.posterPixel == nil {
		p.posterPixel = ebiten.NewImage(1, 1)
		p.posterPixel.Fill(color.RGBA{8, 19, 33, 255})
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(rect.Dx()), float64(rect.Dy()))
	op.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	op.GeoM.Concat(geometry)
	screen.DrawImage(p.posterPixel, op)
}
