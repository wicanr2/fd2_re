package main

import (
	"image/color"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestEnsureNativeAttackPresentationLoadsSeparatedResourcesWithoutArchive(t *testing.T) {
	animationRoot := separatedAssetPath("animations")
	for _, name := range []string{"FIGANI_168", "FIGANI_379"} {
		if !fileExists(filepath.Join(animationRoot, name, "animation.json")) {
			t.Skip("player-generated separated FIGANI pack is absent")
		}
	}
	t.Setenv("FD2_ORIGINAL_FIGANI", filepath.Join(t.TempDir(), "missing-FIGANI.DAT"))
	g := &Game{nativeUIPalette: loadNativeUIPalette()}
	if err := g.ensureNativeAttackPresentation(126, 56); err != nil {
		t.Fatal(err)
	}
	if !g.nativeAttackPresentationAvailable(126, 56) {
		t.Fatal("late FIGANI pair is not available after exact raw load")
	}
	for _, id := range []int{379, 168} {
		if len(g.figani[id]) == 0 || len(g.figaniDelays[id]) != len(g.figani[id]) ||
			len(g.figMeta[id]) != len(g.figani[id]) {
			t.Fatalf("resource %d was not published as a complete frame/delay/position set", id)
		}
	}
}

func TestPlayerPhysicalAttackPreflightsSeparatedPairBeforeMutation(t *testing.T) {
	seed := int64(73)
	actor := &battle.Unit{
		Name: "亞雷斯", Camp: battle.Own, OnField: true, X: 0, Y: 0,
		HP: 80, MaxHP: 80, AP: 70, HIT: 100, Dir: 2, BattleFig: 126,
	}
	target := &battle.Unit{
		Name: "盜賊", Camp: battle.Enemy, OnField: true, X: 1, Y: 0,
		HP: 100, MaxHP: 100, DP: 10, BattleFig: 56,
	}
	rng := rand.New(rand.NewSource(seed))
	palette := make(color.Palette, 256)
	g := &Game{
		st:  &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}},
		sel: actor, moved: true, curX: 1, curY: 0, rng: rng,
		nativeUIPalette: palette,
	}
	t.Setenv("FD2_ASSET_PACK", t.TempDir())
	t.Setenv("FD2_ORIGINAL_FIGANI", filepath.Join(t.TempDir(), "missing-FIGANI.DAT"))

	g.confirm()
	if target.HP != target.MaxHP || actor.Acted || actor.Dir != 2 || g.sel != actor {
		t.Fatalf("failed preflight mutated player attack: hp=%d acted=%v dir=%d selected=%v", target.HP, actor.Acted, actor.Dir, g.sel == actor)
	}
	if got, want := rng.Int63(), rand.New(rand.NewSource(seed)).Int63(); got != want {
		t.Fatalf("failed preflight consumed RNG: got=%d want=%d", got, want)
	}
	if !strings.Contains(g.loadErr, "FIGANI attack presentation unavailable") {
		t.Fatalf("loadErr=%q", g.loadErr)
	}
}

func TestPlayerPhysicalAttackLoadsDynamicSeparatedPairWithoutArchive(t *testing.T) {
	animationRoot := separatedAssetPath("animations")
	for _, name := range []string{"FIGANI_013", "FIGANI_288"} {
		if !fileExists(filepath.Join(animationRoot, name, "animation.json")) {
			t.Skip("player-generated separated FIGANI pack is absent")
		}
	}
	actor := &battle.Unit{
		Name: "亞雷斯", Camp: battle.Own, OnField: true, X: 0, Y: 0,
		HP: 80, MaxHP: 80, AP: 70, HIT: 100, BattleFig: 4,
	}
	target := &battle.Unit{
		Name: "盜賊", Camp: battle.Enemy, OnField: true, X: 1, Y: 0,
		HP: 100, MaxHP: 100, DP: 10, BattleFig: 96,
	}
	g := &Game{
		st:  &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}},
		sel: actor, moved: true, curX: 1, curY: 0,
		rng: rand.New(rand.NewSource(73)), nativeUIPalette: loadNativeUIPalette(),
	}
	t.Setenv("FD2_ORIGINAL_FIGANI", filepath.Join(t.TempDir(), "missing-FIGANI.DAT"))
	g.confirm()
	if g.atk == nil || target.HP >= target.MaxHP || !actor.Acted {
		t.Fatalf("dynamic separated player attack was not committed: atk=%v hp=%d acted=%v err=%q", g.atk != nil, target.HP, actor.Acted, g.loadErr)
	}
	for _, id := range []int{13, 288} {
		if len(g.figani[id]) == 0 || len(g.figaniDelays[id]) != len(g.figani[id]) || len(g.figMeta[id]) != len(g.figani[id]) {
			t.Fatalf("resource %d was not loaded as one complete separated cache entry", id)
		}
	}
}

func TestEnsureNativeAttackPresentationRejectsWithoutPartialPublish(t *testing.T) {
	t.Setenv("FD2_ORIGINAL_FIGANI", filepath.Join(t.TempDir(), "missing.dat"))
	t.Setenv("FD2_ASSET_PACK", t.TempDir())
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{R: byte(i), G: byte(i), B: byte(i), A: 0xff}
	}
	g := &Game{nativeUIPalette: palette}
	if err := g.ensureNativeAttackPresentation(126, 56); err == nil {
		t.Fatal("missing separated FIGANI resources were accepted")
	}
	if len(g.figani) != 0 || len(g.figaniDelays) != 0 || len(g.figMeta) != 0 {
		t.Fatal("failed preload partially published presentation maps")
	}
}
