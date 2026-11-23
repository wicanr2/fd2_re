use super::*;

impl Game {
    pub(crate) fn eval_story_condition(&self, condition: Option<&campaign::BeatCondition>) -> Result<bool> {
        let condition = condition.ok_or_else(|| game_err!("缺少 handler condition"))?;
        match condition.op.as_str() {
            "native_event_state_nonzero" | "native_event_state_eq" => {
                let state = self.st.as_ref().ok_or_else(|| game_err!("事件條件缺少戰場狀態"))?;
                let index = condition
                    .event_state_index
                    .and_then(|index| usize::try_from(index).ok())
                    .ok_or_else(|| game_err!("事件狀態索引無效"))?;
                let value = *state.native_event_state.get(index).ok_or_else(|| game_err!("事件狀態索引超出範圍"))?;
                if condition.op == "native_event_state_nonzero" {
                    return Ok(value != 0);
                }
                let expected = condition
                    .event_state_value
                    .and_then(|value| u8::try_from(value).ok())
                    .ok_or_else(|| game_err!("事件狀態條件值超出原版範圍"))?;
                let required = condition
                    .required_slot_count
                    .filter(|&count| count > 0)
                    .ok_or_else(|| game_err!("事件狀態條件缺少原生名冊筆數"))?;
                if value == expected && state.units.len() as i64 != required {
                    return Err(game_err!("事件狀態符合，但原生名冊筆數為 {}，預期 {required}", state.units.len()));
                }
                Ok(value == expected)
            }
            "roster_has" => {
                let identity = condition
                    .char_id
                    .filter(|&id| campaign::joinable_character_id(id))
                    .ok_or_else(|| game_err!("永久隊伍條件缺少有效身分"))?;
                if self.party_members.is_empty() {
                    return Err(game_err!("缺少永久隊伍"));
                }
                Ok(self.party_members.get(&identity).copied().unwrap_or(false))
            }
            "any_unit_inactive" | "native_inactive_count_gt" => self.eval_story_inactive_condition(condition),
            "native_inventory_item_present" => {
                let state = self
                    .st
                    .as_ref()
                    .or({ if self.has_story_native_map_view { self.story_native_map_state.as_ref() } else { None } })
                    .ok_or_else(|| game_err!("原生物品條件缺少戰場狀態"))?;
                let item = condition
                    .native_inventory_item_id
                    .and_then(|item| u8::try_from(item).ok())
                    .ok_or_else(|| game_err!("原生物品條件缺少有效物品"))?;
                let records = battle::native_inventory_records(&state.units, 16)?;
                let (unit, _) = battle::find_native_inventory_item(&records, item)?;
                Ok(unit >= 0)
            }
            "native_persistent_identity_present" => {
                let identity = condition
                    .native_persistent_identity
                    .filter(|id| (0..=0xff).contains(id))
                    .ok_or_else(|| game_err!("永久名冊條件缺少原生身分"))?;
                if self.party_roster.is_empty() {
                    return Err(game_err!("缺少永久名冊"));
                }
                for (&id, unit) in &self.party_roster {
                    if !unit.has_native_identity {
                        return Err(game_err!("永久角色 {id} 缺少原生身分"));
                    }
                    if unit.native_identity == identity {
                        return Ok(true);
                    }
                }
                Ok(false)
            }
            "native_round_gt" | "native_round_lt" => {
                let round = self
                    .st
                    .as_ref()
                    .map(|state| state.native_round_counter)
                    .filter(|&round| round > 0)
                    .ok_or_else(|| game_err!("回合條件缺少原生回合數"))?;
                let expected =
                    condition.native_round.filter(|&round| round >= 0).ok_or_else(|| game_err!("回合條件值無效"))?;
                Ok(if condition.op == "native_round_gt" { round > expected } else { round < expected })
            }
            "native_record_word_gte" => {
                let state = self.st.as_ref().ok_or_else(|| game_err!("記錄條件缺少戰場狀態"))?;
                let index = condition
                    .unit_slot
                    .and_then(|slot| usize::try_from(slot).ok())
                    .ok_or_else(|| game_err!("記錄條件缺少原生格位"))?;
                let unit = state
                    .units
                    .get(index)
                    .filter(|unit| unit.has_native_record_word42)
                    .ok_or_else(|| game_err!("原生格位 {index} 缺少 +0x42"))?;
                let expected = condition
                    .native_record_word_value
                    .and_then(|word| u16::try_from(word).ok())
                    .filter(|_| condition.native_record_word_offset == Some(0x42))
                    .ok_or_else(|| game_err!("記錄條件缺少有效的 +0x42 值"))?;
                Ok(unit.native_record_word42 >= expected)
            }
            "native_any_of" => {
                if condition.any.is_empty() {
                    return Err(game_err!("組合條件缺少子條件"));
                }
                let mut error = None;
                for sub in &condition.any {
                    match self.eval_story_condition(Some(sub)) {
                        Ok(true) => return Ok(true),
                        Ok(false) => {}
                        Err(err) => {
                            error.get_or_insert(err);
                        }
                    }
                }
                error.map_or(Ok(false), Err)
            }
            op => Err(game_err!("未知 handler condition {op:?}")),
        }
    }

    fn eval_story_inactive_condition(&self, condition: &campaign::BeatCondition) -> Result<bool> {
        let state = self.st.as_ref().ok_or_else(|| game_err!("狀態條件缺少戰場狀態"))?;
        if condition.unit_slots.is_empty() {
            return Err(game_err!("狀態條件缺少原生格位"));
        }
        let strict = condition.op == "native_inactive_count_gt";
        let threshold = if strict {
            condition.threshold.filter(|&count| count >= 0).ok_or_else(|| game_err!("停用筆數門檻無效"))?
        } else {
            0
        };
        let mut count = 0;
        for &slot in &condition.unit_slots {
            let unit = state.units.get(slot as usize).ok_or_else(|| game_err!("缺少原生格位 {slot}"))?;
            let inactive = if unit.has_native_record_byte5 {
                unit.native_record_byte5 & 1 != 0
            } else if strict {
                return Err(game_err!("原生格位 {slot} 缺少 +5"));
            } else {
                // Legacy scenarios carry the old editable HP/field projection.
                !unit.on_field || !unit.alive()
            };
            count += i64::from(inactive);
        }
        Ok(count > threshold)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn postbattle_civilian_condition_prefers_raw_flags_and_checks_all_slots() {
        let mut state = battle::State::default();
        state.units = vec![battle::Unit::default()];
        state.units[0].has_native_record_byte5 = true;
        let mut game = Game { st: Some(state), ..Default::default() };
        let mut condition =
            campaign::BeatCondition { op: "any_unit_inactive".into(), unit_slots: vec![0], ..Default::default() };
        assert!(!game.eval_story_condition(Some(&condition)).unwrap());
        game.st.as_mut().unwrap().units[0].native_record_byte5 = 1;
        assert!(game.eval_story_condition(Some(&condition)).unwrap());
        condition.unit_slots.push(1);
        assert!(game.eval_story_condition(Some(&condition)).is_err());
    }
}
