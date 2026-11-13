//! Go 來源：`remake/internal/battle/native_ai_item_preselection.go`（206 行）。

use super::*;

/// NativeAI1567EScoreResult preserves the positive best tuple written by
/// 0x1567e to [0x53c33/37/3b/3f]. InventorySlot is the raw slot index saved in
/// [0x53c3f], not the item command byte.
///
/// Go：`native_ai_item_preselection.go:8`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAI1567EScoreResult {
    pub max_score: i64,
    pub x: i64,
    pub y: i64,
    pub inventory_slot: i64,
    pub item_id: i64,
    /// TargetIndices is the detached raw target list for the winning
    /// destination.  It is retained so the execution owner can re-run the
    /// second target/effect stage instead of inventing a target from normalized
    /// party data.
    pub target_indices: Vec<u8>,
    pub has_positive_winner: bool,
}

/// ScoreNativeAI1567E connects the proven item-command preselection slices:
/// 0x1b8a6 count-sized slot scan, caller-specific 0x14818/0x149f8 geometry,
/// 0x15880 scoring, and strict greater-than winner replacement. It does not
/// execute the selected item or infer an effect name.
///
/// Go：`native_ai_item_preselection.go:26`，`func ScoreNativeAI1567E( w, h int, records []byte, count, actor, selector int, itemRows []byte, book []NativeCommandRecord, baseFlags []byte, ) (NativeAI1567EScoreResult, error)`
pub fn score_native_ai1567_e(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
    item_rows: &[u8],
    book: &[NativeCommandRecord],
    base_flags: &[u8],
) -> Result<NativeAI1567EScoreResult> {
    let mut result = NativeAI1567EScoreResult::default();
    if w <= 0
        || h <= 0
        || count <= 0
        || count > 0x100
        || actor < 0
        || actor >= count
        || (selector != 0 && selector != 1)
        || records.len() as i64 != count.wrapping_mul(NATIVE_RECORD_SIZE)
        || base_flags.len() as i64 != w.wrapping_mul(h)
        || item_rows.is_empty()
        || item_rows.len() as i64 % NATIVE_ITEM_EFFECT_ROW_SIZE != 0
    {
        return Err(battle_err!("native AI 0x1567e inputs are malformed"));
    }
    let actor_record = &records[(actor * NATIVE_RECORD_SIZE) as usize..];
    let actor_cell = Cell { x: actor_record[0] as i64, y: actor_record[1] as i64 };
    if actor_cell.x < 0 || actor_cell.y < 0 || actor_cell.x >= w || actor_cell.y >= h {
        return Err(battle_err!("native AI 0x1567e actor is outside the grid"));
    }
    let occupied = native_inventory_occupied_count(records, actor)?;
    let row_size = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;
    for slot in 0..occupied {
        let item_id = actor_record[0x0b + slot as usize * 2] as i64;
        let row_offset = item_id as usize * row_size;
        if row_offset + row_size > item_rows.len() {
            return Err(battle_err!("native AI 0x1567e item {item_id} row is unavailable"));
        }
        let row = &item_rows[row_offset..row_offset + row_size];
        let item_type = row[0x0d] as i64;
        if item_type == 0 {
            continue;
        }
        let command = row[0x10] as i64;
        let mut distance = command;
        let mut inner = 0;
        if command > 0x0f {
            // 0x15738 writes DL=1 and DH=DL before the first 0x14818.
            distance = 1;
            inner = 1;
        }
        let field = native_command_target_field_bytes(w, h, actor_cell, distance, inner, base_flags)?;
        for y in 0..h {
            for x in 0..w {
                if field[(y * w + x) as usize] == 0xff {
                    continue;
                }
                let destination = Cell { x, y };
                let targets = if command > 0x0f {
                    native_ai1567_e_cardinal_targets(w, h, records, count, actor_cell, destination, command - 0x10)?
                } else {
                    let mut target_code = row[0x11] as i64;
                    if selector == 0 {
                        target_code = if target_code == 0 { 1 } else { 0 };
                    }
                    native_ai1567_e_area_targets(
                        w,
                        h,
                        records,
                        count,
                        destination,
                        row[0x12] as i64,
                        target_code,
                        base_flags,
                    )?
                };
                if targets.is_empty() {
                    continue;
                }
                let score = score_native_ai_item_command_targets(
                    records,
                    &targets,
                    item_type,
                    row[0x0e] as i64 | (row[0x0f] as i64) << 8,
                    book,
                )?;
                if score > result.max_score {
                    result = NativeAI1567EScoreResult {
                        max_score: score,
                        x,
                        y,
                        inventory_slot: slot,
                        item_id,
                        target_indices: targets,
                        has_positive_winner: true,
                    };
                }
            }
        }
    }
    Ok(result)
}

/// Go：`native_ai_item_preselection.go:129`，`func nativeAI1567EAreaTargets( w, h int, records []byte, count int, origin Cell, mode, targetCode int, baseFlags []byte, ) ([]byte, error)`
pub(crate) fn native_ai1567_e_area_targets(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    origin: Cell,
    mode: i64,
    target_code: i64,
    base_flags: &[u8],
) -> Result<Vec<u8>> {
    let field = native_command_target_field_bytes(w, h, origin, mode, 0, base_flags)?;
    let mut targets = Vec::new();
    for index in 0..count {
        let record = &records[(index * NATIVE_RECORD_SIZE) as usize..];
        if record[5] & 1 != 0 || !native_ai_scored_raw_target_matches(target_code, record[6]) {
            continue;
        }
        let cell = Cell { x: record[0] as i64, y: record[1] as i64 };
        if cell.x < 0 || cell.y < 0 || cell.x >= w || cell.y >= h {
            return Err(battle_err!("native AI 0x1567e target is outside the grid"));
        }
        if field[(cell.y * w + cell.x) as usize] != 0xff {
            targets.push(index as u8);
        }
    }
    Ok(targets)
}

/// Go：`native_ai_item_preselection.go:161`，`func nativeAI1567ECardinalTargets( w, h int, records []byte, count int, start, destination Cell, steps int, ) ([]byte, error)`
pub(crate) fn native_ai1567_e_cardinal_targets(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    start: Cell,
    destination: Cell,
    steps: i64,
) -> Result<Vec<u8>> {
    if steps < 0
        || start.x < 0
        || start.y < 0
        || start.x >= w
        || start.y >= h
        || destination.x < 0
        || destination.y < 0
        || destination.x >= w
        || destination.y >= h
    {
        return Err(battle_err!("native AI 0x1567e cardinal inputs are malformed"));
    }
    let (mut dx, mut dy) = (0, 0);
    if start.x != destination.x {
        dx = if start.x > destination.x { -1 } else { 1 };
    } else if start.y > destination.y {
        dy = -1;
    } else {
        dy = 1;
    }
    let mut cell = start;
    let mut targets = Vec::new();
    for _ in 0..steps {
        cell.x += dx;
        cell.y += dy;
        if cell.x < 0 || cell.y < 0 || cell.x >= w || cell.y >= h {
            continue;
        }
        for index in 0..count {
            let record = &records[(index * NATIVE_RECORD_SIZE) as usize..];
            if record[5] & 1 != 0 || record[0] as i64 != cell.x || record[1] as i64 != cell.y {
                continue;
            }
            // 0x14AE4..0x14AF4: selector zero collects nonzero raw +6.
            if record[6] != 0 {
                targets.push(index as u8);
            }
            break;
        }
    }
    Ok(targets)
}

#[cfg(test)]
mod native_ai_item_preselection_test;
