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
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type player9StageProbe struct {
	t                               *testing.T
	checker                         unitPresentGPUProbe
	g, control                      *Game
	controlScreen                   *ebiten.Image
	index, draws, done, controlDone int
	seen, switched                  map[string]bool
	out                             string
	rows, completed                 []map[string]any
}

func (p *player9StageProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *player9StageProbe) Update() error {
	if p.checker.err != nil || p.index == 2 {
		return ebiten.Termination
	}
	if p.g != nil {
		return nil
	}
	runtime.GC()
	g, a, b := player9StageGame(p.t, true, p.index == 0)
	control, ca, cb := player9StageGame(p.t, false, p.index == 0)
	p.g, p.control = g, control
	p.done, p.controlDone, p.draws = 0, 0, 0
	p.seen, p.switched = map[string]bool{}, map[string]bool{}
	if err := g.startNativeCommand9PlayerPresentation(a, b, func([]battle.NativeCommandDamageResult) { p.done++ }); err != nil {
		p.checker.err = err
		return nil
	}
	if err := control.startNativeCommand9PlayerPresentation(ca, cb, func([]battle.NativeCommandDamageResult) { p.controlDone++ }); err != nil {
		p.checker.err = err
	}
	return nil
}

func (p *player9StageProbe) Draw(screen *ebiten.Image) {
	if p.checker.err != nil || p.g == nil || p.index == 2 {
		return
	}
	g, control := p.g, p.control
	j, cj := g.nativeCmd9Player, control.nativeCmd9Player
	if j == nil || cj == nil || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) || j.phase != cj.phase || j.frame != cj.frame || j.hold != cj.hold {
		p.checker.err = fmt.Errorf("player9 native control diverged before Draw")
		return
	}
	g.frame++
	control.frame++
	g.Draw(screen)
	control.Draw(p.controlScreen)
	f := g.modernHDMapPresented
	if g.loadErr != "" || !g.modernHDWorldVisible || !j.drawn || !cj.drawn || f == nil || !f.TransientTerrain || len(f.Terrain) == 0 || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
		p.checker.err = fmt.Errorf("player9 stage Game.Draw failed: %s", g.loadErr)
		return
	}
	key := fmt.Sprintf("%d/%d", j.phase, j.frame)
	if !p.seen[key] {
		pixels, ok := p.checker.checkWorld(g, f, healHDCPUWorld(f, p.checker.set), screen, key)
		if !ok {
			return
		}
		digest := sha256.Sum256(pixels)
		point := j.phase == nativeCommand9PlayerPalette && (j.frame == 0 || j.frame == 7) ||
			j.phase == nativeCommand9PlayerEffect && (j.frame == 0 || j.frame == 13 || j.frame == 26) ||
			j.phase == nativeCommand9PlayerResult && (j.frame == 0 || j.frame == 21) || j.phase == nativeCommand9PlayerHold
		if point && !p.switched[key] {
			before, phase, frame, hold := stage42NativeState(g), j.phase, j.frame, j.hold
			if err := g.applyTheme(nil); err != nil {
				p.checker.err = err
				return
			}
			g.Draw(screen)
			if g.modernHDWorldVisible {
				p.checker.err = fmt.Errorf("pixel mode retained HD")
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
				p.checker.err = fmt.Errorf("F2 changed native state or frozen stage")
				return
			}
			p.switched[key] = true
			if err := (&rowShakeGPUProbe{out: p.out}).save(fmt.Sprintf("hit-%v-phase-%d-frame-%d.png", p.index == 0, phase, frame), again); err != nil {
				p.checker.err = err
				return
			}
		}
		p.rows = append(p.rows, map[string]any{"hit": p.index == 0, "phase": j.phase, "frame": j.frame, "final_sha256": fmt.Sprintf("%x", digest), "native_vga_sha256": skyKeySHA(g.nativeMapVGA), "rng": g.nativeRNGState})
		p.seen[key] = true
	}
	g.stepNativeCommand9PlayerPresentation()
	control.stepNativeCommand9PlayerPresentation()
	p.draws++
	if p.draws > 500 {
		p.checker.err = fmt.Errorf("player9 bounded playback exceeded")
		return
	}
	if g.nativeCmd9Player != nil {
		return
	}
	if control.nativeCmd9Player != nil || p.done != 1 || p.controlDone != 1 || len(p.switched) != 8 || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
		p.checker.err = fmt.Errorf("player9 final native transaction differs")
		return
	}
	g.Draw(screen)
	control.Draw(p.controlScreen)
	f = g.modernHDMapPresented
	if g.loadErr != "" || !g.modernHDWorldVisible || f == nil || !f.TransientTerrain || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(control)) {
		p.checker.err = fmt.Errorf("player9 return lost HD stage: %s", g.loadErr)
		return
	}
	pixels, ok := p.checker.checkWorld(g, f, healHDCPUWorld(f, p.checker.set), screen, "return")
	if !ok {
		return
	}
	if err := (&rowShakeGPUProbe{out: p.out}).save(fmt.Sprintf("hit-%v-return.png", p.index == 0), pixels); err != nil {
		p.checker.err = err
		return
	}
	p.completed = append(p.completed, map[string]any{"hit": p.index == 0, "frames": len(p.seen), "draws": p.draws, "f2_round_trips": len(p.switched), "continuations": p.done, "native_control_unchanged": true, "return_sha256": fmt.Sprintf("%x", sha256.Sum256(pixels))})
	p.g, p.control = nil, nil
	if p.checker.presentation != nil {
		p.checker.presentation.Game = nil
	}
	p.index++
}

func TestModernHDPlayer9StageGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PLAYER9_STAGE_GPU") != "1" {
		t.Skip("dedicated stage Game.Draw process")
	}
	out := os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh capture path required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &player9StageProbe{t: t, out: out, controlScreen: ebiten.NewImage(640, 400)}
	p.checker.set = itemHDSet(t)
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.checker.err != nil {
		t.Fatal(p.checker.err)
	}
	if p.index != 2 || len(p.completed) != 2 {
		t.Fatal("incomplete player9 stage matrix")
	}
	d := map[string]any{"status": "PASS-PLAYER9-STAGE-HD-GAME", "map_index": 23, "seed": 7, "clock_unix": 1000, "rgba_tolerance": 2, "cases": p.completed, "rows": p.rows, "scope": "controlled hit/MISS on actual map23; RUNTIME-E1, no original oracle or normal-player claim"}
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
