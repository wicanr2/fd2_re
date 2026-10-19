package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type modernHDCompoundProbe struct {
	g                             *Game
	owner                         *nativeCommand32PresentationJob
	presentation                  *modernHDPresentation
	target                        *battle.Unit
	out                           string
	seen                          []bool
	off, on, finished             bool
	faithful, hd, finals, updates int
	shots                         map[string]bool
	err                           error
}

func (p *modernHDCompoundProbe) stable() [6]int {
	return [6]int{p.owner.frame, p.owner.repeat, int(p.g.nativeRNGState), p.owner.actor.HP, p.owner.actor.MP, p.target.HP}
}
func (p *modernHDCompoundProbe) Update() error {
	p.updates++
	if p.err != nil || p.finished {
		return ebiten.Termination
	}
	if p.updates > 2400 {
		p.err = fmt.Errorf("compound lifecycle exceeded bound")
		return ebiten.Termination
	}
	j := p.g.nativeCmd32Presentation
	if j == nil {
		p.err = fmt.Errorf("owner disappeared before continuation")
		return ebiten.Termination
	}
	if (!p.off && p.hd > 2) || (p.off && !p.on && p.faithful > 0) {
		before := p.stable()
		p.g.cycleTheme()
		if !p.off {
			p.off = true
		} else {
			p.on = true
		}
		if p.stable() != before {
			p.err = fmt.Errorf("F2 changed transaction or cursor")
			return ebiten.Termination
		}
	}
	if !j.published && (p.target.HP != 100 || p.owner.actor.MP != 80 || p.g.nativeRNGState != 7) {
		p.err = fmt.Errorf("transaction published before original boundary")
		return ebiten.Termination
	}
	p.g.stepNativeCommand32Presentation()
	return nil
}
func (p *modernHDCompoundProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *modernHDCompoundProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.presentation.DrawFinalScreen(screen, offscreen, geometry)
	p.finals++
}
func (p *modernHDCompoundProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.finished {
		return
	}
	j := p.g.nativeCmd32Presentation
	if j == nil {
		return
	}
	index := j.frame
	p.g.Draw(screen)
	if p.g.loadErr != "" {
		p.err = fmt.Errorf("Game.Draw: %s", p.g.loadErr)
		return
	}
	if !j.drawn {
		p.err = fmt.Errorf("present %d was not acknowledged", index)
		return
	}
	p.seen[index] = true
	f := &j.frames[index]
	if f.hdSources == nil {
		if p.g.modernHDBattleVisible {
			p.err = fmt.Errorf("HD battle leaked into native phase")
		}
		return
	}
	if p.g.modernStoryPortraits == nil {
		if p.g.modernHDBattleVisible {
			p.err = fmt.Errorf("faithful F2 kept HD layer")
		}
		p.faithful++
		return
	}
	if !p.g.modernHDBattleVisible {
		p.err = fmt.Errorf("registered HD effect was not drawn")
		return
	}
	p.hd++
	last := f.hdSources.Layers[len(f.hdSources.Layers)-1]
	label := fmt.Sprintf("frame-%d-mix-%.1f", last.Index, f.hdMix)
	if last.DX != 0 || p.shots[label] || p.out == "" {
		return
	}
	// 真正RunGame內呼叫正式最終合成器並回讀1920 GPU；不是X11或一般玩家收據。
	final := ebiten.NewImage(1920, 1200)
	defer final.Dispose()
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.presentation.DrawFinalScreen(final, screen, geometry)
	pixels := make([]byte, 1920*1200*4)
	final.ReadPixels(pixels)
	file, err := os.Create(filepath.Join(p.out, label+".png"))
	if err != nil {
		p.err = err
		return
	}
	err = png.Encode(file, &image.RGBA{Pix: pixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		p.err = err
		return
	}
	p.shots[label] = true
}

func TestModernHDCompoundRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_COMPOUND_GPU") != "1" {
		t.Skip("dedicated compound GPU lifecycle required")
	}
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Fatal("explicit candidate catalog and pack required")
	}
	t.Setenv("FD2_MUTE", "1")
	g, actor, target := nativeCommand32PresentationFixture(t)
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	if set.battleEffects[65] == nil {
		t.Fatal("missing complete effect65")
	}
	g.modernStoryPortraits = set
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	entry := set.battleEffects[65]
	saved := entry.paths[3]
	entry.paths[3] = filepath.Join(t.TempDir(), "missing.png")
	if err := g.startNativeCommand32Presentation(actor, target, nil); err == nil || g.nativeCmd32Presentation != nil || target.HP != 100 || actor.MP != 80 || g.nativeRNGState != 7 || !bytes.Equal(beforeWork, g.nativeMapWork) || !bytes.Equal(beforeVGA, g.nativeMapVGA) || entry.Frames[0] != nil {
		t.Fatal("missing HD preflight partially published transaction or frame")
	}
	entry.paths[3] = saved
	probe := &modernHDCompoundProbe{g: g, target: target, out: os.Getenv("FD2_TEST_MODERN_EFFECT_OUT"), shots: make(map[string]bool)}
	if probe.out != "" {
		if err := os.Mkdir(probe.out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	callbacks := 0
	if err := g.startNativeCommand32Presentation(actor, target, func(battle.NativeCompoundCommand32Result) { callbacks++; probe.finished = true }); err != nil {
		t.Fatal(err)
	}
	probe.owner = g.nativeCmd32Presentation
	probe.seen = make([]bool, len(probe.owner.frames))
	probe.presentation = &modernHDPresentation{Game: g}
	for i := 0; i < 3; i++ {
		g.stepNativeCommand32Presentation()
	}
	if probe.owner.frame != 0 || probe.owner.repeat != 0 || probe.owner.published {
		t.Fatal("owner advanced without Draw")
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if g.modernHDBattleGPU != nil {
		defer g.modernHDBattleGPU.close()
	}
	if probe.err != nil || !probe.finished || callbacks != 1 || g.nativeCmd32Presentation != nil {
		t.Fatalf("lifecycle: %v callbacks%d", probe.err, callbacks)
	}
	for index, seen := range probe.seen {
		if !seen {
			t.Fatalf("present %d skipped", index)
		}
	}
	if target.HP != probe.owner.plan.Result.Targets[0].HPAfter || actor.MP != 80 || !actor.Acted || g.nativeRNGState != probe.owner.plan.Result.RNGState || g.rng.Int63() != rand.New(rand.NewSource(32)).Int63() {
		t.Fatal("final transaction or Go RNG differs")
	}
	if probe.hd == 0 || probe.faithful == 0 || !probe.on || probe.finals == 0 {
		t.Fatal("missing HD/F2/final presentation")
	}
	g.syncModernHDBattleOwner()
	if g.modernHDBattleGPU.owner != nil || len(g.modernHDBattleGPU.sources) != 0 {
		t.Fatal("completed compound retained GPU textures")
	}
	report := map[string]any{"status": "PASS-SCOPED-RUNTIME-E1", "scope": "原始分離素材的既有class19／BattleFig4 fixture；修改隊伍位置與command32資料，非一般玩家、建構槽章收據或新oracle。保留原始frame delays及hold。",
		"present_count": len(probe.seen), "hd_draws": probe.hd, "faithful_draws": probe.faithful, "final_draws": probe.finals, "continuations": callbacks, "screenshots": probe.shots, "hp": target.HP, "mp": actor.MP, "native_rng": g.nativeRNGState, "go_rng_unchanged": true}
	if probe.out != "" {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(filepath.Join(probe.out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d original presents; %d HD / %d faithful draws; F2, missing-source atomicity, publication boundary and one continuation; fixture only", len(probe.seen), probe.hd, probe.faithful)
}
