package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestModernHDOrdinaryEffectContract(t *testing.T) {
	for _, resource := range []int{18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 30, 32, 33, 37, 38, 39, 43} {
		t.Run(fmt.Sprint(resource), func(t *testing.T) {
			a, native, root := battleHDAssetFixture(t)
			a.Role, a.SourceArchive, a.SourceResource = "battle_effect_hd_set", "FDOTHER.DAT", resource
			a.ConsumerContract, a.CyclePolicy, a.LayoutPolicy = "native_fdother_ordinary_rgba_hd6_v1", "native_ordinary_frames", "native_descriptor_hd6"
			for _, fault := range []string{"consumer", "descriptor", "none"} {
				bad := a
				bad.NativeFrames = append([]modernBattleHDDescriptor(nil), a.NativeFrames...)
				if fault == "consumer" {
					bad.ConsumerContract = "native_fdother_compound_rgba_hd6_v1"
				}
				if fault == "descriptor" {
					bad.NativeFrames[1].RawByte7 ^= 1
				}
				set := &modernStoryPortraitSet{}
				err := loadModernBattleHDAssetWithNative(set, bad, root, native)
				if fault != "none" {
					if err == nil || len(set.battleEffects) != 0 || len(set.battleAnimations) != 0 {
						t.Fatalf("bad %s published: %v", fault, err)
					}
				} else if err != nil || set.battleEffects[resource] == nil || len(set.battleAnimations) != 0 {
					t.Fatalf("ordinary effect not admitted: %v", err)
				}
			}
		})
	}
	for _, resource := range []int{-1, 0, 5, 17, 29, 31, 34, 36, 40, 42, 44, 64, 69, 409} {
		_, _, _, _, ok := modernBattleHDContract(modernThemeAsset{Role: "battle_effect_hd_set", SourceArchive: "FDOTHER.DAT", SourceResource: resource})
		if ok {
			t.Fatalf("unproved effect %d admitted", resource)
		}
	}
}

// Only fault-injection PNGs. They never enter the catalog or claim finished art.
func ordinaryEffectFaultEntry(t *testing.T, resource int) *modernBattleHDResource {
	t.Helper()
	a, native, root := battleHDAssetFixture(t)
	a.Role, a.SourceArchive, a.SourceResource = "battle_effect_hd_set", "FDOTHER.DAT", resource
	a.ConsumerContract, a.CyclePolicy, a.LayoutPolicy = "native_fdother_ordinary_rgba_hd6_v1", "native_ordinary_frames", "native_descriptor_hd6"
	set := &modernStoryPortraitSet{}
	if err := loadModernBattleHDAssetWithNative(set, a, root, native); err != nil {
		t.Fatal(err)
	}
	return set.battleEffects[resource]
}

