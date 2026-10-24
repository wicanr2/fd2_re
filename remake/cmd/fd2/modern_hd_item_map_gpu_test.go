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
)

type itemMapGPUCase struct {
	name              string
	spec              itemHDCase
	player            bool
	g, control        *Game
	done, controlDone *int
	seen              map[int]bool
	f2                map[string]bool
	draws, updates    int
}
type itemMapGPUProbe struct {
	t             *testing.T
	checker       unitPresentGPUProbe
	cases         []itemMapGPUCase
	index         int
	out           string
	controlScreen *ebiten.Image
	rows, returns []map[string]any
}

func (p *itemMapGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *itemMapGPUProbe) Update() error {
	if p.checker.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	c := &p.cases[p.index]
	if c.g == nil {
		runtime.GC()
		g, a, b := itemHDGame(p.t, c.spec, c.player)
		control, ca, cb := itemHDGame(p.t, c.spec, c.player)
		g.modernStoryPortraits = p.checker.set
		c.g, c.control = g, control
		c.done, c.controlDone = new(int), new(int)
		startItemHD(p.t, g, a, b, c.spec, c.player, c.done)
		startItemHD(p.t, control, ca, cb, c.spec, c.player, c.controlDone)
		c.seen, c.f2 = map[int]bool{}, map[string]bool{}
		p.t.Logf("prepared item GPU case %s", c.name)
	}
	return nil
}
func (p *itemMapGPUProbe) Draw(screen *ebiten.Image) {
	if p.checker.err != nil || p.index == len(p.cases) {
		return
	}
	c := &p.cases[p.index]
	if c.g == nil {
		return
	}
	g, control := c.g, c.control
	j := g.nativeAIItemPresentation
	if j == nil || control.nativeAIItemPresentation == nil {
		p.checker.err = fmt.Errorf("item owner lost %s: %s", c.name, g.loadErr)
		return
	}
	if !reflect.DeepEqual(itemHDState(g), itemHDState(control)) {
		p.checker.err = fmt.Errorf("native state diverged %s/%d", c.name, c.updates)
		return
	}
	if !j.holding {
		before := itemHDState(g)
		g.stepNativeAIItemPresentation()
		if !reflect.DeepEqual(before, itemHDState(g)) {
			p.checker.err = fmt.Errorf("advanced without Draw")
			return
		}
	}
	g.Draw(screen)
	control.Draw(p.controlScreen)
	if g.loadErr != "" || !g.modernHDWorldVisible || !j.drawn || !reflect.DeepEqual(itemHDState(g), itemHDState(control)) {
		p.checker.err = fmt.Errorf("item Game.Draw failed %s/%d: %s", c.name, j.frame, g.loadErr)
		return
	}
	if !c.seen[j.frame] {
		f := &j.frames[j.frame]
		s := f.hdMap
		if s == nil || s.prepared == nil || g.modernHDMapPresented != s.prepared {
			p.checker.err = fmt.Errorf("wrong frozen HD frame")
			return
		}
		pixels, ok := p.checker.checkWorld(g, s.prepared, healHDCPUWorld(s.prepared, p.checker.set), screen, c.name)
		if !ok {
			return
		}
		digest := sha256.Sum256(pixels)
		if j.frame == 2 {
			for _, layer := range s.layers {
				if layer.Resource == 6 && layer.Entry == 51 {
					if err := (&rowShakeGPUProbe{out: p.out}).save(c.name+"-peak.png", pixels); err != nil {
						p.checker.err = err
						return
					}
					break
				}
			}
		}
		if !c.f2[f.hdPhase] {
			before := itemHDState(g)
			if err := g.applyTheme(nil); err != nil {
				p.checker.err = err
				return
			}
			g.Draw(screen)
			if g.modernHDWorldVisible {
				p.checker.err = fmt.Errorf("native theme retained HD item")
				return
			}
			if err := g.applyTheme(p.checker.set); err != nil {
				p.checker.err = err
				return
			}
			g.Draw(screen)
			again, ok := p.checker.checkWorld(g, s.prepared, healHDCPUWorld(s.prepared, p.checker.set), screen, "F2/"+c.name)
			if !ok {
				return
			}
			if sha256.Sum256(again) != digest || !reflect.DeepEqual(before, itemHDState(g)) {
				p.checker.err = fmt.Errorf("F2 changed item pixels/state")
				return
			}
			c.f2[f.hdPhase] = true
			if err := (&rowShakeGPUProbe{out: p.out}).save(fmt.Sprintf("%s-%s.png", c.name, f.hdPhase), pixels); err != nil {
				p.checker.err = err
				return
			}
		}
		p.rows = append(p.rows, map[string]any{"case": c.name, "frame": j.frame, "phase": f.hdPhase, "sources": s.layers, "final_sha256": fmt.Sprintf("%x", digest), "rng": g.nativeRNGState, "native_vga_sha256": skyKeySHA(g.nativeMapVGA), "published": j.published})
		c.seen[j.frame] = true
	}
	g.stepNativeAIItemPresentation()
	control.stepNativeAIItemPresentation()
	c.updates++
	c.draws++
	if c.updates > 1000 {
		p.checker.err = fmt.Errorf("item exceeded bounded playback")
		return
	}
	if g.nativeAIItemPresentation != nil {
		return
	}
	if control.nativeAIItemPresentation != nil || *c.done != 1 || *c.controlDone != 1 || !reflect.DeepEqual(itemHDState(g), itemHDState(control)) {
		p.checker.err = fmt.Errorf("item return diverged %s", c.name)
		return
	}
	g.Draw(screen)
	control.Draw(p.controlScreen)
	if !g.modernHDWorldVisible || g.modernHDMapPresented == nil || !reflect.DeepEqual(itemHDState(g), itemHDState(control)) {
		p.checker.err = fmt.Errorf("normal HD map return failed %s: %s", c.name, g.loadErr)
		return
	}
	f := g.modernHDMapPresented
	pixels, ok := p.checker.checkWorld(g, f, healHDCPUWorld(f, p.checker.set), screen, "return/"+c.name)
	if !ok {
		return
	}
	p.returns = append(p.returns, map[string]any{"case": c.name, "frames": len(c.seen), "updates": c.updates, "f2_round_trips": len(c.f2), "continuations": *c.done, "native_control_unchanged": true, "return_final_sha256": fmt.Sprintf("%x", sha256.Sum256(pixels))})
	p.t.Logf("completed item GPU case %s", c.name)
	c.g, c.control = nil, nil
	if p.checker.presentation != nil {
		p.checker.presentation.Game = nil
	}
	p.index++
}
func TestModernHDItemMapGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_ITEM_MAP_GPU") != "1" {
		t.Skip("dedicated item HD Game.Draw process")
	}
	out := os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &itemMapGPUProbe{t: t, out: out, controlScreen: ebiten.NewImage(640, 400)}
	p.checker.set = itemHDSet(t)
	for _, spec := range itemHDCases {
		for _, player := range []bool{false, true} {
			if spec.restore && player {
				continue
			}
			p.cases = append(p.cases, itemMapGPUCase{name: fmt.Sprintf("%s-player-%v", spec.name, player), spec: spec, player: player})
		}
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.checker.err != nil {
		t.Fatal(p.checker.err)
	}
	if p.index != 14 || len(p.rows) != 556 || len(p.returns) != 14 {
		t.Fatal("incomplete item GPU coverage", p.index, len(p.rows), len(p.returns))
	}
	raw, err := json.MarshalIndent(map[string]any{"status": "PASS-ITEM-MAP-HD-GAME", "fixed_clock_unix": 1000, "native_seed": 7, "rgba_tolerance": 2, "map_index": 0, "initial_actor_tile": []int{4, 3}, "initial_target_tile": []int{5, 3}, "camera_tile": []int{0, 0}, "cases": p.returns, "rows": p.rows, "scope": "controlled real player/AI entries and Game.Draw; RUNTIME-E1, no normal-player or original RGB parity"}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
