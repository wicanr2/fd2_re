package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestModernHDPreparationTransitionRecipesMatchNativeFrames(t *testing.T) {
	for _, kind := range []string{"record", "departure", "confirmation", "required"} {
		for _, closing := range []bool{false, true} {
			if kind == "required" && closing {
				continue // Its closing frames intentionally contain no question.
			}
			t.Run(kind+map[bool]string{false: "_opening", true: "_closing"}[closing], func(t *testing.T) {
				g := newPreparationRecordTestGame(t)
				class, err := loadNativeClassUIAssets()
				if err != nil {
					t.Fatal(err)
				}
				g.nativeClassUI, g.font = class, loadFont()
				_, g.nativeBattleGlyphs, err = loadNativeBattleNameAssets()
				if err != nil {
					t.Fatal(err)
				}
				identity := 1
				if kind == "departure" {
					g.camp.Node().Cancel = "town"
				}
				if kind == "required" {
					g.prepRequiredMissing = &identity
				}
				called := 0
				after := func() { called++ }
				var ok bool
				switch {
				case kind == "required":
					ok = g.beginNativePreparationRequiredOpening()
				case kind == "confirmation" && closing:
					ok = g.beginNativePreparationConfirmationClosing(after)
				case kind == "confirmation":
					ok = g.beginNativePreparationConfirmationOpening()
				case closing:
					ok = g.beginNativePreparationPromptClosing(after)
				default:
					ok = g.beginNativePreparationPromptOpening()
				}
				if !ok || g.nativeClassUIJob.preparationText == nil {
					t.Fatal("transition recipe unavailable")
				}
				job := g.nativeClassUIJob
				d := job.preparationText
				before := make([][]byte, len(job.frames))
				for i, frame := range job.frames {
					before[i] = append([]byte(nil), frame...)
					composed, active := d.compose(i, d.assets.status.Font)
					expected := i >= d.start && i < d.start+4 && (kind != "required" || i == d.start)
					if active != expected || (active && !bytes.Equal(composed, frame)) {
						t.Fatalf("native frame %d differs from its recipe", i)
					}
				}
				// Mutating the live owner must not retarget an existing job.
				g.prepPromptSource[0] ^= 255
				g.prepRequiredMissing = nil
				g.nativePreparationUI.status.Font = nil
				g.camp.Node().Cancel = "other"
				identity = 31
				for i := range job.frames {
					composed, active := d.compose(i, d.assets.status.Font)
					if active && !bytes.Equal(composed, before[i]) {
						t.Fatal("recipe changed with live owner")
					}
				}
				screen := ebiten.NewImage(logicalW, logicalH)
				job.frame = d.start
				for _, hd := range []bool{true, false, true} {
					g.modernHDText = nil
					if hd {
						g.modernHDText = &modernHDTextFrame{}
					}
					if !g.drawNativeClassUIJob(screen) || job.frame != d.start || !job.drawn {
						t.Fatal("draw/theme switch advanced transition")
					}
					if hd && len(g.modernHDText.runs) == 0 {
						t.Fatal("question stage has no HD text")
					}
					if g.font.indexedHD != nil {
						t.Fatal("transition adapter leaked")
					}
				}
				job.frame, job.drawn = 0, false
				count := 0
				for g.nativeClassUIJob != nil && count < 16 {
					g.modernHDText = &modernHDTextFrame{}
					if count%2 == 1 {
						g.modernHDText = nil
					}
					if !g.drawNativeClassUIJob(screen) {
						t.Fatal("native transition stopped drawing")
					}
					g.stepNativeClassUILifecycle(time.Time{})
					count++
				}
				wantCount := len(job.frames)
				if closing {
					wantCount++ // The caller-owned restore is acknowledged too.
				}
				if g.nativeClassUIJob != nil || count != wantCount || called != map[bool]int{false: 0, true: 1}[closing] {
					t.Fatalf("changed schedule/continuation: frames=%d callbacks=%d", count, called)
				}
				for i := range before {
					if !bytes.Equal(before[i], job.frames[i]) {
						t.Fatal("HD modified indexed cache")
					}
				}
			})
		}
	}
}

