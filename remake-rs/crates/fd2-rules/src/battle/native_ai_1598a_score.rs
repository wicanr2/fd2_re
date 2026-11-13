//! Go 來源：`remake/internal/battle/native_ai_1598a_score.go`（100 行）。

use super::*;

/// NativeAI1598AScoreResult preserves the numeric maximum written to
/// [0x53c23]. PositiveWinner is present only when a candidate scored above
/// zero. The all-zero command-word local remains unproven and is not invented.
///
/// Go：`native_ai_1598a_score.go:11`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAI1598AScoreResult {
    pub max_score: i64,
    pub positive_winner: NativeAISpellCandidate,
    pub has_positive_winner: bool,
}

/// nativeAICommandCostRow 是 0x4E555(0) 的成本列：0x55446 的 20 個 1
/// （remake/assets/data/native_movement_cost_rows.json selector 0）。
///
/// Go：`native_ai_1598a_score.go:19`
pub(crate) static NATIVE_AI_COMMAND_COST_ROW: [u8; 20] = [1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1];

/// ScoreNativeAI1598A connects the proven 0x1598a slices without executing a
/// command: availability, destination/target groups, 0x15b77 family scoring,
/// and the strict positive-score winner comparison.
///
/// Go：`native_ai_1598a_score.go:26`，`func ScoreNativeAI1598A( w, h int, records []byte, count, actor, selector int, unit *Unit, book []NativeCommandRecord, baseFlags, terrainMoveCodes, costRow []byte, skip func(int) bool, ) (NativeAI1598AScoreResult, error)`
pub fn score_native_ai1598_a(
    w: i64,
    h: i64,
    records: &[u8],
    count: i64,
    actor: i64,
    selector: i64,
    unit: Option<&Unit>,
    book: &[NativeCommandRecord],
    base_flags: &[u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
    skip: Option<&mut dyn FnMut(i64) -> bool>,
) -> Result<NativeAI1598AScoreResult> {
    let mut skip = skip;
    let Some(unit) = unit.filter(|_| {
        count > 0
            && count <= 0x100
            && actor >= 0
            && actor < count
            && records.len() as i64 == count.wrapping_mul(NATIVE_RECORD_SIZE)
            && w > 0
            && h > 0
            && base_flags.len() as i64 == w.wrapping_mul(h)
            && terrain_move_codes.len() as i64 == w.wrapping_mul(h)
            && cost_row.len() as i64 == NATIVE_MOVEMENT_COST_ROW_SIZE
            && (selector == 0 || selector == 1)
            && book.len() as i64 == NATIVE_COMMAND_RECORD_COUNT
    }) else {
        return Err(battle_err!("native AI 0x1598a inputs are malformed"));
    };
    let actor_record = &records[(actor * NATIVE_RECORD_SIZE) as usize..];
    for (index, &value) in unit.native_command_mask.iter().enumerate() {
        if actor_record[0x1a + index] != value {
            return Err(battle_err!("native AI 0x1598a actor mask disagrees with record"));
        }
    }
    if unit.mp < 0 || unit.mp > 0xffff || u16::from_le_bytes([actor_record[0x44], actor_record[0x45]]) != unit.mp as u16
    {
        return Err(battle_err!("native AI 0x1598a actor MP disagrees with record"));
    }
    for (id, command) in book.iter().enumerate() {
        if command.id != id as i64 {
            return Err(battle_err!("native AI 0x1598a command book is malformed"));
        }
    }
    let mut candidates = Vec::new();
    for command_id in native_available_ai_scored_command_ids(Some(unit), book) {
        let command = book[command_id as usize];
        // 施法落點不走行動者的移動成本：0x159A5 push 0 → 0x4E555 取成本列 0（20 個 1），
        // 0x15A60 把它與指令 +3 距離一起交給 0x4E040。傳進來的 costRow 是行動者那一列，
        // 地形成本大於 1 的路上會把距離 5 的施法算成走不到（第八章 r1 seq 940 記錄 18）。
        let groups = native_ai_scored_command_candidate_groups(
            w,
            h,
            records,
            count,
            actor,
            selector,
            command,
            base_flags,
            terrain_move_codes,
            &NATIVE_AI_COMMAND_COST_ROW,
        )?;
        let scores = score_native_ai_scored_command_groups(
            records,
            command_id,
            command.damage,
            &groups,
            native_ai_reborrow_skip(&mut skip),
        )?;
        for score in scores {
            if score.score < 0 {
                return Err(battle_err!("native AI 0x1598a score is negative"));
            }
            if score.score == 0 {
                continue;
            }
            candidates.push(NativeAISpellCandidate {
                command_id,
                command_value: command.damage,
                x: score.destination.x,
                y: score.destination.y,
                score: score.score,
                target_indices: score.target_indices.clone(),
            });
        }
    }
    let (winner, ok) = select_native_ai_spell_candidate(&candidates);
    if !ok {
        return Ok(NativeAI1598AScoreResult::default());
    }
    Ok(NativeAI1598AScoreResult { max_score: winner.score, positive_winner: winner, has_positive_winner: true })
}

#[cfg(test)]
mod native_ai_1598a_score_test;
