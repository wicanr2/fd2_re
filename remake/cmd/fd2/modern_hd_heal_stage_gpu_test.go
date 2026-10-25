package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type healStageProbe struct {
	t                                                    *testing.T
	checker                                              unitPresentGPUProbe
	g, control                                           *Game
	controlScreen                                        *ebiten.Image
	nativePixels                                         []byte
	ids                                                  []int
	index, draws, calls, done, controlCalls, controlDone int
	seen, switched                                       map[string]bool
	captured                                             map[nativeCommandHealPresentationPhase]bool
	out                                                  string
	rows, completed                                      []map[string]any
}

func (p *healStageProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *healStageProbe) Update() error {
	if p.checker.err != nil || p.index == len(p.ids) {
		return ebiten.Termination
	}
	if p.g != nil {
		return nil
	}
	runtime.GC()
	id := p.ids[p.index]
	g, a, b, targets := healStageScenario(p.t, 13+id/2, true, id%2 == 1)
	control, ca, cb, ct := healStageScenario(p.t, 13+id/2, false, id%2 == 1)
	p.g, p.control = g, control
	p.calls, p.done, p.controlCalls, p.controlDone, p.draws = 0, 0, 0, 0, 0
	p.seen, p.switched = map[string]bool{}, map[string]bool{}
	p.captured = map[nativeCommandHealPresentationPhase]bool{}
	startHealHDTest(p.t, g, a, b, targets, 13+id/2, &p.calls, &p.done)
	startHealHDTest(p.t, control, ca, cb, ct, 13+id/2, &p.controlCalls, &p.controlDone)
	p.t.Logf("prepared heal stage case %d", id)
	return nil
}

func healStageF2Point(j *nativeCommandHealPresentationJob) bool {
	switch j.phase {
	case nativeCommandHealFrames:
		return j.frame == 0 || j.frame == 8 || j.frame == 15
	case nativeCommandHealMidHold, nativeCommandHealFrontHold, nativeCommandHealDigitHold:
		return true
	case nativeCommandHealEffectFrames:
		return j.frame == 0
	case nativeCommandHealMaskFrames:
		return j.frame == 0 || j.frame == 2*j.tailSchedule.MaskPairs
	case nativeCommandHealDigitFrames:
		return j.frame == 0 || j.frame == 21
	}
	return false
}

