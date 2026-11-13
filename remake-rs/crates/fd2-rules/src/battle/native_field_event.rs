//! Go 來源：`remake/internal/battle/native_field_event.go`（471 行）。

use super::*;

/// Go：`native_field_event.go:5`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeFieldEvent75Plan {
    pub event_id: u8,
    pub text_index: i64,
    pub activate: bool,
    pub noop: bool,
    pub(crate) trigger: Option<UnitIdx>,
    pub(crate) rule: Option<NativeFieldEventRule>,
}

/// PlanNativeFieldEvent75 驗證 map28 selector1 handler，不修改 event-state table
/// 或 live turn row。raw +8 != 9 選 FDTXT_029 index0，執行期另以觸發 record +7
/// 呈現肖像；raw +8 == 9 選 index1，並在對話完成後才可啟動 dormant rows。
///
/// Go：`native_field_event.go:18`，`func PlanNativeFieldEvent75(st *State, trigger *Unit, x, y int) (NativeFieldEvent75Plan, error)`
/// 所有權：`trigger` 是 `st.units` 的索引（Go 傳 `*Unit`）。
pub fn plan_native_field_event75(
    st: Option<&State>,
    trigger: Option<UnitIdx>,
    x: i64,
    y: i64,
) -> Result<NativeFieldEvent75Plan> {
    let Some((st, trigger_idx, trigger)) =
        field_trigger(st, trigger).filter(|(st, _, t)| t.x == x && t.y == y && st.has_native_turn_event_control_state)
    else {
        return Err(battle_err!("event75: incomplete trigger or live turn controls"));
    };
    let (event_id, ok) = native_field_event_id_at(Some(st), x, y, 1);
    if !ok || event_id != 75 {
        return Err(battle_err!("event75: selector1 binding is absent"));
    }
    let rule = st.native_field_event_rules.iter().find(|c| c.event_id == 75 && c.selector == 1);
    let want_writes = [NativeEventStateWrite { index: 17, value: 1 }, NativeEventStateWrite { index: 16, value: 4 }];
    let want_turns = [
        NativeTurnActivation { slot: 1, event_id: 76, raw_camp: 2, turn_delta: 1 },
        NativeTurnActivation { slot: 0, event_id: 74, raw_camp: 0, turn_delta: 0 },
    ];
    let Some((rule, chain)) =
        rule.and_then(|rule| rule.turn_chain.as_ref().map(|chain| (rule, chain))).filter(|(rule, chain)| {
            rule.trigger_gate == "record_byte6_nonzero"
                && chain.handler == "0x35c79"
                && chain.trigger_record_byte8 == 9
                && chain.mismatch_text_index == 0
                && chain.success_text_index == 1
                && chain.state_writes.len() == want_writes.len()
                && chain.turn_activations.len() == want_turns.len()
        })
    else {
        return Err(battle_err!("event75: editable rule is incomplete"));
    };
    for (i, want) in want_writes.iter().enumerate() {
        if chain.state_writes[i] != *want {
            return Err(battle_err!("event75: state write {i} differs from handler"));
        }
    }
    for (i, want) in want_turns.iter().enumerate() {
        if chain.turn_activations[i] != *want {
            return Err(battle_err!("event75: turn activation {i} differs from handler"));
        }
    }
    if !trigger.has_native_record_byte6 || !trigger.has_native_record_byte8 {
        return Err(battle_err!("event75: raw +6/+8 provenance is absent"));
    }
    let mut plan =
        NativeFieldEvent75Plan { event_id, trigger: Some(trigger_idx), rule: Some(rule.clone()), ..Default::default() };
    if trigger.native_record_byte6 == 0 || st.native_event_state[17] != 0 {
        plan.noop = true;
        return Ok(plan);
    }
    if trigger.native_record_byte8 != chain.trigger_record_byte8 {
        plan.text_index = chain.mismatch_text_index;
        return Ok(plan);
    }
    if st.native_round_counter <= 0 || st.native_round_counter > 0xfe {
        return Err(battle_err!("event75: native round is unavailable"));
    }
    for activation in &chain.turn_activations {
        let row = st.native_turn_event_controls[activation.slot as usize];
        if row
            != (NativeTurnEventControl {
                turn: 0xff,
                event_id: activation.event_id as u8,
                raw_camp: activation.raw_camp,
            })
        {
            return Err(battle_err!("event75: dormant row {} identity mismatch", activation.slot));
        }
    }
    plan.text_index = chain.success_text_index;
    plan.activate = true;
    Ok(plan)
}

