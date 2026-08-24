package fdother

import (
	"os"
	"testing"
)

func command9LMIEntries(count int) []LMI1Entry {
	entries := make([]LMI1Entry, count)
	for index := range entries {
		entries[index] = LMI1Entry{Width: 1, Height: 1, Pixels: []byte{byte(index)}}
	}
	return entries
}

func TestBuildNativeCommand9PlayerSchedulePreservesRawTables(t *testing.T) {
	s, err := BuildNativeCommand9PlayerSchedule(command9LMIEntries(230), command9LMIEntries(138))
	if err != nil {
		t.Fatal(err)
	}
	if s.EffectStart != 87 || s.EffectFrames != 27 || s.SoundResource != 80 || s.InitialSample != 14 || s.RepeatSample != 15 ||
		s.RepeatSampleFrames != [2]int{15, 19} || s.ResultDigitBias != 94 ||
		s.ResultMissDescriptors != [4]int{116, 117, 118, 118} || s.ResultFrames != 22 {
		t.Fatalf("command9 player schedule=%#v", s)
	}
}

func TestBuildNativeCommand9PlayerScheduleFailsClosed(t *testing.T) {
	if _, err := BuildNativeCommand9PlayerSchedule(command9LMIEntries(113), command9LMIEntries(138)); err == nil {
		t.Fatal("short effect bank was accepted")
	}
	if _, err := BuildNativeCommand9PlayerSchedule(command9LMIEntries(230), command9LMIEntries(118)); err == nil {
		t.Fatal("short result bank was accepted")
	}
}

func TestNativeCommand9PlayerOriginalDescriptors(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(path); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	effect, err := DecodeLMI1Resource(path, NativeCommand9PlayerEffectResource)
	if err != nil {
		t.Fatal(err)
	}
	result, err := DecodeLMI1Resource(path, NativeCommand9ResultResource)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildNativeCommand9PlayerSchedule(effect, result); err != nil {
		t.Fatal(err)
	}
	for _, selector := range []int{0, 14, 15} {
		raw, err := ReadNestedResource(path, NativeCommand9PlayerSoundResource, selector)
		if err != nil || len(raw) == 0 {
			t.Fatalf("FDOTHER #80 selector %d len=%d err=%v", selector, len(raw), err)
		}
	}
}
