package main

import (
	"encoding/binary"
	"image/color"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func physicalSceneTestGame(t *testing.T) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	actor := &battle.Unit{Name: "索爾", Camp: battle.Own, OnField: true,
		HP: 80, MaxHP: 80, BattleFig: 0, HasBattleFig: true,
		HasNativeRecordByte5: true, AtkMin: 1, AtkMax: 1,
		NativeRecordByte6: 2, HasNativeRecordByte6: true,
		NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 1, HasNativeRecordClass: true, Dir: 2}
	target := &battle.Unit{Name: "敵軍", Camp: battle.Enemy, OnField: true, X: 1,
		HP: 80, MaxHP: 80, BattleFig: 78, HasBattleFig: true,
		HasNativeRecordByte5: true,
		HasNativeRecordByte6: true, NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 2, HasNativeRecordClass: true}
	g := &Game{st: &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target},
		HasNativeMapViewState: true, NativeMapRangeMode: 1, HasNativeMapRangeModeState: true,
		NativeTileBlitModes: []byte{255, 0}},
		m:              &MapData{W: 2, H: 1, Tiles: []int{0, 0}, NativeTerrainControl: []byte{0, 0, 55, 0}},
		handlerChapter: 7, nativeUIPalette: loadNativeUIPalette(),
		sel: actor, moved: true, curX: 1, rng: rand.New(rand.NewSource(73)), nativeRNGState: 17791}
	var err error
	g.nativeCommandScene, err = battle.LoadNativeCommandSceneTable(assetPath("assets/data/native_command_scene.json"))
	if err != nil {
		t.Fatal(err)
	}
	return g, actor, target
}

func requirePhysicalScenePack(t *testing.T) {
	t.Helper()
	if !fileExists(filepath.Join(separatedAssetPath("animations"), "FIGANI_001", "animation.json")) {
		t.Skip("player-generated separated pack is absent")
	}
}

func TestNativePhysicalSceneMatchesChapter8NormalOracleSelection(t *testing.T) {
	requirePhysicalScenePack(t)
	g, actor, target := physicalSceneTestGame(t)
	beforeActor, beforeTarget := *actor, *target
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	if scene.selection.BaseBG != 55 || scene.selection.TAI != 55 || scene.selection.HasSeparateBackgrounds ||
		scene.background.Bounds().Dx() != 320 || scene.background.Bounds().Dy() != 100 ||
		scene.pedestal.Bounds().Dx() != 154 || scene.pedestal.Bounds().Dy() != 42 {
		t.Fatalf("original ch08 BG55/TAI55 tuple changed: %+v", scene.selection)
	}
	if !reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) || g.nativeRNGState != 17791 {
		t.Fatal("scene preparation changed battle input")
	}
}

func TestNativePhysicalTerrainUsesMutableCellAndRejectsMalformedGrid(t *testing.T) {
	g, actor, _ := physicalSceneTestGame(t)
	g.m.NativeTerrainControl = append(g.m.NativeTerrainControl, 0, 2, 5, 0)
	g.st.HasNativeMapEventGrid = true
	g.st.NativeMapEventGrid = make([]byte, 12)
	g.st.NativeMapEventGrid[0], g.st.NativeMapEventGrid[2] = 2, 1
	binary.LittleEndian.PutUint16(g.st.NativeMapEventGrid[4:], 0xfc01)
	control, err := g.nativePhysicalTerrainControl(actor)
	if err != nil || control != [4]byte{0, 2, 5, 0} {
		t.Fatalf("scene read initial tile or high bits: %v %v", control, err)
	}
	g.st.NativeMapEventGrid[0] = 3
	if _, err := g.nativePhysicalTerrainControl(actor); err == nil {
		t.Fatal("bad map header fell back to initial tiles")
	}
	g.st.NativeMapEventGrid = g.st.NativeMapEventGrid[:4]
	if _, err := g.nativePhysicalTerrainControl(actor); err == nil {
		t.Fatal("truncated mutable map fell back to initial tiles")
	}
}

func TestNativePhysicalSurfaceRetainsOpaqueIndexZero(t *testing.T) {
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{R: byte(i), A: 255}
	}
	palette[0] = color.RGBA{R: 17, G: 23, B: 31, A: 255}
	frame := fdother.Frame{Width: 2, Height: 1, Indexed: []byte{0, 1}, Mask: []byte{255, 0}}
	img, err := nativePhysicalSurfaceRGBA(frame, palette)
	if err != nil {
		t.Fatal(err)
	}
	pixels := img.Pix
	if !reflect.DeepEqual(pixels, []byte{17, 23, 31, 255, 0, 0, 0, 0}) {
		t.Fatalf("palette zero or retained span changed: %v", pixels)
	}
	frame.Mask[1] = 1
	if _, err := nativePhysicalSurfaceRGBA(frame, palette); err == nil {
		t.Fatal("invalid mask accepted")
	}
}

func TestNativePhysicalSceneMissingBGStopsPlayerAndMode11BeforeSettlement(t *testing.T) {
	requirePhysicalScenePack(t)
	animationRoot, err := filepath.Abs(separatedAssetPath("animations"))
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range []string{"player", "mode11"} {
		t.Run(owner, func(t *testing.T) {
			g, actor, target := physicalSceneTestGame(t)
			if err := g.ensureNativeAttackPresentation(actor.BattleFig, target.BattleFig); err != nil {
				t.Fatal(err)
			}
			pack := t.TempDir()
			if err := os.Symlink(animationRoot, filepath.Join(pack, "animations")); err != nil {
				t.Fatal(err)
			}
			t.Setenv("FD2_ASSET_PACK", pack)
			beforeActor, beforeTarget := *actor, *target
			if owner == "player" {
				g.confirm()
			} else {
				g.aiBusy = true
				g.executeNativeAIMode11Physical(&battle.AIPlan{U: actor, Target: target}, nil)
			}
			if !strings.Contains(g.loadErr, "BG55") || g.atk != nil || g.nativeRNGState != 17791 ||
				!reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) {
				t.Fatalf("missing BG changed transaction: actor=%+v target=%+v err=%s", actor, target, g.loadErr)
			}
			if got, want := g.rng.Int63(), rand.New(rand.NewSource(73)).Int63(); got != want {
				t.Fatal("missing BG consumed legacy RNG")
			}
		})
	}
}
