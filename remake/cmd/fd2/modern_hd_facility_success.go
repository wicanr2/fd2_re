package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type nativeFacilitySuccessPresentation struct {
	owner, locale     string
	resource, variant int
	font              *Font
	source            modernChurchConfirmationSource
	raw, hd           [][]byte // base, original animation entries, restored final
	native            []modernLMIHDDescriptor
	chrome            *modernHDDialogueChrome
	lastDraw          *modernChurchConfirmationDraw
}

type nativeFacilitySuccessStep struct {
	presentation   *nativeFacilitySuccessPresentation
	stage, delta   int // -1 = base; -2 = producer's final; otherwise animation entry
	duration       time.Duration
	frame, effects [32]byte
	palette        [256]color.RGBA
}

var facilitySuccessMaskSHA = map[int]string{
	12: "5097f3b6b028aff4894241596048da4d4b5644bd21bb034657511bbf0dc16d3d",
	29: "d7a0584afe942d1ce0065e1cd4ee75f2f33d12149886eab03463885a59250f3c",
	63: "10b104d0c0d4c9d195ac19212d092fdc156d88fe9d91554c768ab08dce063925",
	14: "6acf95f515743e1c0485d811685cbf247857ba14c40318f2142915fe89a666f6",
}

func facilitySuccessDescription(index int, f fdother.Frame) modernLMIHDDescriptor {
	s := sha256.Sum256(f.Indexed)
	return modernLMIHDDescriptor{Index: 23 + index, Width: f.Width, Height: f.Height, PixelsSHA256: hex.EncodeToString(s[:])}
}

// Called by the existing producers before a timeline is published. Nothing in
// this binding changes a frame, a delay, its ACK, or the transaction callback.
func (g *Game) bindModernFacilitySuccessTimeline(timeline []nativeClassUITimelineStep, resource int, stages, deltas []int) {
	if len(timeline) == 0 || len(stages) != len(timeline) || len(deltas) != len(timeline) || g.nativeClassUI == nil || g.font == nil || g.font.sf == nil ||
		g.font.indexedHD != nil || g.camp == nil || g.camp.Node() == nil {
		return
	}
	d := &nativeFacilitySuccessPresentation{resource: resource, font: g.font, locale: g.localeID, owner: g.camp.Node().Type}
	var compose func() ([]byte, bool)
	var original []fdother.Frame
	var build func([]byte) ([][]byte, []byte, error)
	if resource == 14 {
		if d.owner != "church" || g.churchMode != "revive_confirm" {
			return
		}
		compose = g.composeNativeChurchReviveQuestion
		original = g.nativeClassUI.reviveFX
		build = func(source []byte) ([][]byte, []byte, error) {
			return campaign.ComposeNativeChurchReviveSuccessFrames(source, original, g.nativeClassUI.portrait)
		}
	} else {
		a, portrait, portraitID, ok := g.nativeShopState()
		if !ok || a == nil || d.owner != "shop" || a.ResourceID != resource || g.camp.Node().NativeHubVariant != g.nativeShopVariant {
			return
		}
		d.variant = g.nativeShopVariant
		compose = g.composeNativeShopBare
		original = a.SuccessFrames
		build = func(source []byte) ([][]byte, []byte, error) {
			return campaign.ComposeNativeShopPurchaseSuccessFrames(source, a, portrait, portraitID, d.variant)
		}
	}
	count, w, h, _, _ := facilitySuccessShape(resource)
	if count == 0 || len(original) != count {
		return
	}
	for i, f := range original {
		s := sha256.Sum256(f.Mask)
		if f.Width != w || f.Height != h || len(f.Indexed) != w*h || len(f.Mask) != w*h || hex.EncodeToString(s[:]) != facilitySuccessMaskSHA[resource] {
			return
		}
		d.native = append(d.native, facilitySuccessDescription(i, f))
	}
	var ok bool
	d.source, ok = g.captureModernFacilityConfirmationSource(d.owner, compose)
	if !ok {
		return
	}
	for _, pair := range []struct {
		source []byte
		dest   *[][]byte
	}{{d.source.raw, &d.raw}, {d.source.hd, &d.hd}} {
		animation, final, err := build(pair.source)
		if err != nil || len(animation) != count {
			return
		}
		*pair.dest = append([][]byte{append([]byte(nil), pair.source...)}, animation...)
		*pair.dest = append(*pair.dest, final)
	}
	if resource == 14 {
		h, err := g.prepareModernHDFacilityDialogue()
		if err != nil {
			return
		}
		d.chrome = h.chrome
	}
	bindings := make([]*nativeFacilitySuccessStep, len(timeline))
	for i, step := range timeline {
		stage := stages[i]
		pos := facilitySuccessPosition(stage, count)
		palette, valid := preparationCursorPalette(step.palette)
		if pos < 0 || !valid || deltas[i] < 0 || deltas[i] > 62 || !bytes.Equal(step.frame, d.raw[pos]) {
			return
		}
		s := &nativeFacilitySuccessStep{presentation: d, stage: stage, delta: deltas[i], duration: step.duration, frame: sha256.Sum256(step.frame), effects: sha256.Sum256(step.shopEffectMask)}
		copy(s.palette[:], palette)
		bindings[i] = s
	}
	for i := range timeline {
		timeline[i].facilitySuccess = bindings[i]
	}
}

