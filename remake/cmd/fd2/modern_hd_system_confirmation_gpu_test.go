package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type systemGPUCase struct {
	caller               int
	phase                fullRewardPhase
	frame, choice, pulse int
	accepted, art, ui    bool
	locale               string
}
type systemGPUProbe struct {
	t             *testing.T
	set           *modernStoryPortraitSet
	g             *Game
	state         *nativeSystemEndTurnUIState
	cases         []systemGPUCase
	index, caller int
	out           string
	rows          []map[string]any
	err           error
	final         *ebiten.Image
	pixels        []byte
	saves         []byte
	worldChecked  bool
}

func (p *systemGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *systemGPUProbe) Update() error {
	if p.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	return nil
}

func systemGPUCPU(g *Game, base []byte, frame *modernHDTextFrame, overlay *modernHDOverlay) *image.RGBA {
	result := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	copy(result.Pix, base)
	if frame != nil {
		colors := nativeDialogueChromeSource(g.nativePreparationUI.dialogue, g.nativeClassUI.palette)
		for _, panel := range frame.panels {
			colors.Gold, colors.Brightness = panel.gold, 1
			r := image.Rectangle{Min: panel.rect.Min.Mul(6), Max: panel.rect.Max.Mul(6)}
			for y := max(0, r.Min.Y); y < min(1200, r.Max.Y); y++ {
				for x := max(0, r.Min.X); x < min(1920, r.Max.X); x++ {
					if ink, painted := colors.pixel(x-r.Min.X, y-r.Min.Y); painted {
						result.SetRGBA(x, y, color.RGBA{ink[0], ink[1], ink[2], ink[3]})
					}
				}
			}
		}
		for _, button := range frame.choices {
			cell := g.nativePreparationUI.choices[button.sourceIndex]
			at := func(x, y int) color.RGBA {
				r, gg, b, a := g.nativeClassUI.palette[cell.Pixels[y*24+x]].RGBA()
				return color.RGBA{byte(r >> 8), byte(gg >> 8), byte(b >> 8), byte(a >> 8)}
			}
			r := image.Rectangle{Min: button.rect.Min.Mul(6), Max: button.rect.Max.Mul(6)}
			ink, bg, shadow := at(12, 0), at(3, 3), at(12, 1)
			draw.Draw(result, r, image.NewUniform(ink), image.Point{}, draw.Src)
			draw.Draw(result, r.Inset(3), image.NewUniform(bg), image.Point{}, draw.Src)
			for _, edge := range []image.Rectangle{image.Rect(r.Min.X+3, r.Max.Y-4, r.Max.X-3, r.Max.Y-3), image.Rect(r.Max.X-4, r.Min.Y+3, r.Max.X-3, r.Max.Y-3)} {
				draw.Draw(result, edge, image.NewUniform(shadow), image.Point{}, draw.Src)
			}
			f := g.font
			if regional := f.hdFonts[g.localeID]; regional != nil {
				f = regional
			}
			face, _ := f.faceFor(40)
			catalog, _ := loadOfficialLocale(g.localeID)
			label, _ := catalog.Format([]string{"common.yes", "common.no"}[button.option])
			bounds := text.BoundString(face, label)
			x, y := r.Min.X+(r.Dx()-bounds.Dx())/2-bounds.Min.X, r.Min.Y+(r.Dy()-bounds.Dy())/2+button.pulse*6-bounds.Min.Y
			for i, c := range []color.RGBA{shadow, ink} {
				offset := 1 - i
				d := font.Drawer{Dst: result.SubImage(r.Inset(12)).(*image.RGBA), Src: image.NewUniform(c), Face: face, Dot: fixed.P(x+offset, y+offset)}
				d.DrawString(label)
			}
		}
	}
	if overlay != nil {
		r := image.Rectangle{Min: overlay.rect.Min.Mul(3), Max: overlay.rect.Max.Mul(3)}
		clip := image.Rectangle{Min: overlay.clip.Min.Mul(3), Max: overlay.clip.Max.Mul(3)}.Intersect(r).Intersect(result.Bounds())
		for y := clip.Min.Y; y < clip.Max.Y; y++ {
			for x := clip.Min.X; x < clip.Max.X; x++ {
				sx := x - r.Min.X
				if overlay.mirrorX {
					sx = r.Dx() - 1 - sx
				}
				result.Set(x, y, overlay.portrait.hdImage.At(sx, y-r.Min.Y))
			}
		}
	}
	if frame != nil {
		for _, run := range frame.runs {
			f := run.font
			if regional := f.hdFonts[g.localeID]; regional != nil {
				f = regional
			}
			face, asc := f.faceFor(int(math.Round(run.pixels * 3)))
			r := image.Rectangle{Min: run.clip.Min.Mul(3), Max: run.clip.Max.Mul(3)}.Intersect(result.Bounds())
			if r.Empty() {
				continue
			}
			d := font.Drawer{Dst: result.SubImage(r).(*image.RGBA), Src: image.NewUniform(run.color), Face: face, Dot: fixed.P(int(math.Round(run.x*3)), int(math.Round(run.y*3+asc)))}
			d.DrawString(run.text)
		}
	}
	return result
}

