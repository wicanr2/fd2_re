package main

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type nativeFacilityListPresentation struct {
	variant, resource     int
	mode, locale          string
	closing               bool
	font                  *Font
	palette               [256]color.RGBA
	source, final         []byte
	sourceHD, finalHD     []byte
	sourceRuns, finalRuns []modernHDTextRun
	sourceMask, finalMask []byte
	rawFrames             [][]byte
	panel                 fdother.LMI1Entry
	comparison            [5]fdother.LMI1Entry
	chrome                *modernHDDialogueChrome
	roster                *modernFacilityRoster
}

type modernFacilityListRoster struct {
	job   *nativeClassUIJob
	frame int
	theme *modernStoryPortraitSet
	units []modernFacilityRosterUnit
	shift int
	clip  image.Rectangle
}

func captureModernFacilityListText(g *Game, compose func() ([]byte, bool)) ([]byte, []modernHDTextRun, bool) {
	previousFrame, previousIndexed := g.modernHDText, g.font.indexedHD
	g.modernHDText = &modernHDTextFrame{}
	defer func() { g.modernHDText, g.font.indexedHD = previousFrame, previousIndexed }()
	capture, restoreClass := g.beginModernHDFacilityText()
	defer restoreClass()
	if capture == nil {
		return nil, nil, false
	}
	g.font.indexedHD = capture
	restoreDigits := g.beginModernHDShopDigits()
	defer restoreDigits()
	frame, ok := compose()
	return frame, append([]modernHDTextRun(nil), capture.runs...), ok
}

func (g *Game) composeModernShopListMarker(fill byte, compose func() ([]byte, bool)) ([]byte, bool) {
	original := g.nativeShopUI
	copyUI := *original
	copyUI.shops = make(map[int]*campaign.NativeShopAssets, len(original.shops))
	for k, v := range original.shops {
		copyUI.shops[k] = v
	}
	assets := *original.shops[g.nativeShopVariant]
	assets.Background = bytes.Repeat([]byte{fill}, 64000)
	copyUI.shops[g.nativeShopVariant] = &assets
	copyUI.portraits = make(map[int]dato.Frame, len(original.portraits))
	for k, v := range original.portraits {
		copyUI.portraits[k] = v
	}
	copyUI.portraits[g.nativeShopVariant] = dato.Frame{Width: 80, Height: 80, Pixels: bytes.Repeat([]byte{fill}, 6400)}
	g.nativeShopUI = &copyUI
	defer func() { g.nativeShopUI = original }()
	return compose()
}

func (g *Game) freezeModernShopListWriter(frame []byte, compose func() ([]byte, bool)) ([]byte, bool) {
	a, okA := g.composeModernShopListMarker(0, compose)
	b, okB := g.composeModernShopListMarker(255, compose)
	if !okA || !okB || len(frame) != 64000 || len(a) != 64000 || len(b) != 64000 {
		return nil, false
	}
	mask := make([]byte, 64000)
	for i := range mask {
		if a[i] == b[i] {
			if a[i] != frame[i] {
				return nil, false
			}
			mask[i] = 255
		}
	}
	return mask, true
}

