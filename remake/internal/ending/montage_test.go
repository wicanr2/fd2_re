package ending

import "testing"

func TestNative2C548MontageMapPlansNativePartySlotOrder(t *testing.T) {
	montage, err := LoadMontage("../../assets/endings/native_2c548.json")
	if err != nil {
		t.Fatal(err)
	}
	plans, err := montage.PlanPartyCycle([]byte{4, 0, 96})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 3 || plans[0] != (PartyCyclePlan{LoopIndex: 2, UnitSlot: 2, VisualGroup: 96, PrimaryFIGANI: 289, SecondaryFIGANI: 288, Frames: 20, FrameDelayMS: 1}) || plans[1].LoopIndex != 1 || plans[1].UnitSlot != 0 || plans[1].PrimaryFIGANI != 13 || plans[2].LoopIndex != 0 || plans[2].UnitSlot != 1 || plans[2].PrimaryFIGANI != 1 || plans[2].SecondaryFIGANI != 0 {
		t.Fatalf("party plans = %#v", plans)
	}
}

func TestNative2C548MontageRefusesEmptyParty(t *testing.T) {
	montage, err := LoadMontage("../../assets/endings/native_2c548.json")
	if err != nil {
		t.Fatal(err)
	}
	if plans, err := montage.PlanPartyCycle([]byte{4}); err == nil || plans != nil {
		t.Fatalf("plans=%#v err=%v", plans, err)
	}
}

func TestNative2C548FigureFadeIsNineNonMirroredPasses(t *testing.T) {
	montage, err := LoadMontage("../../assets/endings/native_2c548.json")
	if err != nil {
		t.Fatal(err)
	}
	passes, err := montage.PlanFigureFade(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(passes) != 9 || passes[0] != (FigureFadePass{Stage: 8, SourceOffset: 80, PaletteDelta: 48}) || passes[8] != (FigureFadePass{Stage: 0, SourceOffset: 0, PaletteDelta: 0}) {
		t.Fatalf("fade passes=%#v", passes)
	}
	if passes, err := montage.PlanFigureFade(0); err == nil || passes != nil {
		t.Fatalf("mirrored passes=%#v err=%v", passes, err)
	}
}