func preparationHDTestPhase(g *Game, phase string) {
	g.prepSelecting, g.prepConfirm, g.prepRecordSlots = false, false, false
	g.prepRequiredMissing = nil
	switch phase {
	case "roster":
		g.prepSelecting = true
	case "confirmation":
		g.prepConfirm = true
	case "required":
		identity := 1
		g.prepRequiredMissing = &identity
	}
}

func TestModernHDPreparationBackgroundFrozenSourceAndFallback(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	assets, err := loadNativePreparationUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativePreparationUI = assets
	townHDDepartureTestOwner(t, g, 2)
	g.nativeTownDeparture = nil
	g.setupPreparation(g.camp.Node())
	job := g.nativeClassUIJob
	town := g.prepTownSource
	source := append([]byte(nil), g.prepPromptSource...)
	data, _ := json.Marshal([]any{job.frames, job.restore})
	before := sha256.Sum256(data)
	class, prep := g.nativeClassUI, g.nativePreparationUI
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	unit := g.partyRoster[g.partyJoinOrder[0]]
	unit.MapSelectorKey = 118
	g.partyRoster[g.partyJoinOrder[0]] = unit
	for _, frame := range []int{0, 6, 9} {
		job.frame, job.drawn = frame, false
		for i := 0; i < 2; i++ {
			art, ui := g.modernStoryPortraits, g.interfaceMode
			g.cycleTheme()
			g.cycleInterfaceMode()
			g.Draw(screen)
			if (art == nil) == (g.modernStoryPortraits == nil) || ui == g.interfaceMode {
				t.Fatal("theme/interface shortcut failed", g.msg)
			}
			if job.frame != frame || g.nativeClassUIJob != job || !job.drawn || g.prepTownSource != town || !bytes.Equal(g.prepPromptSource, source) {
				t.Fatal("theme switch changed frozen owner")
			}
			if o := g.modernHDScene; o != nil && (o.cursor == nil || o.cursor.group != 50) {
				t.Fatal("live party retargeted town source")
			}
		}
	}
	t.Run("dialogue-components", func(t *testing.T) { verifyPreparationDialogueCPU(t, g, set) })
	t.Run("roster-components", func(t *testing.T) { verifyPreparationRosterCPU(t, g, set) })
	g.nativeClassUIJob = nil
	g.modernStoryPortraits = set
	g.interfaceMode = playerInterfaceOriginal
	preparationHDTestPhase(g, "prompt")
	originalPictures := set.mapSpritesHD[50]
	for _, fault := range []string{"missing-descriptor", "source-mismatch", "missing-hd", "geometry"} {
		g.prepTownSource, g.prepPromptSource, set.mapSpritesHD[50] = town, source, originalPictures
		switch fault {
		case "missing-descriptor":
			g.prepTownSource = nil
		case "source-mismatch":
			g.prepPromptSource = append([]byte(nil), source...)
			g.prepPromptSource[0] ^= 255
		case "missing-hd":
			delete(set.mapSpritesHD, 50)
		case "geometry":
			set.mapSpritesHD[50] = []image.Image{nil, image.NewNRGBA(image.Rect(0, 0, 72, 72))}
		}
		g.Draw(screen)
		if g.modernHDScene != nil || g.loadErr != "" {
			t.Fatal("missing provenance/HD did not retain native preparation", fault, g.loadErr)
		}
		if _, ok := g.composeNativePreparationStableFrame(); !ok {
			t.Fatal("fallback lost native frame", fault)
		}
	}
	set.mapSpritesHD[50] = originalPictures
	g.camp.Node().Cancel = ""
	g.setupPreparation(g.camp.Node())
	g.nativeClassUIJob = nil
	g.Draw(screen)
	if g.prepTownSource != nil || !bytes.Equal(g.prepPromptSource, make([]byte, 64000)) || g.modernHDScene != nil {
		t.Fatal("standalone black source became a town")
	}
	after, _ := json.Marshal([]any{job.frames, job.restore})
	if sha256.Sum256(after) != before || g.nativeClassUI != class || g.nativePreparationUI != prep || g.font.indexedHD != nil {
		t.Fatal("native cache or adapter changed")
	}
}

