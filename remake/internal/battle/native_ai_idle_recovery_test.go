package battle

import (
	"encoding/binary"
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
