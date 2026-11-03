package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"image"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func modernFieldEvent61Game(t *testing.T) (*Game, *modernStoryPortraitSet) {
	t.Helper()
	set := modifierHDTestTheme(t)
	g, actor := nativeEvent61PlayerGame(t, 0xD0, 0x20)
	// The existing fixture owns startup and selection. Direct entry is explicit.
	g.titlePhase = ""
	if err := g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	plan, err := battle.PlanNativeFieldEvent61(g.st, actor, actor.X, actor.Y)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.beginNativeFieldEvent61Presentation(plan, nil); err != nil {
		t.Fatal(err)
	}
	return g, set
}

func TestModernHDFieldEvent61PreservesEveryNativeWriteAndClock(t *testing.T) {
	g, set := modernFieldEvent61Game(t)
	job := g.nativeFieldEvent61
	decoded, err := fdother.LoadSeparatedEvent61Frames(separatedAssetPath("animations/fdother_045_event61"))
	if err != nil {
		t.Fatal(err)
	}
	native := append([]byte(nil), job.hd.scene.vga...)
	before, _ := json.Marshal(g.st.Units)
	clock, rng := job.clock, g.nativeRNGState
	for i, source := range decoded {
		if err := source.BlitAt(native, 320, 48356, -1); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(native, job.frames[i]) {
			t.Fatal("native cumulative source changed", i)
		}
		frame := job.hd.frames[i]
		patch := frame.Overlays[len(frame.Overlays)-1]
		if patch.Rect != image.Rect(216, 906, 264, 918) || !frame.OverlaysAboveUI {
			t.Fatal("write geometry or ordering changed", i)
		}
		for y := 0; y < 12; y++ {
			for x := 0; x < 48; x++ {
				want := job.hd.scene.palette[native[(151+y/6)*320+36+x/6]]
				if patch.Image.At(x, y) != want {
					// Palette colours may use another concrete Color representation.
					r, gg, b, a := patch.Image.At(x, y).RGBA()
					wr, wg, wb, wa := want.RGBA()
					if r != wr || gg != wg || b != wb || a != wa {
						t.Fatal("write pixel changed", i, x, y)
					}
				}
			}
		}
	}
	job.frame, job.hasTick, job.lastTick, job.drawn = 29, true, 100, true
	for _, ui := range []string{playerInterfaceOriginal, playerInterfaceHD} {
		if err := g.setInterfaceMode(ui); err != nil {
			t.Fatal(err)
		}
		for _, art := range []*modernStoryPortraitSet{nil, set} {
			if err := g.applyTheme(art); err != nil {
				t.Fatal(err)
			}
			if job.frame != 29 || !job.drawn || job.lastTick != 100 || !job.hasTick || job.clock != clock {
				t.Fatal("display switch advanced event")
			}
		}
	}
	after, _ := json.Marshal(g.st.Units)
	if !bytes.Equal(before, after) || rng != g.nativeRNGState || g.st.NativeEventState[12] != 0 {
		t.Fatal("display changed gameplay state")
	}
}

func TestModernHDFieldEvent61RejectsChangedSourcesBeforeThemePublish(t *testing.T) {
	g, set := modernFieldEvent61Game(t)
	job, h := g.nativeFieldEvent61, g.nativeFieldEvent61.hd
	if err := g.applyTheme(nil); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name   string
		change func() func()
	}{
		{"frame", func() func() { job.frames[0][48356] ^= 1; return func() { job.frames[0][48356] ^= 1 } }},
		{"outside rectangle", func() func() {
			old := h.hashes[0]
			job.frames[0][0] ^= 1
			h.hashes[0] = sha256.Sum256(job.frames[0])
			return func() { job.frames[0][0] ^= 1; h.hashes[0] = old }
		}},
		{"frame count", func() func() { old := job.frames; job.frames = job.frames[:58]; return func() { job.frames = old } }},
		{"delay", func() func() {
			job.plan.Presentation.DelayTicks++
			return func() { job.plan.Presentation.DelayTicks-- }
		}},
		{"source", func() func() { h.scene.vga[0] ^= 1; return func() { h.scene.vga[0] ^= 1 } }},
		{"map", func() func() { h.scene.mapIndex++; return func() { h.scene.mapIndex-- } }},
		{"unsealed", func() func() { h.scene.sealed = false; return func() { h.scene.sealed = true } }},
		{"missing snapshot", func() func() { job.hd = nil; return func() { job.hd = h } }},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			restore := c.change()
			defer restore()
			if err := g.applyTheme(set); err == nil {
				t.Fatal("changed source accepted")
			}
			if g.modernStoryPortraits != nil || g.nativeFieldEvent61 != job || job.frame != 0 || job.drawn || job.hasTick {
				t.Fatal("failed preflight published theme or event state")
			}
		})
	}
	partial := *set
	partial.mapSpritesHD = map[int][]image.Image{0: set.mapSpritesHD[0]}
	if err := g.applyTheme(&partial); err == nil || g.modernStoryPortraits != nil {
		t.Fatal("partial world accepted")
	}
	if err := g.applyTheme(set); err != nil {
		t.Fatal("valid source did not recover", err)
	}
}
