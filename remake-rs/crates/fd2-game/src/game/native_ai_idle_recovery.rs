//! Go 來源：`remake/cmd/fd2/native_ai_idle_recovery.go`（350 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 6 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_WORK_BASE: i64 = 0x8088;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_VGA_OFFSET: i64 = 0x504;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_WIDTH: i64 = 312;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_HEIGHT: i64 = 192;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_STRIDE: i64 = 0x1c8;

pub(crate) const NATIVE_AI_IDLE_RECOVERY_VGA_STRIDE: i64 = 320;

/// nativeAIIdleRecoveryJob owns the asynchronous presentation portion of
/// 0x13FD4. The job presents initial, mode2 mask and mode0 raw-restore frames,
/// as specified by docs/data/ida/fd2_ai_idle_recovery_20261004.json, one per
/// 0x17AA9(1) wait.  HP is committed only after the third present acknowledgement
/// and a fresh raw-record preflight.
///
/// Go：`native_ai_idle_recovery.go:29`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeAIIdleRecoveryJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: battle::NativeAIIdleRecoveryPresentation,
    pub(crate) decision: battle::NativeAIIdleRecoveryDecision,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) before_range: i64,
    pub(crate) before_has: bool,
    pub(crate) after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    /// beginNativeAIIdleRecovery is the concrete indexed/audio owner for the
    /// 0x14121→0x13FD4 edge.  It deliberately uses the already verified native map
    /// compositor, FDICON 24×24 RLE path and FDOTHER#31 index-4 sample.  It does not
    /// assign a high-level name to either 0x1DA16 decode mode.
    ///
    /// Go：`native_ai_idle_recovery.go:45`，`func (g *Game) beginNativeAIIdleRecovery( actor *battle.Unit, decision battle.NativeAIIdleRecoveryDecision, after func(), ) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_ai_idle_recovery(
        &mut self,
        actor: Option<UnitIdx>,
        decision: battle::NativeAIIdleRecoveryDecision,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        let (Some(actor), true, true) = (actor, self.st.is_some(), self.m.is_some()) else {
            return Err(game_err!("native AI 0x13fd4: runtime state is unavailable"));
        };
        if self.native_ai_idle_recovery.is_some() {
            return Err(game_err!("native AI 0x13fd4: presentation is already active"));
        }
        if !decision.accepted {
            return Err(game_err!("native AI 0x13fd4: caller supplied a rejected raw decision"));
        }
        let st = self.st.as_ref().unwrap();
        let (before_range, before_has) = (st.native_map_range_mode, st.has_native_map_range_mode_state);
        let restore_range = |g: &mut Game| {
            let st = g.st.as_mut().unwrap();
            st.native_map_range_mode = before_range;
            st.has_native_map_range_mode_state = before_has;
        };
        let records = battle::native_ai_scoring_records(&st.units)
            .map_err(|e| GameError::from(e).wrap("native AI 0x13fd4: raw records: "))?;
        let actor_index = unit_index(Some(st), Some(actor));
        match battle::plan_native_ai_idle_recovery(&records, st.units.len() as i64, actor_index) {
            Err(err) => return Err(GameError::from(err).wrap("native AI 0x13fd4: raw preflight: ")),
            Ok(current) if current != decision => {
                return Err(game_err!("native AI 0x13fd4: raw decision changed before presentation"));
            }
            Ok(_) => {}
        }
        let presentation = battle::build_native_ai_idle_recovery_presentation(decision)?;
        if presentation.coordinate_call.address != 0x12D7B
            || presentation.coordinate_call.unit != actor_index
            || presentation.sample_handle_expr != "[0x53EEC]"
            || presentation.sample_index != 4
            || presentation.sample_loop_count != 1
        {
            return Err(game_err!("native AI 0x13fd4: raw presentation tuple changed"));
        }
        let sample = i64::from(presentation.sample_index);
        if !os_mute_or_shot(Some(self))
            && (sample >= self.sfx.len() as i64 || self.sfx.get(&sample).is_none_or(|s| s.is_empty()))
        {
            return Err(game_err!(
                "native AI 0x13fd4: raw sample unavailable handle={} index={}",
                presentation.sample_handle_expr,
                presentation.sample_index
            ));
        }
        if self.native_map_dac.len() != 256 * 3 || !native_map_assets_available(self.native_map_assets.as_ref()) {
            return Err(game_err!("native AI 0x13fd4: indexed palette/assets are unavailable"));
        }
        // The first proven write is [0x51A83]=0. Apply it before the compositor
        // snapshot so 0x12D7B/0x1DA16 consume the same raw range state; every
        // pre-job failure restores the caller's state.
        let st = self.st.as_mut().unwrap();
        st.native_map_range_mode = 0;
        st.has_native_map_range_mode_state = true;
        let (before_cam_x, before_cam_y) = (self.cam_x, self.cam_y);
        let (before_cur_x, before_cur_y) = (self.cur_x, self.cur_y);
        let (before_battle_view, before_battle_view_has) = (st.native_map_view_state, st.has_native_map_view_state);
        let (before_story_view, before_story_view_has) = (self.story_native_map_view, self.has_story_native_map_view);
        let restore_focus = |g: &mut Game| {
            (g.cam_x, g.cam_y) = (before_cam_x, before_cam_y);
            (g.cur_x, g.cur_y) = (before_cur_x, before_cur_y);
            let st = g.st.as_mut().unwrap();
            st.native_map_view_state = before_battle_view;
            st.has_native_map_view_state = before_battle_view_has;
            g.story_native_map_view = before_story_view;
            g.has_story_native_map_view = before_story_view_has;
        };
        // 0x12D7B 先把 actor 座標交給 0x12CEA。range mode 已是 0，所以原版
        // 不在逐格同步間等待；這裡以同一個安全帶實作同步完成游標／鏡頭，再建立
        // 0x1DA16 畫格，避免把鏡頭外的合法 actor 誤判成損壞資產。
        if self.focus_job.is_some() {
            restore_range(self);
            return Err(game_err!("native AI 0x13fd4: coordinate synchronization is already active"));
        }
        let (target_x, target_y) = {
            let unit = &self.st.as_ref().unwrap().units[actor];
            (unit.x, unit.y)
        };
        self.focus_job = Some(FocusUnitJob {
            target_x,
            target_y,
            native_view: true,
            then: Some(Box::new(|_: &mut Game| {})),
            ..Default::default()
        });
        let m = self.m.as_ref().unwrap();
        let limit = m.w + m.h + 2;
        let mut step = 0;
        while self.focus_job.is_some() && step < limit {
            self.step_focus_unit();
            step += 1;
        }
        if self.focus_job.is_some() || !self.load_err.is_empty() {
            self.focus_job = None;
            restore_focus(self);
            restore_range(self);
            if !self.load_err.is_empty() {
                return Err(game_err!("native AI 0x13fd4: coordinate synchronization: {}", self.load_err));
            }
            return Err(game_err!("native AI 0x13fd4: coordinate synchronization exceeded {limit} steps"));
        }
        if let Err(err) = self.compose_native_map_frame() {
            restore_focus(self);
            restore_range(self);
            return Err(err.wrap("native AI 0x13fd4: source frame: "));
        }
        let frames = match self.build_native_ai_idle_recovery_frames(Some(actor), &presentation) {
            Ok(frames) => frames,
            Err(err) => {
                restore_focus(self);
                restore_range(self);
                return Err(err);
            }
        };
        if frames.len() != 3 {
            restore_focus(self);
            restore_range(self);
            return Err(game_err!("native AI 0x13fd4: frame owner returned {} frames, want 3", frames.len()));
        }
        // Keep a restoration snapshot so a renderer/runtime failure cannot leak a
        // partially consumed range state.
        self.native_ai_idle_recovery = Some(NativeAIIdleRecoveryJob {
            actor: Some(actor),
            plan: presentation,
            decision,
            frames,
            after,
            before_range,
            before_has,
            ..Default::default()
        });
        Ok(())
    }

    /// buildNativeAIIdleRecoveryFrames consumes the proven 24×24 FDICON raw
    /// decoder and the corrected 312×192 viewport-copy ABI. The 0x1402B wait
    /// presents the initial source; 0x14094 presents the C8 write mask, and
    /// 0x140EA presents the mode0 raw restore before HP is committed.
    ///
    /// Go：`native_ai_idle_recovery.go:167`，`func (g *Game) buildNativeAIIdleRecoveryFrames( actor *battle.Unit, presentation battle.NativeAIIdleRecoveryPresentation, ) ([][]byte, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn build_native_ai_idle_recovery_frames(
        &mut self,
        actor: Option<UnitIdx>,
        presentation: &battle::NativeAIIdleRecoveryPresentation,
    ) -> Result<Vec<Vec<u8>>> {
        let (Some(st), Some(assets), Some(actor)) = (self.st.as_ref(), self.native_map_assets.as_ref(), actor) else {
            return Err(game_err!("native AI 0x13fd4: frame inputs are unavailable"));
        };
        if presentation.first_decode.mode != 2
            || presentation.first_decode.tail != 0xfd
            || presentation.second_decode.mode != 0
            || presentation.second_decode.tail != 0
        {
            return Err(game_err!("native AI 0x13fd4: raw decode tuple changed"));
        }
        if presentation.coordinate_call.address != 0x12D7B
            || presentation.sample_handle_expr != "[0x53EEC]"
            || presentation.sample_index != 4
            || presentation.sample_loop_count != 1
            || presentation.wait_ticks != [1, 1, 1]
        {
            return Err(game_err!("native AI 0x13fd4: raw presentation tuple changed"));
        }
        let (entry, ok) = st.units[actor].native_unit_layer_entry();
        let Some(cache) = st.native_map_selector_cache.as_ref().filter(|_| ok) else {
            return Err(game_err!("native AI 0x13fd4: actor lacks FDICON selector provenance"));
        };
        let roster =
            st.native_map_frame_roster().map_err(|e| GameError::from(e).wrap("native AI 0x13fd4: frame roster: "))?;
        let cycle =
            fdicon::native_frame_index(entry.motion_offset, entry.force_base, roster.cycles.idle, roster.cycles.moving)
                .map_err(|e| GameError::from(e).wrap("native AI 0x13fd4: FDICON cycle: "))?;
        // Go `g.nativeMapAssets.Units` 為 nil 時方法呼叫 panic；Rust 同樣以 expect 表達。
        let units = assets.units.as_ref().expect("Go：nativeMapAssets.Units 為 nil 時 panic");
        let sprite = units
            .sprite_for_native_slot(cache, entry.slot, entry.pose, cycle)
            .map_err(|e| GameError::from(e).wrap("native AI 0x13fd4: FDICON selector: "))?;
        let view = st.native_map_view_state;
        let offset = fdicon::native_placement_offset(
            entry.x,
            entry.y,
            view.camera_x,
            view.camera_y,
            entry.pose,
            entry.motion_offset,
            roster.unit_pixel_shift,
            entry.force_base,
        )
        .map_err(|e| GameError::from(e).wrap("native AI 0x13fd4: FDICON placement: "))?;
        if offset < 0 {
            return Err(game_err!("native AI 0x13fd4: FDICON placement is outside work buffer"));
        }
        let base_work = self.native_map_work.clone();
        let base_vga = self.native_map_vga.clone();
        if base_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || base_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
        {
            return Err(game_err!("native AI 0x13fd4: indexed buffers are incomplete"));
        }
        let stride = NATIVE_AI_IDLE_RECOVERY_STRIDE;
        let mut first_work = base_work.clone();
        // 4DDD7 uses arg8 as both stride and fill index; tailFD has no reader.
        sprite
            .blit_constant_mask_at(&mut first_work, stride, offset % stride, offset / stride, (stride & 0xff) as u8)
            .map_err(|e| GameError::from(e).wrap("native AI 0x13fd4: first 24x24 decode: "))?;
        let first_vga = native_ai_idle_recovery_viewport(&first_work, &base_vga)?;
        let mut second_work = base_work.clone();
        // 4DEDA restores raw pixels without the steady compositor's flag LUT.
        sprite
            .blit_for_native_flags_at_offset(&mut second_work, stride, offset, 0)
            .map_err(|e| GameError::from(e).wrap("native AI 0x13fd4: second 24x24 decode: "))?;
        let second_vga = native_ai_idle_recovery_viewport(&second_work, &base_vga)?;
        Ok(vec![base_vga, first_vga, second_vga])
    }
}

