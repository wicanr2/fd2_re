//! 移植自 Go `native_ch23.go`：第 23 章 `0x10652`／`0x24D22`／`0x4DFCC` 的暫存面與 DAC 循環。

use super::separated_surface::{SeparatedSurfaceDocument, read_document};
use super::{FdotherError, Frame, Result, decode_archive_single_frame, load_separated_single_frame};
use std::cell::Cell;
use std::path::Path;

/// `0x24D22` 與 `0x11EEE` 的 `dword_53C03==23` 分支使用的 raw `0x53AFF` 暫存幾何。
pub const NATIVE_CH23_STAGE_STRIDE: i64 = 312;
pub const NATIVE_CH23_STAGE_WIDTH: i64 = 312;
pub const NATIVE_CH23_STAGE_HEIGHT: i64 = 192;
/// `0x10652` 第 23 章分支經 raw `0x111ba` 呼叫傳入的 FDOTHER 封存項目。這是封存索引，不是語意素材名。
pub const NATIVE_CH23_STAGE_RESOURCE: i64 = 42;
/// `0x4DFCC` 寫入的 VGA DAC 索引。原始指令在 OUT 0x3c8 之前是 `mov ah,0xe0`，所以不是較早已撤回的
/// 解讀所說的低 0x20 調色盤窗口。
pub const NATIVE_CH23_PALETTE_BASE: i64 = 0xe0;

const STAGE_BYTES: usize = (NATIVE_CH23_STAGE_STRIDE * NATIVE_CH23_STAGE_HEIGHT) as usize;

fn stage_geometry_ok(frame: &Frame) -> bool {
    frame.x == 0 && frame.y == 0 && frame.width == NATIVE_CH23_STAGE_WIDTH && frame.height == NATIVE_CH23_STAGE_HEIGHT
}

/// 對應已證實的 `0x10652` 第 23 章載入邊界：FDOTHER #42 不經轉換讀入，再驗證為 `0x10809` 處
/// `0x4e63d` 消費的 312×192 單幀酬載。回傳的 [`Frame`] 只帶 raw 幾何與 RLE 位元組；不宣稱背景、
/// 轉場或 UI 角色。
pub fn decode_native_ch23_stage(dat_path: impl AsRef<Path>) -> Result<Frame> {
    let frame = decode_archive_single_frame(dat_path, NATIVE_CH23_STAGE_RESOURCE)?;
    if !stage_geometry_ok(&frame) {
        return Err(FdotherError::message(format!(
            "fdother: ch23 resource #{NATIVE_CH23_STAGE_RESOURCE} geometry is {}x{} at ({},{}), want {NATIVE_CH23_STAGE_WIDTH}x{NATIVE_CH23_STAGE_HEIGHT} at (0,0)",
            frame.width, frame.height, frame.x, frame.y
        )));
    }
    Ok(frame)
}

/// 載入由 FDOTHER #42 匯出的標準索引面。除了一般面契約之外，另驗證這章專屬的 raw 身分，絕不退回
/// 封存。
pub fn load_separated_native_ch23_stage(surface_root: impl AsRef<Path>) -> Result<Frame> {
    let surface_root = surface_root.as_ref();
    let metadata_path = surface_root.join("FDOTHER_042").join("resource.json");
    let document: SeparatedSurfaceDocument =
        read_document(&metadata_path, "fdother: separated ch23 metadata: ", "fdother: separated ch23 metadata: ")?;
    if document.source.raw_size != 59412
        || document.source.raw_md5 != "b9cd7793d8eec9c80bad5a364029a7a8"
        || document.source.raw_sha256 != "4bef756aaf78b95cf97949785c71b4c3e04822a497741f468e6863d61da38d2d"
    {
        return Err(FdotherError::fixed("fdother: separated ch23 raw identity mismatch"));
    }
    let frame = load_separated_single_frame(surface_root, "FDOTHER.DAT", NATIVE_CH23_STAGE_RESOURCE)?;
    if !stage_geometry_ok(&frame) {
        return Err(FdotherError::fixed("fdother: invalid separated ch23 stage geometry"));
    }
    Ok(frame)
}

