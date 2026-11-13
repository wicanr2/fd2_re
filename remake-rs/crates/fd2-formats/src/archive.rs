//! 原版 `LLLLLL` .DAT 容器的資源目錄。
//!
//! 移植自 Go `remake/internal/fdother/frame.go` 的 `ArchiveEntry`、`ArchiveDataResourceCount`、
//! `ArchiveResourceCount`、`ReadResource`、`ReadNestedResource`。這層對應原版 `0x111ba` 只讀
//! 容器、不轉換也不解壓的邊界；FDOTHER、FDTXT、FIGANI、BG 等多個 .DAT 共用，因此獨立成模組，
//! `fdother` 模組再轉出同名函式。錯誤訊息沿用 Go 版的 `fdother:` 前綴與文字。

use std::fmt;
use std::path::{Path, PathBuf};

#[derive(Debug)]
pub enum ArchiveError {
    Io { path: PathBuf, source: std::io::Error },
    MissingMagic,
    InvalidDirectory,
    OutOfRange,
    InvalidBounds,
    Nested { resource: usize, nested: usize, source: Box<ArchiveError> },
}

impl fmt::Display for ArchiveError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            // Go 版直接回傳 os.ReadFile 的錯誤；I/O 錯誤文字因平台而異，不納入黃金比對。
            ArchiveError::Io { path, source } => write!(f, "open {}: {source}", path.display()),
            ArchiveError::MissingMagic => f.write_str("fdother: missing LLLLLL archive magic"),
            ArchiveError::InvalidDirectory => f.write_str("fdother: invalid archive directory"),
            ArchiveError::OutOfRange => f.write_str("fdother: archive resource is out of range"),
            ArchiveError::InvalidBounds => f.write_str("fdother: invalid archive resource bounds"),
            ArchiveError::Nested { resource, nested, source } => {
                write!(f, "fdother: nested resource {resource}/{nested}: {source}")
            }
        }
    }
}

impl std::error::Error for ArchiveError {}

pub type Result<T> = std::result::Result<T, ArchiveError>;

fn le_u32(data: &[u8], at: usize) -> usize {
    u32::from_le_bytes([data[at], data[at + 1], data[at + 2], data[at + 3]]) as usize
}

/// 驗證容器標頭並回傳第一個資源的起點（即目錄結尾）。
fn directory_end(data: &[u8]) -> Result<usize> {
    if data.len() < 10 || &data[..6] != b"LLLLLL" {
        return Err(ArchiveError::MissingMagic);
    }
    let first = le_u32(data, 6);
    if first < 10 || first > data.len() || !(first - 6).is_multiple_of(4) {
        return Err(ArchiveError::InvalidDirectory);
    }
    Ok(first)
}

/// 驗證一份已載入的容器並回傳目錄筆數。巢狀 FDOTHER 資源用這個邊界計數，不必寫暫存檔。
pub fn archive_data_resource_count(data: &[u8]) -> Result<usize> {
    Ok((directory_end(data)? - 6) / 4)
}

/// 從容器位元組取出一筆原始資源。FDOTHER #81 本身就是巢狀容器，所以這個邊界公開給呼叫端，
/// 讓它驗證巢狀出處，而不是把內容當成一般的圖格表。
pub fn archive_entry(data: &[u8], resource: usize) -> Result<&[u8]> {
    let first = directory_end(data)?;
    let count = (first - 6) / 4;
    if resource >= count {
        return Err(ArchiveError::OutOfRange);
    }
    let start = le_u32(data, 6 + 4 * resource);
    let end = if resource + 1 < count { le_u32(data, 6 + 4 * (resource + 1)) } else { data.len() };
    if start < first || start > end || end > data.len() {
        return Err(ArchiveError::InvalidBounds);
    }
    Ok(&data[start..end])
}

fn read(path: &Path) -> Result<Vec<u8>> {
    std::fs::read(path).map_err(|source| ArchiveError::Io { path: path.to_path_buf(), source })
}

/// 回傳 .DAT 容器的資源數量。
pub fn archive_resource_count(path: impl AsRef<Path>) -> Result<usize> {
    archive_data_resource_count(&read(path.as_ref())?)
}

/// 讀出一筆原始資源，不假設它是圖格表；例如 FDOTHER #4 是 FDTXT 點陣字型。
pub fn read_resource(path: impl AsRef<Path>, resource: usize) -> Result<Vec<u8>> {
    Ok(archive_entry(&read(path.as_ref())?, resource)?.to_vec())
}

