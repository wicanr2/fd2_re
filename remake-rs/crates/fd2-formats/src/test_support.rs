//! 單元測試共用的原版輸入位置（只在測試編譯）。
//!
//! Go 測試用相對路徑找玩家自備的原版檔與分離素材包，缺檔就 `t.Skip`。Rust 測試在 Docker 內執行，
//! 一律由四個環境變數指定，名稱全 crate 統一：
//!
//! | 變數 | 內容 |
//! |---|---|
//! | `FD2_FLAME2_DIR` | 原版 `FLAME2` 目錄 |
//! | `FD2_ASSET_PACK` | 分離素材包根目錄（`fd2-original-b97caf22`） |
//! | `FD2_EXTRACTED_RAW_DIR` | `extracted/raw` |
//! | `FD2_REMAKE_ASSETS` | `remake/assets` |
//!
//! 缺檔時沿用 Go 的略過語意並印出「略過」；設 `FD2_REQUIRE_ORIGINAL=1` 時缺檔即失敗，避免把
//! 「沒跑到」當成通過。`tools/rust_formats_test.sh` 掛齊全部輸入並設定這個旗標。

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

/// 原版 `FLAME2/<name>`。
pub(crate) fn flame2(name: &str) -> Option<PathBuf> {
    input("FD2_FLAME2_DIR", name)
}

/// 分離素材包 `<rel>`。
pub(crate) fn asset_pack(rel: &str) -> Option<PathBuf> {
    input("FD2_ASSET_PACK", rel)
}
