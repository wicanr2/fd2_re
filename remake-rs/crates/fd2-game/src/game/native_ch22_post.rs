//! Go 來源：`remake/cmd/fd2/native_ch22_post.go`（248 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// native2189AJob is the exact ten-pass indexed presentation owned by 0x2189A.
/// The address-shaped name is deliberate: evidence proves its buffer schedule,
/// not a universal gameplay or narrative meaning.
///
/// Go：`native_ch22_post.go:18`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct Native2189AJob {
    pub(crate) work_frames: Vec<Vec<u8>>,
    pub(crate) vga_frames: Vec<Vec<u8>>,
    pub(crate) palette: ColorPalette,
    pub(crate) index: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) rollback: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_terrain: Vec<u8>,
    pub(crate) hd_radial: Vec<fdother::RadialLUTRemap>,
    pub(crate) hd_luts: Vec<Vec<u8>>,
    pub(crate) hd_frames: Vec<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) map_stage: Option<NativeCh23AdapterState>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_steady: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_signature: [u8; 32],
}

/// Go：`native_ch22_post.go:39`，`func nativeTransitionInput(in indexedmap.NativeFrameInput) indexedmap.NativeTransitionFrameInput`
///
/// R4 B 組：回傳值借用 `in` 的素材（生命週期 `'a`），骨架的 `'static` 做不到。
pub(crate) fn native_transition_input<'a>(
    r#in: &indexedmap::NativeFrameInput<'a>,
) -> indexedmap::NativeTransitionFrameInput<'a> {
    let f = &r#in.frame;
    indexedmap::NativeTransitionFrameInput {
        parallax: f.parallax,
        parallax_scroll_x: f.parallax_scroll_x,
        parallax_scroll_y: f.parallax_scroll_y,
        terrain_bank: f.terrain_bank,
        unit_bank: f.unit_bank,
        foreground_bank: f.foreground_bank,
        selector_cache: f.selector_cache,
        cells: f.cells.clone(),
        controls: f.controls,
        terrain_lut: f.lut,
        map_width: f.map_width,
        camera_x: f.camera_x,
        camera_y: f.camera_y,
        flip: f.flip,
        terrain_cycle: f.terrain_cycle,
        idle_cycle: f.idle_cycle,
        moving_cycle: f.moving_cycle,
        pixel_shift: f.pixel_shift,
        units: f.units.clone(),
        foreground_units: f.foreground_units.clone(),
    }
}

