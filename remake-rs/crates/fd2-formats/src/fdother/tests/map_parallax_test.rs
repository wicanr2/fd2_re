//! 移植自 `map_parallax_test.go`。

use super::{TempDir, pack};
use crate::fdother::*;

#[test]
fn native_map_parallax_matches_both_fixed_raw_segments() {
    // Go 讀素材包裡 raw/FDOTHER/ 的兩個抽出檔。
    let Some(root) = pack("", "raw/FDOTHER/FDOTHER_016.bin") else { return };
    let mut frames = Vec::new();
    for name in ["FDOTHER_016.bin", "FDOTHER_017.bin"] {
        let raw = std::fs::read(root.join("raw").join("FDOTHER").join(name)).unwrap();
        frames.push(parse_single_frame(&raw).unwrap());
    }
    let want = build_native_map_parallax(&frames[0], &frames[1]).unwrap();
    let got = load_separated_native_map_parallax(root.join("surfaces")).unwrap();
    assert_eq!(got.pixels, want.pixels, "分離山景與固定原版解碼不同");
    let dir = TempDir::new("parallax");
    assert!(load_separated_native_map_parallax(dir.path()).is_err(), "缺山景資產卻成功");
    frames[1].height -= 1;
    assert!(build_native_map_parallax(&frames[0], &frames[1]).is_err(), "錯誤雙段高度卻成功");
}

#[test]
fn native_map_parallax_viewport_edges_scroll_and_atomic_failure() {
    let pixels: Vec<u8> = (0..462 * 226).map(|i: usize| (i * 7 + i / 462) as u8).collect();
    let surface = NativeMapParallaxSurface { pixels };
    for (cx, cy, sx, sy) in [(5, 6, 0, 0), (37, 17, 0, 0), (4, 5, -5, -5), (4, 5, 5, 5)] {
        let mut dst = vec![253u8; 456 * 191 + 312];
        blit_native_map_parallax_viewport(&mut dst, 456, &surface, cx, cy, sx, sy).unwrap();
        // 原始有號除法向零截斷，最後一列只需 312 bytes。
        for row in [0i64, 112, 113, 191] {
            let start = ((2 * cy + sy / 3 + row) * 462 + 3 * cx + sx / 2) as usize;
            let at = (row * 456) as usize;
            assert_eq!(
                &dst[at..at + 312],
                &surface.pixels[start..start + 312],
                "鏡頭／捲動來源錯誤：{:?} row{row}",
                (cx, cy, sx, sy)
            );
        }
        assert_eq!(dst[312], 253, "鋪底污染456 stride padding");
    }
    let mut dst = vec![77u8; 456 * 192];
    let before = dst.clone();
    for (cx, cy, sx, sy) in [(0, 0, -6, 0), (37, 17, 0, 6), (51, 0, 0, 0), (0, 18, 0, 0)] {
        let r = blit_native_map_parallax_viewport(&mut dst, 456, &surface, cx, cy, sx, sy);
        assert!(r.is_err() && dst == before, "越界輸入未原子拒收：{:?}", (cx, cy, sx, sy));
    }
}
