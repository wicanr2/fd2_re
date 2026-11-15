//! Go 來源：`remake/cmd/fd2/native_spawn_intro.go`（372 行）。
//!
//! 骨架：Go 檔的 13 個函式中產生 12 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! R4 B 組：12 個函式移植完；R5 階段 2（r5m）移植 `drawNativeSpawnIntro`（畫格與 `drawn = true`）。無畫面重播不畫像素時，
//! `drawn` 仍由呈現回報設定。

use super::*;

/// nativeSpawnIntroJob owns sub_32999's twelve observable presentation passes.
/// The following 0x1366A call remains the next independent compiled Beat.
///
/// Go：`native_spawn_intro.go:19`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeSpawnIntroJob {
    pub(crate) schedule: Vec<fdother::NativeSpawnIntroStep>,
    pub(crate) work: Vec<u8>,
    pub(crate) vga: Vec<u8>,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) palette: ColorPalette,
    pub(crate) sound: Vec<u8>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// Go：`native_spawn_intro.go:31`，`func cloneStoryUnitPointers(source []battle.Unit) []*battle.Unit`
///
/// R4 B 組：Go 複製一份單位值再取指標；Rust 的單位是值，回傳複本（骨架的 `Vec<UnitIdx>` 指不到這份新陣列）。
pub(crate) fn clone_story_unit_pointers(source: &[battle::Unit]) -> Vec<battle::Unit> {
    source.to_vec()
}

/// Go：`native_spawn_intro.go:40`，`func cloneBattleUnitPointers(source []*battle.Unit) []*battle.Unit`
///
/// R4 B 組：Go 把每個指標指的單位複製一份；Rust 回傳單位複本（骨架的 `&[UnitIdx] -> Vec<UnitIdx>` 不會複製單位）。
/// Go 的 nil 元素保持 nil，Rust 的單位陣列沒有 nil。
pub(crate) fn clone_battle_unit_pointers(source: &[battle::Unit]) -> Vec<battle::Unit> {
    source.to_vec()
}

/// Go：`native_spawn_intro.go:52`，`func cloneNativeBattleSpawnState(source *battle.State) (*battle.State, error)`
///
/// Go 逐欄複製 Units、Roster、選擇器快取與組成位元組；Rust `State::clone` 是深複本，結果相同。
pub(crate) fn clone_native_battle_spawn_state(source: Option<&mut battle::State>) -> Result<Option<battle::State>> {
    let Some(source) = source.filter(|s| {
        s.w > 0
            && s.h > 0
            && !s.roster.is_empty()
            && s.native_map_selector_cache.is_some()
            && s.native_map_selector_error.is_none()
    }) else {
        return Err(GameError::new("native battle spawn lacks a complete runtime roster/selector state"));
    };
    Ok(Some(source.clone()))
}