func (p *systemGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == len(p.cases) {
		return
	}
	c := p.cases[p.index]
	g := p.g
	if p.caller != c.caller {
		g.nativeSystemEndTurnUI = nil
		g.nativeClassUIJob = nil
		g.nativeSystemEndTurnDelay = 0
		g.nativeSystemEndTurnConfirm = false
		g.interfaceMode = playerInterfaceHD
		if err := g.applyTheme(p.set); err != nil {
			p.err = err
			return
		}
		p.state = systemHDTestState(p.t, g, c.caller)
		g.nativeSystemEndTurnUI = p.state
		p.caller = c.caller
		p.worldChecked = false
	}
	s := p.state
	g.nativeClassUIJob = nil
	g.nativeSystemEndTurnDelay = 0
	g.nativeSystemEndTurnConfirm = c.phase == fullRewardChoice
	s.choice = c.choice
	s.acceptedOutcome = c.accepted
	g.nativeClassUIPulse = c.pulse * 2
	s.hd.kind = c.phase
	var frames [][]byte
	switch c.phase {
	case fullRewardOpening:
		frames, _ = campaign.NativePreparationConfirmationOpeningFrames(s.source, s.dialogue, s.question, g.nativePreparationUI.choices)
	case fullRewardChoiceClosing:
		frames, _ = campaign.NativeClassConfirmationClosingFrames(s.question, g.nativePreparationUI.choices)
	case fullRewardBlankClosing:
		frames, _ = campaign.NativeClassListClosingFrames(s.source, s.dialogue)
	case fullRewardResponse:
		frames = s.canceled
		if c.accepted {
			frames = s.accepted
		}
	}
	if c.phase != fullRewardChoice {
		if c.frame == -1 {
			g.nativeSystemEndTurnDelay = 1
		} else {
			index := c.frame
			if index == -2 {
				index = len(frames) - 1
			}
			g.nativeClassUIJob = &nativeClassUIJob{frames: frames, frame: index}
			if c.phase == fullRewardBlankClosing {
				g.nativeClassUIJob.restore = s.source
			}
			s.hd.job = g.nativeClassUIJob
		}
	}
	art := p.set
	if !c.art {
		art = nil
	}
	mode := playerInterfaceOriginal
	if c.ui {
		mode = playerInterfaceHD
	}
	if g.modernStoryPortraits != art {
		if err := g.applyTheme(art); err != nil {
			p.err = err
			return
		}
	}
	if g.interfaceMode != mode {
		if err := g.setInterfaceMode(mode); err != nil {
			p.err = err
			return
		}
	}
	roundtrip := c.phase == fullRewardOpening && c.frame == 6
	if roundtrip {
		for _, set := range []*modernStoryPortraitSet{nil, p.set} {
			if err := g.applyTheme(set); err != nil {
				p.err = err
				return
			}
		}
		for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD} {
			if err := g.setInterfaceMode(mode); err != nil {
				p.err = err
				return
			}
		}
	}
	g.localeID = c.locale
	ack := false
	if job := g.nativeClassUIJob; job != nil {
		ack = job.drawn
		job.drawn = true
	}
	before := fullRewardGPUState(g)
	if job := g.nativeClassUIJob; job != nil {
		job.drawn = ack
	}
	g.Draw(screen)
	if g.loadErr != "" {
		p.err = fmt.Errorf("case%d: %s", p.index, g.loadErr)
		return
	}
	if c.art && !p.worldChecked {
		want := healHDCPUWorld(s.hd.world, p.set)
		actual := make([]byte, len(want.Pix))
		g.modernHDMapGPU.World.ReadPixels(actual)
		for i, v := range actual {
			delta := int(v) - int(want.Pix[i])
			if delta < -2 || delta > 2 {
				p.err = fmt.Errorf("frozen world CPU differs at%d", i)
				return
			}
		}
		p.worldChecked = true
	}
	textFrame, overlay := g.modernHDText, g.modernHDOverlay
	if (textFrame != nil) != c.ui || (overlay != nil) != c.art && c.phase == fullRewardChoice {
		p.err = fmt.Errorf("case%d control followed wrong mode", p.index)
		return
	}
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	presentation := &modernHDPresentation{Game: g}
	g.modernHDText, g.modernHDOverlay = nil, nil
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	base := make([]byte, len(p.pixels))
	p.final.ReadPixels(base)
	g.modernHDText, g.modernHDOverlay = textFrame, overlay
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(p.pixels)
	expected := systemGPUCPU(g, base, textFrame, overlay)
	for i, v := range p.pixels {
		d := int(v) - int(expected.Pix[i])
		if d < -2 || d > 2 {
			p.err = fmt.Errorf("case%d caller%d phase%d frame%d UI%v art%v final differs at%d,%d c%d got%d want%d", p.index, c.caller, c.phase, c.frame, c.ui, c.art, (i/4)%1920, (i/4)/1920, i%4, v, expected.Pix[i])
			return
		}
	}
	if !reflect.DeepEqual(before, fullRewardGPUState(g)) {
		p.err = fmt.Errorf("case%d Draw changed native state", p.index)
		return
	}
	row := map[string]any{"caller": c.caller, "phase": c.phase, "frame": c.frame, "choice": c.choice, "pulse": c.pulse, "accepted": c.accepted, "art_hd": c.art, "ui_hd": c.ui, "locale": c.locale, "full_rgba_pixels": 1920 * 1200, "rgba_tolerance": 2, "source_state_unchanged": true}
	row["f2_f11_roundtrip"] = roundtrip
	if c.phase == fullRewardChoice && c.choice == 0 && c.pulse == 0 && (c.art == c.ui || c.caller == 0) {
		name := fmt.Sprintf("caller%d-art%t-ui%t-%s.png", c.caller, c.art, c.ui, c.locale)
		path := filepath.Join(p.out, name)
		f, err := os.Create(path)
		if err != nil {
			p.err = err
			return
		}
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, p.pixels)
		err = png.Encode(f, pic)
		f.Close()
		if err != nil {
			p.err = err
			return
		}
		raw, _ := os.ReadFile(path)
		row["file"], row["sha256"] = name, fmt.Sprintf("%x", sha256.Sum256(raw))
	}
	p.rows = append(p.rows, row)
	progress, err := os.OpenFile(filepath.Join(p.out, "progress.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		p.err = err
		return
	}
	err = json.NewEncoder(progress).Encode(row)
	progress.Close()
	if err != nil {
		p.err = err
		return
	}
	p.index++
}

func TestModernHDSystemConfirmationGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SYSTEM_CONFIRM_GPU") != "1" {
		t.Skip("explicit full system GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	out := os.Getenv("FD2_HD_SYSTEM_CONFIRM_OUT")
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	set := modifierHDTestTheme(t)
	checkSystemConfirmationWorldAndSwitches(t, set)
	g, _ := rewardStageGame(t, set)
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "system_fixture", Nodes: map[string]*campaign.Node{"system_fixture": {Type: "battle"}}})
	g.interfaceMode = playerInterfaceHD
	g.modernThemeCache = set
	g.fontScale = 1
	if err := g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(saveSlotPath(0))
	if err != nil {
		t.Fatal(err)
	}
	g.msg = ""
	p := &systemGPUProbe{t: t, set: set, g: g, caller: -1, out: out, final: ebiten.NewImage(1920, 1200), pixels: make([]byte, 1920*1200*4)}
	for caller := 0; caller < 5; caller++ {
		for _, art := range []bool{false, true} {
			for _, ui := range []bool{false, true} {
				for choice := 0; choice < 2; choice++ {
					for pulse := 0; pulse < 2; pulse++ {
						p.cases = append(p.cases, systemGPUCase{caller: caller, phase: fullRewardChoice, choice: choice, pulse: pulse, art: art, ui: ui, locale: "zh-Hant"})
					}
				}
			}
		}
		for _, phase := range []fullRewardPhase{fullRewardOpening, fullRewardChoiceClosing, fullRewardResponse, fullRewardBlankClosing} {
			indices := []int{-2}
			if phase == fullRewardOpening {
				indices = []int{0, 5, 6, 9}
			}
			if phase == fullRewardChoiceClosing {
				indices = []int{0, 3}
			}
			if phase == fullRewardBlankClosing {
				indices = []int{0, 4, 5}
			}
			for _, index := range indices {
				p.cases = append(p.cases, systemGPUCase{caller: caller, phase: phase, frame: index, art: true, ui: true, locale: "zh-Hant"})
			}
		}
		p.cases = append(p.cases, systemGPUCase{caller: caller, phase: fullRewardResponse, frame: -2, accepted: true, art: true, ui: true, locale: "zh-Hant"})
		if caller == 0 {
			for _, locale := range []string{"zh-Hans", "en", "ja"} {
				for choice := 0; choice < 2; choice++ {
					for pulse := 0; pulse < 2; pulse++ {
						p.cases = append(p.cases, systemGPUCase{caller: 0, phase: fullRewardChoice, choice: choice, pulse: pulse, art: true, ui: true, locale: locale})
					}
				}
			}
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
		t.Fatal(p.index, len(p.cases))
	}
	if err := g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(saveSlotPath(0))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("GPU changed save", err)
	}
	catalog, _ := os.ReadFile(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"))
	manifest, _ := os.ReadFile(os.Getenv("FD2_HD_SYSTEM_PORTRAIT_MANIFEST"))
	code, count := hdConsumerSourceTree(t)
	report := map[string]any{"status": "PASS-SYSTEM-CONFIRMATION-HD-GPU", "catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(catalog)), "portrait_manifest_sha256": fmt.Sprintf("%x", sha256.Sum256(manifest)), "source_tree_sha256": code, "source_file_count": count, "caller_count": 5, "display_modes": 4, "save_bytes_unchanged": len(before), "rows": p.rows, "scope": "controlled map23 and original native question text; four-language YES/NO; not normal player or original parity"}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
