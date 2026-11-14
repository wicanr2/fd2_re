//! R4 D 組：`freezeModernHDDialogueChrome` 的測試。Go 的兩個測試（`modern_hd_dialogue_chrome_test.go:139`、`:174`）
//! 比對的是 HD 外框貼片（`patch`，R6），這裡只移植來源驗證的部分，另核對嵌入的金色素材。

use super::*;

/// D 組補充：嵌入的金色素材與 Go `//go:embed` 的檔案逐位元組相同（SHA-256 記在常數註解），且解得出 48×48。
#[test]
fn embedded_gold_dialogue_tile_matches_go_source() {
    use sha2::Digest;
    let digest: String =
        sha2::Sha256::digest(&*MODERN_HD_GOLD_DIALOGUE_TILE).iter().map(|b| format!("{b:02x}")).collect();
    assert_eq!(digest, "e03b4aa090221e8145d36e1541c42490546cc37197279327af81df3d0d09afb8");
    let tile = decode_png_picture(&MODERN_HD_GOLD_DIALOGUE_TILE).expect("PNG 解得開");
    assert_eq!((tile.width, tile.height), (48, 48));
}

/// D 組補充：建一組形狀正確的合成格子（每格單色），檢查凍結結果與 Go 的取色規則相同。
fn synthetic_cells() -> (Vec<fdother::RawCell>, ColorPalette) {
    let mut cells = vec![fdother::RawCell::default(); 14];
    for (i, (w, h)) in [
        (1, (3, 3)),
        (2, (3, 3)),
        (3, (3, 3)),
        (4, (3, 3)),
        (9, (16, 3)),
        (10, (3, 16)),
        (11, (3, 16)),
        (12, (16, 3)),
        (13, (16, 16)),
    ] {
        cells[i] = fdother::RawCell { width: w, height: h, pixels: vec![i as u8; (w * h) as usize] };
    }
    let palette = (0..256).map(|i| ColorRgba { r: i as u8, g: 255 - i as u8, b: 7, a: 255 }).collect();
    (cells, palette)
}

#[test]
fn freeze_modern_hd_dialogue_chrome_reads_native_colors() {
    let (cells, palette) = synthetic_cells();
    let chrome = freeze_modern_hd_dialogue_chrome(&cells, &palette).unwrap().unwrap();
    assert_eq!(chrome.background, palette[13]);
    // top, right, bottom, left 依序取 cell 9、11、12、10。
    for (side, source) in [9usize, 11, 12, 10].into_iter().enumerate() {
        assert_eq!(chrome.edges[side], [palette[source]; 3]);
    }
    for corner in 0..4 {
        assert_eq!(chrome.corners[corner], [palette[corner + 1]; 9]);
    }
    assert!(chrome.gold.is_some());
}

/// Go `TestModernHDDialogueChromeRejectsSourcesBeforeCountdown`（`modern_hd_dialogue_chrome_test.go:174`）的來源拒收段，
/// 以合成格子代替 `expiryHDTestGame` 的原版素材（那段經 `beginNativeTransientPhases` 的部分屬 HD，R6）。
#[test]
fn freeze_modern_hd_dialogue_chrome_rejects_bad_sources() {
    let (cells, palette) = synthetic_cells();
    for index in [1usize, 9, 10, 11, 12, 13] {
        let mut bad = cells.clone();
        bad[index].pixels = Vec::new();
        assert!(freeze_modern_hd_dialogue_chrome(&bad, &palette).is_err(), "invalid raw cell accepted {index}");
    }
    let mut bad = cells.clone();
    bad[13].pixels[0] ^= 1;
    let err = freeze_modern_hd_dialogue_chrome(&bad, &palette).unwrap_err();
    assert!(err.message().contains("background is not uniform"), "{err}");
    let mut translucent = palette.clone();
    translucent[13].a = 0;
    assert!(freeze_modern_hd_dialogue_chrome(&cells, &translucent).is_err(), "missing source color accepted");
    assert!(freeze_modern_hd_dialogue_chrome(&cells[..13], &palette).is_err());
}
