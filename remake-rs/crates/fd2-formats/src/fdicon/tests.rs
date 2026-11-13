//! 逐條移植自 Go `remake/internal/fdicon/*_test.go`，期望值不改。每個子模組對應一個 Go
//! 測試檔。需要原版檔的測試與 Go 版一樣，檔案不存在就略過；路徑可用
//! `FD2_FDICON_REAL_FLAME2`（含 FDICON.B24 的目錄）與 `FD2_FDICON_REAL_ASSETS`
//! （分離素材包根目錄）指定，否則沿用 Go 測試的相對位置。

mod binary_tick_test;
mod fdicon_test;
mod foreground_layer_test;
mod indexed_copy_test;
mod separated_fdshap_test;
mod separated_test;
mod terrain_test;
mod unit_layer_test;

use std::path::PathBuf;

/// 玩家提供的 FDICON.B24；Go 測試用 `../../../org_game/炎龍騎士團/FLAME2/FDICON.B24`。
fn original_fdicon_path() -> String {
    match std::env::var("FD2_FDICON_REAL_FLAME2") {
        Ok(dir) => format!("{dir}/FDICON.B24"),
        Err(_) => format!("{}/../../../org_game/炎龍騎士團/FLAME2/FDICON.B24", env!("CARGO_MANIFEST_DIR")),
    }
}

/// Go 測試用 `../../generated-assets/fd2-original-b97caf22`。
fn separated_assets_root() -> String {
    match std::env::var("FD2_FDICON_REAL_ASSETS") {
        Ok(dir) => dir,
        Err(_) => format!("{}/../../../remake/generated-assets/fd2-original-b97caf22", env!("CARGO_MANIFEST_DIR")),
    }
}

/// Go `t.TempDir()`：測試結束時刪除。
struct TempDir(PathBuf);

impl TempDir {
    fn new(tag: &str) -> Self {
        use std::sync::atomic::{AtomicUsize, Ordering};
        static NEXT: AtomicUsize = AtomicUsize::new(0);
        let n = NEXT.fetch_add(1, Ordering::Relaxed);
        let path = std::env::temp_dir().join(format!("fd2-fdicon-{tag}-{}-{n}", std::process::id()));
        std::fs::create_dir_all(&path).unwrap();
        TempDir(path)
    }

    fn path(&self) -> &str {
        self.0.to_str().unwrap()
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}