func facilitySuccessPosition(stage, count int) int {
	if stage == -1 {
		return 0
	}
	if stage == -2 {
		return count + 1
	}
	if stage >= 0 && stage < count {
		return stage + 1
	}
	return -1
}

// Gold-roll producers retain the exact restored face while changing only the
// existing 48x9 digit window. The caller still owns the gold commit and ACK.
func inheritModernFacilitySuccess(final nativeClassUITimelineStep, timeline []nativeClassUITimelineStep) {
	if final.facilitySuccess == nil || final.facilitySuccess.stage != -2 {
		return
	}
	for i := range timeline {
		target := &timeline[i]
		if len(target.frame) != 64000 || len(final.frame) != 64000 {
			return
		}
		for at := range final.frame {
			x, y := at%320, at/320
			if x >= 16 && x < 64 && y >= 98 && y < 107 {
				continue
			}
			if target.frame[at] != final.frame[at] {
				return
			}
		}
	}
	for i := range timeline {
		s := *final.facilitySuccess
		s.frame = sha256.Sum256(timeline[i].frame)
		s.effects = sha256.Sum256(timeline[i].shopEffectMask)
		s.duration = timeline[i].duration
		timeline[i].facilitySuccess = &s
	}
}

func facilitySuccessWhite(ink color.RGBA, delta int) color.RGBA {
	add := delta * 255 / 63
	channel := func(v uint8) uint8 {
		n := int(v) + add
		if n > 255 {
			n = 255
		}
		return uint8(n)
	}
	return color.RGBA{R: channel(ink.R), G: channel(ink.G), B: channel(ink.B), A: ink.A}
}

