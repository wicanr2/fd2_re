package main

import (
	"encoding/json"
	"fmt"
	"image"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type splitStrikeCountedSource struct {
	rand.Source
	calls int
}

func (s *splitStrikeCountedSource) Int63() int64 {
	s.calls++
	return s.Source.Int63()
}

func splitStrikeHDTestGame(t *testing.T, command int) (*Game, *battle.Unit, []*battle.Unit) {
	t.Helper()
	var g *Game
	var actor *battle.Unit
	var targets []*battle.Unit
	if command == 24 {
		var target *battle.Unit
		g, actor, target = newNativeCommand24PresentationTestGame(t)
		targets = []*battle.Unit{target}
	} else if command == 29 {
		g, actor, targets = newNativeCommand29PresentationTestGame(t)
	} else {
		t.Fatal("unsupported split strike command")
	}
	// 提供F2所需的視覺context；單位、terrain及command仍沿原生fixture。
	g.baseMapTileset = image.NewRGBA(image.Rect(0, 0, 24, 24))
	g.m.TileW, g.m.TileH, g.currentMapID = 24, 24, -1
	if value := os.Getenv("FD2_HD_SPLIT_SCENERY"); value != "" {
		var indices []byte
		switch value {
		case "4:5:6":
			indices = []byte{4, 5, 6}
		case "5:6:4":
			indices = []byte{5, 6, 4}
		case "6:4:5":
			indices = []byte{6, 4, 5}
		default:
			t.Fatal("unsupported explicit command scenery", value)
		}
		side := os.Getenv("FD2_HD_SPLIT_SCENERY_SIDE")
		if side != "0" && side != "2" {
			t.Fatal("explicit scenery side0 or2 required")
		}
		g.m.NativeTerrainControl = nil
		for _, index := range indices {
			g.m.NativeTerrainControl = append(g.m.NativeTerrainControl, 0, 0, index, 0)
		}
		for i, unit := range append([]*battle.Unit{actor}, targets...) {
			g.m.Tiles[unit.Y*g.m.W+unit.X] = i
		}
		actor.NativeRecordByte6 = 0
		if side == "2" {
			actor.NativeRecordByte6 = 2
		}
		for _, target := range targets {
			target.NativeRecordByte6 = 1
			if side == "2" {
				target.NativeRecordByte6 = 0
			}
		}
	}
	return g, actor, targets
}

func startSplitStrikeHD(g *Game, command int, actor, target *battle.Unit, then func([]battle.NativeCommand24Damage)) error {
	if command == 24 {
		return g.startNativeCommand24Presentation(actor, target, then)
	}
	return g.startNativeCommand29Presentation(actor, target, then)
}

func TestModernHDSplitStrikePreflightAndF2Atomic(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit complete theme required")
	}
	template, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []int{24, 29} {
		t.Run(fmt.Sprint(command), func(t *testing.T) {
			g, actor, targets := splitStrikeHDTestGame(t, command)
			seed64 := int64(2)
			if command == 29 {
				seed64 = 29
			}
			rngSource := &splitStrikeCountedSource{Source: rand.NewSource(seed64)}
			g.rng = rand.New(rngSource)
			set := cloneModernHDBanksForTest(template)
			g.modernStoryPortraits = set
			a, seed := *actor, g.nativeRNGState
			before := make([]battle.Unit, len(targets))
			for i, target := range targets {
				before[i] = *target
			}
			stable := func() bool {
				if !reflect.DeepEqual(a, *actor) || seed != g.nativeRNGState {
					return false
				}
				for i, target := range targets {
					if !reflect.DeepEqual(before[i], *target) {
						return false
					}
				}
				return true
			}
			resources := []int{actor.BattleFig * 3, actor.BattleFig*3 + 2}
			for _, target := range targets {
				resources = append(resources, target.BattleFig*3)
			}
			empty := func(s *modernStoryPortraitSet) bool {
				for _, resource := range resources {
					for _, frame := range s.battleAnimations[resource].Frames {
						if frame != nil {
							return false
						}
					}
				}
				return true
			}
			// 最後目標的最後幀也必須在發布owner前驗完。
			entry := set.battleAnimations[resources[len(resources)-1]]
			last, saved := len(entry.paths)-1, entry.paths[len(entry.paths)-1]
			entry.paths[last] = filepath.Join(t.TempDir(), "missing-last-target.png")
			if err := startSplitStrikeHD(g, command, actor, targets[0], nil); err == nil ||
				!strings.Contains(err.Error(), "missing-last-target.png") || !stable() || !empty(set) ||
				g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil {
				t.Fatalf("failed preflight partially published: %v", err)
			}
			if rngSource.calls != 0 {
				t.Fatal("failed HD lookup consumed planning RNG")
			}
			entry.paths[last] = saved
			if err := startSplitStrikeHD(g, command, actor, targets[0], nil); err != nil {
				t.Fatal(err)
			}
			planningCalls := rngSource.calls
			var owner any = g.nativeCmd24Presentation
			if command == 29 {
				owner = g.nativeCmd29Presentation
			}
			if err := g.applyTheme(nil); err != nil {
				t.Fatal(err)
			}
			cold := cloneModernHDBanksForTest(template)
			bad := cold.battleAnimations[actor.BattleFig*3+2]
			bad.paths[len(bad.paths)-1] = filepath.Join(t.TempDir(), "missing-cast-f2.png")
			if err := g.applyTheme(cold); err == nil || g.modernStoryPortraits != nil || !stable() || !empty(cold) {
				t.Fatal("failed F2 changed theme, units, RNG or a partial HD bank")
			}
			if command == 24 && g.nativeCmd24Presentation != owner || command == 29 && g.nativeCmd29Presentation != owner {
				t.Fatal("failed F2 replaced the active owner")
			}
			if err := g.applyTheme(set); err != nil {
				t.Fatal(err)
			}
			if command == 24 {
				g.stepNativeCommand24Presentation()
				if g.nativeCmd24Presentation.preludeFrame != 0 {
					t.Fatal("owner advanced without Draw")
				}
			} else {
				g.stepNativeCommand29Presentation()
				if g.nativeCmd29Presentation.preludeFrame != 0 {
					t.Fatal("owner advanced without Draw")
				}
			}
			expected := rand.New(rand.NewSource(seed64))
			for range targets {
				expected.Intn(100)
			}
			if !stable() || rngSource.calls != planningCalls || g.rng.Int63() != expected.Int63() {
				t.Fatal("unacknowledged owner changed units or RNG")
			}
		})
	}
}

