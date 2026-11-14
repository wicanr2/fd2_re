//! 移植自 Go `indexedmap/frame.go`。

use super::{
    Error, NativeMapHUDFrames, NativeMapHUDInput, Result, at, blit_native_map_hud, err, len64, to_fdother, wrap,
};
use fd2_formats::fdicon::{
    self, Bank, NATIVE_MAP_STRIDE, NATIVE_MOTION_PIXELS, NativeForegroundLayerEntry, NativeSelectorCache,
    NativeTerrainCell, NativeUnitLayerEntry,
};
use fd2_formats::fdother::{
    self, IndexedTransitionPass, LMI1Entry, NATIVE_TRANSITION_PRESENT_STRIDE, NATIVE_TRANSITION_STAGE_HEIGHT,
    NATIVE_TRANSITION_STAGE_OFFSET, NATIVE_TRANSITION_STAGE_STRIDE, NATIVE_TRANSITION_STAGE_WIDTH,
    NativeChapterAuxSurface, NativeMapParallaxSurface, UnitPresentLUTFrame,
};

pub(super) const WORK_STRIDE: i64 = NATIVE_MAP_STRIDE;
pub(super) const WORK_BASE: i64 = 0x8088;
/// VGA row stride, not the steady copied width
pub(super) const VIEW_WIDTH: i64 = 320;
pub(super) const VIEW_HEIGHT: i64 = 192;
// 0x11cac calls 0x11eb0(dst=A000:0504, dstStride=320,
// src=work+0x8088, srcStride=456, width=312, height=192).
pub(super) const STEADY_VIEWPORT_OFFSET: i64 = 0x504;
pub(super) const STEADY_VIEWPORT_WIDTH: i64 = 312;
pub const NATIVE_MAP_VGA_SIZE: i64 = 320 * 200;
/// NativeMapViewport… 是 0x11EB0 實際搬到 VGA 的矩形（含端點）。窗格外那四欄／四列留著先前的
/// 內容——原版的橫幅字樣畫在離屏緩衝區裡，搬過去時超出窗格的部分就不會出現在畫面上。
pub const NATIVE_MAP_VIEWPORT_X0: i64 = STEADY_VIEWPORT_OFFSET % VIEW_WIDTH;
pub const NATIVE_MAP_VIEWPORT_Y0: i64 = STEADY_VIEWPORT_OFFSET / VIEW_WIDTH;
pub const NATIVE_MAP_VIEWPORT_X1: i64 = NATIVE_MAP_VIEWPORT_X0 + STEADY_VIEWPORT_WIDTH - 1;
pub const NATIVE_MAP_VIEWPORT_Y1: i64 = NATIVE_MAP_VIEWPORT_Y0 + VIEW_HEIGHT - 1;
/// NativeUnitPresentWorkSize is 0x22253's exact temporary allocation.
pub const NATIVE_UNIT_PRESENT_WORK_SIZE: i64 = 0x25680;

/// SeedNativeCh23Staging reproduces the zero-offset chapter-23 case of
/// 0x11EEE before the shared terrain/range/object/HUD pipeline runs. The raw
/// source is exactly 312x192 with stride 312; the destination begins at
/// work+0x8088 with stride 456. The helper commits atomically and deliberately
/// does not name the staging pixels as a background or transition.
///
/// Go：`frame.go:40`
pub fn seed_native_ch23_staging(work: &mut [u8], staging: &[u8]) -> Result<()> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE || len64(staging) != 312 * VIEW_HEIGHT {
        return Err(err("indexedmap: incomplete native ch23 staging buffers"));
    }
    let last = WORK_BASE + (VIEW_HEIGHT - 1) * WORK_STRIDE + STEADY_VIEWPORT_WIDTH;
    if last > len64(work) {
        return Err(err("indexedmap: native ch23 staging exceeds work buffer"));
    }
    let mut next = work.to_vec();
    let width = at(STEADY_VIEWPORT_WIDTH);
    for row in 0..VIEW_HEIGHT {
        let src = at(row * STEADY_VIEWPORT_WIDTH);
        let dst = at(WORK_BASE + row * WORK_STRIDE);
        next[dst..dst + width].copy_from_slice(&staging[src..src + width]);
    }
    work.copy_from_slice(&next);
    Ok(())
}

/// [`FrameInput::selection_overlay`] 的型別：收到整個工作緩衝、stride 456 與 viewport 基底位移。
pub type SelectionOverlay<'a> = &'a dyn Fn(&mut [u8], i64, i64) -> Result<()>;

/// FrameInput is the raw, already-selected input required by 0x11cac's normal
/// redraw.  It deliberately keeps all native selectors raw: the caller owns
/// resource selection, map lifetime, palette phase, and unit materialization.
///
/// Go：`frame.go:61`。所有權：Go 的指標欄位是呼叫端擁有的資源，這裡借用；`cells` 見模組文件。
#[derive(Clone, Default)]
pub struct FrameInput<'a> {
    pub terrain_bank: Option<&'a Bank>,
    pub range_bank: Option<&'a Bank>,
    pub unit_bank: Option<&'a Bank>,
    pub foreground_bank: Option<&'a Bank>,
    pub selector_cache: Option<&'a NativeSelectorCache>,
    pub cells: Vec<NativeTerrainCell>,
    pub controls: &'a [u8],
    pub lut: &'a [u8],
    pub map_width: i64,
    pub camera_x: i64,
    pub camera_y: i64,
    pub flip: i64,
    pub terrain_cycle: i64,
    pub idle_cycle: i64,
    pub moving_cycle: i64,
    pub pixel_shift: i64,
    pub range_mode: i64,
    pub cursor_x: i64,
    pub cursor_y: i64,
    pub units: Vec<NativeUnitLayerEntry>,
    pub foreground_units: Vec<NativeForegroundLayerEntry>,
    pub parallax: Option<&'a NativeMapParallaxSurface>,
    pub parallax_scroll_x: i64,
    pub parallax_scroll_y: i64,
    pub chapter_aux: Option<&'a NativeChapterAuxSurface>,
    pub chapter_aux_phase: i64,
    /// SelectionOverlay 是選單位之後的重繪比 0x11CAC 多出的一步：0x18B84 在前景
    /// 之後、HUD 之前對 viewport 呼叫 0x18C6D 畫單位資訊視窗；指令環（0x1741C）
    /// 則把四張圖示貼在同一個時點的畫面上。nil 就是 0x11CAC 本來的排程。
    /// frame 是整個工作緩衝，stride 456，viewportBase 是 work+0x8088 的位移。
    pub selection_overlay: Option<SelectionOverlay<'a>>,
}