func TestModernHDOrdinaryEffectPreflightAndF2Atomic(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit full theme required")
	}
	t.Setenv("FD2_MUTE", "1")
	template, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	for command := 0; command <= 8; command++ {
		t.Run(fmt.Sprint(command), func(t *testing.T) {
			var g *Game
			var actor, target *battle.Unit
			switch command {
			case 0:
				g, actor, target = ordinaryCommand0HDTestGame(t)
			case 1:
				g, actor, target = ordinaryCommand1HDTestGame(t)
			case 4, 5:
				g, actor, target = ordinaryCommand5HDTestGame(t)
				if command == 4 {
					record := g.st.NativeCommandBook[5]
					record.ID = 4
					g.st.NativeCommandBook[4] = record
				}
			default:
				g, actor, target = ordinaryRemainingHDTestGame(t, command)
			}
			start := func() error {
				switch command {
				case 0:
					return g.startNativeCommand0Presentation(actor, target, nil)
				case 1:
					return g.startNativeCommand1Presentation(actor, target, nil)
				case 4, 5:
					return g.startNativeCommand45Presentation(actor, target, command, nil)
				}
				return startOrdinaryRemainingHD(g, command, actor, target, nil)
			}
			owner := func() (any, modernHDCommandBankSource) {
				switch command {
				case 0:
					if j := g.nativeCmd0Presentation; j != nil {
						return j, j.hd
					}
				case 1:
					if j := g.nativeCmd1Presentation; j != nil {
						return j, j.hd
					}
				case 4, 5:
					if j := g.nativeCmd5Presentation; j != nil {
						return j, j.hd
					}
				default:
					j := ordinaryRemainingOwner(g, command)
					return j.owner, j.hd
				}
				return nil, nil
			}
			effectID := map[int]int{0: 18, 1: 19, 2: 26, 3: 39, 4: 22, 5: 24, 6: 32, 7: 37, 8: 28}[command]
			if actor.NativeRecordByte6 == 0 {
				effectID = map[int]int{0: 20, 1: 21, 2: 27, 3: 43, 4: 23, 5: 25, 6: 33, 7: 38, 8: 30}[command]
			}
			set := cloneModernHDBanksForTest(template)
			entry := ordinaryEffectFaultEntry(t, effectID)
			set.battleEffects[effectID] = entry
			g.modernStoryPortraits = set
			a, b, seed := *actor, *target, g.nativeRNGState
			pinned := image.NewNRGBA(image.Rect(0, 0, 1, 1))
			set.battleAnimations[12].Frames[0], set.battleEffects[65].Frames[0] = pinned, pinned
			last := len(entry.paths) - 1
			saved := entry.paths[last]
			entry.paths[last] = filepath.Join(t.TempDir(), "missing-late-effect.png")
			err := start()
			active, _ := owner()
			if err == nil || !strings.Contains(err.Error(), "missing-late-effect.png") || active != nil ||
				!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
				t.Fatalf("late effect failure changed transaction: %v", err)
			}
			for id, bank := range set.battleAnimations {
				if id == 12 {
					if bank.Frames[0] != pinned {
						t.Fatal("failure evicted pinned figure")
					}
					continue
				}
				if bank.Frames[0] != nil {
					t.Fatal("failure published partial figure bank", id)
				}
			}
			for id, bank := range set.battleEffects {
				if id == 65 {
					if bank.Frames[0] != pinned {
						t.Fatal("failure evicted pinned effect")
					}
					continue
				}
				if bank.Frames[0] != nil {
					t.Fatal("failure published partial effect bank", id)
				}
			}
			entry.paths[last] = saved
			if err := start(); err != nil {
				t.Fatal(err)
			}
			active, source := owner()
			if active == nil || source == nil || !reflect.DeepEqual(modernCompoundEffectResources(source.sourceFrames()), []int{effectID}) ||
				entry.Frames[last] == nil || set.battleAnimations[12].Frames[0] != nil || set.battleEffects[65].Frames[0] != nil {
				t.Fatal("successful start did not publish both actual banks or release old ones")
			}
			if err := g.applyTheme(nil); err != nil {
				t.Fatal(err)
			}
			reload := cloneModernHDBanksForTest(template)
			reload.battleEffects[effectID] = ordinaryEffectFaultEntry(t, effectID)
			reload.battleEffects[effectID].paths[last] = filepath.Join(t.TempDir(), "missing-f2-effect.png")
			err = g.applyTheme(reload)
			after, _ := owner()
			if err == nil || !strings.Contains(err.Error(), "missing-f2-effect.png") || g.modernStoryPortraits != nil || active != after ||
				!reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
				t.Fatalf("failed F2 changed state/owner: %v", err)
			}
			for id, bank := range reload.battleAnimations {
				if bank.Frames[0] != nil {
					t.Fatal("failed F2 published figure", id)
				}
			}
			for id, bank := range reload.battleEffects {
				if bank.Frames[0] != nil {
					t.Fatal("failed F2 published effect", id)
				}
			}
			if err := g.applyTheme(set); err != nil {
				t.Fatal(err)
			}
			after, _ = owner()
			if active != after || entry.Frames[last] == nil || !reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) || seed != g.nativeRNGState {
				t.Fatal("successful F2 replayed transaction or lost effect")
			}
		})
	}
}