func TestModernHDSplitStrikeLateFailureRestoresTransaction(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit complete theme required")
	}
	g, actor, targets := splitStrikeHDTestGame(t, 29)
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	g.modernStoryPortraits = set
	rng := &splitStrikeCountedSource{Source: rand.NewSource(29)}
	g.rng = rand.New(rng)
	beforeActor, beforeRNG := *actor, g.nativeRNGState
	beforeTargets := []battle.Unit{*targets[0], *targets[1]}
	callbacks := 0
	if err := startSplitStrikeHD(g, 29, actor, targets[0], func([]battle.NativeCommand24Damage) { callbacks++ }); err != nil {
		t.Fatal(err)
	}
	planningCalls := rng.calls
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	defer func() {
		if g.modernHDBattleGPU != nil {
			g.modernHDBattleGPU.close()
		}
	}()
	for steps := 0; g.nativeCmd29Presentation != nil && !g.nativeCmd29Presentation.targets[0].damagePublished && steps < 600; steps++ {
		if !g.drawNativeCommand29Presentation(screen) || !g.modernHDBattleVisible {
			t.Fatal("HD render request failed before first impact")
		}
		g.stepNativeCommand29Presentation()
	}
	if g.nativeCmd29Presentation == nil || !g.nativeCmd29Presentation.targets[0].damagePublished {
		t.Fatal("first target marker was not reached")
	}
	targets[1].HP-- // 沿既有原生反例，破壞下一目標的交易前置條件。
	for steps := 0; g.nativeCmd29Presentation != nil && steps < 600; steps++ {
		g.drawNativeCommand29Presentation(screen)
		g.stepNativeCommand29Presentation()
	}
	if g.nativeCmd29Presentation != nil || g.loadErr == "" || callbacks != 0 ||
		!reflect.DeepEqual(beforeActor, *actor) || !reflect.DeepEqual(beforeTargets[0], *targets[0]) ||
		!reflect.DeepEqual(beforeTargets[1], *targets[1]) || rng.calls != planningCalls || g.nativeRNGState != beforeRNG {
		t.Fatal("late HD failure did not restore the original transaction")
	}
	g.syncModernHDBattleOwner()
	if g.modernHDBattleGPU.owner != nil || len(g.modernHDBattleGPU.sources) != 0 {
		t.Fatal("failed owner retained GPU sources")
	}
}

