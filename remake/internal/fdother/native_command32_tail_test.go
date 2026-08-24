package fdother

import (
	"os"
	"testing"
)

func TestBuildNativeCommand32TailSchedulePreservesRawContract(t *testing.T) {
	s, err := BuildNativeCommand32TailSchedule(command9LMIEntries(0x4c), command9LMIEntries(0x77))
	if err != nil {
		t.Fatal(err)
	}
	if s.EffectResource != 6 || s.EffectStart != 0x40 || s.EffectFrames != 10 ||
		s.SoundResource != 80 || s.EffectSample != 9 || s.EffectDelayTicks != 1 ||
		s.ToggleA != 0x4a || s.ToggleB != 0x4b || s.TogglePairs != 4 || s.ToggleDelayMS != 90 ||
		s.DigitResource != 5 || s.DigitBias != 0x5e || s.DigitFrames != 22 ||
		s.DigitDelayMS != 2 || s.DigitHoldMS != 500 {
		t.Fatalf("command32 tail schedule=%#v", s)
	}
}

func TestBuildNativeCommand32TailScheduleFailsClosed(t *testing.T) {
	if _, err := BuildNativeCommand32TailSchedule(command9LMIEntries(0x4b), command9LMIEntries(0x77)); err == nil {
		t.Fatal("command32 accepted a missing toggle descriptor")
	}
	if _, err := BuildNativeCommand32TailSchedule(command9LMIEntries(0x4c), command9LMIEntries(0x76)); err == nil {
		t.Fatal("command32 accepted incomplete miss digits")
	}
}

func TestNativeCommand32TailOriginalAssets(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(path); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	effect, err := DecodeLMI1Resource(path, NativeCommand32TailEffectResource)
	if err != nil {
		t.Fatal(err)
	}
	digits, err := DecodeLMI1Resource(path, NativeCommand32TailDigitResource)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildNativeCommand32TailSchedule(effect, digits); err != nil {
		t.Fatal(err)
	}
	raw, err := ReadNestedResource(path, NativeCommand32TailSoundResource, NativeCommand32TailEffectSample)
	if err != nil || len(raw) == 0 {
		t.Fatalf("FDOTHER #80 selector9 len=%d err=%v", len(raw), err)
	}
}