/// CommitNativeFieldEvent75 在 FDTXT_029 index1 完成後重驗 plan，再原子提交
/// event-state 與 typed live rows；只有 state 帶完整 raw field provenance 時才一併
/// 核對並更新該 raw view。
///
/// Go：`native_field_event.go:86`，`func CommitNativeFieldEvent75(st *State, plan NativeFieldEvent75Plan) error`
pub fn commit_native_field_event75(st: Option<&mut State>, plan: &NativeFieldEvent75Plan) -> Result<()> {
    let Some((st, trigger, rule)) = st.and_then(|st| {
        let trigger = plan.trigger.filter(|&t| t < st.units.len())?;
        let rule = plan.rule.as_ref().filter(|_| plan.activate)?;
        Some((st, trigger, rule))
    }) else {
        return Err(battle_err!("event75: activation plan is unavailable"));
    };
    let (tx, ty) = (st.units[trigger].x, st.units[trigger].y);
    let recheck = plan_native_field_event75(Some(st), Some(trigger), tx, ty);
    match &recheck {
        Ok(recheck) if recheck.activate && recheck.text_index == plan.text_index => {}
        _ => {
            let err = recheck.err().map_or_else(|| "<nil>".to_string(), |err| err.to_string());
            return Err(battle_err!("event75: activation changed before commit: {err}"));
        }
    }
    // Go 的 `plan.rule` 指向 State 內的規則；重驗剛確認過同一條規則仍在，這裡讀計畫保存的副本。
    let chain = rule.turn_chain.as_ref().expect("重驗已確認 turn_chain 存在");
    let mut candidate_state = st.native_event_state;
    let mut candidate_rows = st.native_turn_event_controls;
    let mut candidate_raw = st.native_field_control_raw.clone();
    for write in &chain.state_writes {
        candidate_state[write.index as usize] = write.value;
    }
    for activation in &chain.turn_activations {
        let turn = st.native_round_counter.wrapping_add(activation.turn_delta);
        if !(0..=0xfe).contains(&turn) {
            return Err(battle_err!("event75: scheduled turn is outside byte range"));
        }
        candidate_rows[activation.slot as usize].turn = turn as u8;
        if st.has_native_field_control_state {
            let offset = (3 + activation.slot * 3) as usize;
            if candidate_raw.len() <= offset + 2
                || candidate_raw[offset] != 0xff
                || candidate_raw[offset + 1] != activation.event_id as u8
                || candidate_raw[offset + 2] != activation.raw_camp
            {
                return Err(battle_err!("event75: raw row {} disagrees with typed controls", activation.slot));
            }
            candidate_raw[offset] = turn as u8;
        }
    }
    st.native_event_state = candidate_state;
    st.native_turn_event_controls = candidate_rows;
    if st.has_native_field_control_state {
        st.native_field_control_raw = candidate_raw;
    }
    Ok(())
}

/// NativeFieldEventIDAt 保存 0x13a44 的格子事件 selector：
/// 該格必須具有已驗證的 0..15 slot，event_id 不得為 0xff，且列內 selector
/// 必須等於 caller selector。資料缺失時失敗即關閉。
///
/// Go：`native_field_event.go:127`，`func NativeFieldEventIDAt( st *State, x, y int, selector byte, ) (byte, bool)`
pub fn native_field_event_id_at(st: Option<&State>, x: i64, y: i64, selector: u8) -> (u8, bool) {
    let Some(st) = st.filter(|st| {
        st.w > 0
            && st.h > 0
            && st.native_field_event_slots.len() as i64 == st.w.wrapping_mul(st.h)
            && st.native_field_events.len() == 16
            && x >= 0
            && x < st.w
            && y >= 0
            && y < st.h
    }) else {
        return (0, false);
    };
    let slot = st.native_field_event_slots[(y * st.w + x) as usize];
    if slot < 0 || slot >= st.native_field_events.len() as i64 {
        return (0, false);
    }
    let event = st.native_field_events[slot as usize];
    if event.event_id == 0xff || event.selector != selector {
        return (0, false);
    }
    (event.event_id, true)
}

