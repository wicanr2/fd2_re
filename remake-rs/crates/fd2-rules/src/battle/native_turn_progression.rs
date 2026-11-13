//! Go 來源：`remake/internal/battle/native_turn_progression.go`（109 行）。

use super::*;

/// Go：`native_turn_progression.go:8`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeTurnEvent76Plan {
    pub r#final: bool,
    pub(crate) event: NativeTurnEvent,
}

/// Go：`native_turn_progression.go:14`，`func validateNativeTurnEvent76(event NativeTurnEvent) error`
pub(crate) fn validate_native_turn_event76(event: &NativeTurnEvent) -> Result<()> {
    let valid = event.event_id == 76
        && event.raw_camp == 2
        && event.handler.eq_ignore_ascii_case("0x35d60")
        && event.progression.as_ref().is_some_and(|p| {
            p.state_index == 17
                && p.repeat_until == 4
                && p.mark_unit_index == 1
                && p.mark_helper.eq_ignore_ascii_case("0x13512")
                && p.control_slot == 1
                && p.reschedule_delta == 1
                && p.final_text_index == 2
                && p.spawn_group == 1
                && p.spawn_helper.eq_ignore_ascii_case("0x10b4e")
                && p.raw_placement_gate == 0
                && p.spawn_count == 3
                && p.base_state_index == 21
                && p.final_activation == NativeTurnActivation { slot: 2, event_id: 79, raw_camp: 0, turn_delta: 0 }
                && p.pulse_handler.eq_ignore_ascii_case("0x35e5a")
                && p.pulse_count == 6
                && p.extra_delay_ms == 400
                && p.tail_text_indices == [3, 4, 5, 6]
        });
    if !valid {
        return Err(battle_err!("event76: editable progression differs from recovered handler"));
    }
    Ok(())
}

/// PlanNativeTurnEvent76 驗證 raw-camp2 的 live row 與 state byte，不修改單位、
/// 排程或 roster。state17 小於4走 repeat branch；等於4才進 final branch。
///
/// Go：`native_turn_progression.go:32`，`func PlanNativeTurnEvent76(st *State, event NativeTurnEvent) (NativeTurnEvent76Plan, error)`
pub fn plan_native_turn_event76(st: Option<&State>, event: &NativeTurnEvent) -> Result<NativeTurnEvent76Plan> {
    let Some(st) = st.filter(|st| {
        st.has_native_turn_event_control_state && st.native_round_counter > 0 && st.native_round_counter <= 0xfe
    }) else {
        return Err(battle_err!("event76: native round or live controls unavailable"));
    };
    validate_native_turn_event76(event)?;
    let p = event.progression.as_ref().expect("驗證已確認 progression");
    let row = st.native_turn_event_controls[p.control_slot as usize];
    if row != (NativeTurnEventControl { turn: st.native_round_counter as u8, event_id: 76, raw_camp: 2 }) {
        return Err(battle_err!("event76: live row identity mismatch"));
    }
    let progress = st.native_event_state[p.state_index as usize] as i64;
    if progress < 0 || progress > p.repeat_until {
        return Err(battle_err!("event76: state progress {progress} outside recovered range"));
    }
    if progress < p.repeat_until {
        let mark = p.mark_unit_index as usize;
        if mark >= st.units.len() || !st.units[mark].has_native_record_byte5 {
            return Err(battle_err!("event76: slot1 raw +5 provenance unavailable"));
        }
        if st.has_native_runtime_unit_projection
            && (st.native_runtime_records.len() != st.units.len()
                || st.native_runtime_records[mark].raw[5] != st.units[mark].native_record_byte5)
        {
            return Err(battle_err!("event76: saved raw slot1 disagrees with typed unit"));
        }
    }
    Ok(NativeTurnEvent76Plan { r#final: progress == p.repeat_until, event: event.clone() })
}

/// CommitNativeTurnEvent76Repeat 重驗 repeat plan，並原子提交 slot1 raw +5 bit7、
/// state17 increment及row1下一回合；若存在CONTINUE raw views則同步雙寫。
///
/// Go：`native_turn_progression.go:66`，`func CommitNativeTurnEvent76Repeat(st *State, plan NativeTurnEvent76Plan) error`
pub fn commit_native_turn_event76_repeat(st: Option<&mut State>, plan: &NativeTurnEvent76Plan) -> Result<()> {
    let Some((st, p)) = st.and_then(|st| plan.event.progression.as_ref().filter(|_| !plan.r#final).map(|p| (st, p)))
    else {
        return Err(battle_err!("event76: repeat plan unavailable"));
    };
    match plan_native_turn_event76(Some(st), &plan.event) {
        Ok(recheck) if !recheck.r#final => {}
        recheck => {
            let err = recheck.err().map_or_else(|| "<nil>".to_string(), |err| err.to_string());
            return Err(battle_err!("event76: repeat plan changed before commit: {err}"));
        }
    }
    let next = st.native_round_counter + p.reschedule_delta;
    if next <= 0 || next > 0xfe {
        return Err(battle_err!("event76: rescheduled turn outside byte range"));
    }
    let mark = p.mark_unit_index as usize;
    let candidate_byte5 = st.units[mark].native_record_byte5 | 0x80;
    let mut candidate_state = st.native_event_state;
    let mut candidate_rows = st.native_turn_event_controls;
    let mut candidate_raw_field = st.native_field_control_raw.clone();
    let mut candidate_runtime = st.native_runtime_records.clone();
    candidate_state[p.state_index as usize] = candidate_state[p.state_index as usize].wrapping_add(1);
    candidate_rows[p.control_slot as usize].turn = next as u8;
    if st.has_native_field_control_state {
        let offset = (3 + p.control_slot * 3) as usize;
        let row = st.native_turn_event_controls[p.control_slot as usize];
        if candidate_raw_field.len() <= offset + 2
            || candidate_raw_field[offset] != row.turn
            || candidate_raw_field[offset + 1] != row.event_id
            || candidate_raw_field[offset + 2] != row.raw_camp
        {
            return Err(battle_err!("event76: raw field row disagrees with typed controls"));
        }
        candidate_raw_field[offset] = next as u8;
    }
    if st.has_native_runtime_unit_projection {
        candidate_runtime[mark].raw[5] = candidate_byte5;
    }
    st.units[mark].native_record_byte5 = candidate_byte5;
    st.native_event_state = candidate_state;
    st.native_turn_event_controls = candidate_rows;
    if st.has_native_field_control_state {
        st.native_field_control_raw = candidate_raw_field;
    }
    if st.has_native_runtime_unit_projection {
        st.native_runtime_records = candidate_runtime;
    }
    Ok(())
}
