//! Go 來源：`remake/internal/battle/native_ai_scored_candidates.go`（160 行）。

use super::*;

/// NativeAIScoredCommandCandidateGroup is one 0x1598a destination followed by
/// the record indices emitted by its 0x14818 call. Groups retain 0x14b16's
/// row-major destination order and the runtime roster's target order.
///
/// Go：`native_ai_scored_candidates.go:8`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAIScoredCommandCandidateGroup {
    pub destination: Cell,
    pub target_indices: Vec<u8>,
}

/// NativeAIScoredCommandCandidateGroups preserves 0x1598a's geometry pipeline:
/// command record +3 -> 0x4e040 -> 0x14b16, then command record +4 -> 0x14818.
/// Unlike 0x14237's physical path, this caller does not invoke 0x145cd/0x146d1
/// to derive occupancy flags from the roster; baseFlags must therefore be the
/// exact caller-owned grid snapshot.
///
/// Go：`native_ai_scored_candidates.go:18`，`func NativeAIScoredCommandCandidateGroups( w, h int, records []byte, count, actor, selector int, command NativeCommandRecord, baseFlags, terrainMoveCodes, costRow []byte, ) ([]NativeAIScoredCommandCandidateGroup, error)`
pub fn native_ai_scored_command_candidate_groups(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
    command: NativeCommandRecord,
    base_flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<Vec<NativeAIScoredCommandCandidateGroup>> {
    let destinations = native_ai_scored_command_destinations(
        w,
        h,
        records,
        count,
        actor,
        command.selection_mode,
        base_flags,
        terrain_move_codes,
        cost_row,
    )?;
    let target_code = native_ai_scored_command_target_code(command.target_code, selector)?;
    let mut groups = Vec::with_capacity(destinations.len());
    for destination in destinations {
        let targets = native_ai_scored_command_target_indices(
            w,
            h,
            records,
            count,
            destination,
            command.effect_mode,
            target_code,
            base_flags,
        )?;
        if targets.is_empty() {
            continue;
        }
        groups.push(NativeAIScoredCommandCandidateGroup { destination, target_indices: targets });
    }
    Ok(groups)
}

/// Go：`native_ai_scored_candidates.go:55`，`func nativeAIScoredCommandDestinations( w, h int, records []byte, count, actor, initialBudget int, baseFlags, terrainMoveCodes, costRow []byte, ) ([]Cell, error)`
pub(crate) fn native_ai_scored_command_destinations(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    initial_budget: i64,
    base_flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<Vec<Cell>> {
    if w <= 0
        || h <= 0
        || base_flags.len() as i64 != w.wrapping_mul(h)
        || terrain_move_codes.len() as i64 != w.wrapping_mul(h)
    {
        return Err(battle_err!("native AI scored destination grid is malformed"));
    }
    if !(0..=0x100).contains(&count)
        || records.len() as i64 != count * NATIVE_RECORD_SIZE
        || actor < 0
        || actor >= count
    {
        return Err(battle_err!("native AI scored destination roster/actor is malformed"));
    }
    if !(0..=0xff).contains(&initial_budget) || cost_row.len() as i64 != NATIVE_MOVEMENT_COST_ROW_SIZE {
        return Err(battle_err!("native AI scored destination movement inputs are malformed"));
    }
    for (cell, &code) in terrain_move_codes.iter().enumerate() {
        if code as usize >= cost_row.len() {
            return Err(battle_err!(
                "native AI scored destination terrain code {code} at cell {cell} is out of bounds"
            ));
        }
    }
    let actor_record = &records[(actor * NATIVE_RECORD_SIZE) as usize..];
    let origin = Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 };
    if origin.x < 0 || origin.y < 0 || origin.x >= w || origin.y >= h {
        return Err(battle_err!("native AI scored destination actor is outside the grid"));
    }
    let mut field = vec![0xffu8; (w * h) as usize];
    field[(origin.y * w + origin.x) as usize] = initial_budget as u8;
    native_ai_propagate_destination_budget(
        w,
        h,
        origin,
        initial_budget as u8,
        base_flags,
        terrain_move_codes,
        cost_row,
        &mut field,
    );
    let mut result = Vec::new();
    for y in 0..h {
        for x in 0..w {
            if field[(y * w + x) as usize] != 0xff {
                result.push(Cell { x, y });
            }
        }
    }
    Ok(result)
}

/// Go：`native_ai_scored_candidates.go:104`，`func nativeAIScoredCommandTargetCode(commandTargetCode, selector int) (int, error)`
pub(crate) fn native_ai_scored_command_target_code(command_target_code: i64, selector: i64) -> Result<i64> {
    if !(0..=0xff).contains(&command_target_code) || (selector != 0 && selector != 1) {
        return Err(battle_err!("native AI scored target selector is malformed"));
    }
    if selector != 0 {
        return Ok(command_target_code);
    }
    if command_target_code == 0 {
        return Ok(1);
    }
    Ok(0)
}

/// Go：`native_ai_scored_candidates.go:118`，`func nativeAIScoredCommandTargetIndices( w, h int, records []byte, count int, origin Cell, effectMode, targetCode int, baseFlags []byte, ) ([]byte, error)`
pub(crate) fn native_ai_scored_command_target_indices(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    origin: Cell,
    effect_mode: i64,
    target_code: i64,
    base_flags: &[u8],
) -> Result<Vec<u8>> {
    if !(0..=0x100).contains(&count) || records.len() as i64 != count * NATIVE_RECORD_SIZE {
        return Err(battle_err!("native AI scored target roster is malformed"));
    }
    let cells = native_command_target_cells(w, h, origin, effect_mode, base_flags)?;
    let mut result = Vec::new();
    for index in 0..count {
        let record = &records[(index * NATIVE_RECORD_SIZE) as usize..];
        if record[5] & 1 != 0 || !native_ai_scored_raw_target_matches(target_code, record[6]) {
            continue;
        }
        let cell = Cell { x: record[0] as i64, y: record[1] as i64 };
        if cells.get(&cell).copied().unwrap_or(false) {
            result.push(index as u8);
        }
    }
    Ok(result)
}

/// Go：`native_ai_scored_candidates.go:147`，`func nativeAIScoredRawTargetMatches(targetCode int, recordByte6 byte) bool`
pub(crate) fn native_ai_scored_raw_target_matches(target_code: i64, record_byte6: u8) -> bool {
    match target_code {
        0 => record_byte6 == 0,
        1 => record_byte6 != 0,
        2 => record_byte6 == 1,
        3 => record_byte6 == 2,
        _ => false,
    }
}

#[cfg(test)]
mod native_ai_scored_candidates_test;
