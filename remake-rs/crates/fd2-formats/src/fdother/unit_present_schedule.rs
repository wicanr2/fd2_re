//! 移植自 Go `unit_present_schedule.go`：`0x22253` 的單位出現／消失呈現。

use super::indexed_transition::{apply_indexed_transition_pass_at, validate_radial_geometry, validate_rect_geometry};
use super::separated_frame_bank::wrap_go;
use super::{CenteredRectLUTRemap, FdotherError, IndexedTransitionPass, LMI1Entry, RadialLUTRemap, Result};

pub(super) const NATIVE_UNIT_PRESENT_STRIDE: i64 = 0x1c8;
const NATIVE_UNIT_PRESENT_TILE_PIXELS: i64 = 0x18;
pub(super) const NATIVE_UNIT_PRESENT_BASE: i64 = 0x8088;
/// Go `nativeUnitPresentWorkBytes`（未匯出）；Go 測試直接引用，這裡公開給黃金差分建構輸入。
pub const NATIVE_UNIT_PRESENT_WORK_BYTES: i64 = 0x25680;

/// `0x22470` 的目的位址式。第一個 LMI 階段收到地圖座標 x/y，寫在可見地形原點下一格列。回傳原生
/// stride 456 索引工作緩衝內的位元組偏移，不自創畫面錨點。負值照樣保留：視窗有效性歸原版呼叫端。
/// 乘加與 Go 一樣以 64 位元繞回。
pub fn native_unit_present_byte_origin(x: i64, y: i64, camera_x: i64, camera_y: i64) -> i64 {
    NATIVE_UNIT_PRESENT_BASE
        .wrapping_add(NATIVE_UNIT_PRESENT_TILE_PIXELS.wrapping_mul(x.wrapping_sub(camera_x)))
        .wrapping_add(
            (NATIVE_UNIT_PRESENT_TILE_PIXELS * NATIVE_UNIT_PRESENT_STRIDE).wrapping_mul(y.wrapping_sub(camera_y)),
        )
        .wrapping_add(NATIVE_UNIT_PRESENT_STRIDE)
}

/// `0x22547` 的中段垂直界線。較早的轉接層把它誤標成變動的半徑；直接把參數對到 `0x22046` 證明半徑
/// 是另一個常值 11。原生計算 centerY=24*raw53ABD+15，startY=trunc(centerY/5)*LUTIndex。
/// 乘加與 Go 一樣以 64 位元繞回；Go 的 `/` 向零截斷，與 Rust 相同。
pub fn native_unit_present_contract_start_y(raw_53abd: i64, lut_index: i64) -> Result<i64> {
    if !(0..=5).contains(&lut_index) {
        return Err(FdotherError::fixed("fdother: unit-present LUT index is outside 0..5"));
    }
    Ok((24i64.wrapping_mul(raw_53abd).wrapping_add(15) / 5).wrapping_mul(lut_index))
}

/// 把一次 `0x22547`／`0x22656` 呼叫對到 `0x22046` 的兩次放射狀加一個矩形的幾何。兩次放射狀都用
/// 半徑 11、比例 16。第一次涵蓋 startY..192，中間重繪，第二次涵蓋 centerY..192，最後寬 34 的矩形
/// 涵蓋 startY..centerY。可見寬度是原生 312 位元組視窗。
pub fn native_unit_present_lut_pass(center_x: i64, center_y: i64, start_y: i64) -> Result<IndexedTransitionPass> {
    let radial =
        |start_y| RadialLUTRemap { center_x, center_y, radius: 11, scale: 16, start_y, end_y: 192, clip_width: 312 };
    let pass = IndexedTransitionPass {
        first_radial: radial(start_y),
        second_radial: radial(center_y),
        final_rect: CenteredRectLUTRemap { center_x, horizontal_radius: 17, start_y, end_y: center_y, clip_width: 312 },
    };
    // Go 以 make([]byte, 456*192) 與 make([]byte, 256) 呼叫驗證；只用到長度。
    let (n, lut) = ((NATIVE_UNIT_PRESENT_STRIDE * 192) as usize, [0u8; 256]);
    validate_radial_geometry(n, NATIVE_UNIT_PRESENT_STRIDE, &lut, pass.first_radial)
        .map_err(|e| wrap_go(e, "fdother: invalid unit-present first geometry: ".to_string()))?;
    validate_radial_geometry(n, NATIVE_UNIT_PRESENT_STRIDE, &lut, pass.second_radial)
        .map_err(|e| wrap_go(e, "fdother: invalid unit-present second geometry: ".to_string()))?;
    validate_rect_geometry(n, NATIVE_UNIT_PRESENT_STRIDE, &lut, pass.final_rect)
        .map_err(|e| wrap_go(e, "fdother: invalid unit-present rectangle geometry: ".to_string()))?;
    Ok(pass)
}