/// 保留戰鬥音效庫這類兩層 `0x111BA` 邊界（例如 command0 的 FDOTHER #82/sub1），只回傳原始位元組；
/// 取樣率、聲道與播放仍由呼叫端負責。
pub fn read_nested_resource(path: impl AsRef<Path>, resource: usize, nested: usize) -> Result<Vec<u8>> {
    let outer = read_resource(path, resource)?;
    match archive_entry(&outer, nested) {
        Ok(entry) => Ok(entry.to_vec()),
        Err(source) => Err(ArchiveError::Nested { resource, nested, source: Box::new(source) }),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// 原版檔案目錄：`FD2_FLAME2_DIR`；未設定時跳過（Go 版以 t.Skip 處理缺檔），
    /// `FD2_REQUIRE_ORIGINAL=1` 時缺檔即失敗。
    pub(crate) fn original(name: &str) -> Option<PathBuf> {
        crate::test_support::flame2(name)
    }

    fn archive(entries: &[&[u8]]) -> Vec<u8> {
        let first = 6 + 4 * entries.len();
        let mut out = b"LLLLLL".to_vec();
        let mut at = first;
        for e in entries {
            out.extend_from_slice(&(at as u32).to_le_bytes());
            at += e.len();
        }
        for e in entries {
            out.extend_from_slice(e);
        }
        out
    }

    // Go: TestReadOriginalCommand0NestedSample
    #[test]
    fn read_original_command0_nested_sample() {
        let Some(path) = original("FDOTHER.DAT") else { return };
        assert_eq!(read_nested_resource(path, 82, 1).unwrap().len(), 13424);
    }

    // Go: TestReadNestedResourceRejectsNonContainerAndOutOfRange
    #[test]
    fn read_nested_resource_rejects_non_container_and_out_of_range() {
        let mut data = b"LLLLLL".to_vec();
        data.extend_from_slice(&10u32.to_le_bytes());
        data.extend_from_slice(b"not nested");
        let dir = std::env::temp_dir().join(format!("fd2-archive-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("outer.dat");
        std::fs::write(&path, data).unwrap();
        let err = read_nested_resource(&path, 0, 0).unwrap_err();
        assert_eq!(err.to_string(), "fdother: nested resource 0/0: fdother: missing LLLLLL archive magic");
        std::fs::remove_dir_all(dir).unwrap();
    }

    // Go: TestFDOTHER081ArchiveShape
    #[test]
    fn fdother_081_archive_shape() {
        let Some(path) = original("FDOTHER.DAT") else { return };
        let entry = read_resource(path, 0x51).unwrap();
        assert!(entry.len() >= 10 && &entry[..6] == b"LLLLLL" && le_u32(&entry, 6) == 0x12);
        for (resource, want) in [0x22ce, 0x2636].into_iter().enumerate() {
            assert_eq!(archive_entry(&entry, resource).unwrap().len(), want, "#81 nested entry {resource}");
        }
        assert!(archive_entry(&entry, 2).unwrap().is_empty(), "#81 nested terminator entry");
        assert!(archive_entry(&entry, 3).is_err(), "#81 accepted a fourth nested entry");
    }

    #[test]
    fn rejects_malformed_directories() {
        assert!(matches!(archive_entry(b"LLLLL", 0), Err(ArchiveError::MissingMagic)));
        let mut bad = archive(&[b"ab"]);
        bad[6] = 11; // (first-6) 不是 4 的倍數
        assert!(matches!(archive_entry(&bad, 0), Err(ArchiveError::InvalidDirectory)));
        let good = archive(&[b"ab", b"", b"cde"]);
        assert_eq!(archive_data_resource_count(&good).unwrap(), 3);
        assert_eq!(archive_entry(&good, 0).unwrap(), b"ab");
        assert_eq!(archive_entry(&good, 1).unwrap(), b"");
        assert_eq!(archive_entry(&good, 2).unwrap(), b"cde");
        assert!(matches!(archive_entry(&good, 3), Err(ArchiveError::OutOfRange)));
        let mut swapped = good.clone();
        swapped[10..14].copy_from_slice(&40u32.to_le_bytes());
        assert!(matches!(archive_entry(&swapped, 0), Err(ArchiveError::InvalidBounds)));
    }
}
