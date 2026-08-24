package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func command32TailEntries(count int) []fdother.LMI1Entry {
	entries := make([]fdother.LMI1Entry, count)
	for index := range entries {
		entries[index] = fdother.LMI1Entry{Width: 1, Height: 1, Pixels: []byte{byte(index)}}
	}
	return entries
}

func TestBuildNativeCommand32TailFramesPreservesAllPhases(t *testing.T) {
	effect, digits := command32TailEntries(0x4c), command32TailEntries(0x77)
	schedule, err := fdother.BuildNativeCommand32TailSchedule(effect, digits)
	if err != nil {
		t.Fatal(err)
	}
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	vga := make([]byte, indexedmap.NativeMapVGASize)
	targets := []indexedmap.NativeCommandHealTailTarget{{RecordIndex: 3, X: 1, Y: 1}}
	frames, err := BuildNativeCommand32TailFrames(work, vga, work, vga, effect, digits, targets,
		[]NativeCommand32TailResult{{TargetRecord: 3, Hit: true, Damage: 42}}, 0, 0, schedule)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames.Effect) != 10 || len(frames.Toggle) != 8 || len(frames.Result) != 22 || len(frames.Queue) != 4 {
		t.Fatalf("command32 tail frames=%+v", frames)
	}
	if frames.Queue[0].Digit != 0 || frames.Queue[2].Digit != schedule.DigitBias+4 || frames.Queue[3].Digit != schedule.DigitBias+2 {
		t.Fatalf("command32 damage queue=%+v", frames.Queue)
	}
}

func TestBuildNativeCommand32TailFramesMissAndFailureAreAtomic(t *testing.T) {
	effect, digits := command32TailEntries(0x4c), command32TailEntries(0x77)
	schedule, _ := fdother.BuildNativeCommand32TailSchedule(effect, digits)
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	vga := make([]byte, indexedmap.NativeMapVGASize)
	targets := []indexedmap.NativeCommandHealTailTarget{{RecordIndex: 3, X: 1, Y: 1}}
	frames, err := BuildNativeCommand32TailFrames(work, vga, work, vga, effect, digits, targets,
		[]NativeCommand32TailResult{{TargetRecord: 3}}, 0, 0, schedule)
	if err != nil {
		t.Fatal(err)
	}
	if got := []int{frames.Queue[0].Digit, frames.Queue[1].Digit, frames.Queue[2].Digit, frames.Queue[3].Digit}; got[0] != 0x74 || got[1] != 0x75 || got[2] != 0x76 || got[3] != 0x76 {
		t.Fatalf("command32 miss queue=%v", got)
	}
	beforeWork, beforeVGA := append([]byte(nil), work...), append([]byte(nil), vga...)
	if _, err := BuildNativeCommand32TailFrames(work, vga, work, vga, effect[:0x4b], digits, targets,
		[]NativeCommand32TailResult{{TargetRecord: 3}}, 0, 0, schedule); err == nil {
		t.Fatal("command32 accepted missing toggle descriptor")
	}
	if string(work) != string(beforeWork) || string(vga) != string(beforeVGA) {
		t.Fatal("failed command32 tail build mutated caller buffers")
	}
}