/// NativeFrameInput is the complete, directly composable steady redraw slice.
/// It binds the separately verified 0x1acf3 resources/input to 0x11cac's
/// terrain/range/unit/foreground scheduler without allowing a caller to swap
/// in an approximation at the HUD boundary.
///
/// Go：`frame.go:87`
#[derive(Clone, Default)]
pub struct NativeFrameInput<'a> {
    pub frame: FrameInput<'a>,
    pub hud: NativeMapHUDInput,
    pub frames: NativeMapHUDFrames,
    pub hud_terrain: Option<&'a Bank>,
    pub hud_units: Option<&'a Bank>,
    pub hud_cache: Option<&'a NativeSelectorCache>,
}

/// FrameStage identifies one completed pass of the recovered steady native
/// map compositor.  It is exposed only so evidence generators and regression
/// tests can distinguish admission from a later overwrite; production callers
/// continue to use ComposeFrame or ComposeNativeFrame without an observer.
///
/// Go：`frame.go:99`，`type FrameStage uint8`（iota 常數）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct FrameStage(pub u8);

pub const FRAME_STAGE_TERRAIN: FrameStage = FrameStage(0);
pub const FRAME_STAGE_RANGE: FrameStage = FrameStage(1);
pub const FRAME_STAGE_UNITS: FrameStage = FrameStage(2);
pub const FRAME_STAGE_FOREGROUND: FrameStage = FrameStage(3);
pub const FRAME_STAGE_HUD: FrameStage = FrameStage(4);
pub const FRAME_STAGE_VIEWPORT: FrameStage = FrameStage(5);

/// FrameObserver receives detached work/VGA snapshots after each completed
/// pass. Mutating either slice cannot affect the compositor transaction.
///
/// Go：`frame.go:112`
pub type FrameObserver<'a> = &'a mut dyn FnMut(FrameStage, &mut [u8], &mut [u8]) -> Result<()>;

/// ComposeFrame 的 HUD callback（Go `func([]byte) error`）。
pub type RenderHUD<'a> = &'a mut dyn FnMut(&mut [u8]) -> Result<()>;

/// NativeTransitionFrameInput is the raw layer subset used by 0x24618. Unlike
/// the steady tactical redraw, the transition starts with terrain only; the
/// middle 0x127a9 callback adds unit and foreground layers between two LUT
/// remaps. Range and HUD are intentionally absent from this contract.
///
/// Go：`frame.go:118`
#[derive(Debug, Clone, Default)]
pub struct NativeTransitionFrameInput<'a> {
    pub parallax: Option<&'a NativeMapParallaxSurface>,
    pub parallax_scroll_x: i64,
    pub parallax_scroll_y: i64,
    pub terrain_bank: Option<&'a Bank>,
    pub unit_bank: Option<&'a Bank>,
    pub foreground_bank: Option<&'a Bank>,
    pub selector_cache: Option<&'a NativeSelectorCache>,
    pub cells: Vec<NativeTerrainCell>,
    pub controls: &'a [u8],
    pub terrain_lut: &'a [u8],
    pub map_width: i64,
    pub camera_x: i64,
    pub camera_y: i64,
    pub flip: i64,
    pub terrain_cycle: i64,
    pub idle_cycle: i64,
    pub moving_cycle: i64,
    pub pixel_shift: i64,
    pub units: Vec<NativeUnitLayerEntry>,
    pub foreground_units: Vec<NativeForegroundLayerEntry>,
}

/// NativeUnitPresentStripLayout is 0x22390..0x22434's direct work-buffer to
/// VGA bridge. Offsets are relative to their supplied buffers, not original
/// process addresses.
///
/// Go：`frame.go:135`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeUnitPresentStripLayout {
    pub work_offset: i64,
    pub vga_offset: i64,
    pub rows: i64,
}

/// NativeUnitPresentStripLayoutFor reproduces the two native row ranges.
/// A target on cameraY copies 18 rows beginning at the target row. Every other
/// target copies 24 rows beginning six rows above it.
///
/// Go：`frame.go:144`
pub fn native_unit_present_strip_layout_for(
    map_x: i64,
    map_y: i64,
    camera_x: i64,
    camera_y: i64,
) -> NativeUnitPresentStripLayout {
    let (dx, dy) = (map_x.wrapping_sub(camera_x), map_y.wrapping_sub(camera_y));
    let mut work = WORK_BASE.wrapping_add(dx.wrapping_mul(24)).wrapping_add(dy.wrapping_mul(24 * WORK_STRIDE));
    let mut vga = dx.wrapping_mul(24).wrapping_add(dy.wrapping_mul(24 * VIEW_WIDTH));
    let mut rows = 24;
    if map_y == camera_y {
        rows = 18;
    } else {
        work = work.wrapping_sub(6 * WORK_STRIDE);
        vga = vga.wrapping_sub(6 * VIEW_WIDTH);
    }
    NativeUnitPresentStripLayout { work_offset: work, vga_offset: vga, rows }
}

/// 逐列延遲 callback（Go `func(row int) error`）。Go 的閉包可以同時讀取呼叫端的 VGA 緩衝，觀察「這一列
/// 在延遲之前已經可見」；Rust 的 `vga` 在呼叫期間被可變借用，所以把目前的 VGA 內容唯讀交給 callback
/// （同 `fdother::NativeUnitPresentTick` 把 `dst` 交給 callback 的做法）。
pub type RowDelay<'a> = &'a mut dyn FnMut(i64, &[u8]) -> Result<()>;

