//! F 組測試共用：在子行程裡以指定的環境變數重跑一個測試（只在測試編譯）。
//!
//! Go 測試以 `t.Setenv` 設定 `XDG_DATA_HOME`、`FD2_NATIVE_SAVE`。Rust 工作區禁止 `unsafe`
//! （`std::env::set_var` 在 2024 版是 `unsafe`），平行執行的測試也共用同一份環境，所以要改環境變數的
//! 測試改成重新執行測試執行檔本身、只跑這一個測試，並把環境變數交給子行程。
//!
//! 用法：`let Some(dir) = f_child_env(module_path!(), "測試名", |dir| vec![...]) else { return };`。
//! 父行程建立暫存目錄、以 `envs(dir)` 與 `FD2_GAME_F_TEST_DIR=dir` 啟動子行程並斷言它通過，回傳 `None`；
//! 子行程回傳同一個目錄，測試本體在子行程裡執行。

use std::ffi::OsString;
use std::path::{Path, PathBuf};

const CHILD_DIR: &str = "FD2_GAME_F_TEST_DIR";

pub(crate) fn f_child_env(
    module: &str,
    name: &str,
    envs: impl Fn(&Path) -> Vec<(&'static str, OsString)>,
) -> Option<PathBuf> {
    if let Some(dir) = std::env::var_os(CHILD_DIR) {
        return Some(PathBuf::from(dir));
    }
    let dir = crate::test_support::temp_dir(name);
    let path = format!("{}::{name}", module.strip_prefix("fd2_game::").unwrap_or(module));
    let exe = std::env::current_exe().expect("測試執行檔路徑");
    let mut command = std::process::Command::new(exe);
    command
        .args([path.as_str(), "--exact", "--include-ignored", "--nocapture", "--test-threads=1"])
        .env(CHILD_DIR, &dir);
    for (key, value) in envs(&dir) {
        command.env(key, value);
    }
    let output = command.output().expect("啟動子行程");
    let stdout = String::from_utf8_lossy(&output.stdout);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(output.status.success(), "子行程 {path} 失敗：\n{stdout}\n{stderr}");
    assert!(stdout.contains("1 passed"), "子行程沒有執行 {path}：\n{stdout}\n{stderr}");
    None
}
