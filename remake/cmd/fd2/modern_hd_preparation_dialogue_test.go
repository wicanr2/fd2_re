package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	xfont "golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func TestNativePreparationPortraitLayoutUsesOriginalBlit(t *testing.T) {
	ui, err := loadNativePreparationUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	if !preparationPortrait75Matches(ui) {
		t.Fatal("canonical DATO75 index fingerprint differs")
	}
	rect, mirror, err := campaign.NativePreparationPortraitLayout(ui.portrait)
	if err != nil || !mirror || rect != image.Rect(8, 115, 88, 195) {
		t.Fatal("portrait layout changed", rect, mirror, err)
	}
	frame, err := campaign.ComposeNativePreparationConfirmationDialogue(make([]byte, 64000), ui.dialogue, ui.portrait)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			if frame[(rect.Min.Y+y)*320+rect.Min.X+x] != ui.portrait.Pixels[y*80+79-x] {
				t.Fatalf("native portrait projection differs (%d,%d)", x, y)
			}
		}
	}
	bad := ui.portrait
	bad.Width--
	if _, _, err := campaign.NativePreparationPortraitLayout(bad); err == nil {
		t.Fatal("invalid portrait shape accepted")
	}
}

func appendPreparationDialogueExtraCases(t *testing.T, p *facilityHDProbe, set *modernStoryPortraitSet, jobs *[]*nativeClassUIJob, before *[][32]byte) {
	t.Helper()
	g := p.g
	camp, town, source := g.camp, g.prepTownSource, append([]byte(nil), g.prepPromptSource...)
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		for choice := 0; choice < 2; choice++ {
			for pulse := 0; pulse < 2; pulse++ {
				locale, choice, pulse := locale, choice, pulse
				if choice == 1 && pulse == 1 {
					p.captureCases[len(p.cases)] = true
				}
				p.cases = append(p.cases, func() {
					g.camp, g.prepTownSource, g.prepPromptSource = camp, town, source
					g.nativeClassUIJob = nil
					preparationHDTestPhase(g, "prompt")
					g.modernStoryPortraits, g.interfaceMode = set, playerInterfaceHD
					g.localeID, g.prepConfirmSel, g.nativeClassUIPulse = locale, choice, pulse*2
				})
			}
		}
	}
	blackCamp := campaign.NewRunner(&campaign.Campaign{Start: "prep", Nodes: map[string]*campaign.Node{"prep": {Type: "preparation"}}})
	g.camp = blackCamp
	g.setupPreparation(g.camp.Node())
	g.nativeClassUIJob = nil
	black := append([]byte(nil), g.prepPromptSource...)
	if g.prepTownSource != nil || !bytes.Equal(black, make([]byte, 64000)) {
		t.Fatal("standalone source was not black")
	}
	setup := func(style int) {
		g.camp, g.prepTownSource, g.prepPromptSource = blackCamp, nil, black
		preparationHDTestPhase(g, "prompt")
		g.localeID, g.prepConfirmSel, g.nativeClassUIPulse = "zh-Hant", 0, 0
		g.modernStoryPortraits = nil
		if style >= 2 {
			g.modernStoryPortraits = set
		}
		g.interfaceMode = playerInterfaceOriginal
		if style%2 == 1 {
			g.interfaceMode = playerInterfaceHD
		}
	}
	for style := 0; style < 4; style++ {
		style := style
		p.captureCases[len(p.cases)] = true
		p.cases = append(p.cases, func() { setup(style); g.nativeClassUIJob = nil })
	}
	for _, closing := range []bool{false, true} {
		setup(3)
		var ok bool
		if closing {
			ok = g.beginNativePreparationPromptClosing(nil)
		} else {
			ok = g.beginNativePreparationPromptOpening()
		}
		if !ok {
			t.Fatal("standalone prompt job unavailable")
		}
		job := g.nativeClassUIJob
		*jobs = append(*jobs, job)
		data, _ := json.Marshal([]any{job.frames, job.restore})
		*before = append(*before, sha256.Sum256(data))
		count := len(job.frames)
		if closing {
			count++
		}
		for frame := 0; frame < count; frame++ {
			frame := frame
			p.cases = append(p.cases, func() { setup(3); g.nativeClassUIJob = job; job.frame, job.drawn = frame, false })
		}
	}
}

