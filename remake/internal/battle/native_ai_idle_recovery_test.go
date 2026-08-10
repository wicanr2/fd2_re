package battle

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestApplyNativeAIIdleRecoveryUsesOneFifthAndCaps(t *testing.T) {
	records := make([]byte, nativeRecordSize)
	binary.LittleEndian.PutUint16(records[0x40:0x42], 70)
	binary.LittleEndian.PutUint16(records[0x42:0x44], 100)

	applied, err := ApplyNativeAIIdleRecovery(records, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !applied || binary.LittleEndian.Uint16(records[0x40:0x42]) != 90 {
		t.Fatalf("first recovery = (%v,%d), want (true,90)",
			applied, binary.LittleEndian.Uint16(records[0x40:0x42]))
	}
	applied, err = ApplyNativeAIIdleRecovery(records, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !applied || binary.LittleEndian.Uint16(records[0x40:0x42]) != 100 {
		t.Fatalf("capped recovery = (%v,%d), want (true,100)",
			applied, binary.LittleEndian.Uint16(records[0x40:0x42]))
	}
}

func TestPlanNativeAIIdleRecoveryIsPureAndPreservesRawDecisionFields(t *testing.T) {
	records := make([]byte, nativeRecordSize)
	binary.LittleEndian.PutUint16(records[0x40:0x42], 70)
	binary.LittleEndian.PutUint16(records[0x42:0x44], 100)
	records[0x25] = 0x12
	records[0x26] = 0x34
	want := append([]byte(nil), records...)
	decision, err := PlanNativeAIIdleRecovery(records, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Accepted || decision.CurrentHP != 70 || decision.MaximumHP != 100 ||
		decision.NextHP != 0 || decision.RawGate25 != 0x12 || decision.RawGate26 != 0x34 {
		t.Fatalf("decision=%+v", decision)
	}
	if string(records) != string(want) {
		t.Fatal("pure recovery planner mutated raw record")
	}

	records[0x25], records[0x26] = 0, 0
	decision, err = PlanNativeAIIdleRecovery(records, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Accepted || decision.NextHP != 90 {
		t.Fatalf("accepted decision=%+v", decision)
	}
}

func TestApplyNativeAIIdleRecoveryHonorsRawGates(t *testing.T) {
	for _, offset := range []int{0x25, 0x26} {
		records := make([]byte, nativeRecordSize)
		binary.LittleEndian.PutUint16(records[0x40:0x42], 50)
		binary.LittleEndian.PutUint16(records[0x42:0x44], 100)
		records[offset] = 1
		applied, err := ApplyNativeAIIdleRecovery(records, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		if applied || binary.LittleEndian.Uint16(records[0x40:0x42]) != 50 {
			t.Fatalf("gate +%#x mutated HP", offset)
		}
	}
}

func TestApplyNativeAIIdleRecoveryRejectsShortInput(t *testing.T) {
	if _, err := ApplyNativeAIIdleRecovery(make([]byte, nativeRecordSize-1), 1, 0); err == nil {
		t.Fatal("short record unexpectedly accepted")
	}
}

func TestBuildNativeAIIdleRecoveryPresentationPreservesRawSequence(t *testing.T) {
	decision := NativeAIIdleRecoveryDecision{
		Unit: 3, Accepted: true, CurrentHP: 70, MaximumHP: 100, NextHP: 90,
	}
	presentation, err := BuildNativeAIIdleRecoveryPresentation(decision)
	if err != nil {
		t.Fatal(err)
	}
	if presentation.Unit != 3 || presentation.BeforeHP != 70 || presentation.AfterHP != 90 ||
		presentation.SampleHandleExpr != "[0x53EEC]" || presentation.SampleIndex != 4 ||
		presentation.SampleLoopCount != 1 || presentation.WaitTicks != [3]uint32{1, 1, 1} {
		t.Fatalf("presentation header=%+v", presentation)
	}
	if presentation.BeforeGlobalWrite != (NativeAIIdleRecoveryGlobalWrite{Address: 0x51A83, Value: 0}) ||
		presentation.CoordinateCall != (NativeAIIdleRecoveryCall{Address: 0x12D7B, Unit: 3}) ||
		presentation.AfterGlobalWrite != (NativeAIIdleRecoveryGlobalWrite{Address: 0x51A83, Value: 1}) {
		t.Fatalf("raw calls/writes=%+v/%+v/%+v", presentation.BeforeGlobalWrite, presentation.CoordinateCall, presentation.AfterGlobalWrite)
	}
	if presentation.FirstDecode != (NativeAIIdleRecoveryDecode{
		ResourceBase: "[0x53A49]+0x8088", Mode: 2, Tail: 0xFD,
	}) || presentation.SecondDecode != (NativeAIIdleRecoveryDecode{
		ResourceBase: "[0x53A49]+0x8088", Mode: 0, Tail: 0,
	}) {
		t.Fatalf("decode passes=%+v/%+v", presentation.FirstDecode, presentation.SecondDecode)
	}
	wantCopy := NativeAIIdleRecoveryCopy{
		SourceOffset: 0xA0504, DestinationOffset: 0x140,
		SourceStride: 0x1C8, DestinationStride: 0x138,
		Rows: 0xC0, Length: 0x1C8,
	}
	if presentation.FirstCopy != wantCopy || presentation.SecondCopy != wantCopy {
		t.Fatalf("copy passes=%+v/%+v", presentation.FirstCopy, presentation.SecondCopy)
	}
}

func TestApplyNativeAIIdleRecoveryWithPresentationCommitsAfterCallback(t *testing.T) {
	records := make([]byte, nativeRecordSize)
	binary.LittleEndian.PutUint16(records[0x40:0x42], 70)
	binary.LittleEndian.PutUint16(records[0x42:0x44], 100)
	called := false
	applied, err := ApplyNativeAIIdleRecoveryWithPresentation(records, 1, 0,
		func(p NativeAIIdleRecoveryPresentation) error {
			called = true
			if p.BeforeHP != 70 || p.AfterHP != 90 {
				t.Fatalf("callback presentation=%+v", p)
			}
			if got := binary.LittleEndian.Uint16(records[0x40:0x42]); got != 70 {
				t.Fatalf("HP committed before callback returned: %d", got)
			}
			return nil
		})
	if err != nil || !applied || !called {
		t.Fatalf("applied=%v called=%v err=%v", applied, called, err)
	}
	if got := binary.LittleEndian.Uint16(records[0x40:0x42]); got != 90 {
		t.Fatalf("committed HP=%d, want 90", got)
	}
}

func TestApplyNativeAIIdleRecoveryWithPresentationFailsClosed(t *testing.T) {
	records := make([]byte, nativeRecordSize)
	binary.LittleEndian.PutUint16(records[0x40:0x42], 70)
	binary.LittleEndian.PutUint16(records[0x42:0x44], 100)
	if _, err := ApplyNativeAIIdleRecoveryWithPresentation(records, 1, 0, nil); err == nil {
		t.Fatal("missing presentation owner unexpectedly accepted")
	}
	if got := binary.LittleEndian.Uint16(records[0x40:0x42]); got != 70 {
		t.Fatalf("missing owner changed HP=%d", got)
	}
	if _, err := ApplyNativeAIIdleRecoveryWithPresentation(records, 1, 0,
		func(NativeAIIdleRecoveryPresentation) error { return fmt.Errorf("renderer unavailable") }); err == nil {
		t.Fatal("callback failure unexpectedly accepted")
	}
	if got := binary.LittleEndian.Uint16(records[0x40:0x42]); got != 70 {
		t.Fatalf("callback failure changed HP=%d", got)
	}
}

func TestApplyNativeAIIdleRecoveryWithPresentationRejectsChangedRawRecord(t *testing.T) {
	records := make([]byte, nativeRecordSize)
	binary.LittleEndian.PutUint16(records[0x40:0x42], 70)
	binary.LittleEndian.PutUint16(records[0x42:0x44], 100)
	applied, err := ApplyNativeAIIdleRecoveryWithPresentation(records, 1, 0,
		func(NativeAIIdleRecoveryPresentation) error {
			binary.LittleEndian.PutUint16(records[0x40:0x42], 71)
			return nil
		})
	if err == nil || applied {
		t.Fatalf("changed raw record accepted: applied=%v err=%v", applied, err)
	}
	if got := binary.LittleEndian.Uint16(records[0x40:0x42]); got != 70 {
		t.Fatalf("callback change was not rolled back: %d", got)
	}
}
