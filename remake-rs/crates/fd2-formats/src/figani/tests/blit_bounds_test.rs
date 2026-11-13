//! Go `blit_bounds_test.go`。

use super::*;
use sha2::{Digest, Sha256};

/// 子測試：（名稱，幀，目的長度，base，預期寫入）。
type SpanCase = (&'static str, Frame, usize, i64, &'static [(usize, u8)]);
/// 子測試：（名稱，目的長度，跨距，base，改幀）。
type InvalidCase = (&'static str, usize, i64, i64, Box<dyn Fn(&mut Frame)>);

fn pane_frame(x: i64, y: i64, pixels: &[u8], mask: &[u8], height: i64) -> Frame {
    Frame { x, y, width: 2, height, pixels: pixels.to_vec(), mask: mask.to_vec(), ..Frame::default() }
}

// Go: TestBlitAtBaseLastRowSpan（四個子測試）
#[test]
fn blit_at_base_last_row_span() {
    let cases: [SpanCase; 4] = [
        ("right-pane", pane_frame(2, 1, &[7, 8], &[1, 0], 1), 16, 4, &[(14, 7)]),
        ("left-pane", pane_frame(2, 1, &[7, 8], &[1, 0], 1), 16, 0, &[(10, 7)]),
        ("multiple-rows", pane_frame(1, 0, &[2, 3, 4, 5], &[1, 0, 1, 1], 2), 16, 4, &[(5, 2), (13, 4), (14, 5)]),
        ("partial-final-row", pane_frame(1, 0, &[2, 3, 4, 5], &[1, 0, 1, 1], 2), 15, 4, &[(5, 2), (13, 4), (14, 5)]),
    ];
    for (name, frame, length, base, writes) in cases {
        let mut dst = vec![9u8; length];
        let mut want = dst.clone();
        for &(offset, value) in writes {
            want[offset] = value;
        }
        frame.blit_at_base(&mut dst, 8, base).unwrap();
        assert_eq!(dst, want, "{name}: 非opaque來源或鄰格被改寫");
    }
}

// Go: TestBlitAtBaseInvalidSpanPreservesDestination（十四個子測試）
#[test]
fn blit_at_base_invalid_span_preserves_destination() {
    let valid = pane_frame(1, 0, &[2, 3, 4, 5], &[1, 1, 1, 1], 2);
    let max = i64::MAX;
    let cases: Vec<InvalidCase> = vec![
        ("one-byte-short", 14, 8, 4, Box::new(|_| {})),
        ("negative-base", 16, 8, -1, Box::new(|_| {})),
        ("empty-at-base", 16, 8, 16, Box::new(|_| {})),
        ("base-outside", 16, 8, 17, Box::new(|_| {})),
        ("zero-stride", 16, 0, 4, Box::new(|_| {})),
        ("negative-x", 16, 8, 4, Box::new(|f| f.x = -1)),
        ("negative-y", 16, 8, 4, Box::new(|f| f.y = -1)),
        ("row-overflow", 16, 8, 4, Box::new(|f| f.x = 7)),
        ("missing-mask", 16, 8, 4, Box::new(|f| f.mask.clear())),
        ("wrong-pixels", 16, 8, 4, Box::new(|f| f.pixels.truncate(3))),
        ("huge-y", 16, 8, 4, Box::new(move |f| f.y = max)),
        ("huge-width", 16, 8, 4, Box::new(move |f| f.width = max)),
        ("huge-height", 16, 8, 4, Box::new(move |f| f.height = max)),
        ("huge-stride", 16, max, 4, Box::new(|_| {})),
    ];
    for (name, length, stride, base, change) in cases {
        let mut frame = valid.clone();
        change(&mut frame);
        let mut dst = vec![9u8; length];
        let before = dst.clone();
        let result = frame.blit_at_base(&mut dst, stride, base);
        assert!(result.is_err() && dst == before, "{name}: 非法span未拒收或發布部分像素：{result:?} dst={dst:?}");
    }
}

// Go: TestOriginalFIGANI51FitsRightWorkPane
#[test]
fn original_figani51_fits_right_work_pane() {
    let Some(archive) = original("FIGANI.DAT") else { return };
    let Some(pack) = original_pack() else { return };
    let raw = std::fs::read(&archive).unwrap();
    let digest: String = Sha256::digest(&raw).iter().map(|b| format!("{b:02x}")).collect();
    assert_eq!(digest, "e909314ce37b70701238fd24febc8f61603bbe8d81a9ecbf8f289c97a5b67a1f", "unknown FIGANI.DAT source");
    let original = decode_resource(&archive, 51).unwrap();
    let separated = load_separated_resource(&format!("{pack}/animations"), 51).unwrap();
    assert_eq!(original, separated, "分離FANI51與固定原版不相同");
    let first = &original.frames[0];
    assert_eq!(
        (first.x, first.y, first.width, first.height),
        (184, 35, 109, 165),
        "FANI51 frame0 geometry與原始header不相同"
    );
    for (index, frame) in separated.frames.iter().enumerate() {
        let mut dst = vec![9u8; 640 * 200];
        let mut want = dst.clone();
        // 先在 320 寬右半原生畫面形成預期，再核對 640 跨距暫存區的全部位元組。
        let mut right = vec![9u8; 320 * 200];
        frame.blit_at(&mut right, 320).unwrap();
        for y in 0..200 {
            want[y * 640 + 320..y * 640 + 640].copy_from_slice(&right[y * 320..(y + 1) * 320]);
        }
        frame.blit_at_base(&mut dst, 640, 320).unwrap_or_else(|e| panic!("frame{index}: {e}"));
        assert!(dst == want, "frame{index}右半staging與原生畫面不相同");
    }
}
