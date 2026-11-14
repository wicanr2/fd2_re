//! Go 來源：`remake/internal/campaign/native_continue_pending_groups.go`（320 行）。

use super::*;

/// Go：`native_continue_pending_groups.go:13`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeContinueTurnKey {
    pub(crate) turn: i64,
    pub(crate) event_id: i64,
}

/// MaterializeNativeContinuePendingGroups 將仍未觸發的可編輯回合增援，綁到
/// CONTINUE 戰場的未物化 FDFIELD roster。原版存檔選單在 0x117E7 的玩家互動
/// 階段呼叫 0x19DF7；只有 0x13565 判定玩家可行動列耗盡後才進 0x1A30B。
/// 其中 raw selector 1/0 在增加回合前掃描，selector 2 則在增加後掃描；所以
/// saved turn 的 selector 1/0 仍待執行，selector 2 已於上一輪尾端執行。
///
/// 這個適配器只接受 live turn/event pair 與可編輯 scenario 完全相符的章節。
/// 原版 handler 會改寫 live turn byte；尚未資料化的改寫一律拒絕，不用靜態
/// FDFIELD 或舊 scenario 覆寫存檔。所有檢查與深複製成功後才提交 State。
///
/// Go：`native_continue_pending_groups.go:27`，`func MaterializeNativeContinuePendingGroups( state *battle.State, input fdsave.ContinueRuntimeInput, assetChapter int, assetState *battle.State, scenario *battle.Scenario, itemRows []byte, ) error`
pub fn materialize_native_continue_pending_groups(
    state: Option<&mut battle::State>,
    input: &save::ContinueRuntimeInput,
    asset_chapter: i64,
    asset_state: Option<&battle::State>,
    scenario: Option<&battle::Scenario>,
    item_rows: &[u8],
) -> Result<()> {
    let (Some(state), Some(asset_state), Some(scenario)) = (state, asset_state, scenario) else {
        return Err(campaign_err!("native CONTINUE pending groups: missing state or scenario"));
    };
    if !input.validated_for_runtime_bridge() {
        return Err(campaign_err!("native CONTINUE pending groups: input did not pass preflight"));
    }
    if asset_chapter != input.context.chapter
        || scenario.chapter != asset_chapter + 1
        || scenario.map != asset_chapter
        || asset_state.w != input.context.field_width
        || asset_state.h != input.context.field_height
        || state.w != asset_state.w
        || state.h != asset_state.h
    {
        return Err(campaign_err!("native CONTINUE pending groups: chapter asset mismatch"));
    }
    if !state.has_native_field_control_state
        || !state.has_native_runtime_unit_projection
        || state.native_round_counter != i64::from(input.header.turn_counter)
    {
        return Err(campaign_err!("native CONTINUE pending groups: prior runtime boundary is incomplete"));
    }
    if asset_state.units.len() != state.native_field_unit_controls.len()
        || asset_state.native_composition_event_bytes.len() as i64 != asset_state.w.wrapping_mul(asset_state.h)
        || state.native_composition_event_bytes.len() as i64 != state.w.wrapping_mul(state.h)
        || state.native_composition_event_bytes != asset_state.native_composition_event_bytes
        || state.native_field_event_slots != asset_state.native_field_event_slots
        || state.native_field_event_rules != asset_state.native_field_event_rules
    {
        return Err(campaign_err!("native CONTINUE pending groups: FDFIELD asset topology mismatch"));
    }

    let schedule =
        native_continue_spawn_schedule(scenario).map_err(|err| err.wrap("native CONTINUE pending groups: "))?;
    if !scenario.runtime_append_groups {
        if !schedule.is_empty() {
            return Err(campaign_err!("native CONTINUE pending groups: static scenario declares future groups"));
        }
        for rule in &asset_state.native_field_event_rules {
            if rule.spawn_group.is_some() {
                return Err(campaign_err!(
                    "native CONTINUE pending groups: static scenario field event {} declares a future group",
                    rule.event_id
                ));
            }
        }
        let mut candidate = state.clone();
        candidate.roster = Vec::new();
        candidate.pending_groups = BTreeMap::new();
        candidate
            .bind_native_future_item_rows(item_rows)
            .map_err(|err| CampaignError::from(err).wrap("native CONTINUE pending groups: "))?;
        candidate.has_native_pending_group_binding = true;
        *state = candidate;
        return Ok(());
    }
    let current_turn = i64::from(input.header.turn_counter);
    if current_turn <= 0 {
        return Err(campaign_err!("native CONTINUE pending groups: saved turn is zero"));
    }
    // Go 以 map 迭代 schedule；累積結果與順序無關，順序只影響同時有多個錯誤時先報哪一個。
    // 下面三個錯誤在 Go 原樣回傳、沒有加 "native CONTINUE pending groups: " 前綴（Go 第 104～137 行）。
    let mut pending = BTreeSet::new();
    for (key, groups) in &schedule {
        let mut matches = 0;
        let mut raw_selector = 0u8;
        for live in &state.native_turn_event_controls {
            if i64::from(live.turn) == key.turn && i64::from(live.event_id) == key.event_id {
                if live.raw_camp > 2 {
                    return Err(campaign_err!(
                        "live turn {} event {} has raw selector {}",
                        key.turn,
                        key.event_id,
                        live.raw_camp
                    ));
                }
                raw_selector = live.raw_camp;
                matches += 1;
            }
        }
        if matches != 1 {
            return Err(campaign_err!("live turn {} event {} matched {matches} rows", key.turn, key.event_id));
        }
        // 先核對所有原始排程列，再依 saved turn 分類。若 handler 把一個
        // 已過回合的列延後，先略過舊 turn 會把仍待執行的 group 靜默遺失。
        if key.turn < current_turn {
            continue;
        }
        if key.turn == current_turn && raw_selector == 2 {
            // 初始 turn1 沒有可編輯 selector2 spawn schedule；若未來資料
            // 出現此形狀，需要另證 battle bootstrap 是否先掃 selector2。
            if current_turn == 1 {
                return Err(campaign_err!("live turn 1 event {} uses ambiguous raw selector 2", key.event_id));
            }
            continue;
        }
        pending.extend(groups.iter().copied());
    }
    // 已資料化的格子事件也可能建構 future group（目前為 map25/event61）。
    // 只有其 once-state raw byte 能證明是否已消費；缺少該 producer 時不能
    // 從 current runtime record 反猜 group identity。
    for rule in &asset_state.native_field_event_rules {
        let Some(group) = rule.spawn_group else { continue };
        if !native_continue_field_spawn_rule_is_live(Some(state), Some(asset_state), rule) {
            return Err(campaign_err!(
                "native CONTINUE pending groups: field event {} lacks live selector/slot provenance",
                rule.event_id
            ));
        }
        let once = match rule.once_state {
            Some(once) if once >= 0 && once < state.native_event_state.len() as i64 => once as usize,
            _ => {
                return Err(campaign_err!(
                    "native CONTINUE pending groups: field event {} lacks once-state provenance",
                    rule.event_id
                ));
            }
        };
        if !(0..=0xfe).contains(&group) {
            return Err(campaign_err!(
                "native CONTINUE pending groups: field event {} has invalid group {group}",
                rule.event_id
            ));
        }
        if state.native_event_state[once] == 0 {
            pending.insert(group);
        }
    }

    let mut roster = Vec::new();
    let mut seen_groups: BTreeMap<i64, i64> = BTreeMap::new();
    for unit in &asset_state.units {
        if !pending.contains(&unit.group) {
            continue;
        }
        let clone = clone_native_continue_future_unit(Some(unit))
            .map_err(|err| err.wrap("native CONTINUE pending groups: "))?;
        roster.push(clone);
        *seen_groups.entry(unit.group).or_default() += 1;
    }
    // Go 先 `sort.Ints` 再檢查；BTreeSet 已是遞增順序。
    for group in &pending {
        if !(0..=0xfe).contains(group) || seen_groups.get(group).copied().unwrap_or(0) == 0 {
            return Err(campaign_err!("native CONTINUE pending groups: group {group} has no FDFIELD rows"));
        }
    }

    let mut candidate = state.clone();
    candidate.roster = roster;
    candidate.pending_groups = pending.iter().map(|group| (*group, true)).collect();
    candidate
        .bind_native_future_item_rows(item_rows)
        .map_err(|err| CampaignError::from(err).wrap("native CONTINUE pending groups: "))?;
    candidate.has_native_pending_group_binding = true;
    *state = candidate;
    Ok(())
}

