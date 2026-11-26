//! Go 來源：`remake/cmd/fd2/native_turn_staging.go`（505 行）。
//!
//! 骨架：Go 檔的 13 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_turn_staging.go:46`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeTurnStagingPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_TURN_STAGING_PAN: NativeTurnStagingPhase = NativeTurnStagingPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_TURN_STAGING_DELAY: NativeTurnStagingPhase = NativeTurnStagingPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_TURN_STAGING_FLASH: NativeTurnStagingPhase = NativeTurnStagingPhase(2);

/// nativeTurnStagingJob 擁有已閉合 handler 的阻塞式 0x35822 呼叫。raw DAC 與
/// indexed frame 分開保存；第一個鏡頭 tick 前先私下建立所有 roster 快照，避免
/// 後續 call 失敗時留下半套已物化 group。
///
/// Go：`native_turn_staging.go:57`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeTurnStagingJob {
    pub(crate) event: battle::NativeTurnEvent,
    pub(crate) states: Vec<battle::State>,
    pub(crate) call: i64,
    pub(crate) phase: NativeTurnStagingPhase,
    pub(crate) ticks: i64,
    pub(crate) vga: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) drawn: bool,
    pub(crate) indexed: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// Go：`native_turn_staging.go:115`，`func cloneNativeTurnStagingState(source *battle.State) (*battle.State, error)`
///
/// Go 淺複製 `*source` 後逐一深複製單位、名冊、選擇器快取與三個位元組陣列；Rust 的 `State::clone` 本來就是
/// 深複製（R2 所有權第 5 條），結果相同。簽名（C 組）：`source` 只讀，收 `Option<&State>`。
pub(crate) fn clone_native_turn_staging_state(source: Option<&battle::State>) -> Result<Option<battle::State>> {
    let Some(source) = source.filter(|s| {
        s.w > 0 && s.h > 0 && s.native_map_selector_cache.is_some() && s.native_map_selector_error.is_none()
    }) else {
        return Err(game_err!("event63 lacks a complete runtime roster/selector state"));
    };
    Ok(Some(source.clone()))
}

/// Go：`native_death_program_runtime.go:20`
pub(crate) const NATIVE_DEATH_STAGING_HANDLER: &str = "death:0x35822";

/// 以 0x35822 本體的常數建立一個死亡程式呼叫；常數與回合事件 63 的 helper 相同（同一支函式）。
///
/// Go：`native_death_program_runtime.go:185`，`func nativeDeathStagingEvent(id int, op battle.NativeDeathOp, source string) battle.NativeTurnEvent`
pub(crate) fn native_death_staging_event(id: i64, op: &battle::NativeDeathOp, source: &str) -> battle::NativeTurnEvent {
    battle::NativeTurnEvent {
        event_id: id,
        raw_camp: -1,
        handler: NATIVE_DEATH_STAGING_HANDLER.into(),
        staging: battle::NativeTurnStaging {
            helper: "0x35822".into(),
            pan_helper: "0x135dd".into(),
            spawn_helper: "0x10b4e".into(),
            delay_before_flash_ms: 300,
            palette_helper: "0x11df2".into(),
            palette_start: 0,
            palette_end: 255,
            flash_delta: 255,
            flash_hold_ms: 200,
            restore_delta: 0,
            redraw_helper: "0x11cac".into(),
            raw_placement_gate: 0,
            calls: vec![battle::NativeTurnStagingCall { group: op.group, x: op.x, y: op.y, source: source.into() }],
        },
        ..Default::default()
    }
}