/// ApplyNativeFieldModeEvent 執行目前已閉合的 mode-range 規則。
/// selector 的呼叫時機仍由上層保存；非 mode 規則（包含 event61）不在此猜測執行。
///
/// Go：`native_field_event.go:154`，`func ApplyNativeFieldModeEvent( st *State, trigger *Unit, x, y int, selector byte, ) (byte, bool)`
/// 所有權：`trigger` 是 `st.units` 的索引（Go 傳 `*Unit`）。
pub fn apply_native_field_mode_event(
    st: Option<&mut State>,
    trigger: Option<UnitIdx>,
    x: i64,
    y: i64,
    selector: u8,
) -> (u8, bool) {
    let (event_id, ok) = native_field_event_id_at(st.as_deref(), x, y, selector);
    let Some(st) = st.filter(|st| ok && trigger.is_some_and(|t| t < st.units.len())) else {
        return (0, false);
    };
    let trigger = trigger.expect("上方已檢查 trigger");
    let Some(rule_index) = st
        .native_field_event_rules
        .iter()
        .position(|rule| rule.event_id == event_id as i64 && rule.selector == selector)
        .filter(|&i| !st.native_field_event_rules[i].set_mode_ranges.is_empty())
    else {
        return (0, false);
    };
    let rule = &st.native_field_event_rules[rule_index];
    if rule.set_state_index.is_none() != rule.set_state_value.is_none()
        || rule.set_state_index.is_some_and(|index| {
            let value = rule.set_state_value.unwrap_or_default();
            index < 0 || index >= st.native_event_state.len() as i64 || !(0..=0xff).contains(&value)
        })
    {
        return (0, false);
    }
    match rule.trigger_gate.as_str() {
        "record_byte6_nonzero" => {
            let t = &st.units[trigger];
            if !t.has_native_record_byte6 || t.native_record_byte6 == 0 {
                return (0, false);
            }
        }
        "always" => {
            // 35F5A沒有actor gate；此新增契約只授權map29的80／selector1。
            if event_id != 80 || selector != 1 || rule.handler != "0x35f5a" {
                return (0, false);
            }
        }
        _ => return (0, false),
    }
    for mode_range in &rule.set_mode_ranges {
        if mode_range.start < 0 || mode_range.end >= st.units.len() as i64 {
            return (0, false);
        }
        for index in mode_range.start..=mode_range.end {
            if !st.units[index as usize].has_native_record_byte34 {
                return (0, false);
            }
        }
    }
    let rule = st.native_field_event_rules[rule_index].clone();
    for mode_range in &rule.set_mode_ranges {
        for index in mode_range.start..=mode_range.end {
            let u = &mut st.units[index as usize];
            u.native_record_byte34 = (u.native_record_byte34 & 0xF0) | mode_range.mode;
            let byte34 = u.native_record_byte34;
            if event_id == 80 {
                st.sync_native_runtime_record(index, 0x34, byte34 as i64, 1);
            }
        }
    }
    if let (Some(index), Some(value)) = (rule.set_state_index, rule.set_state_value) {
        st.native_event_state[index as usize] = value as u8;
    }
    (event_id, true)
}

