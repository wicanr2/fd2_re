//! 移植自 Go `indexedmap/mosaic.go`。

use super::frame::{NATIVE_MAP_VGA_SIZE, STEADY_VIEWPORT_WIDTH, VIEW_HEIGHT, VIEW_WIDTH};
use super::{Result, at, err, len64};

/// PhaseBannerSteps 是原版回合橫幅的步數。
///
/// dosgolem 逐幀收據（docs/data/ui-traces/fd2-phase-banner-20260909.json）：
/// 33 張畫面的馬賽克方塊邊長是 `1 + min(f, 32-f)`，由 1 漲到 17 再縮回 1；
/// 第 f 張與第 32-f 張逐位元組相同，第 32 張與第 0 張相同。整段期間地圖完全
/// 不重繪，所以馬賽克是對**保留下來的已合成畫面**每一步重算，不是逐步累積。
/// `PLAYER PHASE` 與 `ENEMY PHASE` 共用同一段排程。
///
/// 33 步是兩支函式接起來的：`sub_1F1CC` 進場 16 步（邊長 1..16），`sub_1F30A`
/// 退場 17 步（邊長 17..1），中間隔一次 `delay()`。原始指令見
/// docs/data/fd2_phase_banner_presentation_disasm.txt，量測見
/// docs/knowledge-base/105-phase-banner-timing-20260910.md。
pub const PHASE_BANNER_STEPS: i64 = 33;

/// PhaseBannerEnterSteps 是 `sub_1F1CC` 進場迴圈的步數（`ebp` ＝ 1..16）。
/// 之後才是中段 `delay()`，再進 `sub_1F30A` 的 PhaseBannerExitSteps 步。
pub const PHASE_BANNER_ENTER_STEPS: i64 = 16;

/// PhaseBannerExitSteps 是 `sub_1F30A` 退場迴圈的步數（`ebp` 由 0x11 遞減）。
pub const PHASE_BANNER_EXIT_STEPS: i64 = PHASE_BANNER_STEPS - PHASE_BANNER_ENTER_STEPS;

/// PhaseBannerStepTickHz 是每一步停留時間的來源：`sub_17AA9(1)` 輪詢 BIOS tick
/// 計數 0000:046C 直到差值 ≥ 1，所以每步固定一個 BIOS tick。
pub const PHASE_BANNER_STEP_TICK_HZ: f64 = 18.2065097;

/// PhaseBannerStepMillis 是一個 BIOS tick 的毫秒數（約 54.925）。
///
/// 這個節奏不依賴 CPU 速度，前提是畫一步的時間小於一個 tick。dosgolem 上這個
/// 前提不成立（虛擬 CPU 每指令 1 微秒，畫一步約 356,000 指令），所以收據裡的
/// 步距是模擬器工作量，不是原版節奏；節奏寫在 `sub_17AA9` 的參數裡。
///
/// Go 的 `1000.0 / PhaseBannerStepTickHz` 是無型別常數運算：以任意精度算出 1000／18.2065097 的精確商
/// 再捨入成 `float64`。Rust 的 `f64` 除法用的是已捨入的 18.2065097，結果差 1 ulp，所以這裡直接寫 Go
/// 常數的捨入結果（精確商的最近 `f64`，黃金差分逐位元核對）。
pub const PHASE_BANNER_STEP_MILLIS: f64 = 54.925409453960306;

/// PhaseBannerDim 回傳第 step 步要套在 DAC 上的減量。
///
/// 原版每一步都呼叫 `sub_11D40(0x10, 0xFF, level)`：索引 0x10..0xFF 的 R／G／B
/// 各從主調色盤減去 level 並夾在 0。level 就是迴圈變數（進場 0..15、退場
/// 16..0），也就是方塊邊長減一。索引 0x00..0x0F 不動，橫幅字樣因此不變暗。
///
/// Go：`mosaic.go:44`
pub fn phase_banner_dim(step: i64) -> i64 {
    if !(0..PHASE_BANNER_STEPS).contains(&step) {
        return 0;
    }
    phase_banner_block(step) - 1
}

/// PhaseBannerDimRange 是 `sub_11D40` 在橫幅期間套用的 DAC 索引區間。
pub const PHASE_BANNER_DIM_FIRST_INDEX: i64 = 0x10;
pub const PHASE_BANNER_DIM_LAST_INDEX: i64 = 0xFF;