impl Game {
    /// 0x2189A 十段圓形 LUT 演出：十段畫面加上 0x219A3 呼叫 0x11CAC(0) 的穩定畫面全部預先算好，才發布第一段。
    /// HD 綁定（指令 10～12 的前奏）與其他 Rust 演出相同另期。
    ///
    /// Go：`native_ch22_post.go:52`，`func (g *Game) startNative2189A(loop campaign.Native2189ALoop, then func()) error`
    pub(crate) fn start_native_2189a(
        &mut self,
        lp: &campaign::Native2189ALoop,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<()> {
        if self.native2189_a.is_some()
            || self.native_ch20_sky_key.is_some()
            || self.native_ch23_loop.is_some()
            || self.indexed_transition.is_some()
        {
            return Err(GameError::new("native 0x2189A presentation is already active"));
        }
        let (slot, initial_radius, radius_step) = (lp.slot, lp.initial_radius, lp.radius_step);
        let ch22_call = (slot == 10 && initial_radius == 15 && radius_step == 1)
            || (slot == 16 && initial_radius == 30 && radius_step == 1);
        let command_call =
            slot >= 0 && ((initial_radius == 15 && radius_step == 10) || (initial_radius == 30 && radius_step == 16));
        if !(ch22_call || command_call)
            || lp.repeat != 10
            || lp.work_offset != 0x8088
            || lp.work_stride != 456
            || lp.clip_width != 312
            || lp.clip_height != 192
        {
            return Err(GameError::new("native 0x2189A call differs from recovered ch22/command11-12 contracts"));
        }
        if self.native_cmd1012.is_some() {
            return Err(GameError::new("native 0x2189A command10-12 HD prelude is not ported"));
        }
        if self.st.as_ref().is_none_or(|st| !st.has_native_map_view_state)
            || self.m.is_none()
            || !native_map_assets_available(self.native_map_assets.as_ref())
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
        {
            return Err(GameError::new("native 0x2189A indexed map state is unavailable"));
        }
        let Some(HandlerUnitRef::Battle(unit)) = self.handler_unit_at(slot) else {
            return Err(game_err!("native 0x2189A slot {slot} lacks raw +0/+1 presentation coordinates"));
        };
        let presentation = {
            let u = &self.st.as_ref().unwrap().units[unit];
            if !u.has_native_map_presentation {
                return Err(game_err!("native 0x2189A slot {slot} lacks raw +0/+1 presentation coordinates"));
            }
            u.native_map_presentation
        };
        let assets = self.native_map_assets.as_ref().unwrap();
        for i in 0..10 {
            if assets.luts.get(i).is_none_or(|lut| lut.len() != 256) {
                return Err(game_err!("native 0x2189A FDOTHER#3 LUT {i} is unavailable"));
            }
        }
        let (hud, ok) = self.native_map_hud_input();
        if !ok {
            return Err(GameError::new("native 0x2189A HUD provenance is unavailable"));
        }
        let stage = if self.native_map_assets.as_ref().unwrap().map_index == 23 {
            Some(
                self.native_ch23_state
                    .clone()
                    .ok_or_else(|| GameError::new("native 0x2189A same-compose stage absent"))?,
            )
        } else {
            None
        };
        let mut palette = self.native_map_assets.as_ref().unwrap().palette.clone();
        if self.native_map_dac.len() == 256 * 3
            && let Ok(current) = fdother::vga_palette_from_dac(&self.native_map_dac)
        {
            palette = color_palette_from_nrgba(&current);
        }
        let (map_work, map_vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        let assets = self.native_map_assets.as_ref().unwrap();
        let mut input = build_native_map_frame_input(
            Some(assets),
            self.m.as_ref(),
            self.st.as_ref(),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let transition = native_transition_input(&input);
        let mut snapshot = map_work.clone();
        if let Some(stage) = stage.as_ref() {
            indexedmap::seed_native_ch23_staging(&mut snapshot, &stage.staging)?;
        }
        indexedmap::compose_native_unit_present_terrain_snapshot(&mut snapshot, &transition)?;
        let center_x = (i64::from(presentation.x) - transition.camera_x) * 24 + 12;
        let center_y = (i64::from(presentation.y) - transition.camera_y) * 24 + 18;
        let (mut work_frames, mut vga_frames) = (Vec::with_capacity(11), Vec::with_capacity(11));
        let offset = lp.work_offset as usize;
        for i in 0..10 {
            let mut work = snapshot.clone();
            let mut vga = map_vga.clone();
            let radius = initial_radius + i as i64 * radius_step;
            fdother::apply_radial_lut_remap(
                &mut work[offset..],
                lp.work_stride,
                &assets.luts[i],
                fdother::RadialLUTRemap {
                    center_x,
                    center_y,
                    radius,
                    scale: 12,
                    start_y: 0,
                    end_y: lp.clip_height,
                    clip_width: lp.clip_width,
                },
            )
            .map_err(|err| game_err!("native 0x2189A pass {i}: {err}"))?;
            indexedmap::redraw_native_unit_present_objects(&mut work, &transition)
                .map_err(|err| game_err!("native 0x2189A pass {i} object redraw: {err}"))?;
            indexedmap::copy_native_unit_present_viewport(&mut vga, &work)
                .map_err(|err| game_err!("native 0x2189A pass {i} viewport: {err}"))?;
            work_frames.push(work);
            vga_frames.push(vga);
        }
        // 0x219A3 在十段之後呼叫一般 0x11CAC(0)；先建好穩定畫面，尾段失敗時不留下半套節拍。
        let (mut steady_work, mut steady_vga) = (map_work.clone(), map_vga.clone());
        if let Some(stage) = stage.as_ref() {
            indexedmap::seed_native_ch23_staging(&mut steady_work, &stage.staging)?;
        }
        indexedmap::compose_native_frame(&mut steady_work, &mut steady_vga, &mut input)
            .map_err(|err| game_err!("native 0x2189A steady restore: {err}"))?;
        work_frames.push(steady_work);
        vga_frames.push(steady_vga);
        let rollback: Box<dyn FnOnce(&mut Game) + Send> = Box::new(move |g: &mut Game| {
            (g.native_map_work, g.native_map_vga) = (map_work, map_vga);
        });
        self.native2189_a = Some(Native2189AJob {
            work_frames,
            vga_frames,
            palette,
            then: Some(then),
            rollback: Some(rollback),
            map_stage: stage,
            ..Default::default()
        });
        self.publish_native_2189a_frame();
        Ok(())
    }

    /// Go：`native_ch22_post.go:193`，`func (g *Game) publishNative2189AFrame()`
    pub(crate) fn publish_native_2189a_frame(&mut self) {
        let Some(j) = self.native2189_a.as_mut() else {
            return;
        };
        let Some(index) = usize::try_from(j.index).ok().filter(|&i| i < j.vga_frames.len()) else {
            return;
        };
        self.native_map_work.clone_from(&j.work_frames[index]);
        self.native_map_vga.clone_from(&j.vga_frames[index]);
        j.drawn = false;
    }

    /// Go：`native_ch22_post.go:203`，`func (g *Game) stepNative2189A()`
    pub(crate) fn step_native_2189a(&mut self) {
        let Some(j) = self.native2189_a.as_mut().filter(|j| j.drawn) else {
            return;
        };
        j.index += 1;
        if j.index < j.vga_frames.len() as i64 {
            self.publish_native_2189a_frame();
            return;
        }
        let mut j = self.native2189_a.take().unwrap();
        j.rollback = None;
        if let Some(then) = j.then.take() {
            then(self);
        }
    }

    /// Go：`native_ch22_post.go:221`，`func (g *Game) failNative2189A(err error)`
    pub(crate) fn fail_native_2189a(&mut self, err: &str) {
        if let Some(rollback) = self.native2189_a.take().and_then(|mut j| j.rollback.take()) {
            rollback(self);
        }
        self.load_err = format!("native 0x2189A: {err}");
    }

    /// 原版索引色分支：以保存的調色盤畫目前發布的 VGA 並承認呈現。
    ///
    /// Go：`native_ch22_post.go:229`，`func (g *Game) drawNative2189A(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_2189a(&mut self, screen: Option<&mut Picture>) -> bool {
        let vga_ok = self.native_map_vga.len() as i64 == indexedmap::NATIVE_MAP_VGA_SIZE;
        let Some(j) = self.native2189_a.as_mut().filter(|j| j.palette.len() == 256 && vga_ok) else {
            return false;
        };
        if let Some(screen) = screen {
            let img = indexed_frame(320, 200, &self.native_map_vga, j.palette.clone());
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(&Picture::from_indexed(&img), &options);
        }
        j.drawn = true;
        true
    }
}
