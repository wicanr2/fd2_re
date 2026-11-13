//! Go 來源：`remake/internal/battle/native_movement_cost.go`（98 行）。

use super::*;

pub const NATIVE_MOVEMENT_COST_ROW_COUNT: i64 = 29;

pub const NATIVE_MOVEMENT_COST_ROW_SIZE: i64 = 20;

/// Go：`native_movement_cost.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct NativeMovementCostRowJSON {
    /// Go 標籤：`json:"selector"`
    pub selector: i64,
    /// Go 標籤：`json:"raw"`
    #[serde(deserialize_with = "null_as_default")]
    pub raw: String,
}

/// Go：`native_movement_cost.go:20`，`func LoadNativeMovementCostRows(path string) ([][]byte, error)`
pub fn load_native_movement_cost_rows(path: &str) -> Result<Vec<Vec<u8>>> {
    let raw = read_file(path)?;
    let rows: Vec<NativeMovementCostRowJSON> = decode_json(&raw)?;
    if rows.len() as i64 != NATIVE_MOVEMENT_COST_ROW_COUNT {
        return Err(battle_err!("native movement rows={} want {}", rows.len(), NATIVE_MOVEMENT_COST_ROW_COUNT));
    }
    let mut result = Vec::with_capacity(rows.len());
    for (index, row) in rows.iter().enumerate() {
        if row.selector != index as i64 {
            return Err(battle_err!("native movement selector={} at index={}", row.selector, index));
        }
        let Some(decoded) = decode_hex(&row.raw).filter(|d| d.len() as i64 == NATIVE_MOVEMENT_COST_ROW_SIZE) else {
            return Err(battle_err!("native movement selector={index} has invalid raw row"));
        };
        result.push(decoded);
    }
    Ok(result)
}

/// NativeRelocationDestinationAllowed reproduces 0x115b6 mode 6's Enter
/// predicate. targetUnit is excluded from occupancy; any other record on the
/// destination with raw +5 bit0 clear blocks it. The target's raw fields select
/// a 0x4e555 row; the destination is refused when that row's terrain entry is
/// 20（不可通行）。
///
/// 原版：`0x11702 movzx eax,[ebx+eax]` 取 `costRow[地形碼]`，`0x11706 cmp eax,0x14`
/// 相等就 `je 0x117A9` 回到輸入迴圈（不接受），否則 `0x1170F mov eax,1` 接受。
/// 早先寫成「等於 20 才允許」是把極性寫反了；那時成本表本身也錯位（碼 2 變成 20），
/// 兩個錯誤在森林格上互相抵消，所以測不出來。
///
/// Go：`native_movement_cost.go:56`，`func NativeRelocationDestinationAllowed( records []byte, count, targetUnit int, destinationX, destinationY byte, terrainIndex int, costRows [][]byte, ) (bool, error)`
pub fn native_relocation_destination_allowed(
    records: &[u8],
    count: i64,
    target_unit: i64,
    destination_x: u8,
    destination_y: u8,
    terrain_index: i64,
    cost_rows: &[Vec<u8>],
) -> Result<bool> {
    if count < 0 || count > records.len() as i64 / NATIVE_RECORD_SIZE {
        return Err(battle_err!("native relocation count={count} is out of bounds"));
    }
    if target_unit < 0 || target_unit >= count {
        return Err(record_bounds_error(target_unit));
    }
    if cost_rows.len() as i64 != NATIVE_MOVEMENT_COST_ROW_COUNT
        || !(0..NATIVE_MOVEMENT_COST_ROW_SIZE).contains(&terrain_index)
    {
        return Err(battle_err!("native relocation movement table/index is invalid"));
    }
    for (selector, row) in cost_rows.iter().enumerate() {
        if row.len() as i64 != NATIVE_MOVEMENT_COST_ROW_SIZE {
            return Err(battle_err!("native movement selector={selector} row is malformed"));
        }
    }
    for unit in 0..count {
        if unit == target_unit {
            continue;
        }
        let record = &records[(unit * NATIVE_RECORD_SIZE) as usize..];
        if record[0] == destination_x && record[1] == destination_y && record[5] & 1 == 0 {
            return Ok(false);
        }
    }
    let target = &records[(target_unit * NATIVE_RECORD_SIZE) as usize..];
    let mut selector = target[0x20] as i64;
    if target[0x07] == 0x1c {
        selector = 1;
    } else if target[0x20] == 0x13 || target[0x1f] == 4 || target[0x1f] == 5 {
        selector = 19;
    }
    if selector < 0 || selector >= cost_rows.len() as i64 {
        return Err(battle_err!("native relocation selector={selector} is out of bounds"));
    }
    Ok(cost_rows[selector as usize][terrain_index as usize] != 20)
}

#[cfg(test)]
mod native_movement_cost_test;
