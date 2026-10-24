package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type itemHDCase struct {
	name                  string
	item, frames, publish int
	restore               bool
}

var itemHDCases = []itemHDCase{
	{"restore5", 192, 40, 18, true}, {"restore13", 40, 40, 18, true},
	{"damage20", 11, 40, 18, false}, {"damage20-command0", 56, 40, 18, false},
	{"damage24", 79, 40, 18, false}, {"type21-command1", 38, 38, 16, false},
	{"type21-command6", 29, 40, 18, false}, {"type21-command7", 51, 40, 18, false},
}

// These tests share immutable decoded art; every game and transaction is new.
var itemHDTestTemplates = map[string]*modernStoryPortraitSet{}

func itemHDSet(t *testing.T) *modernStoryPortraitSet {
	t.Helper()
	catalog, pack := os.Getenv("FD2_TEST_UNIT_PRESENT_CATALOG"), os.Getenv("FD2_TEST_UNIT_PRESENT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit full HD catalog and pack required")
	}
	key := catalog + "\x00" + pack
	if set := itemHDTestTemplates[key]; set != nil {
		return set
	}
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	itemHDTestTemplates[key] = set
	return set
}

// Keep the full roster and use the real player and AI entry points. The fixed
// seed and high HP isolate presentation without clearing the battle early.
func itemHDGame(t *testing.T, c itemHDCase, player bool) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	t.Setenv("FD2_MUTE", "1")
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	actor, target := g.st.Units[4], g.st.Units[0]
	if player {
		actor, target = g.st.Units[0], g.st.Units[4]
	}
	if c.restore {
		if player {
			target = g.st.Units[1]
		} else {
			target = g.st.Units[5]
		}
	}
	actor.SetMapPlacement(4, 3, 3)
	target.SetMapPlacement(5, 3, 3)
	for _, u := range []*battle.Unit{actor, target} {
		if err := u.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
		u.NativeRecordByte5 = 0
		u.HasNativeRecordByte5 = true
		u.HasNativeRecordByte6 = true
		u.NativeRecordByte6 = 0
		if u.Camp == battle.Enemy {
			u.NativeRecordByte6 = 1
		}
		u.Acted = false
		u.HP = 1000
		u.MaxHP = 1000
	}
	if c.restore {
		target.HP = 20
	}
	actor.Inventory = []int{c.item}
	actor.Equipped = []bool{false}
	actor.InventorySlots = []int{c.item, 255, 255, 255, 255, 255, 255, 255}
	actor.NativeInventoryFlags = []int{0, 128, 128, 128, 128, 128, 128, 128}
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	g.nativeCommandBook, err = battle.LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	g.nativeCommandResistances, err = battle.LoadNativeCommandResistances("../../assets/data/native_command_resistances.json")
	if err != nil {
		t.Fatal(err)
	}
	g.st.NativeCommandBook = g.nativeCommandBook
	g.st.NativeCommandResistances = g.nativeCommandResistances
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 5, CursorY: 3, VisibleCursorX: 5, VisibleCursorY: 3}); err != nil {
		t.Fatal(err)
	}
	g.curX, g.curY = 5, 3
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	g.nativeRNGState = 7
	return g, actor, target
}

func startItemHD(t *testing.T, g *Game, a, b *battle.Unit, c itemHDCase, player bool, done *int) {
	t.Helper()
	if player {
		g.sel, g.moved, g.itemOpen = a, true, true
		if ok, err := g.beginNativeTargetItem(0, c.item); err != nil || !ok {
			t.Fatalf("player targeting: %v %v", ok, err)
		}
		if ok, err := g.applyNativeTargetItem(b); err != nil || !ok {
			t.Fatalf("player use: %v %v", ok, err)
		}
	} else {
		index := -1
		for i, u := range g.st.Units {
			if u == b {
				index = i
			}
		}
		plan := &battle.AIPlan{U: a, Target: b, NativeActionKind: battle.NativeAIActionItem, NativeItemSlot: 0, NativeItemID: c.item, NativeItemTargetIndices: []byte{byte(index)}}
		if err := g.executeNativeAIActionWithContinuation(plan, func() {}); err != nil {
			t.Fatal(err)
		}
	}
	j := g.nativeAIItemPresentation
	if j == nil || len(j.frames) != c.frames || j.publishAt != c.publish {
		t.Fatalf("wrong schedule for %s", c.name)
	}
	then := j.then
	j.then = func() {
		*done++
		if then != nil {
			then()
		}
	}
}