/// 重現 `0x22470` 在那個階段重繪單位並呈現 320x192 視窗之前的一次 LMI 格寫入。原生 `0x4e85b` 經
/// `0x4e916` 解碼，解出零時保留目的端像素；[`LMI1Entry::blit_at`] 正是這條透明規則。
///
/// 呼叫端提供完整原生工作緩衝（不是裁切後的視窗）。畫面外的原點拒收，不模擬原生未檢查的記憶體
/// 寫入；未來的繪圖器必須明定裁切政策。
pub fn blit_native_unit_present_lmi(
    entry: &LMI1Entry,
    dst: &mut [u8],
    x: i64,
    y: i64,
    camera_x: i64,
    camera_y: i64,
) -> Result<()> {
    let origin = native_unit_present_byte_origin(x, y, camera_x, camera_y);
    if origin < 0 || origin >= dst.len() as i64 {
        return Err(FdotherError::fixed("fdother: unit-present LMI origin is outside destination"));
    }
    entry.blit_at(
        dst,
        NATIVE_UNIT_PRESENT_STRIDE,
        origin % NATIVE_UNIT_PRESENT_STRIDE,
        origin / NATIVE_UNIT_PRESENT_STRIDE,
        false,
    )
}

/// `0x22470` 每次寫入之後的「重繪單位、呈現、等一個 BIOS tick」callback。Go 版是無參數的
/// `func() error`，閉包捕捉同一個 `dst` 切片來重繪；Rust 不能讓閉包另外持有可變借用，所以把
/// `dst` 交給 callback。nil 時拒收；Rust 以 `None` 保留那條分支。
pub type NativeUnitPresentTick<'a> = &'a mut dyn FnMut(&mut [u8]) -> Result<()>;

/// 執行 `0x22470` 的前 11 次呈現。每個 FDOTHER#6 項目 0x72..0x7c 都寫在同一個還原出的目的端，之後
/// `redraw_present_tick` 必須執行原生中間的單位重繪、320x192 呈現與一個 BIOS tick。callback 是必要的，
/// 避免這個輔助函式悄悄把 11 次呈現合併。
#[allow(clippy::too_many_arguments)]
pub fn run_native_unit_present_lmi_intro(
    entries: &[LMI1Entry],
    dst: &mut [u8],
    x: i64,
    y: i64,
    camera_x: i64,
    camera_y: i64,
    redraw_present_tick: Option<NativeUnitPresentTick>,
) -> Result<()> {
    let (true, Some(tick)) = (entries.len() > 0x7c, redraw_present_tick) else {
        return Err(FdotherError::fixed("fdother: incomplete unit-present LMI intro adapter"));
    };
    for (index, entry) in entries.iter().enumerate().take(0x7d).skip(0x72) {
        blit_native_unit_present_lmi(entry, dst, x, y, camera_x, camera_y)
            .map_err(|e| wrap_go(e, format!("fdother: unit-present LMI entry {index:#x}: ")))?;
        tick(dst).map_err(|e| wrap_go(e, format!("fdother: unit-present LMI entry {index:#x} present: ")))?;
    }
    Ok(())
}

