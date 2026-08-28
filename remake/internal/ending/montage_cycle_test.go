package ending

import (
	"os"
	"testing"
)

func montageCyclePlayerPaths() MontageArchivePaths {
	const base = "../../../org_game/炎龍騎士團/FLAME2/"
	return MontageArchivePaths{
		FDOTHER:       base + "FDOTHER.DAT",
		SurfaceRoot:   "../../generated-assets/fd2-original-b97caf22/surfaces",
		AnimationRoot: "../../generated-assets/fd2-original-b97caf22/animations",
		PortraitRoot:  "../../generated-assets/fd2-original-b97caf22/portraits",
		TextRoot:      "../../generated-assets/fd2-original-b97caf22/text",
		FontRoot:      "../../generated-assets/fd2-original-b97caf22/fonts",
	}
}

func montageCycleUnits() [][]byte {
	units := make([][]byte, 2)
	for i := range units {
		units[i] = make([]byte, 0x21)
		if i == 1 {
			// 0x2918f is a zero/nonzero test, not an equality test for one.
			units[i][6] = 2
		}
		units[i][7] = 4    // player-provided FIGANI/DATO group
		units[i][8] = 4    // permanent FDTXT character index source
		units[i][0x20] = 2 // permanent FDTXT class index source
	}
	return units
}

func montageCycleThreeUnits() [][]byte {
	units := montageCycleUnits()
	unit := make([]byte, 0x21)
	unit[6], unit[7], unit[8], unit[0x20] = 2, 4, 4, 2
	return append(units, unit)
}

func TestLoadMontageCycleAssetsUsesOnlyProvenanceBoundPlayerResources(t *testing.T) {
	paths := montageCyclePlayerPaths()
	for _, path := range []string{paths.FDOTHER, paths.SurfaceRoot + "/TAI_003/resource.json", paths.AnimationRoot + "/FIGANI_012/animation.json", paths.PortraitRoot + "/DATO_004_m0.png", paths.TextRoot + "/FDTXT_031/resource.json"} {
		if _, err := os.Stat(path); err != nil {
			t.Skip("player-provided original archives are absent")
		}
	}
	montage, err := LoadMontage("../../assets/endings/native_2c548.json")
	if err != nil {
		t.Fatal(err)
	}
	assets, err := LoadMontageCycleAssets(*montage, paths, montageCycleUnits())
	if err != nil {
		t.Fatal(err)
	}
	if assets.Backdrop.Width != Width || assets.Backdrop.Height != Height || len(assets.Grid) != Bytes || !isTransparentTAI003(assets.TAI003) {
		t.Fatalf("assets geometry=%dx%d grid=%d tai=%dx%d", assets.Backdrop.Width, assets.Backdrop.Height, len(assets.Grid), assets.TAI003.Width, assets.TAI003.Height)
	}
	if assets.Primary[13] == nil || assets.Secondary[12] == nil || len(assets.Portraits[4]) != 4 {
		t.Fatalf("missing group 4 FIGANI/DATO assets: primary=%v secondary=%v portraits=%d", assets.Primary[13] != nil, assets.Secondary[12] != nil, len(assets.Portraits[4]))
	}
}

func TestMontageCycleExecutesBothNativeSideBranchesAndFinalPaletteFade(t *testing.T) {
	paths := montageCyclePlayerPaths()
	for _, path := range []string{paths.FDOTHER, paths.SurfaceRoot + "/TAI_003/resource.json", paths.AnimationRoot + "/FIGANI_012/animation.json", paths.PortraitRoot + "/DATO_004_m0.png", paths.TextRoot + "/FDTXT_031/resource.json"} {
		if _, err := os.Stat(path); err != nil {
			t.Skip("player-provided original archives are absent")
		}
	}
	montage, err := LoadMontage("../../assets/endings/native_2c548.json")
	if err != nil {
		t.Fatal(err)
	}
	units := montageCycleUnits()
	assets, err := LoadMontageCycleAssets(*montage, paths, units)
	if err != nil {
		t.Fatal(err)
	}
	c := NewIndexedCompositor()
	if err := c.PresentANI(make([]byte, Bytes), make([]byte, len(c.Palette))); err != nil {
		t.Fatal(err)
	}
	cycle, err := NewMontageCycle(*montage, assets, units, []byte{4, 4}, c)
	if err != nil {
		t.Fatal(err)
	}
	steps := 0
	for !cycle.Ready() && steps < 2000 {
		if err := cycle.Step(byte(steps * 17)); err != nil {
			t.Fatalf("step %d phase=%s: %v", steps, cycle.Phase, err)
		}
		steps++
	}
	if !cycle.Ready() {
		t.Fatalf("native cycle did not complete after %d steps; phase=%s plan=%d", steps, cycle.Phase, cycle.PlanIndex)
	}
	if cycle.PlanIndex != 2 || cycle.FadeOut != 64 {
		t.Fatalf("completed cycle state=%#v", cycle)
	}
}

