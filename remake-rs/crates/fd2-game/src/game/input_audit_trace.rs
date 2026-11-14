//! Go 來源：`remake/cmd/fd2/input_audit_trace.go`（567 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// recordInputAuditState 是唯讀的普通輸入觀測器；不指定關卡、亂數或遊戲狀態。
    /// 回放工具可等待真實 node／turn 邊界，避免固定延遲把按鍵送進下一個選單。
    ///
    /// Go：`input_audit_trace.go:18`，`func (g *Game) recordInputAuditState()`
    pub(crate) fn record_input_audit_state(&mut self) {
        todo!("input_audit_trace.go:recordInputAuditState")
    }
}
