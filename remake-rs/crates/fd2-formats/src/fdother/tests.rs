//! 逐條移植自 Go `remake/internal/fdother` 核心與分離載入器、以及依賴它們的第三組的 `_test.go`，
//! 期望值不改。每個子模組對應一個 Go 測試檔；第三組只有一到三項測試的小檔合併在
//! `flow_schedules_test`。
//!
//! 需要原版檔的測試與 Go 版一樣，檔案不存在就略過；`FD2_REQUIRE_ORIGINAL=1` 時缺檔即失敗。
//! 原版目錄由 `FD2_FLAME2_DIR` 指定，分離素材包根目錄由 `FD2_ASSET_PACK` 指定。Go 測試讀的
//! `extracted/raw/<DAT>/<DAT>_NNN.bin` 是同一封存項目的抽出檔（Go 的 `TestFDOTHER054ArchiveLoader`
//! 就驗證兩者相等），這裡改由 `FD2_FLAME2_DIR` 的封存直接取出，不另設環境變數。

mod action_overlay_test;
mod chapter_aux_surface_test;
mod dialogue_grid_test;
mod flow_schedules_test;
mod frame_test;
mod frame_text_test;
mod lmi1_test;
mod lutbank_test;
mod map_parallax_test;
mod native_ch23_test;
mod palette_test;
mod preparation_roster_test;
mod range_overlay_test;
mod rawcells_test;
mod remap_test;
mod separated_action_cells_test;
mod separated_chapter_aux_surface_test;
mod separated_church_ui_test;
mod separated_frame_bank_test;
mod separated_item_panel_test;
mod separated_lmi1_bank_test;
mod separated_load_slots_test;
mod separated_lutbank_test;
mod separated_palette_test;
mod separated_range_overlay_test;
mod separated_sound_test;
mod spawn_intro_test;
mod split_slide_test;
mod sprite_bank_test;
mod terrain_lut_test;
mod terrain_phase_test;
mod unit_present_schedule_test;

use std::path::{Path, PathBuf};

fn require_original() -> bool {
    std::env::var_os("FD2_REQUIRE_ORIGINAL").is_some_and(|v| v == "1")
}

fn skip(what: &str) {
    assert!(!require_original(), "FD2_REQUIRE_ORIGINAL=1 但缺少 {what}");
    eprintln!("略過：沒有 {what}");
}

/// `FD2_FLAME2_DIR` 底下的原版檔；不存在時略過（或在要求原版檔時失敗）。
fn original(name: &str) -> Option<PathBuf> {
    let path = std::env::var_os("FD2_FLAME2_DIR").map(|d| PathBuf::from(d).join(name)).filter(|p| p.is_file());
    if path.is_none() {
        skip(&format!("原版 {name}"));
    }
    path
}

/// 原版封存的一筆原始項目（Go 測試的 `extracted/raw/...bin`）。
fn original_entry(dat: &str, resource: usize) -> Option<Vec<u8>> {
    original(dat).map(|path| crate::archive::read_resource(path, resource).unwrap())
}

/// `FD2_ASSET_PACK` 底下的分離素材路徑；`probe` 不存在時略過（或在要求原版檔時失敗）。
fn pack(rel: &str, probe: &str) -> Option<PathBuf> {
    let root = std::env::var_os("FD2_ASSET_PACK").map(|d| PathBuf::from(d).join(rel));
    let root = root.filter(|r| r.join(probe).exists());
    if root.is_none() {
        skip(&format!("分離素材 {rel}/{probe}"));
    }
    root
}

/// Go `t.TempDir()`：測試結束時刪除。
struct TempDir(PathBuf);

impl TempDir {
    fn new(tag: &str) -> Self {
        use std::sync::atomic::{AtomicUsize, Ordering};
        static NEXT: AtomicUsize = AtomicUsize::new(0);
        let n = NEXT.fetch_add(1, Ordering::Relaxed);
        let path = std::env::temp_dir().join(format!("fd2-fdother-{tag}-{}-{n}", std::process::id()));
        std::fs::create_dir_all(&path).unwrap();
        TempDir(path)
    }

    fn path(&self) -> &Path {
        &self.0
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}
