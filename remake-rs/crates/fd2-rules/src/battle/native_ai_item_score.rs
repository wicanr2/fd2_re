//! Go 來源：`remake/internal/battle/native_ai_item_score.go`（70 行）。

use super::*;

/// ScoreNativeAIItemCommandTargets reproduces 0x15880 for one item row and
/// its already-built target-index list. itemType is row +0x0d and itemWord is
/// row +0x0e. No gameplay effect name is assigned to either field.
///
/// Go：`native_ai_item_score.go:11`，`func ScoreNativeAIItemCommandTargets( records []byte, targets []byte, itemType int, itemWord int, book []NativeCommandRecord, ) (int, error)`
pub fn score_native_ai_item_command_targets(
    records: &[u8],
    targets: &[u8],
    item_type: i64,
    item_word: i64,
    book: &[NativeCommandRecord],
) -> Result<i64> {
    if records.len() as i64 % NATIVE_RECORD_SIZE != 0
        || !(0..=0xff).contains(&item_type)
        || !(0..=0xffff).contains(&item_word)
    {
        return Err(battle_err!("native AI item score inputs are malformed"));
    }
    let count = records.len() as i64 / NATIVE_RECORD_SIZE;
    for &index in targets {
        if index as i64 >= count {
            return Err(battle_err!("native AI item score target {index} is out of bounds"));
        }
    }
    let record_at = |index: u8| &records[index as usize * NATIVE_RECORD_SIZE as usize..];
    match item_type {
        5 | 0x0d => {
            let mut score = 0;
            for &index in targets {
                let record = record_at(index);
                let current = u16::from_le_bytes([record[0x40], record[0x41]]) as i64;
                let maximum = u16::from_le_bytes([record[0x42], record[0x43]]) as i64;
                let mut value = 0;
                if current <= maximum / 3 {
                    value = 8;
                } else if current <= maximum / 2 {
                    value = 3;
                }
                if record[0x34] & 0x80 != 0 {
                    value *= 3;
                }
                score += value;
            }
            Ok(score)
        }
        0x14 | 0x15 | 0x18 => {
            let mut threshold = item_word;
            if item_type != 0x18 {
                if item_word >= book.len() as i64
                    || book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT
                    || book[item_word as usize].id != item_word
                {
                    return Err(battle_err!("native AI item score command record is unavailable"));
                }
                threshold = book[item_word as usize].damage;
            }
            let mut score = 0;
            for &index in targets {
                let record = record_at(index);
                let current = u16::from_le_bytes([record[0x40], record[0x41]]) as i64;
                if current <= threshold {
                    score += 0x12;
                } else {
                    score += 8;
                }
            }
            Ok(score)
        }
        _ => Ok(0),
    }
}

#[cfg(test)]
mod native_ai_item_score_test;