func TestMontageCycleInputChangeFinishesCurrentPortraitThenJumpsToFinalLoop(t *testing.T) {
	paths := montageCyclePlayerPaths()
	for _, path := range []string{paths.FDOTHER, paths.SurfaceRoot + "/TAI_003/resource.json", paths.AnimationRoot + "/FIGANI_012/animation.json", paths.PortraitRoot + "/DATO_004_m0.png", paths.TextRoot + "/FDTXT_031/resource.json"} {
		if _, err := os.Stat(path); err != nil {
			t.Skip("player-provided original archives are unavailable")
		}
	}
	montage, err := LoadMontage("../../assets/endings/native_2c548.json")
	if err != nil {
		t.Fatal(err)
	}
	units := montageCycleThreeUnits()
	assets, err := LoadMontageCycleAssets(*montage, paths, units)
	if err != nil {
		t.Fatal(err)
	}
	compositor := NewIndexedCompositor()
	if err := compositor.PresentANI(make([]byte, Bytes), make([]byte, len(compositor.Palette))); err != nil {
		t.Fatal(err)
	}
	cycle, err := NewMontageCycle(*montage, assets, units, []byte{4, 4, 4}, compositor)
	if err != nil {
		t.Fatal(err)
	}
	steps := 0
	for cycle.Phase != MontagePhasePortrait && steps < 2000 {
		if err := cycle.Step(byte(steps)); err != nil {
			t.Fatalf("reach portrait step %d phase=%s: %v", steps, cycle.Phase, err)
		}
		steps++
	}
	if cycle.Phase != MontagePhasePortrait || cycle.PlanIndex != 0 {
		t.Fatalf("first portrait state=%s plan=%d after %d steps", cycle.Phase, cycle.PlanIndex, steps)
	}
	// 0x2c950 polls after the portrait is presented and its one-tick wait has
	// elapsed.  An input before that boundary must not alter the pending frame.
	if cycle.ObserveInputChange() {
		t.Fatal("input was admitted before the first portrait presentation")
	}
	if err := cycle.Step(byte(steps)); err != nil {
		t.Fatalf("render first portrait step %d: %v", steps, err)
	}
	steps++
	if !cycle.ObserveInputChange() {
		t.Fatal("portrait input change was not admitted")
	}
	for cycle.PlanIndex == 0 && steps < 2600 {
		if err := cycle.Step(byte(steps)); err != nil {
			t.Fatalf("portrait skip step %d phase=%s: %v", steps, cycle.Phase, err)
		}
		steps++
	}
	if cycle.PlanIndex != len(cycle.Plans)-1 || cycle.Phase != MontagePhaseFigureFade {
		t.Fatalf("input change did not preserve current portrait then select final loop: phase=%s plan=%d plans=%d", cycle.Phase, cycle.PlanIndex, len(cycle.Plans))
	}
}

func TestMontageCycleDoesNotAdvanceRandomDuringPortraitBoundary(t *testing.T) {
	cycle := &MontageCycle{
		Phase:                   MontagePhasePortrait,
		PortraitSM:              MontagePortraitState{Countdown: 0},
		portraitBoundaryPending: true,
	}
	if cycle.NeedsRandomByte() {
		t.Fatal("portrait boundary requested a random byte that no native render iteration consumes")
	}
	cycle.portraitBoundaryPending = false
	if !cycle.NeedsRandomByte() {
		t.Fatal("next portrait render did not request its native random byte")
	}
}
