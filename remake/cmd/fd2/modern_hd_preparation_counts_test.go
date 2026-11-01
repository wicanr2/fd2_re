package main

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

func preparationCountRuns(g *Game, y int) string {
	if g.modernHDText == nil {
		return ""
	}
	result := ""
	for _, x := range []int{61, 67} {
		found := ""
		for _, run := range g.modernHDText.runs {
			if run.inkTop && run.x == float64(x*2) && run.y == float64(y*2) {
				found += run.text
			}
		}
		result += found
	}
	return result
}

func verifyPreparationCountsCPU(t *testing.T, g *Game, set *modernStoryPortraitSet) {
	t.Helper()
	saved := *g
	defer func() { *g = saved }()
	g.nativeClassUIJob = nil
	preparationHDTestPhase(g, "roster")
	preparationRosterThirty(g)
	g.modernStoryPortraits, g.modernThemeCache, g.interfaceMode = set, set, playerInterfaceHD
	ui := g.nativePreparationUI
	if !preparationCountSourceMatches(ui.roster.UpperLeft) {
		t.Fatal("pinned Frame137 source differs")
	}
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	var first *modernHDPreparationCounts
	for _, limit := range []int{0, 1, 9, 10, 15, 30} {
		for _, selected := range []int{0, limit} {
			g.prepLimit = limit
			for i, id := range g.prepIDs {
				g.partyDeploy[id] = i < selected
			}
			before, ok := g.composeNativePreparationStableFrame()
			if !ok {
				t.Fatal("native count source missing")
			}
			g.Draw(screen)
			h := g.prepareModernHDPreparationCounts(g.nativePreparationUI)
			if h == nil {
				t.Fatal("HD counts missing")
			}
			if first == nil {
				first = h
			} else if first != h {
				t.Fatal("changing digits rebuilt static captions")
			}
			if got := preparationCountRuns(g, 35); got != fmt.Sprintf("%02d", limit) {
				t.Fatal("quota digit producer differs", limit, got)
			}
			if got := preparationCountRuns(g, 73); got != fmt.Sprintf("%02d", limit-selected) {
				t.Fatal("remaining digit producer differs", limit, selected, got)
			}
			after, ok := g.composeNativePreparationStableFrame()
			if !ok || !bytes.Equal(before, after) || g.nativePreparationUI != ui || g.nativePreparationUI.roster != ui.roster || g.font.indexedHD != nil {
				t.Fatal("adapter changed native source/cache")
			}
		}
	}
	if &first.chrome[0] == &first.chrome[1] {
		t.Fatal("different panel sizes share GPU texture cache")
	}
	for _, fault := range []string{"pixels", "mask", "geometry", "palette", "font", "cursor", "confirm", "slots", "job", "original"} {
		state := *g
		copy := *ui
		roster := *ui.roster
		copy.roster = &roster
		g.nativePreparationUI = &copy
		switch fault {
		case "pixels":
			roster.UpperLeft.Indexed = append([]byte(nil), roster.UpperLeft.Indexed...)
			roster.UpperLeft.Indexed[0] ^= 255
		case "mask":
			roster.UpperLeft.Mask = append([]byte(nil), roster.UpperLeft.Mask...)
			roster.UpperLeft.Mask[0] ^= 255
		case "geometry":
			roster.UpperLeft.Width = 85
		case "palette":
			c := *g.nativeClassUI
			c.palette = nil
			g.nativeClassUI = &c
		case "font":
			g.font = nil
		case "cursor":
			g.prepSel = 30
		case "confirm":
			g.prepConfirm = true
		case "slots":
			g.prepRecordSlots = true
		case "job":
			g.nativeClassUIJob = &nativeClassUIJob{}
		case "original":
			g.modernHDText = nil
		}
		if g.prepareModernHDPreparationCounts(g.nativePreparationUI) != nil {
			t.Fatal("invalid count source/owner admitted", fault)
		}
		*g = state
	}
	g.prepLimit = 30
	g.modernStoryPortraits = nil
	g.Draw(screen)
	if g.prepareModernHDPreparationCounts(ui) == nil {
		t.Fatal("original art lost HD captions")
	}
}

func TestModernHDPreparationCountsRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PREPARATION_COUNTS_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	runModernHDPreparationRosterAndCountsMatrix(t, true, true)
}