/// Go：`native_ai_idle_recovery.go:240`，`func nativeAIIdleRecoveryViewport(work, baseVGA []byte) ([]byte, error)`
pub(crate) fn native_ai_idle_recovery_viewport(work: &[u8], base_vga: &[u8]) -> Result<Vec<u8>> {
    if work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
        || base_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
    {
        return Err(game_err!("native AI 0x13fd4: viewport buffers are malformed"));
    }
    let mut vga = base_vga.to_vec();
    let src_end = NATIVE_AI_IDLE_RECOVERY_WORK_BASE
        + (NATIVE_AI_IDLE_RECOVERY_HEIGHT - 1) * NATIVE_AI_IDLE_RECOVERY_STRIDE
        + NATIVE_AI_IDLE_RECOVERY_WIDTH;
    let dst_end = NATIVE_AI_IDLE_RECOVERY_VGA_OFFSET
        + (NATIVE_AI_IDLE_RECOVERY_HEIGHT - 1) * NATIVE_AI_IDLE_RECOVERY_VGA_STRIDE
        + NATIVE_AI_IDLE_RECOVERY_WIDTH;
    if src_end > work.len() as i64 || dst_end > vga.len() as i64 {
        return Err(game_err!("native AI 0x13fd4: viewport copy exceeds indexed buffers"));
    }
    fdicon::copy_native_indexed_region(
        &mut vga[NATIVE_AI_IDLE_RECOVERY_VGA_OFFSET as usize..],
        NATIVE_AI_IDLE_RECOVERY_VGA_STRIDE,
        &work[NATIVE_AI_IDLE_RECOVERY_WORK_BASE as usize..],
        NATIVE_AI_IDLE_RECOVERY_STRIDE,
        NATIVE_AI_IDLE_RECOVERY_WIDTH,
        NATIVE_AI_IDLE_RECOVERY_HEIGHT,
    )
    .map_err(|e| GameError::from(e).wrap("native AI 0x13fd4: viewport copy: "))?;
    Ok(vga)
}

