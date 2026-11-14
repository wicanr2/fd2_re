//! 移植自 `remake/internal/ending/montage_tail_test.go`。
//!
//! Go 測試以 `../../generated-assets/fd2-original-b97caf22` 找分離素材；Rust 經 `FD2_ASSET_PACK`。

use super::*;
use crate::test_support::{input, remake_asset, temp_dir};

pub(crate) fn tail() -> Option<MontageTail> {
    let path = remake_asset("endings/native_2c194_tail.json")?;
    Some(load_montage_tail(&path).unwrap())
}

fn pack(rel: &str) -> Option<String> {
    input("FD2_ASSET_PACK", rel).map(|p| p.to_string_lossy().into_owned())
}

/// 分離素材的 tail 視覺根目錄；先確認 Go 測試檢查的三個檔案存在。
pub(crate) fn tail_visual_paths() -> Option<MontageTailVisualPaths> {
    for rel in
        ["surfaces/BG_000/resource.json", "surfaces/TAI_000/resource.json", "animations/FIGANI_309/animation.json"]
    {
        input("FD2_ASSET_PACK", rel)?;
    }
    Some(MontageTailVisualPaths { surface_root: pack("surfaces")?, animation_root: pack("animations")? })
}

/// 分離素材的 tail 資源根目錄；先確認 `FDOTHER_058/animation.json` 存在。
pub(crate) fn tail_asset_paths() -> Option<MontageTailAssetPaths> {
    input("FD2_ASSET_PACK", "animations/FDOTHER_058/animation.json")?;
    Some(MontageTailAssetPaths {
        surface_root: pack("surfaces")?,
        palette_root: pack("palette")?,
        animation_root: pack("animations")?,
    })
}

#[test]
fn test_load_montage_tail_plans_provenance_bound_raw_tables() {
    let Some(tail) = tail() else { return };
    let entries = tail.plan().unwrap();
    let want_record0: [u8; 20] =
        [51, 110, 19, 105, 54, 117, 30, 123, 39, 127, 64, 81, 52, 125, 26, 115, 41, 91, 31, 126];
    let want_record1: [u8; 20] = [103, 20, 83, 28, 124, 38, 93, 34, 112, 44, 86, 53, 80, 55, 120, 36, 106, 60, 122, 50];
    let want_global: [u8; 20] = [4, 3, 51, 14, 25, 18, 40, 53, 22, 24, 28, 17, 30, 31, 50, 33, 34, 52, 36, 47];
    assert_eq!(entries.len(), want_record0.len(), "raw tail plan entries");
    for (i, entry) in entries.iter().enumerate() {
        let want = MontageTailEntry {
            index: i as i64,
            global540_ff: want_global[i],
            record0_byte6: tail_record_byte6(want_record0[i]),
            record0_byte7: want_record0[i],
            record1_byte6: tail_record_byte6(want_record1[i]),
            record1_byte7: want_record1[i],
        };
        assert_eq!(*entry, want, "raw tail plan entry {i}");
    }
}

#[test]
fn test_montage_tail_rejects_short_or_non_byte_raw_table() {
    let mut tail = MontageTail {
        r#loop: MontageTailLoop { count: 2, ..Default::default() },
        raw_tables: MontageTailRawTable { global540_ff: vec![1], record0_byte7: vec![1, 2], record1_byte7: vec![1, 2] },
        ..Default::default()
    };
    assert!(tail.plan().is_err(), "short raw table accepted");
    tail.raw_tables.global540_ff = vec![1, 0x100];
    assert!(tail.plan().is_err(), "non-byte raw table accepted");
}