func (p *healStageProbe) Draw(screen *ebiten.Image) {
	if p.checker.err != nil || p.g == nil || p.index == len(p.ids) {
		return
	}
	g, control := p.g, p.control
	j, cj := g.nativeHealPresentation, control.nativeHealPresentation
	if j == nil || cj == nil || j.phase != cj.phase || j.frame != cj.frame || j.hold != cj.hold || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
		p.checker.err = fmt.Errorf("heal native control diverged before Draw")
		return
	}
	g.frame++
	control.frame++
	g.Draw(screen)
	control.Draw(p.controlScreen)
	f := g.modernHDMapPresented
	needsAck := j.phase == nativeCommandHealFrames || j.phase == nativeCommandHealEffectFrames || j.phase == nativeCommandHealMaskFrames || j.phase == nativeCommandHealDigitFrames
	if g.loadErr != "" || !g.modernHDWorldVisible || needsAck && (!j.drawn || !cj.drawn) || f == nil || !f.TransientTerrain || len(f.Terrain) == 0 || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
		p.checker.err = fmt.Errorf("heal stage Game.Draw failed phase%d frame%d hdAck%t nativeAck%t: %s", j.phase, j.frame, j.drawn, cj.drawn, g.loadErr)
		return
	}
	key := fmt.Sprintf("%d/%d", j.phase, j.frame)
	if !p.seen[key] {
		if j.phase == nativeCommandHealMidHold {
			if f != j.hdFrames.Front[j.frontSchedule.MidFrame-1] {
				p.checker.err = fmt.Errorf("mid hold lost the ninth HD effect")
				return
			}
			if len(p.nativePixels) != 640*400*4 {
				p.nativePixels = make([]byte, 640*400*4)
			}
			p.controlScreen.ReadPixels(p.nativePixels)
			for y := 0; y < 200; y++ {
				for x := 0; x < 320; x++ {
					r, g, b, a := cj.palette[cj.frontFrames[cj.frontSchedule.MidFrame-1][y*320+x]].RGBA()
					at := (2*y*640 + 2*x) * 4
					if p.nativePixels[at] != byte(r>>8) || p.nativePixels[at+1] != byte(g>>8) || p.nativePixels[at+2] != byte(b>>8) || p.nativePixels[at+3] != byte(a>>8) {
						p.checker.err = fmt.Errorf("native mid hold differs at %d,%d", x, y)
						return
					}
				}
			}
		}
		pixels, ok := p.checker.checkWorld(g, f, healHDCPUWorld(f, p.checker.set), screen, key)
		if !ok {
			return
		}
		digest := sha256.Sum256(pixels)
		if healStageF2Point(j) && !p.switched[key] {
			before, phase, frame, hold := stage42NativeState(g), j.phase, j.frame, j.hold
			if err := g.applyTheme(nil); err != nil {
				p.checker.err = err
				return
			}
			g.Draw(screen)
			if g.modernHDWorldVisible {
				p.checker.err = fmt.Errorf("pixel heal retained HD")
				return
			}
			if err := g.applyTheme(p.checker.set); err != nil {
				p.checker.err = err
				return
			}
			g.Draw(screen)
			again, ok := p.checker.checkWorld(g, g.modernHDMapPresented, healHDCPUWorld(g.modernHDMapPresented, p.checker.set), screen, "F2/"+key)
			if !ok {
				return
			}
			if sha256.Sum256(again) != digest || !reflect.DeepEqual(before, stage42NativeState(g)) || phase != j.phase || frame != j.frame || hold != j.hold {
				p.checker.err = fmt.Errorf("heal F2 changed source or transaction")
				return
			}
			p.switched[key] = true
		}
		capture := !p.captured[j.phase] || j.phase == nativeCommandHealFrames && j.frame == 15
		if capture {
			name := fmt.Sprintf("case-%d-phase-%d-frame-%d.png", p.ids[p.index], j.phase, j.frame)
			if err := (&rowShakeGPUProbe{out: p.out}).save(name, pixels); err != nil {
				p.checker.err = err
				return
			}
			p.captured[j.phase] = true
		}
		p.rows = append(p.rows, map[string]any{"case": p.ids[p.index], "phase": j.phase, "frame": j.frame, "final_sha256": fmt.Sprintf("%x", digest), "native_vga_sha256": skyKeySHA(g.nativeMapVGA), "rng": g.nativeRNGState, "transactions": p.calls, "mid_hold_native_verified": j.phase == nativeCommandHealMidHold})
		p.seen[key] = true
	}
	g.stepNativeCommandHealPresentation()
	control.stepNativeCommandHealPresentation()
	p.draws++
	if p.draws > 500 {
		p.checker.err = fmt.Errorf("heal bounded playback exceeded")
		return
	}
	if g.nativeHealPresentation != nil {
		return
	}
	if control.nativeHealPresentation != nil || p.calls != 1 || p.done != 1 || p.controlCalls != 1 || p.controlDone != 1 || len(p.switched) != 11 || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
		p.checker.err = fmt.Errorf("heal final transaction/control differs")
		return
	}
	g.Draw(screen)
	control.Draw(p.controlScreen)
	f = g.modernHDMapPresented
	if g.loadErr != "" || !g.modernHDWorldVisible || f == nil || !f.TransientTerrain || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
		p.checker.err = fmt.Errorf("heal return lost stage: %s", g.loadErr)
		return
	}
	pixels, ok := p.checker.checkWorld(g, f, healHDCPUWorld(f, p.checker.set), screen, "return")
	if !ok {
		return
	}
	if err := (&rowShakeGPUProbe{out: p.out}).save(fmt.Sprintf("case-%d-return.png", p.ids[p.index]), pixels); err != nil {
		p.checker.err = err
		return
	}
	p.completed = append(p.completed, map[string]any{"case": p.ids[p.index], "command": 13 + p.ids[p.index]/2, "multi": p.ids[p.index]%2 == 1, "frames": len(p.seen), "draws": p.draws, "f2_round_trips": len(p.switched), "transactions": p.calls, "continuations": p.done, "native_control_unchanged": true, "return_sha256": fmt.Sprintf("%x", sha256.Sum256(pixels))})
	p.t.Logf("completed heal stage case %d", p.ids[p.index])
	p.g, p.control = nil, nil
	if p.checker.presentation != nil {
		p.checker.presentation.Game = nil
	}
	p.index++
}

func TestModernHDHealStageGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_HEAL_STAGE_GPU") != "1" {
		t.Skip("dedicated heal stage Game.Draw process")
	}
	out := os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh capture path required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &healStageProbe{t: t, out: out, controlScreen: ebiten.NewImage(640, 400), ids: []int{0, 1, 2, 3, 4, 5, 6, 7}}
	if os.Getenv("FD2_HD_HEAL_STAGE_CASES") == "0,7" {
		p.ids = []int{0, 7}
	}
	p.checker.set = itemHDSet(t)
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.checker.err != nil {
		t.Fatal(p.checker.err)
	}
	expectedFrames := 16 + 7 + 22 + (2*fdother.NativeCommandHealTailMaskPairs + 1) + 3
	if p.index != len(p.ids) || len(p.rows) != expectedFrames*len(p.ids) {
		t.Fatal("incomplete heal stage coverage", p.index, len(p.rows))
	}
	d := map[string]any{"status": "PASS-HEAL-STAGE-HD-GAME", "map_index": 23, "seed": 7, "clock_unix": 1000, "rgba_tolerance": 2, "case_ids": p.ids, "cases": p.completed, "rows": p.rows, "scope": "controlled actual map23; native snapshot contract and remake pixel-theme control; RUNTIME-E1, not original oracle or normal-player proof"}
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