// This producer is called before publishing the original job. Its optional
// presentation recipe never changes the native buffers or continuation.
func (g *Game) bindModernShopRosterList(job *nativeClassUIJob, source, final []byte, closing bool) *nativeClassUIJob {
	if job == nil || g.nativeClassUI == nil || g.nativeShopUI == nil || g.font == nil || g.font.sf == nil || g.font.indexedHD != nil {
		return job
	}
	var compose func() ([]byte, bool)
	switch g.nativeShopMode {
	case "recipient_consumable", "recipient_equipment":
		compose = g.composeNativeShopRecipient
	case "sell_roster":
		compose = g.composeNativeShopSellRoster
	case "equip_roster":
		compose = g.composeNativeShopEquipRoster
	case "transfer_source", "transfer_dest":
		compose = g.composeNativeShopTransferRoster
	default:
		return job
	}
	roster := g.prepareNativeFacilityRoster(final)
	assets, _, _, ok := g.nativeShopState()
	palette, paletteOK := preparationCursorPalette(g.nativeClassUI.palette)
	if roster == nil || !ok || assets == nil || !paletteOK || len(source) != 64000 {
		return job
	}
	if rebuilt, ok := g.composeNativeShopStable(); !ok || !bytes.Equal(source, rebuilt) {
		return job
	}
	if rebuilt, ok := compose(); !ok || !bytes.Equal(final, rebuilt) {
		return job
	}
	d := &nativeFacilityListPresentation{variant: g.nativeShopVariant, resource: assets.ResourceID, mode: g.nativeShopMode, locale: g.localeID, closing: closing, font: g.font,
		source: append([]byte(nil), source...), final: append([]byte(nil), final...), panel: assets.Panel, comparison: assets.CompareCells, roster: roster}
	copy(d.palette[:], palette)
	var err error
	if closing {
		d.rawFrames, err = campaign.NativeClassListClosingFrames(d.source, d.final)
	} else {
		d.rawFrames, err = campaign.NativeClassListOpeningFrames(d.source, d.final)
	}
	if err != nil || len(d.rawFrames) != len(job.frames) || closing && !bytes.Equal(job.restore, d.source) {
		return job
	}
	for i := range d.rawFrames {
		if !bytes.Equal(d.rawFrames[i], job.frames[i]) {
			return job
		}
	}
	d.sourceMask, ok = g.freezeModernShopListWriter(source, g.composeNativeShopStable)
	if !ok {
		return job
	}
	d.finalMask, ok = g.freezeModernShopListWriter(final, compose)
	if !ok {
		return job
	}
	d.sourceHD, d.sourceRuns, ok = captureModernFacilityListText(g, g.composeNativeShopStable)
	if !ok {
		return job
	}
	d.finalHD, d.finalRuns, ok = captureModernFacilityListText(g, compose)
	if !ok {
		return job
	}
	for i := range source {
		if d.sourceMask[i] == 0 && d.sourceHD[i] != source[i] || d.finalMask[i] == 0 && d.finalHD[i] != final[i] {
			return job
		}
	}
	h, err := g.prepareModernHDFacilityDialogue()
	if err != nil || !facilityDialogueBorderVisible(source, h.borders[1]) || !facilityDialogueBorderVisible(final, h.borders[0]) {
		return job
	}
	d.chrome = h.chrome
	// Raw LMI slices outlive this job; own the small mutable source values.
	d.panel.Pixels = append([]byte(nil), d.panel.Pixels...)
	for i := range d.comparison {
		d.comparison[i].Pixels = append([]byte(nil), d.comparison[i].Pixels...)
	}
	job.facilityList = d
	return job
}

func outsideFacilityList(rect, cover image.Rectangle) []image.Rectangle {
	i := rect.Intersect(cover)
	if i.Empty() {
		return []image.Rectangle{rect}
	}
	var result []image.Rectangle
	for _, r := range []image.Rectangle{
		image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, i.Min.Y), image.Rect(rect.Min.X, i.Max.Y, rect.Max.X, rect.Max.Y),
		image.Rect(rect.Min.X, i.Min.Y, i.Min.X, i.Max.Y), image.Rect(i.Max.X, i.Min.Y, rect.Max.X, i.Max.Y),
	} {
		if !r.Empty() {
			result = append(result, r)
		}
	}
	return result
}

func (d *nativeFacilityListPresentation) placement(index int) (shift int, rect image.Rectangle, ok bool) {
	if d == nil || index < 0 || index > len(d.rawFrames) {
		return
	}
	if index == len(d.rawFrames) {
		return 0, image.Rectangle{}, d.closing
	}
	shift = (5 - index) * 13
	if d.closing {
		shift = (index + 1) * 13
	}
	return shift, image.Rect(5, 112+shift, 315, 198+shift).Intersect(image.Rect(0, 0, 320, 200)), true
}

