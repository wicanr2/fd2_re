//! 逐條移植自 Go `remake/internal/dato` 的 `dato_test.go`、`mouth_test.go`、
//! `confirmation_mouth_test.go`、`separated_test.go`，期望值不改。最後幾項是 Rust 端補的
//! 錯誤文字與 PNG 邊界檢查，Go 版沒有對應測試。

use super::*;
use std::path::PathBuf;

/// 原版檔案目錄：`FD2_FLAME2_DIR`；未設定時跳過（Go 版以 t.Skip 處理缺檔），
/// `FD2_REQUIRE_ORIGINAL=1` 時缺檔即失敗。與 `archive` 模組測試的同名輔助函式相同。
fn original(name: &str) -> Option<PathBuf> {
    crate::test_support::flame2(name)
}

/// 離線分離頭像目錄：`FD2_ASSET_PACK/portraits`（對應 Go 測試的
/// `remake/generated-assets/fd2-original-b97caf22/portraits`），缺檔規則同 [`original`]。
fn portraits(probe: &str) -> Option<PathBuf> {
    crate::test_support::asset_pack(&format!("portraits/{probe}")).and_then(|p| p.parent().map(PathBuf::from))
}

/// 每個測試自己的暫存目錄（Go 的 t.TempDir）。
struct TempDir(PathBuf);