// 獨立按mask裁切的8位元參照，避免拿HD配方自行生成期望位置。
func splitStrikeNativeBlit(dst []byte, frame figani.Frame, dx int) {
	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			px, py := frame.X+dx+x, frame.Y+y
			if px >= 0 && px < 320 && py >= 0 && py < 200 && frame.Mask[y*frame.Width+x] != 0 {
				dst[py*320+px] = frame.Pixels[y*frame.Width+x]
			}
		}
	}
}

func splitStrikeNativeGPUCases(t *testing.T) []modernHDBattleGPUCase {
	t.Helper()
	dac, palette, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	var bg [3]fdother.Frame
	for i := range bg {
		bg[i], err = fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", i)
		if err != nil {
			t.Fatal(err)
		}
	}
	platform, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", 0)
	if err != nil {
		t.Fatal(err)
	}
	var cases []modernHDBattleGPUCase
	g := &Game{}
	add := func(name string, f nativeCompoundPresentedFrame, indexed []byte) {
		plan, err := g.modernHDBattlePlan(f.hdSources, f.hdPalette, f.hdSubtract)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, modernHDBattleGPUCase{name: name, plan: plan, indexed: indexed})
	}
	for _, figure := range []int{32, 34} {
		idle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), figure*3)
		if err != nil {
			t.Fatal(err)
		}
		effect, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), figure*3+2)
		if err != nil {
			t.Fatal(err)
		}
		command := 24
		if figure == 34 {
			command = 29
		}
		schedule, err := figani.BuildNativeCommandDerivedStrikeSchedule(effect, command)
		if err != nil {
			t.Fatal(err)
		}
		for _, side := range []byte{0, 2} {
			base, after := make([]byte, 64000), make([]byte, 64000)
			for i := range base {
				base[i], after[i] = byte(i%251), byte((i+19)%251)
			}
			in := battlepresent.NativeCommandPreludeInput{Base: base, ActorIdle: idle.Frames[0], Platform: &platform, RawSide: side, Mode: 1, BaselineDAC: dac}
			prelude, err := battlepresent.BuildNativeCommandPreludeFrames(in)
			if err != nil {
				t.Fatal(err)
			}
			hd, err := bindModernHDSplitStrike(in, prelude, figure*3, effect, figure*3+2, schedule.ActorImpactFrame,
				schedule.TargetStart, schedule.TargetImpactFrame, base, after, palette)
			if err != nil {
				t.Fatal(err)
			}
			for i, f := range prelude {
				add(fmt.Sprintf("%d/%d/prelude/%d", command, side, i), hd.prelude[i], f.Pixels)
			}
			source := append([]byte(nil), after...)
			splitStrikeNativeBlit(source, effect.Frames[schedule.TargetStart-1], 0)
			sourceHD := hd.actorEndSources()
			targetIDs := []int{0}
			if command == 29 {
				targetIDs = append(targetIDs, 9)
			}
			for index, resource := range targetIDs {
				target, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), resource)
				if err != nil {
					t.Fatal(err)
				}
				a, b := make([]byte, 64000), make([]byte, 64000)
				for i := range a {
					a[i], b[i] = byte((i+37+index)%251), byte((i+79+index)%251)
				}
				presents, err := battlepresent.BuildNativeCommand24BackgroundPresents(battlepresent.NativeCommand24BackgroundInputs{Layers: bg, Source: source, Target: a, TargetIdle: target.Frames[0]})
				if err != nil {
					t.Fatal(err)
				}
				if err := hd.addTarget(a, b, target, resource, sourceHD, presents, bg); err != nil {
					t.Fatal(err)
				}
				for i, p := range presents {
					add(fmt.Sprintf("%d/%d/target%d/scroll/%d", command, side, index, i), hd.targets[index].transitions[i], p.Pixels)
				}
				begin := 0
				if index > 0 {
					begin = schedule.TargetStart
				}
				for f := begin; f < len(effect.Frames); f++ {
					shakes := []int{0}
					if f >= schedule.TargetStart {
						shakes = []int{0, 4, 9, 14, 18}
					}
					for _, shake := range shakes {
						which := f % len(target.Frames)
						background := base
						if f >= schedule.ActorImpactFrame {
							background = after
						}
						if f >= schedule.TargetStart {
							background = a
							if f >= schedule.TargetImpactFrame {
								background = b
							}
						}
						want := append([]byte(nil), background...)
						if f >= schedule.TargetStart {
							splitStrikeNativeBlit(want, target.Frames[which], -shake)
						}
						splitStrikeNativeBlit(want, effect.Frames[f], 0)
						add(fmt.Sprintf("%d/%d/target%d/frame%d/shake%d", command, side, index, f, shake),
							*hd.frame(9, -1, index, f, which, f >= schedule.TargetImpactFrame, shake), want)
					}
				}
				source = append([]byte(nil), b...)
				splitStrikeNativeBlit(source, target.Frames[len(target.Frames)-1], 0)
				splitStrikeNativeBlit(source, effect.Frames[len(effect.Frames)-1], 0)
				sourceHD = hd.targetEndSources(index, len(target.Frames)-1)
			}
		}
	}
	return cases
}

func TestModernHDSplitStrikeNativeSourceGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SPLIT_STRIKE_GPU") != "1" {
		t.Skip("dedicated split strike GPU process required")
	}
	requirePhysicalScenePack(t)
	cases := splitStrikeNativeGPUCases(t)
	probe := &modernHDBattleGPUProbe{cases: cases}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	defer probe.gpu.close()
	if probe.err != nil || probe.index != len(cases) {
		t.Fatalf("split strike source GPU: %v (%d/%d)", probe.err, probe.index, len(cases))
	}
	if out := os.Getenv("FD2_TEST_MODERN_EFFECT_OUT"); out != "" {
		if err := os.Mkdir(out, 0755); err != nil {
			t.Fatal(err)
		}
		names := make([]string, len(cases))
		for i, c := range cases {
			names[i] = c.name
		}
		data, _ := json.MarshalIndent(map[string]any{"status": "PASS-SPLIT-STRIKE-NATIVE-SOURCE-GPU", "cases": names,
			"scope": "existing indexed producer versus complete native GPU; both sides; no original oracle or HD-art claim"}, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d full native GPU cases include both sides, all split frames, BG presents and shake offsets", len(cases))
}

func TestModernHDSplitStrikeSceneryNativeAndAtomic(t *testing.T) {
	requirePhysicalScenePack(t)
	set := modernHDBattleFixtureSet(t, os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"), nil)
	panelAssets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	canvases, rejections := 0, 0
	for _, triple := range []string{"4:5:6", "5:6:4", "6:4:5"} {
		t.Setenv("FD2_HD_SPLIT_SCENERY", triple)
		for _, side := range []string{"0", "2"} {
			t.Setenv("FD2_HD_SPLIT_SCENERY_SIDE", side)
			for _, command := range []int{24, 29} {
				g, actor, targets := splitStrikeHDTestGame(t, command)
				rng := &splitStrikeCountedSource{Source: rand.NewSource(int64(command))}
				g.rng = rand.New(rng)
				g.modernStoryPortraits = set
				if err := startSplitStrikeHD(g, command, actor, targets[0], nil); err != nil {
					t.Fatal(err)
				}
				h := g.nativeCmd24Presentation
				var hd *modernHDSplitStrike
				var plan *battle.NativeCommandDerivedStrikePlan
				if command == 24 {
					hd, plan = h.hd, h.plan
				} else {
					hd, plan = g.nativeCmd29Presentation.hd, g.nativeCmd29Presentation.plan
				}
				check := func(label string, s *modernHDBattleSources, want []byte) {
					t.Helper()
					if s == nil || !reflect.DeepEqual(surfaceIndexedReference(s), want) {
						t.Fatalf("command%d %s side%s %s native canvas differs", command, triple, side, label)
					}
					for _, source := range append(append([]modernHDBattleSource(nil), s.BaseScenery...), s.Layers...) {
						if source.Archive == "BG.DAT" || source.Archive == "TAI.DAT" {
							im, _, err := set.battleSurfaceHD(source)
							if err != nil || im == nil {
								t.Fatal("lost pinned scenery source", err)
							}
						}
					}
					canvases++
				}
				base := func(unit *battle.Unit, mp, hp int, platform *fdother.Frame) []byte {
					t.Helper()
					selector, err := nativeCommand24BGSelector(g.m, unit)
					if err != nil {
						t.Fatal(err)
					}
					bg, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", selector)
					if err != nil {
						t.Fatal(err)
					}
					copyUnit := *unit
					copyUnit.MP, copyUnit.HP = mp, hp
					record, err := battle.NativeBattlePanelRecordForUnit(&copyUnit)
					if err != nil {
						t.Fatal(err)
					}
					index, err := nativeCommand24RuntimeUnitIndex(g.st, unit)
					if err != nil {
						t.Fatal(err)
					}
					out, err := g.localizedNativeCommand24BackgroundBase(bg, panelAssets, record, index, g.handlerChapter, platform)
					if err != nil {
						t.Fatal(err)
					}
					return out
				}
				var platform *fdother.Frame
				actorSelector, _ := nativeCommand24BGSelector(g.m, actor)
				if side == "2" {
					f, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", actorSelector)
					if err != nil {
						t.Fatal(err)
					}
					platform = &f
				}
				before, after := base(actor, actor.MP, actor.HP, platform), base(actor, plan.MPAfter, actor.HP, platform)
				idle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3)
				if err != nil {
					t.Fatal(err)
				}
				dac, _, err := loadNativeBattlePalette()
				if err != nil {
					t.Fatal(err)
				}
				prelude, err := battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{Base: base(actor, actor.MP, actor.HP, nil), ActorIdle: idle.Frames[0], Platform: platform, RawSide: actor.NativeRecordByte6, Mode: 1, BaselineDAC: dac})
				if err != nil {
					t.Fatal(err)
				}
				for i, f := range prelude {
					check("prelude", hd.prelude[i].hdSources, f.Pixels)
				}
				for f := 0; f < hd.targetStart; f++ {
					want := append([]byte(nil), before...)
					if f >= hd.actorImpact {
						copy(want, after)
					}
					splitStrikeNativeBlit(want, hd.effect.Frames[f], 0)
					check("actor", hd.frame(9, -1, 0, f, 0, false, 0).hdSources, want)
				}
				source := append([]byte(nil), after...)
				splitStrikeNativeBlit(source, hd.effect.Frames[hd.targetStart-1], 0)
				check("actor-end", hd.actorEndSources(), source)
				var bg [3]fdother.Frame
				for i := range bg {
					bg[i], err = fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", i)
					if err != nil {
						t.Fatal(err)
					}
				}
				for i, target := range targets {
					a, b := base(target, target.MP, target.HP, nil), base(target, target.MP, plan.Results[i].HPAfter, nil)
					ti := hd.targets[i]
					presents, err := battlepresent.BuildNativeCommand24BackgroundPresents(battlepresent.NativeCommand24BackgroundInputs{Layers: bg, Source: source, Target: a, TargetIdle: ti.idle.Frames[0]})
					if err != nil {
						t.Fatal(err)
					}
					for n, p := range presents {
						check("transition", ti.transitions[n].hdSources, p.Pixels)
					}
					for f := hd.targetStart; f < len(hd.effect.Frames); f++ {
						for idleIndex := range ti.idle.Frames {
							for _, shake := range []int{0, 4, 9, 14, 18} {
								want := append([]byte(nil), a...)
								if f >= hd.impact {
									copy(want, b)
								}
								splitStrikeNativeBlit(want, ti.idle.Frames[idleIndex], -shake)
								splitStrikeNativeBlit(want, hd.effect.Frames[f], 0)
								check("target", hd.frame(9, -1, i, f, idleIndex, f >= hd.impact, shake).hdSources, want)
							}
						}
					}
					last := len(ti.idle.Frames) - 1
					if command == 29 {
						last, err = nativeCommand29FinalIdleFrame(idleDelaysForScenery(ti.idle), g.nativeCmd29Presentation.schedule)
						if err != nil {
							t.Fatal(err)
						}
					}
					source = append([]byte(nil), b...)
					splitStrikeNativeBlit(source, ti.idle.Frames[last], 0)
					splitStrikeNativeBlit(source, hd.effect.Frames[len(hd.effect.Frames)-1], 0)
					check("target-end", hd.targetEndSources(i, last), source)
				}
				wanted := []struct {
					archive string
					index   int
				}{{"BG.DAT", actorSelector}}
				for _, target := range targets {
					index, _ := nativeCommand24BGSelector(g.m, target)
					wanted = append(wanted, struct {
						archive string
						index   int
					}{"BG.DAT", index})
				}
				if side == "2" {
					wanted = append(wanted, struct {
						archive string
						index   int
					}{"TAI.DAT", actorSelector})
				}
				for _, source := range wanted {
					bad := *set
					entry := *set.battleSurfaces[source.archive][source.index]
					entry.picture = nil
					bad.battleSurfaces = map[string]map[int]*modernBattleSurfaceHD{source.archive: {source.index: &entry}}
					state := func() string {
						units, _ := json.Marshal(g.st.Units)
						return fmt.Sprintf("%s/%d/%d", units, g.nativeRNGState, rng.calls)
					}
					beforeState := state()
					if err := g.applyTheme(&bad); err == nil || g.modernStoryPortraits != set || state() != beforeState {
						t.Fatal("bad command scenery F2 changed owner/state", source)
					}
					fresh, a, ts := splitStrikeHDTestGame(t, command)
					counter := &splitStrikeCountedSource{Source: rand.NewSource(int64(command))}
					fresh.rng = rand.New(counter)
					fresh.modernStoryPortraits = &bad
					units, _ := json.Marshal(fresh.st.Units)
					if err := startSplitStrikeHD(fresh, command, a, ts[0], nil); err == nil || counter.calls != 0 || fresh.nativeCmd24Presentation != nil || fresh.nativeCmd29Presentation != nil {
						t.Fatal("bad scenery reached planning or owner", source)
					}
					afterUnits, _ := json.Marshal(fresh.st.Units)
					if string(units) != string(afterUnits) {
						t.Fatal("bad scenery changed units")
					}
					rejections++
				}
			}
		}
	}
	t.Logf("%d complete native canvases, %d early RNG/owner and %d live F2 scenery rejections; three triples, both sides, command24/29", canvases, rejections, rejections)
}

