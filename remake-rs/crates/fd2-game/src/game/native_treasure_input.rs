//! Go 來源：`remake/cmd/fd2/native_treasure_input.go`（121 行）。

use super::*;

/// Go：`native_treasure_input.go:10`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeTreasurePrompt {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) reward: battle::Treasure,
    pub(crate) x: i64,
    pub(crate) y: i64,
    pub(crate) await_ack: bool,
    /// ClaimTreasure 已成立（YES 且交易通過）
    pub(crate) claimed: bool,
}

impl Game {
    pub(crate) fn begin_native_treasure_prompt(&mut self, actor: UnitIdx, reward: battle::Treasure) -> Result<()> {
        let unit = &self.st.as_ref().ok_or_else(|| game_err!("寶物確認缺少戰場"))?.units[actor];
        if self.native_class_ui.is_none() || self.native_preparation_ui.is_none() || !unit.has_battle_fig {
            return Err(game_err!("寶物確認缺少介面素材或角色頭像編號"));
        }
        let prompt = NativeTreasurePrompt { actor: Some(actor), reward, x: unit.x, y: unit.y, ..Default::default() };
        let portraits = load_native_separated_portrait(unit.battle_fig)?;
        if portraits.len() <= 3 {
            return Err(game_err!("寶物確認的角色頭像不完整"));
        }
        self.compose_native_map_frame_for_action_background(true)?;
        let ui = self.native_preparation_ui.as_ref().unwrap();
        let source = self.native_map_vga.clone();
        let dialogue =
            campaign::compose_native_preparation_confirmation_dialogue(&source, &ui.dialogue, &portraits[0])?;
        let (strings, font) = (ui.status.strings.as_ref(), ui.status.font.as_ref());
        let question = campaign::compose_native_treasure_item_question(
            &dialogue,
            &portraits[0],
            strings.ok_or_else(|| game_err!("寶物確認缺少原版文字"))?,
            font.ok_or_else(|| game_err!("寶物確認缺少原版字型"))?,
            prompt.reward.hidden,
        )?;
        let accepted = match prompt.reward.kind.as_str() {
            "item" => campaign::native_treasure_item_response_frames(
                &question,
                strings,
                font,
                prompt.reward.value,
                prompt.reward.hidden,
            )?,
            "gold" => campaign::native_treasure_gold_response_frames(
                &question,
                strings,
                font,
                prompt.reward.value,
                prompt.reward.hidden,
            )?,
            kind => return Err(game_err!("寶物種類 {kind} 無確認流程")),
        };
        let canceled = campaign::native_battle_end_turn_response_frames(&question, strings, font, false)?;
        let frames =
            campaign::native_preparation_confirmation_opening_frames(&source, &dialogue, &question, &ui.choices)?;
        self.native_system_end_turn_ui = Some(Box::new(NativeSystemEndTurnUIState {
            source,
            dialogue,
            question,
            portraits,
            accepted,
            canceled,
            treasure: Some(prompt),
            ..Default::default()
        }));
        self.native_system_end_turn_confirm = true;
        self.ring = false;
        self.msg.clear();
        self.reset_native_class_ui_pulse();
        self.native_class_ui_job = Some(NativeClassUIJob { frames, ..Default::default() });
        Ok(())
    }

    pub(crate) fn finish_native_treasure_choice(&mut self, accepted: bool) {
        let Some((state, ui)) = self.native_system_end_turn_ui.as_mut().zip(self.native_preparation_ui.as_ref()) else {
            return;
        };
        let frames = match campaign::native_class_confirmation_closing_frames(&state.question, &ui.choices) {
            Ok(frames) => frames,
            Err(err) => {
                self.load_err = err.to_string();
                return;
            }
        };
        self.native_system_end_turn_confirm = false;
        state.accepted_outcome = accepted;
        let response = if accepted { &state.accepted } else { &state.canceled }.clone();
        self.native_class_ui_job = Some(NativeClassUIJob {
            frames,
            after: Some(Box::new(move |g| {
                g.native_class_ui_job = Some(NativeClassUIJob {
                    frames: response,
                    after: Some(Box::new(move |g| g.finish_native_treasure_response(accepted))),
                    ..Default::default()
                });
            })),
            ..Default::default()
        });
    }

    fn finish_native_treasure_response(&mut self, accepted: bool) {
        if !accepted {
            // 0x17259 的 200 ms 回覆停留。
            self.native_system_end_turn_delay = 12;
            return;
        }
        let Some(prompt) = self.native_system_end_turn_ui.as_mut().and_then(|state| state.treasure.as_mut()) else {
            return;
        };
        if prompt.claimed {
            return;
        }
        let claimed = self.st.as_mut().is_some_and(|state| {
            let (current, available) = state.treasure_at(prompt.x, prompt.y);
            let actor = prompt.actor.and_then(|index| state.units.get(index));
            let actor_ready =
                actor.is_some_and(|unit| unit.alive() && unit.on_field && (unit.x, unit.y) == (prompt.x, prompt.y));
            available && current == prompt.reward && actor_ready
                // 金錢在關框後才入帳並標記取得。
                && (prompt.reward.kind == "gold"
                    || state.claim_treasure_before_closing(prompt.actor, prompt.x, prompt.y).1)
        });
        prompt.claimed = claimed;
        prompt.await_ack = claimed;
        if !claimed {
            self.load_err = String::from("寶物確認時來源或領取狀態改變");
            self.native_system_end_turn_delay = 1;
        }
    }

    pub(crate) fn finish_native_treasure_prompt(&mut self, prompt: NativeTreasurePrompt) {
        if prompt.claimed {
            let state = self.st.as_mut().expect("寶物確認期間保留原戰場");
            if prompt.reward.kind == "gold" {
                let (current, available) = state.treasure_at(prompt.x, prompt.y);
                if !available
                    || current != prompt.reward
                    || !state.claim_treasure_before_closing(prompt.actor, prompt.x, prompt.y).1
                {
                    self.load_err = String::from("關框後無法領取寶物金錢");
                    return;
                }
                self.gold += prompt.reward.value;
            }
            if !state.complete_treasure_map_update() {
                self.load_err = String::from("關框後無法更新寶物地圖");
                return;
            }
        }
        self.finish_successful_unit_action(
            prompt.actor,
            Some(Box::new(|g| (g.sel, g.reach, g.moved) = (None, BTreeMap::new(), false))),
        );
    }
}
