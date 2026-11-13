//! 移植自 `lmi1_test.go`。

use super::original;
use crate::fdother::*;

/// Go `native_command_heal_tail.go` 的常數（第二組移植範圍）；這裡只為本測試照抄同值。
const NATIVE_COMMAND_HEAL_TAIL_EFFECT_RESOURCE: i64 = 6;
const NATIVE_COMMAND_HEAL_TAIL_DIGIT_RESOURCE: i64 = 5;
const NATIVE_COMMAND_HEAL_TAIL_EFFECT_START: usize = 0x39;
const NATIVE_COMMAND_HEAL_TAIL_EFFECT_FRAMES: usize = 7;
const NATIVE_COMMAND_HEAL_TAIL_DIGIT_BIAS: usize = 0x69;

fn lmi1_header(count: u16, offsets: &[u32]) -> Vec<u8> {
    let mut data = b"LMI1".to_vec();
    data.extend_from_slice(&count.to_le_bytes());
    for off in offsets {
        data.extend_from_slice(&off.to_le_bytes());
    }
    data
}

#[test]
fn parse_lmi1_native_codec() {
    // 兩筆：逐字 0、逐字 0xc0，然後是 7 的 0xc3 重複。
    let mut data = lmi1_header(2, &[14, 22]);
    data.extend_from_slice(&[3, 0, 1, 0, 0, 0xc0, 1, 2]);
    data.extend_from_slice(&[2, 0, 1, 0, 0xc2, 7]);
    let entries = parse_lmi1(&data).unwrap();
    assert!(entries.len() == 2 && entries[0].width == 3 && entries[0].height == 1, "entries={entries:?}");
    assert_eq!(entries[0].pixels, [0, 0xc0, 1]);
    assert_eq!(entries[1], LMI1Entry { width: 2, height: 1, pixels: vec![7, 7] });
}

#[test]
fn parse_lmi1_rejects_malformed_codec() {
    let mut data = lmi1_header(1, &[10]);
    data.extend_from_slice(&2u16.to_le_bytes());
    data.extend_from_slice(&1u16.to_le_bytes());
    assert!(parse_lmi1(&data).is_err(), "truncated LMI1 stream must fail closed");
}

#[test]
fn parse_opaque_run_cell_native_codec() {
    // 不透明的 0x4e8af 路徑沒有 LMI1 容器：項目直接以 u16 width/height 開頭，接同一種高位
    // 連續段串流。
    let entry = parse_opaque_run_cell(&[3, 0, 1, 0, 0, 0xc0, 0xc2, 7]).unwrap();
    assert!(entry.width == 3 && entry.height == 1, "geometry");
    assert_eq!(entry.pixels, [0, 0xc0, 7]);
}

#[test]
fn parse_opaque_run_cell_rejects_malformed_input() {
    // 最後一筆：重複命令缺少調色盤索引位元組。
    for data in [&[][..], &[1, 0, 0, 0], &[1, 0, 1, 0, 0xc2]] {
        assert!(parse_opaque_run_cell(data).is_err(), "malformed opaque-run cell {data:?} was accepted");
    }
}

#[test]
fn lmi1_blit_preserves_transparent_and_mirrors() {
    let e = LMI1Entry { width: 3, height: 1, pixels: vec![1, 0, 2] };
    let mut dst = vec![9u8; 16];
    e.blit_at(&mut dst, 8, 1, 0, false).unwrap();
    assert_eq!(&dst[..5], &[9, 1, 9, 2, 9], "forward blit");
    e.blit_at(&mut dst, 8, 1, 1, true).unwrap();
    assert_eq!(&dst[8..13], &[9, 2, 9, 1, 9], "mirrored blit");
}

#[test]
fn lmi1_opaque_blit_matches_4e8af_zero_writes() {
    let e = LMI1Entry { width: 3, height: 1, pixels: vec![1, 0, 2] };
    let mut dst = vec![9u8; 8];
    e.blit_opaque_at(&mut dst, 8, 1, 0, false).unwrap();
    assert_eq!(&dst[..5], &[9, 1, 0, 2, 9], "opaque blit");
    e.blit_opaque_at(&mut dst, 8, 4, 0, true).unwrap();
    assert_eq!(&dst[4..7], &[2, 0, 1], "opaque mirror");
}

#[test]
fn fdother_005_lmi1_ui_container() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let entries = decode_lmi1_resource(&path, 5).unwrap();
    assert_eq!(entries.len(), 138, "FDOTHER#5 LMI1 entry count");
    // 原生 0x1f42d 為進入戰鬥的左右滑入選 LMI1 entry #0x52。
    let e = &entries[0x52];
    assert!(e.width == 72 && e.height == 14 && e.pixels.len() == 72 * 14, "FDOTHER#5 LMI1 entry#0x52");
}

