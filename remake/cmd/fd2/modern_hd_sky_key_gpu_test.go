package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type skyKeyGPUProbe struct {
	checker                               unitPresentGPUProbe
	g, control                            *Game
	now                                   time.Time
	out                                   string
	controlScreen                         *ebiten.Image
	rows                                  []map[string]any
	counts                                map[nativeCh20SkyKeyPhase]int
	switches, continued, controlContinued int
	done                                  bool
}

func (p *skyKeyGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *skyKeyGPUProbe) Update() error {
	if p.done || p.checker.err != nil {
		return ebiten.Termination
	}
	return nil
}
func (p *skyKeyGPUProbe) Draw(screen *ebiten.Image) {
	if p.done || p.checker.err != nil {
		return
	}
	g, j := p.g, p.g.nativeCh20SkyKey
	if j == nil {
		p.checker.err = fmt.Errorf("sky key owner lost: %s", g.loadErr)
		return
	}
	before := skyKeyNativeState(g)
	g.stepNativeCh20SkyKey()
	if !reflect.DeepEqual(before, skyKeyNativeState(g)) {
		p.checker.err = fmt.Errorf("sky key advanced before Draw")
		return
	}
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || !j.drawn || !reflect.DeepEqual(before, skyKeyNativeState(g)) {
		p.checker.err = fmt.Errorf("sky key Draw changed native source: %s", g.loadErr)
		return
	}
	f := j.hdFrame
	if f == nil {
		p.checker.err = fmt.Errorf("HD sky key frame absent")
		return
	}
	if j.phase == nativeCh20SkyKeyPan || j.phase == nativeCh20SkyKeyFirstFrames || j.phase == nativeCh20SkyKeyTailFrames {
		if len(f.Units) != 2 || f.Units[0].ColourImage == nil || f.Units[1].ColourImage == nil {
			p.checker.err = fmt.Errorf("HD world layers absent")
			return
		}
	} else if f.Viewport.Dx() != 1920 || len(f.Terrain) != 1 {
		p.checker.err = fmt.Errorf("full ANI viewport absent")
		return
	}
	pixels, ok := p.checker.checkWorld(g, f, healHDCPUWorld(f, p.checker.set), screen, fmt.Sprint(j.phase, j.frame))
	if !ok {
		return
	}
	digest := sha256.Sum256(pixels)
	sample := p.counts[j.phase] == 0 || j.phase == nativeCh20SkyKeyFirstFrames && (j.frame == 34 || j.frame == 68) || j.phase == nativeCh20SkyKeyANI && (j.frame == 48 || j.frame == 95) || j.phase == nativeCh20SkyKeyTailFrames && (j.frame == 85 || j.frame == 100)
	if sample {
		if err := g.applyTheme(nil); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.checker.err = fmt.Errorf("native sky key retained HD")
			return
		}
		if err := g.applyTheme(p.checker.set); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		again, ok := p.checker.checkWorld(g, j.hdFrame, healHDCPUWorld(j.hdFrame, p.checker.set), screen, "F2")
		if !ok {
			return
		}
		if sha256.Sum256(again) != digest || !reflect.DeepEqual(before, skyKeyNativeState(g)) {
			p.checker.err = fmt.Errorf("sky key F2 changed source or pixels")
			return
		}
		p.switches++
		if err := (&rowShakeGPUProbe{out: p.out}).save(fmt.Sprintf("phase%d-frame%03d.png", j.phase, j.frame), pixels); err != nil {
			p.checker.err = err
			return
		}
	}
	p.control.Draw(p.controlScreen)
	if !p.control.nativeCh20SkyKey.drawn || !reflect.DeepEqual(skyKeyNativeState(g), skyKeyNativeState(p.control)) {
		p.checker.err = fmt.Errorf("native sky key control diverged")
		return
	}
	p.rows = append(p.rows, map[string]any{"phase": j.phase, "frame": j.frame, "ticks": j.ticks, "final_sha256": fmt.Sprintf("%x", digest), "native_vga_sha256": skyKeySHA(j.vga), "dac_sha256": skyKeySHA(j.dac), "rng": g.nativeRNGState})
	p.counts[j.phase]++
	ticks := j.ticks
	if ticks < 1 {
		ticks = 1
	}
	for tick := 0; tick < ticks; tick++ {
		p.now = p.now.Add(time.Second / 60)
		g.stepNativeCh20SkyKey()
		p.control.stepNativeCh20SkyKey()
	}
	g.frame++
	p.control.frame++
	if g.nativeCh20SkyKey != nil {
		return
	}
	if p.continued != 1 || p.controlContinued != 1 || p.control.nativeCh20SkyKey != nil || !reflect.DeepEqual(skyKeyNativeState(g), skyKeyNativeState(p.control)) {
		p.checker.err = fmt.Errorf("sky key return diverged: %s", g.loadErr)
		return
	}
	g.Draw(screen)
	p.control.Draw(p.controlScreen)
	if !g.modernHDWorldVisible || g.modernHDMapPresented == nil || !reflect.DeepEqual(skyKeyNativeState(g), skyKeyNativeState(p.control)) {
		p.checker.err = fmt.Errorf("sky key normal HD return failed: %s", g.loadErr)
		return
	}
	pixels, ok = p.checker.checkWorld(g, g.modernHDMapPresented, healHDCPUWorld(g.modernHDMapPresented, p.checker.set), screen, "return")
	if !ok {
		return
	}
	p.checker.err = (&rowShakeGPUProbe{out: p.out}).save("return.png", pixels)
	p.done = true
}

func TestModernHDSkyKeyGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SKY_KEY_GPU") != "1" {
		t.Skip("dedicated sky key GPU process")
	}
	out := os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &skyKeyGPUProbe{out: out, now: time.Unix(1000, 0), counts: map[nativeCh20SkyKeyPhase]int{}, controlScreen: ebiten.NewImage(640, 400)}
	p.g, p.control = skyKeyHDTestGame(t, &p.now), skyKeyHDTestGame(t, &p.now)
	p.checker.set = skyKeyHDTestSet(t)
	p.g.modernStoryPortraits = p.checker.set
	spec := recoveredCh20SkyKeySpec(t)
	if err := p.g.startNativeCh20SkyKeySequence(spec, func() { p.continued++ }); err != nil {
		t.Fatal(err)
	}
	if err := p.control.startNativeCh20SkyKeySequence(spec, func() { p.controlContinued++ }); err != nil {
		t.Fatal(err)
	}
	ebiten.SetWindowSize(1920, 1200)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.checker.err != nil {
		t.Fatal(p.checker.err)
	}
	if !p.done || len(p.rows) != 202 || p.switches != 12 || p.continued != 1 {
		t.Fatal("incomplete sky key sequence", len(p.rows), p.switches, p.counts)
	}
	raw, err := json.MarshalIndent(map[string]any{"status": "PASS-SKY-KEY-HD-GAME", "fixed_clock_unix": 1000, "native_seed": 7, "rgba_tolerance": 2, "f2_round_trips": p.switches, "continuations": p.continued, "native_control_unchanged": true, "map_index": 20, "rows": p.rows, "counts": p.counts, "scope": "actual map20 separated banks, two controlled actors; RUNTIME-E1; no original-EXE or normal-player parity"}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
