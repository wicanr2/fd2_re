//! Go 來源：`remake/internal/battle/native_ai_destination.go`（169 行）。

use super::*;

/// NativeAIPhysicalDestinations reproduces the destination-grid portion of
/// FD2.EXE 0x14237:
///
///     0x145cd -> 0x4e040 -> 0x146d1 -> 0x14b16
///
/// selector is kept raw.  Zero and nonzero select opposite record+6 groups;
/// this function does not rename that caller-owned distinction as a camp or
/// phase.  The returned cells follow 0x14b16's stable row-major order.
///
/// Go：`native_ai_destination.go:13`，`func NativeAIPhysicalDestinations( w, h int, records []byte, count, actor, selector, initialBudget int, baseFlags, terrainMoveCodes, costRow []byte, ) ([]Cell, error)`
pub fn native_ai_physical_destinations(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
    initial_budget: i64,
    base_flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<Vec<Cell>> {
    let field = native_movement_destination_field(
        w,
        h,
        records,
        count,
        actor,
        selector,
        initial_budget,
        base_flags,
        terrain_move_codes,
        cost_row,
    )?;
    let mut result = Vec::new();
    for (index, &value) in field.iter().enumerate() {
        if value != 0xff {
            result.push(Cell { x: index as i64 % w, y: index as i64 / w });
        }
    }
    Ok(result)
}

/// nativeMovementDestinationField 保留 0x4E040 的剩餘預算，供玩家地形繪圖與 AI 共用。
/// 它是 0x145CD→0x4E040→0x146D1 的組合：預算場之後再把同組占位清成 0xff。
///
/// Go：`native_ai_destination.go:34`，`func nativeMovementDestinationField( w, h int, records []byte, count, actor, selector, initialBudget int, baseFlags, terrainMoveCodes, costRow []byte, ) ([]byte, error)`
pub(crate) fn native_movement_destination_field(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
    initial_budget: i64,
    base_flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<Vec<u8>> {
    let mut field = native_movement_budget_field(
        w,
        h,
        records,
        count,
        actor,
        selector,
        initial_budget,
        base_flags,
        terrain_move_codes,
        cost_row,
    )?;
    // 0x146d1 removes occupied cells in the selector's own group after the
    // flood-fill.  The actor itself is explicitly skipped.
    for unit in 0..count {
        if unit == actor {
            continue;
        }
        let record = &records[(unit * NATIVE_RECORD_SIZE) as usize..];
        if record[5] & 1 != 0 || !native_ai_same_selector_group(record[6], selector) {
            continue;
        }
        let (x, y) = (record[0] as i64, record[1] as i64);
        if x < 0 || y < 0 || x >= w || y >= h {
            return Err(battle_err!("native AI destination unit {unit} is outside the grid"));
        }
        field[(y * w + x) as usize] = 0xff;
    }
    Ok(field)
}

/// nativeMovementBudgetField 是 0x145CD（對方群組 0x40／鄰格 0x80）加 0x4E040
/// 預算傳播的結果，不含 0x146D1 的同組占位清除；0x14B78 沿長路徑找「走得到的
/// 最遠一格」用的就是這一版（同組占位格在那一步仍算走得到）。
///
/// Go：`native_ai_destination.go:64`，`func nativeMovementBudgetField( w, h int, records []byte, count, actor, selector, initialBudget int, baseFlags, terrainMoveCodes, costRow []byte, ) ([]byte, error)`
pub(crate) fn native_movement_budget_field(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
    initial_budget: i64,
    base_flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<Vec<u8>> {
    let cells = w.wrapping_mul(h);
    if w <= 0 || h <= 0 || base_flags.len() as i64 != cells || terrain_move_codes.len() as i64 != cells {
        return Err(battle_err!("native AI destination grid is malformed"));
    }
    if count < 0 || count > records.len() as i64 / NATIVE_RECORD_SIZE || actor < 0 || actor >= count {
        return Err(battle_err!("native AI destination roster/actor is malformed"));
    }
    if !(0..=0xff).contains(&initial_budget) || cost_row.len() as i64 != NATIVE_MOVEMENT_COST_ROW_SIZE {
        return Err(battle_err!("native AI destination movement inputs are malformed"));
    }
    for (cell, &code) in terrain_move_codes.iter().enumerate() {
        if code as usize >= cost_row.len() {
            return Err(battle_err!("native AI destination terrain code {code} at cell {cell} is out of bounds"));
        }
    }

    let mut flags = base_flags.to_vec();
    for unit in 0..count {
        let record = &records[(unit * NATIVE_RECORD_SIZE) as usize..];
        if record[5] & 1 != 0 || !native_ai_opposite_selector_group(record[6], selector) {
            continue;
        }
        let (x, y) = (record[0] as i64, record[1] as i64);
        if x < 0 || y < 0 || x >= w || y >= h {
            return Err(battle_err!("native AI destination unit {unit} is outside the grid"));
        }
        flags[(y * w + x) as usize] |= NATIVE_COMMAND_GRID_BLOCKED;
        for delta in NATIVE_AI_DESTINATION_NEIGHBOURS {
            let (nx, ny) = (x + delta[0], y + delta[1]);
            if nx >= 0 && ny >= 0 && nx < w && ny < h {
                flags[(ny * w + nx) as usize] |= NATIVE_COMMAND_GRID_ZERO_BUDGET;
            }
        }
    }

    let mut field = vec![0xffu8; cells as usize];
    let actor_record = &records[(actor * NATIVE_RECORD_SIZE) as usize..];
    let origin = Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 };
    if origin.x < 0 || origin.y < 0 || origin.x >= w || origin.y >= h {
        return Err(battle_err!("native AI destination actor is outside the grid"));
    }
    field[(origin.y * w + origin.x) as usize] = initial_budget as u8;
    native_ai_propagate_destination_budget(
        w,
        h,
        origin,
        initial_budget as u8,
        &flags,
        terrain_move_codes,
        cost_row,
        &mut field,
    );
    Ok(field)
}

/// Go：`native_ai_destination.go:118`
pub(crate) static NATIVE_AI_DESTINATION_NEIGHBOURS: &[[i64; 2]] = &[[1, 0], [-1, 0], [0, 1], [0, -1]];

/// Go：`native_ai_destination.go:125`，`func nativeAIOppositeSelectorGroup(recordByte6 byte, selector int) bool`
pub(crate) fn native_ai_opposite_selector_group(record_byte6: u8, selector: i64) -> bool {
    if selector == 0 { record_byte6 != 0 } else { record_byte6 == 0 }
}

/// Go：`native_ai_destination.go:132`，`func nativeAISameSelectorGroup(recordByte6 byte, selector int) bool`
pub(crate) fn native_ai_same_selector_group(record_byte6: u8, selector: i64) -> bool {
    if selector == 0 { record_byte6 == 0 } else { record_byte6 != 0 }
}

/// Go：`native_ai_destination.go:139`，`func nativeAIPropagateDestinationBudget( w, h int, cell Cell, budget byte, flags, terrainMoveCodes, costRow, field []byte, )`
pub(crate) fn native_ai_propagate_destination_budget(
    w: i64,
    h: i64,
    cell: Cell,
    budget: u8,
    flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
    field: &mut [u8],
) {
    for delta in NATIVE_AI_DESTINATION_NEIGHBOURS {
        let next = Cell { x: cell.x + delta[0], y: cell.y + delta[1] };
        if next.x < 0 || next.y < 0 || next.x >= w || next.y >= h {
            continue;
        }
        let index = (next.y * w + next.x) as usize;
        let cost = cost_row[terrain_move_codes[index] as usize];
        if cost > budget {
            continue;
        }
        let mut next_budget = budget - cost;
        // 0x4e18f uses signed JLE.  Initial 0xff therefore behaves as -1,
        // while ordinary nonnegative budgets retain only strict improvements.
        if (next_budget as i8) <= (field[index] as i8) || flags[index] & NATIVE_COMMAND_GRID_BLOCKED != 0 {
            continue;
        }
        if flags[index] & NATIVE_COMMAND_GRID_ZERO_BUDGET != 0 {
            next_budget = 0;
        }
        field[index] = next_budget;
        native_ai_propagate_destination_budget(w, h, next, next_budget, flags, terrain_move_codes, cost_row, field);
    }
}

#[cfg(test)]
mod native_ai_destination_test;