func appendPreparationCountCases(p *facilityHDProbe, set *modernStoryPortraitSet) {
	g := p.g
	camp, town, source := g.camp, g.prepTownSource, append([]byte(nil), g.prepPromptSource...)
	for _, limit := range []int{0, 1, 9, 10, 15, 30} {
		for _, selected := range []int{0, limit} {
			for style := 0; style < 4; style++ {
				limit, selected, style := limit, selected, style
				if style == 3 && selected == 0 {
					p.captureCases[len(p.cases)] = true
				}
				p.cases = append(p.cases, func() {
					g.camp, g.prepTownSource, g.prepPromptSource = camp, town, source
					g.nativeClassUIJob = nil
					preparationHDTestPhase(g, "roster")
					g.prepLimit, g.prepSel = limit, 29
					for i, id := range g.prepIDs {
						g.partyDeploy[id] = i < selected
					}
					g.modernStoryPortraits = nil
					if style >= 2 {
						g.modernStoryPortraits = set
					}
					g.interfaceMode = playerInterfaceOriginal
					if style%2 == 1 {
						g.interfaceMode = playerInterfaceHD
					}
				})
			}
		}
	}
}

func verifyPreparationCountsGPU(p *facilityHDProbe, pixels []byte) error {
	g := p.g
	if g.interfaceMode == playerInterfaceOriginal {
		return nil
	}
	h := g.prepareModernHDPreparationCounts(g.nativePreparationUI)
	if h == nil {
		return fmt.Errorf("case%d counts missing", p.index)
	}
	colors := nativeDialogueChromeSource(g.nativePreparationUI.dialogue, g.nativeClassUI.palette)
	edges, labels := 0, 0
	for _, r := range []image.Rectangle{image.Rect(5, 7, 91, 93), image.Rect(5, 94, 315, 193)} {
		for y := 0; y < r.Dy()*6; y++ {
			for x := 0; x < r.Dx()*6; x++ {
				if c, ok := itemPanelChromePixel(colors, x, y, r.Dx()*6, r.Dy()*6); ok {
					at := (((r.Min.Y*6+y)+72)*1920 + r.Min.X*6 + x) * 4
					if !bytes.Equal(pixels[at:at+4], c[:]) {
						return fmt.Errorf("case%d count/frame edge %v %d,%d", p.index, r, x, y)
					}
					edges++
				}
			}
		}
	}
	// Static label raster uses the already verified panel-ink renderer; compare
	// its nontransparent source to final pixels, including original text erasure.
	for y := 0; y < h.labels.picture.Bounds().Dy(); y++ {
		for x := 0; x < h.labels.picture.Bounds().Dx(); x++ {
			c := h.labels.picture.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			at := ((7*6+y+72)*1920 + 5*6 + x) * 4
			if !bytes.Equal(pixels[at:at+4], []byte{c.R, c.G, c.B, c.A}) {
				return fmt.Errorf("case%d count caption source %d,%d", p.index, x, y)
			}
			labels++
		}
	}
	selected := 0
	for _, id := range g.prepIDs {
		if g.partyDeploy[id] {
			selected++
		}
	}
	if preparationCountRuns(g, 35) != fmt.Sprintf("%02d", g.prepLimit) || preparationCountRuns(g, 73) != fmt.Sprintf("%02d", g.prepLimit-selected) {
		return fmt.Errorf("case%d native count text differs", p.index)
	}
	for _, run := range g.modernHDText.runs {
		if !run.inkTop || run.y != 70 && run.y != 146 {
			continue
		}
		face, baseline := modernHDTextFace(g.font, run, g.fontScale)
		ink := text.BoundString(face, run.text).Add(image.Pt(int(math.Round(run.x*3)), int(baseline)))
		clip := image.Rectangle{Min: run.clip.Min.Mul(3), Max: run.clip.Max.Mul(3)}
		if !ink.In(clip) {
			return fmt.Errorf("case%d count glyph clipped: %s %v outside %v", p.index, run.text, ink, clip)
		}
	}
	if labels == 0 || edges == 0 {
		return fmt.Errorf("missing count geometry")
	}
	last := p.checks[len(p.checks)-1]
	last["count_edge_pixels"], last["count_label_pixels"] = edges, labels
	return nil
}