/// Go：`native_death_program_runtime.go:197`，`func validateNativeDeathStaging(event battle.NativeTurnEvent) error`
pub(crate) fn validate_native_death_staging(event: &battle::NativeTurnEvent) -> Result<()> {
    let want = native_death_staging_event(event.event_id, &battle::NativeDeathOp::default(), "");
    let (s, w) = (&event.staging, &want.staging);
    if event.handler != NATIVE_DEATH_STAGING_HANDLER
        || s.helper != w.helper
        || s.pan_helper != w.pan_helper
        || s.spawn_helper != w.spawn_helper
        || s.delay_before_flash_ms != w.delay_before_flash_ms
        || s.palette_helper != w.palette_helper
        || s.palette_start != w.palette_start
        || s.palette_end != w.palette_end
        || s.flash_delta != w.flash_delta
        || s.flash_hold_ms != w.flash_hold_ms
        || s.restore_delta != w.restore_delta
        || s.redraw_helper != w.redraw_helper
        || s.raw_placement_gate != w.raw_placement_gate
        || s.calls.len() != 1
        || s.calls[0].source.is_empty()
    {
        return Err(game_err!("event{} death staging differs from the 0x35822 helper", event.event_id));
    }
    Ok(())
}

fn staging_signature_ok(s: &battle::NativeTurnStaging) -> bool {
    s.helper.eq_ignore_ascii_case("0x35822")
        && s.pan_helper.eq_ignore_ascii_case("0x135dd")
        && s.spawn_helper.eq_ignore_ascii_case("0x10b4e")
        && s.delay_before_flash_ms == 300
        && s.palette_helper.eq_ignore_ascii_case("0x11df2")
        && s.palette_start == 0
        && s.palette_end == 255
        && s.flash_delta == 255
        && s.flash_hold_ms == 200
        && s.restore_delta == 0
        && s.redraw_helper.eq_ignore_ascii_case("0x11cac")
        && s.raw_placement_gate == 0
}

/// Go：`native_turn_staging.go:72`，`func validateNativeEvent63(event battle.NativeTurnEvent) error`
pub(crate) fn validate_native_event63(event: &battle::NativeTurnEvent) -> Result<()> {
    let s = &event.staging;
    if event.event_id != 63
        || event.raw_camp != 0
        || !event.handler.eq_ignore_ascii_case("0x358c7")
        || !staging_signature_ok(s)
        || s.calls.len() != 2
    {
        return Err(GameError::new("event63 editable staging signature differs from recovered handler"));
    }
    let want = [(1, 3, 27, "0x358d7"), (2, 15, 27, "0x358e5")];
    for (i, call) in s.calls.iter().enumerate() {
        let (group, x, y, source) = want[i];
        if call.group != group || call.x != x || call.y != y || !call.source.eq_ignore_ascii_case(source) {
            return Err(game_err!("event63 staging call {i} differs from recovered handler"));
        }
    }
    Ok(())
}

/// Go：`native_turn_staging.go:95`，`func validateNativeEvent74(event battle.NativeTurnEvent) error`
pub(crate) fn validate_native_event74(event: &battle::NativeTurnEvent) -> Result<()> {
    let s = &event.staging;
    let dynamic_ok = event.dynamic_group.as_ref().is_some_and(|d| {
        d.state_index == 16
            && d.minimum == 4
            && d.maximum == 7
            && d.control_slot == 0
            && d.reschedule_delta == 1
            && d.stop_value == 7
            && d.increment == 1
    });
    if event.event_id != 74
        || event.raw_camp != 0
        || !event.handler.eq_ignore_ascii_case("0x35c32")
        || !dynamic_ok
        || !staging_signature_ok(s)
        || s.calls.len() != 1
    {
        return Err(GameError::new("event74 editable staging signature differs from recovered handler"));
    }
    let call = &s.calls[0];
    if call.group != -1 || call.x != 10 || call.y != 29 || !call.source.eq_ignore_ascii_case("0x35c4a") {
        return Err(GameError::new("event74 dynamic staging call differs from recovered handler"));
    }
    Ok(())
}

fn retains_pan_range(event_id: i64) -> bool {
    matches!(event_id, 66 | 68 | 70 | 72 | 74)
}

