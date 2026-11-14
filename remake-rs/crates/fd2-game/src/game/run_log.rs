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
use std::io::Write as _;

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

/// Go `log.Printf` 寫到 `runLogFile`。Go 的 log 前綴（主機時間 `LstdFlags|Lmicroseconds`）
/// 不寫：`fd2-game` 不讀主機時間（KNOWN_DIFFERENCES `run-log-no-wall-timestamp`）。
fn run_log_printf(line: &str) {
    if let Some(file) = RUN_LOG_FILE.write().unwrap().as_mut() {
        let _ = writeln!(file, "{line}");
    }
}

impl Game {
    /// Go：`run_log.go:51`，`func (g *Game) traceRunLog()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn trace_run_log(&mut self) {
        if RUN_LOG_FILE.read().unwrap().is_none() {
            return;
        }
        if self.load_err != self.run_log.load_err {
            self.run_log.load_err = self.load_err.clone();
            if !self.load_err.is_empty() {
                run_log_printf(&format!(
                    "frame={} node={} loadErr={}",
                    self.frame,
                    self.run_log_node(),
                    battle::go_quote(&self.load_err)
                ));
            }
        }
        if self.msg != self.run_log.msg {
            self.run_log.msg = self.msg.clone();
            if !self.msg.is_empty() {
                run_log_printf(&format!(
                    "frame={} node={} msg={}",
                    self.frame,
                    self.run_log_node(),
                    battle::go_quote(&self.msg)
                ));
            }
        }
    }

    /// Go：`run_log.go:69`，`func (g *Game) runLogNode() string`（`r8`：只有寫檔分支用到，
    /// 第一章的重播沒有開 run log；這裡隨 `traceRunLog` 一起移植，簽名不在骨架）。
    fn run_log_node(&self) -> String {
        match &self.camp {
            None => String::from("-"),
            Some(camp) => camp.cur.clone(),
        }
    }
}
