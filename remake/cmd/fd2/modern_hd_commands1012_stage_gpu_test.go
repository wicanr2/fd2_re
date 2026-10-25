package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type commands1012StageProbe struct {
	t                               *testing.T
	checker                         unitPresentGPUProbe
	g, control                      *Game
	controlScreen                   *ebiten.Image
	ids                             []int
	index, done, controlDone, draws int
	f2, captured                    map[string]bool
	cpu                             map[string]*image.RGBA
	rows, cases                     []map[string]any
	out                             string
}

func (p *commands1012StageProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *commands1012StageProbe) Update() error {
	if p.checker.err != nil || p.index == len(p.ids) {
		return ebiten.Termination
	}
	if p.g != nil {
		return nil
	}
	runtime.GC()
	id := p.ids[p.index]
	g, a, b := commands1012StageGame(p.t, 10+id/3, id%3, true)
	c, ca, cb := commands1012StageGame(p.t, 10+id/3, id%3, false)
	p.g, p.control = g, c
	p.done, p.controlDone, p.draws = 0, 0, 0
	p.f2, p.captured = map[string]bool{}, map[string]bool{}
	p.cpu = map[string]*image.RGBA{}
	if err := g.startNativeCommand1012Presentation(a, b, 10+id/3, func([]battle.NativeCommandDamageResult) { p.done++ }); err != nil {
		p.t.Fatal(err)
	}
	if err := c.startNativeCommand1012Presentation(ca, cb, 10+id/3, func([]battle.NativeCommandDamageResult) { p.controlDone++ }); err != nil {
		p.t.Fatal(err)
	}
	want := 1
	if id%3 == 2 {
		want = 2
	}
	if len(g.nativeCmd1012.plan.Results) != want || g.nativeCmd1012.plan.Results[0].Hit != (id%3 != 1) {
		p.t.Fatal("stage variant did not exercise hit/MISS/dual", id, len(g.nativeCmd1012.plan.Results))
	}
	p.t.Logf("prepared commands1012 stage case%d", id)
	return nil
}

func commands1012StageKey(g *Game) (string, int) {
	j := g.nativeCmd1012
	if g.native2189A != nil {
		return "prelude", g.native2189A.index
	}
	switch j.phase {
	case nativeCommand1012Main:
		return "main", j.frame
	case nativeCommand1012Result:
		return "result", j.frame
	default:
		return "hold", j.hold
	}
}

func (p *commands1012StageProbe) check(screen *ebiten.Image, key string) ([]byte, bool) {
	f := p.g.modernHDMapPresented
	want := p.cpu[key]
	if want == nil {
		want = healHDCPUWorld(f, p.checker.set)
		p.cpu[key] = want
	}
	return p.checker.checkWorld(p.g, f, want, screen, key)
}

