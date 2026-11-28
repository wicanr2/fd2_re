//! Go 來源：`remake/cmd/fd2/native_ch20_sky_key.go`（390 行）。
//!
//! Go 檔的函式都已移植；HD 的天空之鑰世界凍結、預檢與 HD 幀在 R6（PORTING.md「D 組：HD 預凍結延後」）。

use super::*;

/// Go：`native_ch20_sky_key.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCh20SkyKeyPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_CH20_SKY_KEY_PAN: NativeCh20SkyKeyPhase = NativeCh20SkyKeyPhase(0);
pub(crate) const NATIVE_CH20_SKY_KEY_FIRST_FRAMES: NativeCh20SkyKeyPhase = NativeCh20SkyKeyPhase(1);
pub(crate) const NATIVE_CH20_SKY_KEY_ANI: NativeCh20SkyKeyPhase = NativeCh20SkyKeyPhase(2);
pub(crate) const NATIVE_CH20_SKY_KEY_FLASH: NativeCh20SkyKeyPhase = NativeCh20SkyKeyPhase(3);
pub(crate) const NATIVE_CH20_SKY_KEY_RESTORE: NativeCh20SkyKeyPhase = NativeCh20SkyKeyPhase(4);
pub(crate) const NATIVE_CH20_SKY_KEY_TAIL_FRAMES: NativeCh20SkyKeyPhase = NativeCh20SkyKeyPhase(5);

/// nativeCh20SkyKeyJob owns the complete blocking presentation called at
/// 0x242c9. Indexed framebuffer and DAC snapshots remain separate throughout:
/// the white flash and baseline restore must recolour the final ANI indices,
/// not replace them with a generic RGBA overlay.
///
/// Go：`native_ch20_sky_key.go:32`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCh20SkyKeyJob {
    pub(crate) hd_world: Option<ModernSkyKeyWorld>,
    pub(crate) hd_frame: Option<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) spec: campaign::NativeCh20SkyKeySequence,
    pub(crate) fd_frames: Vec<fdother::Frame>,
    pub(crate) ani_pixels: Vec<Vec<u8>>,
    pub(crate) ani_dac: Vec<Vec<u8>>,
    pub(crate) base: Vec<u8>,
    pub(crate) vga: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) phase: NativeCh20SkyKeyPhase,
    pub(crate) frame: i64,
    pub(crate) ticks: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    /// Go 是 `func()` 閉包；快照含不能跨執行緒的 `NativeFrameInput`，`Box<dyn FnOnce + Send>` 裝不下，
    /// Rust 改存快照本身，由 [`NativeCh20SkyKeySnapshot::restore`] 還原。
    pub(crate) rollback: Option<Box<NativeCh20SkyKeySnapshot>>,
}

/// Go：`native_ch20_sky_key.go:55`，`func validateNativeCh20SkyKeyAssets(spec campaign.NativeCh20SkyKeySequence, frames []fdother.Frame, clip *afm.Clip) error`
/// Go 的 `clip` 可為 nil；Rust 收 `Option`。
pub(crate) fn validate_native_ch20_sky_key_assets(
    spec: &campaign::NativeCh20SkyKeySequence,
    frames: &[fdother::Frame],
    clip: Option<&afm::Clip>,
) -> Result<()> {
    if !spec.is_recovered_contract() {
        return Err(game_err!("native 0x24336 payload differs from recovered contract"));
    }
    if frames.len() as i64 != spec.fdother_frame_count {
        return Err(game_err!(
            "FDOTHER #{} frames={}, want {}",
            spec.fdother_resource,
            frames.len(),
            spec.fdother_frame_count
        ));
    }
    // 鏡頭或戰役狀態改變之前先驗每一條 RLE；實際合成要等原版捲到 (14,8) 才做。
    for (index, frame) in frames.iter().enumerate() {
        if let Err(err) = frame.blit(&mut vec![0u8; indexedmap::NATIVE_MAP_VGA_SIZE as usize], 320, -1) {
            return Err(game_err!("FDOTHER #{} frame {index}: {err}", spec.fdother_resource));
        }
    }
    let count = spec.ani_frame_count;
    let Some(clip) =
        clip.filter(|clip| clip.indexed_frames.len() as i64 == count && clip.palettes.len() as i64 == count)
    else {
        return Err(game_err!("ANI #{} frame snapshots are incomplete", spec.ani_resource));
    };
    for index in 0..count as usize {
        if clip.indexed_frames[index].len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || clip.palettes[index].len() != 256 * 3
        {
            return Err(game_err!("ANI #{} frame {index} geometry is incomplete", spec.ani_resource));
        }
        if let Err(err) = fdother::vga_palette_from_dac(&clip.palettes[index]) {
            return Err(game_err!("ANI #{} frame {index} palette: {err}", spec.ani_resource));
        }
    }
    Ok(())
}

