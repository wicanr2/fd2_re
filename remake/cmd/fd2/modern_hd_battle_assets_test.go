package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func battleHDAssetFixture(t *testing.T) (modernThemeAsset, *figani.Animation, string) {
	t.Helper()
	root := t.TempDir()
	native := &figani.Animation{HeaderByte2: 2, HeaderByte4: 6, Frames: []figani.Frame{
		{X: 20, Y: 30, Width: 2, Height: 2, Delay: 2, RawByte5: 3},
		{X: 19, Y: 31, Width: 2, Height: 2, Delay: 3, RawByte4: 1, RawByte7: 1},
	}}
	a := modernThemeAsset{Role: "battle_animation_hd_set", Status: "runtime_candidate", SourceArchive: "FIGANI.DAT", SourceResource: 91,
		ConsumerContract: "native_figani_rgba_hd6_v1", AlphaContract: "coverage", Width: 1920, Height: 1200,
		CyclePolicy: "native_descriptor_key_poses", LayoutPolicy: "contain_native_descriptor_hd6_candidate", FrameCount: 2, NativeHeader: []int{0, 2, 6}}
	for i, name := range []string{"a.png", "b.png"} {
		picture := image.NewNRGBA(image.Rect(0, 0, 12, 12))
		picture.SetNRGBA(2, 2, color.NRGBA{R: byte(80 + i), G: 140, B: 230, A: 255})
		picture.SetNRGBA(3, 2, color.NRGBA{R: 80, G: 140, B: 230, A: 128})
		f, err := os.Create(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, picture); err != nil {
			t.Fatal(err)
		}
		f.Close()
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(raw)
		a.Files = append(a.Files, name)
		a.FrameSHA256 = append(a.FrameSHA256, hex.EncodeToString(digest[:]))
		frame := native.Frames[i]
		a.NativeFrames = append(a.NativeFrames, modernBattleHDDescriptor{X: frame.X, Y: frame.Y, Width: frame.Width, Height: frame.Height,
			Delay: frame.Delay, RawByte4: frame.RawByte4, RawByte5: frame.RawByte5, RawByte7: frame.RawByte7})
	}
	return a, native, root
}