/// `0x22253` 裡一個完整視窗 `0x11eb0` 呈現邊界。收縮與釋放之間另外還原出的 18／24 條漸進直寫 VGA
/// 列刻意不併入這份清單。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct UnitPresentStep {
    pub phase: String,
    pub resource: i64,
    pub entry: i64,
    pub delay_ms: i64,
    pub delay_ticks: i64,
}

/// `0x22253` 的五參數 ABI。`visual_x`/`visual_y` 是開場、收縮與直寫 VGA 橋接的錨點。
/// `new_x`/`new_y` 只在收縮之後寫入 runtime unit+0/+1。原生以 0xff 當地圖外的改寫值；視覺錨點仍由
/// 呼叫端另外提供。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeUnitPresentCall {
    pub unit_slot: i64,
    pub new_x: u8,
    pub new_y: u8,
    pub visual_x: u8,
    pub visual_y: u8,
}

/// 保留位元組截斷界線，同時拒收可編輯劇本不可能透過還原出的五個整數參數提供的值。
pub fn plan_native_unit_present_call(
    unit_slot: i64,
    new_x: i64,
    new_y: i64,
    visual_x: i64,
    visual_y: i64,
) -> Result<NativeUnitPresentCall> {
    if [unit_slot, new_x, new_y, visual_x, visual_y].iter().any(|v| !(0..=0xff).contains(v)) {
        return Err(FdotherError::fixed("fdother: unit-present call argument outside byte range"));
    }
    Ok(NativeUnitPresentCall {
        unit_slot,
        new_x: new_x as u8,
        new_y: new_y as u8,
        visual_x: visual_x as u8,
        visual_y: visual_y as u8,
    })
}

/// LMI 開場之後 `0x22253` 的十六次 `0x22046` 呼叫之一。`Raw53AB9`／`Raw53ABD` 維持以位址命名：程式
/// 證明了它們的像素中心算術，但沒有證明較高層的玩法名稱。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct UnitPresentLUTFrame {
    pub phase: String,
    pub lut_index: i64,
    pub pass: IndexedTransitionPass,
    pub delay_ms: i64,
    pub delay_ticks: i64,
}

/// 實體化六次收縮與十次釋放的幾何。收縮幀把第一次處理與矩形的起始界線移動
/// floor(centerY/5)*index；釋放幀一律從第 0 列開始。乘加與 Go 一樣以 64 位元繞回。
pub fn native_unit_present_lut_frames(raw_53ab9: i64, raw_53abd: i64) -> Result<Vec<UnitPresentLUTFrame>> {
    let center_x = 24i64.wrapping_mul(raw_53ab9).wrapping_add(12);
    let center_y = 24i64.wrapping_mul(raw_53abd).wrapping_add(15);
    let mut frames = Vec::with_capacity(16);
    for index in (0..=5).rev() {
        let start_y = native_unit_present_contract_start_y(raw_53abd, index)?;
        let pass = native_unit_present_lut_pass(center_x, center_y, start_y)?;
        frames.push(UnitPresentLUTFrame {
            phase: "lut_contract".to_string(),
            lut_index: index,
            pass,
            delay_ms: 10,
            delay_ticks: if index == 0 { 2 } else { 0 },
        });
    }
    for index in 0..10 {
        let pass = native_unit_present_lut_pass(center_x, center_y, 0)?;
        frames.push(UnitPresentLUTFrame {
            phase: "lut_release".to_string(),
            lut_index: index,
            pass,
            delay_ms: 0,
            delay_ticks: 1,
        });
    }
    Ok(frames)
}

/// `run_native_unit_present_lut_frame` 的重繪 callback：收到完整工作緩衝。
pub type NativeUnitPresentRedraw<'a> = &'a mut dyn FnMut(&mut [u8]) -> Result<()>;
/// `run_native_unit_present_lut_frame` 的呈現 callback：收到完整工作緩衝與這一幀。
pub type NativeUnitPresentPresent<'a> = &'a mut dyn FnMut(&mut [u8], &UnitPresentLUTFrame) -> Result<()>;

