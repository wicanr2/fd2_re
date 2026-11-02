package main

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type nativeServiceMenuPresentation struct {
	owner, locale     string
	variant, resource int
	closing           bool
	font              *Font
	palette           [256]color.RGBA
	source            modernChurchConfirmationSource
	cells             [4]fdother.RawCell
	rawFrames         [][]byte
	chrome            *modernHDDialogueChrome
	shopJob           *nativeClassUIJob
	churchJob         *nativeChurchUIJob
	iconSources       [4]*modernHDServiceIcon
	icons             [4]*modernHDServiceIcon
	lastDraw          *modernChurchConfirmationDraw
}

func (g *Game) bindModernFacilityServiceMenu(shopJob *nativeClassUIJob, churchJob *nativeChurchUIJob, closing bool) *nativeServiceMenuPresentation {
	if g.nativeClassUI == nil || g.font == nil || g.font.sf == nil || g.font.indexedHD != nil || g.camp == nil || g.camp.Node() == nil ||
		g.nativeShopUIJob != nil || g.nativeClassUIJob != nil || g.nativeChurchUIJob != nil {
		return nil
	}
	d := &nativeServiceMenuPresentation{owner: g.camp.Node().Type, locale: g.localeID, font: g.font, closing: closing, shopJob: shopJob, churchJob: churchJob}
	var compose func() ([]byte, bool)
	var frames [][]byte
	var restore []byte
	switch d.owner {
	case "shop":
		a, _, _, ok := g.nativeShopState()
		if !ok || a == nil || shopJob == nil || churchJob != nil || g.nativeShopMode != "menu" || g.camp.Node().NativeHubVariant != g.nativeShopVariant ||
			a.ResourceID != map[int]int{1: 12, 3: 29, 5: 63}[g.nativeShopVariant] {
			return nil
		}
		d.variant, d.resource = g.nativeShopVariant, a.ResourceID
		compose = g.composeNativeShopStable
		for i := range d.cells {
			d.cells[i] = a.ServiceCells[i][0]
		}
		frames, restore = shopJob.frames, shopJob.restore
	case "church":
		if churchJob == nil || shopJob != nil || g.churchMode != "menu" || len(g.nativeClassUI.entries) <= 10 {
			return nil
		}
		d.resource = 14
		compose = g.composeNativeChurchSourceFrame
		for i := range d.cells {
			c := g.nativeClassUI.entries[3+2*i]
			d.cells[i] = fdother.RawCell{Width: c.Width, Height: c.Height, Pixels: c.Pixels}
		}
		frames, restore = churchJob.frames, churchJob.restore
	default:
		return nil
	}
	for i, c := range d.cells {
		if c.Width != 24 || c.Height != 20 || len(c.Pixels) != 480 {
			return nil
		}
		d.cells[i].Pixels = append([]byte(nil), c.Pixels...)
	}
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok {
		return nil
	}
	copy(d.palette[:], palette)
	d.source, ok = g.captureModernFacilityConfirmationSource(d.owner, compose)
	if !ok {
		return nil
	}
	h, err := g.prepareModernHDFacilityDialogue()
	if err != nil || !facilityDialogueBorderVisible(d.source.raw, h.borders[1]) {
		return nil
	}
	d.chrome = h.chrome
	if len(frames) != 4 || closing && !bytes.Equal(restore, d.source.raw) || !closing && len(restore) != 0 {
		return nil
	}
	for i := range frames {
		raw, _ := d.compose(i, false)
		if !bytes.Equal(raw.pixels, frames[i]) {
			return nil
		}
		d.rawFrames = append(d.rawFrames, append([]byte(nil), frames[i]...))
	}
	return d
}

// All four entries are normal cells. Preserve native signed division and order.
func (d *nativeServiceMenuPresentation) compose(index int, hd bool) (p modernChurchConfirmationFrame, ok bool) {
	if d == nil || index < 0 || index > 4 || index == 4 && !d.closing {
		return
	}
	p.pixels, p.mask = d.source.raw, d.source.mask
	if hd {
		p.pixels = d.source.hd
		p.runs = d.source.runs
		p.panels = []modernHDDialoguePanel{{chrome: d.chrome, rect: image.Rect(5, 112, 315, 198), gold: true}}
	}
	p.name = "restore"
	if index == 4 {
		return p, true
	}
	p.name = "menu"
	p.pixels = append([]byte(nil), p.pixels...)
	p.mask = append([]byte(nil), p.mask...)
	divisor := 4 - index
	if d.closing {
		divisor = index + 1
	}
	for option, offset := range []int{-39, -13, 13, 39} {
		cell := d.cells[option]
		x0 := 240 + offset/divisor
		if cell.BlitAt(p.pixels, 320, x0, 169) != nil {
			return p, false
		}
		for y := 0; y < 20; y++ {
			for x := 0; x < 24; x++ {
				if cell.Pixels[y*24+x] != 0 {
					p.mask[(169+y)*320+x0+x] = 255
				}
			}
		}
	}
	return p, true
}