impl Game {
    /// 第 23 章 map22 事件 52 在第 13／22 回合的空群組呼叫：原版不追加單位。
    ///
    /// Go：`native_turn_staging.go:18`，`func (g *Game) nativeEvent52EmptyGroup(event battle.NativeTurnEvent, call battle.NativeTurnStagingCall) bool`
    pub(crate) fn native_event52_empty_group(
        &self,
        event: &battle::NativeTurnEvent,
        call: &battle::NativeTurnStagingCall,
    ) -> bool {
        let (Some(sc), Some(st)) = (self.sc.as_ref(), self.st.as_ref()) else {
            return false;
        };
        if sc.chapter != 23
            || sc.map != 22
            || event.event_id != 52
            || event.handler != NATIVE_DEATH_STAGING_HANDLER
            || event.staging.calls.len() != 1
            || call.y != 11
        {
            return false;
        }
        let round = st.native_round_counter;
        if round != 13 && round != 22 {
            return false;
        }
        let addend = if call.x == 2 && call.source.eq_ignore_ascii_case("0x352ec") {
            0
        } else if call.x == 26 && call.source.eq_ignore_ascii_case("0x35305") {
            1
        } else {
            return false;
        };
        if call.group != i64::from((2 * (round - 14) + addend) as u8) || st.roster.is_empty() {
            return false;
        }
        !st.roster.iter().any(|u| u.group == call.group)
    }

    /// 以候選狀態試合成一幀（不發布候選狀態、畫布或時鐘）。Go 以 `probe := *g` 淺複製整個 Game。
    pub(super) fn probe_native_map_frame(&mut self, state: battle::State, cam: Option<(f64, f64)>) -> Result<Vec<u8>> {
        let saved = self.save_native_game_preview(state);
        (self.native_map_work, self.native_map_vga) = (Vec::new(), Vec::new());
        if let Some((x, y)) = cam {
            (self.cam_x, self.cam_y) = (x, y);
        }
        let result = self.compose_native_map_frame().map(|_| self.native_map_vga.clone());
        self.restore_native_transient_preview(saved);
        result
    }