/// Go：`native_continue_pending_groups.go:214`，`func nativeContinueSpawnSchedule( scenario *battle.Scenario, ) (map[nativeContinueTurnKey][]int, error)`
pub(crate) fn native_continue_spawn_schedule(
    scenario: &battle::Scenario,
) -> Result<BTreeMap<NativeContinueTurnKey, Vec<i64>>> {
    let mut schedule: BTreeMap<NativeContinueTurnKey, Vec<i64>> = BTreeMap::new();
    let mut seen_groups: BTreeMap<i64, NativeContinueTurnKey> = BTreeMap::new();
    for (event_index, event) in scenario.events.iter().enumerate() {
        for (action_index, action) in event.r#do.iter().enumerate() {
            if action.r#type != "spawn_group" {
                continue;
            }
            let (Some(when), Some(native_event_id)) = (event.when.as_ref(), action.native_event_id) else {
                return Err(campaign_err!(
                    "scenario event {event_index} action {action_index} lacks exact native schedule provenance"
                ));
            };
            if event.trigger != "on_turn_end"
                || when.turn <= 0
                || !event.once
                || action.groups.is_empty()
                || action.native_spawns.len() != action.groups.len()
            {
                return Err(campaign_err!(
                    "scenario event {event_index} action {action_index} lacks exact native schedule provenance"
                ));
            }
            let key = NativeContinueTurnKey { turn: when.turn, event_id: native_event_id };
            for (index, &group) in action.groups.iter().enumerate() {
                let call = &action.native_spawns[index];
                if !(0..=0xfe).contains(&group)
                    || call.group != group
                    || call.raw_placement_gate.is_none()
                    || call.source.is_empty()
                    || (call.via != "spawn_group" && call.via != "spawn_group_with_intro")
                {
                    return Err(campaign_err!(
                        "scenario event {event_index} action {action_index} group {group} lacks native call provenance"
                    ));
                }
                if let Some(prior) = seen_groups.get(&group) {
                    return Err(campaign_err!(
                        "group {group} is scheduled by both turn {}/event {} and turn {}/event {}",
                        prior.turn,
                        prior.event_id,
                        key.turn,
                        key.event_id
                    ));
                }
                seen_groups.insert(group, key);
                schedule.entry(key).or_default().push(group);
            }
        }
    }
    Ok(schedule)
}