/// ApplyNativeFieldTurnActivationEvent 執行已閉合的 event62 mutation：selector0
/// 對到 event62 時，把休眠 row0（event63/raw camp0）排到 native round+1，並
/// 設定 battle-local state byte17。editable rule 與完整 raw row 必須先一致，
/// 才能改動任一值。
///
/// Go：`native_field_event.go:223`，`func ApplyNativeFieldTurnActivationEvent( st *State, x, y int, selector byte, ) (byte, error)`
pub fn apply_native_field_turn_activation_event(st: Option<&mut State>, x: i64, y: i64, selector: u8) -> Result<u8> {
    let Some(st) = st.filter(|st| st.has_native_turn_event_control_state) else {
        return Err(battle_err!("turn activation: complete native turn controls are absent"));
    };
    let (event_id, ok) = native_field_event_id_at(Some(st), x, y, selector);
    if !ok || event_id != 62 || selector != 0 {
        return Err(battle_err!("turn activation: event62 selector0 binding is absent"));
    }
    let want = NativeTurnActivation { slot: 0, event_id: 63, raw_camp: 0, turn_delta: 1 };
    let Some(rule) = st
        .native_field_event_rules
        .iter()
        .find(|rule| rule.event_id == event_id as i64 && rule.selector == selector)
        .filter(|rule| rule.once_state == Some(17) && rule.turn_activation == Some(want))
        .cloned()
    else {
        return Err(battle_err!("turn activation: editable event62 rule is incomplete"));
    };
    // 0x35898的已啟用guard直接返回；第27章同狀態原版多次跨線後仍繼續。
    // READY勘誤見fd2_ch27_flow_20261016.json event62_repeat_contract。
    if st.native_event_state[17] != 0 {
        return Ok(event_id);
    }
    if st.native_round_counter <= 0 || st.native_round_counter > 0xfe {
        return Err(battle_err!("turn activation: state or native round is not activatable"));
    }
    commit_native_field_turn_activation(st, event_id, &rule)
}

/// ApplyNativeChapter28FieldTurnActivation 保存3599B／35AB8的區域啟動。
/// 65不讀觸發者陣營；69先讀觸發者raw +6，再查state17／18。
/// READY與原始定位見fd2_ch28_flow_20261016.json。
///
/// Go：`native_field_event.go:264`，`func ApplyNativeChapter28FieldTurnActivation(st *State, trigger *Unit, x, y int, selector byte) (byte, error)`
/// 所有權：`trigger` 是 `st.units` 的索引（Go 傳 `*Unit`）。
pub fn apply_native_chapter28_field_turn_activation(
    st: Option<&mut State>,
    trigger: Option<UnitIdx>,
    x: i64,
    y: i64,
    selector: u8,
) -> Result<u8> {
    let Some(st) = st.filter(|st| st.has_native_turn_event_control_state) else {
        return Err(battle_err!("chapter28 activation: native turn controls are absent"));
    };
    let (event_id, ok) = native_field_event_id_at(Some(st), x, y, selector);
    if !ok || (event_id != 65 && event_id != 69) || selector != 0 {
        return Err(battle_err!("chapter28 activation: selector0 binding is absent"));
    }
    let (mut want_state, mut want_slot, mut want_event) = (16, 0, 66);
    if event_id == 69 {
        (want_state, want_slot, want_event) = (17, 2, 70);
    }
    let want = NativeTurnActivation { slot: want_slot, event_id: want_event, raw_camp: 0, turn_delta: 0 };
    let Some(rule) = st
        .native_field_event_rules
        .iter()
        .find(|rule| rule.event_id == event_id as i64 && rule.selector == selector)
        .filter(|rule| rule.once_state == Some(want_state) && rule.turn_activation == Some(want))
        .cloned()
    else {
        return Err(battle_err!("chapter28 activation: editable rule is incomplete"));
    };
    if event_id == 65 {
        if !rule.trigger_gate.is_empty() || rule.required_nonzero_state.is_some() {
            return Err(battle_err!("event65: unexpected trigger guard"));
        }
    } else {
        // Go 的 trigger 是 nil 或不在 `st.units` 時同樣拒絕。
        let trigger_unit = trigger.and_then(|t| st.units.get(t));
        let Some(trigger_unit) = trigger_unit.filter(|t| {
            rule.trigger_gate == "record_byte6_nonzero"
                && rule.required_nonzero_state == Some(18)
                && t.has_native_record_byte6
        }) else {
            return Err(battle_err!("event69: raw trigger or state guard is absent"));
        };
        if trigger_unit.native_record_byte6 == 0 {
            return Ok(event_id);
        }
    }
    if st.native_event_state[want_state as usize] != 0
        || rule.required_nonzero_state.is_some_and(|index| st.native_event_state[index as usize] == 0)
    {
        return Ok(event_id);
    }
    commit_native_field_turn_activation(st, event_id, &rule)
}