/// RunNativeUnitPresentStripBridge reproduces the progressive direct VGA writes
/// after 0x22253's bridge-only 0x22046 pass. Each row copies 24 bytes from the
/// 456-stride work buffer to the 320-stride VGA buffer, then invokes delay10ms.
/// Unlike a full 0x11eb0 viewport present, every completed row is immediately
/// observable. Bounds are preflighted before the first write.
///
/// Go：`frame.go:163`
#[allow(clippy::too_many_arguments)]
pub fn run_native_unit_present_strip_bridge(
    work: &[u8],
    vga: &mut [u8],
    map_x: i64,
    map_y: i64,
    camera_x: i64,
    camera_y: i64,
    delay10ms: Option<RowDelay>,
) -> Result<()> {
    let Some(delay10ms) = delay10ms else {
        return Err(err("indexedmap: incomplete native unit-present strip bridge"));
    };
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE || len64(vga) < VIEW_WIDTH * VIEW_HEIGHT {
        return Err(err("indexedmap: incomplete native unit-present strip bridge"));
    }
    let layout = native_unit_present_strip_layout_for(map_x, map_y, camera_x, camera_y);
    let last_work = layout.work_offset.wrapping_add((layout.rows - 1) * WORK_STRIDE).wrapping_add(24);
    let last_vga = layout.vga_offset.wrapping_add((layout.rows - 1) * VIEW_WIDTH).wrapping_add(24);
    if layout.work_offset < 0 || layout.vga_offset < 0 || last_work > len64(work) || last_vga > len64(vga) {
        return Err(err("indexedmap: native unit-present strip exceeds buffer"));
    }
    for row in 0..layout.rows {
        let src = at(layout.work_offset + row * WORK_STRIDE);
        let dst = at(layout.vga_offset + row * VIEW_WIDTH);
        vga[dst..dst + 24].copy_from_slice(&work[src..src + 24]);
        delay10ms(row, vga).map_err(wrap(format!("indexedmap: unit-present strip row {row} delay: ")))?;
    }
    Ok(())
}

fn terrain_input_ok(in_: &NativeTransitionFrameInput) -> bool {
    in_.terrain_bank.is_some()
        && in_.map_width > 0
        && len64(&in_.cells) % in_.map_width == 0
        && in_.terrain_lut.len() == 256
}

/// ComposeNativeUnitPresentTerrainSnapshot reproduces 0x22253's initial
/// 0x11eee terrain-only draw into its exact 0x25680-byte buffer. Units,
/// foreground, range and HUD are intentionally excluded: 0x22470/0x22046 call
/// 0x127a9 at their own recovered points.
///
/// Go：`frame.go:192`
pub fn compose_native_unit_present_terrain_snapshot(work: &mut [u8], in_: &NativeTransitionFrameInput) -> Result<()> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE {
        return Err(err("indexedmap: native unit-present work must be exactly 0x25680 bytes"));
    }
    if !terrain_input_ok(in_) {
        return Err(err("indexedmap: incomplete native unit-present terrain input"));
    }
    let terrain = in_.terrain_bank.expect("已檢查");
    let mut frame = work.to_vec();
    let (base_x, base_y) = (WORK_BASE % WORK_STRIDE, WORK_BASE / WORK_STRIDE);
    seed_native_map_parallax(
        &mut frame,
        in_.parallax,
        in_.camera_x,
        in_.camera_y,
        in_.parallax_scroll_x,
        in_.parallax_scroll_y,
    )?;
    terrain
        .blit_native_terrain_region(
            &mut frame,
            WORK_STRIDE,
            base_x,
            base_y,
            in_.map_width,
            &in_.cells,
            in_.controls,
            in_.camera_x,
            in_.camera_y,
            13,
            8,
            in_.flip,
            in_.terrain_cycle,
            in_.terrain_lut,
        )
        .map_err(wrap("indexedmap: unit-present terrain: "))?;
    work.copy_from_slice(&frame);
    Ok(())
}

/// ComposeNative24B4DStaging reproduces 0x24B4D's one-time 0x11EEE terrain
/// draw. It differs from the unit-present snapshot only in the proven ninth
/// tile row, which supplies the second row-shifted 312x192 viewport.
///
/// Go：`frame.go:220`
pub fn compose_native24_b4_d_staging(work: &mut [u8], in_: &NativeTransitionFrameInput) -> Result<()> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE {
        return Err(err("indexedmap: native 0x24B4D work must be exactly 0x25680 bytes"));
    }
    if !terrain_input_ok(in_) {
        return Err(err("indexedmap: incomplete native 0x24B4D terrain input"));
    }
    let terrain = in_.terrain_bank.expect("已檢查");
    let mut frame = work.to_vec();
    let (base_x, base_y) = (WORK_BASE % WORK_STRIDE, WORK_BASE / WORK_STRIDE);
    seed_native_map_parallax(
        &mut frame,
        in_.parallax,
        in_.camera_x,
        in_.camera_y,
        in_.parallax_scroll_x,
        in_.parallax_scroll_y,
    )?;
    terrain
        .blit_native_terrain_region(
            &mut frame,
            WORK_STRIDE,
            base_x,
            base_y,
            in_.map_width,
            &in_.cells,
            in_.controls,
            in_.camera_x,
            in_.camera_y,
            13,
            9,
            in_.flip,
            in_.terrain_cycle,
            in_.terrain_lut,
        )
        .map_err(wrap("indexedmap: native 0x24B4D terrain: "))?;
    work.copy_from_slice(&frame);
    Ok(())
}

/// RedrawNativeUnitPresentObjects is the isolated 0x127a9-equivalent callback
/// required after the LMI intro writes and between 0x22046's radial passes.
/// It commits atomically and never redraws terrain/range/HUD.
///
/// Go：`frame.go:248`
pub fn redraw_native_unit_present_objects(work: &mut [u8], in_: &NativeTransitionFrameInput) -> Result<()> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE {
        return Err(err("indexedmap: native unit-present work must be exactly 0x25680 bytes"));
    }
    let (Some(unit_bank), Some(foreground_bank), Some(cache)) =
        (in_.unit_bank, in_.foreground_bank, in_.selector_cache)
    else {
        return Err(err("indexedmap: incomplete native unit-present object input"));
    };
    if in_.map_width <= 0 || len64(&in_.cells) % in_.map_width != 0 || in_.terrain_lut.len() != 256 {
        return Err(err("indexedmap: incomplete native unit-present object input"));
    }
    let mut frame = work.to_vec();
    unit_bank
        .blit_native_unit_layer(
            &mut frame,
            WORK_STRIDE,
            cache,
            &in_.units,
            in_.camera_x,
            in_.camera_y,
            12,
            7,
            in_.idle_cycle,
            in_.moving_cycle,
            in_.pixel_shift,
        )
        .map_err(wrap("indexedmap: unit-present units: "))?;
    foreground_bank
        .blit_native_foreground_layer(
            &mut frame,
            WORK_STRIDE,
            &in_.foreground_units,
            in_.map_width,
            &in_.cells,
            in_.controls,
            in_.camera_x,
            in_.camera_y,
            12,
            7,
            in_.flip,
            in_.terrain_lut,
        )
        .map_err(wrap("indexedmap: unit-present foreground: "))?;
    work.copy_from_slice(&frame);
    Ok(())
}