func (g *Game) drawModernHDFacilitySuccess(screen *ebiten.Image, job *nativeClassUIJob, index int) bool {
	if job == nil || index < 0 || index >= len(job.timeline) {
		return false
	}
	step := job.timeline[index]
	s := step.facilitySuccess
	if s == nil || s.presentation == nil {
		return false
	}
	d := s.presentation
	d.lastDraw = nil
	palette, ok := preparationCursorPalette(step.palette)
	if !ok || len(palette) != 256 || step.duration != s.duration || sha256.Sum256(step.frame) != s.frame || sha256.Sum256(step.shopEffectMask) != s.effects ||
		g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != d.owner || g.nativeClassUI == nil || g.font != d.font || g.localeID != d.locale ||
		g.helpVisible || g.transientUI || g.nativeFullDACBlack || g.nativeFullDACWhite || g.nativePlayerStatus != nil {
		return false
	}
	for i, c := range palette {
		if c != s.palette[i] {
			return false
		}
	}
	if d.owner == "shop" {
		if g.nativeShopUIJob != job || g.nativeShopVariant != d.variant || g.camp.Node().NativeHubVariant != d.variant ||
			(g.nativeShopMode != "success" && g.nativeShopMode != "sell_success") {
			return false
		}
	} else if d.owner != "church" || g.nativeClassUIJob != job || g.churchMode != "revive_confirm" {
		return false
	}
	uiHD, artHD := g.modernHDText != nil, g.modernStoryPortraits != nil
	if !uiHD && !artHD {
		return false
	}
	count, _, _, offset, _ := facilitySuccessShape(d.resource)
	pos := facilitySuccessPosition(s.stage, count)
	if pos < 0 || len(d.raw) != count+2 || len(d.hd) != count+2 {
		return false
	}
	var scene *modernFacilityScene
	var patch *modernHDOverlay
	if artHD {
		b := g.modernStoryPortraits.facilitySuccess[d.resource]
		scene = g.modernFacility(d.resource)
		if b == nil || b.scene != scene || scene == nil || b.sceneImage != scene.image || len(b.frames) != count || len(b.native) != count {
			return false
		}
		var originals []fdother.Frame
		if d.resource == 14 {
			originals = g.nativeClassUI.reviveFX
		} else {
			a, _, _, valid := g.nativeShopState()
			if !valid || a.ResourceID != d.resource {
				return false
			}
			originals = a.SuccessFrames
		}
		if len(originals) != count {
			return false
		}
		for i, f := range originals {
			sum := sha256.Sum256(f.Mask)
			if b.native[i] != d.native[i] || facilitySuccessDescription(i, f) != d.native[i] || hex.EncodeToString(sum[:]) != facilitySuccessMaskSHA[d.resource] {
				return false
			}
		}
		if s.stage >= 0 {
			if b.frames[s.stage] == nil || len(b.pictures) != count || len(b.digests) != count || b.pictures[s.stage] == nil || b.frames[s.stage].hdImage != b.pictures[s.stage] || sha256.Sum256(b.pictures[s.stage].Pix) != b.digests[s.stage] {
				return false
			}
			f := b.frames[s.stage]
			if f == nil || f.hdImage == nil || f.hdImage.Bounds() != image.Rect(0, 0, 480, 480) {
				return false
			}
			x, y := offset%320*2, offset/320*2
			patch = &modernHDOverlay{portrait: f, rect: image.Rect(x, y, x+160, y+160)}
		}
	}
	frame := d.raw[pos]
	runs := []modernHDTextRun(nil)
	if uiHD {
		frame = d.hd[pos]
		runs = append(runs, d.source.runs...)
	}
	if job.goldText != nil {
		if d.owner != "shop" || s.stage != -2 || len(job.goldText.windows) != len(job.timeline) {
			return false
		}
		// The immutable HD source contains the pre-roll text; remove that window.
		filtered := runs[:0]
		for _, run := range runs {
			if run.x >= 32 && run.x < 128 && run.y >= 190 && run.y < 216 {
				continue
			}
			filtered = append(filtered, run)
		}
		runs = filtered
		if uiHD {
			var rolling []modernHDTextRun
			frame, rolling, ok = job.goldText.compose(frame, index, g.font, step.palette)
			if !ok {
				return false
			}
			runs = append(runs, rolling...)
		} else {
			frame = step.frame
		}
	}
	if artHD {
		if !g.presentModernFacilityFrame(screen, frame, d.source.mask, step.palette, &modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 1920, 1200), success: patch, whiteDelta: s.delta}) {
			return false
		}
	} else {
		g.presentNativeClassFrameWithPalette(screen, frame, step.palette)
	}
	if uiHD {
		for i := range runs {
			runs[i].color = facilitySuccessWhite(runs[i].color, s.delta)
		}
		g.modernHDText.runs = append(g.modernHDText.runs, runs...)
		if d.chrome != nil {
			g.modernHDText.panels = append(g.modernHDText.panels, modernHDDialoguePanel{chrome: d.chrome, rect: image.Rect(5, 112, 315, 198), gold: true, whiteDelta: s.delta})
		}
	}
	d.lastDraw = &modernChurchConfirmationDraw{frame: index, choiceFrame: s.stage, phase: "success", interfaceHD: uiHD, artHD: artHD}
	return true
}

func drawModernHDFacilitySuccessPatch(target modernHDImageTarget, geometry ebiten.GeoM, overlay *modernHDOverlay, delta int) {
	f := overlay.portrait
	if f.hdGPU == nil {
		f.hdGPU = ebiten.NewImageFromImage(f.hdImage)
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/3, 1.0/3)
	op.GeoM.Translate(float64(overlay.rect.Min.X), float64(overlay.rect.Min.Y))
	op.GeoM.Concat(geometry)
	if delta > 0 {
		d := float64(delta) / 63
		op.ColorM.Translate(d, d, d, 0)
	}
	target.DrawImage(f.hdGPU, op)
}
