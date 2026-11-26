//! Go 來源：`remake/cmd/fd2/native_turn_event76.go`（188 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// startNativeRawCamp2TurnEvents 是 round increment 後、玩家重新取得輸入前的
    /// sub_1A813(2) owner。缺少唯一 typed consumer 時維持失敗即關閉。
    ///
    /// Go：`native_turn_event76.go:112`，`func (g *Game) startNativeRawCamp2TurnEvents(then func()) (bool, error)`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_raw_camp2_turn_events(
        &mut self,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<bool> {
        // R4 B 組：最後一輪（`plan.Final`）的預檢與演出呼叫 `preflightNativeEvent76Final`、
        // `event76DialogueActions`、`runNativeEvent76Presentation`（都是 `r8`，第 29 章），停在 `todo!`。
        let Some(sc) = self.sc.as_ref().filter(|sc| !sc.native_turn_events.is_empty()) else {
            return Ok(false);
        };
        let events = sc.native_turn_events_at(self.st.as_ref(), 2)?;
        if events.is_empty() {
            return Ok(false);
        }
        if events.len() != 1 || events[0].event_id != 76 {
            return Err(game_err!("raw camp2 has {} handlers without a unique event76 adapter", events.len()));
        }
        let event = &events[0];
        let plan = battle::plan_native_turn_event76(self.st.as_ref(), event)?;
        if !plan.r#final {
            battle::commit_native_turn_event76_repeat(self.st.as_mut(), &plan)?;
            if let Some(then) = then {
                then(self);
            }
            return Ok(true);
        }
        self.preflight_native_event76_final(event)?;
        let progression = event.progression.as_ref().ok_or_else(|| game_err!("event76 progression unavailable"))?;
        let Some(mut actions) = event76_dialogue_actions(self, progression.final_text_index) else {
            return Err(game_err!("event76 FDTXT_029 index2 unavailable"));
        };
        let event = event.clone();
        self.start_battle_event(
            &mut actions,
            Some(Box::new(move |g: &mut Game| {
                let (candidate, frame) = match g.preflight_native_event76_final(&event) {
                    Ok(result) => result,
                    Err(err) => {
                        g.load_err = err.to_string();
                        return;
                    }
                };
                g.adopt_native_state_candidate(Some(candidate));
                g.native_map_vga = frame;
                g.run_native_event76_presentation(event, 0, then);
            })),
        );
        Ok(true)
    }

    /// 最後一輪的候選狀態：加入 spawn group、把 base 寫進事件狀態表，並把沉睡的 event79 列排到本回合。
    ///
    /// Go：`native_turn_event76.go:54`，`func (g *Game) preflightNativeEvent76FinalState(event battle.NativeTurnEvent) (*battle.State, error)`
    fn preflight_native_event76_final_state(&self, event: &battle::NativeTurnEvent) -> Result<battle::State> {
        let plan = battle::plan_native_turn_event76(self.st.as_ref(), event)
            .map_err(|err| game_err!("event76 final plan unavailable: {err}"))?;
        if !plan.r#final {
            return Err(game_err!("event76 final plan unavailable"));
        }
        let progression = event.progression.as_ref().ok_or_else(|| game_err!("event76 progression unavailable"))?;
        let mut candidate = clone_native_turn_staging_state(self.st.as_ref())?
            .ok_or_else(|| game_err!("event76 candidate state unavailable"))?;
        let base = candidate.units.len();
        let appended =
            candidate.append_group_with_native_placement(progression.spawn_group, progression.raw_placement_gate as u8);
        match appended {
            Ok(n) if n == progression.spawn_count && base <= 0xff => {}
            other => {
                return Err(game_err!(
                    "event76 group{} append={:?} base={base}",
                    progression.spawn_group,
                    other.map_err(|err| err.to_string())
                ));
            }
        }
        let activation = progression.final_activation;
        let slot = usize::try_from(activation.slot)
            .ok()
            .filter(|&slot| slot < candidate.native_turn_event_controls.len())
            .ok_or_else(|| game_err!("event76 dormant event79 row identity mismatch"))?;
        let expected = battle::NativeTurnEventControl {
            turn: 0xff,
            event_id: activation.event_id as u8,
            raw_camp: activation.raw_camp,
        };
        if candidate.native_turn_event_controls[slot] != expected {
            return Err(game_err!("event76 dormant event79 row identity mismatch"));
        }
        let state_index = usize::try_from(progression.base_state_index)
            .ok()
            .filter(|&index| index < candidate.native_event_state.len())
            .ok_or_else(|| game_err!("event76 base state index unavailable"))?;
        candidate.native_event_state[state_index] = base as u8;
        candidate.native_turn_event_controls[slot].turn = candidate.native_round_counter as u8;
        if candidate.has_native_field_control_state {
            let offset = 3 + slot * 3;
            let raw = &mut candidate.native_field_control_raw;
            if raw.len() <= offset + 2
                || raw[offset] != 0xff
                || raw[offset + 1] != activation.event_id as u8
                || raw[offset + 2] != activation.raw_camp
            {
                return Err(game_err!("event76 raw event79 row disagrees with typed controls"));
            }
            raw[offset] = candidate.native_round_counter as u8;
        }
        Ok(candidate)
    }

    /// 候選狀態加上以它合成的一幀索引色地圖；調色盤演出器必須閒置。
    ///
    /// Go：`native_turn_event76.go:88`，`func (g *Game) preflightNativeEvent76Final(event battle.NativeTurnEvent) (*battle.State, []byte, error)`
    fn preflight_native_event76_final(&mut self, event: &battle::NativeTurnEvent) -> Result<(battle::State, Vec<u8>)> {
        let candidate = self.preflight_native_event76_final_state(event)?;
        if self.native_palette_presenter_active()
            || self.native_unit_present.is_some()
            || self.transition_reveal.is_some()
            || self.indexed_transition.is_some()
            || !native_map_assets_available(self.native_map_assets.as_ref())
        {
            return Err(game_err!("event76 indexed palette presenter unavailable"));
        }
        let frame = self
            .probe_native_map_frame(candidate.clone(), None)
            .map_err(|err| game_err!("event76 group1 indexed frame: {err}"))?;
        if frame.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE {
            return Err(game_err!("event76 indexed frame size={}", frame.len()));
        }
        Ok((candidate, frame))
    }

    /// 依序播 `pulse_count` 次調色盤脈衝；前兩次後各延遲 `extra_delay_ms`，之後每次接一段尾段對白。
    ///
    /// Go：`native_turn_event76.go:160`，`func (g *Game) runNativeEvent76Presentation(event battle.NativeTurnEvent, pulse int, then func())`
    fn run_native_event76_presentation(
        &mut self,
        event: battle::NativeTurnEvent,
        pulse: i64,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) {
        let Some(progression) = event.progression.clone().filter(|p| (0..=p.pulse_count).contains(&pulse)) else {
            self.load_err = String::from("event76 presentation state unavailable");
            return;
        };
        if pulse == progression.pulse_count {
            if let Some(then) = then {
                then(self);
            }
            return;
        }
        let result = self.start_native_palette_pulse(
            &native_event76_pulse_spec(),
            Box::new(move |g: &mut Game| {
                let next: Box<dyn FnOnce(&mut Game) + Send> =
                    Box::new(move |g: &mut Game| g.run_native_event76_presentation(event, pulse + 1, then));
                if pulse < 2 {
                    let mut delay = [battle::Action {
                        r#type: String::from("delay"),
                        ms: progression.extra_delay_ms,
                        ..Default::default()
                    }];
                    g.start_battle_event(&mut delay, Some(next));
                    return;
                }
                let tail = progression.tail_text_indices.get((pulse - 2) as usize).copied();
                let Some(mut actions) = tail.and_then(|index| event76_dialogue_actions(g, index)) else {
                    g.load_err = format!("event76 tail dialogue {} unavailable", pulse - 2);
                    return;
                };
                g.start_battle_event(&mut actions, Some(next));
            }),
        );
        if let Err(err) = result {
            self.load_err = format!("event76 palette pulse: {err}");
        }
    }
}

/// FDTXT_029 index2..6 對應 `assets/story/ch29.json` 的句段。
///
/// Go：`native_turn_event76.go:11`，`func event76DialogueActions(g *Game, textIndex int) ([]battle.Action, bool)`
fn event76_dialogue_actions(g: &mut Game, text_index: i64) -> Option<Vec<battle::Action>> {
    let (scene, line, count) = match text_index {
        2 => (0, 6, 3),
        3 => (0, 9, 1),
        4 => (0, 10, 1),
        5 => (0, 11, 1),
        6 => (1, 0, 5),
        _ => return None,
    };
    super::native_field_event75::ch29_story_dialogue_actions(g, scene, line, count, "event76")
}

/// Go：`native_turn_event76.go:46`，`func nativeEvent76PulseSpec() campaign.NativePalettePulse`
fn native_event76_pulse_spec() -> campaign::NativePalettePulse {
    campaign::NativePalettePulse {
        rise_start: 0,
        rise_end: 63,
        rise_delay_ms: 8,
        hold_ms: 400,
        fall_start: 62,
        fall_end: 0,
        fall_delay_ms: 8,
    }
}