/// CopyNativeUnitPresentViewport reproduces the 0x11eb0 calls in
/// 0x22470/0x22547/0x22656: source work+0x8088, source stride456,
/// width312, height192, destination stride320.
///
/// Go：`frame.go:279`
pub fn copy_native_unit_present_viewport(vga: &mut [u8], work: &[u8]) -> Result<()> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE || len64(vga) < VIEW_WIDTH * VIEW_HEIGHT {
        return Err(err("indexedmap: incomplete native unit-present viewport buffers"));
    }
    fdother::copy_native_transition_viewport(vga, VIEW_WIDTH, work, WORK_BASE, WORK_STRIDE, 312, VIEW_HEIGHT)
        .map_err(Error::from)
}

/// ComposeNativeUnitPresentIntroFrame executes one 0x22470 frame atomically:
/// restore terrain snapshot, blit one FDOTHER#6 LMI entry at the recovered map
/// coordinate, redraw objects, then copy the 312×192 viewport.
///
/// Go：`frame.go:292`
pub fn compose_native_unit_present_intro_frame(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    in_: &NativeTransitionFrameInput,
    entry: &LMI1Entry,
    map_x: i64,
    map_y: i64,
) -> Result<()> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(snapshot) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(vga) < VIEW_WIDTH * VIEW_HEIGHT
    {
        return Err(err("indexedmap: incomplete native unit-present intro buffers"));
    }
    let (mut frame, mut viewport) = (snapshot.to_vec(), vga.to_vec());
    fdother::blit_native_unit_present_lmi(entry, &mut frame, map_x, map_y, in_.camera_x, in_.camera_y)
        .map_err(wrap("indexedmap: unit-present intro LMI: "))?;
    redraw_native_unit_present_objects(&mut frame, in_)?;
    copy_native_unit_present_viewport(&mut viewport, &frame)?;
    work.copy_from_slice(&frame);
    vga.copy_from_slice(&viewport);
    Ok(())
}

/// ComposeNativeUnitPresentLUTSnapshot reproduces the phase boundary at the
/// start of 0x22547. Native keeps the original 0x22253 allocation, restores its
/// terrain-only contents, and blits the final intro LMI entry #0x7c into that
/// allocation once. The resulting terrain+LMI snapshot is then shared by all
/// six contract frames and all ten release frames; object layers are redrawn
/// only after each frame restores this snapshot.
///
/// Go：`frame.go:326`
pub fn compose_native_unit_present_lut_snapshot(
    dst: &mut [u8],
    terrain_snapshot: &[u8],
    in_: &NativeTransitionFrameInput,
    final_intro: &LMI1Entry,
    map_x: i64,
    map_y: i64,
) -> Result<()> {
    if len64(dst) != NATIVE_UNIT_PRESENT_WORK_SIZE || len64(terrain_snapshot) != NATIVE_UNIT_PRESENT_WORK_SIZE {
        return Err(err("indexedmap: incomplete native unit-present LUT snapshot buffers"));
    }
    let mut frame = terrain_snapshot.to_vec();
    fdother::blit_native_unit_present_lmi(final_intro, &mut frame, map_x, map_y, in_.camera_x, in_.camera_y)
        .map_err(wrap("indexedmap: unit-present LUT snapshot LMI: "))?;
    dst.copy_from_slice(&frame);
    Ok(())
}

/// ComposeNativeUnitPresentLUTFrame binds fdother's exact snapshot/remap
/// transaction to indexedmap's native object redraw and viewport copy.
/// Contract and release both consume the one snapshot built at 0x22547.
///
/// Go：`frame.go:348`
pub fn compose_native_unit_present_lut_frame(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    lut: &[u8],
    in_: &NativeTransitionFrameInput,
    frame: &UnitPresentLUTFrame,
) -> Result<()> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(snapshot) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(vga) < VIEW_WIDTH * VIEW_HEIGHT
    {
        return Err(err("indexedmap: incomplete native unit-present LUT buffers"));
    }
    let (mut next_work, mut next_vga) = (work.to_vec(), vga.to_vec());
    let mut redraw = |full: &mut [u8]| redraw_native_unit_present_objects(full, in_).map_err(to_fdother);
    let mut present = |full: &mut [u8], _: &UnitPresentLUTFrame| {
        copy_native_unit_present_viewport(&mut next_vga, full).map_err(to_fdother)
    };
    fdother::run_native_unit_present_lut_frame(
        &mut next_work,
        snapshot,
        lut,
        frame,
        Some(&mut redraw),
        Some(&mut present),
    )
    .map_err(wrap("indexedmap: unit-present LUT frame: "))?;
    work.copy_from_slice(&next_work);
    vga.copy_from_slice(&next_vga);
    Ok(())
}