/// Go：`native_ch20_sky_key.go:84`，`func loadNativeCh20SkyKeyANI(spec campaign.NativeCh20SkyKeySequence) (*afm.Clip, error)`
pub(crate) fn load_native_ch20_sky_key_ani(spec: &campaign::NativeCh20SkyKeySequence) -> Result<afm::Clip> {
    Ok(afm::load_separated_resource(separated_asset_path("animations"), spec.ani_resource)?)
}

/// snapshotNativeCh20SkyKeyState protects the caller-visible map transaction.
/// Asset decoding is already fully preflighted, but a later indexed HUD/frame
/// rejection must still restore the view, timing and framebuffer atomically.
///
/// Go：`native_ch20_sky_key.go:161`，`func snapshotNativeCh20SkyKeyState(g *Game) func()`
/// Go 回還原閉包；Rust 回快照，呼叫端以 `restore` 還原。
pub(crate) fn snapshot_native_ch20_sky_key_state(g: &Game) -> Box<NativeCh20SkyKeySnapshot> {
    Box::new(NativeCh20SkyKeySnapshot {
        state: g.st.clone(),
        cursor_camera: (g.cur_x, g.cur_y, g.cam_x, g.cam_y),
        work: g.native_map_work.clone(),
        vga: g.native_map_vga.clone(),
        dac: g.native_map_dac.clone(),
        clock: g.native_map_clock,
        palette_phase_tick: (g.native_fdother_palette_phase, g.native_fdother_palette_tick),
        source: g.native_map_frame_source.clone(),
        hd_frame: g.modern_hd_map_frame.clone(),
        hd_presented: g.modern_hd_map_presented.clone(),
    })
}

/// Go `snapshotNativeCh20SkyKeyState` 閉包捕捉的值。
pub(crate) struct NativeCh20SkyKeySnapshot {
    state: Option<battle::State>,
    cursor_camera: (i64, i64, f64, f64),
    work: Vec<u8>,
    vga: Vec<u8>,
    dac: Vec<u8>,
    clock: NativeBIOSClock,
    palette_phase_tick: (i64, i64),
    source: Option<indexedmap::NativeFrameInput<'static>>,
    hd_frame: Option<ModernHDMapFrame>,
    hd_presented: Option<ModernHDMapFrame>,
}

impl NativeCh20SkyKeySnapshot {
    /// Go 只在 `g.st` 仍是同一個指標時還原整份 State；Rust 的 `st` 沒有身分，`st` 還在就還原。
    pub(crate) fn restore(self, g: &mut Game) {
        if g.st.is_some() {
            g.st = self.state;
        }
        (g.cur_x, g.cur_y, g.cam_x, g.cam_y) = self.cursor_camera;
        (g.native_map_work, g.native_map_vga, g.native_map_dac) = (self.work, self.vga, self.dac);
        g.native_map_clock = self.clock;
        (g.native_fdother_palette_phase, g.native_fdother_palette_tick) = self.palette_phase_tick;
        g.native_map_frame_source = self.source;
        (g.modern_hd_map_frame, g.modern_hd_map_presented) = (self.hd_frame, self.hd_presented);
    }
}