impl Game {
    /// Go：`native_spawn_intro.go:65`，`func (g *Game) bindNativeFutureItemRows(state *battle.State) error`
    pub(crate) fn bind_native_future_item_rows(&self, state: Option<&mut battle::State>) -> Result<()> {
        let Some(state) = state else {
            return Err(GameError::new("native future constructor state unavailable"));
        };
        let rows = battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))?;
        Ok(state.bind_native_future_item_rows(&rows)?)
    }

    /// prepareNativeStorySpawn executes the proven future-group placement against
    /// private clones. The caller commits the returned values only after every
    /// visual pass has preflighted successfully.
    ///
    /// Go：`native_spawn_intro.go:81`，`func (g *Game) prepareNativeStorySpawn(group int, rawGate byte) ([]battle.Unit, []battle.Unit, error)`
    pub(crate) fn prepare_native_story_spawn(
        &self,
        group: i64,
        raw_gate: u8,
    ) -> Result<(Vec<battle::Unit>, Vec<battle::Unit>)> {
        let field_ok = self
            .m
            .as_ref()
            .is_some_and(|m| m.w > 0 && m.h > 0 && self.story_composition_event_bytes.len() as i64 == m.w * m.h);
        // Go 還檢查 `g.storySpawned == nil`；Rust 的 map 不分 nil 與空。
        if !field_ok || self.story_roster.is_empty() || self.story_spawned.get(&group).copied().unwrap_or(false) {
            return Err(game_err!("native story spawn group {} lacks unmaterialized FDFIELD state", group));
        }
        let m = self.m.as_ref().unwrap();
        let active = clone_story_unit_pointers(&self.story_actors);
        let roster = clone_story_unit_pointers(&self.story_roster);
        let mut state = battle::State::default();
        (state.w, state.h, state.roster) = (m.w, m.h, roster);
        state.native_composition_event_bytes = self.story_composition_event_bytes.clone();
        self.bind_native_future_item_rows(Some(&mut state))
            .map_err(|err| game_err!("native story spawn item rows: {err}"))?;
        state
            .append_native_map_selector_batch(active)
            .map_err(|err| game_err!("native story spawn existing roster: {err}"))?;
        let n = state.append_group_with_native_placement(group, raw_gate)?;
        if n <= 0 {
            return Err(game_err!("native story spawn group {} appended no units", group));
        }
        Ok((state.units, state.roster))
    }

    /// Go：`native_spawn_intro.go:114`，`func (g *Game) buildNativeSpawnIntroJob( oldInput, candidateInput indexedmap.NativeTransitionFrameInput, oldCount int, then func(), ) (*nativeSpawnIntroJob, error)`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn build_native_spawn_intro_job(
        &self,
        old_input: &indexedmap::NativeTransitionFrameInput<'_>,
        candidate_input: &indexedmap::NativeTransitionFrameInput<'_>,
        old_count: i64,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<Option<NativeSpawnIntroJob>> {
        if self.spawn_intro_transition.is_some() || self.indexed_transition.is_some() {
            return Err(GameError::new("native spawn-intro transition already active"));
        }
        let Some(a) = self
            .native_map_assets
            .as_ref()
            .filter(|a| native_spawn_intro_assets_available(a) && !self.sfx_spawn_intro.is_empty())
        else {
            return Err(GameError::new("native spawn-intro visual/audio assets unavailable"));
        };
        if old_count < 0 || old_count > candidate_input.units.len() as i64 || old_input.units.len() as i64 != old_count
        {
            return Err(GameError::new("native spawn-intro old/new roster boundary unavailable"));
        }

        let mut work = vec![0u8; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
        indexedmap::compose_native_unit_present_terrain_snapshot(&mut work, old_input)
            .map_err(|err| game_err!("native spawn-intro baseline terrain: {err}"))?;
        indexedmap::redraw_native_unit_present_objects(&mut work, old_input)
            .map_err(|err| game_err!("native spawn-intro baseline objects: {err}"))?;
        let mut snapshot = indexedmap::seed_native_spawn_intro_snapshot(&work)?;
        let mut vga = vec![0u8; indexedmap::NATIVE_MAP_VGA_SIZE as usize];
        let schedule = fdother::native_spawn_intro_schedule();
        let input =
            indexedmap::NativeSpawnIntroFrameInput { frame: candidate_input.clone(), old_unit_count: old_count };
        let mut frames: Vec<Vec<u8>> = Vec::with_capacity(schedule.len());
        for step in &schedule {
            let entry = a.spawn_intro.get(step.lmi_entry as usize).expect("Go：LMI 索引越界 panic");
            indexedmap::compose_native_spawn_intro_pass(&mut work, &mut vga, &mut snapshot, &input, entry, *step)
                .map_err(|err| game_err!("native spawn-intro preflight pass {}: {err}", step.pass))?;
            frames.push(vga.clone());
        }
        if frames.len() as i64 != fdother::NATIVE_SPAWN_INTRO_PASS_COUNT {
            return Err(GameError::new("native spawn-intro preflight did not produce twelve frames"));
        }
        Ok(Some(NativeSpawnIntroJob {
            schedule,
            work,
            vga: frames[0].clone(),
            frames,
            palette: a.palette.clone(),
            sound: self.sfx_spawn_intro.clone(),
            then,
            ..Default::default()
        }))
    }

    /// Go：`native_spawn_intro.go:163`，`func (g *Game) startNativeSpawnIntro(group int, rawGate byte, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_spawn_intro(
        &mut self,
        group: i64,
        raw_gate: u8,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        let old_count = self.story_actors.len() as i64;
        let (mut candidate_actors, candidate_roster) = self.prepare_native_story_spawn(group, raw_gate)?;
        let candidate_map_state = self
            .prepare_native_story_map_spawn_state(&mut candidate_actors, old_count)
            .map_err(|err| game_err!("native spawn-intro candidate map state: {err}"))?;
        let old_input = self
            .build_native_indexed_transition_input_for_actors(&self.story_actors)
            .map_err(|err| game_err!("native spawn-intro baseline input: {err}"))?;
        let candidate_input = self
            .build_native_indexed_transition_input_for_actors(&candidate_actors)
            .map_err(|err| game_err!("native spawn-intro candidate input: {err}"))?;
        let a = self.native_map_assets.as_ref().expect("build_native_indexed_transition_input 已檢查素材庫");
        let job = self
            .build_native_spawn_intro_job(&old_input.input(a), &candidate_input.input(a), old_count, then)?
            .expect("buildNativeSpawnIntroJob 成功時不回 nil");

        // Commit constructor state only after the full renderer transaction has
        // succeeded. Native mutates the unit array before pass 0, so the blocked
        // presentation owns this already-constructed roster until completion.
        self.story_actors = candidate_actors;
        self.story_native_map_state = candidate_map_state;
        self.story_roster = candidate_roster;
        self.story_spawned.insert(group, true);
        self.spawn_intro_transition = Some(job);
        Ok(())
    }
}