    /// 預檢全部 0x35822 呼叫：先私下建出每個呼叫後的名冊快照與畫面，失敗時不留下半套已物化群組。
    ///
    /// Go：`native_turn_staging.go:130`，`func (g *Game) preflightNativeTurnStaging(event battle.NativeTurnEvent) (battle.NativeTurnEvent, []*battle.State, []byte, bool, error)`
    pub(crate) fn preflight_native_turn_staging(
        &mut self,
        event: &battle::NativeTurnEvent,
    ) -> Result<(battle::NativeTurnEvent, Vec<battle::State>, Vec<u8>, bool)> {
        let mut resolved = event.clone();
        let (Some(st), Some(m)) = (self.st.as_ref(), self.m.as_ref()) else {
            return Err(game_err!("event{} map/state assets unavailable", event.event_id));
        };
        if self.tileset.is_none()
            || self.tiles.is_empty()
            || m.tile_w <= 0
            || m.tile_h <= 0
            || st.w != m.w
            || st.h != m.h
        {
            return Err(game_err!("event{} map/state assets unavailable", event.event_id));
        }
        let (map_w, map_h, tile_w, tile_h) = (m.w, m.h, m.tile_w, m.tile_h);
        match event.event_id {
            63 => validate_native_event63(event)?,
            74 => {
                validate_native_event74(event)?;
                let dynamic = event.dynamic_group.as_ref().unwrap();
                let group = i64::from(st.native_event_state[dynamic.state_index as usize]);
                if group < dynamic.minimum || group > dynamic.maximum {
                    return Err(game_err!("event74 dynamic group {group} is outside recovered range"));
                }
                resolved.staging.calls[0].group = group;
            }
            _ => {
                if event.handler != NATIVE_DEATH_STAGING_HANDLER {
                    return Err(game_err!("event{} has no typed staging adapter", event.event_id));
                }
                validate_native_death_staging(event)?;
            }
        }
        let base = clone_native_turn_staging_state(self.st.as_ref())?.unwrap();
        let mut baseline_frame = Vec::new();
        let mut indexed = false;
        if native_map_assets_available(self.native_map_assets.as_ref()) {
            let claims_indexed_state = base.has_native_map_view_state
                && base.has_native_map_range_mode_state
                && base.has_native_map_hud_state
                && base.has_native_map_cycle_state
                && base.native_map_selector_cache.is_some()
                && base.native_map_frame_roster().is_ok();
            let saved = self.save_native_game_preview(base.clone());
            (self.native_map_work, self.native_map_vga) = (Vec::new(), Vec::new());
            let (_, hud_ok) = self.native_map_hud_input();
            let result = self.compose_native_map_frame().map(|_| self.native_map_vga.clone());
            self.restore_native_transient_preview(saved);
            match result {
                Ok(frame) => (baseline_frame, indexed) = (frame, true),
                Err(err) if claims_indexed_state && hud_ok => {
                    return Err(game_err!("event{} claimed indexed baseline frame: {}", event.event_id, err.message()));
                }
                Err(_) => {}
            }
        }
        let mut candidate = clone_native_turn_staging_state(self.st.as_ref())?.unwrap();
        let mut states = Vec::with_capacity(resolved.staging.calls.len());
        for (i, call) in resolved.staging.calls.clone().iter().enumerate() {
            if call.x + 13 > map_w || call.y + 8 > map_h {
                return Err(game_err!("event{} staging call {i} camera exceeds native viewport", event.event_id));
            }
            if !self.native_event52_empty_group(&resolved, call) {
                match candidate
                    .append_group_with_native_placement(call.group, resolved.staging.raw_placement_gate as u8)
                {
                    Ok(n) if n > 0 => {}
                    result => {
                        return Err(game_err!(
                            "event{} staging call {i} group {}: append={:?}",
                            event.event_id,
                            call.group,
                            result.map_err(|e| e.to_string())
                        ));
                    }
                }
            }
            if event.event_id == 74 {
                let dynamic = event.dynamic_group.as_ref().unwrap();
                let slot = dynamic.control_slot as usize;
                let row = candidate.native_turn_event_controls[slot];
                let want = battle::NativeTurnEventControl {
                    turn: candidate.native_round_counter as u8,
                    event_id: 74,
                    raw_camp: 0,
                };
                if !candidate.has_native_turn_event_control_state || row != want {
                    return Err(GameError::new("event74 live row identity differs before commit"));
                }
                let offset = 3 + slot * 3;
                if candidate.has_native_field_control_state
                    && (candidate.native_field_control_raw.len() <= offset + 2
                        || candidate.native_field_control_raw[offset] != row.turn
                        || candidate.native_field_control_raw[offset + 1] != row.event_id
                        || candidate.native_field_control_raw[offset + 2] != row.raw_camp)
                {
                    return Err(GameError::new("event74 raw row disagrees with typed controls"));
                }
                if call.group != dynamic.stop_value {
                    let next = candidate.native_round_counter + dynamic.reschedule_delta;
                    if !(0..=0xfe).contains(&next) {
                        return Err(GameError::new("event74 rescheduled turn is outside byte range"));
                    }
                    candidate.native_turn_event_controls[slot].turn = next as u8;
                    if candidate.has_native_field_control_state {
                        candidate.native_field_control_raw[offset] = next as u8;
                    }
                }
                let index = dynamic.state_index as usize;
                candidate.native_event_state[index] =
                    candidate.native_event_state[index].wrapping_add(dynamic.increment as u8);
            }
            let snapshot = clone_native_turn_staging_state(Some(&candidate))?.unwrap();
            if indexed {
                let cam = ((call.x * tile_w) as f64, (call.y * tile_h) as f64);
                if let Err(err) = self.probe_native_map_frame(snapshot.clone(), Some(cam)) {
                    return Err(game_err!("event{} staging call {i} frame: {}", event.event_id, err.message()));
                }
            }
            states.push(snapshot);
        }
        Ok((resolved, states, baseline_frame, indexed))
    }

