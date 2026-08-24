package fdother

import (
	"os"
	"testing"
)

func TestBuildNativeCommand10To12SchedulePreservesRawTables(t *testing.T) {
	for id := 10; id <= 12; id++ {
		s, err := BuildNativeCommand10To12Schedule(id)
		if err != nil {
			t.Fatal(err)
		}
		if s.SoundResource != 80 || s.XOrigins != [3]int{128, 0, -128} || s.YOrigins != [3]int{128, 0, 128} ||
			s.SamplingIncrements != [3]int{131, 128, 125} || s.SurfaceCycle != [4]int{0, 1, 2, 3} ||
			s.Frames != 60 || s.DelayMS != 10 || s.MainSample != 13 || s.MainSampleFrames != [8]int{0, 6, 12, 18, 24, 30, 36, 42} ||
			s.ResultResource != 5 || s.ResultDigitBias != 94 || s.ResultMissDescriptors != [4]int{116, 117, 118, 118} || s.ResultFrames != 22 || s.ResultHoldMS != 500 {
			t.Fatalf("command %d schedule=%#v", id, s)
		}
	}
	s10, _ := BuildNativeCommand10To12Schedule(10)
	s11, _ := BuildNativeCommand10To12Schedule(11)
	s12, _ := BuildNativeCommand10To12Schedule(12)
	if s10.Prelude.Enabled || s11.Prelude != (NativeCommand10To12Prelude{Enabled: true, InitialRadius: 15, RadiusStep: 10, Repeat: 10, Sample: 2}) ||
		s12.Prelude != (NativeCommand10To12Prelude{Enabled: true, InitialRadius: 30, RadiusStep: 16, Repeat: 10, Sample: 2}) {
		t.Fatalf("preludes: 10=%#v 11=%#v 12=%#v", s10.Prelude, s11.Prelude, s12.Prelude)
	}
	if _, err := BuildNativeCommand10To12Schedule(9); err == nil {
		t.Fatal("command9 accepted")
	}
	if _, err := BuildNativeCommand10To12Schedule(13); err == nil {
		t.Fatal("command13 accepted")
	}
}

func TestNativeCommand10To12OriginalSoundSelectors(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(path); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	for _, selector := range []int{2, 13} {
		raw, err := ReadNestedResource(path, NativeCommand10To12SoundResource, selector)
		if err != nil || len(raw) == 0 {
			t.Fatalf("FDOTHER #80 selector %d len=%d err=%v", selector, len(raw), err)
		}
	}
}