/// Go：`native_ai_idle_recovery.go:260`，`func osMuteOrShot(g *Game) bool`
pub(crate) fn os_mute_or_shot(g: Option<&Game>) -> bool {
    let Some(g) = g else {
        return true;
    };
    std::env::var("FD2_MUTE").is_ok_and(|v| !v.is_empty()) || !g.shot_path.is_empty()
}

/// Go：`native_ai_idle_recovery.go:264`，`func unitIndex(st *battle.State, actor *battle.Unit) int`
pub(crate) fn unit_index(st: Option<&battle::State>, actor: Option<UnitIdx>) -> i64 {
    // Go 以指標相等在 `st.Units` 裡找；Rust 的 `actor` 就是 `st.units` 的索引。
    match (st, actor) {
        (Some(st), Some(actor)) if actor < st.units.len() => actor as i64,
        _ => -1,
    }
}

impl Game {
    /// Go：`native_ai_idle_recovery.go:276`，`func (g *Game) stepNativeAIIdleRecovery()`
    pub(crate) fn step_native_ai_idle_recovery(&mut self) {
        let Some(job) = self.native_ai_idle_recovery.as_ref() else {
            return;
        };
        if !job.drawn {
            return;
        }
        // 0x1403D starts sample4 after the initial 0x1402B wait, before mode2.
        if job.frame == 0 && !os_mute_or_shot(Some(self)) {
            let sample = i64::from(job.plan.sample_index);
            self.play_sfx(sample);
        }
        let job = self.native_ai_idle_recovery.as_mut().unwrap();
        job.drawn = false;
        job.frame += 1;
        if job.frame < job.frames.len() as i64 {
            return;
        }
        let st = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic");
        let actor = job.actor;
        let idx = unit_index(Some(st), actor);
        let records = match battle::native_ai_scoring_records(&st.units) {
            Ok(records) => records,
            Err(err) => {
                self.fail_native_ai_idle_recovery_commit(err.message());
                return;
            }
        };
        let job = self.native_ai_idle_recovery.as_ref().unwrap();
        let actor_hp = actor.map(|a| st.units[a].hp).expect("Go：job.actor 為 nil 時解參考 panic");
        let checked = match battle::plan_native_ai_idle_recovery(&records, st.units.len() as i64, idx) {
            Err(err) => Err(err.message().to_string()),
            Ok(current) if current != job.decision || actor_hp != i64::from(job.plan.before_hp) => {
                Err(String::from("raw record changed during presentation"))
            }
            Ok(_) => Ok(()),
        };
        if let Err(message) = checked {
            self.fail_native_ai_idle_recovery_commit(&message);
            return;
        }
        let mut job = self.native_ai_idle_recovery.take().unwrap();
        let st = self.st.as_mut().unwrap();
        st.units[actor.unwrap()].hp = i64::from(job.plan.after_hp);
        st.native_map_range_mode = i64::from(job.plan.after_global_write.value);
        st.has_native_map_range_mode_state = true;
        if let Some(after) = job.after.take() {
            after(self);
        }
    }

    /// `stepNativeAIIdleRecovery` 兩條提交失敗路徑的共同收尾（Rust 拆出）：
    /// `restoreNativeAIIdleRecoveryRange`（`r8`，同檔 `:329`）、清 job、寫 `loadErr`、`aiBusy = false`。
    fn fail_native_ai_idle_recovery_commit(&mut self, message: &str) {
        if let Some(job) = self.native_ai_idle_recovery.take() {
            let st = self.st.as_mut().unwrap();
            st.native_map_range_mode = job.before_range;
            st.has_native_map_range_mode_state = job.before_has;
        }
        self.load_err = format!("native AI 0x13fd4 commit: {message}");
        self.ai_busy = false;
    }
}

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_ai_idle_recovery.go:328`，`func (g *Game) drawNativeAIIdleRecovery(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_ai_idle_recovery(&mut self, screen: &mut Picture) -> bool {
        todo!("r5:native_ai_idle_recovery.go:drawNativeAIIdleRecovery")
    }
}