/// ComposeNativeUnitPresentStripBridge binds 0x22253's transition between the
/// six contract and ten release frames. Native restores the shared
/// terrain+final-LMI snapshot, applies one extra 0x22046 pass with the
/// pointer+1 LUT returned by 0x22547, redraws objects between radial passes,
/// then progressively copies the target strip directly to VGA. There is no
/// full-viewport 0x11eb0 present in this transaction.
///
/// Go：`frame.go:379`。Go 把 `frame[workBase:]` 交給 `fdother.ApplyIndexedTransitionPass`，中段重繪的
/// 閉包改寫整個 `frame`（兩個切片共用底層陣列）。`fdother` 只公開收單一切片的版本，所以這裡把
/// `0x8088` 之前的位元組另存，重繪時接成完整緩衝，重繪後再拆回兩段，結果與共用陣列相同。
#[allow(clippy::too_many_arguments)]
pub fn compose_native_unit_present_strip_bridge(
    work: &mut [u8],
    vga: &mut [u8],
    snapshot: &[u8],
    bridge_lut: &[u8],
    in_: &NativeTransitionFrameInput,
    map_x: i64,
    map_y: i64,
    raw_53ab9: i64,
    raw_53abd: i64,
    delay10ms: Option<RowDelay>,
) -> Result<()> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(snapshot) != NATIVE_UNIT_PRESENT_WORK_SIZE
        || len64(vga) < VIEW_WIDTH * VIEW_HEIGHT
        || bridge_lut.len() != 256
        || delay10ms.is_none()
    {
        return Err(err("indexedmap: incomplete native unit-present strip composition"));
    }
    let pass = fdother::native_unit_present_lut_pass(
        24i64.wrapping_mul(raw_53ab9).wrapping_add(12),
        24i64.wrapping_mul(raw_53abd).wrapping_add(15),
        0,
    )
    .map_err(wrap("indexedmap: unit-present strip geometry: "))?;
    let base = at(WORK_BASE);
    let mut prefix = snapshot[..base].to_vec();
    let mut tail = snapshot[base..].to_vec();
    let mut redraw = |dst: &mut [u8]| -> fdother::Result<()> {
        let mut full = Vec::with_capacity(prefix.len() + dst.len());
        full.extend_from_slice(&prefix);
        full.extend_from_slice(dst);
        redraw_native_unit_present_objects(&mut full, in_).map_err(to_fdother)?;
        prefix.copy_from_slice(&full[..base]);
        dst.copy_from_slice(&full[base..]);
        Ok(())
    };
    fdother::apply_indexed_transition_pass(&mut tail, WORK_STRIDE, bridge_lut, pass, Some(&mut redraw))
        .map_err(wrap("indexedmap: unit-present strip LUT: "))?;
    // Match native observability: work has completed the remap before the first
    // progressive VGA row becomes visible.
    work[..base].copy_from_slice(&prefix);
    work[base..].copy_from_slice(&tail);
    run_native_unit_present_strip_bridge(work, vga, map_x, map_y, in_.camera_x, in_.camera_y, delay10ms)
}

/// BuildNativeTerrainCells materializes the exact per-cell pair exported by
/// tools/export_engine_assets.py. The tile word and event high byte are kept
/// separate because native 0x11eee consumes both; missing or mismatched arrays
/// are rejected instead of defaulting old PNG-only maps into a native path.
///
/// Go：`frame.go:415`
pub fn build_native_terrain_cells(tiles: &[i64], blit_modes: &[u8]) -> Result<Vec<NativeTerrainCell>> {
    if tiles.is_empty() || tiles.len() != blit_modes.len() {
        return Err(err("indexedmap: native terrain cell arrays are incomplete"));
    }
    let mut cells = vec![NativeTerrainCell::default(); tiles.len()];
    for (i, &tile) in tiles.iter().enumerate() {
        if !(0..=0x3ff).contains(&tile) {
            return Err(err(format!("indexedmap: native terrain tile {i}={tile} outside 10-bit range")));
        }
        cells[i] = NativeTerrainCell { tile: tile as u16, blit_mode: blit_modes[i] };
    }
    Ok(cells)
}

/// ComposeNativeFrame is the strict native-HUD form of ComposeFrame. It uses
/// the exact indexed HUD assembly at its recovered position, rather than
/// accepting an arbitrary callback. All source data remain explicit and any
/// rejection keeps work/VGA unchanged through ComposeFrame's transaction.
///
/// Go：`frame.go:433`
pub fn compose_native_frame(work: &mut [u8], vga: &mut [u8], in_: &mut NativeFrameInput) -> Result<()> {
    compose_native_frame_observed(work, vga, in_, None)
}

/// ComposeNativeFrameObserved preserves ComposeNativeFrame's strict HUD owner
/// while exposing detached stage snapshots to bounded diagnostics.
///
/// Go：`frame.go:439`
pub fn compose_native_frame_observed(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &mut NativeFrameInput,
    observer: Option<FrameObserver>,
) -> Result<()> {
    let NativeFrameInput { frame, hud, frames, hud_terrain, hud_units, hud_cache } = in_;
    let mut render = |dst: &mut [u8]| -> Result<()> {
        // 0x11cfa passes [0x53a49]+0x8088 to 0x1acf3, not the allocation
        // base. HUD row/column offsets are therefore viewport-relative, just
        // like the preceding 0x11eee terrain destination.
        if len64(dst) < WORK_BASE {
            return Err(err("indexedmap: native HUD viewport base outside work buffer"));
        }
        blit_native_map_hud(frames, *hud_terrain, *hud_units, *hud_cache, &mut dst[at(WORK_BASE)..], hud)
    };
    compose_frame_inner(work, vga, frame, Some(&mut render), observer)
}

/// ComposeNativeStepFrame is the redraw the original runs while a unit is
/// walking. It is a different path from 0x11cac, not 0x11cac with layers
/// switched off.
///
/// dosgolem 逐幀收據（docs/data/ui-traces/fd2-move-frame-compare-20260909.json
/// 與同日的 eip-watch 量測）：確認移動之後到抵達為止，每一幀
/// `0x11EEE`（地形）與 `0x127A9`（前景）各進入一次、`0x127E0`（單位繪製公式）
/// 進入十一次，而 `0x11CAC`（整幀排程）、`0x122DC`（範圍／游標圖示）與
/// `0x1AD72`（HUD）**一次都沒有**。畫面上因此沒有游標白框、也沒有左下 HUD
/// 面板，而 overlay selector `[0x51A83]` 整段仍是 1——這個差異只有看畫面才
/// 看得出來。
///
/// 視窗複製沒有單獨量到位址，但走行期間畫面確實在更新，所以保留與整幀排程
/// 相同的 312×192 複製。章節輔助表面同樣沒有單獨量到，保留原本行為。
///
/// Go：`frame.go:465`
pub fn compose_native_step_frame(work: &mut [u8], vga: &mut [u8], in_: &FrameInput) -> Result<()> {
    compose_native_step_frame_observed(work, vga, in_, None)
}

/// Go：`frame.go:469`
pub fn compose_native_step_frame_observed(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &FrameInput,
    observer: Option<FrameObserver>,
) -> Result<()> {
    compose_native_step_frame_observed_inner(work, vga, in_, WORK_BASE, in_.camera_y, 8, WORK_BASE, observer)
}