func (g *Game) drawModernHDFacilityList(screen *ebiten.Image, job *nativeClassUIJob) bool {
	g.facilityListRosterHD = nil
	if job == nil || g.nativeClassUI == nil {
		return false
	}
	d := job.facilityList
	if d == nil || g.nativeShopUIJob != job || g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != "shop" ||
		g.camp.Node().NativeHubVariant != d.variant || g.nativeShopVariant != d.variant || g.nativeShopMode != d.mode ||
		g.localeID != d.locale || g.font != d.font || g.helpVisible || g.transientUI || g.nativePlayerStatus != nil || g.nativeFullDACBlack || g.nativeFullDACWhite {
		return false
	}
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok || !slices.Equal(palette, d.palette[:]) {
		return false
	}
	shift, rect, ok := d.placement(job.frame)
	if !ok {
		return false
	}
	if job.frame < len(d.rawFrames) {
		if len(job.frames) != len(d.rawFrames) || !bytes.Equal(job.frames[job.frame], d.rawFrames[job.frame]) {
			return false
		}
	} else if !bytes.Equal(job.restore, d.source) {
		return false
	}
	art, ui := g.modernStoryPortraits != nil, g.modernHDText != nil
	if !art && !ui {
		return false
	}
	if art {
		for _, unit := range d.roster.units {
			frames := g.modernStoryPortraits.mapSpritesHD[unit.group]
			if unit.frame >= len(frames) || frames[unit.frame] == nil || frames[unit.frame].Bounds() != image.Rect(0, 0, 144, 144) {
				return false
			}
		}
	}
	source, final := d.source, d.final
	if ui {
		source, final = d.sourceHD, d.finalHD
	}
	if art && !rect.Empty() {
		final = append([]byte(nil), final...)
		for _, unit := range d.roster.units {
			for y := 0; y < 24; y++ {
				at := (unit.origin.Y+y)*320 + unit.origin.X
				start := (unit.origin.Y+y-112)*310 + unit.origin.X - 5
				copy(final[at:at+24], d.panel.Pixels[start:start+24])
			}
		}
	}
	frame := source
	mask := d.sourceMask
	if !rect.Empty() {
		frame = append([]byte(nil), source...)
		mask = append([]byte(nil), d.sourceMask...)
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			from := (y-shift)*320 + 5
			to := y*320 + 5
			copy(frame[to:to+310], final[from:from+310])
			copy(mask[to:to+310], d.finalMask[from:from+310])
		}
	}
	if art {
		scene := g.modernFacility(d.resource)
		if scene == nil || !g.presentModernFacilityFrame(screen, frame, mask, g.nativeClassUI.palette, &modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 1920, 1200)}) {
			return false
		}
		if !rect.Empty() {
			g.facilityListRosterHD = &modernFacilityListRoster{job: job, frame: job.frame, theme: g.modernStoryPortraits, units: d.roster.units, shift: shift, clip: rect}
		}
	} else {
		g.presentNativeClassFrame(screen, frame)
	}
	if ui {
		cover := image.Rectangle{Min: rect.Min.Mul(2), Max: rect.Max.Mul(2)}
		for _, run := range d.sourceRuns {
			run = run.freezeLayout()
			for _, clip := range outsideFacilityList(run.clip, cover) {
				copyRun := run
				copyRun.clip = clip
				g.modernHDText.runs = append(g.modernHDText.runs, copyRun)
			}
		}
		for _, clip := range outsideFacilityList(image.Rect(5, 112, 315, 198), rect) {
			g.modernHDText.panels = append(g.modernHDText.panels, modernHDDialoguePanel{chrome: d.chrome, rect: image.Rect(5, 112, 315, 198), clip: clip, gold: true})
		}
		if !rect.Empty() {
			panel := image.Rect(5, 112, 315, 198)
			for _, run := range d.finalRuns {
				run = run.freezeLayout()
				clip := run.clip.Intersect(image.Rectangle{Min: panel.Min.Mul(2), Max: panel.Max.Mul(2)})
				if clip.Empty() {
					continue
				}
				run.y += float64(shift * 2)
				run.clip = clip.Add(image.Pt(0, shift*2)).Intersect(cover)
				if !run.clip.Empty() {
					g.modernHDText.runs = append(g.modernHDText.runs, run)
				}
			}
			g.modernHDText.panels = append(g.modernHDText.panels, modernHDDialoguePanel{chrome: d.chrome, rect: panel.Add(image.Pt(0, shift)), clip: rect})
			if d.mode == "recipient_equipment" {
				if h := g.prepareModernHDShopComparison(d.comparison); h != nil {
					for _, p := range shopComparisonPlacements(len(d.roster.source.ids)) {
						g.modernHDText.serviceIcons = append(g.modernHDText.serviceIcons, modernHDServiceIconPlacement{icon: h.icons[p.cell], origin: p.origin.Add(image.Pt(0, shift)), clip: rect})
					}
				}
			}
		}
	}
	return true
}

func (p *modernHDPresentation) drawModernHDFacilityListRoster(screen ebiten.FinalScreen, geometry ebiten.GeoM, brightness float64) {
	g := p.Game
	plan := g.facilityListRosterHD
	if plan == nil || plan.theme != g.modernStoryPortraits || g.nativeShopUIJob != plan.job || plan.job.frame != plan.frame {
		return
	}
	set := plan.theme
	if set.townCursorGPU == nil {
		set.townCursorGPU = make(map[[2]int]*ebiten.Image)
	}
	clip := image.Rectangle{Min: plan.clip.Min.Mul(6), Max: plan.clip.Max.Mul(6)}
	for _, unit := range plan.units {
		origin := unit.origin.Add(image.Pt(0, plan.shift)).Mul(6)
		rect := image.Rectangle{Min: origin, Max: origin.Add(image.Pt(144, 144))}
		visible := rect.Intersect(clip)
		if visible.Empty() {
			continue
		}
		key := [2]int{unit.group, unit.frame}
		picture := set.townCursorGPU[key]
		if picture == nil {
			picture = ebiten.NewImageFromImage(set.mapSpritesHD[unit.group][unit.frame])
			set.townCursorGPU[key] = picture
		}
		part := picture.SubImage(visible.Sub(rect.Min)).(*ebiten.Image)
		options := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		options.GeoM.Scale(1.0/3, 1.0/3)
		options.GeoM.Translate(float64(visible.Min.X)/3, float64(visible.Min.Y)/3)
		options.GeoM.Concat(geometry)
		value := float32(math.Max(0, math.Min(1, brightness)))
		options.ColorScale.Scale(value, value, value, 1)
		screen.DrawImage(part, options)
	}
}