impl NativeCh20SkyKeyJob {
    /// Go：`native_ch20_sky_key.go:193`，`func (j *nativeCh20SkyKeyJob) refreshPalette() error`
    fn refresh_palette(&mut self) -> Result<()> {
        self.palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&self.dac)?);
        self.drawn = false;
        Ok(())
    }

    /// 0x135DD 每一步同時移動鏡頭與絕對游標，可見游標不動；先走 X 再走 Y，到 (14,8) 才開始合成。
    ///
    /// Go：`native_ch20_sky_key.go:202`，`func (j *nativeCh20SkyKeyJob) advancePan(g *Game) error`
    fn advance_pan(&mut self, g: &mut Game) -> Result<()> {
        let mut view = g.st.as_ref().unwrap().native_map_view_state;
        if view.camera_x == self.spec.pan_grid_x && view.camera_y == self.spec.pan_grid_y {
            return self.begin_first_frames(g);
        }
        if view.camera_x < self.spec.pan_grid_x {
            view.camera_x += 1;
            view.cursor_x += 1;
        } else if view.camera_x > self.spec.pan_grid_x {
            view.camera_x -= 1;
            view.cursor_x -= 1;
        } else if view.camera_y < self.spec.pan_grid_y {
            view.camera_y += 1;
            view.cursor_y += 1;
        } else {
            view.camera_y -= 1;
            view.cursor_y -= 1;
        }
        if let Err(err) = g.st.as_mut().unwrap().materialize_native_map_view_state(view) {
            return Err(game_err!("pan step: {err}"));
        }
        if !g.sync_native_map_view() {
            return Err(game_err!("pan step could not publish native map view"));
        }
        g.compose_native_map_frame().map_err(|err| game_err!("pan redraw: {}", err.message()))?;
        // Go 在這裡以 `freezeModernSkyKeyWorld` 凍結 HD 世界（結果只給 HD 幀讀）；Rust 留到 R6。
        self.vga.clone_from(&g.native_map_vga);
        self.dac.clone_from(&g.native_map_dac);
        self.ticks = 0;
        self.refresh_palette()
    }

    /// Go：`native_ch20_sky_key.go:236`，`func (j *nativeCh20SkyKeyJob) beginFirstFrames(g *Game) error`
    fn begin_first_frames(&mut self, g: &mut Game) -> Result<()> {
        if g.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE {
            return Err(game_err!("native 0x24336 baseline framebuffer is incomplete"));
        }
        self.base.clone_from(&g.native_map_vga);
        let base = self.spec.base_frame;
        if let Err(err) = self.fd_frames[base as usize].blit(&mut self.base, 320, -1) {
            return Err(game_err!("base frame {base}: {err}"));
        }
        (self.phase, self.frame) = (NATIVE_CH20_SKY_KEY_FIRST_FRAMES, self.spec.first_frame_start);
        self.prepare_fd_frame(g, true)
    }

    /// Go：`native_ch20_sky_key.go:249`，`func (j *nativeCh20SkyKeyJob) prepareFDFrame(g *Game, cycle bool) error`
    fn prepare_fd_frame(&mut self, g: &mut Game, cycle: bool) -> Result<()> {
        self.vga.clone_from(&self.base);
        let frame = self.frame;
        if let Err(err) = self.fd_frames[frame as usize].blit(&mut self.vga, 320, -1) {
            return Err(game_err!("frame {frame}: {err}"));
        }
        if cycle {
            // 0x4DFCC 在 BIOS tick 閘門後推進行程全域的 0..15 相位；重製保留相對循環與行程壽命，
            // 精確相位另以動態 E2 比對。
            g.native_fdother_palette_phase = (g.native_fdother_palette_phase + 1) & 15;
            fdother::apply_native_dac_palette_cycle_e0ef(&mut self.dac, g.native_fdother_palette_phase)?;
        }
        self.ticks = native_delay_ticks(self.spec.frame_wait_bios_ticks * 55);
        self.refresh_palette()
    }

    /// Go：`native_ch20_sky_key.go:267`，`func (j *nativeCh20SkyKeyJob) prepareANIFrame() error`
    fn prepare_ani_frame(&mut self) -> Result<()> {
        self.vga.clone_from(&self.ani_pixels[self.frame as usize]);
        self.dac.clone_from(&self.ani_dac[self.frame as usize]);
        self.ticks = native_delay_ticks(self.spec.ani_frame_delay_ms);
        self.refresh_palette()
    }

    /// Go：`native_ch20_sky_key.go:274`，`func (j *nativeCh20SkyKeyJob) beginFlash(baseline []byte) error`
    fn begin_flash(&mut self, baseline: &[u8]) -> Result<()> {
        self.phase = NATIVE_CH20_SKY_KEY_FLASH;
        self.dac = baseline.to_vec();
        let spec = &self.spec;
        fdother::apply_vga_palette_delta(
            &mut self.dac,
            baseline,
            spec.palette_start,
            spec.palette_end,
            spec.flash_delta,
        )?;
        self.ticks = native_delay_ticks(self.spec.flash_hold_ms);
        self.refresh_palette()
    }

    /// Go：`native_ch20_sky_key.go:284`，`func (j *nativeCh20SkyKeyJob) beginRestore(baseline []byte) error`
    fn begin_restore(&mut self, baseline: &[u8]) -> Result<()> {
        self.phase = NATIVE_CH20_SKY_KEY_RESTORE;
        self.dac = baseline.to_vec();
        let spec = &self.spec;
        fdother::apply_vga_palette_delta(
            &mut self.dac,
            baseline,
            spec.palette_start,
            spec.palette_end,
            spec.restore_delta,
        )?;
        self.ticks = native_delay_ticks(self.spec.restore_hold_ms);
        self.refresh_palette()
    }

    /// Go：`native_ch20_sky_key.go:294`，`func (j *nativeCh20SkyKeyJob) beginTail(g *Game) error`
    fn begin_tail(&mut self, g: &mut Game) -> Result<()> {
        (self.phase, self.frame) = (NATIVE_CH20_SKY_KEY_TAIL_FRAMES, self.spec.tail_frame_start);
        self.dac.clone_from(&g.native_map_assets.as_ref().unwrap().palette_dac);
        self.prepare_fd_frame(g, false)
    }
}

