package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type command1012GPUCase struct {
	g             *Game
	actor, target *battle.Unit
	job           *nativeCommand1012Job
	hit           bool
	done          *int
	f2            map[string]bool
}
type command1012GPUProbe struct {
	set          *modernStoryPortraitSet
	cases        []command1012GPUCase
	index        int
	err          error
	rows         []map[string]any
	worlds       map[string]*image.RGBA
	pixels       []byte
	final        *ebiten.Image
	presentation *modernHDPresentation
	out          string
}

func (p *command1012GPUProbe) Update() error {
	if p.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *command1012GPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *command1012GPUProbe) check(g *Game, frame *modernHDMapFrame, key string, screen *ebiten.Image) bool {
	want := p.worlds[key]
	if want == nil {
		want = player9CPUWorld(frame, p.set)
		p.worlds[key] = want
	}
	if g.modernHDMapGPU == nil {
		p.err = fmt.Errorf("missing HD GPU world")
		return false
	}
	g.modernHDMapGPU.World.ReadPixels(p.pixels)
	compare := func() bool {
		for i, v := range want.Pix {
			d := int(p.pixels[i]) - int(v)
			if d < -2 || d > 2 {
				p.err = fmt.Errorf("%s GPU byte%d got%d want%d", key, i, p.pixels[i], v)
				return false
			}
		}
		return true
	}
	if !compare() {
		return false
	}
	if p.final == nil {
		p.final = ebiten.NewImage(1920, 1200)
		p.presentation = &modernHDPresentation{}
	}
	p.presentation.Game = g
	p.final.Clear()
	var geom ebiten.GeoM
	geom.Scale(3, 3)
	p.presentation.DrawFinalScreen(p.final, screen, geom)
	p.final.ReadPixels(p.pixels)
	return compare()
}
func (p *command1012GPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= len(p.cases) {
		return
	}
	c := &p.cases[p.index]
	g, j := c.g, c.job
	phase, frame, hold := j.phase, j.frame, j.hold
	prelude := g.native2189A
	if prelude != nil {
		frame = prelude.index
	}
	expectedMP := j.actorMPBefore
	if phase == nativeCommand1012Result || phase == nativeCommand1012Hold || phase == nativeCommand1012Main && frame > 0 {
		expectedMP = j.plan.MPAfter
	}
	expectedHP, expectedRNG := j.plan.Results[0].HPBefore, j.rngBefore
	if phase == nativeCommand1012Result || phase == nativeCommand1012Hold {
		expectedHP = j.plan.Results[0].HPAfter
		expectedRNG = j.plan.RNGAfter
	}
	if c.actor.MP != expectedMP || c.target.HP != expectedHP || g.nativeRNGState != expectedRNG || c.actor.Acted {
		p.err = fmt.Errorf("transaction boundary cmd%d hit%t phase%d frame%d", j.schedule.CommandID, c.hit, phase, frame)
		return
	}
	for _, result := range j.plan.Results {
		hp := result.HPBefore
		if phase == nativeCommand1012Result || phase == nativeCommand1012Hold {
			hp = result.HPAfter
		}
		if result.Target.HP != hp {
			p.err = fmt.Errorf("multi-target publication boundary")
			return
		}
	}
	g.frame++
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || g.modernHDMapPresented == nil {
		p.err = fmt.Errorf("Game.Draw did not publish full HD: %s", g.loadErr)
		return
	}
	if prelude != nil && !prelude.drawn || prelude == nil && !j.drawn {
		p.err = fmt.Errorf("HD did not acknowledge actual Draw")
		return
	}
	key := fmt.Sprintf("%d/%d", phase, frame)
	if phase == nativeCommand1012Main {
		key = fmt.Sprintf("main/%d", frame%4)
	}
	if !p.check(g, g.modernHDMapPresented, key, screen) {
		return
	}
	digest := sha256.Sum256(p.pixels)
	row := map[string]any{"command": j.schedule.CommandID, "hit": c.hit, "phase": phase, "frame": frame, "hold": hold, "mp": c.actor.MP, "hp": c.target.HP, "rng": g.nativeRNGState, "gpu_sha256": hex.EncodeToString(digest[:]), "rgba_tolerance": 2}
	row["target_count"] = len(j.plan.Results)
	f2key := fmt.Sprintf("%d/%d", phase, frame)
	f2point := phase == nativeCommand1012Prelude && (frame == 0 || frame == 9 || frame == 10) || phase == nativeCommand1012Main && (frame < 4 || frame == 59) || phase == nativeCommand1012Result && (frame == 0 || frame == 21) || phase == nativeCommand1012Hold
	if f2point && !c.f2[f2key] {
		if err := g.applyTheme(nil); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.err = fmt.Errorf("faithful F2 retained HD")
			return
		}
		if err := g.applyTheme(p.set); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if g.nativeCmd1012 != j || j.phase != phase || j.hold != hold || prelude == nil && j.frame != frame || prelude != nil && (g.native2189A != prelude || prelude.index != frame) || c.actor.MP != expectedMP || c.target.HP != expectedHP || g.nativeRNGState != expectedRNG || c.actor.Acted {
			p.err = fmt.Errorf("F2 changed owner or transaction")
			return
		}
		if !p.check(g, g.modernHDMapPresented, key, screen) || sha256.Sum256(p.pixels) != digest {
			if p.err == nil {
				p.err = fmt.Errorf("F2 changed HD world")
			}
			return
		}
		c.f2[f2key] = true
		row["f2_round_trip"] = true
	}
	if c.hit && (phase == nativeCommand1012Main && frame < 4 || phase == nativeCommand1012Prelude && (frame == 0 || frame == 5 || frame == 9) || phase == nativeCommand1012Result && frame == 4) {
		name := fmt.Sprintf("command%d-targets%d-phase%d-frame%02d.png", j.schedule.CommandID, len(j.plan.Results), phase, frame)
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, p.pixels)
		f, err := os.Create(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		err = png.Encode(f, pic)
		closeErr := f.Close()
		if err != nil {
			p.err = err
			return
		}
		if closeErr != nil {
			p.err = closeErr
			return
		}
		row["capture"] = name
	}
	p.rows = append(p.rows, row)
	if prelude != nil {
		g.stepNative2189A()
	} else {
		g.stepNativeCommand1012Presentation()
	}
	if g.nativeCmd1012 == nil {
		wantF2 := 8
		if j.schedule.CommandID != 10 {
			wantF2 = 11
		}
		if *c.done != 1 || c.actor.MP != j.plan.MPAfter || !c.actor.Acted || g.nativeRNGState != j.plan.RNGAfter || c.target.HP != j.plan.Results[0].HPAfter || len(c.f2) != wantF2 {
			p.err = fmt.Errorf("incomplete transaction/callback/F2 cmd%d", j.schedule.CommandID)
			return
		}
		fmt.Printf("command%d hit%t complete; f2=%d callback=%d\n", j.schedule.CommandID, c.hit, len(c.f2), *c.done)
		p.index++
		p.worlds = make(map[string]*image.RGBA)
	}
}
func TestModernHDCommand1012CompleteGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_COMMAND1012_GPU") != "1" {
		t.Skip("dedicated complete Game.Draw process required")
	}
	id, err := strconv.Atoi(os.Getenv("FD2_TEST_COMMAND1012_ID"))
	if err != nil || id < 10 || id > 12 {
		t.Fatal("explicit command ID required")
	}
	out := os.Getenv("FD2_TEST_COMMAND1012_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_MUTE", "1")
	set := command1012HDTestTheme(t)
	p := &command1012GPUProbe{set: set, pixels: make([]byte, 1920*1200*4), worlds: make(map[string]*image.RGBA), out: out}
	for _, variant := range []int{0, 1, 2} {
		hit := variant != 1
		g, actor, target := command1012HDTestGame(t, id, hit)
		if variant == 2 {
			g, actor, target = command1012HDTestMultiGame(t, id)
		}
		g.modernStoryPortraits = set
		done := new(int)
		if err := g.startNativeCommand1012Presentation(actor, target, id, func([]battle.NativeCommandDamageResult) { *done++ }); err != nil {
			t.Fatal(err)
		}
		if g.nativeCmd1012.hdFrames == nil {
			t.Fatal("missing full HD bundle")
		}
		p.cases = append(p.cases, command1012GPUCase{g: g, actor: actor, target: target, job: g.nativeCmd1012, hit: hit, done: done, f2: map[string]bool{}})
	}
	ebiten.SetTPS(ebiten.SyncWithFPS)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	want := 339
	if id != 10 {
		want = 372
	}
	if len(p.rows) != want || p.index != 3 {
		t.Fatalf("incomplete Game.Draw %d", len(p.rows))
	}
	data, err := json.MarshalIndent(map[string]any{"status": "PASS-COMMAND1012-COMPLETE-GAME-GPU", "command": id, "full_hd_draws": len(p.rows), "rows": p.rows, "scope": "seed7 controlled hit/MISS, damage999, relocated actors; RUNTIME-E1, not normal player or original parity"}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "gpu.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
