package battle

import "fmt"

// NativePlayerAttackTargetField 重現玩家在指令環選攻擊之後（0x18f22 分支）的
// 目標格標記：0x18f6a 以 `0x14818(x, y, out, row+0xc, row+0xb, 0)` 在 0x4dbfc
// 基底旗標上做 flood，寫進 composition byte+3，0x115b6 選目標期間 0x11eee 就把
// 這些格染色；0x115b6 一回來 0x4dbfc 立刻清掉。row 是 0x1b83d 找到的已裝備
// 武器 record（0x18e00／0x18e0b 讀的兩個 byte，與 AI 的 0x14237 同一對）。
// 沒有原版物品表或沒有已裝備武器時回傳錯誤，呼叫端維持不標記。
func (s *State) NativePlayerAttackTargetField(u *Unit) ([]byte, error) {
	if s == nil || u == nil {
		return nil, fmt.Errorf("native player attack field: state or unit unavailable")
	}
	slot, ok := nativeEquippedWeaponSlot(u)
	if !ok {
		return nil, fmt.Errorf("native player attack field: no equipped weapon")
	}
	offset, err := NativeItemEffectRowOffset(u.Inventory[slot])
	if err != nil {
		return nil, fmt.Errorf("native player attack field: %w", err)
	}
	if offset+NativeItemEffectRowSize > len(s.nativeFutureItemRows) {
		return nil, fmt.Errorf("native player attack field: item table unavailable")
	}
	row := s.nativeFutureItemRows[offset : offset+NativeItemEffectRowSize]
	flags, err := s.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	return NativeCommandTargetFieldBytes(
		s.W, s.H, Cell{X: u.X, Y: u.Y}, int(row[0x0c]), int(row[0x0b]), flags,
	)
}
