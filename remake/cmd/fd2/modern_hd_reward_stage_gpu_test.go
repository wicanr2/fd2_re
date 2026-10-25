package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Keep the established text/portrait/panel checker and add an independent
// frozen-world raster, native-theme controller and full return verification.
type rewardStageGPUProbe struct {
	base         rewardHDGPUProbe
	control      []*Game
	controlCalls []*int
	want         []*image.RGBA
	checker      unitPresentGPUProbe
	returns      []map[string]any
}

func (p *rewardStageGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *rewardStageGPUProbe) Update() error              { return p.base.Update() }
func (p *rewardStageGPUProbe) Draw(screen *ebiten.Image) {
	b := &p.base
	if b.err != nil || b.index == len(b.cases) {
		return
	}
	index, phase := b.index, b.frame
	c, control := &b.cases[index], p.control[index]
	g := c.g
	m := g.nativeSystemEndTurnUI.rewardMessage
	if m.hdWorld == nil || len(m.hdWorld.Terrain) == 0 || !m.hdWorld.Terrain[0].Stage {
		b.err = fmt.Errorf("reward stage layer missing")
		return
	}
	if p.want[index] == nil {
		p.want[index] = healHDCPUWorld(m.hdWorld, b.set)
	}
	_, region, ok := modernHDRewardRegion(m, g.nativeClassUIJob)
	if !ok {
		b.err = fmt.Errorf("stage reward phase unavailable")
		return
	}
	layer := m.hdWorld.Terrain[0]
	visible := 0
	for y := 0; y < layer.Image.Bounds().Dy(); y += 6 {
		for x := 0; x < layer.Image.Bounds().Dx(); x += 6 {
			if layer.Image.RGBAAt(x, y).A != 0 && !image.Pt((layer.Rect.Min.X+x)/6, (layer.Rect.Min.Y+y)/6).In(region) {
				visible++
			}
		}
	}
	if visible == 0 {
		b.err = fmt.Errorf("no visible HD stage outside reward panel")
		return
	}
	control.frame++
	if p.checker.controlScreen == nil {
		p.checker.controlScreen = ebiten.NewImage(640, 400)
	}
	p.checker.controlScreen.Clear()
	control.Draw(p.checker.controlScreen)
	b.Draw(screen)
	if b.err != nil {
		return
	}
	if !p.checker.compare(b.world, p.want[index].Pix, "reward stage CPU/GPU") {
		b.err = p.checker.err
		return
	}
	if phase == 6 {
		control.acknowledgeNativeDeathRewardMessage()
	} else {
		control.stepNativeClassUILifecycle(time.Time{})
	}
	if control.loadErr != "" || !reflect.DeepEqual(rewardStageNativeState(g), rewardStageNativeState(control)) || *p.controlCalls[index] != *c.calls {
		b.err = fmt.Errorf("reward phase%d native controller diverged: %s", phase, control.loadErr)
		return
	}
	row := b.rows[len(b.rows)-1]
	row["native_seed"] = g.nativeRNGState
	row["native_control_unchanged"] = true
	row["native_vga_sha256"] = fmt.Sprintf("%x", sha256.Sum256(g.nativeMapVGA))
	row["world_sha256"] = fmt.Sprintf("%x", sha256.Sum256(b.world))
	row["visible_stage_pixels"] = visible
	if b.index == index {
		return
	}

	g.frame++
	control.frame++
	g.Draw(screen)
	p.checker.controlScreen.Clear()
	control.Draw(p.checker.controlScreen)
	if !g.modernHDWorldVisible || g.modernHDMapPresented == nil || g.modernHDOverlay != nil || g.loadErr != "" || !reflect.DeepEqual(rewardStageNativeState(g), rewardStageNativeState(control)) {
		b.err = fmt.Errorf("reward did not return to current HD world: %s", g.loadErr)
		return
	}
	frame := g.modernHDMapPresented
	pixels, ok := p.checker.checkWorld(g, frame, healHDCPUWorld(frame, b.set), screen, "reward stage return")
	if !ok {
		b.err = p.checker.err
		return
	}
	digest := sha256.Sum256(pixels)
	before := rewardStageNativeState(g)
	if err := g.applyTheme(nil); err != nil {
		b.err = err
		return
	}
	g.Draw(screen)
	if g.modernHDWorldVisible {
		b.err = fmt.Errorf("pixel return retained HD")
		return
	}
	if err := g.applyTheme(b.set); err != nil {
		b.err = err
		return
	}
	g.Draw(screen)
	frame = g.modernHDMapPresented
	pixels, ok = p.checker.checkWorld(g, frame, healHDCPUWorld(frame, b.set), screen, "reward stage return F2")
	if !ok {
		b.err = p.checker.err
		return
	}
	if sha256.Sum256(pixels) != digest || !reflect.DeepEqual(before, rewardStageNativeState(g)) || *c.calls != 1 || *p.controlCalls[index] != 1 {
		b.err = fmt.Errorf("reward return F2 changed state or continuation")
		return
	}
	name := fmt.Sprintf("reward-kind%d-return.png", c.kind)
	pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	copy(pic.Pix, pixels)
	f, err := os.Create(filepath.Join(b.out, name))
	if err != nil {
		b.err = err
		return
	}
	err = png.Encode(f, pic)
	closed := f.Close()
	if err != nil || closed != nil {
		b.err = fmt.Errorf("return PNG: %v %v", err, closed)
		return
	}
	p.returns = append(p.returns, map[string]any{"kind": c.kind, "value": c.value, "gold": g.gold, "native_seed": g.nativeRNGState, "continuations": *c.calls, "native_control_unchanged": true, "f2_roundtrip": true, "final_sha256": fmt.Sprintf("%x", digest), "capture": name})
}

func TestModernHDRewardStageGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_REWARD_STAGE_GPU") != "1" {
		t.Skip("dedicated map23 reward process")
	}
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_REWARD_CATALOG"), os.Getenv("FD2_TEST_REWARD_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	out := os.Getenv("FD2_TEST_REWARD_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &rewardStageGPUProbe{base: rewardHDGPUProbe{set: set, out: out, final: ebiten.NewImage(1920, 1200), pixels: make([]byte, 1920*1200*4), world: make([]byte, 1920*1200*4)}, want: make([]*image.RGBA, 2)}
	var sources []map[string]any
	for _, spec := range [][2]int{{0, 192}, {1, 123456}} {
		g, a := rewardStageGame(t, set)
		control, ca := rewardStageGame(t, nil)
		done, cdone := new(int), new(int)
		for i, game := range []*Game{g, control} {
			actor, count := a, done
			if i == 1 {
				actor, count = ca, cdone
			}
			game.pendingNativeRewardMsgs = []pendingNativeDeathRewardMessage{{killer: actor, kind: spec[0], value: spec[1]}}
			if !game.runPendingNativeDeathRewardMessages(actor, func() { *count++ }) || game.loadErr != "" || game.nativeSystemEndTurnUI == nil {
				t.Fatal("pending reward queue", game.loadErr)
			}
		}
		if set.portraits[a.BattleFig] == nil || set.portraits[a.BattleFig].hdImage == nil {
			t.Fatal("actual HD portrait required")
		}
		m := g.nativeSystemEndTurnUI.rewardMessage
		if m.hdScene == nil || !m.hdScene.sealed || m.hdScene.stage == nil {
			t.Fatal("sealed stage source missing")
		}
		sources = append(sources, map[string]any{"kind": spec[0], "map_index": m.hdScene.mapIndex, "scene_sha256": fmt.Sprintf("%x", m.hdScene.signature), "staging_sha256": fmt.Sprintf("%x", sha256.Sum256(m.hdScene.stage.staging))})
		p.base.cases = append(p.base.cases, rewardHDGPUCase{g: g, actor: a, kind: spec[0], value: spec[1], calls: done, snapshot: *a, rng: g.nativeRNGState, final: append([]byte(nil), m.final...)})
		p.control = append(p.control, control)
		p.controlCalls = append(p.controlCalls, cdone)
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.base.err != nil || p.base.index != 2 || len(p.base.rows) != 26 || len(p.returns) != 2 {
		t.Fatal("incomplete stage rewards", p.base.err, p.base.index, len(p.base.rows), len(p.returns))
	}
	d := map[string]any{"status": "PASS-REWARD-STAGE-HD-RUNTIME-E1", "cases": 2, "map_index": 23, "game_frames": 28, "f2_round_trips": 28, "native_seed": 7, "fixed_clock_unix": 1000, "rgba_tolerance": 2, "rows": p.base.rows, "returns": p.returns, "sources": sources, "normal_player_verified": false, "scope": "controlled real-map23 records and pending reward queue, full Game.Draw and native-theme control; no new original oracle or normal-player proof"}
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