#[test]
fn test_load_montage_tail_rejects_previous_schema() {
    let Some(path) = remake_asset("endings/native_2c194_tail.json") else { return };
    let raw = std::fs::read_to_string(&path).unwrap();
    let obsolete = raw.replacen(r#""schema_version": 2"#, r#""schema_version": 1"#, 1);
    assert_ne!(obsolete, raw, "tail asset did not contain schema version 2");
    let obsolete_path = temp_dir("tail-obsolete").join("native_2c194_tail_obsolete.json");
    std::fs::write(&obsolete_path, obsolete).unwrap();
    assert!(load_montage_tail(&obsolete_path.to_string_lossy()).is_err(), "obsolete tail schema was accepted");
}

#[test]
fn test_tail_record_byte6_uses_native_threshold() {
    assert_eq!(tail_record_byte6(0x4b), 2, "selector 0x4b");
    assert_eq!(tail_record_byte6(0x4c), 0, "selector 0x4c");
}

#[test]
fn test_montage_tail_visual_resource_plan_keeps_nonzero_branch_arithmetic_raw() {
    let Some(tail) = tail() else { return };
    let entries = tail.plan().unwrap();
    let resources = tail.plan_visual_resources().unwrap();
    assert_eq!(resources.len(), 20, "visual resource entries");
    for (index, resource) in resources.iter().enumerate() {
        let entry = &entries[index];
        let mut want_tai = i64::from(entry.global540_ff);
        if entry.record0_byte7 == 0x1a || entry.record0_byte7 == 0x36 || entry.record1_byte7 == 0x37 {
            want_tai = 3;
        }
        assert!(
            resource.index == index as i64
                && resource.tai == want_tai
                && resource.bg == i64::from(entry.global540_ff)
                && resource.record1_figani_base == i64::from(entry.record1_byte7) * 3
                && resource.record1_figani_aux == i64::from(entry.record1_byte7) * 3 + 1
                && resource.record0_figani_base == i64::from(entry.record0_byte7) * 3
                && resource.record0_figani_aux == i64::from(entry.record0_byte7) * 3 + 1,
            "visual resource entry {index}={resource:?}"
        );
    }
    for index in [4, 13, 14] {
        assert_eq!(resources[index].tai, 3, "raw TAI substitution entry {index}");
    }
}

#[test]
fn test_load_montage_tail_visual_sets_preflights_all_twenty_original_transactions() {
    let Some(paths) = tail_visual_paths() else { return };
    let Some(tail) = tail() else { return };
    let sets = load_montage_tail_visual_sets(&tail, &paths).unwrap();
    assert_eq!(sets.len(), 20, "tail visual sets");
    for (index, set) in sets.iter().enumerate() {
        assert!(
            set.plan.index == index as i64
                && set.tai.width > 0
                && set.bg.width > 0
                && set.bg.width <= WIDTH
                && set.bg.height > 0
                && set.bg.height <= HEIGHT - 50
                && set.record1_figani_base.is_some()
                && set.record1_figani_aux.is_some()
                && set.record0_figani_base.is_some()
                && set.record0_figani_aux.is_some(),
            "tail visual set {index} failed preflight: TAI={}x{} BG={}x{}",
            set.tai.width,
            set.tai.height,
            set.bg.width,
            set.bg.height
        );
    }
    for index in [4, 13, 14] {
        let s = &sets[index];
        assert!(s.plan.tai == 3 && s.tai.width == 10 && s.tai.height == 3, "tail TAI substitution set {index}");
    }
}

#[test]
fn test_load_montage_tail_visual_sets_fails_closed_before_partial_playback() {
    let Some(tail) = tail() else { return };
    let sets = load_montage_tail_visual_sets(&tail, &MontageTailVisualPaths::default());
    assert!(sets.is_err(), "missing visual archives returned sets");
}

#[test]
fn test_build_montage_tail_loader_baseline_uses_exact_selector1_e_deployment() {
    let Some(fdfield_root) = pack("fields/fdfield") else { return };
    input("FD2_ASSET_PACK", "sprites/fdicon/bank.json");
    let Some(fdicon_root) = pack("sprites/fdicon") else { return };
    let Some(tail) = tail() else { return };
    let Some(rows_path) = remake_asset("data/native_item_effect_rows.json") else { return };
    let items = battle::load_native_item_effect_row_prefix(&rows_path).unwrap();
    let paths = MontageTailLoaderPaths { fdfield_root, fdicon_root };
    let mut records = vec![save::PersistentRecord::default(); 2];
    for (index, record) in records.iter_mut().enumerate() {
        for (offset, b) in record.raw.iter_mut().enumerate() {
            *b = (index * 0x40 + offset) as u8;
        }
        record.raw[5] = 0x80 | index as u8;
        record.raw[7] = (4 + index * 5) as u8;
        for slot in 0..8 {
            record.raw[0x0a + slot * 2] &= !0x40;
        }
        record.raw[0x37..0x39].copy_from_slice(&(10 + index as u16).to_le_bytes());
        record.raw[0x39..0x3b].copy_from_slice(&(20 + index as u16).to_le_bytes());
        record.raw[0x3e..0x40].copy_from_slice(&(30 + index as u16).to_le_bytes());
    }
    let before = records.clone();

    let baseline = build_montage_tail_loader_baseline(&tail, &records, &paths, &items).unwrap();
    assert_eq!(baseline.runtime_count(), NATIVE_MONTAGE_TAIL_DEPLOY_RECORD_COUNT, "runtime count");
    let first = baseline.runtime_record(0).unwrap();
    let second = baseline.runtime_record(1).unwrap();
    assert!(
        first[0] == 17 && first[1] == 18 && second[0] == 1 && second[1] == 43,
        "selector 0x1e deployment coordinates={},{} / {},{}",
        first[0],
        first[1],
        second[0],
        second[1]
    );
    assert!(
        first[2] == 0 && second[2] == 1 && first[3] == 0 && first[4] == 0 && first[6] == 2 && first[0x31] == 0xff,
        "first loader overwrite={:x?}",
        &first[..0x32]
    );
    assert!(
        first[5] == before[0].raw[5]
            && first[7] == before[0].raw[7]
            && first[8] == before[0].raw[8]
            && second[5] == before[1].raw[5]
            && second[7] == before[1].raw[7]
            && second[8] == before[1].raw[8],
        "persistent raw provenance was not preserved"
    );
    for offset in 0x22..=0x27 {
        assert!(first[offset] == 0 && second[offset] == 0, "transient offset {offset:#x} was not cleared");
    }
    let inactive = baseline.runtime_record(2).unwrap();
    assert!(inactive[5] == 1 && inactive[7] == 0 && inactive[0x31] == 0, "inactive deployment slot={inactive:x?}");
    let mut pair = baseline.first_pair().unwrap();
    pair[0][0] ^= 0xff;
    let _ = pair; // Go 改的是回傳的複本；下面確認原記錄不受影響
    let still_first = baseline.runtime_record(0);
    assert!(
        still_first.as_ref().is_ok_and(|r| r[0] == first[0]),
        "baseline record unexpectedly aliases a returned pair: record={still_first:?}"
    );
    assert_eq!(records, before, "loader baseline mutated persistent source records");
}

#[test]
fn test_build_montage_tail_loader_baseline_fails_closed_before_admitting_tail_renderer() {
    let Some(tail) = tail() else { return };
    let paths = MontageTailLoaderPaths::default();
    assert!(
        build_montage_tail_loader_baseline(&tail, &[save::PersistentRecord::default()], &paths, &[]).is_err(),
        "one persistent record unexpectedly admitted"
    );
    let too_many = vec![save::PersistentRecord::default(); NATIVE_MONTAGE_TAIL_DEPLOY_RECORD_COUNT as usize + 1];
    assert!(
        build_montage_tail_loader_baseline(&tail, &too_many, &paths, &[]).is_err(),
        "32 persistent records unexpectedly admitted to 31 deployment slots"
    );
    let empty = MontageTailLoaderBaseline::default();
    assert!(empty.first_pair().is_err(), "empty loader baseline unexpectedly exposed a tail pair");
}

#[test]
fn test_montage_tail_assets_preserve_palette_frames_and_terminal_image() {
    let Some(paths) = tail_asset_paths() else { return };
    let Some(tail) = tail() else { return };
    let assets = load_montage_tail_assets(&tail, &paths).unwrap();
    assert!(
        assets.loop_palette.len() == 768
            && assets.loop_frames.len() == 20
            && assets.intro.width == WIDTH
            && assets.intro.height == HEIGHT
            && assets.r#final.width == WIDTH
            && assets.r#final.height == HEIGHT,
        "tail assets palette={} frames={} intro={}x{} final={}x{}",
        assets.loop_palette.len(),
        assets.loop_frames.len(),
        assets.intro.width,
        assets.intro.height,
        assets.r#final.width,
        assets.r#final.height
    );
    let mut compositor = new_indexed_compositor();
    compositor.palette.copy_from_slice(&assets.loop_palette);
    compositor.baseline.copy_from_slice(&assets.loop_palette);
    compositor.baseline_known = true;
    assets.present_final(Some(&mut compositor)).unwrap();
    let visible = compositor.vga.iter().filter(|&&p| p != 0).count();
    assert!(visible >= 100, "terminal frame has only {visible} visible indexed pixels");
}

#[test]
fn test_montage_tail_separated_assets_fail_closed_before_partial_playback() {
    let Some(tail) = tail() else { return };
    let assets = load_montage_tail_assets(&tail, &MontageTailAssetPaths::default());
    assert!(assets.is_err(), "missing separated ending assets returned frames");
}

#[test]
fn test_montage_tail_assets_keep_native_frame_table_geometry() {
    let Some(paths) = tail_asset_paths() else { return };
    let Some(tail) = tail() else { return };
    let assets = load_montage_tail_assets(&tail, &paths).unwrap();
    let want: [[i64; 4]; 20] = [
        [15, 7, 55, 98],
        [210, 85, 55, 98],
        [164, 9, 55, 98],
        [41, 91, 71, 98],
        [160, 36, 71, 98],
        [45, 11, 71, 98],
        [221, 33, 55, 98],
        [74, 74, 60, 115],
        [131, 28, 70, 98],
        [20, 11, 71, 98],
        [193, 83, 71, 98],
        [226, 12, 71, 98],
        [57, 84, 71, 98],
        [115, 67, 70, 98],
        [165, 13, 70, 115],
        [31, 86, 78, 98],
        [195, 73, 70, 115],
        [77, 7, 100, 115],
        [226, 5, 78, 115],
        [39, 71, 237, 28],
    ];
    assert_eq!(assets.loop_frames.len(), want.len(), "native frame count");
    for (i, frame) in assets.loop_frames.iter().enumerate() {
        assert_eq!([frame.x, frame.y, frame.width, frame.height], want[i], "native frame {i} geometry");
    }
}

#[test]
fn test_separated_montage_tail_assets_match_fixed_archive_blits() {
    let Some(archive) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let Some(tail) = tail() else { return };
    let Some(paths) = tail_asset_paths() else { return };
    let separated = load_montage_tail_assets(&tail, &paths).unwrap();
    let original = load_montage_tail_assets_archive(&tail, &archive.to_string_lossy()).unwrap();
    assert!(
        separated.loop_palette == original.loop_palette && separated.loop_frames.len() == original.loop_frames.len(),
        "separated ending palette or frame count differs from fixed archive"
    );
    let compare_blit = |label: &str, left: &fdother::Frame, right: &fdother::Frame| {
        let mut left_dst: Vec<u8> = (0..BYTES as usize).map(|i| (i * 37 + 11) as u8).collect();
        let mut right_dst = left_dst.clone();
        left.blit(&mut left_dst, WIDTH, -1).unwrap();
        right.blit(&mut right_dst, WIDTH, -1).unwrap();
        assert!(
            left.x == right.x
                && left.y == right.y
                && left.width == right.width
                && left.height == right.height
                && left_dst == right_dst,
            "{label} differs from fixed archive"
        );
    };
    compare_blit("FDOTHER#60", &separated.intro, &original.intro);
    compare_blit("FDOTHER#59", &separated.r#final, &original.r#final);
    for index in 0..separated.loop_frames.len() {
        compare_blit(&format!("FDOTHER#58 frame {index}"), &separated.loop_frames[index], &original.loop_frames[index]);
    }
}