/// Go：`native_field_event.go:310`，`func commitNativeFieldTurnActivation(st *State, eventID byte, rule *NativeFieldEventRule) (byte, error)`
pub(crate) fn commit_native_field_turn_activation(
    st: &mut State,
    event_id: u8,
    rule: &NativeFieldEventRule,
) -> Result<u8> {
    if st.native_round_counter <= 0 || st.native_round_counter > 0xfe {
        return Err(battle_err!("turn activation: native round is unavailable"));
    }
    let activation = rule.turn_activation.expect("呼叫端已驗證 turn_activation");
    if activation.slot < 0 || activation.slot >= st.native_turn_event_controls.len() as i64 {
        return Err(battle_err!("turn activation: row slot is out of range"));
    }
    let row = st.native_turn_event_controls[activation.slot as usize];
    if row
        != (NativeTurnEventControl { turn: 0xff, event_id: activation.event_id as u8, raw_camp: activation.raw_camp })
    {
        return Err(battle_err!("turn activation: dormant row identity mismatch"));
    }
    let raw_offset = (3 + activation.slot * 3) as usize;
    if st.has_native_field_control_state
        && (st.native_field_control_raw.len() <= raw_offset + 2
            || st.native_field_control_raw[raw_offset] != row.turn
            || st.native_field_control_raw[raw_offset + 1] != row.event_id
            || st.native_field_control_raw[raw_offset + 2] != row.raw_camp)
    {
        return Err(battle_err!("turn activation: live raw row disagrees with typed controls"));
    }
    let turn = st.native_round_counter.wrapping_add(activation.turn_delta) as u8;
    st.native_turn_event_controls[activation.slot as usize].turn = turn;
    if st.has_native_field_control_state {
        st.native_field_control_raw[raw_offset] = turn;
    }
    st.native_event_state[rule.once_state.expect("呼叫端已驗證 once_state") as usize] = 1;
    Ok(event_id)
}

/// Go：`native_field_event.go:342`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeFieldEvent61Plan {
    pub event_id: u8,
    pub missing_item: bool,
    pub text_index: i64,
    pub final_text: i64,
    pub presentation: NativeFieldPresentation,
    pub join_character: i64,
    pub(crate) trigger: Option<UnitIdx>,
    pub(crate) item_index: i64,
    pub(crate) state_index: i64,
    pub(crate) spawn_group: i64,
    pub(crate) required_frames: i64,
}