#[test]
fn fdother_005_map_hud_uses_4mode_frame_entries() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    // 0x1aeb1 的原生立即值是 0x83/0x84，所以是十六進位目錄索引 131/132，不是十進位 83/84。
    for (index, want) in [(0x83, [6, 7]), (0x84, [6, 5]), (130, [69, 34])] {
        let frame = decode_lmi1_frame_resource(&path, 5, index).unwrap();
        assert_eq!([frame.width, frame.height], want, "entry {index:#x} geometry");
        frame
            .blit_at(&mut vec![0u8; 320 * 200], 320, 320 * 157 + 1, -1)
            .unwrap_or_else(|e| panic!("entry {index:#x} 0x4e63d decode: {e}"));
    }
}

#[test]
fn fdother_005_dialogue_frame_uses_raw_lmi1_entry_path() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let cell = decode_lmi1_raw_entry(&path, 5, 1).unwrap();
    assert!(
        cell.width == 3 && cell.height == 3 && cell.pixels.len() == 9 && cell.pixels[..3] == [0x60, 0xbe, 0xbd],
        "raw dialogue cell={cell:?}"
    );
    // 0xbe/bd 在這條原生路徑是逐字索引像素，不是重複碼。
    cell.blit_at(&mut [0u8; 16], 4, 0, 0).unwrap();
}

#[test]
fn parse_lmi1_frame_entry_rejects_other_container_or_index() {
    assert!(parse_lmi1_frame_entry(b"bad", 0).is_err(), "non-LMI1 container was accepted");
    let data = lmi1_header(1, &[10]);
    assert!(parse_lmi1_frame_entry(&data, 1).is_err(), "out-of-range LMI1 index was accepted");
}

#[test]
fn parse_lmi1_opaque_entry_uses_4e916_high_run_codec() {
    let mut data = lmi1_header(1, &[10]);
    data.extend_from_slice(&[3, 0, 2, 0, 0xc3, 7, 1, 2, 3]);
    let entry = parse_lmi1_opaque_entry(&data, 0).unwrap();
    assert_eq!(entry, LMI1Entry { width: 3, height: 2, pixels: vec![7, 7, 7, 1, 2, 3] });
}

#[test]
fn fdother_006_native_unit_present_bank() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let entries = decode_lmi1_resource(&path, 6).unwrap();
    assert_eq!(entries.len(), 230, "FDOTHER#6 LMI1 entry count");
    // 0x22470 讀 #6 entries 0x72..0x7c；0x22253 的 +0x1f6 目錄指標是 entry 0x7c。這些幾何錨點
    // 讓原生單位出場資源與 #81 未使用的配置分開。
    assert!(entries[0x72].width == 12 && entries[0x72].height == 21, "FDOTHER#6 entry#0x72");
    for (index, entry) in entries.iter().enumerate().take(0x7b + 1).skip(0x73) {
        assert!(entry.width == 20 && entries[index].height == 22, "FDOTHER#6 entry#{index:#x}");
    }
    assert!(entries[0x7c].width == 24 && entries[0x7c].height == 23, "FDOTHER#6 entry#0x7c");
}

#[test]
#[allow(clippy::needless_range_loop)] // 照 Go 以描述子索引迴圈，訊息印出索引。
fn fdother_005_and_006_command_heal_tail_descriptors() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let digits = decode_lmi1_resource(&path, NATIVE_COMMAND_HEAL_TAIL_DIGIT_RESOURCE).unwrap();
    let effect = decode_lmi1_resource(&path, NATIVE_COMMAND_HEAL_TAIL_EFFECT_RESOURCE).unwrap();
    assert!(digits.len() > NATIVE_COMMAND_HEAL_TAIL_DIGIT_BIAS + 9, "FDOTHER#5 count={}", digits.len());
    assert!(
        effect.len() >= NATIVE_COMMAND_HEAL_TAIL_EFFECT_START + NATIVE_COMMAND_HEAL_TAIL_EFFECT_FRAMES,
        "FDOTHER#6 count={}",
        effect.len()
    );
    let well_formed = |e: &LMI1Entry| e.width > 0 && e.height > 0 && e.pixels.len() as i64 == e.width * e.height;
    for index in NATIVE_COMMAND_HEAL_TAIL_DIGIT_BIAS..=NATIVE_COMMAND_HEAL_TAIL_DIGIT_BIAS + 9 {
        assert!(well_formed(&digits[index]), "FDOTHER#5 descriptor {index:#x} malformed");
    }
    for index in NATIVE_COMMAND_HEAL_TAIL_EFFECT_START
        ..NATIVE_COMMAND_HEAL_TAIL_EFFECT_START + NATIVE_COMMAND_HEAL_TAIL_EFFECT_FRAMES
    {
        assert!(well_formed(&effect[index]), "FDOTHER#6 descriptor {index:#x} malformed");
    }
}
