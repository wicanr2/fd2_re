//! Go 來源：`remake/cmd/fd2/native_ch28_post_present.go`（363 行）。
//!
//! 骨架：Go 檔的 9 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! R4 B 組：5 個函式移植完。`0x1DB65` 是第 28 章戰後與一般死亡事件共用的單位消失演出；第一章經
//! `startNativeUnitDeathPresentMode`（D 組的死亡演出呼叫）走到。第 24 章地圖（map 23）的同幀舞台與 HD
//! 死亡素材分支停在 `todo!("R8／R6: …")`。

use super::*;

/// nativeCh28PostPresentJob owns sub_1DB65's 13 pose frames, two six-frame
/// FDOTHER#5 overlay loops, raw sample index 3, and final 0x11CAC redraw. The
/// address-shaped name avoids inventing a player-facing effect name.
///
/// Go：`native_ch28_post_present.go:18`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
///
/// R4 B 組：`retained_state` 由 `Option<battle::State>` 改成 `bool`。Go 存的是 `g.st` 指標本身
/// （`job.retainedState = g.st`），發布與 rollback 時再指回同一個物件；Rust 的 `g.st` 由 `Game` 擁有，
/// 保留模式就是不替換 `g.st`，只同步 HUD 狀態。`hd_inputs` 只給 HD（map 23 舞台與 HD 死亡素材）用，
/// 忠實模式不填。
#[derive(Default)]
pub(crate) struct NativeCh28PostPresentJob {
    pub(crate) work_frames: Vec<Vec<u8>>,
    pub(crate) vga_frames: Vec<Vec<u8>>,
    pub(crate) state_frames: Vec<battle::State>,
    pub(crate) waits: Vec<i64>,
    pub(crate) palette: ColorPalette,
    pub(crate) frame: i64,
    pub(crate) wait: i64,
    pub(crate) drawn: bool,
    pub(crate) sfx_at: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) rollback: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd_inputs: Vec<indexedmap::NativeFrameInput<'static>>,
    pub(crate) hd_stage: Option<ModernHDDeathStageSource>,
    pub(crate) hd_targets: Vec<battle::NativeCh28PostOverlayTarget>,
    pub(crate) hd_frames: Vec<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) presented_frame: i64,
    pub(crate) presented_hd: bool,
    /// ordinary settlement keeps its live State/Unit identities
    pub(crate) retained_state: bool,
    pub(crate) kind: String,
}

/// Go：`native_ch28_post_present.go:41`，`func cloneNativeCh28PostState(source *battle.State) (*battle.State, error)`
pub(crate) fn clone_native_ch28_post_state(source: &mut battle::State) -> Result<Option<battle::State>> {
    clone_native_turn_staging_state(Some(source))
}

