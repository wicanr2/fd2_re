package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestModernHDGoldStoryInterfaceRejectsBeforePublish(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g, _ := expiryHDTestGame(t, nil, "en")
	g.interfaceMode = playerInterfaceOriginal
	g.modernThemeCache = &modernStoryPortraitSet{hasHD: true}
	g.dialog = []battle.DialogLine{{NativeDialogue: &battle.NativeDialogueLayout{Control: "FFEC"}}}
	bad := *g.nativeClassUI
	bad.dialogue = append([]fdother.RawCell(nil), bad.dialogue...)
	bad.dialogue[13].Pixels = append([]byte(nil), bad.dialogue[13].Pixels...)
	bad.dialogue[13].Pixels[0] ^= 1
	g.nativeClassUI = &bad
	before, _ := json.Marshal(g.st)
	rng := g.nativeRNGState
	if err := g.setInterfaceMode(playerInterfaceHD); err == nil || !strings.Contains(err.Error(), "background is not uniform") {
		t.Fatal("story source did not reject F11", err)
	}
	after, _ := json.Marshal(g.st)
	if g.interfaceMode != playerInterfaceOriginal || g.modernStoryPortraits != nil || g.nativeDialogueChrome != nil || !bytes.Equal(before, after) || rng != g.nativeRNGState {
		t.Fatal("failed F11 published state")
	}
}

func TestModernHDGoldDialogueMaterial(t *testing.T) {
	a, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	c, err := freezeModernHDDialogueChrome(a.dialogue, a.palette)
	if err != nil {
		t.Fatal(err)
	}
	ref := independentGoldReference()
	patch := c.goldPatchFor(1860, 516)
	for y := 0; y < 516; y++ {
		for x := 0; x < 1860; x++ {
			if patch.RGBAAt(x, y) != ref.At(x, y) { // Decode returns *image.NRGBA; compare channels below.
				r, g, b, a := ref.At(x, y).RGBA()
				got := patch.RGBAAt(x, y)
				if got.R != byte(r>>8) || got.G != byte(g>>8) || got.B != byte(b>>8) || got.A != byte(a>>8) {
					t.Fatalf("SVG reference differs at %d,%d", x, y)
				}
			}
		}
	}
	for _, control := range []string{"FFEC", "FFEE", "FFED", "FFEF"} {
		layout := &campaign.NativeDialogueLayout{SourceDAT: "FDTXT_001", Control: control, Pages: [][]string{{"Gold"}}}
		opening, e := campaign.ComposeNativeStoryDialogueOpeningFrames(make([]byte, 64000), a.dialogue, layout)
		if e != nil {
			t.Fatal(e)
		}
		for index, pixels := range opening {
			r, e := campaign.NativeStoryDialogueFrameBounds(a.dialogue, control, campaign.NativeDialogueOpening, index)
			if e != nil {
				t.Fatal(e)
			}
			// Native composition on two different backgrounds independently
			// identifies all writes; the proposed rectangle must match them.
			other, e := campaign.ComposeNativeStoryDialogueOpeningFrames(bytes.Repeat([]byte{255}, 64000), a.dialogue, layout)
			if e != nil {
				t.Fatal(e)
			}
			for at, v := range pixels {
				if (v == other[index][at]) != image.Pt(at%320, at/320).In(r) {
					t.Fatal("native frame bounds differ", control, index, at)
				}
			}
		}
		for _, index := range []int{4, 5, 20} {
			r, e := campaign.NativeStoryDialogueFrameBounds(a.dialogue, control, campaign.NativeDialogueClosing, index)
			if e != nil || !r.Empty() {
				t.Fatal("cursor tail retained gold", r, e)
			}
		}
	}
}

type goldStoryCase struct {
	control, phase string
	index          int
	art, ui        bool
	legacy         bool
}
type goldStoryProbe struct {
	t                *testing.T
	g                *Game
	set              *modernStoryPortraitSet
	cases            []goldStoryCase
	index            int
	final, base      *ebiten.Image
	pixels, baseline []byte
	rows             []map[string]any
	out              string
	err              error
}

