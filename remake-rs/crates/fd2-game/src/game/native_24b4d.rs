//! Go 來源：`remake/cmd/fd2/native_24b4d.go`（162 行）。0x24B4D 兩列交替的上下震動呈現。
//!
//! 原版索引色分支；HD 列震動（`prepareModernHDRowShake`）與其他 Rust 演出相同另期。

use super::*;

impl Game {
    /// 0x24B4D：一次 0x11EEE 地形繪製產生兩個錯一列的 312×192 視窗，再以 20ms 交替呈現 `frame_count` 次。
    /// 故事場景（`st` 為 nil）借用場景的原生地圖載體組裝輸入，非同步呈現不把故事角色變成可操作戰鬥；
    /// 預檢失敗時復原故事角色、場景狀態與索引緩衝。
    ///
    /// Go：`native_24b4d.go:15`，`func (g *Game) startNative24B4D(frameCount, delayMs int, then func()) (err error)`
    pub(crate) fn start_native_24b4d(
        &mut self,
        frame_count: i64,
        delay_ms: i64,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<()> {
        if self.transition_reveal.is_some()
            || self.native2189_a.is_some()
            || self.native_ch20_sky_key.is_some()
            || self.native_ch23_loop.is_some()
            || self.indexed_transition.is_some()
        {
            return Err(GameError::new("native 0x24B4D presentation is already active"));
        }
        if frame_count <= 0 || frame_count > 255 || delay_ms != 20 {
            return Err(GameError::new("native 0x24B4D frame count or 20ms delay differs from recovered ABI"));
        }
        let saved = (
            self.story_actors.clone(),
            self.story_native_map_state.clone(),
            self.native_map_work.clone(),
            self.native_map_vga.clone(),
            self.native_map_dac.clone(),
        );
        let borrowed = self.st.is_none();
        let result = self.prepare_native_24b4d(frame_count, delay_ms, then);
        if borrowed {
            // 第25章在 LOADCH 故事中呼叫 0x24B4D；借用只限輸入組裝。
            self.story_native_map_state = self.st.take().or(self.story_native_map_state.take());
        }
        if result.is_err() {
            (
                self.story_actors,
                self.story_native_map_state,
                self.native_map_work,
                self.native_map_vga,
                self.native_map_dac,
            ) = saved;
            self.transition_reveal = None;
        }
        result
    }

    fn prepare_native_24b4d(
        &mut self,
        frame_count: i64,
        delay_ms: i64,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<()> {
        if self.st.is_none() {
            if self.story_native_map_state.is_none() {
                let source = self.story_native_map_source.clone();
                self.materialize_native_story_map_state(source.as_ref())
                    .map_err(|err| game_err!("native 0x24B4D story state: {}", err.message()))?;
            }
            self.compose_native_story_map_frame()
                .map_err(|err| game_err!("native 0x24B4D story frame: {}", err.message()))?;
            self.st = self.story_native_map_state.take();
        }
        if self.st.is_none()
            || self.m.is_none()
            || !native_map_assets_available(self.native_map_assets.as_ref())
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
        {
            return Err(GameError::new("native 0x24B4D indexed map state is unavailable"));
        }
        let (hud, ok) = self.native_map_hud_input();
        if !ok {
            return Err(GameError::new("native 0x24B4D HUD provenance is unavailable"));
        }
        let mut palette = self.native_map_assets.as_ref().unwrap().palette.clone();
        if self.native_map_dac.len() == 256 * 3
            && let Ok(current) = fdother::vga_palette_from_dac(&self.native_map_dac)
        {
            palette = color_palette_from_nrgba(&current);
        }
        let (map_work, map_vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        let mut input = build_native_map_frame_input(
            self.native_map_assets.as_ref(),
            self.m.as_ref(),
            self.st.as_ref(),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let mut work = map_work.clone();
        indexedmap::compose_native24_b4_d_staging(&mut work, &native_transition_input(&input))?;
        let mut steady_vga = map_vga.clone();
        indexedmap::compose_native_frame(&mut work, &mut steady_vga, &mut input)
            .map_err(|err| game_err!("native 0x24B4D steady 0x11CAC: {err}"))?;
        let mut frames: [Vec<u8>; 2] = [steady_vga.clone(), steady_vga];
        for (row, frame) in frames.iter_mut().enumerate() {
            fdother::copy_native_transition_viewport(
                &mut frame[0x504..],
                320,
                &work,
                fdother::NATIVE_TRANSITION_STAGE_OFFSET + row as i64 * fdother::NATIVE_TRANSITION_STAGE_STRIDE,
                fdother::NATIVE_TRANSITION_STAGE_STRIDE,
                fdother::NATIVE_TRANSITION_STAGE_WIDTH,
                fdother::NATIVE_TRANSITION_STAGE_HEIGHT,
            )
            .map_err(|err| game_err!("native 0x24B4D row {row} viewport: {err}"))?;
        }
        let delay = (delay_ms * 60 / 1000).max(1);
        let rollback: Box<dyn FnOnce(&mut Game) + Send> = Box::new(move |g: &mut Game| {
            (g.native_map_work, g.native_map_vga) = (map_work, map_vga);
        });
        self.native_map_work.clone_from(&work);
        self.native_map_vga.clone_from(&frames[0]);
        self.transition_reveal = Some(TransitionRevealJob {
            work,
            frames,
            palette,
            remaining: frame_count,
            delay,
            then: Some(then),
            rollback: Some(rollback),
            ..Default::default()
        });
        Ok(())
    }

    /// Go：`native_24b4d.go:134`，`func (g *Game) failNative24B4D(err error)`
    pub(crate) fn fail_native_24b4d(&mut self, err: &str) {
        if let Some(rollback) = self.transition_reveal.take().and_then(|mut job| job.rollback.take()) {
            rollback(self);
        }
        self.load_err = format!("native 0x24B4D: {err}");
    }

    /// 原版索引色分支：以保存的調色盤畫目前發布的 VGA 並承認呈現。
    ///
    /// Go：`native_24b4d.go:142`，`func (g *Game) drawNative24B4D(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_24b4d(&mut self, screen: Option<&mut Picture>) -> bool {
        let vga_ok = self.native_map_vga.len() as i64 == indexedmap::NATIVE_MAP_VGA_SIZE;
        let Some(job) = self.transition_reveal.as_mut().filter(|j| j.palette.len() == 256 && vga_ok) else {
            return false;
        };
        if let Some(screen) = screen {
            let img = indexed_frame(320, 200, &self.native_map_vga, job.palette.clone());
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(&Picture::from_indexed(&img), &options);
        }
        job.drawn = true;
        true
    }
}