func itemHDState(g *Game) any {
	var owner any
	if j := g.nativeAIItemPresentation; j != nil {
		owner = []any{j.frame, j.repeat, j.holding, j.hold, j.published, j.drawn}
	}
	return []any{compoundMapUnitSnapshot(g), g.nativeRNGState, append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...), append([]byte(nil), g.nativeMapDAC...), owner, g.nativeItemTargeting}
}

func TestNativeMissDescriptorConsumers(t *testing.T) {
	// Exercise real preflight producers. Raw transient flags select their
	// existing no-application paths; the official seed stays fixed at seven.
	check := func(t *testing.T, frames []nativeCompoundPresentedFrame) {
		t.Helper()
		seen := map[int]bool{}
		for _, f := range frames {
			if f.hdMap == nil {
				continue
			}
			for _, l := range f.hdMap.layers {
				if l.Resource != 5 {
					continue
				}
				if l.Entry >= 74 && l.Entry <= 76 {
					t.Fatal("decimal smoke descriptor in MISS producer", l)
				}
				seen[l.Entry] = true
			}
		}
		for _, descriptor := range []int{116, 117, 118} {
			if !seen[descriptor] {
				t.Fatal("MISS letter missing from real producer", descriptor, seen)
			}
		}
	}
	for _, c := range modifierHDCases() {
		if !c.Failed {
			continue
		}
		t.Run(fmt.Sprintf("modifier-%d", c.ID), func(t *testing.T) {
			g, _, _ := modifierHDTestGame(t, c)
			done := 0
			startModifierHDTest(t, g, c, &done)
			check(t, g.modernCompoundMapFrames())
		})
	}
	t.Run("summon34", func(t *testing.T) {
		g, a, b := nativeCommand34PresentationFixture(t)
		for _, u := range []*battle.Unit{a, b} {
			u.NativeTransient = [6]byte{3, 3, 3, 3, 3, 3}
		}
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		if err := g.startNativeCommand34Presentation(a, b, nil); err != nil {
			t.Fatal(err)
		}
		check(t, g.nativeCmd34Presentation.frames)
	})
	t.Run("summon35", func(t *testing.T) {
		g, a, b := nativeCommand35PresentationFixture(t)
		for _, u := range []*battle.Unit{a, b} {
			u.NativeTransient = [6]byte{3, 3, 3, 3, 3, 3}
		}
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		if err := g.startNativeCommand35Presentation(a, b, nil); err != nil {
			t.Fatal(err)
		}
		check(t, g.nativeCmd35Presentation.frames)
	})
}

func TestModernHDItemMapNativeControl(t *testing.T) {
	set := itemHDSet(t)
	for _, c := range itemHDCases {
		for _, player := range []bool{false, true} {
			if c.restore && player {
				continue
			} // Player restore currently settles directly.
			t.Run(fmt.Sprintf("%s/player-%v", c.name, player), func(t *testing.T) {
				g, a, b := itemHDGame(t, c, player)
				control, ca, cb := itemHDGame(t, c, player)
				g.modernStoryPortraits = set
				done, cdone := 0, 0
				startItemHD(t, g, a, b, c, player, &done)
				startItemHD(t, control, ca, cb, c, player, &cdone)
				before := itemHDState(g)
				g.stepNativeAIItemPresentation()
				if !reflect.DeepEqual(before, itemHDState(g)) {
					t.Fatal("advanced without Draw")
				}
				phases := map[string]bool{}
				layers := 0
				for _, f := range g.nativeAIItemPresentation.frames {
					if f.hdMap == nil || f.hdMap.prepared == nil || f.hdMap.theme != set {
						t.Fatal("HD frame not bound")
					}
					phases[f.hdPhase] = true
					for _, l := range f.hdMap.layers {
						bank := set.mapEffects
						if l.Resource == 5 {
							if l.Entry >= 74 && l.Entry <= 76 {
								t.Fatal("MISS queue used decimal smoke descriptors", l)
							}
							bank = set.mapOverlays
						}
						if bank == nil || bank.frames[l.Entry] == nil {
							t.Fatal("unexpected native LMI fallback", l)
						}
						layers++
					}
				}
				if len(phases) != 3 || layers == 0 {
					t.Fatal("incomplete HD phases", phases, layers)
				}
				updates := 0
				for g.nativeAIItemPresentation != nil && updates < 1000 {
					if !reflect.DeepEqual(itemHDState(g), itemHDState(control)) {
						t.Fatal("native state differs", updates)
					}
					j, cj := g.nativeAIItemPresentation, control.nativeAIItemPresentation
					if !j.holding {
						j.drawn = true
						cj.drawn = true
					}
					g.stepNativeAIItemPresentation()
					control.stepNativeAIItemPresentation()
					updates++
				}
				if g.nativeAIItemPresentation != nil || control.nativeAIItemPresentation != nil || done != 1 || cdone != 1 || !reflect.DeepEqual(itemHDState(g), itemHDState(control)) {
					t.Fatal("incomplete or differing continuation", updates, done, cdone, g.loadErr)
				}
				if player && (!a.Acted || g.sel != nil || g.nativeItemTargeting) {
					t.Fatal("player action did not return")
				}
				if c.item == 192 {
					if a.InventorySlots[0] != 255 {
						t.Fatal("type5 not consumed")
					}
				} else if a.InventorySlots[0] != c.item {
					t.Fatal("retained item consumed")
				}
			})
		}
	}
}