/// PlanNativeFieldEvent61 validates the complete editable/native boundary
/// before the UI starts text #3 and the 59-frame presentation. Missing item is
/// a valid, non-mutating native outcome; malformed provenance fails closed.
///
/// Go：`native_field_event.go:360`，`func PlanNativeFieldEvent61( st *State, trigger *Unit, x, y int, ) (NativeFieldEvent61Plan, error)`
/// 所有權：`trigger` 是 `st.units` 的索引（Go 傳 `*Unit`）。
pub fn plan_native_field_event61(
    st: Option<&State>,
    trigger: Option<UnitIdx>,
    x: i64,
    y: i64,
) -> Result<NativeFieldEvent61Plan> {
    let Some((st, trigger_idx, trigger)) = field_trigger(st, trigger).filter(|(_, _, t)| t.x == x && t.y == y) else {
        return Err(battle_err!("event61: invalid trigger"));
    };
    let (event_id, ok) = native_field_event_id_at(Some(st), x, y, 1);
    if !ok || event_id != 61 {
        return Err(battle_err!("event61: selector1 binding is absent"));
    }
    let want_presentation = NativeFieldPresentation {
        archive: "FDOTHER.DAT".into(),
        resource: 45,
        frames: 59,
        helper: "0x2935b".into(),
        destination_offset: 48356,
        stride: 320,
        transparent: -1,
        delay_helper: "0x17aa9".into(),
        delay_ticks: 2,
    };
    let Some(rule) = st.native_field_event_rules.iter().find(|r| r.event_id == 61 && r.selector == 1).filter(|rule| {
        rule.once_state == Some(12)
            && rule.required_item == Some(0xD0)
            && rule.consume_item
            && rule.spawn_group == Some(1)
            && rule.join_character == Some(31)
            && rule.text_indices == Some(NativeFieldTextIndices { missing_item: 2, success: 3, r#final: 4 })
            && rule.presentation.as_ref() == Some(&want_presentation)
    }) else {
        return Err(battle_err!("event61: editable rule is incomplete"));
    };
    let once_state = rule.once_state.expect("上方已檢查");
    let required_item = rule.required_item.expect("上方已檢查");
    let text_indices = rule.text_indices.expect("上方已檢查");
    let presentation = rule.presentation.clone().expect("上方已檢查");
    if st.native_event_state[once_state as usize] != 0 {
        return Err(battle_err!("event61: once state is already set"));
    }
    if let Err(err) = validate_native_inventory_projection(Some(trigger)) {
        return Err(battle_err!("event61: {err}"));
    }
    let item_index = trigger.inventory.iter().position(|&item| item == required_item).map_or(-1, |i| i as i64);
    let mut plan = NativeFieldEvent61Plan {
        event_id: 61,
        missing_item: item_index < 0,
        text_index: text_indices.success,
        final_text: text_indices.r#final,
        required_frames: presentation.frames,
        presentation,
        join_character: rule.join_character.expect("上方已檢查"),
        trigger: Some(trigger_idx),
        item_index,
        state_index: once_state,
        spawn_group: rule.spawn_group.expect("上方已檢查"),
    };
    if item_index < 0 {
        plan.text_index = text_indices.missing_item;
        return Ok(plan);
    }
    let mut pending = 0;
    for unit in &st.roster {
        if unit.group == plan.spawn_group {
            if unit.fig != plan.join_character || unit.camp != OWN {
                return Err(battle_err!("event61: pending group identity mismatch"));
            }
            pending += 1;
        }
    }
    if pending != 1 {
        return Err(battle_err!("event61: pending group cardinality is {pending}"));
    }
    Ok(plan)
}

/// CommitNativeFieldEvent61 applies only the mutations after the caller has
/// presented all native frames. It revalidates the plan so cancellation or an
/// intervening inventory change cannot consume an item or partially set state.
/// Persistent JOIN is intentionally returned to the campaign owner.
///
/// Go：`native_field_event.go:436`，`func CommitNativeFieldEvent61( st *State, plan NativeFieldEvent61Plan, presentedFrames int, ) (int, error)`
pub fn commit_native_field_event61(
    st: Option<&mut State>,
    plan: &NativeFieldEvent61Plan,
    presented_frames: i64,
) -> Result<i64> {
    let Some((st, trigger)) = st.and_then(|st| {
        let trigger = plan.trigger.filter(|&t| t < st.units.len())?;
        let ok = !plan.missing_item
            && presented_frames == plan.required_frames
            && plan.state_index == 12
            && st.native_event_state[plan.state_index as usize] == 0;
        ok.then_some((st, trigger))
    }) else {
        return Err(battle_err!("event61: plan is not committable"));
    };
    let unit = &st.units[trigger];
    if validate_native_inventory_projection(Some(unit)).is_err()
        || plan.item_index < 0
        || plan.item_index >= unit.inventory.len() as i64
        || unit.inventory[plan.item_index as usize] != 0xD0
    {
        return Err(battle_err!("event61: trigger inventory changed"));
    }
    let pending = st
        .roster
        .iter()
        .filter(|u| u.group == plan.spawn_group && u.fig == plan.join_character && u.camp == OWN)
        .count();
    if pending != 1 {
        return Err(battle_err!("event61: pending group changed"));
    }
    remove_native_compact_inventory(Some(&mut st.units[trigger]), plan.item_index)?;
    st.native_event_state[plan.state_index as usize] = 1;
    if st.append_group(plan.spawn_group) != 1 {
        // Preflight made this unreachable for the current representation. Keep
        // the failure explicit instead of claiming campaign JOIN succeeded.
        return Err(battle_err!("event61: group append changed after preflight"));
    }
    Ok(plan.join_character)
}

/// 取出觸發單位：Go 的 `trigger *Unit` 在 Rust 是 `st.units` 的索引；`st` 為 nil、索引為 `None`
/// 或越界都對應 Go 的 nil 指標。
fn field_trigger(st: Option<&State>, trigger: Option<UnitIdx>) -> Option<(&State, UnitIdx, &Unit)> {
    let st = st?;
    let idx = trigger?;
    st.units.get(idx).map(|u| (st, idx, u))
}

#[cfg(test)]
mod native_field_event_test;
