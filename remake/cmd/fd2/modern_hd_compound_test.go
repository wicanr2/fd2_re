package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func appendModernCompoundGPUCases(t *testing.T, cases []modernHDBattleGPUCase, palette color.Palette) []modernHDBattleGPUCase {
	t.Helper()
	dac, _, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{}
	base := make([]byte, 64000)
	for i := range base {
		base[i] = byte(i % 256)
	}
	actor := figani.Frame{Width: 1, Height: 1, Pixels: []byte{0}, Mask: []byte{0}}
	for command := 32; command <= 35; command++ {
		effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", command+33)
		if err != nil {
			t.Fatal(err)
		}
		schedule, err := figani.BuildNativeCompoundPresentationSchedule(command, effect)
		if err != nil {
			t.Fatal(err)
		}
		original, err := battlepresent.BuildNativeCompoundCommonFrames(base, actor, effect, schedule)
		if err != nil {
			t.Fatal(err)
		}
		for _, phase := range []struct {
			name   string
			frames [][]byte
		}{{"slide", original.EffectSlide}, {"main", original.Main}, {"tail", original.Tail}} {
			if len(phase.frames) == 0 {
				continue
			}
			frames := make([]nativeCompoundPresentedFrame, len(phase.frames))
			if err := bindModernHDCompoundFrames(frames, base, effect, schedule, phase.name, dac); err != nil {
				t.Fatal(err)
			}
			for i, frame := range frames {
				plan, err := g.modernHDBattlePlan(frame.hdSources, frame.hdPalette, 0)
				if err != nil {
					t.Fatal(err)
				}
				plan.HDMix, plan.HDTarget = frame.hdMix, frame.hdTarget
				cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("compound/%d/%s/%d", command, phase.name, i), plan: plan, indexed: phase.frames[i]})
			}
		}
	}
	// 新RGB插值只作用在HD層；原生fallback在上面的22個tail使用真正DAC。
	hd := image.NewNRGBA(image.Rect(0, 0, 12, 12))
	for y := 1; y < 11; y++ {
		for x := 1; x < 11; x++ {
			a := byte(255)
			if x%2 == 0 {
				a = 128
			}
			hd.SetNRGBA(x, y, color.NRGBA{R: 230, G: 70, B: 10, A: a})
		}
	}
	for _, raw := range [][3]byte{{63, 63, 63}, {51, 57, 63}, {53, 0, 0}, {53, 58, 9}} {
		for _, delta := range []int{40, 20, 0} {
			mix := 1 - float32(delta)/40
			var target [3]float32
			for c, v := range raw {
				target[c] = float32((v<<2)|(v>>4)) / 255
			}
			pos := image.Pt(40, 60)
			plan := modernHDBattlePlan{Base: make([]byte, 64000), Palette: palette, HDMix: mix, HDTarget: target,
				Layers: []modernHDBattleLayer{{HD: hd, HDKey: "compound-tint", Rect: hd.Bounds().Add(pos), Fill: -1}}}
			want := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
			draw.Draw(want, want.Bounds(), &image.Uniform{C: palette[0]}, image.Point{}, draw.Src)
			art := image.NewNRGBA(hd.Bounds())
			for y := 0; y < 12; y++ {
				for x := 0; x < 12; x++ {
					v := hd.NRGBAAt(x, y)
					channels := []byte{v.R, v.G, v.B}
					for c := range channels {
						channels[c] = byte(math.Round(float64(float32(channels[c])*(1-mix) + target[c]*255*mix)))
					}
					art.SetNRGBA(x, y, color.NRGBA{R: channels[0], G: channels[1], B: channels[2], A: v.A})
				}
			}
			draw.Draw(want, hd.Bounds().Add(pos), art, image.Point{}, draw.Over)
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("compound-hd-tint/%v/%d", raw, delta), plan: plan, want: want, tolerance: 1})
		}
	}
	return cases
}

func TestModernHDCompoundAssetsNamespacesAndAtomicPreload(t *testing.T) {
	a, native, root := battleHDAssetFixture(t)
	a.SourceResource = 65
	set := &modernStoryPortraitSet{}
	if err := loadModernBattleHDAssetWithNative(set, a, root, native); err != nil {
		t.Fatal(err)
	}
	a.Role, a.SourceArchive = "battle_effect_hd_set", "FDOTHER.DAT"
	a.ConsumerContract, a.CyclePolicy, a.LayoutPolicy = "native_fdother_compound_rgba_hd6_v1", "native_compound_frames", "native_descriptor_hd6"
	if err := loadModernBattleHDAssetWithNative(set, a, root, native); err != nil {
		t.Fatal(err)
	}
	if set.battleEffects[65] == set.battleAnimations[65] {
		t.Fatal("archive namespaces collided")
	}
	for _, fault := range []string{"archive", "resource", "contract", "layout"} {
		bad := a
		switch fault {
		case "archive":
			bad.SourceArchive = "FIGANI.DAT"
		case "resource":
			bad.SourceResource = 64
		case "contract":
			bad.ConsumerContract = "native_figani_rgba_hd6_v1"
		case "layout":
			bad.LayoutPolicy = "contain_native_descriptor_hd6_candidate"
		}
		if err := loadModernBattleHDAssetWithNative(&modernStoryPortraitSet{}, bad, root, native); err == nil {
			t.Fatalf("bad %s accepted", fault)
		}
	}
	entry := set.battleEffects[65]
	held := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	set.battleEffects[66] = &modernBattleHDResource{Frames: []image.Image{held}, paths: []string{"held.png"}}
	saved := entry.paths[1]
	entry.paths[1] = filepath.Join(root, "missing.png")
	if err := set.prepareBattleHDEffects(65); err == nil || entry.Frames[0] != nil || set.battleEffects[66].Frames[0] != held {
		t.Fatal("failed preload partially published or evicted old frame")
	}
	entry.paths[1] = saved
	if err := set.prepareBattleHDEffects(65); err != nil {
		t.Fatal(err)
	}
	if entry.Frames[0] == nil || set.battleEffects[66].Frames[0] != nil || set.battleAnimations[65].Frames[0] != nil {
		t.Fatal("preload crossed archive or retained unrelated effect")
	}
	if _, err := set.battleHDFrame(65, 0); err == nil {
		t.Fatal("effect preload accidentally prepared FIGANI65")
	}
}