func TestModernHDPlayerRestoreKeepsNativeRoute(t *testing.T) {
	set := itemHDSet(t)
	for _, c := range itemHDCases[:2] {
		t.Run(c.name, func(t *testing.T) {
			g, a, b := itemHDGame(t, c, true)
			control, ca, cb := itemHDGame(t, c, true)
			g.modernStoryPortraits = set
			for i, game := range []*Game{g, control} {
				actor, target := a, b
				if i == 1 {
					actor, target = ca, cb
				}
				game.sel, game.moved, game.itemOpen = actor, true, true
				if ok, err := game.beginNativeTargetItem(0, c.item); !ok || err != nil {
					t.Fatal(ok, err)
				}
				if ok, err := game.applyNativeTargetItem(target); !ok || err != nil {
					t.Fatal(ok, err)
				}
			}
			if g.nativeAIItemPresentation != nil || control.nativeAIItemPresentation != nil || b.HP <= 20 || !a.Acted || g.nativeItemTargeting || !reflect.DeepEqual(itemHDState(g), itemHDState(control)) {
				t.Fatal("player immediate restore route changed")
			}
		})
	}
}

func TestModernHDItemMapRejectsAndF2(t *testing.T) {
	set := itemHDSet(t)
	c := itemHDCases[2]
	g, a, b := itemHDGame(t, c, false)
	g.modernStoryPortraits = set
	done := 0
	startItemHD(t, g, a, b, c, false, &done)
	j := g.nativeAIItemPresentation
	last := j.frames[len(j.frames)-1].hdMap
	state := itemHDState(g)
	active := last.prepared
	// Cached theme must revalidate even a future source before publication.
	last.pixels[0] ^= 1
	if err := g.applyTheme(set); err == nil || last.prepared != active || g.modernStoryPortraits != set || !reflect.DeepEqual(state, itemHDState(g)) {
		t.Fatal("edited future source accepted")
	}
	last.pixels[0] ^= 1
	bad := *set
	bank := *set.mapEffects
	bad.mapEffects = &bank
	bank.frames = map[int]*image.RGBA{}
	for k, v := range set.mapEffects.frames {
		bank.frames[k] = v
	}
	delete(bank.frames, 56)
	if err := g.applyTheme(&bad); err == nil || last.prepared != active || !reflect.DeepEqual(state, itemHDState(g)) {
		t.Fatal("missing declared last flash accepted")
	}
	bank.frames[56] = image.NewRGBA(image.Rect(0, 0, 1, 1))
	if err := g.applyTheme(&bad); err == nil || last.prepared != active || !reflect.DeepEqual(state, itemHDState(g)) {
		t.Fatal("wrong-sized declared flash accepted")
	}
	partial := *set
	partial.mapSpritesHD = map[int][]image.Image{}
	for k, v := range set.mapSpritesHD {
		partial.mapSpritesHD[k] = v
	}
	for k := range partial.mapSpritesHD {
		delete(partial.mapSpritesHD, k)
		break
	}
	if err := g.applyTheme(&partial); err == nil || last.prepared != active || !reflect.DeepEqual(state, itemHDState(g)) {
		t.Fatal("partial HD roster accepted")
	}
	for _, frame := range []int{0, 8, c.publish, c.frames - 1} {
		j.frame = frame
		state = itemHDState(g)
		if err := g.applyTheme(nil); err != nil {
			t.Fatal(err)
		}
		if err := g.applyTheme(set); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(state, itemHDState(g)) {
			t.Fatal("F2 mutated item owner")
		}
	}
	before := j.frames[0].hdMap.scene
	before.work[0] ^= 1
	if _, err := g.stageModernHDCompoundMap(j.frames, set); err == nil {
		t.Fatal("edited frozen work accepted")
	}
	before.work[0] ^= 1
}