func verifyPreparationDialogueGPU(p *facilityHDProbe, pixels []byte) error {
	g := p.g
	job := g.nativeClassUIJob
	shift, visible, buttons := 0, true, false
	if job == nil {
		visible = g.prepConfirm || g.nativePreparationPromptActive()
		buttons = visible && (g.prepConfirm || g.prepRequiredMissing == nil)
	} else {
		d := job.preparationText
		visible = d != nil && job.frame < len(job.frames)
		if visible {
			if !d.closing {
				if job.frame < 6 {
					shift = (5 - job.frame) * 13
				} else {
					buttons = d.kind != nativePreparationRequiredQuestion
				}
			} else if d.kind == nativePreparationRequiredQuestion {
				shift = (job.frame + 1) * 13
			} else if job.frame < 4 {
				buttons = true
			} else {
				shift = (job.frame - 3) * 13
			}
		}
	}
	wantHead := visible && g.modernStoryPortraits != nil
	if (g.modernHDOverlay != nil) != wantHead {
		return fmt.Errorf("case%d portrait presence differs", p.index)
	}
	heads, gold, choicePixels := 0, 0, 0
	if wantHead {
		o := g.modernHDOverlay
		if !o.mirrorX || o.portrait != g.modernStoryPortraits.portraits[75] || o.rect != image.Rect(16, (115+shift)*2, 176, (195+shift)*2) {
			return fmt.Errorf("case%d wrong portrait identity/geometry", p.index)
		}
		visible := o.rect.Intersect(o.clip)
		for y := visible.Min.Y * 3; y < visible.Max.Y*3; y += 3 {
			for x := visible.Min.X * 3; x < visible.Max.X*3; x += 3 {
				want := color.NRGBAModel.Convert(o.portrait.hdImage.At(o.rect.Max.X*3-1-x, y-o.rect.Min.Y*3)).(color.NRGBA)
				at := ((y+72)*1920 + x) * 4
				if !bytes.Equal(pixels[at:at+4], []byte{want.R, want.G, want.B, want.A}) {
					return fmt.Errorf("case%d HD75 source differs (%d,%d)", p.index, x, y)
				}
				heads++
			}
		}
		if heads < 1000 {
			return fmt.Errorf("case%d insufficient HD75 pixels: %d", p.index, heads)
		}
	}
	if g.interfaceMode == playerInterfaceHD {
		frame := g.modernHDText
		if frame == nil {
			return fmt.Errorf("case%d HD UI frame absent", p.index)
		}
		if visible && (len(frame.panels) != 1 || !frame.panels[0].gold) || !visible && len(frame.panels) != 0 {
			return fmt.Errorf("case%d wrong gold dialogue presence", p.index)
		}
		if buttons && len(frame.choices) != 2 || !buttons && len(frame.choices) != 0 {
			return fmt.Errorf("case%d wrong confirmation presence", p.index)
		}
		for _, panel := range frame.panels {
			picture := panel.chrome.goldPatchFor(panel.rect.Dx()*6, panel.rect.Dy()*6)
			rect := image.Rectangle{Min: panel.rect.Min.Mul(6), Max: panel.rect.Max.Mul(6)}
			area := rect.Intersect(image.Rect(0, 0, 1920, 1200))
			for y := area.Min.Y; y < area.Max.Y; y += 7 {
				for x := area.Min.X; x < area.Max.X; x += 7 {
					want := picture.RGBAAt(x-rect.Min.X, y-rect.Min.Y)
					if want.A != 255 {
						continue
					}
					at := ((y+72)*1920 + x) * 4
					if !bytes.Equal(pixels[at:at+4], []byte{want.R, want.G, want.B, 255}) {
						return fmt.Errorf("case%d gold material differs (%d,%d)", p.index, x, y)
					}
					gold++
				}
			}
			if gold < 100 {
				return fmt.Errorf("case%d insufficient gold checks", p.index)
			}
		}
		for i, button := range frame.choices {
			want := button.cell.patch()
			face, label, atText := button.owner.labelLayout(button)
			target := want.SubImage(want.Bounds().Inset(12)).(*image.RGBA)
			for layer, ink := range []color.RGBA{button.cell.shadow, button.cell.ink} {
				offset := 1 - layer
				drawer := &xfont.Drawer{Dst: target, Src: image.NewUniform(ink), Face: face, Dot: fixed.P(atText.X+offset, atText.Y+offset)}
				drawer.DrawString(label)
			}
			origin := button.rect.Min.Mul(6)
			for y := 0; y < want.Bounds().Dy(); y += 3 {
				for x := 0; x < 144; x += 3 {
					point := image.Pt(origin.X+x, origin.Y+y)
					covered := false
					for _, later := range frame.choices[i+1:] {
						if point.In(image.Rectangle{Min: later.rect.Min.Mul(6), Max: later.rect.Max.Mul(6)}) {
							covered = true
						}
					}
					if covered {
						continue
					}
					c := want.RGBAAt(x, y)
					at := ((point.Y+72)*1920 + point.X) * 4
					for channel, value := range []byte{c.R, c.G, c.B, 255} {
						if delta := int(pixels[at+channel]) - int(value); delta < -2 || delta > 2 {
							return fmt.Errorf("case%d locale%s choice%d differs (%d,%d)", p.index, button.locale, button.option, x, y)
						}
					}
					choicePixels++
				}
			}
		}
	}
	if p.index >= 223 {
		if g.prepTownSource != nil || g.modernHDScene != nil || !bytes.Equal(g.prepPromptSource, make([]byte, 64000)) {
			return fmt.Errorf("standalone source became town")
		}
		c := color.NRGBAModel.Convert(g.nativeClassUI.palette[0]).(color.NRGBA)
		at := ((20*6+72)*1920 + 200*6) * 4
		if !bytes.Equal(pixels[at:at+4], []byte{c.R, c.G, c.B, 255}) {
			return fmt.Errorf("standalone background changed")
		}
	}
	check := p.checks[len(p.checks)-1]
	check["portrait75_pixels"], check["gold_pixels"], check["confirmation_pixels"] = heads, gold, choicePixels
	check["locale"], check["choice"], check["pulse"] = g.localeID, g.prepConfirmSel, g.nativeClassUIPulse/2
	return nil
}