/// ComposeNativeUpStepFrame preserves 0x13185's six terrain/copy passes.
/// The terrain camera and destination differ from the unit/foreground camera.
/// Source: fd2_terrain_mode3_review_20261001.json, walk_terrain_writer.
///
/// Go：`frame.go:476`
pub fn compose_native_up_step_frame(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &FrameInput,
    unit_y: i64,
    motion: i64,
) -> Result<()> {
    compose_native_up_step_frame_observed(work, vga, in_, unit_y, motion, None)
}

/// NativeUpStepLayout is shared by the native copy and HD object projection.
/// Values retain the existing 0x13185 caller's work-address space.
///
/// Go：`frame.go:482`，回傳 `(terrainBase, terrainCameraY, terrainRows, viewportBase int, err error)`。
pub fn native_up_step_layout(in_: &FrameInput, unit_y: i64, motion: i64) -> Result<(i64, i64, i64, i64)> {
    if !(1..=6).contains(&motion)
        || in_.camera_y < 0
        || in_.map_width <= 0
        || unit_y < 0
        || unit_y >= len64(&in_.cells) / in_.map_width
        || in_.parallax.is_some()
        || in_.chapter_aux.is_some()
    {
        return Err(err("indexedmap: invalid native up-step input"));
    }
    let (mut terrain_base, mut terrain_camera_y, mut viewport_base) = (WORK_BASE, in_.camera_y, WORK_BASE);
    if in_.camera_y > 0 {
        (terrain_base, terrain_camera_y) = (0x55C8, in_.camera_y - 1);
        if unit_y.wrapping_sub(in_.camera_y) < 2 {
            viewport_base -= motion * NATIVE_MOTION_PIXELS * WORK_STRIDE;
        }
    }
    Ok((terrain_base, terrain_camera_y, 9, viewport_base))
}

/// Go：`frame.go:498`
pub fn compose_native_up_step_frame_observed(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &FrameInput,
    unit_y: i64,
    motion: i64,
    observer: Option<FrameObserver>,
) -> Result<()> {
    if len64(work) != NATIVE_UNIT_PRESENT_WORK_SIZE {
        return Err(err("indexedmap: invalid native up-step input"));
    }
    let (terrain_base, terrain_camera_y, rows, viewport_base) = native_up_step_layout(in_, unit_y, motion)?;
    compose_native_step_frame_observed_inner(
        work,
        vga,
        in_,
        terrain_base,
        terrain_camera_y,
        rows,
        viewport_base,
        observer,
    )
}

/// Go：`frame.go:509`。Go 版也沒有呼叫端，照樣保留。
#[allow(dead_code)]
#[allow(clippy::too_many_arguments)]
fn compose_native_step_frame_inner(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &FrameInput,
    terrain_base: i64,
    terrain_camera_y: i64,
    terrain_rows: i64,
    viewport_base: i64,
) -> Result<()> {
    compose_native_step_frame_observed_inner(
        work,
        vga,
        in_,
        terrain_base,
        terrain_camera_y,
        terrain_rows,
        viewport_base,
        None,
    )
}

/// Go：`frame.go:513`
#[allow(clippy::too_many_arguments)]
fn compose_native_step_frame_observed_inner(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &FrameInput,
    terrain_base: i64,
    terrain_camera_y: i64,
    terrain_rows: i64,
    viewport_base: i64,
    mut observer: Option<FrameObserver>,
) -> Result<()> {
    if len64(work) % WORK_STRIDE != 0
        || len64(vga) < NATIVE_MAP_VGA_SIZE
        || in_.map_width <= 0
        || len64(&in_.cells) % in_.map_width != 0
    {
        return Err(err("indexedmap: incomplete native step frame input"));
    }
    let (Some(terrain), Some(unit_bank), Some(foreground_bank), Some(cache)) =
        (in_.terrain_bank, in_.unit_bank, in_.foreground_bank, in_.selector_cache)
    else {
        return Err(err("indexedmap: missing native step frame bank"));
    };
    let mut frame = work.to_vec();
    let (base_x, base_y) = (terrain_base % WORK_STRIDE, terrain_base / WORK_STRIDE);
    if let Some(aux) = in_.chapter_aux {
        fdother::blit_native_chapter_aux_viewport(&mut frame[at(WORK_BASE)..], WORK_STRIDE, aux, in_.chapter_aux_phase)
            .map_err(wrap("indexedmap: step chapter auxiliary surface: "))?;
    }
    seed_native_map_parallax(
        &mut frame,
        in_.parallax,
        in_.camera_x,
        in_.camera_y,
        in_.parallax_scroll_x,
        in_.parallax_scroll_y,
    )?;
    terrain
        .blit_native_terrain_region(
            &mut frame,
            WORK_STRIDE,
            base_x,
            base_y,
            in_.map_width,
            &in_.cells,
            in_.controls,
            in_.camera_x,
            terrain_camera_y,
            13,
            terrain_rows,
            in_.flip,
            in_.terrain_cycle,
            in_.lut,
        )
        .map_err(wrap("indexedmap: step terrain: "))?;
    observe_frame_stage(&mut observer, FRAME_STAGE_TERRAIN, &frame, vga)?;
    unit_bank
        .blit_native_unit_layer(
            &mut frame,
            WORK_STRIDE,
            cache,
            &in_.units,
            in_.camera_x,
            in_.camera_y,
            12,
            7,
            in_.idle_cycle,
            in_.moving_cycle,
            in_.pixel_shift,
        )
        .map_err(wrap("indexedmap: step units: "))?;
    foreground_bank
        .blit_native_foreground_layer(
            &mut frame,
            WORK_STRIDE,
            &in_.foreground_units,
            in_.map_width,
            &in_.cells,
            in_.controls,
            in_.camera_x,
            in_.camera_y,
            12,
            7,
            in_.flip,
            in_.lut,
        )
        .map_err(wrap("indexedmap: step foreground: "))?;
    let mut copy_frame = vga.to_vec();
    fdicon::copy_native_indexed_region(
        &mut copy_frame[at(STEADY_VIEWPORT_OFFSET)..],
        VIEW_WIDTH,
        &frame[at(viewport_base)..],
        WORK_STRIDE,
        STEADY_VIEWPORT_WIDTH,
        VIEW_HEIGHT,
    )
    .map_err(wrap("indexedmap: step viewport copy: "))?;
    work.copy_from_slice(&frame);
    vga.copy_from_slice(&copy_frame);
    Ok(())
}