/// 重現 `0x10809` 的 raw `0x4e63d` 呼叫。只接受 `0x107dd` 配置的 0xea00 位元組暫存面與已驗證的
/// 透明模式；不合法的輸入在寫入前拒收。
pub fn blit_native_ch23_stage(frame: &Frame, staging: &mut [u8]) -> Result<()> {
    if staging.len() != STAGE_BYTES {
        return Err(FdotherError::fixed("fdother: invalid ch23 staging surface"));
    }
    if !stage_geometry_ok(frame) {
        return Err(FdotherError::fixed("fdother: invalid ch23 stage frame geometry"));
    }
    // 先解到私有面。格式錯誤的 RLE 串流不可留下部分改寫的原生暫存緩衝。
    let mut next = staging.to_vec();
    frame.blit(&mut next, NATIVE_CH23_STAGE_STRIDE, -1)?;
    staging.copy_from_slice(&next);
    Ok(())
}

/// `0x4DFCC` 使用、位於線性 `0x60003` 的固定 31×RGB 位元組窗口。輔助函式取 3*byte_60002 的位元組
/// 偏移，把接下來 16 組 RGB 寫到 DAC 索引 0xe0..0xef。值是從固定版 FD2.EXE 複製的六位元 VGA 分量，
/// 不是由截圖推論。
const NATIVE_DAC_PALETTE_CYCLE_E0EF: [[u8; 3]; 31] = [
    [0x0e, 0x15, 0x26],
    [0x0d, 0x14, 0x25],
    [0x0d, 0x14, 0x25],
    [0x0d, 0x14, 0x25],
    [0x0c, 0x13, 0x24],
    [0x0c, 0x13, 0x24],
    [0x0b, 0x12, 0x23],
    [0x0b, 0x12, 0x23],
    [0x0b, 0x12, 0x23],
    [0x0b, 0x12, 0x23],
    [0x0c, 0x13, 0x24],
    [0x0c, 0x13, 0x24],
    [0x0d, 0x14, 0x25],
    [0x0e, 0x15, 0x26],
    [0x0e, 0x15, 0x26],
    [0x0e, 0x15, 0x26],
    [0x0e, 0x15, 0x26],
    [0x0d, 0x14, 0x25],
    [0x0d, 0x14, 0x25],
    [0x0d, 0x14, 0x25],
    [0x0c, 0x13, 0x24],
    [0x0c, 0x13, 0x24],
    [0x0b, 0x12, 0x23],
    [0x0b, 0x12, 0x23],
    [0x0b, 0x12, 0x23],
    [0x0b, 0x12, 0x23],
    [0x0c, 0x13, 0x24],
    [0x0c, 0x13, 0x24],
    [0x0d, 0x14, 0x25],
    [0x0e, 0x15, 0x26],
    [0x0e, 0x15, 0x26],
];

/// 重現 `0x24D22` 的 arg==0 列複製分支。最後 latch 列繞回頂端；其他列下移 latch。不合法的輸入在
/// 目的端改變前拒收，保留 raw 輔助函式的有界閘門。
///
/// Go 只要求 `len(buffer) >= 312*192`；比它長的緩衝只動前 312*192 位元組。
pub fn rotate_native_ch23_rows(buffer: &mut [u8], latch: i64) -> Result<()> {
    if buffer.len() < STAGE_BYTES || !(0..=NATIVE_CH23_STAGE_HEIGHT).contains(&latch) {
        return Err(FdotherError::fixed("fdother: invalid ch23 staging latch"));
    }
    if latch == 0 {
        return Ok(());
    }
    let (stride, height, latch) =
        (NATIVE_CH23_STAGE_STRIDE as usize, NATIVE_CH23_STAGE_HEIGHT as usize, latch as usize);
    let width = NATIVE_CH23_STAGE_WIDTH as usize;
    let top = buffer[(height - latch) * stride..height * stride].to_vec();
    for row in (0..height - latch).rev() {
        let (src, dst) = (row * stride, (row + latch) * stride);
        buffer.copy_within(src..src + width, dst);
    }
    buffer[..latch * stride].copy_from_slice(&top);
    Ok(())
}

