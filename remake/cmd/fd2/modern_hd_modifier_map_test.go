package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type modifierHDCase struct {
	ID             int
	Player, Failed bool
}

func modifierHDCases() []modifierHDCase {
	var cases []modifierHDCase
	for _, id := range []int{17, 18, 19, 20, 21, 22, 23, 25, 26, 27} {
		cases = append(cases, modifierHDCase{ID: id, Player: true})
		if id != 23 && id != 25 {
			cases = append(cases, modifierHDCase{ID: id})
		}
	}
	for _, id := range []int{20, 22, 25} {
		cases = append(cases, modifierHDCase{ID: id, Player: true, Failed: true})
	}
	return cases
}

func modifierHDTestGame(t *testing.T, c modifierHDCase) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, actor, target := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	// Use the original enemy records whose AI fields are already materialized.
	// Normalized Camp alone cannot supply NativeAIScoringRecords provenance.
	if len(g.st.Units) < 6 {
		t.Fatal("two original AI records required")
	}
	actor, target = g.st.Units[4], g.st.Units[5]
	g.st.Units = []*battle.Unit{actor, target}
	for i, u := range g.st.Units {
		u.SetMapPlacement(2+i, 3, 3)
		if err := u.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
		u.Camp = battle.Enemy
		if c.Player {
			u.Camp = battle.Own
		}
		u.HP, u.MaxHP, u.AP, u.DP, u.HIT, u.EV = 50, 100, 100+i*100, 50, 80, 30
		u.MP, u.Acted, u.OnField = 80, false, true
		u.NativeRecordByte5, u.NativeRecordByte6 = 0, 0
		u.HasNativeRecordByte5, u.HasNativeRecordByte6 = true, true
		u.NativeRecordClass, u.HasNativeRecordClass = 1, true
		u.NativeTransient = [6]byte{}
		if c.ID == 20 || c.ID == 21 {
			u.NativeTransient[3], u.NativeTransient[4] = 3, 3
		}
		if c.ID == 22 || c.ID == 26 || c.ID == 27 {
			u.NativeRecordByte6 = 1
		}
		if c.ID == 25 && !c.Failed {
			u.NativeRecordByte5 = 0x80
		}
		if c.Failed && c.ID == 20 {
			u.NativeTransient = [6]byte{}
		}
		if c.Failed && c.ID == 22 {
			u.NativeTransient[5] = 3
		}
	}
	for id := range g.st.NativeCommandBook {
		g.st.NativeCommandBook[id] = battle.NativeCommandRecord{ID: id}
	}
	g.st.NativeCommandBook[10] = battle.NativeCommandRecord{ID: 10, Damage: 10}
	for _, id := range []int{17, 18, 19, 20, 21, 22, 23, 25, 26, 27} {
		mode := 0
		if !c.Player {
			mode = 1
		}
		g.st.NativeCommandBook[id] = battle.NativeCommandRecord{ID: id, SelectionMode: 1, EffectMode: mode, MPCost: 2, TargetCode: 1}
	}
	g.st.NativeCommandBook[18].MPCost = 4
	g.st.NativeCommandBook[17].MPCost = 99
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 3, CursorY: 3, VisibleCursorX: 3, VisibleCursorY: 3}); err != nil {
		t.Fatal(err)
	}
	if !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatal("modifier HUD unavailable")
	}
	g.curX, g.curY, g.nativeRNGState = 3, 3, 7
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g, actor, target
}

func startModifierHDTest(t *testing.T, g *Game, c modifierHDCase, done *int) {
	t.Helper()
	a, b := g.st.Units[0], g.st.Units[1]
	var err error
	if c.Player && (c.ID < 20 || c.ID == 23) {
		preflight := func() error { return nil }
		transaction := func() error { return nil }
		if c.ID < 20 {
			preflight = func() error {
				targets, e := g.st.NativeCommandModifierTargets(a, b, c.ID)
				if e != nil {
					return e
				}
				return g.st.ValidateNativeCommandModifierTargets(a, targets, c.ID)
			}
			transaction = func() error {
				result, e := g.st.ExecuteNativeCommandModifier(a, b, c.ID, g.nativeRNGState)
				if e == nil {
					g.nativeRNGState = result.RNGState
				}
				return e
			}
		}
		err = g.startNativeCommandPalettePresentation(c.ID, preflight, transaction, func() { *done++ })
	} else if !c.Player {
		err = g.executeNativeAIActionWithContinuation(&battle.AIPlan{U: a, Target: b, NativeActionKind: battle.NativeAIActionCommand, NativeCommandID: c.ID, NativeActionDestination: battle.Cell{X: a.X, Y: a.Y}}, func() { *done++ })
	} else {
		err = g.startNativeCommand2022Presentation(a, b, c.ID, c.Player, func(*battle.NativeAICommand2022Plan) { *done++ })
	}
	if err != nil {
		t.Fatal(c, err)
	}
}