func idleDelaysForScenery(animation *figani.Animation) []int {
	delays := make([]int, len(animation.Frames))
	for i, f := range animation.Frames {
		delays[i] = f.Delay
	}
	return delays
}

type splitSceneryOnlyCase struct {
	g      *Game
	frame  nativeCompoundPresentedFrame
	wanted bool
	want   *image.RGBA
}

type splitSceneryOnlyProbe struct {
	cases []splitSceneryOnlyCase
	index int
	err   error
}

func (p *splitSceneryOnlyProbe) Update() error {
	if p.err != nil || p.index == len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *splitSceneryOnlyProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *splitSceneryOnlyProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= len(p.cases) {
		return
	}
	c := &p.cases[p.index]
	c.g.modernHDBattleVisible = false
	ok, err := c.g.drawModernHDCompoundFrame(screen, &c.frame)
	if err != nil || ok != c.wanted || c.g.modernHDBattleVisible != c.wanted {
		p.err = fmt.Errorf("surface-only draw %d wanted%v got%v: %v", p.index, c.wanted, ok, err)
		return
	}
	if c.wanted {
		pixels := make([]byte, len(c.want.Pix))
		c.g.modernHDBattleGPU.World.ReadPixels(pixels)
		for i, w := range c.want.Pix {
			delta := int(pixels[i]) - int(w)
			if delta < -1 || delta > 1 {
				p.err = fmt.Errorf("surface-only%d RGBA byte%d got%d want%d", p.index, i, pixels[i], w)
				return
			}
		}
	}
	p.index++
}