/// Go：`native_spawn_intro.go:197`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeBattleIntroEvidence {
    pub(crate) event_id: i64,
    pub(crate) group: i64,
    pub(crate) acting_resource: i64,
    pub(crate) spawn_source: String,
    pub(crate) acting_source: String,
}

/// Go：`native_spawn_intro.go:202`
pub(crate) static NATIVE_BATTLE_INTRO_EVIDENCE_TABLE: LazyLock<Vec<NativeBattleIntroEvidence>> = LazyLock::new(|| {
    vec![
        NativeBattleIntroEvidence {
            event_id: 1,
            group: 4,
            acting_resource: 3,
            spawn_source: String::from("0x342ce"),
            acting_source: String::from("0x342e7"),
        },
        NativeBattleIntroEvidence {
            event_id: 2,
            group: 5,
            acting_resource: 4,
            spawn_source: String::from("0x34336"),
            acting_source: String::from("0x3434f"),
        },
    ]
});

/// nativeBattleIntroCall admits only the two proven global callers. The raw
/// addresses remain authoritative; editable metadata may describe the calls,
/// but it cannot silently redefine faithful-mode executable provenance.
///
/// Go：`native_spawn_intro.go:210`，`func nativeBattleIntroCall(action battle.Action) (battle.NativeSpawnCall, bool, error)`
///
/// R4 B 組：Go `strings.EqualFold` 比較十六進位位址字串，Rust 用 ASCII 不分大小寫比較（只含 ASCII）。
pub(crate) fn native_battle_intro_call(action: &battle::Action) -> Result<(battle::NativeSpawnCall, bool)> {
    let intro_count = action.native_spawns.iter().filter(|c| c.via == "spawn_group_with_intro").count();
    if intro_count == 0 {
        return Ok((battle::NativeSpawnCall::default(), false));
    }
    let Some(event_id) = action.native_event_id.filter(|_| {
        action.r#type == "spawn_group"
            && intro_count == 1
            && action.native_spawns.len() == 1
            && action.groups.len() == 1
            && action.camp == "enemy"
            && !action.act_immediately
    }) else {
        return Err(GameError::new("native battle intro action differs from the proven single-call event"));
    };
    let call = &action.native_spawns[0];
    let Some(following) =
        call.following_acting.as_ref().filter(|_| call.raw_placement_gate == Some(0) && call.group == action.groups[0])
    else {
        return Err(GameError::new("native battle intro call lacks exact gate/group/acting metadata"));
    };
    for evidence in NATIVE_BATTLE_INTRO_EVIDENCE_TABLE.iter() {
        if event_id == evidence.event_id
            && call.group == evidence.group
            && call.source.eq_ignore_ascii_case(&evidence.spawn_source)
            && following.resource == evidence.acting_resource
            && following.source.eq_ignore_ascii_case(&evidence.acting_source)
        {
            return Ok((call.clone(), true));
        }
    }
    Err(GameError::new("native battle intro provenance differs from event1/event2 callers"))
}

