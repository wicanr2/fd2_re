package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type titleInterfaceGPUCase struct {
	phase                     string
	selection, flash, dataset int
	art, ui                   bool
	locale                    string
	scale                     float64
}
type titleInterfaceGPUProbe struct {
	t             *testing.T
	g             *Game
	set           *modernStoryPortraitSet
	cases         []titleInterfaceGPUCase
	index         int
	err           error
	out           string
	final         *ebiten.Image
	pixels        []byte
	rows          []map[string]any
	slotSource    []byte
	activeDataset int
}

func (p *titleInterfaceGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *titleInterfaceGPUProbe) Update() error {
	if p.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func titleInterfaceGPUState(g *Game) string {
	seals := []any{}
	for i := 0; i < 3; i++ {
		for s := 0; s < 2; s++ {
			seals = append(seals, titleMenuHDSeal(g.titleAssets.hdItems[i][s]))
		}
	}
	seals = append(seals, titleLoadSlotsHDSeal(g.nativeLoadSlotsUI, g.nativeLoadSlotsUI.hd))
	raw, _ := json.Marshal([]any{g.titlePhase, g.titleSel, g.titleFlash, g.titleSlotSel, g.gold, g.nativeRNGState, unitPresentGPUState(g), seals})
	return string(raw)
}
func (p *titleInterfaceGPUProbe) installSlots(dataset int) error {
	if dataset == p.activeDataset {
		return nil
	}
	for slot := 0; slot < 4; slot++ {
		path := saveSlotPath(slot)
		if dataset == 1 && (slot == 1 || slot == 2) {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, p.slotSource, 0644); err != nil {
				return err
			}
		} else if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	p.activeDataset = dataset
	return nil
}
func (p *titleInterfaceGPUProbe) expected(c titleInterfaceGPUCase, frame *modernHDTextFrame) (*image.RGBA, error) {
	g := p.g
	raw := make([]byte, 64000)
	if c.phase == "menu" {
		hdBackdrop := c.art && p.set.openingHD != nil
		var backdrop image.Image
		if hdBackdrop {
			bank := p.set.openingHD.banks[modernOpeningKey{"FDOTHER.DAT", 7}]
			if bank == nil {
				return nil, fmt.Errorf("missing admitted title bank")
			}
			var err error
			backdrop, err = png.Decode(bytes.NewReader(bank.encoded[0]))
			if err != nil {
				return nil, err
			}
		}
		_, palette, err := fdother.LoadSeparatedFDOTHERPalette(separatedAssetPath("palette"), 8)
		if err != nil {
			return nil, err
		}
		base, err := fdother.LoadSeparatedNestedSingleFrame(separatedAssetPath("surfaces"), 7, 0)
		if err != nil {
			return nil, err
		}
		if err := base.Blit(raw, 320, -1); err != nil {
			return nil, err
		}
		for i := 0; i < 3; i++ {
			state := 0
			if i == c.selection && (c.flash == 0 || (c.flash/3)%2 == 0) {
				state = 1
			}
			source, err := fdother.LoadSeparatedNestedSingleFrame(separatedAssetPath("surfaces"), 7, 1+i*2+state)
			if err != nil {
				return nil, err
			}
			if c.ui {
				source, _, _ = titlePrototypeItem(p.t, source, palette, g.font, i)
			}
			source.X, source.Y = (320-source.Width)/2, titleMenuY(i)
			if err := source.Blit(raw, 320, -1); err != nil {
				return nil, err
			}
		}
		native := titlePrototypeRaster(raw, palette)
		if hdBackdrop {
			// Rebuild the title and each original menu-cell rectangle independently.
			// Native pixels outside those rectangles must not cover the HD title.
			draw.Draw(native, native.Bounds(), backdrop, image.Point{}, draw.Src)
			for i := 0; i < 3; i++ {
				state := 0
				if i == c.selection && (c.flash == 0 || (c.flash/3)%2 == 0) {
					state = 1
				}
				source, err := fdother.LoadSeparatedNestedSingleFrame(separatedAssetPath("surfaces"), 7, 1+i*2+state)
				if err != nil {
					return nil, err
				}
				r := image.Rect((320-source.Width)/2*6, titleMenuY(i)*6, (320+source.Width)/2*6, (titleMenuY(i)+source.Height)*6)
				original := titlePrototypeRaster(raw, palette)
				draw.Draw(native, r, original, r.Min, draw.Src)
			}
		}
		return titleInterfaceGPUCPU(g, native.Pix, frame), nil
	}
	a := g.nativeLoadSlotsUI
	slots, ok := nativeLoadSlotMetadata()
	if !ok {
		return nil, fmt.Errorf("fixture metadata unavailable")
	}
	font := a.font
	if c.ui {
		capture := &modernHDIndexedText{palette: a.palette}
		font = g.modernHDFacilityFont(font, capture)
	}
	native, err := campaign.ComposeNativeLoadSlotsFrame(raw, a.dialogueBox, a.strings, font, slots, c.selection)
	if err != nil {
		return nil, err
	}
	raster := titlePrototypeRaster(native, a.palette)
	return titleInterfaceGPUCPU(g, raster.Pix, frame), nil
}

// Keep the existing layout planner, then rasterize the final fonts and clips
// independently on CPU. The shared old probe assumes fontScale=1.
func titleInterfaceGPUCPU(g *Game, base []byte, frame *modernHDTextFrame) *image.RGBA {
	if frame == nil {
		return systemGPUCPU(g, base, nil, nil)
	}
	panels := *frame
	panels.runs = nil
	result := systemGPUCPU(g, base, &panels, nil)
	glyphs := titleInterfaceGPUText(g, frame)
	draw.Draw(result, result.Bounds(), glyphs, image.Point{}, draw.Over)
	return result
}

func titleInterfaceGPUText(g *Game, frame *modernHDTextFrame) *image.RGBA {
	glyphs := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	for _, run := range frame.runs {
		f := run.font
		if regional := f.hdFonts[g.localeID]; regional != nil {
			f = regional
		}
		face, baseline := modernHDTextFace(f, run, g.fontScale)
		clip := image.Rectangle{Min: run.clip.Min.Mul(3), Max: run.clip.Max.Mul(3)}.Intersect(glyphs.Bounds())
		if face == nil || clip.Empty() {
			continue
		}
		drawer := font.Drawer{Dst: glyphs.SubImage(clip).(*image.RGBA), Src: image.NewUniform(run.color), Face: face, Dot: fixed.P(int(math.Round(run.x*3)), int(math.Round(baseline)))}
		drawer.DrawString(run.text)
	}
	return glyphs
}

func (p *titleInterfaceGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == len(p.cases) {
		return
	}
	c := p.cases[p.index]
	g := p.g
	if err := p.installSlots(c.dataset); err != nil {
		p.err = err
		return
	}
	g.titlePhase, g.titleSel, g.titleFlash, g.titleSlotSel = c.phase, c.selection, c.flash, c.selection
	g.localeID, g.fontScale = c.locale, c.scale
	var art *modernStoryPortraitSet
	if c.art {
		art = p.set
	}
	if g.modernStoryPortraits != art {
		if err := g.applyTheme(art); err != nil {
			p.err = err
			return
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
	slotsBefore := [][]byte{}
	for slot := 0; slot < 4; slot++ {
		raw, err := os.ReadFile(saveSlotPath(slot))
		if err != nil && !os.IsNotExist(err) {
			p.err = err
			return
		}
		slotsBefore = append(slotsBefore, raw)
	}
	before := titleInterfaceGPUState(g)
	g.Draw(screen)
	if g.loadErr != "" {
		p.err = fmt.Errorf("case%d: %s", p.index, g.loadErr)
		return
	}
	frame := g.modernHDText
	if (frame != nil) != c.ui {
		p.err = fmt.Errorf("interface followed art control")
		return
	}
	if frame != nil {
		if c.phase == "menu" {
			if len(frame.runs) != 6 || len(frame.panels) != 0 {
				p.err = fmt.Errorf("menu text count/panel differs")
				return
			}
		} else if len(frame.panels) != 1 || frame.panels[0].gold || frame.panels[0].rect != image.Rect(5, 112, 315, 198) {
			p.err = fmt.Errorf("slot border missing/wrong")
			return
		}
	}
	want, err := p.expected(c, frame)
	if err != nil {
		p.err = err
		return
	}
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.final.Clear()
	presentation := &modernHDPresentation{Game: g}
	presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(p.pixels)
	// Verify each stage independently. CPU font blending truncates differently
	// from GL; do not accumulate that difference through another opaque blend.
	var gpuGlyphs []byte
	if frame != nil {
		gpuGlyphs = make([]byte, len(p.pixels))
		presentation.textSurface.ReadPixels(gpuGlyphs)
		cpuGlyphs := titleInterfaceGPUText(g, frame)
		for i, v := range gpuGlyphs {
			delta := int(v) - int(cpuGlyphs.Pix[i])
			if delta < -2 || delta > 2 {
				p.dumpFailure(c, frame, screen, presentation, want, geometry)
				p.err = fmt.Errorf("case%d glyph layer differs at byte%d: %d/%d", p.index, i, v, cpuGlyphs.Pix[i])
				return
			}
		}
		empty := *frame
		empty.runs = nil
		g.modernHDText = &empty
		p.final.Clear()
		presentation.DrawFinalScreen(p.final, screen, geometry)
		actualBase := make([]byte, len(p.pixels))
		p.final.ReadPixels(actualBase)
		g.modernHDText = frame
		expectedBase, err := p.expected(c, &empty)
		if err != nil {
			p.err = err
			return
		}
		for i, v := range actualBase {
			if v != expectedBase.Pix[i] {
				p.dumpFailure(c, frame, screen, presentation, want, geometry)
				p.err = fmt.Errorf("case%d native/chrome base differs at byte%d: %d/%d", p.index, i, v, expectedBase.Pix[i])
				return
			}
		}
		// Independent integer source-over of the separately verified GPU glyph
		// surface over the independently rebuilt source/chrome background.
		want = expectedBase
		for at := 0; at < len(want.Pix); at += 4 {
			inverse := 255 - int(gpuGlyphs[at+3])
			for channel := 0; channel < 3; channel++ {
				value := int(gpuGlyphs[at+channel]) + (int(want.Pix[at+channel])*inverse+127)/255
				if value > 255 {
					value = 255
				}
				want.Pix[at+channel] = byte(value)
			}
			want.Pix[at+3] = 255
		}
	}
	for i, v := range p.pixels {
		tolerance := 0
		if gpuGlyphs != nil && gpuGlyphs[i/4*4+3] != 0 {
			tolerance = 1
		}
		delta := int(v) - int(want.Pix[i])
		if delta < -tolerance || delta > tolerance {
			p.dumpFailure(c, frame, screen, presentation, want, geometry)
			p.err = fmt.Errorf("case%d final composition differs at byte%d: %d/%d tolerance%d", p.index, i, v, want.Pix[i], tolerance)
			return
		}
	}
	if titleInterfaceGPUState(g) != before {
		p.err = fmt.Errorf("Draw changed source or gameplay")
		return
	}
	for slot, raw := range slotsBefore {
		after, err := os.ReadFile(saveSlotPath(slot))
		if err != nil && !os.IsNotExist(err) {
			p.err = err
			return
		}
		if !bytes.Equal(raw, after) {
			p.err = fmt.Errorf("Draw changed slot%d", slot)
			return
		}
	}
	row := map[string]any{"phase": c.phase, "selection": c.selection, "flash": c.flash, "dataset": c.dataset, "art_hd": c.art, "ui_hd": c.ui, "locale": c.locale, "font_scale": c.scale, "native_base_tolerance": 0, "glyph_layer_tolerance": 2, "final_glyph_composite_tolerance": 1, "source_and_save_unchanged": true}
	if c.scale == 1 && c.locale == "zh-Hant" && ((c.phase == "menu" && c.selection == 0 && c.flash == 0) || (c.phase == "loadslots" && c.selection == 1 && c.dataset == 1)) {
		name := fmt.Sprintf("%s-art%t-ui%t.png", c.phase, c.art, c.ui)
		file, err := os.Create(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, p.pixels)
		err = png.Encode(file, pic)
		file.Close()
		if err != nil {
			p.err = err
			return
		}
		raw, _ := os.ReadFile(filepath.Join(p.out, name))
		row["file"], row["sha256"] = name, fmt.Sprintf("%x", sha256.Sum256(raw))
	}
	p.rows = append(p.rows, row)
	p.index++
	raw, _ := json.Marshal(row)
	file, err := os.OpenFile(filepath.Join(p.out, "progress.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		p.err = err
		return
	}
	_, p.err = file.Write(append(raw, '\n'))
	file.Close()
}

func (p *titleInterfaceGPUProbe) dumpFailure(c titleInterfaceGPUCase, frame *modernHDTextFrame, screen *ebiten.Image, presentation *modernHDPresentation, want *image.RGBA, geometry ebiten.GeoM) {
	dump := func(name string, raw []byte) {
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, raw)
		file, err := os.Create(filepath.Join(p.out, name))
		if err == nil {
			_ = png.Encode(file, pic)
			_ = file.Close()
		}
	}
	stats := func(a, b []byte) map[string]any {
		maxDelta, changed := 0, 0
		hist := map[int]int{}
		for i, x := range a {
			d := int(x) - int(b[i])
			if d < 0 {
				d = -d
			}
			if d > 0 {
				changed++
				hist[d]++
			}
			if d > maxDelta {
				maxDelta = d
			}
		}
		return map[string]any{"max_delta": maxDelta, "changed_bytes": changed, "absolute_delta_histogram": hist}
	}
	actual := append([]byte(nil), p.pixels...)
	cpuText := titleInterfaceGPUText(p.g, frame)
	gpuText := make([]byte, len(actual))
	presentation.textSurface.ReadPixels(gpuText)
	dump("failure-actual.png", actual)
	dump("failure-expected.png", want.Pix)
	dump("failure-cpu-text.png", cpuText.Pix)
	dump("failure-gpu-text.png", gpuText)
	empty := *frame
	empty.runs = nil
	p.g.modernHDText = &empty
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	baseline := make([]byte, len(actual))
	p.final.ReadPixels(baseline)
	p.g.modernHDText = frame
	baselineExpected, err := p.expected(c, &empty)
	if err != nil {
		return
	}
	dump("failure-base-actual.png", baseline)
	dump("failure-base-expected.png", baselineExpected.Pix)
	samples := []map[string]any{}
	for at := 0; at < len(actual); at += 4 {
		mismatch := false
		for channel := 0; channel < 4; channel++ {
			d := int(actual[at+channel]) - int(want.Pix[at+channel])
			if d < -2 || d > 2 {
				mismatch = true
			}
		}
		if mismatch && len(samples) < 32 {
			rgba := func(b []byte) []int { return []int{int(b[at]), int(b[at+1]), int(b[at+2]), int(b[at+3])} }
			samples = append(samples, map[string]any{"x": (at / 4) % 1920, "y": (at / 4) / 1920, "actual": rgba(actual), "expected": rgba(want.Pix), "gpu_text": rgba(gpuText), "cpu_text": rgba(cpuText.Pix), "base_actual": rgba(baseline), "base_expected": rgba(baselineExpected.Pix)})
		}
	}
	report := map[string]any{"scope": "diagnostic only; no passed GPU claim", "final": stats(actual, want.Pix), "base": stats(baseline, baselineExpected.Pix), "text_layer": stats(gpuText, cpuText.Pix), "samples": samples}
	raw, _ := json.MarshalIndent(report, "", "  ")
	_ = os.WriteFile(filepath.Join(p.out, "failure-diagnostics.json"), append(raw, '\n'), 0644)
}

func TestModernHDTitleInterfaceRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_TITLE_INTERFACE_GPU") != "1" {
		t.Skip("dedicated title interface GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_NATIVE_SAVE", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	out := os.Getenv("FD2_HD_TITLE_INTERFACE_GPU_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	set := modifierHDTestTheme(t)
	a := titleInterfaceTestAssets(t)
	slots, err := loadNativeLoadSlotsUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g, _ := rewardStageGame(t, set)
	// Use the existing complete native map owner required by actual F2 theme
	// switching, even while the title covers its world layer.
	g.titleAssets, g.nativeLoadSlotsUI = a, slots
	g.modernThemeCache, g.interfaceMode, g.fontScale, g.localeID = set, playerInterfaceOriginal, 1, "zh-Hant"
	// Normal startup has already staged the first story fade behind the title.
	// Its zero brightness must not hide the title menu's independent HD text.
	g.fade = &storyFade{total: storyFadeFrames}
	source := filepath.Join("..", "..", "..", "work", "ch01-flow-20261017", "remake-full-r10", "hotel-save.json")
	// The complete existing hotel JSON is copied unchanged into temporary slots.
	sourceRaw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var save saveData
	if err := json.Unmarshal(sourceRaw, &save); err != nil || save.Chapter <= 0 {
		t.Fatal("source save invalid", err)
	}
	p := &titleInterfaceGPUProbe{t: t, g: g, set: set, out: out, final: ebiten.NewImage(1920, 1200), pixels: make([]byte, 1920*1200*4), slotSource: sourceRaw, activeDataset: -1}
	for _, art := range []bool{false, true} {
		for _, ui := range []bool{false, true} {
			for selection := 0; selection < 3; selection++ {
				for _, flash := range []int{0, 3} {
					p.cases = append(p.cases, titleInterfaceGPUCase{phase: "menu", selection: selection, flash: flash, art: art, ui: ui, locale: "zh-Hant", scale: 1})
				}
			}
			for dataset := 0; dataset < 2; dataset++ {
				for selection := 0; selection < 4; selection++ {
					p.cases = append(p.cases, titleInterfaceGPUCase{phase: "loadslots", selection: selection, dataset: dataset, art: art, ui: ui, locale: "zh-Hant", scale: 1})
				}
			}
		}
	}
	for _, phase := range []string{"menu", "loadslots"} {
		for _, locale := range []string{"zh-Hans", "en", "ja"} {
			p.cases = append(p.cases, titleInterfaceGPUCase{phase: phase, selection: 1, dataset: 1, art: true, ui: true, locale: locale, scale: 1})
		}
		for _, scale := range []float64{0.6, 1.6} {
			p.cases = append(p.cases, titleInterfaceGPUCase{phase: phase, selection: 1, dataset: 1, art: true, ui: true, locale: "zh-Hant", scale: scale})
		}
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if p.index != len(p.cases) {
		t.Fatal("incomplete GPU matrix", p.index, len(p.cases))
	}
	code, count := hdConsumerSourceTree(t)
	report := map[string]any{"status": "PASS-TITLE-INTERFACE-GPU", "frames": p.index, "display_modes": 4, "rows": p.rows, "source_tree_sha256": code, "source_file_count": count, "slot_source_sha256": fmt.Sprintf("%x", sha256.Sum256(sourceRaw)), "slot_source_chapter": save.Chapter, "save_bytes_unchanged": true, "native_base_tolerance": 0, "glyph_layer_tolerance": 2, "final_glyph_composite_tolerance": 1, "scope": "actual Game.Draw/DrawFinalScreen; controlled title states and complete existing hotel JSON copied to temporary slots; no normal keyboard route or new original oracle"}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "title-gpu-report.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("complete title/slots Draw=%d; four controls; all slots and original source bytes kept", p.index)
}