func TestModernHDSplitStrikeSceneryOnlyGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SPLIT_STRIKE_GPU") != "1" {
		t.Skip("dedicated scenery-only GPU process required")
	}
	requirePhysicalScenePack(t)
	set := modernHDBattleFixtureSet(t, os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"), nil)
	var cases []splitSceneryOnlyCase
	for _, triple := range []string{"4:5:6", "5:6:4", "6:4:5"} {
		t.Setenv("FD2_HD_SPLIT_SCENERY", triple)
		for _, side := range []string{"0", "2"} {
			t.Setenv("FD2_HD_SPLIT_SCENERY_SIDE", side)
			for _, command := range []int{24, 29} {
				g, actor, targets := splitStrikeHDTestGame(t, command)
				g.modernStoryPortraits = set
				if err := startSplitStrikeHD(g, command, actor, targets[0], nil); err != nil {
					t.Fatal(err)
				}
				var hd *modernHDSplitStrike
				if command == 24 {
					hd = g.nativeCmd24Presentation.hd
				} else {
					hd = g.nativeCmd29Presentation.hd
				}
				for _, archive := range []string{"BG.DAT", "TAI.DAT"} {
					only := &modernStoryPortraitSet{hasHD: true, battleSurfaces: map[string]map[int]*modernBattleSurfaceHD{archive: set.battleSurfaces[archive]}}
					frames := []nativeCompoundPresentedFrame{hd.prelude[4], *hd.frame(9, -1, 0, 0, 0, false, 0), *hd.frame(9, -1, 0, hd.targetStart, 0, false, 0)}
					for i, frame := range frames {
						copyGame := *g
						copyGame.modernStoryPortraits = only
						wanted := archive == "BG.DAT" || side == "2" && i < 2
						c := splitSceneryOnlyCase{g: &copyGame, frame: frame, wanted: wanted}
						if wanted {
							plan, err := copyGame.modernHDBattlePlan(frame.hdSources, frame.hdPalette, frame.hdSubtract)
							if err != nil {
								t.Fatal(err)
							}
							c.want = surfaceCPUWorld(plan)
						}
						cases = append(cases, c)
					}
				}
			}
		}
	}
	p := &splitSceneryOnlyProbe{cases: cases}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, c := range cases {
			if c.g.modernHDBattleGPU != nil {
				c.g.modernHDBattleGPU.close()
			}
		}
	}()
	if p.err != nil || p.index != len(cases) {
		t.Fatalf("surface-only GPU %d/%d: %v", p.index, len(cases), p.err)
	}
	t.Logf("%d command24/29 BG-only/TAI-only activation and native-fallback cases; 48 actual GPU worlds against independent CPU Over tolerance1", len(cases))
}