impl Game {
    /// Go：`native_spawn_intro.go:240`，`func (g *Game) loadNativeBattleFollowingActing( meta *battle.NativeFollowingActing, unitCount int, ) ([]campaign.ActingFrame, error)`
    pub(crate) fn load_native_battle_following_acting(
        &self,
        meta: Option<&battle::NativeFollowingActing>,
        unit_count: i64,
    ) -> Result<Vec<campaign::ActingFrame>> {
        let (Some(sc), Some(meta)) = (self.sc.as_ref(), meta) else {
            return Err(GameError::new("native battle following acting resource set unavailable"));
        };
        if sc.native_acting_resources.is_empty() {
            return Err(GameError::new("native battle following acting resource set unavailable"));
        }
        let resources = campaign::load_acting_resource_set(&asset_path(&sc.native_acting_resources))
            .map_err(|err| game_err!("native battle acting resources: {err}"))?;
        let Some(frames) = resources.get(&meta.resource).filter(|f| !f.is_empty()) else {
            return Err(game_err!("native battle acting resource {} unavailable", meta.resource));
        };
        for (frame_index, frame) in frames.iter().enumerate() {
            if frame.beats < 0 || frame.beats > 0x7f || (!frame.special && frame.beats == 0) || frame.units.is_empty() {
                return Err(game_err!(
                    "native battle acting resource {} frame {} has invalid timing/targets",
                    meta.resource,
                    frame_index
                ));
            }
            let mut seen = BTreeSet::new();
            for unit in &frame.units {
                let valid = unit.slot.is_some_and(|s| {
                    s >= 0 && s < unit_count && unit.pose >= 0 && unit.pose <= 3 && !seen.contains(&s)
                });
                if !valid {
                    return Err(game_err!(
                        "native battle acting resource {} frame {} has invalid runtime slot/pose",
                        meta.resource,
                        frame_index
                    ));
                }
                seen.insert(unit.slot.unwrap());
            }
        }
        Ok(frames.clone())
    }

