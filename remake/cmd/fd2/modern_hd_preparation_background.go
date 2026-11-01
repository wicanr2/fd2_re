package main

import (
	"bytes"
	"image"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

func (d *nativeTownFrameSource) matches(source []byte) bool {
	return d != nil && d.variant >= 0 && d.variant < 3 && len(d.indexed) == 64000 &&
		len(source) == 64000 && bytes.Equal(d.indexed, source)
}

// Only the existing stable compositor sees the temporary source. It reads no
// clock or input and writes a new frame; the caller is restored before return.
func (g *Game) composeNativePreparationOn(source []byte) ([]byte, bool) {
	return g.composeNativePreparationOnLayers(source, true)
}

func (g *Game) composeNativePreparationOnLayers(source []byte, drawUnits bool) ([]byte, bool) {
	return g.composeNativePreparationOnParts(source, drawUnits, true)
}

func (g *Game) composeNativePreparationOnParts(source []byte, drawUnits, drawCursor bool) ([]byte, bool) {
	if len(source) != 64000 {
		return nil, false
	}
	original := g.prepPromptSource
	g.prepPromptSource = source
	defer func() { g.prepPromptSource = original }()
	return g.composeNativePreparationStableFrameParts(drawUnits, drawCursor)
}

// Clip frozen town text by actual writes made by the new preparation panel.
// Horizontal spans coalesce vertically, keeping partial opening/closing ink.
func modernTownTextOutsideWrites(runs []modernHDTextRun, mask []byte) []modernHDTextRun {
	if len(mask) != 64000 {
		return nil
	}
	var result []modernHDTextRun
	for _, run := range runs {
		run = run.freezeLayout()
		start := len(result)
		clip := run.clip.Intersect(image.Rect(0, 0, 640, 400))
		spans := make(map[[2]int]image.Rectangle)
		flush := func(rect image.Rectangle) {
			copy := run
			copy.clip = rect.Intersect(clip)
			if !copy.clip.Empty() {
				result = append(result, copy)
			}
		}
		for y := clip.Min.Y / 2; y < (clip.Max.Y+1)/2; y++ {
			next := make(map[[2]int]image.Rectangle)
			for x := clip.Min.X / 2; x < (clip.Max.X+1)/2; {
				if mask[y*320+x] != 0 {
					x++
					continue
				}
				start := x
				for x < (clip.Max.X+1)/2 && mask[y*320+x] == 0 {
					x++
				}
				key := [2]int{start, x}
				rect := image.Rect(start*2, y*2, x*2, (y+1)*2)
				if prior, ok := spans[key]; ok {
					rect.Min.Y = prior.Min.Y
					delete(spans, key)
				}
				next[key] = rect
			}
			for _, rect := range spans {
				flush(rect)
			}
			spans = next
		}
		for _, rect := range spans {
			flush(rect)
		}
		parts := result[start:]
		sort.Slice(parts, func(i, j int) bool {
			a, b := parts[i].clip, parts[j].clip
			if a.Min.Y != b.Min.Y {
				return a.Min.Y < b.Min.Y
			}
			if a.Min.X != b.Min.X {
				return a.Min.X < b.Min.X
			}
			if a.Max.Y != b.Max.Y {
				return a.Max.Y < b.Max.Y
			}
			return a.Max.X < b.Max.X
		})
	}
	return result
}

// Each pass replays the same native producer and font policy. Its mask passes
// never publish text or advance the preparation job.
func (g *Game) drawModernHDPreparationBackground(screen *ebiten.Image, town *nativeTownFrameSource,
	source []byte, compose func([]byte) ([]byte, bool), capture *modernHDIndexedText) bool {
	if !town.matches(source) || g.nativeClassUI == nil {
		return false
	}
	scene := g.modernFacility([3]int{11, 61, 62}[town.variant])
	if scene == nil && g.modernHDText == nil {
		return false
	}
	townText := &modernHDIndexedText{palette: g.nativeClassUI.palette}
	font := g.modernHDFacilityFont(town.font, townText)
	clearRuns := func() {
		townText.runs = nil
		if capture != nil {
			capture.runs = nil
		}
	}
	published := false
	defer func() {
		if !published {
			clearRuns()
		}
	}()
	var writes []byte
	if g.modernHDText != nil {
		a, okA := compose(make([]byte, 64000))
		b, okB := compose(bytes.Repeat([]byte{255}, 64000))
		if !okA || !okB || len(a) != 64000 || len(b) != 64000 {
			return false
		}
		writes = make([]byte, 64000)
		for i := range writes {
			if a[i] == b[i] {
				writes[i] = 255
			}
		}
	}
	var overlay *modernFacilitySceneOverlay
	var mask []byte
	if scene != nil {
		cursor, err := g.modernTownCursorFor(town.variant, town.selection, town.pulse, town.group)
		if err != nil {
			return false
		}
		var layers [2][]byte
		for i, fill := range []byte{0, 255} {
			background, ok := town.compose(&fill, false, font)
			if !ok {
				return false
			}
			layers[i], ok = compose(background)
			if !ok || len(layers[i]) != 64000 {
				return false
			}
		}
		mask = make([]byte, 64000)
		for i := range mask {
			if layers[0][i] == layers[1][i] {
				mask[i] = 255
			}
		}
		overlay = &modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 312*6, 192*6), dest: image.Pt(8, 8), cursor: cursor}
	}
	clearRuns()
	background, ok := town.compose(nil, scene == nil, font)
	if !ok {
		return false
	}
	frame, ok := compose(background)
	if !ok {
		return false
	}
	if overlay != nil {
		if !g.presentModernFacilityFrame(screen, frame, mask, g.nativeClassUI.palette, overlay) {
			return false
		}
	} else {
		g.presentNativeClassFrame(screen, frame)
	}
	if g.modernHDText != nil {
		g.modernHDText.runs = append(g.modernHDText.runs, modernTownTextOutsideWrites(modernTownTextRuns(townText.runs), writes)...)
		if capture != nil {
			g.modernHDText.runs = append(g.modernHDText.runs, capture.runs...)
		}
	}
	published = true
	return true
}
