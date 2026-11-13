//! 移植自 `frame_test.go`。

use super::{original, original_entry, pack};
use crate::fdother::*;

fn test_frame(x: u16, y: u16, w: u16, h: u16, rle: &[u8]) -> Vec<u8> {
    let mut d = vec![0u8; 13];
    d[0..2].copy_from_slice(&x.to_le_bytes());
    d[2..4].copy_from_slice(&y.to_le_bytes());
    d[9..11].copy_from_slice(&w.to_le_bytes());
    d[11..13].copy_from_slice(&h.to_le_bytes());
    d.extend_from_slice(rle);
    d
}

fn test_container(frames: &[Vec<u8>]) -> Vec<u8> {
    let mut d = vec![0u8; 8 + 4 * frames.len()];
    d[0..2].copy_from_slice(&(frames.len() as u16).to_le_bytes());
    let mut off = d.len();
    for (i, f) in frames.iter().enumerate() {
        d[8 + 4 * i..12 + 4 * i].copy_from_slice(&(off as u32).to_le_bytes());
        off += f.len();
    }
    for f in frames {
        d.extend_from_slice(f);
    }
    d
}

#[test]
fn frame_blit_preserves_transparent_destination() {
    // run(2,5), dither(1,8), literal(1,9), skip(1) -> 5,5,old,8,9,old.
    let data = test_container(&[test_frame(2, 1, 6, 1, &[1, 5, 0x40, 8, 0x80, 9, 0xc0])]);
    let frames = parse_frames(&data).unwrap();
    let mut dst = vec![7u8; 30];
    frames[0].blit(&mut dst, 10, -1).unwrap();
    assert_eq!(&dst[12..18], &[5, 5, 7, 8, 9, 7]);
}

#[test]
fn frame_indexed_layers_preserve_opaque_zero_and_destination_mask() {
    // opaque zero, dither-preserve/opaque 8, skip => indexed [0,0,8,0], mask [255,0,255,0].
    let frame = parse_single_frame(&[4, 0, 1, 0, 0, 0, 0x40, 8, 0xc0]).unwrap();
    let (indexed, mask) = frame.indexed_layers().unwrap();
    assert!(indexed == [0, 0, 8, 0] && mask == [255, 0, 255, 0], "indexed={indexed:?} mask={mask:?}");
    let mut dst = vec![7u8, 7, 7, 7];
    let separated = Frame::separated(0, 0, 4, 1, indexed, mask);
    separated.blit(&mut dst, 4, -1).unwrap();
    assert_eq!(dst, [0, 7, 8, 7], "separated blit");
}

#[test]
fn frame_rejects_malformed_rle() {
    let frames = parse_frames(&test_container(&[test_frame(0, 0, 2, 1, &[1])])).unwrap();
    assert!(frames[0].blit(&mut [0u8; 2], 2, -1).is_err(), "truncated RLE was accepted");
}

/// 兩個 Go 測試共用：解析整張圖格表、核對錨點並把每格解碼進 320x200。
fn check_frame_table(data: &[u8], count: usize, anchors: &[(usize, [i64; 4])]) {
    let frames = parse_frames(data).unwrap();
    assert_eq!(frames.len(), count, "frame count");
    for &(index, want) in anchors {
        let got = &frames[index];
        assert_eq!([got.x, got.y, got.width, got.height], want, "frame {index}");
    }
    // 每個結局呼叫端都對 320 寬畫面用已驗證的透明分支。這裡解碼每一格，抓出錯誤的 RLE
    // 文法或圖格表邊界，又不必提交原版美術。
    for (i, frame) in frames.iter().enumerate() {
        frame
            .blit(&mut vec![0u8; 320 * 200], 320, -1)
            .unwrap_or_else(|e| panic!("frame {i} does not decode into 320x200: {e}"));
    }
}

#[test]
fn fdother_054_frame_table() {
    let Some(data) = original_entry("FDOTHER.DAT", 54) else { return };
    check_frame_table(
        &data,
        111,
        &[(0, [0, 23, 320, 132]), (9, [116, 39, 86, 81]), (108, [116, 39, 86, 81]), (110, [0, 0, 320, 200])],
    );
}

#[test]
fn fdother_034_sky_key_frame_table() {
    let Some(data) = original_entry("FDOTHER.DAT", 34) else { return };
    check_frame_table(
        &data,
        101,
        &[(0, [148, 94, 24, 24]), (68, [146, 95, 32, 23]), (69, [146, 95, 32, 23]), (100, [148, 94, 24, 36])],
    );
}