/// ComposeNativeTransitionFrame performs one verified indexed 0x24618 frame:
/// terrain redraw → first LUT pass → 0x127a9 unit/foreground redraw → second
/// LUT pass → centered rectangle LUT → 312×192 viewport copy. It clones the
/// work buffer before every operation, so missing raw banks or malformed
/// coordinates cannot partially mutate either caller buffer.
///
/// Go：`frame.go:561`
pub fn compose_native_transition_frame(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &NativeTransitionFrameInput,
    pass: IndexedTransitionPass,
    lut: &[u8],
) -> Result<()> {
    if len64(work) < NATIVE_TRANSITION_STAGE_OFFSET + NATIVE_TRANSITION_STAGE_HEIGHT * NATIVE_TRANSITION_STAGE_STRIDE
        || len64(vga) < NATIVE_TRANSITION_PRESENT_STRIDE * NATIVE_TRANSITION_STAGE_HEIGHT
    {
        return Err(err("indexedmap: incomplete native transition buffers"));
    }
    let (Some(terrain), Some(unit_bank), Some(foreground_bank), Some(cache)) =
        (in_.terrain_bank, in_.unit_bank, in_.foreground_bank, in_.selector_cache)
    else {
        return Err(err("indexedmap: incomplete native transition input"));
    };
    if in_.map_width <= 0 || len64(&in_.cells) % in_.map_width != 0 || in_.terrain_lut.len() != 256 || lut.len() != 256
    {
        return Err(err("indexedmap: incomplete native transition input"));
    }
    let mut frame = work.to_vec();
    let (base_x, base_y) = (WORK_BASE % WORK_STRIDE, WORK_BASE / WORK_STRIDE);
    seed_native_map_parallax(
        &mut frame,
        in_.parallax,
        in_.camera_x,
        in_.camera_y,
        in_.parallax_scroll_x,
        in_.parallax_scroll_y,
    )?;
    terrain
        .blit_native_terrain_region(
            &mut frame,
            WORK_STRIDE,
            base_x,
            base_y,
            in_.map_width,
            &in_.cells,
            in_.controls,
            in_.camera_x,
            in_.camera_y,
            13,
            8,
            in_.flip,
            in_.terrain_cycle,
            in_.terrain_lut,
        )
        .map_err(wrap("indexedmap: transition terrain: "))?;
    let mut redraw = |dst: &mut [u8]| -> fdother::Result<()> {
        transition_redraw(dst, in_, unit_bank, foreground_bank, cache).map_err(to_fdother)
    };
    fdother::apply_indexed_transition_pass(&mut frame, WORK_STRIDE, lut, pass, Some(&mut redraw))
        .map_err(wrap("indexedmap: transition pass: "))?;
    let mut viewport = vec![0u8; at(NATIVE_TRANSITION_PRESENT_STRIDE * NATIVE_TRANSITION_STAGE_HEIGHT)];
    fdother::copy_native_transition_viewport(
        &mut viewport,
        NATIVE_TRANSITION_PRESENT_STRIDE,
        &frame,
        NATIVE_TRANSITION_STAGE_OFFSET,
        NATIVE_TRANSITION_STAGE_STRIDE,
        NATIVE_TRANSITION_STAGE_WIDTH,
        NATIVE_TRANSITION_STAGE_HEIGHT,
    )
    .map_err(wrap("indexedmap: transition viewport: "))?;
    work.copy_from_slice(&frame);
    vga[..viewport.len()].copy_from_slice(&viewport);
    Ok(())
}

/// `ComposeNativeTransitionFrame` 的中段重繪閉包（Go `frame.go:576`）。
fn transition_redraw(
    dst: &mut [u8],
    in_: &NativeTransitionFrameInput,
    unit_bank: &Bank,
    foreground_bank: &Bank,
    cache: &NativeSelectorCache,
) -> Result<()> {
    unit_bank
        .blit_native_unit_layer(
            dst,
            WORK_STRIDE,
            cache,
            &in_.units,
            in_.camera_x,
            in_.camera_y,
            12,
            7,
            in_.idle_cycle,
            in_.moving_cycle,
            in_.pixel_shift,
        )
        .map_err(wrap("indexedmap: transition units: "))?;
    foreground_bank
        .blit_native_foreground_layer(
            dst,
            WORK_STRIDE,
            &in_.foreground_units,
            in_.map_width,
            &in_.cells,
            in_.controls,
            in_.camera_x,
            in_.camera_y,
            12,
            7,
            in_.flip,
            in_.terrain_lut,
        )
        .map_err(wrap("indexedmap: transition foreground: "))
}

/// ComposeFrame performs the recovered steady order:
///
///    0x11eee optional chapter surface + terrain → 0x122dc range → 0x127a9 unit/foreground
///    → required 0x1acf3-equivalent HUD callback → 0x11eb0 viewport copy.
///
/// HUD is mandatory because copying before it would silently alter native draw
/// order. All work happens on a private clone first, so rejected editable input
/// or a HUD error never leaves either caller buffer partially changed.
///
/// Go：`frame.go:605`。所有權：成功時把 mode 6 改寫的格子寫回 `in_.cells`（Go 寫回呼叫端共用的陣列）。
pub fn compose_frame(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &mut FrameInput,
    render_hud: Option<RenderHUD>,
) -> Result<()> {
    compose_frame_inner(work, vga, in_, render_hud, None)
}

/// ComposeFrameObserved is the callback-HUD form used by package-level
/// diagnostics. Observer failure follows the same atomic rejection contract as
/// any compositor pass.
///
/// Go：`frame.go:612`
pub fn compose_frame_observed(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &mut FrameInput,
    render_hud: Option<RenderHUD>,
    observer: Option<FrameObserver>,
) -> Result<()> {
    compose_frame_inner(work, vga, in_, render_hud, observer)
}

