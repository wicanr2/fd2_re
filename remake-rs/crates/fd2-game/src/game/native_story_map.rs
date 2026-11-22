//! Go 來源：`remake/cmd/fd2/native_story_map.go`（182 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 6 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! R4 B 組：Go 的場景載體 `storyNativeMapState.Units[i]` 是 `&g.storyActors[i]`（指標共用）。Rust 的
//! `battle::State` 擁有單位，所以載體的 `units` 是 `story_actors` 的複本；本檔讀載體單位前先以
//! [`Game::sync_story_native_map_units`] 對齊，長度在建立與 append 時與 `story_actors` 相同。

use super::*;

impl Game {
    /// 第十二章的原生聚焦沿 0x12CEA 重繪，不能只發布六個視圖全域。
    /// 已證實範圍見 fd2_terrain_mode3_review_20261001.json story_frame_continuity。
    ///
    /// Go：`native_story_map.go:13`，`func (g *Game) nativeCh12StoryFocus() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ch12_story_focus(&self) -> bool {
        self.camp.as_ref().is_some_and(|c| c.node_id() == "story_ch12")
            && self.native_map_assets.as_ref().is_some_and(|a| a.map_index == 11)
            && self.st.is_none()
            && self.has_story_native_map_view
            && self.story_native_map_state.is_some()
    }

    /// Go：`native_story_map.go:19`，`func (g *Game) nativeCh12StoryWorkHandoff(unitsPath, scenarioPath string) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ch12_story_work_handoff(&self, units_path: &str, scenario_path: &str) -> bool {
        self.camp.as_ref().is_some_and(|c| c.node_id() == "battle_ch12")
            && self.native_map_assets.as_ref().is_some_and(|a| a.map_index == 11)
            && self.has_story_native_map_view
            && self.story_native_map_state.is_some()
            && !self.story_actors.is_empty()
            && !units_path.is_empty()
            && !scenario_path.is_empty()
            && self.story_roster_path == units_path
            && self.story_party_scenario == scenario_path
            && self.native_map_work.len() as i64 == indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            && self.native_map_vga.len() as i64 == indexedmap::NATIVE_MAP_VGA_SIZE
    }

    /// materializeNativeStoryMapState 建立 LOADCH 場景專用的索引地圖載體。
    /// 場景沿用原始 terrain/unit constructor 與 raw view，但明確不顯示戰鬥 HUD；
    /// 它不會把 cutscene 單位升格成可操作的 battle.State。
    ///
    /// Go：`native_story_map.go:32`，`func (g *Game) materializeNativeStoryMapState(source *battle.State) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn materialize_native_story_map_state(&mut self, source: Option<&battle::State>) -> Result<()> {
        let (Some(source), Some(m)) = (source, self.m.as_ref()) else {
            return Err(GameError::new("native story map: source, field, or raw view is unavailable"));
        };
        if !self.has_story_native_map_view {
            return Err(GameError::new("native story map: source, field, or raw view is unavailable"));
        }
        if m.w <= 0 || m.h <= 0 || m.tiles.len() as i64 != m.w * m.h || m.native_tile_blit_modes.len() != m.tiles.len()
        {
            return Err(GameError::new("native story map: LOADCH terrain renderer inputs are incomplete"));
        }
        let mut candidate = source.clone();
        // 地形尺寸屬於目前 LOADCH；素材中的 cell +3 只作來源證據。
        // 原版 0x4DBFC 在載入後逐格寫入 0xff，與 battle.LoadScenario 相同。
        // 複製 serialized 值會讓整個故事背景誤走 0x11EEE 的亮色 LUT 分支。
        (candidate.w, candidate.h) = (m.w, m.h);
        candidate.native_tile_blit_modes = vec![0xff; m.native_tile_blit_modes.len()];
        candidate.units = Vec::new();
        candidate.roster = Vec::new();
        candidate.native_map_selector_cache = None;
        candidate.native_map_selector_error = None;
        candidate.has_native_map_cycle_state = false;
        candidate.has_native_terrain_phase_state = false;
        candidate.has_native_map_binary_timing_state = false;
        // 快取重建不是再次建構角色；constructor 會把 pose/motion 清零。
        // 私有複本只用來取得 slot，不覆蓋已完成的 ACT／scroll_step。
        candidate.append_native_map_selector_batch(self.story_actors.clone())?;
        for index in 0..self.story_actors.len() {
            let original = &self.story_actors[index];
            if !original.has_native_map_presentation
                && !candidate.units[index].set_map_placement(original.x, original.y, original.dir)
            {
                return Err(GameError::new("native story map: spawned actor placement is invalid"));
            }
        }
        let view = self.story_native_map_view;
        if let Err(err) = candidate.materialize_native_map_view_state(view) {
            return Err(game_err!("native story map state view {}: {err}", go_native_map_view_plus_v(view)));
        }
        if !candidate.materialize_native_map_range_mode(0) {
            return Err(GameError::new("native story map: opening selector 0 is unavailable"));
        }
        // 場景視窗不畫戰鬥 HUD；anchor 仍保留資料映像的合法初值，避免把
        // 隱藏狀態誤解成不存在的版面資料。
        if !candidate.materialize_native_map_hud_state(0, 0, 1) {
            return Err(GameError::new("native story map: hidden HUD state is invalid"));
        }
        for index in 0..self.story_actors.len() {
            let actor = &candidate.units[index];
            let original = &mut self.story_actors[index];
            if !original.has_native_map_presentation {
                original.native_map_presentation = actor.native_map_presentation;
                original.has_native_map_presentation = true;
            }
            original.map_selector_slot = actor.map_selector_slot;
            original.has_map_selector_slot = actor.has_map_selector_slot;
        }
        // Go：candidate.Units[index] = &g.storyActors[index]
        candidate.units = self.story_actors.clone();
        self.story_native_map_state = Some(candidate);
        Ok(())
    }

    /// appendNativeStoryMapActors 在 SPAWN 後延續同一 first-seen selector cache。
    /// storyActors append 可能搬移底層陣列，所以成功後一併重建全部 unit pointers。
    ///
    /// Go：`native_story_map.go:99`，`func (g *Game) appendNativeStoryMapActors(start int) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn append_native_story_map_actors(&mut self, start: i64) -> Result<()> {
        let Some(state) = self.story_native_map_state.as_mut() else {
            return Err(GameError::new("native story map: append boundary is unavailable"));
        };
        append_native_story_map_actors_into(state, &mut self.story_actors, start)
    }

    /// R4 B 組新增：Go 載體的 `Units` 是指向 `g.storyActors` 的指標，Rust 存複本；讀取載體單位前呼叫，
    /// 讓複本等於目前的劇情角色（ACT、走位、姿態之後的狀態）。
    pub(crate) fn sync_story_native_map_units(&mut self) {
        if let Some(state) = self.story_native_map_state.as_mut() {
            state.units = self.story_actors.clone();
        }
    }

    /// 出場預檢只改候選角色與私有載體。成功後須把兩者一起發布，否則
    /// 原生底圖及HD移動會讀到出場前的角色名冊。
    ///
    /// Go：`native_story_map.go:119`，`func (g *Game) prepareNativeStoryMapSpawnState(actors []battle.Unit, oldCount int) (*battle.State, error)`
    ///
    /// R4 B 組：`actors` 改收 `&mut`。Go 以 `view := *g; view.storyActors = actors` 共用呼叫端的底層陣列，
    /// `appendNativeStoryMapActors` 寫入的 selector slot 留在呼叫端的 `actors` 上；Rust 不複製整個 `Game`，
    /// 直接對候選載體與 `actors` 做同一件事。
    pub(crate) fn prepare_native_story_map_spawn_state(
        &self,
        actors: &mut [battle::Unit],
        old_count: i64,
    ) -> Result<Option<battle::State>> {
        let Some(story_state) = self.story_native_map_state.as_ref() else {
            return Ok(None);
        };
        if old_count != self.story_actors.len() as i64
            || old_count > actors.len() as i64
            || story_state.units.len() as i64 != old_count
            || story_state.native_map_selector_cache.is_none()
        {
            return Err(GameError::new("native story spawn: map roster boundary unavailable"));
        }
        // Go：state := *g.storyNativeMapState；selector cache Clone、Units 另起一份指標切片。
        let mut state = story_state.clone();
        append_native_story_map_actors_into(&mut state, actors, old_count)?;
        Ok(Some(state))
    }

    /// Go：`native_story_map.go:138`，`func (g *Game) composeNativeStoryMapFrame() error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// R4 B 組：Go 暫時把 `g.st` 換成私有載體再呼叫 `composeNativeMapFrame`；Rust 以 `Option::replace`
    /// 換入、合成後換回（呼叫前 `g.st` 必為 nil）。
    pub(crate) fn compose_native_story_map_frame(&mut self) -> Result<()> {
        if self.st.is_some() || self.story_native_map_state.is_none() || !self.has_story_native_map_view {
            return Err(GameError::new("native story map: caller-owned scene state is unavailable"));
        }
        self.sync_story_native_map_units();
        let view = self.story_native_map_view;
        let state = self.story_native_map_state.as_mut().unwrap();
        if let Err(err) = state.materialize_native_map_view_state(view) {
            return Err(game_err!("native story map frame view {}: {err}", go_native_map_view_plus_v(view)));
        }
        // map32 前兩筆特殊劇情角色不在目前閉合的 constructor table 範圍，
        // 因而沒有 0x129EC gate 的 raw race/class。99% 玩家可見模式只在本次
        // 背景合成的私有 clone 採保守前景重畫；不可污染 storyActors、戰鬥或存檔。
        let mut frame_state = state.clone();
        approximate_story_foreground(&mut frame_state.units)?;
        let previous = self.st.replace(frame_state);
        let err = self.compose_native_map_frame();
        // composeNativeMapFrame 會更新時序全域值，但私有角色clone只供畫面使用；
        // 只發布時序，不以近似角色取代正式場景角色。
        let frame = std::mem::replace(&mut self.st, previous).expect("composeNativeMapFrame 不會清掉 g.st");
        let state = self.story_native_map_state.as_mut().unwrap();
        state.native_map_cycle_state = frame.native_map_cycle_state;
        state.has_native_map_cycle_state = frame.has_native_map_cycle_state;
        state.native_terrain_phase_state = frame.native_terrain_phase_state;
        state.has_native_terrain_phase_state = frame.has_native_terrain_phase_state;
        state.native_terrain_flip_state = frame.native_terrain_flip_state;
        state.native_unit_pixel_shift_state = frame.native_unit_pixel_shift_state;
        state.has_native_map_binary_timing_state = frame.has_native_map_binary_timing_state;
        err
    }

    pub(crate) fn has_hd_story_map_source(&self) -> bool {
        self.st.is_none()
            && self.story_bg
            && self.has_story_native_map_view
            && self.story_native_map_state.is_some()
            && native_map_assets_available(self.native_map_assets.as_ref())
            && self.m.as_ref().is_some_and(|map| (map.tile_w, map.tile_h) == (24, 24))
    }

    /// 沿原生場景載體準備 HD 圖層。繪製取樣只更新私有候選。
    pub(crate) fn prepare_hd_story_map(&self) -> Result<ModernHDMapFrame> {
        let assets = self.native_map_assets.as_ref().ok_or_else(|| game_err!("modern HD story: missing assets"))?;
        let mut state = self
            .story_native_map_state
            .as_ref()
            .filter(|state| state.native_map_selector_cache.is_some())
            .cloned()
            .ok_or_else(|| game_err!("modern HD story: selector cache absent"))?;
        state.units.clone_from(&self.story_actors);
        approximate_story_foreground(&mut state.units)?;
        let map = self.m.as_ref().ok_or_else(|| game_err!("modern HD story: missing field"))?;
        let mut view = self.story_native_map_view;
        let moving = !self.story_walks.is_empty()
            || self.cam_x != (view.camera_x * map.tile_w) as f64
            || self.cam_y != (view.camera_y * map.tile_h) as f64;
        if moving {
            if !(0.0..=((map.w - 13) * 24) as f64).contains(&self.cam_x)
                || !(0.0..=((map.h - 8) * 24) as f64).contains(&self.cam_y)
            {
                return Err(game_err!("modern HD story: camera outside drawable map"));
            }
            view.camera_x = (self.cam_x / 24.0).floor() as i64;
            view.camera_y = (self.cam_y / 24.0).floor() as i64;
        }
        state.materialize_native_map_view_state(view)?;
        let mut player_clock = self.player_clock;
        let mut map_clock = self.native_map_clock;
        if !sample_native_map_timing(&mut state, &mut map_clock, player_clock.sample(self.virtual_now)) {
            return Err(game_err!("modern HD story: native map timing unavailable"));
        }
        let (raw_tick, sampled) = map_clock.current();
        if !sampled {
            return Err(game_err!("modern HD story: auxiliary timing unavailable"));
        }
        let mut dac =
            if self.native_map_dac.is_empty() { assets.palette_dac.clone() } else { self.native_map_dac.clone() };
        fdother::advance_native_dac_palette_cycle_e0ef(
            &mut dac,
            self.native_fdother_palette_phase,
            self.native_fdother_palette_tick,
            raw_tick,
        )?;
        let runtime = NativeMapFrameRuntime {
            hud: indexedmap::NativeMapHUDInput::default(),
            chapter_aux_phase: self.native_chapter_aux_phase_override.unwrap_or(raw_tick) & 15,
        };
        let mut input = build_native_map_frame_input(Some(assets), self.m.as_ref(), Some(&state), runtime)?;
        // LOADCH 的劇情畫面只顯示世界。原生對白稍後在獨立介面上畫。
        input.hud.display_gate_a = false;
        input.hud.display_gate_b = false;
        input.frame.range_mode = 0;
        let stage = self.prepare_native_ch23_steady_stage(raw_tick)?;
        let motion =
            if self.banner_t > 0 || self.walk.is_some() { ModernHDMapMotion::Step } else { ModernHDMapMotion::Steady };
        self.prepare_hd_map(&input, stage.as_ref(), &dac, motion, moving)
    }
}

