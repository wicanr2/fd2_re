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

func TestPhysicalBattleAuditSourcesReadOnly(t *testing.T) {
	frame := figani.Frame{X: 32, Y: 40, Width: 120, Height: 130}
	scene := &nativePhysicalScene{body: &nativePhysicalBodyPlayback{jobs: []nativePhysicalBodyJob{{hdPhase: "counter",
		hdSources: &modernHDBattleSources{Layers: []modernHDBattleSource{{Frame: frame, Resource: 7, Index: 3, DX: 4, DY: 5, Fill: -1}}}}}}}
	index, phase, rows := physicalBattleAuditSources(scene)
	if index != 0 || phase != "counter" || len(rows) != 1 || rows[0]["resource"] != 7 || rows[0]["frame"] != 3 || rows[0]["x"] != 36 || rows[0]["y"] != 45 {
		t.Fatalf("unexpected audit sources: %d %s %v", index, phase, rows)
	}
	rows[0]["resource"] = 99
	if scene.body.index != 0 || scene.body.jobs[0].hdSources.Layers[0].Resource != 7 || !reflect.DeepEqual(scene.body.jobs[0].hdSources.Layers[0].Frame, frame) {
		t.Fatal("audit mutated playback or source")
	}
	for _, state := range []string{"nil", "prelude", "past-end", "cue-only"} {
		t.Run(state, func(t *testing.T) {
			copyScene := *scene
			copyBody := *scene.body
			copyScene.body = &copyBody
			switch state {
			case "nil":
				copyScene.body = nil
			case "prelude":
				copyScene.preludeImages = append(copyScene.preludeImages, nil)
			case "past-end":
				copyBody.index = len(copyBody.jobs)
			case "cue-only":
				copyBody.jobs = []nativePhysicalBodyJob{{}}
			}
			_, _, layers := physicalBattleAuditSources(&copyScene)
			if len(layers) != 0 {
				t.Fatalf("non-present state emitted sources: %v", layers)
			}
		})
	}
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

func TestModernHDBattleEmptyFrameRequiresCompleteNativeMask(t *testing.T) {
	for _, kind := range []string{"native-empty", "missing-mask", "short-mask", "nonempty-native", "opaque-ink", "soft-ink", "late-opaque-ink", "late-soft-ink", "other-archive"} {
		t.Run(kind, func(t *testing.T) {
			a, native, root := battleHDAssetFixture(t)
			native.Frames[1].Mask = make([]byte, 4)
			picture := image.NewNRGBA(image.Rect(0, 0, 12, 12))
			switch kind {
			case "missing-mask":
				native.Frames[1].Mask = nil
			case "short-mask":
				native.Frames[1].Mask = make([]byte, 3)
			case "nonempty-native":
				native.Frames[1].Mask[3] = 1
			case "opaque-ink":
				picture.SetNRGBA(1, 1, color.NRGBA{R: 255, A: 255})
			case "soft-ink":
				picture.SetNRGBA(1, 1, color.NRGBA{R: 255, A: 1})
			case "late-opaque-ink":
				picture.SetNRGBA(11, 11, color.NRGBA{R: 255, A: 255})
			case "late-soft-ink":
				picture.SetNRGBA(11, 11, color.NRGBA{R: 255, A: 1})
			case "other-archive":
				a.Role, a.SourceArchive, a.SourceResource = "battle_effect_hd_set", "FDOTHER.DAT", 65
				a.ConsumerContract, a.CyclePolicy, a.LayoutPolicy = "native_fdother_compound_rgba_hd6_v1", "native_compound_frames", "native_descriptor_hd6"
			}
			path := filepath.Join(root, a.Files[1])
			file, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, picture); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(raw)
			a.FrameSHA256[1] = hex.EncodeToString(digest[:])
			set := &modernStoryPortraitSet{}
			err = loadModernBattleHDAssetWithNative(set, a, root, native)
			if kind != "native-empty" {
				if err == nil || len(set.battleAnimations) != 0 || len(set.battleEffects) != 0 {
					t.Fatal("invalid empty frame published")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := set.prepareBattleHDResources(91); err != nil {
				t.Fatal(err)
			}
			blank, err := set.battleHDFrame(91, 1)
			if err != nil || blank == nil {
				t.Fatal("empty frame treated as missing", err)
			}
			for y := 0; y < 12; y++ {
				for x := 0; x < 12; x++ {
					_, _, _, a := blank.At(x, y).RGBA()
					if a != 0 {
						t.Fatal("empty frame gained ink")
					}
				}
			}
		})
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

// 攻防fixture原本也只使用battleAnimations。完整主題另由來源GPU與冷預檢驗證。
// 這裡仍沿正式逐組loader核對每一個場景實際請求的PNG／SHA／原描述子。
func modernHDBattleFixtureSet(t *testing.T, catalog, pack string, resources []int) *modernStoryPortraitSet {
	t.Helper()
	raw, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var doc modernThemeCatalog
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	wanted := make(map[int]bool)
	for _, resource := range resources {
		wanted[resource] = true
	}
	set := &modernStoryPortraitSet{hasHD: true}
	for _, asset := range doc.Assets {
		if asset.Role != "battle_animation_hd_set" || !wanted[asset.SourceResource] {
			continue
		}
		if err := loadModernBattleHDAsset(set, asset, pack); err != nil {
			t.Fatal(err)
		}
	}
	for resource := range wanted {
		if set.battleAnimations[resource] == nil {
			t.Fatalf("physical fixture requested missing HD bank %d", resource)
		}
	}
	return set
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
// TestModernHDBattleBatchAssetsAndPreflight 共用兩份完整主題載入，逐人物保留獨立交易案例。
// 成功預載與缺件交易使用不同 set；缺件案例不共用已預載的動畫。
func TestModernHDBattleBatchAssetsAndPreflight(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK")
	values := os.Getenv("FD2_TEST_MODERN_BATTLE_FIGURES")
	if catalog == "" || pack == "" || values == "" {
		t.Skip("requires explicit batch catalog/pack/figures")
	}
	requirePhysicalScenePack(t)
	var figures []int
	selected := map[int]bool{}
	for _, value := range strings.Split(values, ",") {
		figure, err := strconv.Atoi(value)
		if err != nil || figure < 0 || figure > 135 || selected[figure] {
			t.Fatal("invalid or duplicate batch figure")
		}
		selected[figure] = true
		figures = append(figures, figure)
	}
	valid, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	for _, figure := range figures {
		for _, resource := range []int{figure * 3, figure*3 + 1} {
			native, err := figani.LoadSeparatedResource(filepath.Join(os.Getenv("FD2_TEST_ORIGINAL_PACK"), "animations"), resource)
			if err != nil {
				t.Fatal(err)
			}
			bank := valid.battleAnimations[resource]
			if bank == nil || len(bank.Frames) != len(native.Frames) || bank.Frames[0] != nil {
				t.Fatalf("batch resource %d missing or eagerly retained", resource)
			}
		}
		if err := valid.prepareBattleHDResources(figure*3, figure*3+1); err != nil {
			t.Fatal(err)
		}
		for _, resource := range []int{figure * 3, figure*3 + 1} {
			bank := valid.battleAnimations[resource]
			if bank.Frames[0] == nil || bank.Frames[len(bank.Frames)-1] == nil {
				t.Fatalf("batch resource %d incomplete", resource)
			}
		}
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
		if asset.Role != "battle_animation_hd_set" || !selected[asset.SourceResource/3] {
			continue
		}
		asset.SourceMetadataSHA256 = strings.Repeat("0", 64)
		isolated := &modernStoryPortraitSet{}
		if err := loadModernBattleHDAsset(isolated, asset, pack); err == nil || len(isolated.battleAnimations) != 0 {
			t.Fatal("batch wrong-source resource accepted or published")
		}
	}
	cold, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	for _, figure := range figures {
		for _, owner := range []string{"player", "mode11"} {
			t.Run(strconv.Itoa(figure)+"/"+owner, func(t *testing.T) {
				idle, attack := figure*3, figure*3+1
				bank := cold.battleAnimations[idle]
				if bank == nil || cold.battleAnimations[attack] == nil {
					t.Fatal("batch cold bank missing")
				}
				originalPath := bank.paths[0]
				bank.paths[0] = filepath.Join(t.TempDir(), "missing-hd-frame.png")
				defer func() { bank.paths[0] = originalPath }()
				g, actor, target := physicalSceneTestGame(t)
				actor.BattleFig = figure
				g.modernStoryPortraits = cold
				beforeActor, beforeTarget := *actor, *target
				if owner == "player" {
					g.confirm()
				} else {
					g.aiBusy = true
					g.executeNativeAIMode11Physical(&battle.AIPlan{U: actor, Target: target}, nil)
				}
				if !strings.Contains(g.loadErr, "HD source preflight") || g.atk != nil || g.nativeRNGState != 17791 ||
					!reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) {
					t.Fatalf("batch failure did not stop before settlement: %s", g.loadErr)
				}
				if g.rng.Int63() != rand.New(rand.NewSource(73)).Int63() {
					t.Fatal("batch failure consumed Go RNG")
				}
				for _, other := range figures {
					for _, resource := range []int{other * 3, other*3 + 1} {
						for _, frame := range cold.battleAnimations[resource].Frames {
							if frame != nil {
								t.Fatal("batch failure published or reused a partial bank")
							}
						}
					}
				}
			})
		}
	}
	t.Logf("batch figures %v: complete native banks and wrong-source rejection; independent player/mode11 failures preserve HP/RNG and all cold banks", figures)
}

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

// 135只有405待機；缺最後一幀必須在結算前拒收，不請求406／407。
func TestModernHDRemainingFIGANIIdleTargetPreflight(t *testing.T) {
	if os.Getenv("FD2_HD_REMAINING_FIGANI_GPU") != "1" {
		t.Skip("dedicated remaining FIGANI idle-target preflight process required")
	}
	requirePhysicalScenePack(t)
	catalog, pack := os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK")
	valid, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	if valid.battleAnimations[405] == nil || len(valid.battleAnimations[405].Frames) != 4 ||
		valid.battleAnimations[406] != nil || valid.battleAnimations[407] != nil {
		t.Fatal("idle-only source contract differs")
	}
	g, actor, target := physicalSceneTestGame(t)
	actor.BattleFig, target.BattleFig = 76, 135
	g.modernStoryPortraits = valid
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil || scene == nil || scene.bodyResources.counter != nil {
		t.Fatalf("idle target must prepare without counter bank: %v", err)
	}
	for _, frame := range valid.battleAnimations[405].Frames {
		if frame == nil {
			t.Fatal("idle target preflight omitted a frame")
		}
	}
	cold, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	bank := cold.battleAnimations[405]
	bank.paths[len(bank.paths)-1] = filepath.Join(t.TempDir(), "missing-last-idle.png")
	for _, owner := range []string{"player", "mode11"} {
		t.Run(owner, func(t *testing.T) {
			g, actor, target := physicalSceneTestGame(t)
			actor.BattleFig, target.BattleFig = 76, 135
			g.modernStoryPortraits = cold
			a, b := *actor, *target
			if owner == "player" {
				g.confirm()
			} else {
				g.aiBusy = true
				g.executeNativeAIMode11Physical(&battle.AIPlan{U: actor, Target: target}, nil)
			}
			if !strings.Contains(g.loadErr, "HD source preflight") || g.atk != nil ||
				g.nativeRNGState != 17791 || !reflect.DeepEqual(a, *actor) || !reflect.DeepEqual(b, *target) ||
				g.rng.Int63() != rand.New(rand.NewSource(73)).Int63() {
				t.Fatalf("idle target failure changed settlement: %s", g.loadErr)
			}
			for _, resource := range []int{228, 229, 405} {
				for _, frame := range cold.battleAnimations[resource].Frames {
					if frame != nil {
						t.Fatal("idle target failure published a partial bank")
					}
				}
			}
		})
	}
}