// 字樣滑入與滑出。`sub_1F1CC` 在馬賽克之前跑七次 `sub_1F42D`（參數
// 100／75／50／25／0／1／0），`sub_1F30A` 在重繪地形之後跑五次（0／25／50／
// 75／100）；每一步同樣以 `sub_17AA9(1)` 等一個 BIOS tick。
//
// 每一步畫的是兩塊，而且**相向移動**：第一塊（`PLAYER` 或 `ENEMY`）在
// x ＝ `0x55 − 參數`，第二塊（固定 `PHASE`）在 x ＝ `參數 + 0xA5`，y 都是
// `0x52`（`sub_1F42D` 0x1F43F 的 `mov ebx, 0x55` 與 0x1F473 的 `add eax, 0xa5`）。
// 那是離屏緩衝區的座標，`sub_11EB0` 再整塊搬到 VGA 的 (4,4)，所以換算成 VGA
// 就是停住時的 (0x59,0x56) 與 (0xA9,0x56)——與馬賽克期間那兩塊同一個位置。
// 量測見 docs/knowledge-base/105-phase-banner-timing-20260910.md。
const PHASE_BANNER_SLIDE_IN_ARGS: [i64; 7] = [100, 75, 50, 25, 0, 1, 0];
const PHASE_BANNER_SLIDE_OUT_ARGS: [i64; 5] = [0, 25, 50, 75, 100];

/// PhaseBannerSlideInSteps 與 PhaseBannerSlideOutSteps 是兩段滑動的步數。
pub const PHASE_BANNER_SLIDE_IN_STEPS: i64 = PHASE_BANNER_SLIDE_IN_ARGS.len() as i64;
pub const PHASE_BANNER_SLIDE_OUT_STEPS: i64 = PHASE_BANNER_SLIDE_OUT_ARGS.len() as i64;

/// PhaseBannerSlideOriginX 是字樣停住時的原版 x（`0x55`）。滑動的每一步都相對
/// 它偏移，偏移量就是 `sub_1F42D` 的參數。
pub const PHASE_BANNER_SLIDE_ORIGIN_X: i64 = 0x55;

/// PhaseBannerSlideInOffset 回傳滑入第 step 步**第一塊**相對停住位置的 x 偏移
/// （負值代表還在左邊畫面外）。第二塊取相反數。界外回 0，也就是停住的位置。
///
/// Go：`mosaic.go:88`
pub fn phase_banner_slide_in_offset(step: i64) -> i64 {
    if !(0..PHASE_BANNER_SLIDE_IN_STEPS).contains(&step) {
        return 0;
    }
    -PHASE_BANNER_SLIDE_IN_ARGS[at(step)]
}

/// PhaseBannerSlideOutOffset 回傳滑出第 step 步的 x 偏移。
///
/// Go：`mosaic.go:96`
pub fn phase_banner_slide_out_offset(step: i64) -> i64 {
    if !(0..PHASE_BANNER_SLIDE_OUT_STEPS).contains(&step) {
        return 0;
    }
    -PHASE_BANNER_SLIDE_OUT_ARGS[at(step)]
}

/// PhaseBannerBlock 回傳第 step 步的方塊邊長。
///
/// Go：`mosaic.go:104`
pub fn phase_banner_block(step: i64) -> i64 {
    if !(0..PHASE_BANNER_STEPS).contains(&step) {
        return 1;
    }
    let other = PHASE_BANNER_STEPS - 1 - step;
    if other < step {
        return 1 + other;
    }
    1 + step
}

/// MosaicNativeMapViewport 把 src 的 312×192 地圖窗格以 block 為邊長馬賽克化後
/// 寫進 dst，窗格外（四邊各 4 px 的黑邊）原樣保留。
///
/// 取樣規則是**方塊左上角**那一個來源像素：對邊長 4／8／16 逐方塊比對原始幀，
/// 1872/1872、468/468、114/114 全中，中心與右下角則只有個位數百分比。格線原點
/// 就是窗格原點 (4,4)。block 為 1 時等同整幀複製。
///
/// Go：`mosaic.go:120`
pub fn mosaic_native_map_viewport(dst: &mut [u8], src: &[u8], block: i64) -> Result<()> {
    if len64(dst) < NATIVE_MAP_VGA_SIZE || len64(src) < NATIVE_MAP_VGA_SIZE {
        return Err(err("indexedmap: mosaic buffers are smaller than one native frame"));
    }
    if !(1..=STEADY_VIEWPORT_WIDTH).contains(&block) {
        return Err(err("indexedmap: mosaic block is outside the viewport"));
    }
    let size = at(NATIVE_MAP_VGA_SIZE);
    dst[..size].copy_from_slice(&src[..size]);
    if block == 1 {
        return Ok(());
    }
    const ORIGIN_X: i64 = 4;
    const ORIGIN_Y: i64 = 4;
    let mut y = 0;
    while y < VIEW_HEIGHT {
        let mut x = 0;
        while x < STEADY_VIEWPORT_WIDTH {
            let v = src[at((ORIGIN_Y + y) * VIEW_WIDTH + ORIGIN_X + x)];
            let mut dy = 0;
            while dy < block && y + dy < VIEW_HEIGHT {
                let row = (ORIGIN_Y + y + dy) * VIEW_WIDTH;
                let mut dx = 0;
                while dx < block && x + dx < STEADY_VIEWPORT_WIDTH {
                    dst[at(row + ORIGIN_X + x + dx)] = v;
                    dx += 1;
                }
                dy += 1;
            }
            x += block;
        }
        y += block;
    }
    Ok(())
}

#[cfg(test)]
mod mosaic_test;