    /// 預檢全部 0x35822 呼叫後才開始第一個鏡頭 tick；回合事件與死亡事件共用同一個工作。
    ///
    /// Go：`native_turn_staging.go:316`，`func (g *Game) beginNativeStagingJob(event battle.NativeTurnEvent, then func()) error`
    pub(crate) fn begin_native_staging_job(
        &mut self,
        event: &battle::NativeTurnEvent,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<()> {
        let (resolved, states, frame, indexed) = self.preflight_native_turn_staging(event)?;
        let mut job = NativeTurnStagingJob {
            event: resolved,
            states,
            vga: frame,
            indexed,
            then: Some(then),
            ..Default::default()
        };
        if indexed {
            let a = self.native_map_assets.as_ref().unwrap();
            (job.dac, job.baseline) = (a.palette_dac.clone(), a.palette_dac.clone());
            job.palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&job.dac)?);
        }
        self.native_full_dac_white = false;
        self.native_turn_staging = Some(job);
        (self.sel, self.reach, self.moved) = (None, Default::default(), false);
        self.start_native_turn_staging_call();
        Ok(())
    }

    /// Go：`native_turn_staging.go:343`，`func (g *Game) startNativeTurnStagingCall()`
    pub(crate) fn start_native_turn_staging_call(&mut self) {
        let Some(job) = self.native_turn_staging.as_ref() else {
            self.fail_native_turn_staging("call index unavailable");
            return;
        };
        let Some(call) = usize::try_from(job.call).ok().and_then(|i| job.event.staging.calls.get(i)).cloned() else {
            self.fail_native_turn_staging("call index unavailable");
            return;
        };
        if retains_pan_range(job.event.event_id) {
            // 135F2在PAN第一步前清selector。第28／29章READY見各自主契約。
            let ok = self
                .st
                .as_mut()
                .is_some_and(|st| st.has_native_map_range_mode_state && st.materialize_native_map_range_mode(0));
            if !ok {
                self.fail_native_turn_staging("staging pan range source unavailable");
                return;
            }
        }
        let job = self.native_turn_staging.as_mut().unwrap();
        (job.phase, job.drawn) = (NATIVE_TURN_STAGING_PAN, false);
        let m = self.m.as_ref().unwrap();
        self.cam_pan = Some(CamPanJob {
            from_x: self.cam_x,
            from_y: self.cam_y,
            to_x: (call.x * m.tile_w) as f64,
            to_y: (call.y * m.tile_h) as f64,
            tile_step: true,
            then: Some(Box::new(|g: &mut Game| g.commit_native_turn_staging_spawn())),
            ..Default::default()
        });
    }

    /// 平移最後一幀先拍下再發布新群組；0x10B4E 與白閃之間不呼叫 0x11CAC。
    ///
    /// Go：`native_turn_staging.go:366`，`func (g *Game) commitNativeTurnStagingSpawn()`
    pub(crate) fn commit_native_turn_staging_spawn(&mut self) {
        let Some(job) = self.native_turn_staging.as_ref() else {
            self.fail_native_turn_staging("preflighted spawn unavailable");
            return;
        };
        let Some(call) = usize::try_from(job.call).ok().filter(|&i| i < job.states.len()) else {
            self.fail_native_turn_staging("preflighted spawn unavailable");
            return;
        };
        let (indexed, event_id) = (job.indexed, job.event.event_id);
        if indexed {
            if let Err(err) = self.compose_native_map_frame() {
                self.fail_native_turn_staging(&format!("final pan frame: {}", err.message()));
                return;
            }
            let vga = self.native_map_vga.clone();
            go_copy(&mut self.native_turn_staging.as_mut().unwrap().vga, &vga);
        }
        let mut candidate = self.native_turn_staging.as_ref().unwrap().states[call].clone();
        // PAN已透過battleEvent owner發布六全域；roster預檢不能回滾視圖。
        // #141、#226、#228 READY分別見第23／28／29章主契約。
        let retains = retains_pan_range(event_id);
        if event_id == 52 || retains {
            let st = self.st.as_ref().unwrap();
            if !st.has_native_map_view_state {
                self.fail_native_turn_staging("completed pan has no native view");
                return;
            }
            let (view, range_ok, range) =
                (st.native_map_view_state, st.has_native_map_range_mode_state, st.native_map_range_mode);
            if let Err(err) = candidate.materialize_native_map_view_state(view) {
                self.fail_native_turn_staging(&format!("completed pan view: {err}"));
                return;
            }
            if retains && (!range_ok || !candidate.materialize_native_map_range_mode(range)) {
                self.fail_native_turn_staging("staging completed pan range unavailable");
                return;
            }
        }
        self.st = Some(candidate);
        let job = self.native_turn_staging.as_mut().unwrap();
        job.phase = NATIVE_TURN_STAGING_DELAY;
        job.ticks = native_delay_ticks(job.event.staging.delay_before_flash_ms);
    }

    /// 原版索引色分支：PAN 階段每幀重合成，之後畫保存的 VGA 與目前 DAC。
    ///
    /// Go：`native_turn_staging.go:486`，`func (g *Game) drawNativeTurnStaging(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_turn_staging(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(job) = self.native_turn_staging.as_ref() else {
            return false;
        };
        if job.vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE || job.palette.len() != 256 {
            return false;
        }
        if job.phase == NATIVE_TURN_STAGING_PAN {
            if let Err(err) = self.compose_native_map_frame() {
                self.fail_native_turn_staging(&format!("pan redraw: {}", err.message()));
                return false;
            }
            let vga = self.native_map_vga.clone();
            go_copy(&mut self.native_turn_staging.as_mut().unwrap().vga, &vga);
        }
        let job = self.native_turn_staging.as_mut().unwrap();
        if let Some(screen) = screen {
            let img = indexed_frame(320, 200, &job.vga, job.palette.clone());
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(&Picture::from_indexed(&img), &options);
        }
        job.drawn = true;
        true
    }
    /// startNativeRawCamp0TurnEvents is the sub_1A813(0) owner. Scenarios opt in by
    /// supplying native_turn_events; once opted in, every matching live row must
    /// have exactly one typed consumer or the enemy AI remains stopped.
    ///
    /// Go：`native_turn_staging.go:253`，`func (g *Game) startNativeRawCamp0TurnEvents() (bool, error)`
    pub(crate) fn start_native_raw_camp0_turn_events(&mut self) -> Result<bool> {
        let Some(sc) = self.sc.as_ref().filter(|sc| !sc.native_turn_events.is_empty()) else {
            return Ok(false);
        };
        let events = sc.native_turn_events_at(self.st.as_ref(), 0)?;
        // 帶轉寫動作的控制列事件由 runEditableTurnEvents(0) 在橫幅之後執行。
        let events: Vec<battle::NativeTurnEvent> =
            events.into_iter().filter(|event| event.actions.is_empty()).collect();
        if events.is_empty() {
            return Ok(false);
        }
        if events.len() != 1 {
            return Err(game_err!("raw camp0 has {} simultaneous handlers; staging is sequential", events.len()));
        }
        if events[0].pair_mutation.is_some() {
            // 79交給既有aiStep於13536之後處理；目前還沒清bit7。
            return Ok(false);
        }
        self.begin_native_staging_job(&events[0], Box::new(|g: &mut Game| g.begin_enemy_phase_after_turn_events()))?;
        Ok(true)
    }

    /// READY: fd2_ch29_flow_20261017.json event79_phase_contract。
    ///
    /// Go：`native_turn_staging.go:286`，`func (g *Game) applyNativeRawCamp0PairEvent() error`
    pub(crate) fn apply_native_raw_camp0_pair_event(&mut self) -> Result<()> {
        let Some(sc) = self.sc.as_ref().filter(|sc| !sc.native_turn_events.is_empty()) else {
            return Ok(());
        };
        if self.st.is_none() {
            return Ok(());
        }
        let events = sc.native_turn_events_at(self.st.as_ref(), 0)?;
        for event in events {
            if event.pair_mutation.is_none() {
                continue;
            }
            let mut seed = self.native_rng_state;
            if let Some(mut input) = self.native_turn_pair_rng_input.take() {
                let result = input(self, event.clone(), seed);
                self.native_turn_pair_rng_input = Some(input);
                seed = result?;
            }
            let (next, _) = battle::apply_native_turn_event79(self.st.as_mut(), &event, seed)?;
            self.native_rng_state = next;
        }
        Ok(())
    }

    /// Go：`native_turn_staging.go:404`，`func (g *Game) stepNativeTurnStaging()`
    pub(crate) fn step_native_turn_staging(&mut self) {
        let Some(job) = self.native_turn_staging.as_mut() else {
            return;
        };
        match job.phase {
            NATIVE_TURN_STAGING_PAN => {}
            NATIVE_TURN_STAGING_DELAY => {
                if job.ticks > 0 {
                    job.ticks -= 1;
                    return;
                }
                if job.indexed {
                    let s = &job.event.staging;
                    if let Err(err) = fdother::apply_vga_palette_delta(
                        &mut job.dac,
                        &job.baseline,
                        s.palette_start,
                        s.palette_end,
                        s.flash_delta,
                    ) {
                        self.fail_native_turn_staging(&format!("white flash: {err}"));
                        return;
                    }
                    match fdother::vga_palette_from_dac(&job.dac) {
                        Ok(palette) => job.palette = color_palette_from_nrgba(&palette),
                        Err(err) => {
                            self.fail_native_turn_staging(&format!("white palette: {err}"));
                            return;
                        }
                    }
                } else {
                    self.native_full_dac_white = true;
                }
                let job = self.native_turn_staging.as_mut().unwrap();
                (job.phase, job.ticks, job.drawn) =
                    (NATIVE_TURN_STAGING_FLASH, native_delay_ticks(job.event.staging.flash_hold_ms), false);
            }
            NATIVE_TURN_STAGING_FLASH => {
                // Go 的 `drawn` 由 drawNativeTurnStaging（Draw）設；Rust 由呈現回報設（見 PORTING.md）。
                if !job.drawn {
                    return;
                }
                if job.ticks > 0 {
                    job.ticks -= 1;
                    return;
                }
                if job.indexed {
                    let s = &job.event.staging;
                    if let Err(err) = fdother::apply_vga_palette_delta(
                        &mut job.dac,
                        &job.baseline,
                        s.palette_start,
                        s.palette_end,
                        s.restore_delta,
                    ) {
                        self.fail_native_turn_staging(&format!("palette restore: {err}"));
                        return;
                    }
                    match fdother::vga_palette_from_dac(&job.dac) {
                        Ok(palette) => job.palette = color_palette_from_nrgba(&palette),
                        Err(err) => {
                            self.fail_native_turn_staging(&format!("restored palette: {err}"));
                            return;
                        }
                    }
                    if let Err(err) = self.compose_native_map_frame() {
                        self.fail_native_turn_staging(&format!("redraw: {}", err.message()));
                        return;
                    }
                    let vga = self.native_map_vga.clone();
                    let job = self.native_turn_staging.as_mut().unwrap();
                    go_copy(&mut job.vga, &vga);
                } else {
                    self.native_full_dac_white = false;
                }
                let job = self.native_turn_staging.as_mut().unwrap();
                job.call += 1;
                if job.call < job.event.staging.calls.len() as i64 {
                    self.start_native_turn_staging_call();
                    return;
                }
                let then = job.then.take();
                self.native_full_dac_white = false;
                self.native_turn_staging = None;
                if let Some(then) = then {
                    then(self);
                }
            }
            _ => {}
        }
    }

    /// Go：`native_turn_staging.go:470`，`func (g *Game) failNativeTurnStaging(message string)`（`r8`，隨
    /// `stepNativeTurnStaging` 補移植）。
    pub(crate) fn fail_native_turn_staging(&mut self, message: &str) {
        self.load_err = format!("native turn staging: {message}");
        self.native_full_dac_white = false;
        self.cam_pan = None;
        self.native_turn_staging = None;
    }
}