// Shares the complete formal theme already loaded by the background owner
// regression, avoiding another large pack load in the same Go process.
func verifyPreparationDialogueCPU(t *testing.T, g *Game, set *modernStoryPortraitSet) {
	t.Helper()
	oldText, oldOverlay, oldArt, oldCache := g.modernHDText, g.modernHDOverlay, g.modernStoryPortraits, g.prepDialogueHD
	oldSelecting, oldConfirm, oldSlots, oldRequired := g.prepSelecting, g.prepConfirm, g.prepRecordSlots, g.prepRequiredMissing
	oldChoice, oldPulse, oldLocale := g.prepConfirmSel, g.nativeClassUIPulse, g.localeID
	defer func() {
		g.modernHDText, g.modernHDOverlay, g.modernStoryPortraits, g.prepDialogueHD = oldText, oldOverlay, oldArt, oldCache
		g.prepSelecting, g.prepConfirm, g.prepRecordSlots, g.prepRequiredMissing = oldSelecting, oldConfirm, oldSlots, oldRequired
		g.prepConfirmSel, g.nativeClassUIPulse, g.localeID = oldChoice, oldPulse, oldLocale
	}()
	preparationHDTestPhase(g, "prompt")
	g.prepConfirmSel, g.nativeClassUIPulse = 0, 0
	ui := g.nativePreparationUI
	h, err := g.prepareModernHDPreparationDialogue(ui)
	if err != nil {
		t.Fatal(err)
	}
	copy := *ui
	copy.status.Font = nil
	if again, err := g.prepareModernHDPreparationDialogue(&copy); err != nil || again != h {
		t.Fatal("temporary font adapter invalidated immutable material cache", err)
	}
	for _, fault := range []string{"dialogue-shape", "choice-shape", "nonuniform-fill", "font"} {
		bad := *ui
		font := g.font
		switch fault {
		case "dialogue-shape":
			bad.dialogue = append([]fdother.RawCell(nil), ui.dialogue...)
			bad.dialogue[1].Width--
		case "choice-shape":
			bad.choices = append([]fdother.RawCell(nil), ui.choices...)
			bad.choices[48].Width--
		case "nonuniform-fill":
			bad.dialogue = append([]fdother.RawCell(nil), ui.dialogue...)
			bad.dialogue[13].Pixels = append([]byte(nil), ui.dialogue[13].Pixels...)
			bad.dialogue[13].Pixels[0] ^= 1
		case "font":
			g.font = nil
		}
		got, err := g.prepareModernHDPreparationDialogue(&bad)
		g.font = font
		if err == nil || got != nil || g.prepDialogueHD != h {
			t.Fatal("invalid material published", fault, err)
		}
	}
	for _, art := range []bool{false, true} {
		for _, hdUI := range []bool{false, true} {
			g.modernStoryPortraits = nil
			if art {
				g.modernStoryPortraits = set
			}
			g.modernHDText = nil
			if hdUI {
				g.modernHDText = &modernHDTextFrame{}
			}
			g.appendModernHDPreparationDialogue(ui, nil)
			if (g.modernHDOverlay != nil) != art {
				t.Fatal("portrait followed interface", art, hdUI)
			}
			if hdUI && (len(g.modernHDText.panels) != 1 || !g.modernHDText.panels[0].gold || len(g.modernHDText.choices) != 2) {
				t.Fatal("HD dialogue materials absent", art)
			}
		}
	}
	g.modernStoryPortraits = set
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		for choice := 0; choice < 2; choice++ {
			for pulse := 0; pulse < 2; pulse++ {
				g.localeID, g.prepConfirmSel, g.nativeClassUIPulse = locale, choice, pulse*2
				g.modernHDText = &modernHDTextFrame{}
				g.appendModernHDPreparationDialogue(ui, nil)
				if len(g.modernHDText.choices) != 2 {
					t.Fatal("localized choices absent", locale)
				}
				for _, button := range g.modernHDText.choices {
					if button.locale != locale || button.owner.labels[locale][button.option] == "" {
						t.Fatal("wrong locale", locale)
					}
				}
			}
		}
	}
	for _, kind := range []nativePreparationQuestionKind{nativePreparationDepartureQuestion, nativePreparationRequiredQuestion} {
		for _, closing := range []bool{false, true} {
			shifts := []int{65, 52, 39, 26, 13, 0, 0, 0, 0, 0}
			if closing {
				shifts = []int{0, 0, 0, 0, 13, 26, 39, 52, 65}
			}
			if kind == nativePreparationRequiredQuestion {
				shifts = shifts[:7]
				if closing {
					shifts = []int{13, 26, 39, 52, 65}
				}
			}
			d := &nativePreparationTransitionText{kind: kind, start: 6, closing: closing}
			if closing {
				d.start = 0
			}
			job := &nativeClassUIJob{frames: make([][]byte, len(shifts)), preparationText: d}
			if closing {
				job.restore = make([]byte, 64000)
			}
			for frame, shift := range shifts {
				job.frame = frame
				g.modernHDText = &modernHDTextFrame{}
				g.appendModernHDPreparationDialogue(ui, job)
				if o := g.modernHDOverlay; o == nil || !o.mirrorX || o.rect != image.Rect(16, (115+shift)*2, 176, (195+shift)*2) {
					t.Fatal("portrait shift/mirror changed", frame, shift, o)
				}
				buttons := kind != nativePreparationRequiredQuestion && ((!closing && frame >= 6) || (closing && frame < 4))
				if (len(g.modernHDText.choices) == 2) != buttons {
					t.Fatal("buttons appeared in blank dialogue stage", frame, closing, kind)
				}
			}
			job.frame = len(job.frames)
			g.modernHDText = &modernHDTextFrame{}
			g.appendModernHDPreparationDialogue(ui, job)
			if g.modernHDOverlay != nil || len(g.modernHDText.panels) != 0 || len(g.modernHDText.choices) != 0 {
				t.Fatal("restore retained prompt artwork")
			}
		}
	}
	for _, phase := range []string{"roster"} {
		preparationHDTestPhase(g, phase)
		g.modernHDText = &modernHDTextFrame{}
		g.appendModernHDPreparationDialogue(ui, nil)
		if g.modernHDOverlay != nil || len(g.modernHDText.panels) != 0 {
			t.Fatal("prompt portrait covered roster")
		}
	}
	preparationHDTestPhase(g, "prompt")
	g.modernHDText = nil
	bad := *ui
	bad.portrait = ui.portrait
	bad.portrait.Pixels = append([]byte(nil), ui.portrait.Pixels...)
	bad.portrait.Pixels[0] ^= 1
	g.appendModernHDPreparationDialogue(&bad, nil)
	if g.modernHDOverlay != nil {
		t.Fatal("wrong native identity reused DATO75 HD")
	}
	portrait := set.portraits[75]
	delete(set.portraits, 75)
	g.appendModernHDPreparationDialogue(ui, nil)
	set.portraits[75] = portrait
	if g.modernHDOverlay != nil {
		t.Fatal("missing DATO75 borrowed another face")
	}
}