/// Go：`frame.go:616` 的 `composeFrame`。
fn compose_frame_inner(
    work: &mut [u8],
    vga: &mut [u8],
    in_: &mut FrameInput,
    render_hud: Option<RenderHUD>,
    mut observer: Option<FrameObserver>,
) -> Result<()> {
    let Some(render_hud) = render_hud else {
        return Err(err("indexedmap: incomplete native frame input"));
    };
    if len64(work) % WORK_STRIDE != 0
        || len64(vga) < NATIVE_MAP_VGA_SIZE
        || in_.map_width <= 0
        || len64(&in_.cells) % in_.map_width != 0
    {
        return Err(err("indexedmap: incomplete native frame input"));
    }
    let (Some(terrain), Some(range_bank), Some(unit_bank), Some(foreground_bank), Some(cache)) =
        (in_.terrain_bank, in_.range_bank, in_.unit_bank, in_.foreground_bank, in_.selector_cache)
    else {
        return Err(err("indexedmap: missing native frame bank"));
    };
    let mut frame = work.to_vec();
    let mut cells = in_.cells.clone();
    let (base_x, base_y) = (WORK_BASE % WORK_STRIDE, WORK_BASE / WORK_STRIDE);
    if let Some(aux) = in_.chapter_aux {
        fdother::blit_native_chapter_aux_viewport(&mut frame[at(WORK_BASE)..], WORK_STRIDE, aux, in_.chapter_aux_phase)
            .map_err(wrap("indexedmap: chapter auxiliary surface: "))?;
    }
    seed_native_map_parallax(
        &mut frame,
        in_.parallax,
        in_.camera_x,
        in_.camera_y,
        in_.parallax_scroll_x,
        in_.parallax_scroll_y,
    )?;
    terrain
        .blit_native_terrain_region(
            &mut frame,
            WORK_STRIDE,
            base_x,
            base_y,
            in_.map_width,
            &cells,
            in_.controls,
            in_.camera_x,
            in_.camera_y,
            13,
            8,
            in_.flip,
            in_.terrain_cycle,
            in_.lut,
        )
        .map_err(wrap("indexedmap: terrain: "))?;
    observe_frame_stage(&mut observer, FRAME_STAGE_TERRAIN, &frame, vga)?;
    if in_.range_mode == 6 {
        if in_.cursor_x < 0
            || in_.cursor_y < 0
            || in_.cursor_x >= in_.map_width
            || in_.cursor_y >= len64(&cells) / in_.map_width
        {
            return Err(err("indexedmap: mode6 cursor outside field"));
        }
        // 0x122dc mutates composition byte+3 after 0x11eee terrain and before
        // 0x127a9 foreground. Commit to caller state only after the full frame
        // succeeds, retaining the adapter's failure-atomic boundary.
        cells[at(in_.cursor_y * in_.map_width + in_.cursor_x)].blit_mode = 0;
    } else {
        fdother::blit_native_range_overlay(
            range_bank,
            &mut frame,
            in_.camera_x,
            in_.camera_y,
            13,
            8,
            in_.range_mode,
            in_.cursor_x,
            in_.cursor_y,
        )
        .map_err(wrap("indexedmap: range: "))?;
    }
    observe_frame_stage(&mut observer, FRAME_STAGE_RANGE, &frame, vga)?;
    unit_bank
        .blit_native_unit_layer(
            &mut frame,
            WORK_STRIDE,
            cache,
            &in_.units,
            in_.camera_x,
            in_.camera_y,
            12,
            7,
            in_.idle_cycle,
            in_.moving_cycle,
            in_.pixel_shift,
        )
        .map_err(wrap("indexedmap: units: "))?;
    observe_frame_stage(&mut observer, FRAME_STAGE_UNITS, &frame, vga)?;
    foreground_bank
        .blit_native_foreground_layer(
            &mut frame,
            WORK_STRIDE,
            &in_.foreground_units,
            in_.map_width,
            &cells,
            in_.controls,
            in_.camera_x,
            in_.camera_y,
            12,
            7,
            in_.flip,
            in_.lut,
        )
        .map_err(wrap("indexedmap: foreground: "))?;
    observe_frame_stage(&mut observer, FRAME_STAGE_FOREGROUND, &frame, vga)?;
    if let Some(overlay) = in_.selection_overlay {
        overlay(&mut frame, WORK_STRIDE, WORK_BASE).map_err(wrap("indexedmap: selection overlay: "))?;
    }
    render_hud(&mut frame).map_err(wrap("indexedmap: HUD: "))?;
    observe_frame_stage(&mut observer, FRAME_STAGE_HUD, &frame, vga)?;
    let mut copy_frame = vga.to_vec();
    fdicon::copy_native_indexed_region(
        &mut copy_frame[at(STEADY_VIEWPORT_OFFSET)..],
        VIEW_WIDTH,
        &frame[at(WORK_BASE)..],
        WORK_STRIDE,
        STEADY_VIEWPORT_WIDTH,
        VIEW_HEIGHT,
    )
    .map_err(wrap("indexedmap: viewport copy: "))?;
    observe_frame_stage(&mut observer, FRAME_STAGE_VIEWPORT, &frame, &copy_frame)?;
    work.copy_from_slice(&frame);
    vga.copy_from_slice(&copy_frame);
    in_.cells.copy_from_slice(&cells);
    Ok(())
}

/// Go：`frame.go:695`
fn observe_frame_stage(observer: &mut Option<FrameObserver>, stage: FrameStage, work: &[u8], vga: &[u8]) -> Result<()> {
    let Some(observer) = observer else {
        return Ok(());
    };
    observer(stage, &mut work.to_vec(), &mut vga.to_vec())
        .map_err(wrap(format!("indexedmap: frame observer stage {}: ", stage.0)))
}

/// 所有地形consumer在切片前先檢查viewport base，缺緩衝不得panic。
///
/// Go：`frame.go:710`
fn seed_native_map_parallax(
    frame: &mut [u8],
    surface: Option<&NativeMapParallaxSurface>,
    camera_x: i64,
    camera_y: i64,
    scroll_x: i64,
    scroll_y: i64,
) -> Result<()> {
    let Some(surface) = surface else {
        return Ok(());
    };
    if len64(frame) < WORK_BASE {
        return Err(err("indexedmap: map parallax work buffer is too small"));
    }
    fdother::blit_native_map_parallax_viewport(
        &mut frame[at(WORK_BASE)..],
        WORK_STRIDE,
        surface,
        camera_x,
        camera_y,
        scroll_x,
        scroll_y,
    )
    .map_err(wrap("indexedmap: map parallax: "))
}

#[cfg(test)]
mod frame_test;
#[cfg(test)]
mod hd_step_observer_test;
