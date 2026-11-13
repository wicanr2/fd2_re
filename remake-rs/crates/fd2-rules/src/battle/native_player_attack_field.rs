//! Go 來源：`remake/internal/battle/native_player_attack_field.go`（34 行）。

use super::*;

impl State {
    /// NativePlayerAttackTargetField 重現玩家在指令環選攻擊之後（0x18f22 分支）的
    /// 目標格標記：0x18f6a 以 `0x14818(x, y, out, row+0xc, row+0xb, 0)` 在 0x4dbfc
    /// 基底旗標上做 flood，寫進 composition byte+3，0x115b6 選目標期間 0x11eee 就把
    /// 這些格染色；0x115b6 一回來 0x4dbfc 立刻清掉。row 是 0x1b83d 找到的已裝備
    /// 武器 record（0x18e00／0x18e0b 讀的兩個 byte，與 AI 的 0x14237 同一對）。
    /// 沒有原版物品表或沒有已裝備武器時回傳錯誤，呼叫端維持不標記。
    ///
    /// Go：`native_player_attack_field.go:11`，`func (s *State) NativePlayerAttackTargetField(u *Unit) ([]byte, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_player_attack_target_field(&self, u: Option<UnitIdx>) -> Result<Vec<u8>> {
        todo!("battle/native_player_attack_field.go:State.NativePlayerAttackTargetField")
    }
}