/// Go：`native_continue_pending_groups.go:259`，`func nativeContinueFieldSpawnRuleIsLive( state *battle.State, assetState *battle.State, rule battle.NativeFieldEventRule, ) bool`
pub(crate) fn native_continue_field_spawn_rule_is_live(
    state: Option<&battle::State>,
    asset_state: Option<&battle::State>,
    rule: &battle::NativeFieldEventRule,
) -> bool {
    let (Some(state), Some(asset_state)) = (state, asset_state) else { return false };
    if state.native_field_events.len() != asset_state.native_field_events.len() {
        return false;
    }
    let mut matched_slot: Option<usize> = None;
    for (slot, event) in asset_state.native_field_events.iter().enumerate() {
        if i64::from(event.event_id) != rule.event_id || event.selector != rule.selector {
            continue;
        }
        if matched_slot.is_some() || state.native_field_events[slot] != *event {
            return false;
        }
        matched_slot = Some(slot);
    }
    let Some(matched_slot) = matched_slot else { return false };
    asset_state.native_field_event_slots.contains(&(matched_slot as i64))
}

/// Go：`native_continue_pending_groups.go:289`，`func cloneNativeContinueFutureUnit(source *battle.Unit) (*battle.Unit, error)`
pub(crate) fn clone_native_continue_future_unit(source: Option<&Unit>) -> Result<Unit> {
    let Some(source) = source else {
        return Err(campaign_err!("nil FDFIELD roster row"));
    };
    // Go 逐一複製切片與指標欄位；Rust 的 clone 本來就是深複製。
    let clone = source.clone();
    if !clone.has_native_position_record
        || clone.native_constructor.is_none()
        || !clone.has_map_selector_key
        || !clone.has_battle_fig
        || !clone.has_native_record_byte6
        || !clone.has_native_record_byte34
        || !clone.has_native_record_byte35
        || !clone.has_native_record_byte36
    {
        return Err(campaign_err!("group {} FDFIELD row lacks native constructor provenance", clone.group));
    }
    Ok(clone)
}

#[cfg(test)]
pub(crate) mod native_continue_pending_groups_test;