impl Game {
    /// 0x242C9 呼叫的 0x24336：素材全部預檢過才動鏡頭；捲動、FDOTHER#34 首段、ANI、白閃、還原、尾段依序播完
    /// 才把最後一幀交回地圖並續行。任何一步失敗都還原地圖交易。
    ///
    /// Go：`native_ch20_sky_key.go:88`，`func (g *Game) startNativeCh20SkyKeySequence(spec campaign.NativeCh20SkyKeySequence, then func()) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn start_native_ch20_sky_key_sequence(
        &mut self,
        spec: campaign::NativeCh20SkyKeySequence,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        if self.native_ch20_sky_key.is_some() {
            return Err(game_err!("native 0x24336 sequence is already active"));
        }
        if !spec.is_recovered_contract()
            || !native_map_assets_available(self.native_map_assets.as_ref())
            || self.m.is_none()
            || !self.st.as_ref().is_some_and(|st| st.has_native_map_view_state)
        {
            return Err(game_err!("native 0x24336 indexed map state is unavailable"));
        }
        if spec.fdother_resource != 34 {
            return Err(game_err!("native 0x24336 requires the recovered FDOTHER #34 contract"));
        }
        let frames =
            fdother::load_separated_ch20_sky_key_frames(separated_asset_path("animations/fdother_034_ch20_sky_key"))
                .map_err(|err| game_err!("FDOTHER #{}: {err}", spec.fdother_resource))?;
        let clip = load_native_ch20_sky_key_ani(&spec)
            .map_err(|err| game_err!("ANI #{}: {}", spec.ani_resource, err.message()))?;
        validate_native_ch20_sky_key_assets(&spec, &frames, Some(&clip))?;
        // Go 的 `preflightModernSkyKey` 在沒有 HD 主題（或主題沒有天空之鑰圖組）時回 nil；Rust 的主題還沒有
        // 天空之鑰圖組欄位，有主題就停在 R6。
        if self.modern_story_portraits.is_some() {
            todo!("R6: modern_hd_sky_key_assets.go:452 preflightModernSkyKey／prepareModernHDSkyKey（HD 主題）")
        }
        let rollback = snapshot_native_ch20_sky_key_state(self);

