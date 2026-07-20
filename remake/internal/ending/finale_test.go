package ending

import "testing"

func TestNative2C405Phase0IsEditableButFailClosed(t *testing.T) {
	phase, err := LoadFinalePhase("../../assets/endings/native_2c405.json")
	if err != nil {
		t.Fatal(err)
	}
	if phase.Ready() {
		t.Fatal("unrecovered finale text compositor must not be playable")
	}
	p := phase.Phase
	if p.SourceDAT != "FDTXT_030" || p.NativeSelector != 44 || p.RawStringIndex != 10 || p.RawUtteranceIndex != 6 || p.Script != "ch30.json" || p.SceneIndex != 3 || p.Line != 6 || p.Count != 1 {
		t.Fatalf("editable script reference = %#v", p)
	}
	if p.StagingBytes != 0x36b00 || p.TextOffset != 0x12c30 || p.Stride != 320 || p.ViewportRows != 200 || p.Iterations != 500 || p.DelayMS != 1 || p.BaselinePaletteInitialDelta != 40 || p.FadeOutThroughIteration != 300 || p.PaletteStepCadence != 5 {
		t.Fatalf("phase-0 native schedule = %#v", p)
	}
}
