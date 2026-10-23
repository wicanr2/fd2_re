package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestModernHDCommand9ContractAndNativeBlank(t *testing.T) {
	for _, fault := range []string{"none", "ordinary-contract", "wrong-count", "wrong-header", "late-ink", "earlier-blank", "other-resource-blank", "late-mask-ink"} {
		t.Run(fault, func(t *testing.T) {
			root := t.TempDir()
			native := &figani.Animation{HeaderByte2: 31, Frames: make([]figani.Frame, 31)}
			a := modernThemeAsset{Role: "battle_effect_hd_set", Status: "runtime_candidate", SourceArchive: "FDOTHER.DAT", SourceResource: 44,
				ConsumerContract: "native_fdother_command9_ai_rgba_hd6_v1", AlphaContract: "coverage", Width: 1920, Height: 1200,
				CyclePolicy: "native_command9_ai_frames", LayoutPolicy: "native_descriptor_hd6", FrameCount: 31, NativeHeader: []int{0, 31, 0}}
			for i := range native.Frames {
				native.Frames[i] = figani.Frame{Width: 2, Height: 2, Mask: []byte{1, 0, 1, 1}}
				if i == 30 {
					native.Frames[i].Mask = make([]byte, 4)
				}
				picture := image.NewNRGBA(image.Rect(0, 0, 12, 12))
				if i != 30 || fault == "late-ink" {
					picture.SetNRGBA(2, 2, color.NRGBA{R: 70, G: 180, A: 255})
				}
				if i == 30 && fault == "late-mask-ink" {
					native.Frames[i].Mask[0] = 1
				}
				if i == 29 && fault == "earlier-blank" {
					native.Frames[i].Mask = make([]byte, 4)
					picture = image.NewNRGBA(picture.Bounds())
				}
				name := fmt.Sprintf("f%02d.png", i)
				f, err := os.Create(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, picture)
				closeErr := f.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
				raw, err := os.ReadFile(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				h := sha256.Sum256(raw)
				a.Files = append(a.Files, name)
				a.FrameSHA256 = append(a.FrameSHA256, hex.EncodeToString(h[:]))
				a.NativeFrames = append(a.NativeFrames, modernBattleHDDescriptor{Width: 2, Height: 2})
			}
			switch fault {
			case "ordinary-contract":
				a.ConsumerContract = "native_fdother_ordinary_rgba_hd6_v1"
			case "wrong-count":
				native.Frames = native.Frames[:30]
			case "wrong-header":
				native.HeaderByte4, a.NativeHeader[2] = 1, 1
			case "other-resource-blank":
				a.SourceResource, a.ConsumerContract, a.CyclePolicy = 43, "native_fdother_ordinary_rgba_hd6_v1", "native_ordinary_frames"
			}
			set := &modernStoryPortraitSet{}
			err := loadModernBattleHDAssetWithNative(set, a, root, native)
			if fault == "none" {
				if err != nil || set.battleEffects[44] == nil {
					t.Fatal("44 native blank rejected", err)
				}
			} else if err == nil || len(set.battleEffects) != 0 || len(set.battleAnimations) != 0 {
				t.Fatalf("%s published invalid bank: %v", fault, err)
			}
		})
	}
}

func TestModernHDCommand9EffectPreflightAndF2Atomic(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit full theme required")
	}
	t.Setenv("FD2_MUTE", "1")
	template, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	g, actor, target := ordinaryRemainingHDTestGame(t, 9)
	set := cloneModernHDBanksForTest(template)
	g.modernStoryPortraits = set
	a, b, seed := *actor, *target, g.nativeRNGState
	entry := set.battleEffects[44]
	if entry == nil || len(entry.paths) != 31 {
		t.Fatal("full44 required")
	}
	saved := entry.paths[30]
	entry.paths[30] = filepath.Join(t.TempDir(), "missing-native-blank.png")
	err = g.startNativeCommand9AIPresentation(actor, target, nil)
	if err == nil || !strings.Contains(err.Error(), "missing-native-blank.png") || g.nativeCmd9AIPresentation != nil ||
		!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
		t.Fatal("start did not atomically reject missing native blank", err)
	}
	for _, bank := range set.battleAnimations {
		if bank.Frames[0] != nil {
			t.Fatal("partial figure bank")
		}
	}
	for _, bank := range set.battleEffects {
		if bank.Frames[0] != nil {
			t.Fatal("partial effect bank")
		}
	}
	entry.paths[30] = saved
	if err := g.startNativeCommand9AIPresentation(actor, target, nil); err != nil {
		t.Fatal(err)
	}
	j := g.nativeCmd9AIPresentation
	if len(j.hd.handler) != 119 || entry.Frames[30] == nil {
		t.Fatal("119 sources or blank not prepared")
	}
	if err := g.applyTheme(nil); err != nil {
		t.Fatal(err)
	}
	reload := cloneModernHDBanksForTest(template)
	reload.battleEffects[44].paths[30] = filepath.Join(t.TempDir(), "missing-f2-blank.png")
	err = g.applyTheme(reload)
	if err == nil || !strings.Contains(err.Error(), "missing-f2-blank.png") || g.modernStoryPortraits != nil || g.nativeCmd9AIPresentation != j ||
		!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
		t.Fatal("F2 did not atomically reject missing native blank", err)
	}
	for _, bank := range reload.battleAnimations {
		if bank.Frames[0] != nil {
			t.Fatal("partial F2 figure bank")
		}
	}
	for _, bank := range reload.battleEffects {
		if bank.Frames[0] != nil {
			t.Fatal("partial F2 effect bank")
		}
	}
	if err := g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	if g.nativeCmd9AIPresentation != j || !reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
		t.Fatal("F2 replayed state")
	}
}