fn approximate_story_foreground(units: &mut [battle::Unit]) -> Result<()> {
    for unit in units {
        if unit.has_native_record_race && unit.has_native_record_class {
            continue;
        }
        if !unit.has_native_map_presentation || !unit.has_native_record_byte5 || !unit.has_battle_fig {
            return Err(game_err!("native story map: approximate foreground gate lacks required provenance"));
        }
        if !unit.has_native_record_race {
            (unit.native_record_race, unit.has_native_record_race) = (0, true);
        }
        if !unit.has_native_record_class {
            (unit.native_record_class, unit.has_native_record_class) = (0, true);
        }
    }
    Ok(())
}

/// R4 B 組新增：`appendNativeStoryMapActors` 對載體與角色陣列的部分；`prepareNativeStoryMapSpawnState`
/// 對候選資料做同一件事。Go `native_story_map.go:99`。
fn append_native_story_map_actors_into(
    state: &mut battle::State,
    actors: &mut [battle::Unit],
    start: i64,
) -> Result<()> {
    if start < 0 || start > actors.len() as i64 {
        return Err(GameError::new("native story map: append boundary is unavailable"));
    }
    battle::materialize_native_map_selector_slots(
        &mut actors[start as usize..],
        state.native_map_selector_cache.as_mut(),
    )?;
    // Go：Units 重建成指向全部角色的指標。
    state.units = actors.to_vec();
    Ok(())
}

/// R4 B 組新增：Go `fmt` 的 `%+v` 印 `battle.NativeMapViewState`（欄位順序同 Go 宣告）。
pub(crate) fn go_native_map_view_plus_v(v: battle::NativeMapViewState) -> String {
    format!(
        "{{CameraX:{} CameraY:{} CursorX:{} CursorY:{} VisibleCursorX:{} VisibleCursorY:{}}}",
        v.camera_x, v.camera_y, v.cursor_x, v.cursor_y, v.visible_cursor_x, v.visible_cursor_y
    )
}