        // 先把整段捲動的終點算出來驗一次，再走第一格；終點要與 advance_pan 逐格走出來的結果相同，
        // 所以這裡也用鏡頭差量，不用 camera + visible 反推。
        let st = self.st.as_ref().unwrap();
        let mut view = st.native_map_view_state;
        view.cursor_x += spec.pan_grid_x - view.camera_x;
        view.cursor_y += spec.pan_grid_y - view.camera_y;
        (view.camera_x, view.camera_y) = (spec.pan_grid_x, spec.pan_grid_y);
        let mut carrier = battle::State::default();
        (carrier.w, carrier.h) = (st.w, st.h);
        if let Err(err) = carrier.materialize_native_map_view_state(view) {
            return Err(game_err!("native 0x24336 pan target: {err}"));
        }
        if !self.sync_native_map_view() {
            rollback.restore(self);
            return Err(game_err!("native 0x24336 current map view is unavailable"));
        }
        if let Err(err) = self.compose_native_map_frame() {
            rollback.restore(self);
            return Err(game_err!("native 0x24336 baseline frame: {}", err.message()));
        }
        let mut job = NativeCh20SkyKeyJob {
            spec,
            fd_frames: frames,
            ani_pixels: clip.indexed_frames,
            ani_dac: clip.palettes,
            dac: self.native_map_dac.clone(),
            phase: NATIVE_CH20_SKY_KEY_PAN,
            then,
            ..Default::default()
        };
        // Go 先把工作放進 `g.nativeCh20SkyKey` 再走第一步；`advancePan` 不讀這個欄位，Rust 走完才放。
        if let Err(err) = job.advance_pan(self) {
            rollback.restore(self);
            return Err(err);
        }
        job.rollback = Some(rollback);
        self.native_ch20_sky_key = Some(job);
        Ok(())
    }

    /// Go：`native_ch20_sky_key.go:300`，`func (g *Game) failNativeCh20SkyKey(err error)`
    pub(crate) fn fail_native_ch20_sky_key(&mut self, err: &str) {
        if let Some(rollback) = self.native_ch20_sky_key.take().and_then(|mut job| job.rollback.take()) {
            rollback.restore(self);
        }
        self.load_err = format!("native 0x24336: {err}");
    }

    /// Go：`native_ch20_sky_key.go:308`，`func (g *Game) stepNativeCh20SkyKey()`
    pub(crate) fn step_native_ch20_sky_key(&mut self) {
        let Some(mut j) = self.native_ch20_sky_key.take() else {
            return;
        };
        if !j.drawn {
            self.native_ch20_sky_key = Some(j);
            return;
        }
        if j.ticks > 0 {
            j.ticks -= 1;
            if j.ticks > 0 {
                self.native_ch20_sky_key = Some(j);
                return;
            }
        }
        let result = match j.phase {
            NATIVE_CH20_SKY_KEY_PAN => j.advance_pan(self),
            NATIVE_CH20_SKY_KEY_FIRST_FRAMES => {
                if j.frame < j.spec.first_frame_end {
                    j.frame += 1;
                    j.prepare_fd_frame(self, true)
                } else {
                    (j.phase, j.frame) = (NATIVE_CH20_SKY_KEY_ANI, 0);
                    j.prepare_ani_frame()
                }
            }
            NATIVE_CH20_SKY_KEY_ANI => {
                if j.frame + 1 < j.spec.ani_frame_count {
                    j.frame += 1;
                    j.prepare_ani_frame()
                } else {
                    let baseline = self.native_map_assets.as_ref().unwrap().palette_dac.clone();
                    j.begin_flash(&baseline)
                }
            }
            NATIVE_CH20_SKY_KEY_FLASH => {
                let baseline = self.native_map_assets.as_ref().unwrap().palette_dac.clone();
                j.begin_restore(&baseline)
            }
            NATIVE_CH20_SKY_KEY_RESTORE => j.begin_tail(self),
            NATIVE_CH20_SKY_KEY_TAIL_FRAMES => {
                if j.frame < j.spec.tail_frame_end {
                    j.frame += 1;
                    j.prepare_fd_frame(self, false)
                } else {
                    let then = j.then.take();
                    j.rollback = None;
                    self.native_map_vga.clone_from(&j.vga);
                    self.native_map_dac.clone_from(&j.dac);
                    if let Some(then) = then {
                        then(self);
                    }
                    return;
                }
            }
            _ => Err(game_err!("unknown presentation phase")),
        };
        // Go 在這裡以 `prepareModernHDSkyKey` 準備 HD 幀；沒有 HD 主題時回 nil（開始時已停在 R6），不做事。
        self.native_ch20_sky_key = Some(j);
        if let Err(err) = result {
            self.fail_native_ch20_sky_key(err.message());
        }
    }

    /// Go：`native_ch20_sky_key.go:370`，`func (g *Game) drawNativeCh20SkyKey(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_ch20_sky_key(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(j) = self.native_ch20_sky_key.as_mut() else {
            return false;
        };
        if j.vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE || j.palette.len() != 256 {
            return false;
        }
        if j.hd_frame.is_some() {
            todo!("r6:native_ch20_sky_key.go:375 drawPreparedModernHDMapFrame（HD 天空之鑰幀）")
        }
        if let Some(screen) = screen {
            let img = indexed_frame(320, 200, &j.vga, j.palette.clone());
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(&Picture::from_indexed(&img), &options);
        }
        j.drawn = true;
        true
    }
}

#[cfg(test)]
#[path = "native_ch20_sky_key/native_ch20_sky_key_test.rs"]
mod native_ch20_sky_key_test;
