//! 移植自 Go `indexedmap/mosaic_test.go`。

use super::*;

/// TestPhaseBannerBlockIsTheMeasuredPalindrome 釘住 33 步的對稱曲線。
#[test]
fn test_phase_banner_block_is_the_measured_palindrome() {
    let want = [
        1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2,
        1,
    ];
    assert_eq!(want.len() as i64, PHASE_BANNER_STEPS, "步數 {PHASE_BANNER_STEPS}，收據是 {}", want.len());
    for (step, &w) in want.iter().enumerate() {
        let got = phase_banner_block(step as i64);
        assert_eq!(got, w, "第 {step} 步邊長 {got}，收據是 {w}");
    }
    for step in 0..PHASE_BANNER_STEPS {
        assert_eq!(
            phase_banner_block(step),
            phase_banner_block(PHASE_BANNER_STEPS - 1 - step),
            "第 {step} 步與其鏡像不對稱"
        );
    }
}

/// TestMosaicNativeMapViewportTakesBlockTopLeftAndKeepsTheBorder 釘住取樣規則。
#[test]
fn test_mosaic_native_map_viewport_takes_block_top_left_and_keeps_the_border() {
    let size = NATIVE_MAP_VGA_SIZE as usize;
    let src: Vec<u8> = (0..size).map(|i| (i % 251 + 1) as u8).collect(); // 每個像素都不同，取樣規則才有鑑別力
    let mut dst = vec![0u8; size];
    mosaic_native_map_viewport(&mut dst, &src, 8).unwrap();
    const ORIGIN_X: usize = 4;
    const ORIGIN_Y: usize = 4;
    let (vw, vh, svw) = (VIEW_WIDTH as usize, VIEW_HEIGHT as usize, STEADY_VIEWPORT_WIDTH as usize);
    for y in 0..vh {
        for x in 0..svw {
            let want = src[(ORIGIN_Y + y / 8 * 8) * vw + ORIGIN_X + x / 8 * 8];
            let got = dst[(ORIGIN_Y + y) * vw + ORIGIN_X + x];
            assert_eq!(got, want, "({x},{y}) = {got}，應取方塊左上角 {want}");
        }
    }
    // 四邊黑邊不得被動到。
    for y in 0..200 {
        for x in 0..320 {
            let inside = x >= ORIGIN_X && x < ORIGIN_X + svw && y >= ORIGIN_Y && y < ORIGIN_Y + vh;
            if inside {
                continue;
            }
            assert_eq!(dst[y * vw + x], src[y * vw + x], "窗格外 ({x},{y}) 被改寫");
        }
    }
    let mut identity = vec![0u8; size];
    mosaic_native_map_viewport(&mut identity, &src, 1).unwrap();
    assert_eq!(identity, src, "邊長 1 應等同整幀複製");
}

/// TestPhaseBannerDimFollowsTheBlockCurve 釘住 `sub_11D40` 的減量：level 就是
/// 方塊邊長減一，進場 0..15、退場 16..0。
#[test]
fn test_phase_banner_dim_follows_the_block_curve() {
    assert_eq!(
        PHASE_BANNER_ENTER_STEPS + PHASE_BANNER_EXIT_STEPS,
        PHASE_BANNER_STEPS,
        "進場 {PHASE_BANNER_ENTER_STEPS} ＋ 退場 {PHASE_BANNER_EXIT_STEPS} ≠ {PHASE_BANNER_STEPS} 步"
    );
    for step in 0..PHASE_BANNER_STEPS {
        let (got, want) = (phase_banner_dim(step), phase_banner_block(step) - 1);
        assert_eq!(got, want, "第 {step} 步減量 {got}，應為邊長減一 {want}");
    }
    // 進場最後一步是 sub_1F1CC 的 ebp=16／edi=15；退場第一步是 sub_1F30A 的
    // ebp=0x11／edi=0x10。
    assert_eq!(phase_banner_dim(PHASE_BANNER_ENTER_STEPS - 1), 15, "進場最後一步減量，收據是 15");
    assert_eq!(phase_banner_dim(PHASE_BANNER_ENTER_STEPS), 16, "退場第一步減量，收據是 16");
    for step in [-1, PHASE_BANNER_STEPS] {
        assert_eq!(phase_banner_dim(step), 0, "界外第 {step} 步減量應為 0");
    }
    // 一個 BIOS tick：18.2065097 Hz。
    assert!(
        (54.9..=54.95).contains(&PHASE_BANNER_STEP_MILLIS),
        "每步停留 {:.4} 毫秒，應為一個 BIOS tick",
        PHASE_BANNER_STEP_MILLIS
    );
}

/// TestPhaseBannerSlideOffsetsMatchTheCallArguments 釘住 `sub_1F42D` 的七次與
/// 五次呼叫參數：x 相對停住位置的偏移就是參數的負值。
#[test]
fn test_phase_banner_slide_offsets_match_the_call_arguments() {
    let want_in = [-100, -75, -50, -25, 0, -1, 0];
    assert_eq!(want_in.len() as i64, PHASE_BANNER_SLIDE_IN_STEPS, "滑入步數");
    for (step, &want) in want_in.iter().enumerate() {
        let got = phase_banner_slide_in_offset(step as i64);
        assert_eq!(got, want, "滑入第 {step} 步偏移 {got}，收據是 {want}");
    }
    let want_out = [0, -25, -50, -75, -100];
    assert_eq!(want_out.len() as i64, PHASE_BANNER_SLIDE_OUT_STEPS, "滑出步數");
    for (step, &want) in want_out.iter().enumerate() {
        let got = phase_banner_slide_out_offset(step as i64);
        assert_eq!(got, want, "滑出第 {step} 步偏移 {got}，收據是 {want}");
    }
    // 界外回停住的位置，不是憑空的偏移。
    for step in [-1, PHASE_BANNER_SLIDE_IN_STEPS] {
        let got = phase_banner_slide_in_offset(step);
        assert_eq!(got, 0, "滑入界外第 {step} 步回 {got}");
    }
    assert_eq!(PHASE_BANNER_SLIDE_ORIGIN_X, 0x55, "停住的 x 是 {PHASE_BANNER_SLIDE_ORIGIN_X}，原版是 0x55");
}
