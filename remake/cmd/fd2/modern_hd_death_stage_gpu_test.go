package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type deathStageGPUProbe struct {
	t                                                  *testing.T
	set                                                *modernStoryPortraitSet
	g, control                                         *Game
	job                                                *nativeCh28PostPresentJob
	live                                               *battle.State
	liveUnits                                          []*battle.Unit
	actor                                              *battle.Unit
	index, calls, controlCalls, draws, prefixDraws, f2 int
	seen                                               [26]bool
	checker                                            unitPresentGPUProbe
	controlScreen                                      *ebiten.Image
	err                                                error
	out                                                string
	rows, returns                                      []map[string]any
}

func (p *deathStageGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *deathStageGPUProbe) Update() error {
	if p.err != nil || p.index == 3 {
		return ebiten.Termination
	}
	return nil
}
func (p *deathStageGPUProbe) kind() string { return []string{"script", "action", "status"}[p.index] }

func deathStageFullGame(t *testing.T, set *modernStoryPortraitSet) (*Game, *battle.Unit) {
	t.Helper()
	g, a := rewardStageGame(t, set)
	a.Acted = false
	a.NativeRecordByte5 &^= 0x80
	// Keep an unacted ally and a living enemy off camera so the action's real
	// continuation neither ends the turn nor wins this controlled battle.
	for _, spec := range [][3]int{{1, 2, 2}, {5, 3, 2}} {
		u := g.st.Units[spec[0]]
		u.OnField = true
		u.SetMapPlacement(spec[1], spec[2], 0)
		u.HP, u.Acted = 100, false
		u.NativeRecordByte5 &^= 0x81
		if err := u.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
	}
	if g.st.Units[1].Camp != battle.Own || g.st.Units[5].Camp != battle.Enemy {
		t.Fatal("off-camera continuation guards have wrong native camps")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g, a
}

func (p *deathStageGPUProbe) begin(screen *ebiten.Image) error {
	p.job, p.actor, p.live, p.liveUnits = nil, nil, nil, nil
	p.g, p.control = nil, nil
	if p.checker.presentation != nil {
		p.checker.presentation.Game = nil
	}
	runtime.GC()
	g, a := deathStageFullGame(p.t, p.set)
	control, _ := deathStageFullGame(p.t, nil)
	p.g, p.control, p.actor = g, control, a
	p.calls, p.controlCalls = 0, 0
	p.seen = [26]bool{}
	p.live, p.liveUnits = g.st, append([]*battle.Unit(nil), g.st.Units...)
	for i, game := range []*Game{g, control} {
		current, isControl := game, i == 1
		after := func() {
			if isControl {
				p.controlCalls++
			} else {
				p.calls++
			}
		}
		switch p.kind() {
		case "script":
			if err := current.startNativeHPClear(4, after); err != nil {
				return err
			}
		case "action":
			actor, dead := current.st.Units[0], current.st.Units[4]
			flags := captureNativeActionDeathFlags(actor, dead)
			dead.ApplyHPDamage(dead.HP)
			current.queueNativeActionDeathPresentation(flags)
			current.finishSuccessfulUnitAction(actor, after)
		case "status":
			dead := current.st.Units[0]
			dead.HP, dead.MaxHP = 1, 100
			dead.NativeTransient = [6]byte{0, 0, 0, 2}
			if err := current.beginNativeTransientPhases([]byte{2}, after); err != nil {
				return err
			}
		}
		if current.loadErr != "" {
			return fmt.Errorf("%s start: %s", p.kind(), current.loadErr)
		}
	}
	if p.kind() == "status" {
		for n := 0; g.nativeCh28PostPresent == nil && n < 1200; n++ {
			g.Draw(screen)
			control.Draw(p.controlScreen)
			p.prefixDraws++
			for _, current := range []*Game{g, control} {
				current.stepNativeStoryDialogueProgress()
				current.stepNativeLevelUpDialogue(true)
				current.stepNativeClassUILifecycle(time.Unix(1000, 0))
				if current.loadErr != "" {
					return fmt.Errorf("status message: %s", current.loadErr)
				}
			}
		}
	}
	p.job = g.nativeCh28PostPresent
	if p.job == nil || control.nativeCh28PostPresent == nil || p.job.kind != p.kind() || p.job.hdStage == nil || p.calls != 0 || p.controlCalls != 0 {
		return fmt.Errorf("%s caller did not enter death owner", p.kind())
	}
	return nil
}

func (p *deathStageGPUProbe) save(name string, pixels []byte) error {
	f, err := os.Create(filepath.Join(p.out, name))
	if err != nil {
		return err
	}
	err = png.Encode(f, &image.RGBA{Pix: pixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
	closed := f.Close()
	if err != nil {
		return err
	}
	return closed
}

func (p *deathStageGPUProbe) roundTrip(screen *ebiten.Image, frame *modernHDMapFrame, want *image.RGBA, digest [32]byte, label string) error {
	g := p.g
	before := deathStageNativeState(g)
	if err := g.applyTheme(nil); err != nil {
		return err
	}
	g.Draw(screen)
	if g.modernHDWorldVisible {
		return fmt.Errorf("%s pixel theme retained HD", label)
	}
	if err := g.applyTheme(p.set); err != nil {
		return err
	}
	g.Draw(screen)
	pixels, ok := p.checker.checkWorld(g, frame, want, screen, label+" F2")
	if !ok {
		return p.checker.err
	}
	if before != deathStageNativeState(g) || sha256.Sum256(pixels) != digest {
		return fmt.Errorf("%s F2 changed state or pixels", label)
	}
	p.f2++
	return nil
}

func (p *deathStageGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == 3 {
		return
	}
	if p.g == nil {
		if p.err = p.begin(screen); p.err != nil {
			return
		}
	}
	g, control := p.g, p.control
	g.frame++
	control.frame++
	p.draws++
	g.Draw(screen)
	control.Draw(p.controlScreen)
	if g.loadErr != "" || control.loadErr != "" || !g.modernHDWorldVisible || deathStageNativeState(g) != deathStageNativeState(control) {
		p.err = fmt.Errorf("%s native control mismatch: %s %s", p.kind(), g.loadErr, control.loadErr)
		return
	}
	if p.kind() != "script" && (g.st != p.live || !sameNativeDeathUnitPointers(g.st.Units, p.liveUnits)) {
		p.err = fmt.Errorf("retained death replaced live identities")
		return
	}
	if g.nativeCh28PostPresent == nil {
		frame := g.modernHDMapPresented
		want := healHDCPUWorld(frame, p.set)
		pixels, ok := p.checker.checkWorld(g, frame, want, screen, p.kind()+" return")
		if !ok {
			p.err = p.checker.err
			return
		}
		if p.calls != 1 || p.controlCalls != 1 || g.nativeRNGState != 7 || g.gold != 123 {
			p.err = fmt.Errorf("%s repeated transaction/continuation", p.kind())
			return
		}
		if p.kind() == "action" && !p.actor.Acted {
			p.err = fmt.Errorf("action did not resume normal completion")
			return
		}
		digest := sha256.Sum256(pixels)
		if p.err = p.roundTrip(screen, frame, want, digest, p.kind()+" return"); p.err != nil {
			return
		}
		name := p.kind() + "-return.png"
		if p.err = p.save(name, pixels); p.err != nil {
			return
		}
		p.returns = append(p.returns, map[string]any{"kind": p.kind(), "continuations": p.calls, "f2_roundtrip": true, "native_control_unchanged": true, "final_sha256": fmt.Sprintf("%x", digest), "capture": name})
		p.index++
		p.g = nil
		return
	}
	j := g.nativeCh28PostPresent
	i := j.frame
	if j != p.job || !j.drawn || j.hdFrames[i] != g.modernHDMapPresented {
		p.err = fmt.Errorf("death present not acknowledged")
		return
	}
	if !p.seen[i] {
		frame := j.hdFrames[i]
		if i < 25 && frame.HUDUnit != nil {
			p.err = fmt.Errorf("event redrew steady HUD unit")
			return
		}
		if i == 25 && frame.HUDUnit == nil && p.kind() != "status" {
			p.err = fmt.Errorf("steady HUD unit absent")
			return
		}
		if i >= 13 && i < 19 && len(frame.Units) != len(j.hdFrames[12].Units) {
			p.err = fmt.Errorf("A removed last visible pose")
			return
		}
		if i >= 19 && len(frame.Units) != len(j.hdFrames[12].Units)-len(j.hdTargets) {
			p.err = fmt.Errorf("B retained dead unit")
			return
		}
		want := healHDCPUWorld(frame, p.set)
		pixels, ok := p.checker.checkWorld(g, frame, want, screen, fmt.Sprintf("%s frame%d", p.kind(), i))
		if !ok {
			p.err = p.checker.err
			return
		}
		visible := 0
		clip := frame.Viewport
		if clip.Empty() {
			clip = modernHDMapViewport()
		}
		for _, layer := range frame.Terrain {
			if layer.Stage {
				for y := 0; y < layer.Image.Bounds().Dy(); y += 6 {
					for x := 0; x < layer.Image.Bounds().Dx(); x += 6 {
						if layer.Image.RGBAAt(x, y).A != 0 && image.Pt(layer.Rect.Min.X+x, layer.Rect.Min.Y+y).In(clip) {
							visible++
						}
					}
				}
			}
		}
		if visible == 0 {
			p.err = fmt.Errorf("%s frame%d has no visible stage", p.kind(), i)
			return
		}
		digest := sha256.Sum256(pixels)
		f2 := i == 0 || i == 12 || i == 13 || i == 18 || i == 19 || i == 24 || i == 25
		if f2 {
			if p.err = p.roundTrip(screen, frame, want, digest, fmt.Sprintf("%s frame%d", p.kind(), i)); p.err != nil {
				return
			}
		}
		row := map[string]any{"kind": p.kind(), "frame": i, "native_control_unchanged": true, "f2_roundtrip": f2, "visible_stage_pixels": visible, "final_sha256": fmt.Sprintf("%x", digest), "native_vga_sha256": fmt.Sprintf("%x", sha256.Sum256(j.vgaFrames[i])), "source_sha256": fmt.Sprintf("%x", j.hdStage.signature), "units": len(frame.Units), "overlays": len(frame.Overlays)}
		if i == 0 || i == 12 || i == 15 || i == 20 || i == 25 {
			name := fmt.Sprintf("%s-frame%02d.png", p.kind(), i)
			if p.err = p.save(name, pixels); p.err != nil {
				return
			}
			row["capture"] = name
		}
		p.rows = append(p.rows, row)
		p.seen[i] = true
	}
	for _, current := range []*Game{g, control} {
		job := current.nativeCh28PostPresent
		beforeFrame, beforeWait := job.frame, job.wait
		job.drawn = false
		current.stepNativeCh28PostPresent()
		if job.frame != beforeFrame || job.wait != beforeWait {
			p.err = fmt.Errorf("undrawn death advanced")
			return
		}
		job.drawn = true
		current.stepNativeCh28PostPresent()
	}
}

func TestModernHDDeathStageGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_DEATH_STAGE_GPU") != "1" {
		t.Skip("dedicated death stage GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	set := itemHDSet(t)
	out := os.Getenv("FD2_TEST_DEATH_STAGE_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &deathStageGPUProbe{t: t, set: set, out: out, controlScreen: ebiten.NewImage(640, 400)}
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 3 || len(p.rows) != 78 || len(p.returns) != 3 || p.f2 != 24 {
		t.Fatal("incomplete death stage", p.index, p.err, len(p.rows), p.f2)
	}
	d := map[string]any{"status": "PASS-DEATH-STAGE-HD-RUNTIME-E1", "cases": 3, "map_index": 23, "native_seed": 7, "fixed_clock_unix": 1000, "normal_player_verified": false, "rgba_tolerance": 2, "game_draws": p.draws, "status_prefix_draws": p.prefixDraws, "f2_round_trips": p.f2, "rows": p.rows, "returns": p.returns}
	raw, _ := json.MarshalIndent(d, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