func TestModernHDCompoundActualStage(t *testing.T) {
	now := time.Unix(1000, 0)
	g := stage42ActualGame(t, &now)
	set := itemHDSet(t)
	before := stage42NativeState(g)
	s := g.newModernHDCompoundMapScene(*g.nativeMapFrameSource, g.nativeMapWork, g.nativeMapVGA, g.nativeMapAssets.Palette, nil)
	if s.err != nil || s.stage == nil {
		t.Fatal("actual map23 source unavailable", s.err)
	}
	frames := make([]nativeCompoundPresentedFrame, 1)
	if err := bindModernHDCompoundMap(frames, s, [][]byte{g.nativeMapVGA}, nil, nil, 0, false, "actual-stage"); err != nil {
		t.Fatal(err)
	}
	publish, err := g.stageModernHDCompoundMap(frames, set)
	if err != nil {
		t.Fatal(err)
	}
	publish()
	f := frames[0].hdMap.prepared
	if len(f.Terrain) == 0 || len(f.Units) != 2 || !reflect.DeepEqual(before, stage42NativeState(g)) {
		t.Fatal("actual stage missing or native source changed")
	}
}

func TestModernHDCompoundStageRemapStartsFromSeed(t *testing.T) {
	g := stage42TestGame(t, true)
	in, err := freezeCompoundMapInput(*g.nativeMapFrameSource)
	if err != nil {
		t.Fatal(err)
	}
	in.Frame.LUT = make([]byte, 256)
	for i := range in.Frame.LUT {
		in.Frame.LUT[i] = byte(i + 1)
	}
	for i := range in.Frame.Cells {
		in.Frame.Cells[i].BlitMode = 0
	}
	work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	if err := indexedmap.SeedNativeCh23Staging(work, g.nativeCh23State.staging); err != nil {
		t.Fatal(err)
	}
	if err := indexedmap.ComposeNativeFrame(work, vga, in); err != nil {
		t.Fatal(err)
	}
	againWork, againVGA := append([]byte(nil), work...), append([]byte(nil), vga...)
	if err := indexedmap.ComposeNativeFrame(againWork, againVGA, in); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(vga, againVGA) {
		t.Fatal("fixture did not exercise destination remapping")
	}
	s := g.newModernHDCompoundMapScene(in, work, vga, g.nativeMapAssets.Palette, nil)
	if s.err != nil {
		t.Fatal("same-compose stage must be seeded before native replay", s.err)
	}
	frames := make([]nativeCompoundPresentedFrame, 1)
	if err := bindModernHDCompoundMap(frames, s, [][]byte{vga}, nil, nil, 0, false, "stage-remap"); err != nil {
		t.Fatal(err)
	}
	publish, err := g.stageModernHDCompoundMap(frames, g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	publish()
	if len(frames[0].hdMap.prepared.Terrain) == 0 {
		t.Fatal("remapped stage lost HD background")
	}
}

func TestModernHDItemMapFailureRestoresTransaction(t *testing.T) {
	set := itemHDSet(t)
	for _, index := range []int{0, 2, 5} {
		c := itemHDCases[index]
		t.Run(c.name, func(t *testing.T) {
			g, a, b := itemHDGame(t, c, false)
			g.modernStoryPortraits = set
			before, err := nativeItemRuntimeRecords(g.st.Units)
			if err != nil {
				t.Fatal(err)
			}
			work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
			done := 0
			startItemHD(t, g, a, b, c, false, &done)
			j := g.nativeAIItemPresentation
			for updates := 0; !j.published && updates < 1000; updates++ {
				j.drawn = true
				g.stepNativeAIItemPresentation()
			}
			if !j.published {
				t.Fatal("transaction publication not reached")
			}
			g.cancelNativeAIItemPresentation()
			after, err := nativeItemRuntimeRecords(g.st.Units)
			if err != nil {
				t.Fatal(err)
			}
			if g.nativeAIItemPresentation != nil || g.nativeRNGState != 7 || done != 0 || !bytes.Equal(before, after) || !bytes.Equal(work, g.nativeMapWork) || !bytes.Equal(vga, g.nativeMapVGA) {
				t.Fatal("late cancellation did not restore transaction")
			}
		})
	}
}

func TestModernHDItemMapMultipleTargets(t *testing.T) {
	set := itemHDSet(t)
	c := itemHDCases[2]
	g, a, b := itemHDGame(t, c, false)
	control, ca, cb := itemHDGame(t, c, false)
	g.modernStoryPortraits = set
	done, cdone := 0, 0
	for i, game := range []*Game{g, control} {
		actor, target, count := a, b, &done
		if i == 1 {
			actor, target, count = ca, cb, &cdone
		}
		extra := game.st.Units[1]
		extra.HP, extra.MaxHP = 1000, 1000
		extra.SetMapPlacement(6, 3, 3)
		if err := extra.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
		if err := game.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		plan := &battle.AIPlan{U: actor, Target: target, NativeActionKind: battle.NativeAIActionItem, NativeItemSlot: 0, NativeItemID: c.item, NativeItemTargetIndices: []byte{0, 1}}
		if err := game.executeNativeAIActionWithContinuation(plan, func() { *count++ }); err != nil {
			t.Fatal(err)
		}
	}
	j := g.nativeAIItemPresentation
	if j == nil || len(j.transaction.targets) != 2 || len(j.frames[0].hdMap.layers) != 2 || len(j.frames[8].hdMap.damageTargets) != 2 {
		t.Fatal("multiple target coverage absent")
	}
	for updates := 0; g.nativeAIItemPresentation != nil && updates < 1000; updates++ {
		if !reflect.DeepEqual(itemHDState(g), itemHDState(control)) {
			t.Fatal("multiple target native control differs", updates)
		}
		if !g.nativeAIItemPresentation.holding {
			g.nativeAIItemPresentation.drawn = true
			control.nativeAIItemPresentation.drawn = true
		}
		g.stepNativeAIItemPresentation()
		control.stepNativeAIItemPresentation()
	}
	if g.nativeAIItemPresentation != nil || control.nativeAIItemPresentation != nil || done != 1 || cdone != 1 || !reflect.DeepEqual(itemHDState(g), itemHDState(control)) {
		t.Fatal("multiple target completion differs")
	}
}

func TestModernHDItemMapPalettePreservesAlpha(t *testing.T) {
	set := itemHDSet(t)
	g, _, _ := itemHDGame(t, itemHDCases[2], false)
	palette := append(color.Palette(nil), g.nativeMapAssets.Palette...)
	for i, c := range palette {
		r, g, b, _ := c.RGBA()
		palette[i] = color.RGBA{255 - byte(r>>8), 255 - byte(g>>8), 255 - byte(b>>8), 255}
	}
	for index := 49; index <= 56; index++ {
		im := set.mapEffects.frames[index]
		before := append([]byte(nil), im.Pix...)
		out, err := g.colourModernHDMapLMI(im, g.nativeMapAssets.FDOTHER6[index].Pixels, palette)
		if err != nil {
			t.Fatal(err)
		}
		changed := false
		for at := 0; at < len(im.Pix); at += 4 {
			if im.Pix[at+3] != out.Pix[at+3] {
				t.Fatal("palette changed authored alpha", index, at)
			}
			changed = changed || !bytes.Equal(im.Pix[at:at+3], out.Pix[at:at+3])
		}
		if !changed || !bytes.Equal(before, im.Pix) {
			t.Fatal("palette ignored or art mutated", index)
		}
	}
}

func TestModernHDCompoundFrozenStage(t *testing.T) {
	g := stage42TestGame(t, true)
	in := *g.nativeMapFrameSource
	s := g.newModernHDCompoundMapScene(in, g.nativeMapWork, g.nativeMapVGA, g.nativeMapAssets.Palette, nil)
	if s.err != nil || s.stage == nil {
		t.Fatal("stage snapshot absent", s.err)
	}
	frames := make([]nativeCompoundPresentedFrame, 1)
	if err := bindModernHDCompoundMap(frames, s, [][]byte{g.nativeMapVGA}, nil, nil, 0, false, "stage"); err != nil {
		t.Fatal(err)
	}
	publish, err := g.stageModernHDCompoundMap(frames, g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	publish()
	f := frames[0].hdMap.prepared
	if len(f.Terrain) == 0 {
		t.Fatal("frozen stage reverted to native background")
	}
	want := healHDCPUWorld(f, g.modernStoryPortraits)
	g.nativeCh23State.staging[0] ^= 1
	g.nativeCh23State.rowShift++
	publish, err = g.stageModernHDCompoundMap(frames, g.modernStoryPortraits)
	if err != nil {
		t.Fatal("live stage leaked into frozen owner", err)
	}
	publish()
	if !bytes.Equal(want.Pix, healHDCPUWorld(frames[0].hdMap.prepared, g.modernStoryPortraits).Pix) {
		t.Fatal("live stage changed frozen image")
	}
	s.stage.staging[0] ^= 1
	if _, err = g.stageModernHDCompoundMap(frames, g.modernStoryPortraits); err == nil {
		t.Fatal("edited captured stage accepted")
	}
}