func (g *Game) modernServiceMenuIcons(d *nativeServiceMenuPresentation, index int) ([]modernHDServiceIconPlacement, bool) {
	if index == 4 {
		return nil, true
	}
	set := g.interfaceAssetTheme()
	if set == nil || len(set.uiServiceIcons) != 40 || d.palette[74] != (color.RGBA{40, 69, 138, 255}) {
		return nil, false
	}
	divisor := 4 - index
	if d.closing {
		divisor = index + 1
	}
	var placements []modernHDServiceIconPlacement
	for option, offset := range []int{-39, -13, 13, 39} {
		c := d.cells[option]
		source := set.uiServiceIcons[[2]int{d.resource, 3 + 2*option}]
		if source == nil || source.picture == nil || source.native != modernUIRawDescription(3+2*option, c) ||
			source.picture.Bounds() != image.Rect(0, 0, 144, 120) || sha256.Sum256(source.picture.Pix) != source.digest {
			return nil, false
		}
		if d.iconSources[option] != source || d.icons[option] == nil {
			// Derive coverage from literal index-zero writes, never from image colors.
			picture := image.NewRGBA(source.picture.Bounds())
			copy(picture.Pix, source.picture.Pix)
			for y := 0; y < 120; y++ {
				for x := 0; x < 144; x++ {
					if c.Pixels[(y/6)*24+x/6] == 0 {
						at := picture.PixOffset(x, y)
						clear(picture.Pix[at : at+4])
					}
				}
			}
			d.iconSources[option] = source
			d.icons[option] = &modernHDServiceIcon{native: source.native, picture: picture, digest: sha256.Sum256(picture.Pix)}
		}
		if sha256.Sum256(d.icons[option].picture.Pix) != d.icons[option].digest {
			return nil, false
		}
		placements = append(placements, modernHDServiceIconPlacement{icon: d.icons[option], origin: image.Pt(240+offset/divisor, 169)})
	}
	return placements, true
}

func (g *Game) drawModernHDFacilityServiceMenu(screen *ebiten.Image, d *nativeServiceMenuPresentation) bool {
	if d == nil {
		return false
	}
	d.lastDraw = nil
	if g.nativeClassUI == nil || g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != d.owner ||
		g.localeID != d.locale || g.font != d.font || g.nativeClassUIJob != nil || g.helpVisible || g.transientUI || g.nativePlayerStatus != nil ||
		g.nativeFullDACBlack || g.nativeFullDACWhite || d.chrome == nil {
		return false
	}
	var frames [][]byte
	var restore []byte
	var index int
	if d.owner == "shop" {
		job := g.nativeShopUIJob
		if job == nil || job != d.shopJob || job.serviceMenu != d || g.nativeChurchUIJob != nil || g.nativeShopMode != "menu" ||
			g.nativeShopVariant != d.variant || g.camp.Node().NativeHubVariant != d.variant {
			return false
		}
		frames, restore, index = job.frames, job.restore, job.frame
	} else if d.owner == "church" {
		job := g.nativeChurchUIJob
		if job == nil || job != d.churchJob || job.serviceMenu != d || g.nativeShopUIJob != nil || g.churchMode != "menu" {
			return false
		}
		frames, restore, index = job.frames, job.restore, job.frame
	} else {
		return false
	}
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok || !slices.Equal(palette, d.palette[:]) || len(frames) != 4 || index < 0 || index > 4 {
		return false
	}
	if index < 4 {
		if !bytes.Equal(frames[index], d.rawFrames[index]) {
			return false
		}
	} else if !d.closing || !bytes.Equal(restore, d.source.raw) {
		return false
	}
	ui, art := g.modernHDText != nil, g.modernStoryPortraits != nil
	if !ui && !art {
		return false
	}
	p, ok := d.compose(index, ui)
	if !ok {
		return false
	}
	var icons []modernHDServiceIconPlacement
	if ui {
		icons, ok = g.modernServiceMenuIcons(d, index)
		if !ok {
			return false
		}
	}
	if art {
		scene := g.modernFacility(d.resource)
		if scene == nil || !g.presentModernFacilityFrame(screen, p.pixels, p.mask, g.nativeClassUI.palette, &modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 1920, 1200)}) {
			return false
		}
	} else {
		g.presentNativeClassFrame(screen, p.pixels)
	}
	if ui {
		g.modernHDText.runs = append(g.modernHDText.runs, p.runs...)
		g.modernHDText.panels = append(g.modernHDText.panels, p.panels...)
		g.modernHDText.serviceIcons = icons
	}
	d.lastDraw = &modernChurchConfirmationDraw{frame: index, phase: p.name, interfaceHD: ui, artHD: art}
	return true
}
