package battle

import "fmt"

// SetNativeRecordByte5One reproduces 0x32975(index): it overwrites (rather
// than ORs) byte +0x05 of the selected 0x50-byte runtime record with 1. The
// byte's higher-level action meaning is intentionally not assigned here.
func SetNativeRecordByte5One(records []byte, index int) error {
	if index < 0 || index >= len(records)/nativeRecordSize {
		return fmt.Errorf("native record index %d is out of bounds", index)
	}
	records[index*nativeRecordSize+5] = 1
	return nil
}

// MarkNativeUnitInactive 把 0x32975(index) 落到型別化狀態：raw +5 整個 byte 寫成 1，
// 同步原始執行期記錄，並把引擎投影 OnField 清掉。原版所有「這一列在不在場上」的
// 消費端都看 +5 bit0（占位格網 0x145CD、AI 目標掃描 0x12C60、預設勝負規則 0x205BE），
// 引擎這邊的 AliveCount／Result、繪圖與點選看 OnField，兩邊必須一起改。
// 寫入端有死亡程式的 mark_inactive 與敵方 AI 模式 7 抵達目的座標（0x13D60）。
// u 不在 s.Units 裡時回傳錯誤，不寫半套。
func (s *State) MarkNativeUnitInactive(u *Unit) error {
	if s == nil || u == nil {
		return fmt.Errorf("native 0x32975: missing state or unit")
	}
	for index, candidate := range s.Units {
		if candidate == u {
			s.markNativeRecordInactive(index)
			return nil
		}
	}
	return fmt.Errorf("native 0x32975: unit is not in the runtime array")
}

func (s *State) markNativeRecordInactive(index int) {
	u := s.Units[index]
	u.OnField = false
	u.NativeRecordByte5, u.HasNativeRecordByte5 = 1, true
	s.syncNativeRuntimeRecord(index, 0x05, 1, 1)
}