#[test]
fn fdother_054_archive_loader() {
    // Go 比對 extracted/raw 抽出檔與封存項目 0x36；這裡抽出檔就是封存項目本身，
    // 所以改為比對 archive_entry 與 read_resource 兩條路徑。
    let Some(path) = original("FDOTHER.DAT") else { return };
    let archive = std::fs::read(&path).unwrap();
    let entry = archive_entry(&archive, 0x36).unwrap();
    assert_eq!(
        entry,
        read_resource(&path, 54).unwrap().as_slice(),
        "FDOTHER archive entry 0x36 does not equal raw FDOTHER_054"
    );
    assert_eq!(decode_resource(&path, 0x36).unwrap().len(), 111, "loader frame count");
}

#[test]
fn fdother_056_single_frame_payload() {
    let Some(data) = original_entry("FDOTHER.DAT", 56) else { return };
    assert!(data.len() >= 4 && data[0..2] == 320u16.to_le_bytes() && data[2..4] == 200u16.to_le_bytes(), "#56 header");
    let frame = parse_single_frame(&data).unwrap();
    frame.blit(&mut vec![0u8; 320 * 200], 320, -1).expect("#56 single-frame RLE decode");
}

#[test]
fn bg_archive_single_frames_use_4mode_payload() {
    let Some(path) = original("BG.DAT") else { return };
    for index in [0, 1, 2] {
        let frame = decode_archive_single_frame(&path, index).unwrap();
        assert!(frame.width == 320 && frame.height == 100, "BG entry {index}={}x{}", frame.width, frame.height);
        frame
            .blit(&mut vec![0u8; 320 * 100], 320, -1)
            .unwrap_or_else(|e| panic!("BG entry {index} 0x4e63d decode: {e}"));
    }
}

#[test]
fn separated_bg_and_tai_surfaces_match_archive_decoder() {
    let Some(root) = pack("surfaces", "BG_000/resource.json") else { return };
    for source in ["BG.DAT", "TAI.DAT"] {
        let Some(archive) = original(source) else { return };
        for resource in 0..56 {
            let want = decode_archive_single_frame(&archive, resource).unwrap();
            let (want_indexed, want_mask) = want.indexed_layers().unwrap();
            let got = load_separated_single_frame(&root, source, resource).unwrap();
            assert!(
                got.width == want.width
                    && got.height == want.height
                    && got.indexed == want_indexed
                    && got.mask == want_mask,
                "{source}#{resource} differs from archive decoder"
            );
            let mut original_dst = vec![0x71u8; (want.width * want.height) as usize];
            let mut separated_dst = original_dst.clone();
            want.blit(&mut original_dst, want.width, -1).unwrap();
            got.blit(&mut separated_dst, got.width, -1).unwrap();
            assert_eq!(separated_dst, original_dst, "{source}#{resource} final blit differs");
        }
    }
    assert!(load_separated_single_frame(&root, "BG.DAT", 56).is_err(), "blocked BG.DAT#56 was accepted");
}

#[test]
fn fdother_081_archive_shape() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let entry = read_resource(path, 0x51).unwrap();
    assert!(!entry.is_empty(), "#81 is empty");
    assert!(entry.len() >= 10 && &entry[..6] == b"LLLLLL" && entry[6..10] == 0x12u32.to_le_bytes(), "#81 shape");
    for (resource, want) in [0x22ce, 0x2636].into_iter().enumerate() {
        let nested = archive_entry(&entry, resource).unwrap();
        assert_eq!(nested.len(), want, "#81 nested entry {resource}");
    }
    assert!(archive_entry(&entry, 2).unwrap().is_empty(), "#81 nested terminator entry");
    assert!(archive_entry(&entry, 3).is_err(), "#81 accepted a fourth nested entry");
}

#[test]
fn finale_tai003_is_transparent_placeholder() {
    let Some(data) = original_entry("TAI.DAT", 3) else { return };
    // TAI #3 不是其他戰鬥路徑使用的可見 154x42 平台：它的 10x3 原始 RLE 本體是三列
    // C9（= 透明跳過十個）。
    assert_eq!(data, [0x0a, 0, 0x03, 0, 0xc9, 0xc9, 0xc9], "TAI_003 want transparent 10x3 placeholder");
}

#[test]
fn frame_blit_lut_at_matches_4e583_four_modes() {
    let frame = Frame::raw(0, 0, 8, 1, vec![8, 0, 1, 0, 0x01, 3, 0x41, 4, 0x80, 5, 0xc0]);
    let lut: Vec<u8> = (0..256).map(|i| (i + 10) as u8).collect();
    let mut dst = vec![9u8; 8];
    frame.blit_lut_at(&mut dst, 8, 0, &lut).unwrap();
    assert_eq!(dst, [13, 13, 9, 14, 9, 14, 15, 9], "0x4e583 modes");
}

#[test]
fn frame_blit_lut_at_rejects_partial_lookup_without_mutation() {
    let frame = Frame::raw(0, 0, 1, 1, vec![1, 0, 1, 0, 0, 3]);
    let mut dst = vec![7u8];
    assert!(frame.blit_lut_at(&mut dst, 1, 0, &[0u8; 255]).is_err() && dst[0] == 7, "partial LUT accepted or mutated");
}
