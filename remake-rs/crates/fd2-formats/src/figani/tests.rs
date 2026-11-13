//! 逐條移植自 Go `remake/internal/figani/*_test.go`，期望值不改。每個子模組對應一個 Go 測試檔。
//!
//! 原版檔案的位置以環境變數提供（Go 版用相對路徑）：
//! - `FD2_FLAME2_DIR`：原版 `FLAME2/` 目錄（FIGANI.DAT、FDOTHER.DAT）。
//! - `FD2_TEST_ORIGINAL_PACK`：分離素材包 `generated-assets/fd2-original-b97caf22`（Go 版的
//!   `../../generated-assets/...` 相對路徑與 `FD2_TEST_ORIGINAL_PACK` 共用這一個）。
//! - `FD2_ASSET_PACK`：Go 版 `TestNativeCommand28Resource59HasEightImpactsAndOwnCues` 用的分離包。
//! - `FD2_REMAKE_ASSETS`：`remake/assets`（未設定時找 crate 旁的 `../../../remake/assets`）。
//!
//! 未設定時略過（Go 版以 `t.Skip` 處理缺檔）；`FD2_REQUIRE_ORIGINAL=1` 時缺檔即失敗。

use super::*;
use std::path::PathBuf;

mod blit_bounds_test;
mod figani_test;
mod native_command0_test;
mod native_command1_test;
mod native_command2_test;
mod native_command3_test;
mod native_command4_test;
mod native_command5_test;
mod native_command6_test;
mod native_command7_test;
mod native_command8_test;
mod native_command9_test;
mod native_command_compound_presentation_test;
mod native_command_indexed_entries_test;
mod native_command_rng_walk_test;
mod separated_original_pack_test;
mod separated_test;

fn require_or_skip(what: &str) {
    assert!(std::env::var_os("FD2_REQUIRE_ORIGINAL").is_none(), "缺少 {what}");
    eprintln!("略過：沒有 {what}");
}

/// 原版 `FLAME2/<name>`。
pub(super) fn original(name: &str) -> Option<String> {
    let dir = std::env::var_os("FD2_FLAME2_DIR").map(PathBuf::from);
    let path = dir.map(|d| d.join(name)).filter(|p| p.is_file());
    if path.is_none() {
        require_or_skip(&format!("原版 {name}"));
    }
    path.map(|p| p.to_string_lossy().into_owned())
}

fn env_dir(var: &str) -> Option<String> {
    let path = std::env::var_os(var).map(PathBuf::from).filter(|p| p.is_dir());
    if path.is_none() {
        require_or_skip(&format!("{var} 分離素材包"));
    }
    path.map(|p| p.to_string_lossy().into_owned())
}

/// 分離素材包根目錄（含 `animations/` 與 `raw/`）。
pub(super) fn original_pack() -> Option<String> {
    env_dir("FD2_ASSET_PACK")
}

pub(super) fn asset_pack() -> Option<String> {
    env_dir("FD2_ASSET_PACK")
}

/// `remake/assets/data/<name>`。
pub(super) fn remake_asset(name: &str) -> Option<String> {
    let base = std::env::var_os("FD2_REMAKE_ASSETS")
        .map(PathBuf::from)
        .unwrap_or_else(|| PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../../../remake/assets"));
    let path = base.join("data").join(name);
    if !path.is_file() {
        require_or_skip(&format!("remake/assets/data/{name}"));
        return None;
    }
    Some(path.to_string_lossy().into_owned())
}

/// 測試專用暫存目錄（對應 Go 的 `t.TempDir()`）。
pub(super) struct TempDir(pub PathBuf);

impl TempDir {
    pub(super) fn new(tag: &str) -> TempDir {
        let dir = std::env::temp_dir().join(format!("fd2-figani-{tag}-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        TempDir(dir)
    }

    pub(super) fn path(&self) -> String {
        self.0.to_string_lossy().into_owned()
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

/// 1x1 測試幀。
pub(super) fn unit_frame(pixel: u8) -> Frame {
    Frame { width: 1, height: 1, pixels: vec![pixel], mask: vec![1], ..Frame::default() }
}

/// 對應各測試檔的 `commandNTestAnimation`：`count` 張 1x1 幀，像素為 `index + offset`，
/// `header_byte2` 為幀數。
pub(super) fn test_animation(count: usize, offset: u8) -> Animation {
    let frames = (0..count).map(|i| unit_frame(i as u8 + offset)).collect();
    Animation { frames, header_byte2: count as u8, ..Animation::default() }
}

/// resolver 的回傳型別。
pub(super) type Resolved = std::result::Result<(u16, bool), ResolveError>;