/// 重現一次 `0x22547`／`0x22656` 緩衝交易。原生在每幀之前還原全部 0x25680 位元組，在
/// work+0x8088 套用 `0x22046`，兩次放射狀處理之間必須重繪物件，然後呈現。callback 收到完整工作
/// 緩衝而不是裁切後的視窗，所以單位重繪不會意外寫到分離的切片。
pub fn run_native_unit_present_lut_frame(
    work: &mut [u8],
    snapshot: &[u8],
    lut: &[u8],
    frame: &UnitPresentLUTFrame,
    redraw: Option<NativeUnitPresentRedraw>,
    present: Option<NativeUnitPresentPresent>,
) -> Result<()> {
    if work.len() as i64 != NATIVE_UNIT_PRESENT_WORK_BYTES || snapshot.len() as i64 != NATIVE_UNIT_PRESENT_WORK_BYTES {
        return Err(FdotherError::fixed("fdother: unit-present work/snapshot must be exactly 0x25680 bytes"));
    }
    let (Some(redraw), Some(present), true) = (redraw, present, lut.len() == 256) else {
        return Err(FdotherError::fixed("fdother: incomplete unit-present LUT frame adapter"));
    };
    work.copy_from_slice(snapshot);
    apply_indexed_transition_pass_at(
        work,
        NATIVE_UNIT_PRESENT_BASE as usize,
        NATIVE_UNIT_PRESENT_STRIDE,
        lut,
        frame.pass,
        redraw,
    )
    .map_err(|e| wrap_go(e, format!("fdother: unit-present {} LUT {}: ", frame.phase, frame.lut_index)))?;
    present(work, frame)
        .map_err(|e| wrap_go(e, format!("fdother: unit-present {} LUT {} present: ", frame.phase, frame.lut_index)))
}

/// 回傳 27 個完整視窗呈現邊界：`0x22470` 的 11 個 FDOTHER#6 LMI 格、`0x22547` 的六次 FDOTHER#3
/// LUT 重映射，以及 `0x22656` 的十次 FDOTHER#3 LUT 重映射。中段之後的兩個 BIOS tick 以最後一次
/// 呈現的 `delay_ticks` 表示，所以它們的位置不會被意外挪到第六次呈現之前。中段直寫 VGA 橋接另外
/// 增加 18 或 24 條可觀察的列寫入，歸 indexedmap。
pub fn native_unit_present_schedule() -> Vec<UnitPresentStep> {
    let step = |phase: &str, resource, entry, delay_ms, delay_ticks| UnitPresentStep {
        phase: phase.to_string(),
        resource,
        entry,
        delay_ms,
        delay_ticks,
    };
    let mut steps = Vec::with_capacity(27);
    for entry in 0x72..=0x7c {
        steps.push(step("lmi_intro", 6, entry, 0, 1));
    }
    for entry in (0..=5).rev() {
        steps.push(step("lut_contract", 3, entry, 10, if entry == 0 { 2 } else { 0 }));
    }
    for entry in 0..=9 {
        steps.push(step("lut_release", 3, entry, 0, 1));
    }
    steps
}

/// 拒收所有對已還原 11+6+10 編排的近似。用途是轉接層在改寫索引幀緩衝之前的邊界檢查。
pub fn validate_native_unit_present_schedule(steps: &[UnitPresentStep]) -> Result<()> {
    let want = native_unit_present_schedule();
    if steps.len() != want.len() {
        return Err(FdotherError::message(format!(
            "fdother: unit-present schedule has {} steps, want {}",
            steps.len(),
            want.len()
        )));
    }
    for (i, (got, want)) in steps.iter().zip(&want).enumerate() {
        if got != want {
            return Err(FdotherError::message(format!(
                "fdother: unit-present step {i} does not match native schedule"
            )));
        }
    }
    Ok(())
}