func modifierHDTestTheme(t *testing.T) *modernStoryPortraitSet {
	t.Helper()
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"), os.Getenv("FD2_TEST_MODERN_EFFECT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func modifierHDReport(t *testing.T, suffix string, v any) {
	t.Helper()
	if prefix := os.Getenv("FD2_HD_MODIFIER_MAP_REPORT"); prefix != "" {
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(prefix+suffix, append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestModernModifierMapSourcesAndAtomic(t *testing.T) {
	if os.Getenv("FD2_HD_MODIFIER_MAP_UNIT") != "1" {
		t.Skip("explicit modifier map source process")
	}
	t.Setenv("FD2_MUTE", "1")
	set := modifierHDTestTheme(t)
	var rows []map[string]any
	for _, c := range modifierHDCases() {
		g, _, _ := modifierHDTestGame(t, c)
		done := 0
		before, rng := compoundMapUnitSnapshot(g), g.nativeRNGState
		startModifierHDTest(t, g, c, &done)
		frames := g.modernCompoundMapFrames()
		if len(frames) == 0 || !reflect.DeepEqual(before, compoundMapUnitSnapshot(g)) || rng != g.nativeRNGState || done != 0 {
			t.Fatal(c, "owner preflight mutated live state")
		}
		publish, err := g.stageModernHDCompoundMap(frames, set)
		if err != nil {
			t.Fatal(c, err)
		}
		for _, f := range frames {
			if f.hdMap == nil || f.hdMap.prepared != nil {
				t.Fatal("partial HD admission")
			}
		}
		publish()
		g.modernStoryPortraits = set
		phases, sources := map[string]int{}, map[string]int{}
		hd, native := 0, 0
		for _, f := range frames {
			if f.hdMap.prepared == nil || f.hdMap.theme != set {
				t.Fatal(c, "missing prepared map")
			}
			phases[f.hdPhase]++
			for _, l := range f.hdMap.layers {
				sources[fmt.Sprintf("%d:%d", l.Resource, l.Entry)]++
				bank := set.mapEffects
				if l.Resource == 5 {
					bank = set.mapOverlays
				}
				if bank != nil && bank.frames[l.Entry] != nil {
					hd++
				} else {
					native++
				}
			}
		}
		last := frames[len(frames)-1].hdMap
		active := last.prepared
		old := last.pixels
		last.pixels = append([]byte(nil), old...)
		last.pixels[0] ^= 1
		bad := *set
		if err := g.applyTheme(&bad); err == nil || g.modernStoryPortraits != set || last.prepared != active || last.theme != set {
			t.Fatal(c, "bad late VGA changed active map")
		}
		last.pixels = old
		bad = *set
		bad.mapSpritesHD = map[int][]image.Image{}
		// An absent HD world is legacy-compatible; a partial world must be refused.
		bad.mapSpritesHD[0] = set.mapSpritesHD[0]
		if err := g.applyTheme(&bad); err == nil || g.modernStoryPortraits != set || last.prepared != active {
			t.Fatal(c, "partial world admitted")
		}
		if !reflect.DeepEqual(before, compoundMapUnitSnapshot(g)) || rng != g.nativeRNGState {
			t.Fatal(c, "F2 admission changed transaction")
		}
		rows = append(rows, map[string]any{"case": c, "frames": len(frames), "phases": phases, "sources": sources, "registered_layers": hd, "native_fallback_layers": native, "atomic_rejections": 2})
	}
	modifierHDReport(t, ".sources.json", map[string]any{"status": "PASS-MODIFIER-MAP-SOURCES", "cases": rows, "seed": 7, "scope": "controlled source fixtures; command23 palette only; no ordinary player or new oracle"})
}

func TestModernModifierMapFixtureAdmission(t *testing.T) {
	if os.Getenv("FD2_HD_MODIFIER_MAP_FIXTURE") != "1" {
		t.Skip("explicit modifier fixture admission")
	}
	t.Setenv("FD2_MUTE", "1")
	for _, c := range modifierHDCases() {
		g, _, _ := modifierHDTestGame(t, c)
		done := 0
		startModifierHDTest(t, g, c, &done)
		frames := g.modernCompoundMapFrames()
		if len(frames) == 0 || done != 0 {
			t.Fatal(c, "no pending owner")
		}
		for _, f := range frames {
			if f.hdMap == nil || f.hdMap.scene == nil || f.hdMap.scene.err != nil {
				t.Fatal(c, "invalid frozen source", f.hdMap)
			}
		}
	}
}

func TestModernModifierMapControlSnapshot(t *testing.T) {
	if os.Getenv("FD2_HD_MODIFIER_MAP_FIXTURE") != "1" {
		t.Skip("explicit modifier control snapshot")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_HD_COMPOUND_MAP", "1")
	c := modifierHDCase{ID: 17, Player: true}
	g, _, _ := modifierHDTestGame(t, c)
	reference, _, _ := modifierHDTestGame(t, c)
	t.Log("initial independent VGA equal", bytes.Equal(g.nativeMapVGA, reference.nativeMapVGA))
	initCompoundMapFixture(t, g)
	t.Log("after HD helper recomposition equal", bytes.Equal(g.nativeMapVGA, reference.nativeMapVGA))
	if err := reference.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(g.nativeMapVGA, reference.nativeMapVGA) {
		count := 0
		var first []int
		for i, v := range g.nativeMapVGA {
			if v != reference.nativeMapVGA[i] {
				count++
				if len(first) < 16 {
					first = append(first, i)
				}
			}
		}
		t.Fatal("two recomposed native controls differ", count, first)
	}
}