impl TempDir {
    fn new(tag: &str) -> Self {
        let dir = std::env::temp_dir().join(format!("fd2-dato-{}-{tag}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        TempDir(dir)
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

fn crc32(data: &[u8]) -> u32 {
    let mut crc = !0u32;
    for &b in data {
        crc ^= u32::from(b);
        for _ in 0..8 {
            crc = if crc & 1 != 0 { 0xedb8_8320 ^ (crc >> 1) } else { crc >> 1 };
        }
    }
    !crc
}

fn chunk(out: &mut Vec<u8>, kind: &[u8; 4], data: &[u8]) {
    out.extend_from_slice(&(data.len() as u32).to_be_bytes());
    let mut body = kind.to_vec();
    body.extend_from_slice(data);
    out.extend_from_slice(&body);
    out.extend_from_slice(&crc32(&body).to_be_bytes());
}

/// 測試用 PNG 編碼：zlib 包一個 stored deflate 區塊，每列濾波 0。`rows` 是已打包的列資料。
fn encode_png(width: u32, height: u32, depth: u8, color_type: u8, palette: Option<&[u8]>, rows: &[&[u8]]) -> Vec<u8> {
    let mut raw = Vec::new();
    for row in rows {
        raw.push(0);
        raw.extend_from_slice(row);
    }
    let (mut a, mut b) = (1u32, 0u32);
    for &x in &raw {
        a = (a + u32::from(x)) % 65521;
        b = (b + a) % 65521;
    }
    let mut z = vec![0x78, 0x01, 0x01];
    z.extend_from_slice(&(raw.len() as u16).to_le_bytes());
    z.extend_from_slice(&(!(raw.len() as u16)).to_le_bytes());
    z.extend_from_slice(&raw);
    z.extend_from_slice(&(b << 16 | a).to_be_bytes());
    let mut out = b"\x89PNG\r\n\x1a\n".to_vec();
    let mut ihdr = width.to_be_bytes().to_vec();
    ihdr.extend_from_slice(&height.to_be_bytes());
    ihdr.extend_from_slice(&[depth, color_type, 0, 0, 0]);
    chunk(&mut out, b"IHDR", &ihdr);
    if let Some(p) = palette {
        chunk(&mut out, b"PLTE", p);
    }
    chunk(&mut out, b"IDAT", &z);
    chunk(&mut out, b"IEND", &[]);
    out
}

/// Go 測試的 writeSeparatedPortrait。Go `png.Encode` 對三色調色盤選 2 位元深度、不加濾波；
/// RGBA 情形是全透明的 NRGBA，編成色彩型態 6。這裡產生同樣型態的 PNG。
fn write_separated_portrait(root: &Path, resource: i64, count: usize, rgba: bool) {
    std::fs::create_dir_all(root).unwrap();
    for mouth in 0..count {
        let path = root.join(format!("DATO_{resource:03}_m{mouth}.png"));
        let data = if rgba {
            encode_png(2, 2, 8, 6, None, &[&[0; 8], &[0; 8]])
        } else {
            let palette = [0, 0, 0, 255, 255, 255, 255, 0, 0];
            let pix = [(mouth % 3) as u8, 1, 2, 0];
            // 2 位元深度、每列兩個像素打包在高位。
            let row0 = [pix[0] << 6 | pix[1] << 4];
            let row1 = [pix[2] << 6 | pix[3] << 4];
            encode_png(2, 2, 2, 3, Some(&palette), &[&row0, &row1])
        };
        std::fs::write(path, data).unwrap();
    }
}

// Go: TestParseDATOHighRunAndOpaqueZero
#[test]
fn parse_dato_high_run_and_opaque_zero() {
    let mut data = vec![0u8; 16 + 4 * (4 + 4)];
    for i in 0..4 {
        data[i * 4..i * 4 + 4].copy_from_slice(&((16 + i * 8) as u32).to_le_bytes());
        data[16 + i * 8..18 + i * 8].copy_from_slice(&2u16.to_le_bytes());
        data[18 + i * 8..20 + i * 8].copy_from_slice(&2u16.to_le_bytes());
        data[20 + i * 8..23 + i * 8].copy_from_slice(&[0xc2, (i + 1) as u8, 0]);
    }
    let frames = parse_resource(&data).unwrap();
    assert!(
        frames.len() == 4
            && frames[0].pixels[0] == 1
            && frames[0].pixels[1] == 1
            && frames[0].pixels[2] == 0
            && frames[3].pixels[0] == 4,
        "frames={frames:?}"
    );
    let mut dst = [9u8, 9, 9, 9];
    frames[0].blit_at(&mut dst, 2, 0, 0).unwrap();
    assert_eq!(dst, [1, 1, 0, 0], "opaque blit");
}

// Go: TestDATOPlayerResourceIfPresent
#[test]
fn dato_player_resource_if_present() {
    let Some(path) = original("DATO.DAT") else { return };
    let frames = decode_resource(path, 37).unwrap();
    assert!(
        frames.len() == 4 && frames[0].width == 80 && frames[0].height == 80 && frames[0].pixels.len() == 6400,
        "DATO#37={frames:?}"
    );
}

// Go: TestDATOFrameBlitAtOffsetIsOpaque
#[test]
fn dato_frame_blit_at_offset_is_opaque() {
    let mut dst = vec![0u8; 320 * 20];
    let frame = Frame { width: 2, height: 2, pixels: vec![0, 7, 8, 0] };
    frame.blit_at_offset(&mut dst, 320, 3 * 320 + 4).unwrap();
    assert!(
        dst[3 * 320 + 4] == 0 && dst[3 * 320 + 5] == 7 && dst[4 * 320 + 4] == 8 && dst[4 * 320 + 5] == 0,
        "DATO offset blit={:?}",
        &dst[3 * 320 + 4..3 * 320 + 6]
    );
}

// Go: TestDATOFrameBlitRightToLeftUsesRightEdgeAnchor
#[test]
fn dato_frame_blit_right_to_left_uses_right_edge_anchor() {
    let mut dst = vec![0u8; 8 * 4];
    let frame = Frame { width: 3, height: 2, pixels: vec![1, 2, 3, 4, 5, 6] };
    frame.blit_right_to_left_at_offset(&mut dst, 8, 8 + 5).unwrap();
    assert_eq!(&dst[8 + 3..8 + 6], &[3, 2, 1], "第一列由右往左寫入");
    assert_eq!(&dst[2 * 8 + 3..2 * 8 + 6], &[6, 5, 4], "第二列stride");
}

// Go: TestMouthStateOpenClosesWithNativeRange
#[test]
fn mouth_state_open_closes_with_native_range() {
    for r in 0..30 {
        let next = MouthState { open: true, countdown: 0 }.tick(r).unwrap();
        assert!(!next.open && next.countdown == r + 2, "r={r}: got {next:?}");
    }
}

// Go: TestMouthStateClosedOpensAtZero
#[test]
fn mouth_state_closed_opens_at_zero() {
    let next = MouthState { open: false, countdown: 1 }.tick(0).unwrap();
    assert!(!next.open && next.countdown == 0 && next.frame_index() == 0, "got {next:?}");
    let next = MouthState::default().tick(0).unwrap();
    assert!(next.open, "zero countdown should open: {next:?}");
}

// Go: TestMouthStateClosedCountsDown
#[test]
fn mouth_state_closed_counts_down() {
    let next = MouthState { open: false, countdown: 4 }.tick(29).unwrap();
    assert!(!next.open && next.countdown == 3 && next.frame_index() == 0, "got {next:?}");
}

// Go: TestMouthStateRejectsInvalidRandom
#[test]
fn mouth_state_rejects_invalid_random() {
    for r in [-1, 30] {
        assert!(MouthState { open: true, countdown: 0 }.tick(r).is_err(), "random={r} accepted");
    }
    assert!(MouthState { open: false, countdown: -1 }.tick(0).is_err(), "negative countdown accepted");
}

// Go: TestConfirmationMouthPostDecrementAndCallerSpecificRestart
#[test]
fn confirmation_mouth_post_decrement_and_caller_specific_restart() {
    for sample in 0..30 {
        let mut s = ConfirmationMouthState::new(sample).unwrap();
        assert!(!s.open && s.countdown == sample + 2 && s.frame_index() == 0, "initial {sample} {s:?}");
        for old in (0..=sample + 2).rev() {
            s = s.tick(0).unwrap();
            assert!(s.countdown == old - 1 && s.open == (old == 0), "post-decrement {sample} {old} {s:?}");
        }
        assert_eq!(s.frame_index(), 3, "open frame {s:?}");
        s = s.tick(sample).unwrap();
        assert!(
            !s.open && s.countdown == sample + 10 && s.frame_index() == 0,
            "confirmation restart must be 10..39 {sample} {s:?}"
        );
    }
}

// Go: TestConfirmationMouthRejectsInvalidSamplesAndStates
#[test]
fn confirmation_mouth_rejects_invalid_samples_and_states() {
    for sample in [-1, 30, 1000] {
        assert!(ConfirmationMouthState::new(sample).is_err(), "invalid initial sample accepted {sample}");
        assert!(ConfirmationMouthState::default().tick(sample).is_err(), "invalid tick sample accepted {sample}");
    }
    for s in [
        ConfirmationMouthState { open: false, countdown: -1 },
        ConfirmationMouthState { open: false, countdown: 40 },
        ConfirmationMouthState { open: true, countdown: 0 },
        ConfirmationMouthState { open: true, countdown: -2 },
    ] {
        assert!(s.tick(0).is_err(), "invalid state accepted {s:?}");
    }
}

// Go: TestLoadSeparatedResourcePreservesIndexedFrames
#[test]
fn load_separated_resource_preserves_indexed_frames() {
    let root = TempDir::new("preserves");
    write_separated_portrait(&root.0, 26, 4, false);
    let frames = load_separated_resource(&root.0, 26).unwrap();
    assert_eq!(frames.len(), 4);
    for (mouth, frame) in frames.iter().enumerate() {
        assert!(
            frame.width == 2 && frame.height == 2 && frame.pixels.len() == 4 && frame.pixels[0] == (mouth % 3) as u8,
            "frame {mouth}={frame:?}"
        );
    }
}

// Go: TestLoadSeparatedResourceFailsClosed
#[test]
fn load_separated_resource_fails_closed() {
    let root = TempDir::new("three");
    write_separated_portrait(&root.0, 26, 3, false);
    assert!(load_separated_resource(&root.0, 26).is_err(), "three-frame portrait was accepted");
    let root = TempDir::new("rgba");
    write_separated_portrait(&root.0, 26, 4, true);
    assert!(load_separated_resource(&root.0, 26).is_err(), "RGBA portrait was accepted as indexed provenance");
}

// Go: TestSeparatedResourceMatchesOriginalArchive
#[test]
fn separated_resource_matches_original_archive() {
    let Some(archive) = original("DATO.DAT") else { return };
    let Some(portrait_root) = portraits("DATO_026_m3.png") else { return };
    let want = decode_resource(archive, 26).unwrap();
    let got = load_separated_resource(portrait_root, 26).unwrap();
    assert_eq!(got.len(), want.len(), "frames");
    for index in 0..want.len() {
        assert!(
            got[index].width == want[index].width
                && got[index].height == want[index].height
                && got[index].pixels == want[index].pixels,
            "frame {index} differs from original archive"
        );
    }
}

// 以下為 Rust 端補充。

#[test]
fn error_texts_match_go() {
    let cases: [(Result<Vec<Frame>>, &str); 5] = [
        (parse_resource(&[0; 15]), "dato: resource is shorter than four offsets"),
        (parse_resource(&[0; 16]), "dato: frame 0 offset is invalid"),
        (load_separated_resource("", 1), "dato: separated portrait input is invalid"),
        (load_separated_resource("x", -1), "dato: separated portrait input is invalid"),
        (
            {
                let mut d = vec![0u8; 24];
                for i in 0..4 {
                    d[i * 4..i * 4 + 4].copy_from_slice(&16u32.to_le_bytes());
                }
                parse_resource(&d)
            },
            "dato: frame 0 header exceeds bounds",
        ),
    ];
    for (result, want) in cases {
        assert_eq!(result.unwrap_err().to_string(), want);
    }
    let err = ConfirmationMouthState { open: true, countdown: 5 }.tick(3).unwrap_err();
    assert_eq!(err.to_string(), "dato: invalid confirmation mouth state/sample: {Open:true Countdown:5}, 3");
    assert_eq!(MouthState::default().tick(30).unwrap_err().to_string(), "dato: random mouth value 30 outside [0,30)");
}

#[test]
fn go_join_clean_is_lexical() {
    assert_eq!(go_join_clean(Path::new("a/../b/"), "f.png"), PathBuf::from("b/f.png"));
    assert_eq!(go_join_clean(Path::new("/../x"), "f.png"), PathBuf::from("/x/f.png"));
    assert_eq!(go_join_clean(Path::new("../x/./"), "f.png"), PathBuf::from("../x/f.png"));
}

#[test]
fn separated_png_low_depth_and_fail_closed() {
    let root = TempDir::new("variants");
    let palette = [0u8, 0, 0, 255, 255, 255];
    // 1 位元深度的 indexed PNG 要逐位元解包；缺 IEND 的截斷檔與非 PNG 內容必須拒收。
    let one_bit = encode_png(3, 1, 1, 3, Some(&palette), &[&[0b1010_0000]]);
    for mouth in 0..4 {
        std::fs::write(root.0.join(format!("DATO_001_m{mouth}.png")), &one_bit).unwrap();
    }
    let frames = load_separated_resource(&root.0, 1).unwrap();
    assert_eq!(frames[0].pixels, vec![1, 0, 1]);

    let mut truncated = one_bit.clone();
    truncated.truncate(one_bit.len() - 12);
    std::fs::write(root.0.join("DATO_003_m0.png"), truncated).unwrap();
    let err = load_separated_resource(&root.0, 3).unwrap_err().to_string();
    assert!(err.starts_with("dato: separated portrait 3 frame 0 decode: "), "{err}");

    std::fs::write(root.0.join("DATO_004_m0.png"), b"GIF89a").unwrap();
    assert_eq!(
        load_separated_resource(&root.0, 4).unwrap_err().to_string(),
        "dato: separated portrait 4 frame 0 decode: image: unknown format"
    );
}
