//! 單元測試共用的原版輸入位置與小工具（只在測試編譯）。
//!
//! 環境變數與 `fd2-formats/src/test_support.rs` 相同，名稱全工作區統一：
//!
//! | 變數 | 內容 |
//! |---|---|
//! | `FD2_FLAME2_DIR` | 原版 `FLAME2` 目錄 |
//! | `FD2_ASSET_PACK` | 分離素材包根目錄（`fd2-original-b97caf22`） |
//! | `FD2_EXTRACTED_RAW_DIR` | `extracted/raw` |
//! | `FD2_REMAKE_ASSETS` | `remake/assets` |
//!
//! Go 測試用相對路徑（例如 `../../assets/...`）找輸入，缺檔就 `t.Skip`。Rust 測試缺檔時沿用略過
//! 語意並印出「略過」；設 `FD2_REQUIRE_ORIGINAL=1` 時缺檔即失敗，避免把「沒跑到」當成通過。
//! `tools/rust_campaign_test.sh` 掛齊全部輸入並設定這個旗標。

use std::path::PathBuf;

pub(crate) fn require_original() -> bool {
    std::env::var_os("FD2_REQUIRE_ORIGINAL").is_some_and(|v| v == "1")
}

/// 缺少輸入時略過；要求原版輸入時失敗。
pub(crate) fn skip(what: &str) {
    assert!(!require_original(), "FD2_REQUIRE_ORIGINAL=1 但缺少 {what}");
    eprintln!("略過：沒有 {what}");
}

/// `<var>/<rel>`，存在才回傳。
pub(crate) fn input(var: &str, rel: &str) -> Option<PathBuf> {
    let path = std::env::var_os(var).map(|d| PathBuf::from(d).join(rel)).filter(|p| p.exists());
    if path.is_none() {
        skip(&format!("{var}/{rel}"));
    }
    path
}

/// 受版控的 `remake/assets/<rel>`（Go 測試的 `../../assets/<rel>`）。
pub(crate) fn remake_asset(rel: &str) -> Option<String> {
    input("FD2_REMAKE_ASSETS", rel).map(|p| p.to_string_lossy().into_owned())
}

/// 受版控的 `docs/<rel>`（Go 測試的 `../../../docs/<rel>`）。以 crate 目錄往上三層定位，與 Go 的
/// 相對路徑相同；`tools/rust_campaign_test.sh` 把 `docs/` 唯讀掛在 `remake-rs` 的上一層。
pub(crate) fn docs_file(rel: &str) -> Option<String> {
    let path = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../../../docs").join(rel);
    if path.exists() {
        Some(path.to_string_lossy().into_owned())
    } else {
        skip(&format!("docs/{rel}"));
        None
    }
}

/// 小端 u16 寫入（Go `binary.LittleEndian.PutUint16(b[off:], v)`）。
pub(crate) fn put16(b: &mut [u8], off: usize, v: u16) {
    b[off..off + 2].copy_from_slice(&v.to_le_bytes());
}

/// 小端 u16 讀取（Go `binary.LittleEndian.Uint16(b[off:])`）。
pub(crate) fn get16(b: &[u8], off: usize) -> u16 {
    u16::from_le_bytes([b[off], b[off + 1]])
}

/// 測試用的暫存目錄（Go `t.TempDir()`）：每次呼叫建立新的空目錄，測試結束不自動刪除，
/// 放在系統暫存目錄下以名稱區分。
pub(crate) fn temp_dir(tag: &str) -> PathBuf {
    use std::sync::atomic::{AtomicUsize, Ordering};
    static NEXT: AtomicUsize = AtomicUsize::new(0);
    let n = NEXT.fetch_add(1, Ordering::Relaxed);
    let dir = std::env::temp_dir().join(format!("fd2-campaign-{}-{tag}-{n}", std::process::id()));
    let _ = std::fs::remove_dir_all(&dir);
    std::fs::create_dir_all(&dir).expect("建立暫存目錄");
    dir
}