func TestModernHDBattleAssetsCompleteAndAtomic(t *testing.T) {
	a, native, root := battleHDAssetFixture(t)
	set := &modernStoryPortraitSet{}
	if err := loadModernBattleHDAssetWithNative(set, a, root, native); err != nil {
		t.Fatal(err)
	}
	if len(set.battleAnimations) != 1 || len(set.battleAnimations[91].Frames) != 2 {
		t.Fatal("complete resource not published")
	}
	if _, err := set.battleHDFrame(91, 0); err == nil {
		t.Fatal("unprepared HD frame silently fell back")
	}
	if err := set.prepareBattleHDResources(91); err != nil {
		t.Fatal(err)
	}
	picture, err := set.battleHDFrame(91, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, alpha := picture.At(3, 2).RGBA()
	if alpha != 128*257 {
		t.Fatal("coverage alpha changed")
	}
	if err := loadModernBattleHDAssetWithNative(set, a, root, native); err == nil {
		t.Fatal("duplicate resource accepted")
	}
	if absent, err := set.battleHDFrame(90, 0); absent != nil || err != nil {
		t.Fatal("optional absent resource rejected")
	}
	if _, err := set.battleHDFrame(91, 2); err == nil {
		t.Fatal("invalid HD frame silently fell back")
	}
	for _, kind := range []string{"header", "descriptor", "missing-frame", "sha", "path", "geometry"} {
		t.Run(kind, func(t *testing.T) {
			bad, native, root := battleHDAssetFixture(t)
			switch kind {
			case "header":
				bad.NativeHeader[2]++
			case "descriptor":
				bad.NativeFrames[1].RawByte5++
			case "missing-frame":
				bad.Files = bad.Files[:1]
			case "sha":
				bad.FrameSHA256[1] = bad.FrameSHA256[0]
			case "path":
				bad.Files[1] = "../b.png"
			case "geometry":
				native.Frames[1].Width = 3
				bad.NativeFrames[1].Width = 3
			}
			set := &modernStoryPortraitSet{}
			if err := loadModernBattleHDAssetWithNative(set, bad, root, native); err == nil {
				t.Fatal("invalid resource accepted")
			}
			if len(set.battleAnimations) != 0 {
				t.Fatal("partial resource published")
			}
		})
	}
}

func TestModernHDBattleAssetsLazySetAtomic(t *testing.T) {
	a, native, root := battleHDAssetFixture(t)
	b, other, otherRoot := battleHDAssetFixture(t)
	b.SourceResource = 90
	set := &modernStoryPortraitSet{}
	if err := loadModernBattleHDAssetWithNative(set, a, root, native); err != nil {
		t.Fatal(err)
	}
	if err := loadModernBattleHDAssetWithNative(set, b, otherRoot, other); err != nil {
		t.Fatal(err)
	}
	if set.battleAnimations[91].Frames[0] != nil || set.battleAnimations[90].Frames[0] != nil {
		t.Fatal("all animations eagerly decoded")
	}
	if err := set.prepareBattleHDResources(91, 91); err != nil {
		t.Fatal(err)
	}
	pinned, _ := set.battleHDFrame(91, 0)
	path := filepath.Join(otherRoot, b.Files[1])
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("corrupt after initial validation"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := set.prepareBattleHDResources(90); err == nil {
		t.Fatal("changed file accepted")
	}
	old, _ := set.battleHDFrame(91, 0)
	if old != pinned || set.battleAnimations[90].Frames[0] != nil {
		t.Fatal("failed preflight published partial resources or evicted old images")
	}
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := set.prepareBattleHDResources(90); err != nil {
		t.Fatal(err)
	}
	if set.battleAnimations[90].Frames[0] == nil || set.battleAnimations[91].Frames[0] != nil {
		t.Fatal("successful new scene failed to release prior decoded images")
	}
}

func modernBattleCandidateFigure(t *testing.T) int {
	t.Helper()
	value := os.Getenv("FD2_TEST_MODERN_BATTLE_FIGURE")
	if value == "" {
		return 30 // 保留較早蓋亞候選的明示測試入口。
	}
	figure, err := strconv.Atoi(value)
	if err != nil || figure < 0 || figure > 135 {
		t.Fatal("candidate figure must have a valid FIGANI idle/attack pair")
	}
	return figure
}

func TestModernHDBattleAssetsRealCandidate(t *testing.T) {
	catalog, pack, original := os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK"), os.Getenv("FD2_TEST_ORIGINAL_PACK")
	if catalog == "" || pack == "" || original == "" {
		t.Skip("requires explicit candidate catalog/pack and legal original pack")
	}
	t.Setenv("FD2_ASSET_PACK", original)
	figure := modernBattleCandidateFigure(t)
	idle, attack := figure*3, figure*3+1
	originalIdle, err := figani.LoadSeparatedResource(filepath.Join(original, "animations"), idle)
	if err != nil {
		t.Fatal(err)
	}
	originalAttack, err := figani.LoadSeparatedResource(filepath.Join(original, "animations"), attack)
	if err != nil {
		t.Fatal(err)
	}
	idleCount, attackCount := len(originalIdle.Frames), len(originalAttack.Frames)
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	if set.battleAnimations[idle] == nil || set.battleAnimations[attack] == nil || len(set.battleAnimations[idle].Frames) != idleCount || len(set.battleAnimations[attack].Frames) != attackCount {
		t.Fatal("selected character animation bank incomplete")
	}
	if set.battleAnimations[idle].Frames[0] != nil || set.battleAnimations[attack].Frames[0] != nil {
		t.Fatal("real animation bank eagerly retained all RGBA frames")
	}
	if err := set.prepareBattleHDResources(idle, attack); err != nil {
		t.Fatal(err)
	}
	if set.battleAnimations[idle].Frames[0] == nil || set.battleAnimations[attack].Frames[attackCount-1] == nil {
		t.Fatal("complete active animations not preloaded")
	}
	raw, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var doc modernThemeCatalog
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, asset := range doc.Assets {
		if asset.Role != "battle_animation_hd_set" {
			continue
		}
		asset.SourceMetadataSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
		isolated := &modernStoryPortraitSet{}
		if err := loadModernBattleHDAsset(isolated, asset, pack); err == nil {
			t.Fatal("wrong original metadata accepted")
		}
		if len(isolated.battleAnimations) != 0 {
			t.Fatal("wrong-source resource published")
		}
	}
	t.Logf("complete existing theme + figure%d resources%d/%d loaded; wrong source rejected; no Game.Draw claim", figure, idle, attack)
}

// 明示fixture：只換BattleFig以命中所選圖源，不當成一般玩家路徑。
func TestNativePhysicalHDPreflightStopsBeforeSettlement(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK")
	if catalog == "" || pack == "" {
		t.Skip("requires explicit HD candidate")
	}
	requirePhysicalScenePack(t)
	figure := modernBattleCandidateFigure(t)
	idle, attack := figure*3, figure*3+1
	for _, owner := range []string{"player", "mode11"} {
		t.Run(owner, func(t *testing.T) {
			set, err := loadModernStoryPortraitSet(catalog, pack)
			if err != nil {
				t.Fatal(err)
			}
			if set.battleAnimations[idle] == nil || set.battleAnimations[attack] == nil {
				t.Fatal("selected candidate bank missing")
			}
			set.battleAnimations[idle].paths[0] = filepath.Join(t.TempDir(), "missing-hd-frame.png")
			g, actor, target := physicalSceneTestGame(t)
			actor.BattleFig = figure
			g.modernStoryPortraits = set
			beforeActor, beforeTarget := *actor, *target
			if owner == "player" {
				g.confirm()
			} else {
				g.aiBusy = true
				g.executeNativeAIMode11Physical(&battle.AIPlan{U: actor, Target: target}, nil)
			}
			if !strings.Contains(g.loadErr, "HD source preflight") || g.atk != nil || g.nativeRNGState != 17791 ||
				!reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) {
				t.Fatalf("HD failure did not stop before settlement: %s", g.loadErr)
			}
			if g.rng.Int63() != rand.New(rand.NewSource(73)).Int63() {
				t.Fatal("HD failure consumed Go RNG")
			}
			if set.battleAnimations[idle].Frames[0] != nil || set.battleAnimations[attack].Frames[0] != nil {
				t.Fatal("HD failure published a partial bank")
			}
		})
	}
}
