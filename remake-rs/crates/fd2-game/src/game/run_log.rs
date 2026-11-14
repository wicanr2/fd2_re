//! Go 來源：`remake/cmd/fd2/run_log.go`（74 行）。
//!
//! run_log.go — 玩家版執行紀錄。
//!
//! Windows 正式版以 -H=windowsgui 建置，沒有主控台；程式崩潰時 Go runtime 寫到
//! stderr 的 panic 與 fatal error 會整段遺失。這裡在啟動時把 stderr 與標準 log
//! 導向使用者資料夾的 fd2-run.log（前一次執行保留為 fd2-run.prev.log），並在
//! 每個 Update 尾端記錄 loadErr／狀態訊息的變化，讓玩家回報時能附上當時的錯誤。
//!
//! 骨架：Go 檔的 3 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`run_log.go:22`
///
/// Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。
pub(crate) static RUN_LOG_FILE: LazyLock<RwLock<Option<std::fs::File>>> =
    LazyLock::new(|| RwLock::new(Default::default()));

/// runLogTrace 記錄玩家看得到的錯誤與狀態訊息轉換；同一訊息不重複寫。
///
/// Go：`run_log.go:47`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct RunLogTrace {
    pub(crate) load_err: String,
    pub(crate) msg: String,
}

impl Game {
    /// Go：`run_log.go:51`，`func (g *Game) traceRunLog()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn trace_run_log(&mut self) {
        todo!("run_log.go:traceRunLog")
    }
}