/// 重現 `0x4DFCC` 的 16 項 DAC 寫入。只改調色盤索引 0xe0..0xef，其他項目不動。相位是 raw
/// byte_60002 值，必須在 0..15。
pub fn apply_native_dac_palette_cycle_e0ef(dac: &mut [u8], phase: i64) -> Result<()> {
    if dac.len() != 256 * 3 || !(0..=15).contains(&phase) {
        return Err(FdotherError::fixed("fdother: invalid native DAC E0..EF palette-cycle input"));
    }
    let base = (NATIVE_CH23_PALETTE_BASE * 3) as usize;
    for i in 0..16 * 3 {
        let k = phase as usize * 3 + i;
        dac[base + i] = NATIVE_DAC_PALETTE_CYCLE_E0EF[k / 3][k % 3];
    }
    Ok(())
}

/// 重現 `0x4DFCC` 的無號 BIOS 字閘門與行程期間的相位更新。只有至少經過兩個低字 tick 時 DAC 才
/// 改變；狀態不合法時每個輸入位元組都不動。
///
/// Go 版回傳 `(nextPhase, nextTimerTick, advanced, err)`；出錯時前三個值固定是輸入的 phase、
/// lastTimerTick 與 false，Rust 只回傳錯誤。
pub fn advance_native_dac_palette_cycle_e0ef(
    dac: &mut [u8],
    phase: i64,
    last_timer_tick: i64,
    raw_timer_tick: i64,
) -> Result<(i64, i64, bool)> {
    if dac.len() != 256 * 3
        || !(0..=15).contains(&phase)
        || !(-0x8000..=0x7fff).contains(&last_timer_tick)
        || !(-0x8000..=0x7fff).contains(&raw_timer_tick)
    {
        return Err(FdotherError::fixed("fdother: invalid native DAC cycle timing state"));
    }
    if (raw_timer_tick as u16).wrapping_sub(last_timer_tick as u16) < 2 {
        return Ok((phase, last_timer_tick, false));
    }
    let next_phase = (phase + 1) & 15;
    apply_native_dac_palette_cycle_e0ef(dac, next_phase)?;
    Ok((next_phase, raw_timer_tick, true))
}

/// raw ch23 處理器的一段固定排程。只描述 `0x24c61..0x24cf2` 的呼叫次數、`0x24d22` stage 值與原始 ESI
/// 參數；不把 `0x11cac` 的畫面內容或 `0x11d40` 的第三參數命名成遊戲語意。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCh23LoopSpec {
    pub phase: String,
    pub repeat: i64,
    pub stage_values: Vec<i64>,
    pub palette: bool,
}

/// 帶一個 raw 整數參數的 callback（latch 的 stage、palette 的 ESI）。
pub type NativeCh23ArgHook<'a> = Box<dyn FnMut(i64) -> Result<()> + 'a>;
/// 無參數的 callback（draw、tick）。
pub type NativeCh23Hook<'a> = Box<dyn FnMut() -> Result<()> + 'a>;

/// 尚未命名的原始呼叫端。呼叫者必須自行提供 `0x24d22` 的 raw latch setter、indexed draw、tick 與
/// （palette 段）`0x11d40` 的 raw ESI 消費端；缺任何 callback 就拒絕執行，避免把純資料排程誤接成
/// generic renderer。latch setter 只代表非零 byte 寫入；它不會在此排程器內偷偷執行 arg==0 的列旋轉，
/// 因為那是 BIOS tick 閘門下的共享消費端副作用，必須由有原始 state provenance 的 adapter 自行決定。
///
/// Go 的 nil callback 對應 `None`。
#[derive(Default)]
pub struct NativeCh23LoopHooks<'a> {
    pub latch: Option<NativeCh23ArgHook<'a>>,
    pub palette: Option<NativeCh23ArgHook<'a>>,
    pub draw: Option<NativeCh23Hook<'a>>,
    pub tick: Option<NativeCh23Hook<'a>>,
}

