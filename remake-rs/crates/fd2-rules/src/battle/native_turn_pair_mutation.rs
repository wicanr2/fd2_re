//! Go 來源：`remake/internal/battle/native_turn_pair_mutation.go`（87 行）。

use super::*;

/// Go：`native_turn_pair_mutation.go:10`，`func validateNativeTurnEvent79(event NativeTurnEvent) error`
pub(crate) fn validate_native_turn_event79(event: &NativeTurnEvent) -> Result<()> {
    let valid = event.event_id == 79
        && event.raw_camp == 0
        && event.handler.eq_ignore_ascii_case("0x35ee6")
        && event.pair_mutation.as_ref().is_some_and(|p| {
            p.base_state_index == 21
                && p.group == 1
                && p.count == 3
                && p.modulo == 3
                && p.second_offset == 1
                && p.mark_helper.eq_ignore_ascii_case("0x13512")
                && p.control_slot == 2
                && p.reschedule_delta == 1
        });
    if !valid {
        return Err(battle_err!("event79: editable pair mutation differs from recovered handler"));
    }
    Ok(())
}

/// ApplyNativeTurnEvent79 驗證並一次提交row2、兩個raw +5 bit7與process-wide RNG。
/// RNG由caller持有；錯誤時回傳原值，State保持不變。
///
/// Go：`native_turn_pair_mutation.go:23`，`func ApplyNativeTurnEvent79(st *State, event NativeTurnEvent, rngState uint16) (uint16, [2]int, error)`
pub fn apply_native_turn_event79(
    st: Option<&mut State>,
    event: &NativeTurnEvent,
    rng_state: u16,
) -> Result<(u16, [i64; 2])> {
    // Go 失敗時回傳 (rngState, [2]int{}, err)：RNG 原值由呼叫端保留，Rust 只回錯誤。
    let Some(st) = st.filter(|st| {
        st.has_native_turn_event_control_state && st.native_round_counter > 0 && st.native_round_counter <= 0xfe
    }) else {
        return Err(battle_err!("event79: native round or live controls unavailable"));
    };
    validate_native_turn_event79(event)?;
    let p = event.pair_mutation.as_ref().expect("驗證已確認 pair_mutation");
    let row = st.native_turn_event_controls[p.control_slot as usize];
    if row != (NativeTurnEventControl { turn: st.native_round_counter as u8, event_id: 79, raw_camp: 0 }) {
        return Err(battle_err!("event79: live row identity mismatch"));
    }
    let base = st.native_event_state[p.base_state_index as usize] as i64;
    if base < 0 || base + p.count > st.units.len() as i64 {
        return Err(battle_err!("event79: group1 base {base} outside runtime"));
    }
    for index in base..base + p.count {
        let unit = &st.units[index as usize];
        if unit.group != p.group || !unit.has_native_record_byte5 {
            return Err(battle_err!("event79: slot{index} lacks group1 raw +5 provenance"));
        }
        if st.has_native_runtime_unit_projection
            && (st.native_runtime_records.len() != st.units.len()
                || st.native_runtime_records[index as usize].raw[5] != unit.native_record_byte5)
        {
            return Err(battle_err!("event79: saved raw slot{index} disagrees with typed unit"));
        }
    }
    let next_rng = fdother::native_rng_step(rng_state);
    let targets = [base + (next_rng % p.modulo as u16) as i64, base + (next_rng as i64 + p.second_offset) % p.modulo];
    let next_round = st.native_round_counter + p.reschedule_delta;
    if next_round <= 0 || next_round > 0xfe || targets[0] == targets[1] {
        return Err(battle_err!("event79: next row or pair is invalid"));
    }
    let mut candidate_rows = st.native_turn_event_controls;
    candidate_rows[p.control_slot as usize].turn = next_round as u8;
    let mut candidate_field = st.native_field_control_raw.clone();
    let mut candidate_runtime = st.native_runtime_records.clone();
    if st.has_native_field_control_state {
        let offset = (3 + p.control_slot * 3) as usize;
        if candidate_field.len() <= offset + 2
            || candidate_field[offset] != row.turn
            || candidate_field[offset + 1] != row.event_id
            || candidate_field[offset + 2] != row.raw_camp
        {
            return Err(battle_err!("event79: raw field row disagrees with typed controls"));
        }
        candidate_field[offset] = next_round as u8;
    }
    for &target in &targets {
        if st.has_native_runtime_unit_projection {
            candidate_runtime[target as usize].raw[5] |= 0x80;
        }
    }
    for &target in &targets {
        st.units[target as usize].native_record_byte5 |= 0x80;
    }
    st.native_turn_event_controls = candidate_rows;
    if st.has_native_field_control_state {
        st.native_field_control_raw = candidate_field;
    }
    if st.has_native_runtime_unit_projection {
        st.native_runtime_records = candidate_runtime;
    }
    Ok((next_rng, targets))
}

#[cfg(test)]
mod native_turn_pair_mutation_test;