func (p *goldStoryProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *goldStoryProbe) Update() error {
	if p.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *goldStoryProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == len(p.cases) {
		return
	}
	c, g := p.cases[p.index], p.g
	p.t.Logf("gold case %d/%d: %+v", p.index+1, len(p.cases), c)
	upper := c.control == "FFED" || c.control == "FFEF"
	art := p.set
	if !c.art {
		art = nil
	}
	if g.modernStoryPortraits != art {
		if err := g.applyTheme(art); err != nil {
			p.err = err
			return
		}
	}
	newUtterance := p.index == 0 || p.cases[p.index-1].control != c.control || p.cases[p.index-1].legacy != c.legacy
	if newUtterance {
		g.dialog = []battle.DialogLine{{Speaker: 0, Text: "Gold dialogue", Upper: &upper, NativeDialogue: &battle.NativeDialogueLayout{SourceDAT: "FDTXT_001", Control: c.control, Pages: [][]string{{"Gold dialogue"}}, HasMotionTargetY: true}}}
		if c.legacy {
			g.dialog[0].NativeDialogue = nil
		}
	}
	mode := playerInterfaceOriginal
	if c.ui {
		mode = playerInterfaceHD
	}
	if g.interfaceMode != mode {
		if err := g.setInterfaceMode(mode); err != nil {
			p.err = err
			return
		}
	}
	if newUtterance {
		if err := g.prepareNativeDialogueFrames(); err != nil {
			p.err = err
			return
		}
	}
	g.nativeDialogueClosingLive = false
	g.dlgShown, g.dlgUpper, g.dlgPhase, g.dlgT = 0, &upper, 0, 0
	g.dlgPage, g.dlgScrollT = 0, 0
	if !c.legacy {
		g.nativeDialogueProgress = len(g.nativeDialogueProgressive[0]) - 1
	}
	switch c.phase {
	case "opening":
		g.dlgPhase, g.dlgT = 2, c.index+1
	case "closing":
		g.nativeDialogueClosingLive, g.nativeDialogueClosingT = true, c.index
	case "typing":
		g.nativeDialogueProgress = 2
	case "scroll":
		g.dlgScrollT, g.dlgScrollFrom = 3, 0
	}
	before, _ := json.Marshal(g.st)
	rng := g.nativeRNGState
	clock := [5]int{g.dlgPhase, g.dlgT, g.nativeDialogueProgress, g.nativeDialogueClosingT, g.dlgScrollT}
	g.Draw(screen)
	if g.loadErr != "" {
		p.err = fmt.Errorf("%+v: %s", c, g.loadErr)
		return
	}
	after, _ := json.Marshal(g.st)
	if !bytes.Equal(before, after) || rng != g.nativeRNGState || clock != [5]int{g.dlgPhase, g.dlgT, g.nativeDialogueProgress, g.nativeDialogueClosingT, g.dlgScrollT} {
		p.err = fmt.Errorf("Draw changed gameplay or dialogue clock")
		return
	}
	frame := g.modernHDText
	var panels []modernHDDialoguePanel
	if frame != nil {
		panels = frame.panels
	}
	var geom ebiten.GeoM
	geom.Scale(3, 3)
	pres := &modernHDPresentation{Game: g}
	p.final.Clear()
	pres.DrawFinalScreen(p.final, screen, geom)
	p.final.ReadPixels(p.pixels)
	if frame != nil {
		frame.panels = nil
	}
	// Compare on the same GPU destination. Separate targets can round the
	// legacy portrait's fractional 2.1x transform differently at edge pixels.
	p.final.Clear()
	pres.DrawFinalScreen(p.final, screen, geom)
	p.final.ReadPixels(p.baseline)
	if frame != nil {
		frame.panels = panels
	}
	var rect image.Rectangle
	if len(panels) > 0 {
		rect = panels[0].physicalRect
		if rect.Empty() {
			rect = image.Rectangle{Min: panels[0].rect.Min.Mul(6), Max: panels[0].rect.Max.Mul(6)}
		}
	}
	wantPanel := c.ui && !(c.phase == "closing" && c.index == 4)
	if (len(panels) == 1) != wantPanel {
		p.err = fmt.Errorf("wrong panel owner %+v: %d", c, len(panels))
		return
	}
	ref := independentGoldReference()
	mapAxis := func(at, size, refSize int) int {
		if at < 18 {
			return at
		}
		if at >= size-18 {
			return refSize - size + at
		}
		return 30
	}
	changed, checked := 0, 0
	for at := 0; at < len(p.pixels); at += 4 {
		x, y := (at/4)%1920, (at/4)/1920
		local := image.Pt(x, y).Sub(rect.Min)
		band := image.Pt(x, y).In(rect) && (local.X < 18 || local.Y < 18 || local.X >= rect.Dx()-18 || local.Y >= rect.Dy()-18)
		if !band {
			if !bytes.Equal(p.pixels[at:at+4], p.baseline[at:at+4]) {
				p.err = fmt.Errorf("gold changed interior/outside %d,%d", x, y)
				return
			}
			continue
		}
		r, gg, b, a := ref.At(mapAxis(local.X, rect.Dx(), 1860), mapAxis(local.Y, rect.Dy(), 516)).RGBA()
		for ch, want := range []byte{byte(r >> 8), byte(gg >> 8), byte(b >> 8), byte(a >> 8)} {
			d := int(p.pixels[at+ch]) - int(want)
			if d < -1 || d > 1 {
				p.err = fmt.Errorf("gold SVG pixel differs %+v at %d,%d channel%d got%d want%d", c, x, y, ch, p.pixels[at+ch], want)
				return
			}
		}
		checked++
		if !bytes.Equal(p.pixels[at:at+4], p.baseline[at:at+4]) {
			changed++
		}
	}
	row := map[string]any{"control": c.control, "phase": c.phase, "index": c.index, "art": c.art, "ui": c.ui, "legacy": c.legacy, "gold_pixels": checked, "changed_pixels": changed, "gameplay_and_clocks_unchanged": true, "final_sha256": fmt.Sprintf("%x", sha256.Sum256(p.pixels))}
	if c.phase == "steady" && c.ui || c.control == "FFEC" && c.phase == "opening" && c.index == 0 {
		name := fmt.Sprintf("story-%s-%s-%d-art%t-ui%t-legacy%t.png", c.control, c.phase, c.index, c.art, c.ui, c.legacy)
		f, e := os.Create(filepath.Join(p.out, name))
		if e != nil {
			p.err = e
			return
		}
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, p.pixels)
		e = png.Encode(f, pic)
		closeErr := f.Close()
		if e != nil {
			p.err = e
			return
		}
		if closeErr != nil {
			p.err = closeErr
			return
		}
		row["capture"] = name
	}
	p.rows = append(p.rows, row)
	p.index++
}

func TestModernHDGoldStoryGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_GOLD_STORY_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	out := os.Getenv("FD2_TEST_GOLD_OUT")
	if out == "" {
		t.Fatal("output required")
	}
	if e := os.Mkdir(out, 0755); e != nil {
		t.Fatal(e)
	}
	set, e := loadModernStoryPortraitSet(assetPath("assets/themes/modern/catalog.json"), os.Getenv("FD2_MODERN_THEME_PACK"))
	if e != nil {
		t.Fatal(e)
	}
	g, _ := expiryHDTestGame(t, set, "en")
	g.modernThemeCache = set
	p := &goldStoryProbe{t: t, g: g, set: set, out: out, final: ebiten.NewImage(1920, 1200), base: ebiten.NewImage(1920, 1200), pixels: make([]byte, 1920*1200*4), baseline: make([]byte, 1920*1200*4)}
	for _, control := range []string{"FFEC", "FFEE", "FFED", "FFEF"} {
		for _, phase := range []string{"opening", "closing"} {
			for index := 0; index < 5; index++ {
				p.cases = append(p.cases, goldStoryCase{control, phase, index, true, true, false})
			}
		}
		for _, phase := range []string{"typing", "scroll"} {
			p.cases = append(p.cases, goldStoryCase{control, phase, 0, true, true, false})
		}
		for _, art := range []bool{false, true} {
			for _, ui := range []bool{false, true} {
				p.cases = append(p.cases, goldStoryCase{control, "steady", 0, art, ui, false})
			}
		}
	}
	for _, control := range []string{"FFEC", "FFED"} {
		for _, art := range []bool{false, true} {
			for _, ui := range []bool{false, true} {
				p.cases = append(p.cases, goldStoryCase{control, "steady", 0, art, ui, true})
			}
		}
		p.cases = append(p.cases, goldStoryCase{control, "opening", 0, true, true, true})
	}
	ebiten.SetVsyncEnabled(false)
	if e := ebiten.RunGame(p); e != nil {
		t.Fatal(e)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	data, _ := json.MarshalIndent(map[string]any{"status": "PASS-GOLD-STORY-GAME", "scope": "controlled native/legacy story, four controls and four art/interface combinations; not normal player evidence", "normal_player_verified": false, "rows": p.rows}, "", "  ")
	if e := os.WriteFile(filepath.Join(out, "receipt.json"), append(data, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
}