func (p *commands1012StageProbe) Draw(screen *ebiten.Image) {
	if p.checker.err != nil || p.g == nil || p.index == len(p.ids) {
		return
	}
	g, c := p.g, p.control
	j, cj := g.nativeCmd1012, c.nativeCmd1012
	if j == nil || cj == nil || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(c)) {
		p.checker.err = fmt.Errorf("commands1012 stage native control diverged")
		return
	}
	phase, n := commands1012StageKey(g)
	cp, cn := commands1012StageKey(c)
	if cp != phase || cn != n {
		p.checker.err = fmt.Errorf("native phase diverged")
		return
	}
	g.frame++
	c.frame++
	g.Draw(screen)
	c.Draw(p.controlScreen)
	ack := j.drawn && cj.drawn
	if phase == "prelude" {
		ack = g.native2189A.drawn && c.native2189A.drawn
	}
	if g.loadErr != "" || !g.modernHDWorldVisible || g.modernHDMapPresented == nil || !ack || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(c)) {
		p.checker.err = fmt.Errorf("commands1012 stage Draw %s/%d: %s", phase, n, g.loadErr)
		return
	}
	key := fmt.Sprintf("%s/%d", phase, n)
	cache := key
	if phase == "main" {
		cache = fmt.Sprintf("main/%d", n%4)
	}
	if phase == "hold" {
		cache = "hold"
	}
	pixels, ok := p.check(screen, cache)
	if !ok {
		return
	}
	digest := sha256.Sum256(pixels)
	f2 := phase == "prelude" && (n == 0 || n == 9 || n == 10) || phase == "main" && (n < 4 || n == 59) || phase == "result" && (n == 0 || n == 21) || phase == "hold" && n == 30
	if f2 && !p.f2[key] {
		before := stage42NativeState(g)
		if err := g.applyTheme(nil); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.checker.err = fmt.Errorf("pixel theme retained stage HD")
			return
		}
		if err := g.applyTheme(p.checker.set); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		again, ok := p.check(screen, cache)
		if !ok {
			return
		}
		pp, nn := commands1012StageKey(g)
		if sha256.Sum256(again) != digest || pp != phase || nn != n || !reflect.DeepEqual(before, stage42NativeState(g)) {
			p.checker.err = fmt.Errorf("F2 changed stage source or transaction")
			return
		}
		p.f2[key] = true
	}
	capture := phase == "prelude" && (n == 0 || n == 9 || n == 10) || phase == "main" && (n == 0 || n == 1) || phase == "result" && n == 0 || phase == "hold" && n == 30
	if capture && !p.captured[key] {
		name := fmt.Sprintf("case-%d-%s-%d.png", p.ids[p.index], phase, n)
		if err := (&rowShakeGPUProbe{out: p.out}).save(name, pixels); err != nil {
			p.checker.err = err
			return
		}
		p.captured[key] = true
	}
	p.rows = append(p.rows, map[string]any{"case": p.ids[p.index], "phase": phase, "frame": n, "final_sha256": fmt.Sprintf("%x", digest), "native_vga_sha256": skyKeySHA(g.nativeMapVGA), "rng": g.nativeRNGState, "f2_round_trip": f2})
	if phase == "prelude" {
		g.stepNative2189A()
		c.stepNative2189A()
	} else {
		g.stepNativeCommand1012Presentation()
		c.stepNativeCommand1012Presentation()
	}
	p.draws++
	if p.draws > 150 {
		p.checker.err = fmt.Errorf("commands1012 stage playback exceeded bound")
		return
	}
	if g.nativeCmd1012 != nil {
		return
	}
	wantDraws, wantF2 := 113, 8
	if j.schedule.CommandID != 10 {
		wantDraws, wantF2 = 124, 11
	}
	if c.nativeCmd1012 != nil || g.native2189A != nil || c.native2189A != nil || p.done != 1 || p.controlDone != 1 || p.draws != wantDraws || len(p.f2) != wantF2 || !reflect.DeepEqual(stage42NativeState(g), stage42NativeState(c)) {
		p.checker.err = fmt.Errorf("commands1012 stage final state differs")
		return
	}
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || g.modernHDMapPresented == nil || !g.modernHDMapPresented.TransientTerrain {
		p.checker.err = fmt.Errorf("commands1012 return lost stage: %s", g.loadErr)
		return
	}
	pixels, ok = p.check(screen, "return")
	if !ok {
		return
	}
	if err := (&rowShakeGPUProbe{out: p.out}).save(fmt.Sprintf("case-%d-return.png", p.ids[p.index]), pixels); err != nil {
		p.checker.err = err
		return
	}
	p.cases = append(p.cases, map[string]any{"case": p.ids[p.index], "command": j.schedule.CommandID, "variant": p.ids[p.index] % 3, "target_count": len(j.plan.Results), "draws": p.draws, "f2_round_trips": len(p.f2), "continuations": p.done, "native_control_unchanged": true, "return_sha256": fmt.Sprintf("%x", sha256.Sum256(pixels))})
	p.t.Logf("completed commands1012 stage case%d", p.ids[p.index])
	p.g, p.control = nil, nil
	p.cpu = nil
	if p.checker.presentation != nil {
		p.checker.presentation.Game = nil
	}
	p.index++
}

func TestModernHDCommand1012StageGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_COMMAND1012_STAGE_GPU") != "1" {
		t.Skip("dedicated stage Game.Draw process")
	}
	out := os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh capture path required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &commands1012StageProbe{t: t, out: out, ids: []int{0, 1, 2, 3, 4, 5, 6, 7, 8}, controlScreen: ebiten.NewImage(640, 400)}
	if os.Getenv("FD2_HD_COMMAND1012_STAGE_CASES") == "0,8" {
		p.ids = []int{0, 8}
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
	if p.index != len(p.ids) {
		t.Fatal("incomplete stage cases")
	}
	d := map[string]any{"status": "PASS-COMMAND1012-STAGE-GAME", "map_index": 23, "seed": 7, "clock_unix": 1000, "rgba_tolerance": 2, "case_ids": p.ids, "cases": p.cases, "rows": p.rows, "scope": "controlled actual map23; remake pixel-theme control; RUNTIME-E1, not original oracle or normal-player proof"}
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