fn native_ch23_loop_spec_valid(spec: &NativeCh23LoopSpec) -> bool {
    match spec.phase.as_str() {
        "initial" => !spec.palette && spec.repeat == 30 && equal_native_ch23_stages(&spec.stage_values, 2, 9),
        "palette" => spec.palette && spec.repeat == 12 && equal_native_ch23_stages(&spec.stage_values, 10, 14),
        _ => false,
    }
}

impl NativeCh23LoopSpec {
    /// 回報這個具型別迴圈是否為 raw ch23 處理器接受的兩個精確排程之一。執行期轉接層用同一個閘門，
    /// 不重複 stage／repeat 契約。
    pub fn is_recovered_contract(&self) -> bool {
        native_ch23_loop_spec_valid(self)
    }
}

fn equal_native_ch23_stages(values: &[i64], first: i64, last: i64) -> bool {
    values.len() as i64 == last - first + 1 && values.iter().enumerate().all(|(i, &v)| v == first + i as i64)
}

/// 執行已證實的 raw staging 排程，但不發布畫面、不改 battle/campaign state，也不自行解釋 palette。
/// 每次呼叫前先複製 staging 與 DAC；任何 callback 或 shape 錯誤都回復兩個 buffer，保留原子拒絕。
///
/// Go 的 callback 捕捉同一個 `staging`／`dac` 切片，可以在排程途中直接改寫，失敗時再由這裡回復。
/// Rust 以 `&[Cell<u8>]` 保留這種共用別名；呼叫端手上若是 `&mut [u8]`，用
/// `Cell::from_mut(slice).as_slice_of_cells()` 轉換。
pub fn run_native_ch23_loop(
    spec: &NativeCh23LoopSpec,
    staging: &[Cell<u8>],
    dac: &[Cell<u8>],
    hooks: &mut NativeCh23LoopHooks,
) -> Result<()> {
    if !native_ch23_loop_spec_valid(spec) {
        return Err(FdotherError::fixed("fdother: invalid ch23 raw loop spec"));
    }
    let (true, Some(latch), Some(draw), Some(tick)) =
        (staging.len() == STAGE_BYTES, hooks.latch.as_mut(), hooks.draw.as_mut(), hooks.tick.as_mut())
    else {
        return Err(FdotherError::fixed("fdother: ch23 loop requires exact latch/draw/tick callbacks"));
    };
    if spec.palette && (hooks.palette.is_none() || dac.len() != 256 * 3) {
        return Err(FdotherError::fixed("fdother: ch23 palette loop requires raw ESI callback and DAC"));
    }
    let before_stage: Vec<u8> = staging.iter().map(Cell::get).collect();
    let before_dac: Vec<u8> = dac.iter().map(Cell::get).collect();
    let rollback = |err: FdotherError| {
        for (cell, &v) in staging.iter().zip(&before_stage) {
            cell.set(v);
        }
        for (cell, &v) in dac.iter().zip(&before_dac) {
            cell.set(v);
        }
        Err(err)
    };
    let mut raw_esi = 0i64;
    for &stage in &spec.stage_values {
        // 原生 0x24c81/0x24cf2 只呼叫 0x24d22(stage)，其非零分支寫入 byte_0x51a10。不要推論 staging
        // 列已在此旋轉：0x11eee 的 BIOS tick 消費端可能稍後呼叫 0x24d22(0)，那個時序與狀態歸轉接層。
        if let Err(e) = latch(stage) {
            return rollback(e);
        }
        for _ in 0..spec.repeat {
            if spec.palette {
                let palette = hooks.palette.as_mut().expect("palette 段已檢查 callback");
                if let Err(e) = palette(raw_esi) {
                    return rollback(e);
                }
                raw_esi += 1;
            }
            if let Err(e) = draw() {
                return rollback(e);
            }
            if let Err(e) = tick() {
                return rollback(e);
            }
        }
    }
    Ok(())
}
