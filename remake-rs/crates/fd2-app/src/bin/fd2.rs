//! 《炎龍騎士團 2》重製（Rust 版）主程式，對應 Go `remake/cmd/fd2` 的 `main()`（`main.go:13466`）。
//!
//! Go 的 `applyPackagedPlayerDefaults`（`packaged_startup.go:22`）在 `FD2_CAMPAIGN` 完全沒有設定時把它設成
//! `canonical`；明確設成空字串是測試與工具停用戰役的契約，不覆蓋。工作區禁用 `unsafe`，不能在行程內改環境變數，
//! 所以這裡以同一個執行檔、補上該變數重新啟動一次，結束碼照傳。

use std::process::ExitCode;

fn main() -> ExitCode {
    if std::env::var_os("FD2_CAMPAIGN").is_none() {
        let exe = std::env::current_exe().expect("取得執行檔路徑");
        let status = std::process::Command::new(exe)
            .args(std::env::args_os().skip(1))
            .env("FD2_CAMPAIGN", "canonical")
            .status()
            .expect("重新啟動");
        return ExitCode::from(status.code().unwrap_or(1).clamp(0, 255) as u8);
    }
    match fd2_app::shell::run() {
        bevy::app::AppExit::Success => ExitCode::SUCCESS,
        bevy::app::AppExit::Error(code) => ExitCode::from(code.get()),
    }
}
