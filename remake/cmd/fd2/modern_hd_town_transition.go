package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// Geometry uses the same fixed source center and spacing as native sampling.
// Truecolor subtraction is an art approximation, not indexed DAC parity.
type modernTownDepartureTransform struct {
	geometrySource campaign.NativeTownDepartureGeometry
	delta          int
}

func (d *modernTownDepartureTransform) geometry() ebiten.GeoM {
	var result ebiten.GeoM
	s := float64(d.geometrySource.Spacing)
	result.Scale(128/s, 128/s)
	result.Translate(2*(160-float64(d.geometrySource.Center.X)/s),
		2*(100-float64(d.geometrySource.Center.Y)/s))
	return result
}

func (d *modernTownDepartureTransform) darken(op *ebiten.DrawImageOptions) {
	amount := -float64(d.delta) / 63
	op.ColorM.Translate(amount, amount, amount, 0)
}

func (d *modernTownDepartureTransform) text(run modernHDTextRun) modernHDTextRun {
	m := d.geometry()
	x0, y0 := m.Apply(float64(run.clip.Min.X), float64(run.clip.Min.Y))
	x1, y1 := m.Apply(float64(run.clip.Max.X), float64(run.clip.Max.Y))
	run.clip = image.Rect(int(math.Floor(x0)), int(math.Floor(y0)), int(math.Ceil(x1)), int(math.Ceil(y1))).Intersect(image.Rect(0, 0, 640, 400))
	run.x, run.y = m.Apply(run.x, run.y)
	run.pixels *= 128 / float64(d.geometrySource.Spacing)
	for _, c := range []*uint8{&run.color.R, &run.color.G, &run.color.B} {
		*c = uint8(math.Max(0, math.Round(float64(*c)-255*float64(d.delta)/63)))
	}
	return run
}

func modernTownTextRuns(runs []modernHDTextRun) []modernHDTextRun {
	var result []modernHDTextRun
	for _, run := range runs {
		run.x += 8
		run.y += 8
		run.inkTop = true
		run.clip = run.clip.Add(image.Pt(8, 8)).Intersect(image.Rect(8, 8, 632, 392))
		if !run.clip.Empty() {
			result = append(result, run)
		}
	}
	return result
}

func (g *Game) drawModernHDTownDeparture(screen *ebiten.Image, job *nativeTownDepartureJob) bool {
	if job == nil || job.source == nil || job.step < 0 || job.step >= 10 || g.nativeClassUI == nil {
		return false
	}
	d := job.source
	scene := g.modernFacility([3]int{11, 61, 62}[d.variant])
	if scene == nil && g.modernHDText == nil {
		return false
	}
	geometry, err := campaign.NativeTownDepartureGeometryFor(d.variant, 2, job.step+1)
	if err != nil {
		return false
	}
	transform := &modernTownDepartureTransform{geometrySource: geometry, delta: 4 * (job.step + 1)}
	capture := &modernHDIndexedText{palette: g.nativeClassUI.palette}
	font := g.modernHDFacilityFont(d.font, capture)
	var overlay *modernFacilitySceneOverlay
	var mask []byte
	if scene != nil {
		cursor, err := g.modernTownCursorFor(d.variant, d.selection, d.pulse, d.group)
		if err != nil {
			return false
		}
		var layers [2][]byte
		for i, fill := range []byte{0, 255} {
			source, ok := d.compose(&fill, false, font)
			if !ok {
				return false
			}
			layers[i], err = campaign.ComposeNativeTownDepartureFrame(source, d.variant, 2, job.step+1)
			if err != nil {
				return false
			}
		}
		mask = make([]byte, 64000)
		for i := range mask {
			if layers[0][i] == layers[1][i] {
				mask[i] = 255
			}
		}
		overlay = &modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 312*6, 192*6),
			dest: image.Pt(8, 8), cursor: cursor, departure: transform}
	}
	capture.runs = nil
	source, ok := d.compose(nil, scene == nil, font)
	if !ok {
		return false
	}
	frame, err := campaign.ComposeNativeTownDepartureFrame(source, d.variant, 2, job.step+1)
	if err != nil {
		return false
	}
	if overlay != nil {
		if !g.presentModernFacilityFrame(screen, frame, mask, job.frames[job.step].palette, overlay) {
			return false
		}
	} else {
		g.presentNativeClassFrameWithPalette(screen, frame, job.frames[job.step].palette)
	}
	if g.modernHDText != nil {
		for _, run := range modernTownTextRuns(capture.runs) {
			run = transform.text(run)
			if !run.clip.Empty() {
				g.modernHDText.runs = append(g.modernHDText.runs, run)
			}
		}
	}
	return true
}
