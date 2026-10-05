package battle

import "fmt"

// NativeAI14EF0Tail 是 0x14ef0 在三個 raw score producer 完成後的
// 尾端路由。名稱只保留原始位址，不把 0x15311、0x1548e 或 0x15055
// 猜成攻擊、物品、法術或其他玩法效果。
type NativeAI14EF0Tail uint8

const (
	NativeAI14EF0NoTail NativeAI14EF0Tail = iota
	NativeAI14EF0Call1548E
	NativeAI14EF0Call15311
	NativeAI14EF0Call15055
)

// NativeAI14EF0Input 保存 0x14ef0 直接讀取的 raw 值。HasRaw* 欄位是
// provenance 閘門；normalized 分數、Camp、HP 或 command 名稱不可補入。
// CommandID／CommandWord 只有在 c4f==c23>c33 的分支會被消費，但該分支
// 仍要求兩者明示存在，避免把缺少 0x4e516 結果的資料誤派發。
type NativeAI14EF0Input struct {
	ScoreC4F int32
	ScoreC23 int32
	ScoreC33 int32

	CommandID   int32
	CommandWord uint16

	Record34     byte
	ActorWord48  uint16
	TargetWord4A uint16

	HasRawScoreC4F     bool
	HasRawScoreC23     bool
	HasRawScoreC33     bool
	HasRawCommandID    bool
	HasRawCommandWord  bool
	HasRawRecord34     bool
	HasRawActorWord48  bool
	HasRawTargetWord4A bool
}

// NativeAI14EF0Decision 分開保存尾端動作與 caller 消費的回傳值。
// 三者同分且至少6時，0x1504B仍回1，不能因 NoTail 進入移動後備。
// 證據：docs/data/ida/fd2_ai_14ef0_return_20261006.json。
type NativeAI14EF0Decision struct {
	Tail     NativeAI14EF0Tail
	Accepted bool
}

func PlanNativeAI14EF0Dispatch(in NativeAI14EF0Input) (NativeAI14EF0Decision, error) {
	tail, err := SelectNativeAI14EF0Tail(in)
	if err != nil {
		return NativeAI14EF0Decision{}, err
	}
	return NativeAI14EF0Decision{
		Tail: tail, Accepted: in.ScoreC4F >= 6 || in.ScoreC23 >= 6 || in.ScoreC33 >= 6,
	}, nil
}

// SelectNativeAI14EF0Tail 只保存 0x14ef0 的 raw 尾端決策樹。呼叫端必須
// 另行執行並驗證前置的 0x14237、0x1598a、0x1567e producer。本函式不修改
// 狀態，刻意不接 NextAIPlan 或正式 AI 執行器。
func SelectNativeAI14EF0Tail(in NativeAI14EF0Input) (NativeAI14EF0Tail, error) {
	if !in.HasRawScoreC4F || !in.HasRawScoreC23 || !in.HasRawScoreC33 ||
		!in.HasRawRecord34 || !in.HasRawActorWord48 || !in.HasRawTargetWord4A {
		return NativeAI14EF0NoTail, fmt.Errorf("native AI 0x14ef0 raw inputs are incomplete")
	}

	// 0x14f4c..0x14f5d：record+0x34 bit 0x40 與 unsigned words +0x48/+0x4a。
	flag40 := in.Record34 & 0x40
	delta := int32(in.ActorWord48) - int32(in.TargetWord4A)

	// 0x14f62..0x14f7f：沒有 producer 達到 signed threshold。
	if in.ScoreC4F < 6 && in.ScoreC23 < 6 && in.ScoreC33 < 6 {
		return NativeAI14EF0NoTail, nil
	}

	// 0x14f84..0x14f9b：c4f 是 strict maximum。
	if in.ScoreC4F > in.ScoreC23 && in.ScoreC4F > in.ScoreC33 {
		return NativeAI14EF0Call1548E, nil
	}

	// 0x14fa5..0x14fe5：c4f 與 c23 平手且高於 c33。0x4e516 提供 command
	// word；command IDs >= 11 改用 record+0x34 bit 0x40。
	if in.ScoreC4F == in.ScoreC23 && in.ScoreC4F > in.ScoreC33 {
		if !in.HasRawCommandID || !in.HasRawCommandWord {
			return NativeAI14EF0NoTail, fmt.Errorf("native AI 0x14ef0 command tie inputs are incomplete")
		}
		if in.CommandID >= 11 {
			if flag40 == 0 {
				return NativeAI14EF0Call15311, nil
			}
			return NativeAI14EF0Call1548E, nil
		}
		if int32(in.CommandWord) >= delta {
			return NativeAI14EF0Call15311, nil
		}
		return NativeAI14EF0Call1548E, nil
	}

	// 0x14feb..0x15006：c4f 與 c33 平手且高於 c23。raw bit 選擇
	// 0x1548e 或 0x15055。
	if in.ScoreC4F == in.ScoreC33 && in.ScoreC4F > in.ScoreC23 {
		if flag40 != 0 {
			return NativeAI14EF0Call1548E, nil
		}
		return NativeAI14EF0Call15055, nil
	}

	// 0x1500d..0x15039：c23 勝出（包括與 c33 平手），或 c33 是 strict
	// maximum。其餘平手直接通過，不呼叫尾端函式。
	if in.ScoreC23 > in.ScoreC4F && in.ScoreC23 >= in.ScoreC33 {
		return NativeAI14EF0Call15311, nil
	}
	if in.ScoreC33 > in.ScoreC4F && in.ScoreC33 > in.ScoreC23 {
		return NativeAI14EF0Call15055, nil
	}
	return NativeAI14EF0NoTail, nil
}
