package battle

import "fmt"

// NativeResultRule 保存已閉合的章節結果覆寫；Source 只是原始 writer 的
// 出處，正式執行依具型別條件，不依 FD2.EXE 位址分派。
// READY 契約與主證據見56及fd2_ch13_result_conditions_20261001.json、
// fd2_ch15_result_conditions_20261002.json；原版無對白列的Actions為空。
type NativeResultRule struct {
	ID                 string   `json:"id"`
	Source             string   `json:"source"`
	RecordsAllInactive []int    `json:"records_all_inactive"`
	RoundGreaterThan   *int     `json:"round_greater_than,omitempty"`
	Code               int      `json:"code"`
	Actions            []Action `json:"actions"`
}

func (r NativeResultRule) Validate() error {
	if r.ID == "" || r.Source == "" || r.Code != 1 || len(r.RecordsAllInactive) == 0 {
		return fmt.Errorf("原生結果列缺少來源、記錄或已閉合的code1")
	}
	if r.RoundGreaterThan != nil && *r.RoundGreaterThan < 0 {
		return fmt.Errorf("原生結果列回合門檻為負值")
	}
	seen := map[int]bool{}
	for _, index := range r.RecordsAllInactive {
		if index < 0 || seen[index] {
			return fmt.Errorf("原生結果列記錄索引%d無效或重複", index)
		}
		seen[index] = true
	}
	for i, action := range r.Actions {
		ref := action.NativeDialogueRef
		if action.Type != "dialogue" || action.NativeSource == "" || action.NativeTextIndex == nil ||
			ref == nil || *action.NativeTextIndex != ref.StringIndex || ref.Script == "" ||
			ref.SourceDAT == "" || ref.SceneIndex < 0 || ref.Line < 0 || len(ref.Pages) == 0 ||
			action.NativeWhen != nil || action.NativeEventID != nil {
			return fmt.Errorf("原生結果列動作%d缺少一致的獨立對白來源", i)
		}
	}
	return nil
}

// Match 在該列真正執行時讀 raw 狀態；回合閘先於記錄查詢，不能因
// 尚未生成的59而提前拒絕，也不能用HP代替sub_3453E的+5 bit0。
func (r NativeResultRule) Match(st *State) (bool, error) {
	if err := r.Validate(); err != nil {
		return false, err
	}
	if st == nil {
		return false, fmt.Errorf("原生結果列缺少戰場")
	}
	if r.RoundGreaterThan != nil {
		if st.NativeRoundCounter <= 0 {
			return false, fmt.Errorf("原生結果列缺少原始回合計數")
		}
		if st.NativeRoundCounter <= *r.RoundGreaterThan {
			return false, nil
		}
	}
	all := true
	for _, index := range r.RecordsAllInactive {
		if index >= len(st.Units) || st.Units[index] == nil || !st.Units[index].HasNativeRecordByte5 {
			return false, fmt.Errorf("原生結果列缺少記錄%d的raw +5", index)
		}
		if st.Units[index].NativeRecordByte5&1 == 0 {
			all = false
		}
	}
	return all, nil
}

// NativeDefaultResultCode 是sub_205BE的已閉合三值基礎規則：僅掃已生成的
// runtime陣列；raw+6=0且+5 bit0=0使結果2改0，最後記錄0的bit0使其改1。
func NativeDefaultResultCode(st *State) (int, error) {
	if st == nil || len(st.Units) == 0 {
		return 0, fmt.Errorf("原生基礎結果缺少runtime記錄")
	}
	code := 2
	for index, u := range st.Units {
		if u == nil || !u.HasNativeRecordByte5 || !u.HasNativeRecordByte6 {
			return 0, fmt.Errorf("原生基礎結果缺少記錄%d的raw +5／+6", index)
		}
		if u.NativeRecordByte6 == 0 && u.NativeRecordByte5&1 == 0 {
			code = 0
		}
	}
	if st.Units[0].NativeRecordByte5&1 != 0 {
		code = 1
	}
	return code, nil
}
