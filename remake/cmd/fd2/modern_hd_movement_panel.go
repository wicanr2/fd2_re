package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"golang.org/x/image/font/sfnt"
)

type modernHDMovementPanel struct {
	actor                *battle.Unit
	plan                 *battle.NativePlayerMovement
	sourceImage          *ebiten.Image
	record, raw, palette []byte
	x                    int
	locale               string
	primary, font        *Font
	primarySF, fontSF    *sfnt.Font
	recipe               *modernBattlePanelText
	seal                 [32]byte
	gpu, gpuIdentity     *ebiten.Image
	pictureIdentity      *image.RGBA
	pictureKey           string
}
type modernHDMovementPanelDraw struct {
	image     *ebiten.Image
	rect      image.Rectangle
	hd, artHD bool
	key       string
}

func movementPanelSourceSeal(h *modernHDMovementPanel) [32]byte {
	if h == nil || h.recipe == nil {
		return [32]byte{}
	}
	p := h.recipe
	c := p.chrome
	raw, _ := json.Marshal([]any{h.record, h.raw, h.palette, h.x, h.locale, p.native.X, p.native.Y, p.native.Width, p.native.Height, p.native.Pixels, p.native.Mask, p.name, p.values, p.background, p.digitInk, c.ready, c.border, c.track, c.ink, c.width})
	return sha256.Sum256(raw)
}

func (g *Game) freezeModernHDMovementPanel(actor *battle.Unit, plan *battle.NativePlayerMovement, source *ebiten.Image, record, raw []byte, x int, panels []*modernBattlePanelText) (*modernHDMovementPanel, error) {
	if len(panels) == 0 {
		return nil, nil
	} // Legacy native-only owners have no verified font/name collector.
	if len(panels) != 1 || actor == nil || plan == nil || source == nil || len(record) < 0x48 || source.Bounds() != image.Rect(0, 0, 149, 42) || (x != 9 && x != 160) || g.font == nil || g.font.sf == nil {
		return nil, errors.New("modern HD movement panel: source unavailable")
	}
	p := panels[0]
	if p == nil || p.font == nil || p.font.sf == nil || p.native.Width != 149 || p.native.Height != 42 || len(raw) != 149*42 || !bytes.Equal(raw, p.native.Pixels) {
		return nil, errors.New("modern HD movement panel: source crop differs")
	}
	palette, err := modernHDBattlePalette(g.nativeUIPalette)
	if err != nil {
		return nil, err
	}
	h := &modernHDMovementPanel{actor: actor, plan: plan, sourceImage: source, record: append([]byte(nil), record...), raw: append([]byte(nil), raw...), palette: append([]byte(nil), palette...), x: x, locale: g.localeID, primary: g.font, primarySF: g.font.sf, font: p.font, fontSF: p.font.sf, recipe: p}
	h.seal = movementPanelSourceSeal(h)
	return h, nil
}

func (g *Game) prepareModernHDMovementPanel() (*modernHDMovementPanel, error) {
	h := g.movementPanelHD
	if h == nil || g.sel == nil || h.actor != g.sel || h.plan == nil || h.plan != g.nativeMovePlan || h.sourceImage == nil || h.sourceImage != g.nativeMovePanel || h.x != g.nativeMovePanelX ||
		g.font != h.primary || g.font == nil || g.font.sf != h.primarySF || h.recipe == nil || h.recipe.font != h.font || h.font == nil || h.font.sf != h.fontSF || h.fontSF == nil || h.seal != movementPanelSourceSeal(h) {
		return nil, errors.New("modern HD movement panel: changed source or owner")
	}
	palette, err := modernHDBattlePalette(g.nativeUIPalette)
	if err != nil || !bytes.Equal(palette, h.palette) {
		return nil, errors.New("modern HD movement panel: changed palette")
	}
	pic, key, err := h.recipe.raster(g.nativeUIPalette)
	if err != nil {
		return nil, err
	}
	rgba, ok := pic.(*image.RGBA)
	if !ok || rgba.Bounds() != image.Rect(0, 0, 894, 252) || key != fmt.Sprintf("panel-text/%x", sha256.Sum256(rgba.Pix)) {
		return nil, errors.New("modern HD movement panel: changed raster cache")
	}
	if h.gpu != nil && h.gpu != h.gpuIdentity {
		return nil, errors.New("modern HD movement panel: changed GPU identity")
	}
	if h.gpu == nil || h.pictureKey != key || h.pictureIdentity != rgba {
		h.gpu = ebiten.NewImageFromImage(rgba)
		h.gpuIdentity = h.gpu
		h.pictureIdentity = rgba
		h.pictureKey = key
	}
	return h, nil
}

func (g *Game) preflightModernHDMovementPanel() error {
	if g.titlePhase != "" || g.sel == nil || g.nativeMovePanel == nil {
		return nil
	}
	_, err := g.prepareModernHDMovementPanel()
	return err
}

func (p *modernHDPresentation) drawModernHDMovementPanel(screen ebiten.FinalScreen, geometry ebiten.GeoM, brightness float64) {
	d := p.Game.movementPanelDraw
	if d == nil || !d.hd || d.image == nil {
		return
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(float64(d.rect.Dx())/894, float64(d.rect.Dy())/252)
	op.GeoM.Translate(float64(d.rect.Min.X), float64(d.rect.Min.Y))
	op.GeoM.Concat(geometry)
	v := float32(math.Max(0, math.Min(1, brightness)))
	op.ColorScale.Scale(v, v, v, 1)
	screen.DrawImage(d.image, op)
}
