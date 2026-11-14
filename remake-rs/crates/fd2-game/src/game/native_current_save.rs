//! Go 來源：`remake/cmd/fd2/native_current_save.go`（157 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_current_save.go:14`，`func nativeCurrentSavePath() string`
pub(crate) fn native_current_save_path() -> String {
    // Writes require an explicit writable overlay. Never infer the pristine
    // original directory from FDOTHER and overwrite research input.
    std::env::var("FD2_NATIVE_SAVE").unwrap_or_default()
}

/// Go：`native_current_save.go:115`，`func replaceNativeCurrentSaveAtomic(path string, stored []byte) error`
pub(crate) fn replace_native_current_save_atomic(path: &str, stored: &[u8]) -> Result<()> {
    use std::io::Write as _;
    if path.is_empty() || stored.len() != fdsave::FILE_SIZE {
        return Err(game_err!("原版目前戰況存檔：原子寫入輸入無效"));
    }
    let wrap = |prefix: &'static str| move |err: std::io::Error| GameError::new(err.to_string()).wrap(prefix);
    let info = std::fs::metadata(path).map_err(wrap("原版目前戰況存檔：讀取目標屬性："))?;
    // Go `filepath.Dir`：沒有目錄部分時是 "."。
    let dir = match std::path::Path::new(path).parent() {
        Some(p) if !p.as_os_str().is_empty() => p.to_path_buf(),
        _ => std::path::PathBuf::from("."),
    };
    let (mut tmp, tmp_path) = create_temp_in(&dir, ".FD2.SAV.tmp-").map_err(wrap("原版目前戰況存檔：建立暫存檔："))?;
    // Go 的 defer：改名成功前離開都移除暫存檔。
    struct RemoveUnlessKept {
        path: std::path::PathBuf,
        remove: bool,
    }
    impl Drop for RemoveUnlessKept {
        fn drop(&mut self) {
            if self.remove {
                let _ = std::fs::remove_file(&self.path);
            }
        }
    }
    let mut guard = RemoveUnlessKept { path: tmp_path.clone(), remove: true };
    chmod_file(&tmp, file_mode_perm(&info)).map_err(wrap("原版目前戰況存檔：套用權限："))?;
    tmp.write_all(stored).map_err(wrap("原版目前戰況存檔：寫入暫存檔："))?;
    tmp.sync_all().map_err(wrap("原版目前戰況存檔：同步暫存檔："))?;
    drop(tmp);
    std::fs::rename(&tmp_path, path).map_err(wrap("原版目前戰況存檔：取代目標："))?;
    guard.remove = false;
    sync_dir(&dir);
    Ok(())
}

/// Go `info.Mode().Perm()`。非 Unix 平台 `chmod_file` 不使用這個值。
fn file_mode_perm(info: &std::fs::Metadata) -> u32 {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        info.permissions().mode() & 0o777
    }
    #[cfg(not(unix))]
    {
        let _ = info;
        0o644
    }
}

#[cfg(test)]
mod native_current_save_test;