impl Game {
    /// Go：`native_ch28_post_present.go:45`，`func (g *Game) nativeCh28PostFrameInput(state *battle.State) (indexedmap.NativeFrameInput, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// R4 B 組：Go 暫時把 `g.st` 指向 `state` 取 HUD 輸入再換回；Rust 把 `state` 的內容換進 `self.st`、
    /// 取完再換回呼叫端（不複製狀態）。回傳的輸入借用 `self` 的素材庫與 `state`（C 組
    /// `build_native_map_frame_input` 的借用簽名，120「保存資源、每次重組輸入」）。
    pub(crate) fn native_ch28_post_frame_input<'a>(
        &'a mut self,
        state: Option<&'a mut battle::State>,
    ) -> Result<indexedmap::NativeFrameInput<'a>> {
        let Some(state) = state else {
            return Err(GameError::new("native ch28 post: state unavailable"));
        };
        let before = self.st.replace(std::mem::take(state));
        let (hud, ok) = self.native_map_hud_input();
        *state = std::mem::replace(&mut self.st, before).expect("nativeMapHUDInput 不會清掉 g.st");
        if !ok {
            return Err(GameError::new("native ch28 post: HUD provenance unavailable"));
        }
        let this: &'a Game = self;
        let state: &'a battle::State = state;
        build_native_map_frame_input(
            this.native_map_assets.as_ref(),
            this.m.as_ref(),
            Some(state),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )
    }

    /// Go：`native_ch28_post_present.go:69`，`func (g *Game) startNativeUnitDeathPresentMode(prelude func(*battle.State) error, then func(), retainState bool) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    ///
    /// R4 B 組：`prelude` 改收 `&mut battle::State`（Go 的 prelude 改寫候選狀態，例如清 +0x40）。
    pub(crate) fn start_native_unit_death_present_mode(
        &mut self,
        prelude: &mut dyn FnMut(&mut battle::State) -> Result<()>,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
        retain_state: bool,
    ) -> Result<()> {
        if self.native_ch28_post_present.is_some()
            || self.native_unit_present.is_some()
            || self.native2189_a.is_some()
            || self.transition_reveal.is_some()
            || self.indexed_transition.is_some()
            || self.native_palette_ramp.is_some()
            || self.native_palette_pulse.is_some()
            || self.native_ch20_sky_key.is_some()
            || self.native_ch23_loop.is_some()
        {
            return Err(GameError::new("native 0x1DB65 presentation is already active"));
        }
        if self.st.is_none()
            || self.m.is_none()
            || !native_map_assets_available(self.native_map_assets.as_ref())
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || !self.st.as_ref().unwrap().has_native_map_view_state
        {
            return Err(GameError::new("native 0x1DB65 indexed map state is unavailable"));
        }
        let assets = self.native_map_assets.as_ref().unwrap();
        fdother::validate_native_ch28_post_lmi(&assets.command_heal_digits)?;
        let heal_digits = assets.command_heal_digits.clone();
        let (asset_palette, map_index) = (assets.palette.clone(), assets.map_index);
        let sample_missing = self.sfx.get(&fdother::NATIVE_CH28_POST_SFX_INDEX).is_none_or(|s| s.is_empty());
        if !os_mute_or_shot(Some(self)) && sample_missing {
            return Err(game_err!(
                "native 0x1DB65 raw sample unavailable: resource={} index={}",
                fdother::NATIVE_CH28_POST_SFX_RESOURCE,
                fdother::NATIVE_CH28_POST_SFX_INDEX
            ));
        }

        let mut candidate =
            clone_native_ch28_post_state(self.st.as_mut().unwrap())?.expect("cloneNativeCh28PostState 成功時不回 nil");
        prelude(&mut candidate)?;
        let view = candidate.native_map_view_state;
        let plan = battle::plan_native_ch28_post_presentation(Some(&candidate), view.camera_x, view.camera_y)?;

        let mut job = NativeCh28PostPresentJob {
            palette: asset_palette,
            sfx_at: fdother::NATIVE_CH28_POST_POSE_FRAMES,
            then,
            hd_targets: plan.targets.clone(),
            presented_frame: -1,
            kind: String::from("script"),
            ..Default::default()
        };
        if retain_state {
            job.retained_state = true;
        }
        if map_index == 23 {
            todo!("R8: native_ch28_post_present.go:hdStage（map 23 的同幀舞台；第一章沒有執行到）")
        }
        if self.native_map_dac.len() == 256 * 3
            && let Ok(current) = fdother::vga_palette_from_dac(&self.native_map_dac)
        {
            job.palette = color_palette_from_nrgba(&current);
        }
        // Go 的 appendFrame 閉包（hdStage 為 nil 時的部分；HD 輸入不收）。
        fn append_frame(
            job: &mut NativeCh28PostPresentJob,
            work: &[u8],
            vga: &[u8],
            state: &mut battle::State,
            wait: i64,
        ) -> Result<()> {
            let snapshot = clone_native_ch28_post_state(state)?.expect("cloneNativeCh28PostState 成功時不回 nil");
            job.work_frames.push(work.to_vec());
            job.vga_frames.push(vga.to_vec());
            job.state_frames.push(snapshot);
            job.waits.push(wait.max(1));
            Ok(())
        }

        let mut work = self.native_map_work.clone();
        let mut vga = self.native_map_vga.clone();
        for frame in 0..fdother::NATIVE_CH28_POST_POSE_FRAMES {
            battle::apply_native_ch28_post_pose_frame(Some(&mut candidate), frame)?;
            let r#in = self.native_ch28_post_frame_input(Some(&mut candidate))?;
            let transition = native_transition_input(&r#in);
            if let Some(stage) = job.hd_stage.as_ref() {
                stage.seed(&mut work)?;
            }
            indexedmap::compose_native_unit_present_terrain_snapshot(&mut work, &transition)
                .map_err(|err| game_err!("native 0x1DB65 pose frame {} terrain: {err}", frame))?;
            indexedmap::redraw_native_unit_present_objects(&mut work, &transition)
                .map_err(|err| game_err!("native 0x1DB65 pose frame {} objects: {err}", frame))?;
            indexedmap::copy_native_unit_present_viewport(&mut vga, &work)
                .map_err(|err| game_err!("native 0x1DB65 pose frame {} viewport: {err}", frame))?;
            append_frame(&mut job, &work, &vga, &mut candidate, 3)?;
        }
        battle::apply_native_ch28_post_inactive_mark(Some(&mut candidate))?;
        let after_input = self.native_ch28_post_frame_input(Some(&mut candidate))?;
        let transition = native_transition_input(&after_input);
        let mut staged = vec![0u8; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
        if let Some(stage) = job.hd_stage.as_ref() {
            stage.seed(&mut staged)?;
        }
        indexedmap::compose_native_unit_present_terrain_snapshot(&mut staged, &transition)
            .map_err(|err| game_err!("native 0x1DB65 staged terrain: {err}"))?;
        indexedmap::redraw_native_unit_present_objects(&mut staged, &transition)
            .map_err(|err| game_err!("native 0x1DB65 staged objects: {err}"))?;
        for (index, frame) in fdother::native_ch28_post_overlay_frames().iter().enumerate() {
            if frame.phase == "overlay_b" {
                work.clear();
                work.extend_from_slice(&staged);
            }
            for target in &plan.targets {
                let entry = heal_digits.get(frame.entry as usize).expect("Go：LMI 索引越界 panic");
                fdother::blit_native_ch28_post_overlay(entry, &mut work, target.origin)
                    .map_err(|err| game_err!("native 0x1DB65 overlay frame {} slot{}: {err}", index, target.slot))?;
            }
            indexedmap::copy_native_unit_present_viewport(&mut vga, &work)
                .map_err(|err| game_err!("native 0x1DB65 overlay frame {} viewport: {err}", index))?;
            append_frame(&mut job, &work, &vga, &mut candidate, 3)?;
        }
        let mut steady_work = work.clone();
        let mut steady_vga = vga.clone();
        // 0x1DEAE 只在最後 steady redraw 評估。candidate 不發布到前25幀，
        // 任何合成失敗仍由既有預檢／rollback 保留正式狀態。
        // Go：redraw := Game{st: candidate}; redraw.redrawNativeMapHUD()
        redraw_native_map_hud_state(&mut candidate);
        let mut after_input = self.native_ch28_post_frame_input(Some(&mut candidate))?;
        if let Some(stage) = job.hd_stage.as_ref() {
            stage.seed(&mut steady_work)?;
        }
        indexedmap::compose_native_frame(&mut steady_work, &mut steady_vga, &mut after_input)
            .map_err(|err| game_err!("native 0x1DB65 final steady redraw: {err}"))?;
        append_frame(&mut job, &steady_work, &steady_vga, &mut candidate, 1)?;
        if job.vga_frames.len() != 26
            || job.work_frames.len() != job.vga_frames.len()
            || job.state_frames.len() != job.vga_frames.len()
            || job.waits.len() != job.vga_frames.len()
        {
            return Err(GameError::new("native 0x1DB65 preflight produced an invalid 13+6+6+steady schedule"));
        }
        if job.hd_stage.is_some() {
            todo!("R8: modern_hd_death_stage.go:deathStageDigest／validateModernHDDeathStageSource（map 23）")
        }
        if has_modern_hd_death_bank(self.modern_story_portraits.as_ref()) {
            todo!("R6: modern_hd_death_runtime.go:prepareModernHDDeathFrames（HD 死亡素材）")
        }

        let before_state =
            clone_native_ch28_post_state(self.st.as_mut().unwrap())?.expect("cloneNativeCh28PostState 成功時不回 nil");
        let before_work = self.native_map_work.clone();
        let before_vga = self.native_map_vga.clone();
        let retained = job.retained_state;
        job.rollback = Some(Box::new(move |g: &mut Game| {
            if retained {
                // Go：g.st = job.retainedState（就是當時的 g.st）後只還原 HUD 狀態。
                if let Some(st) = g.st.as_mut() {
                    st.native_map_hud_state = before_state.native_map_hud_state;
                }
            } else {
                g.st = Some(before_state);
            }
            g.native_map_work = before_work;
            g.native_map_vga = before_vga;
        }));
        self.native_ch28_post_present = Some(job);
        self.publish_native_ch28_post_frame();
        Ok(())
    }

    /// Go：`native_ch28_post_present.go:283`，`func (g *Game) publishNativeCh28PostFrame()`
    ///
    /// R4 B 組：Go 非保留模式讓 `g.st` 指向 job 的快照（共用指標），Rust 複製快照；下一幀發布時兩邊都整個換掉。
    pub(crate) fn publish_native_ch28_post_frame(&mut self) {
        let Some(j) =
            self.native_ch28_post_present.as_mut().filter(|j| j.frame >= 0 && j.frame < j.vga_frames.len() as i64)
        else {
            return;
        };
        let frame = j.frame as usize;
        if j.retained_state {
            if frame == j.state_frames.len() - 1
                && let Some(st) = self.st.as_mut()
            {
                st.native_map_hud_state = j.state_frames[frame].native_map_hud_state;
            }
        } else {
            self.st = Some(j.state_frames[frame].clone());
        }
        self.native_map_work.clear();
        self.native_map_work.extend_from_slice(&j.work_frames[frame]);
        self.native_map_vga.clear();
        self.native_map_vga.extend_from_slice(&j.vga_frames[frame]);
        j.drawn = false;
        j.wait = j.waits[frame] - 1;
        if j.wait < 0 {
            j.wait = 0;
        }
    }

    /// Go：`native_ch28_post_present.go:305`，`func (g *Game) stepNativeCh28PostPresent()`
    ///
    /// R4 B 組：`drawn` 由呈現回報設定（Go 在 `drawNativeCh28PostPresent` 設）。
    pub(crate) fn step_native_ch28_post_present(&mut self) {
        let Some(j) = self.native_ch28_post_present.as_mut().filter(|j| j.drawn) else {
            return;
        };
        if j.wait > 0 {
            j.wait -= 1;
            return;
        }
        let next = j.frame + 1;
        if next >= j.vga_frames.len() as i64 {
            let mut j = self.native_ch28_post_present.take().unwrap();
            let then = j.then.take();
            j.rollback = None;
            if let Some(then) = then {
                then(self);
            }
            return;
        }
        let play = next == j.sfx_at;
        if play {
            self.play_sfx(fdother::NATIVE_CH28_POST_SFX_INDEX);
        }
        if let Some(j) = self.native_ch28_post_present.as_mut() {
            j.frame = next;
        }
        self.publish_native_ch28_post_frame();
    }
}

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_ch28_post_present.go:339`，`func (g *Game) drawNativeCh28PostPresent(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5b：逐行。`drawn`、`presentedFrame`、`presentedHD` 照 Go 在這裡設（`draw` 路徑）；HD 死亡素材
    /// 的幀（`hasModernHDDeathBank`）是 `todo!("r6:…")`，忠實模式下 `prepare` 也不會建立 HD 幀。
    pub(crate) fn draw_native_ch28_post_present(&mut self, screen: &mut Picture) -> bool {
        let Some(j) = self.native_ch28_post_present.as_ref() else {
            return false;
        };
        if j.palette.len() != 256 || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE {
            return false;
        }
        if has_modern_hd_death_bank(self.modern_story_portraits.as_ref()) {
            todo!("r6:native_ch28_post_present.go:344 drawPreparedModernHDMapFrame（HD 死亡素材的幀）")
        }
        // Go：image.NewPaletted(320×200, j.palette)，copy(img.Pix, g.nativeMapVGA)（長度已核對）。
        let img =
            IndexedFrame { width: 320, height: 200, pix: self.native_map_vga.clone(), palette: j.palette.clone() };
        let mut op = DrawImageOptions::default();
        op.geo_m.scale(2.0, 2.0);
        screen.draw_image(&Picture::from_indexed(&img), &op);
        let j = self.native_ch28_post_present.as_mut().unwrap();
        (j.presented_frame, j.presented_hd) = (j.frame, false);
        j.drawn = true;
        true
    }
}