func verifyPreparationHDScene(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
	g := p.g
	base := make([]byte, 640*424*4)
	offscreen.ReadPixels(base)
	checks := map[string]any{"case": p.index, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil}
	if job := g.nativeClassUIJob; job != nil {
		checks["frame"] = job.frame
		checks["frames"] = len(job.frames)
	}
	backgrounds, cores, nativeChecks, finalChecks := 0, 0, 0, 0
	if g.modernStoryPortraits != nil && g.prepTownSource != nil {
		o := g.modernHDScene
		if o == nil || o.cursor == nil || o.cursor.group != 50 || o.cursor.frame != 1 {
			return fmt.Errorf("case%d missing frozen town source", p.index)
		}
		inText := func(x, y int) bool {
			if g.modernHDText != nil {
				for _, run := range g.modernHDText.runs {
					if image.Pt(x/3, y/3).In(run.clip) {
						return true
					}
				}
			}
			return false
		}
		for y := 24; y < 1176; y += 11 {
			for x := 24; x < 1896; x += 11 {
				if base[((y/3+24)*640+x/3)*4+3] != 0 || inText(x, y) {
					continue
				}
				if image.Pt(x, y).In(o.cursor.rect.Inset(-2)) {
					continue
				}
				want := color.NRGBAModel.Convert(o.scene.image.At(x-24, y-24)).(color.NRGBA)
				at := ((y+72)*1920 + x) * 4
				if !bytes.Equal(pixels[at:at+4], []byte{want.R, want.G, want.B, 255}) {
					return fmt.Errorf("case%d HD backdrop mismatch (%d,%d)", p.index, x, y)
				}
				backgrounds++
			}
		}
		if backgrounds < 300 {
			return fmt.Errorf("case%d too few exposed backdrop pixels: %d", p.index, backgrounds)
		}
		cursor := g.modernStoryPortraits.mapSpritesHD[o.cursor.group][o.cursor.frame]
		visible := o.cursor.rect.Intersect(o.cursor.clip)
		for y := visible.Min.Y + 1; y < visible.Max.Y-1; y += 2 {
			for x := visible.Min.X + 1; x < visible.Max.X-1; x += 2 {
				if base[((y/3+24)*640+x/3)*4+3] != 0 || inText(x, y) {
					continue
				}
				want := color.NRGBAModel.Convert(cursor.At(x-o.cursor.rect.Min.X, y-o.cursor.rect.Min.Y)).(color.NRGBA)
				if want.A < 250 {
					continue
				}
				background := color.NRGBAModel.Convert(o.scene.image.At(x-24, y-24)).(color.NRGBA)
				at := ((y+72)*1920 + x) * 4
				for c, values := range [][2]byte{{want.R, background.R}, {want.G, background.G}, {want.B, background.B}} {
					v := (int(values[0])*int(want.A) + int(values[1])*(255-int(want.A)) + 127) / 255
					if delta := int(pixels[at+c]) - v; delta < -2 || delta > 2 {
						return fmt.Errorf("case%d visible cursor mismatch (%d,%d)", p.index, x, y)
					}
				}
				cores++
			}
		}
	} else if g.modernHDScene != nil {
		return fmt.Errorf("case%d original art retained HD scene", p.index)
	}
	if g.interfaceMode == playerInterfaceOriginal {
		var frame []byte
		if job := g.nativeClassUIJob; job != nil {
			frame = job.restore
			if job.frame < len(job.frames) {
				frame = job.frames[job.frame]
			}
		} else {
			var ok bool
			frame, ok = g.composeNativePreparationStableFrameLayers(g.preparationRosterHD == nil)
			if !ok {
				return fmt.Errorf("native control missing")
			}
		}
		for y := 0; y < 400; y++ {
			for x := 0; x < 640; x++ {
				at := ((y+24)*640 + x) * 4
				if base[at+3] != 255 {
					continue
				}
				want := color.NRGBAModel.Convert(g.nativeClassUI.palette[frame[y/2*320+x/2]]).(color.NRGBA)
				if !bytes.Equal(base[at:at+4], []byte{want.R, want.G, want.B, 255}) {
					return fmt.Errorf("case%d original UI changed (%d,%d)", p.index, x, y)
				}
				final := (((y+24)*3+1)*1920 + x*3 + 1) * 4
				portrait := g.modernHDOverlay
				covered := portrait != nil && image.Pt(x, y).In(portrait.rect.Intersect(portrait.clip))
				if plan := g.preparationRosterHD; plan != nil {
					for _, unit := range plan.units {
						if image.Pt(x*3, y*3).In(unit.Rect) {
							covered = true
							break
						}
					}
				}
				if !covered && !bytes.Equal(base[at:at+4], pixels[final:final+4]) {
					return fmt.Errorf("case%d foreground leaked (%d,%d)", p.index, x, y)
				}
				if !covered {
					finalChecks++
				}
				nativeChecks++
			}
		}
	}
	checks["background_pixels"], checks["cursor_cores"], checks["native_pixels"] = backgrounds, cores, nativeChecks
	checks["native_final_pixels"] = finalChecks
	p.checks = append(p.checks, checks)
	return verifyPreparationRosterGPU(p, pixels, base)
}

func TestModernHDPreparationTownBackgroundRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PREPARATION_BACKGROUND_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	runModernHDPreparationMatrix(t, false)
}

func TestModernHDPreparationDialogueRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PREPARATION_DIALOGUE_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	runModernHDPreparationMatrix(t, true)
}

func runModernHDPreparationMatrix(t *testing.T, dialogue bool) {
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	assets, err := loadNativePreparationUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativePreparationUI = assets
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}, captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool), verifyScene: verifyPreparationHDScene}
	if dialogue {
		p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
			if err := verifyPreparationHDScene(p, pixels, offscreen); err != nil {
				return err
			}
			return verifyPreparationDialogueGPU(p, pixels)
		}
	}
	var jobs []*nativeClassUIJob
	var sources []*nativeTownFrameSource
	var before [][32]byte
	for variant := 0; variant < 3; variant++ {
		townHDDepartureTestOwner(t, g, variant)
		g.nativeTownDeparture = nil
		g.setupPreparation(g.camp.Node())
		g.nativeClassUIJob = nil
		camp, town, source := g.camp, g.prepTownSource, append([]byte(nil), g.prepPromptSource...)
		sources = append(sources, town)
		for _, phase := range []string{"prompt", "roster", "confirmation", "required"} {
			for style := 0; style < 4; style++ {
				phase, style := phase, style
				if variant == 0 && (phase == "prompt" || phase == "roster" || style == 3) || variant > 0 && phase == "roster" && style == 3 {
					p.captureCases[len(p.cases)] = true
				}
				p.cases = append(p.cases, func() {
					g.camp, g.prepTownSource, g.prepPromptSource = camp, town, source
					g.nativeClassUIJob = nil
					preparationHDTestPhase(g, phase)
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
		for _, phase := range []string{"prompt", "confirmation", "required"} {
			for _, closing := range []bool{false, true} {
				g.camp, g.prepTownSource, g.prepPromptSource = camp, town, source
				preparationHDTestPhase(g, phase)
				var ok bool
				switch {
				case phase == "required" && closing:
					ok = g.beginNativePreparationRequiredClosing(nil)
				case phase == "required":
					ok = g.beginNativePreparationRequiredOpening()
				case phase == "confirmation" && closing:
					ok = g.beginNativePreparationConfirmationClosing(nil)
				case phase == "confirmation":
					ok = g.beginNativePreparationConfirmationOpening()
				case closing:
					ok = g.beginNativePreparationPromptClosing(nil)
				default:
					ok = g.beginNativePreparationPromptOpening()
				}
				if !ok {
					t.Fatal("native job unavailable", phase, closing)
				}
				job := g.nativeClassUIJob
				jobs = append(jobs, job)
				data, _ := json.Marshal([]any{job.frames, job.restore})
				before = append(before, sha256.Sum256(data))
				count := len(job.frames)
				if closing {
					count++
				}
				for frame := 0; frame < count; frame++ {
					frame, phase := frame, phase
					if variant == 0 && phase == "prompt" && (frame == 0 || frame == 6 || closing && frame == count-1) {
						p.captureCases[len(p.cases)] = true
					}
					p.cases = append(p.cases, func() {
						g.camp, g.prepTownSource, g.prepPromptSource = camp, town, source
						preparationHDTestPhase(g, phase)
						g.nativeClassUIJob = job
						job.frame, job.drawn = frame, false
						g.modernStoryPortraits, g.interfaceMode = set, playerInterfaceHD
					})
				}
			}
		}
	}
	if dialogue {
		appendPreparationDialogueExtraCases(t, p, set, &jobs, &before)
	}
	unit := g.partyRoster[g.partyJoinOrder[0]]
	unit.MapSelectorKey = 118
	g.partyRoster[g.partyJoinOrder[0]] = unit
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	wantCases := 207
	if dialogue {
		wantCases = 247
	}
	if len(p.checks) != wantCases || len(p.checks) != len(p.cases) {
		t.Fatal("incomplete preparation matrix", len(p.checks), len(p.cases))
	}
	for i, job := range jobs {
		data, _ := json.Marshal([]any{job.frames, job.restore})
		if sha256.Sum256(data) != before[i] {
			t.Fatal("native frame cache changed", i)
		}
	}
	for _, source := range sources {
		got, ok := source.compose(nil, true, source.font)
		if !ok || !bytes.Equal(got, source.indexed) {
			t.Fatal("frozen town source changed")
		}
	}
	if g.prepPromptSource == nil || g.nativeClassUI.font == nil || g.font.indexedHD != nil {
		t.Fatal("preparation source/font adapter leaked")
	}
	status, scope := "PASS-PREPARATION-TOWN-GPU", "fixture Game.Draw; 48 stable four-style cases and 159 complete transition/restore cases; no new original E2"
	if dialogue {
		status, scope = "PASS-PREPARATION-DIALOGUE-GPU", "fixture Game.Draw; 207 town cases, 16 localized choices/pulses and 24 standalone black cases; HD75/gold/buttons source verified; no new original E2"
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": status, "scope": scope, "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestModernHDPreparationWholeJobMatchesNativeFrames(t *testing.T) {
	for _, kind := range []string{"record", "departure", "confirmation", "required"} {
		for _, closing := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "_opening", true: "_closing"}[closing], func(t *testing.T) {
				g := newPreparationRecordTestGame(t)
				class, err := loadNativeClassUIAssets()
				if err != nil {
					t.Fatal(err)
				}
				g.nativeClassUI = class
				if kind == "departure" {
					g.camp.Node().Cancel = "town"
				}
				identity := 1
				if kind == "required" {
					g.prepRequiredMissing = &identity
				}
				var ok bool
				switch {
				case kind == "required" && closing:
					ok = g.beginNativePreparationRequiredClosing(nil)
				case kind == "required":
					ok = g.beginNativePreparationRequiredOpening()
				case kind == "confirmation" && closing:
					ok = g.beginNativePreparationConfirmationClosing(nil)
				case kind == "confirmation":
					ok = g.beginNativePreparationConfirmationOpening()
				case closing:
					ok = g.beginNativePreparationPromptClosing(nil)
				default:
					ok = g.beginNativePreparationPromptOpening()
				}
				if !ok || g.nativeClassUIJob.preparationText == nil {
					t.Fatal("whole job unavailable")
				}
				job := g.nativeClassUIJob
				d := job.preparationText
				for i, expected := range job.frames {
					got, active := d.composeFrame(i, d.source, d.assets.status.Font)
					if !active || !bytes.Equal(got, expected) {
						t.Fatalf("whole native frame changed %d", i)
					}
				}
				if closing {
					got, ok := d.composeFrame(len(job.frames), d.source, d.assets.status.Font)
					if !ok || !bytes.Equal(got, job.restore) {
						t.Fatal("restore source changed")
					}
				}
				for _, index := range []int{-1, len(job.frames) + 1} {
					if _, ok := d.composeFrame(index, d.source, d.assets.status.Font); ok {
						t.Fatal("invalid frame admitted", index)
					}
				}
				if _, ok := d.composeFrame(0, d.source[:63999], d.assets.status.Font); ok {
					t.Fatal("short source admitted")
				}
			})
		}
	}
}

func TestModernHDPreparationTownTextWriterMaskKeepsLayout(t *testing.T) {
	run := modernHDTextRun{x: 20, y: 40, pixels: 28, clip: image.Rect(20, 40, 54, 74), inkTop: true}
	mask := make([]byte, 64000)
	for y := 20; y < 37; y++ {
		for x := 10; x < 27; x++ {
			if (x+3*y)%5 == 0 || (x > 18 && y > 29) {
				mask[y*320+x] = 255
			}
		}
	}
	parts := modernTownTextOutsideWrites([]modernHDTextRun{run}, mask)
	if len(parts) == 0 {
		t.Fatal("all partial ink disappeared")
	}
	for repeat := 0; repeat < 20; repeat++ {
		if !reflect.DeepEqual(parts, modernTownTextOutsideWrites([]modernHDTextRun{run}, mask)) {
			t.Fatal("nondeterministic clip order")
		}
	}
	for _, part := range parts {
		if part.x != run.x || part.y != run.y || part.pixels != run.pixels || part.layoutSize != run.clip.Size() || !part.clip.In(run.clip) {
			t.Fatal("clipping changed original glyph layout", part)
		}
	}
	for y := run.clip.Min.Y; y < run.clip.Max.Y; y++ {
		for x := run.clip.Min.X; x < run.clip.Max.X; x++ {
			count := 0
			for _, part := range parts {
				if image.Pt(x, y).In(part.clip) {
					count++
				}
			}
			want := 1
			if mask[y/2*320+x/2] != 0 {
				want = 0
			}
			if count != want {
				t.Fatalf("writer coverage (%d,%d) = %d want%d", x, y, count, want)
			}
		}
	}
}

func TestModernHDPreparationTransitionInvalidRecipeKeepsNativeJob(t *testing.T) {
	g := newPreparationRecordTestGame(t)
	class, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeClassUI = class
	g.font, g.modernHDText = loadFont(), &modernHDTextFrame{}
	if !g.beginNativePreparationPromptOpening() {
		t.Fatal("native opening unavailable")
	}
	job := g.nativeClassUIJob
	job.frame = 6
	job.preparationText.kind = 255
	screen := ebiten.NewImage(logicalW, logicalH)
	if g.drawModernHDPreparationTransition(screen, job) || !g.drawNativeClassUIJob(screen) || !job.drawn || job.frame != 6 || len(g.modernHDText.runs) != 0 {
		t.Fatal("invalid recipe did not retain native acknowledgement path")
	}
}