    /// Go：`native_spawn_intro.go:270`，`func (g *Game) startNativeBattleFollowingActing(frames []campaign.ActingFrame, then func())`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_battle_following_acting(
        &mut self,
        frames: &mut [campaign::ActingFrame],
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) {
        let mut job = ActPoseJob { acting: frames.to_vec(), then, ..Default::default() };
        self.begin_acting_frame(&mut job);
        self.act_job = Some(job);
    }

    /// Go：`native_spawn_intro.go:276`，`func (g *Game) startNativeBattleSpawnIntro( action battle.Action, call battle.NativeSpawnCall, then func(), ) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_battle_spawn_intro(
        &mut self,
        action: &battle::Action,
        call: &battle::NativeSpawnCall,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        let (Some(gate), Some(following)) = (call.raw_placement_gate, call.following_acting.as_ref()) else {
            return Err(GameError::new("native battle intro state/call unavailable"));
        };
        let Some(old_count) = self.st.as_ref().map(|st| st.units.len()) else {
            return Err(GameError::new("native battle intro state/call unavailable"));
        };
        let mut candidate =
            clone_native_battle_spawn_state(self.st.as_mut())?.expect("cloneNativeBattleSpawnState 成功時不回 nil");
        let appended = candidate.append_group_with_native_placement(call.group, gate as u8)?;
        if appended <= 0 || candidate.units.len() as i64 != old_count as i64 + appended {
            return Err(GameError::new("native battle intro candidate append boundary unavailable"));
        }
        // #243 READY：32999沿10EED保留raw camp及+5=0，新敵軍在同phase行動。
        // Camp／Acted已由原生constructor投影，不再套用正規化ActImmediately。
        let acting = self.load_native_battle_following_acting(Some(following), candidate.units.len() as i64)?;
        let old_input = self
            .build_native_indexed_transition_input_for_state(self.st.as_ref())
            .map_err(|err| game_err!("native battle intro baseline input: {err}"))?;
        let candidate_input = self
            .build_native_indexed_transition_input_for_state(Some(&candidate))
            .map_err(|err| game_err!("native battle intro candidate input: {err}"))?;
        let a = self.native_map_assets.as_ref().expect("build_native_indexed_transition_input_for_state 已檢查素材庫");
        let job = self
            .build_native_spawn_intro_job(
                &old_input.input(a),
                &candidate_input.input(a),
                old_count as i64,
                Some(Box::new(move |g: &mut Game| {
                    let mut acting = acting;
                    g.start_native_battle_following_acting(&mut acting, then);
                })),
            )?
            .expect("buildNativeSpawnIntroJob 成功時不回 nil");

        // Publish only the fields changed by the proven constructor. Existing unit
        // pointers remain stable; newly appended records and the selector cache come
        // from the fully preflighted private state.
        let new_units = clone_battle_unit_pointers(&candidate.units[old_count..]);
        let st = self.st.as_mut().unwrap();
        st.units.extend(new_units);
        st.roster = clone_battle_unit_pointers(&candidate.roster);
        st.native_map_selector_cache = candidate.native_map_selector_cache;
        st.native_map_selector_error = candidate.native_map_selector_error;
        st.native_map_cycle_state = candidate.native_map_cycle_state;
        st.has_native_map_cycle_state = candidate.has_native_map_cycle_state;
        st.native_terrain_phase_state = candidate.native_terrain_phase_state;
        st.has_native_terrain_phase_state = candidate.has_native_terrain_phase_state;
        st.native_terrain_flip_state = candidate.native_terrain_flip_state;
        st.native_unit_pixel_shift_state = candidate.native_unit_pixel_shift_state;
        st.has_native_map_binary_timing_state = candidate.has_native_map_binary_timing_state;
        self.spawn_intro_transition = Some(job);
        Ok(())
    }

    /// Go：`native_spawn_intro.go:337`，`func (g *Game) stepNativeSpawnIntro()`
    ///
    /// R4 B 組：`drawn` 由呈現回報設定（Go 在 `drawNativeSpawnIntro` 設）。
    pub(crate) fn step_native_spawn_intro(&mut self) {
        let Some(j) = self.spawn_intro_transition.as_mut().filter(|j| j.drawn) else {
            return;
        };
        if j.frame + 1 < j.frames.len() as i64 {
            j.frame += 1;
            let frame = j.frame as usize;
            j.vga.copy_from_slice(&j.frames[frame]);
            j.drawn = false;
            if j.schedule[frame].sound_resource == fdother::NATIVE_SPAWN_INTRO_SOUND_RESOURCE {
                let sound = j.sound.clone();
                self.play_raw(&sound);
            }
            return;
        }
        let mut j = self.spawn_intro_transition.take().unwrap();
        let then = j.then.take();
        self.native_map_work.clear();
        self.native_map_work.extend_from_slice(&j.work);
        self.native_map_vga.clear();
        self.native_map_vga.extend_from_slice(&j.vga);
        if let Some(then) = then {
            then(self);
        }
    }
}

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_spawn_intro.go:360`，`func (g *Game) drawNativeSpawnIntro(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_spawn_intro(&mut self, screen: &mut Picture) -> bool {
        let Some(j) = self.spawn_intro_transition.as_mut() else {
            return false;
        };
        if j.vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE || j.palette.len() != 256 {
            return false;
        }
        let img = IndexedFrame { width: 320, height: 200, pix: j.vga.clone(), palette: j.palette.clone() };
        let mut op = DrawImageOptions::default();
        op.geo_m.scale(2.0, 2.0);
        screen.draw_image(&Picture::from_indexed(&img), &op);
        j.drawn = true;
        true
    }
}
