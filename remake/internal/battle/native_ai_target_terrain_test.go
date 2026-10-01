package battle

import (
	"encoding/binary"
	"testing"
)

func TestNativeAIPhysicalTargetTerrainReadsTargetCell(t *testing.T) {
	actor, target := make([]byte, nativeRecordSize), make([]byte, nativeRecordSize)
	actor[0x20], target[0x1f] = 2, 4
	target[0], target[1], target[8] = 1, 1, 17
	binary.LittleEndian.PutUint16(actor[0x48:], 200)
	binary.LittleEndian.PutUint16(actor[0x4a:], 78)
	binary.LittleEndian.PutUint16(target[0x48:], 216)
	binary.LittleEndian.PutUint16(target[0x4a:], 154)
	binary.LittleEndian.PutUint16(target[0x40:], 157)
	state := &State{W: 3, H: 2, NativeTerrainMoveCodes: []byte{0, 0, 0, 0, 1, 2}}
	itemRows := make([]byte, NativeItemEffectRowSize)
	for _, destination := range []Cell{{X: 0, Y: 1}, {X: 2, Y: 1}} {
		got, err := state.nativeAIPhysicalScoreInput(NativeAIPhysicalAttackRawCandidate{
			ActorRecord: actor, TargetRecord: target, Destination: destination,
		}, itemRows)
		if err != nil {
			t.Fatal(err)
		}
		if got.TargetWord48 != 216 || got.TargetWord4A != 154 || got.ActorWord48 != 200 || got.ActorWord4A != 78 {
			t.Fatalf("目的地%+v 混入目標格地形：%+v", destination, got)
		}
	}
	// 相同 raw 閘門仍只對非零 sub_1F183 的 actor 套目的地地形。
	actor[0x1f] = 4
	got, err := state.nativeAIPhysicalScoreInput(NativeAIPhysicalAttackRawCandidate{
		ActorRecord: actor, TargetRecord: target, Destination: Cell{X: 2, Y: 1},
	}, itemRows)
	if err != nil || got.ActorWord48 != 190 || got.ActorWord4A != 85 || got.TargetWord48 != 216 || got.TargetWord4A != 154 {
		t.Fatalf("actor／target 地形來源未分開：%+v 錯誤=%v", got, err)
	}
}
