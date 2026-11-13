//! 逐條移植自 Go `remake/internal/afm/separated_test.go`，期望值不改。
//!
//! 本機分離素材包同樣 gitignore；Rust 版另外接受環境變數 `FD2_ASSET_PACK`（其下 `animations/`）。

use super::super::tests::ani_path;
use super::super::{FRAME_BYTES, decode_resource};
use super::*;

fn separated_animation_root() -> Option<String> {
    let has_metadata = |root: &str| std::path::Path::new(root).join("ANI_000").join("animation.json").is_file();
    if let Ok(dir) = std::env::var("FD2_ASSET_PACK") {
        let p = format!("{dir}/animations");
        let found = has_metadata(&p).then_some(p);
        if found.is_none() {
            crate::test_support::skip("FD2_ASSET_PACK/animations");
        }
        return found;
    }
    if crate::test_support::require_original() {
        crate::test_support::skip("FD2_ASSET_PACK");
    }
    let manifest = env!("CARGO_MANIFEST_DIR");
    [
        "../../../remake/generated-assets/fd2-original-b97caf22/animations",
        "../../../../remake/generated-assets/fd2-original-b97caf22/animations",
    ]
    .iter()
    .map(|rel| format!("{manifest}/{rel}"))
    .find(|p| has_metadata(p))
}

/// Go `strings.TrimRight(s, "\x00 ")`；ASCII cutset 走逐位元組路徑。
fn trim_right_nul_space(s: &[u8]) -> &[u8] {
    let end = s.iter().rposition(|&b| b != 0 && b != b' ').map_or(0, |i| i + 1);
    &s[..end]
}

/// Go `t.TempDir()` 的替代：每個測試自己的目錄，結束時刪除。
struct TempDir(std::path::PathBuf);

impl TempDir {
    fn new(tag: &str) -> Self {
        let nanos = std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).unwrap().as_nanos();
        let dir = std::env::temp_dir().join(format!("fd2-afm-{tag}-{}-{nanos}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        TempDir(dir)
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

#[test]
fn separated_ani_exactly_matches_archive_oracle() {
    let (Some(archive), Some(root)) = (ani_path(), separated_animation_root()) else {
        eprintln!("玩家自備 ANI.DAT 或本機分離素材包不存在");
        return;
    };
    let mut total = 0;
    for (resource, &want) in EXPECTED_FRAME_COUNTS.iter().enumerate() {
        let oracle =
            decode_resource(&archive, resource as i64).unwrap_or_else(|e| panic!("archive ANI #{resource}: {e}"));
        let separated = load_separated_resource(&root, resource as i64)
            .unwrap_or_else(|e| panic!("separated ANI #{resource}: {e}"));
        assert!(
            separated.title == trim_right_nul_space(&oracle.title) && separated.header_frames == want,
            "ANI #{resource} metadata mismatch"
        );
        for frame in 0..want {
            assert_eq!(separated.indexed_frames[frame].len(), FRAME_BYTES);
            assert!(
                separated.indexed_frames[frame] == oracle.indexed_frames[frame],
                "ANI #{resource} frame {frame} indexed pixels differ"
            );
            assert!(
                separated.palettes[frame] == oracle.palettes[frame],
                "ANI #{resource} frame {frame} palette differs"
            );
        }
        total += want;
    }
    assert_eq!(total, 289, "total ANI frames");
    assert!(!std::path::Path::new(&root).join("ANI_009").exists(), "zero-length ANI #9 must not be exported");
}

#[test]
fn separated_ani_fails_closed_on_invalid_metadata() {
    let root = TempDir::new("separated");
    let directory = root.0.join("ANI_000");
    std::fs::create_dir_all(&directory).unwrap();
    std::fs::write(directory.join("animation.json"), b"{}\n").unwrap();
    let root_str = root.0.to_str().unwrap();
    assert!(load_separated_resource(root_str, 0).is_err(), "invalid metadata must fail closed");
    assert!(load_separated_resource(root_str, 9).is_err(), "zero-length tail resource must never load");
}
