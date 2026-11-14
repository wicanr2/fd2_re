//! Go 來源：`remake/cmd/fd2/input_audit_trace.go`（567 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// recordInputAuditState 是唯讀的普通輸入觀測器；不指定關卡、亂數或遊戲狀態。
    /// 回放工具可等待真實 node／turn 邊界，避免固定延遲把按鍵送進下一個選單。
    ///
    /// Go：`input_audit_trace.go:18`，`func (g *Game) recordInputAuditState()`
    ///
    /// 這是只在 `FD2_INPUT_AUDIT_LOG` 設定時寫 JSONL 的除錯觀測器，不改任何狀態。120 範圍表把
    /// 「截圖模式、效能追蹤、執行紀錄」歸為除錯工具（視需要放 `fd2-tools`）；這裡只移植入口的
    /// 環境變數閘門（第一章重播沒有設定，覆蓋率 0.8% 即此行），紀錄本體不在 R4。
    pub(crate) fn record_input_audit_state(&mut self) {
        if std::env::var_os("FD2_INPUT_AUDIT_LOG").is_none_or(|p| p.is_empty()) {
            return;
        }
        todo!("fd2-tools: input_audit_trace.go:recordInputAuditState（輸入稽核 JSONL，除錯工具）")
    }
}
