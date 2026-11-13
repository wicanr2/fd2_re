//! Go `separated_original_pack_test.go`：完整唯讀分離包的資料回歸，並非 GUI、原版執行或新增動畫
//! 完成證據。與工具的固定 FIGANI 容器 SHA 核對合用，避免只信 manifest 的 PNG 雜湊。

use super::*;

// Go: TestSeparatedOriginalPackAllFIGANIFrames
#[test]
fn separated_original_pack_all_figani_frames() {
    let Some(pack) = original_pack() else { return };
    let (mut resources, mut frames) = (0, 0);
    for resource in 0..409 {
        let root = format!("{pack}/animations/FIGANI_{resource:03}");
        if !std::path::Path::new(&format!("{root}/animation.json")).exists() {
            continue;
        }
        let raw = std::fs::read(format!("{pack}/raw/FIGANI/FIGANI_{resource:03}.bin")).unwrap();
        let original = parse(&raw).unwrap_or_else(|e| panic!("resource {resource}: {e}"));
        let separated = load_separated_resource(&format!("{pack}/animations"), resource)
            .unwrap_or_else(|e| panic!("resource {resource}: {e}"));
        assert_eq!(
            original, separated,
            "resource {resource}: separated pixels/mask/descriptor differ from raw decoder"
        );
        resources += 1;
        frames += original.frames.len();
    }
    assert_eq!((resources, frames), (264, 2118), "fixed original coverage changed");
    eprintln!("{resources} resources / {frames} frames: raw decoder equals separated pixels, masks and descriptors");
}

// Go: TestSeparatedOriginalCompoundFDOTHERFrames
#[test]
fn separated_original_compound_fdother_frames() {
    let Some(pack) = original_pack() else { return };
    let mut frames = 0;
    for resource in 65..=68 {
        let raw = std::fs::read(format!("{pack}/raw/FDOTHER/FDOTHER_{resource:03}.bin")).unwrap();
        let original = parse(&raw).unwrap();
        let separated =
            load_separated_archive_resource(&format!("{pack}/animations"), "FDOTHER.DAT", resource).unwrap();
        assert_eq!(original, separated, "FDOTHER{resource}: raw and separated pixels/mask/descriptor differ");
        frames += original.frames.len();
    }
    assert_eq!(frames, 37, "fixed compound source frame count differs");
    eprintln!("4 FDOTHER resources / {frames} frames: original parser equals separated sources");
}
