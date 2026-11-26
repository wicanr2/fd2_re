//! Go 來源：`remake/internal/battle/native_record_byte5.go`（14 行）。

use super::*;

/// SetNativeRecordByte5One reproduces 0x32975(index): it overwrites (rather
/// than ORs) byte +0x05 of the selected 0x50-byte runtime record with 1. The
/// byte's higher-level action meaning is intentionally not assigned here.
///
/// Go：`native_record_byte5.go:8`，`func SetNativeRecordByte5One(records []byte, index int) error`
pub fn set_native_record_byte5_one(records: &mut [u8], index: i64) -> Result<()> {
    if index < 0 || index >= records.len() as i64 / NATIVE_RECORD_SIZE {
        return Err(battle_err!("native record index {index} is out of bounds"));
    }
    records[(index * NATIVE_RECORD_SIZE + 5) as usize] = 1;
    Ok(())
}

impl State {
    /// MarkNativeUnitInactive 把 0x32975(index) 落到型別化狀態：raw +5 整個 byte 寫成 1，
    /// 同步原始執行期記錄，並把引擎投影 on_field 清掉。原版所有「這一列在不在場上」的
    /// 消費端都看 +5 bit0（占位格網 0x145CD、AI 目標掃描 0x12C60、預設勝負規則 0x205BE），
    /// 引擎這邊的 alive_count／result、繪圖與點選看 on_field，兩邊必須一起改。
    /// 寫入端有死亡程式的 mark_inactive 與敵方 AI 模式 7 抵達目的座標（0x13D60）。
    /// index 超出 units 時回傳錯誤，不寫半套。
    ///
    /// Go：`native_record_byte5.go`，`func (s *State) MarkNativeUnitInactive(u *Unit) error`
    pub fn mark_native_unit_inactive(&mut self, index: usize) -> Result<()> {
        if index >= self.units.len() {
            return Err(battle_err!("native 0x32975: unit is not in the runtime array"));
        }
        let u = &mut self.units[index];
        u.on_field = false;
        (u.native_record_byte5, u.has_native_record_byte5) = (1, true);
        self.sync_native_runtime_record(index as i64, 0x05, 1, 1);
        Ok(())
    }
}

#[cfg(test)]
mod native_record_byte5_test;
