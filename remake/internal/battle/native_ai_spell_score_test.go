package battle

import (
	"encoding/binary"
	"testing"
)

func TestScoreNativeAISpellAttackPreservesRawBranches(t *testing.T) {
	records := make([]byte, 2*nativeRecordSize)
	binary.LittleEndian.PutUint16(records[0x40:], 10)
	binary.LittleEndian.PutUint16(records[nativeRecordSize+0x40:], 20)
	got, err := ScoreNativeAISpellAttack(records, []byte{0, 1}, 0, 15, nil)
	if err != nil || got != 48 { // 24*1.5 + 8*1.5, toward-zero
		t.Fatalf("score=%d err=%v", got, err)
	}
	if _, err := ScoreNativeAISpellAttack(records, []byte{0}, 10, 15, nil); err == nil {
		t.Fatal("ID10 accepted without its raw inactive gate")
	}
	got, err = ScoreNativeAISpellAttack(records, []byte{0, 1}, 10, 15, func(index int) bool { return index == 1 })
	if err != nil || got != 36 {
		t.Fatalf("gated score=%d err=%v", got, err)
	}
}

func TestNativeAITargetGateFeedsAttackIDsTenThroughTwelve(t *testing.T) {
	records := make([]byte, 5*nativeRecordSize)
	for i := 0; i < 5; i++ {
		record := records[i*nativeRecordSize : (i+1)*nativeRecordSize]
		record[8] = 1
		binary.LittleEndian.PutUint16(record[0x40:], 10)
	}
	records[0x20] = 19
	records[nativeRecordSize+0x1f] = 4
	records[2*nativeRecordSize+0x1f] = 5
	// +7=0x1C優先回0，不能因race/class成立就略過這一筆。
	records[3*nativeRecordSize+7] = 0x1c
	records[3*nativeRecordSize+0x20] = 19
	records[3*nativeRecordSize+0x1f] = 4
	gate := nativeAI1F183TargetGate(records)
	for _, command := range []int{10, 11, 12} {
		got, err := ScoreNativeAISpellAttack(records, []byte{0, 1, 2, 3, 4}, command, 15, gate)
		if err != nil || got != 48 {
			t.Fatalf("command%d score=%d err=%v", command, got, err)
		}
	}
	if _, err := ScoreNativeAISpellAttack(records, []byte{5}, 10, 15, gate); err == nil {
		t.Fatal("out-of-range target reached the raw record callback")
	}
}

func TestScoreNativeAISpellRecoveryUsesHalfAndRawBit(t *testing.T) {
	records := make([]byte, nativeRecordSize)
	binary.LittleEndian.PutUint16(records[0x40:], 4)
	binary.LittleEndian.PutUint16(records[0x42:], 10)
	records[0x34] = 1
	got, err := ScoreNativeAISpellRecovery(records, []byte{0}, 13)
	if err != nil || got != 6 {
		t.Fatalf("score=%d err=%v", got, err)
	}
	binary.LittleEndian.PutUint16(records[0x40:], 2)
	got, err = ScoreNativeAISpellRecovery(records, []byte{0}, 13)
	if err != nil || got != 16 {
		t.Fatalf("below-third score=%d err=%v", got, err)
	}
	binary.LittleEndian.PutUint16(records[0x40:], 5)
	got, err = ScoreNativeAISpellRecovery(records, []byte{0}, 13)
	if err != nil || got != 0 {
		t.Fatalf("half-health score=%d err=%v", got, err)
	}
}

func TestScoreNativeAISpellFlagUsesVerifiedOffsets(t *testing.T) {
	records := make([]byte, 2*nativeRecordSize)
	records[0x25] = 1
	records[nativeRecordSize+0x26] = 1
	got, err := ScoreNativeAISpellFlag(records, []byte{0, 1}, 20)
	if err != nil || got != 6 {
		t.Fatalf("ID20 score=%d err=%v", got, err)
	}
	got, err = ScoreNativeAISpellFlag(records, []byte{0, 1}, 21)
	if err != nil || got != 6 {
		t.Fatalf("ID21 score=%d err=%v", got, err)
	}
	got, err = ScoreNativeAISpellZeroFlag(records, []byte{0, 1}, 26)
	if err != nil || got != 4 {
		t.Fatalf("ID26 zero score=%d err=%v", got, err)
	}
	got, err = ScoreNativeAISpellZeroFlag(records, []byte{0, 1}, 27)
	if err != nil || got != 4 {
		t.Fatalf("ID27 zero score=%d err=%v", got, err)
	}
}

func TestScoreNativeAISpellZeroFlagUsesTransientOffsets(t *testing.T) {
	records := make([]byte, nativeRecordSize)
	for _, id := range []int{17, 18, 19} {
		got, err := ScoreNativeAISpellZeroFlag(records, []byte{0}, id)
		if err != nil || got != 3 {
			t.Fatalf("ID%d score=%d err=%v", id, got, err)
		}
	}
}

func TestScoreNativeAISpell22UsesRawGateAndBitRange(t *testing.T) {
	records := make([]byte, 2*nativeRecordSize)
	records[0x1a] = 1
	records[nativeRecordSize+0x1f-1] = 1
	records[nativeRecordSize+0x27] = 1
	got, err := ScoreNativeAISpell22(records, []byte{0, 1})
	if err != nil || got != 6 {
		t.Fatalf("score=%d err=%v", got, err)
	}
}

func TestScoreNativeAISpellRejectsMalformedCandidates(t *testing.T) {
	if _, err := ScoreNativeAISpellRecovery(make([]byte, nativeRecordSize-1), []byte{0}, 13); err == nil {
		t.Fatal("malformed record buffer accepted")
	}
	if _, err := ScoreNativeAISpellAttack(make([]byte, nativeRecordSize), []byte{1}, 0, 1, nil); err == nil {
		t.Fatal("out-of-range candidate accepted")
	}
}
