//! Go 來源：`remake/internal/battle/native_movement_cost.go`（98 行）。

use super::*;

pub const NATIVE_MOVEMENT_COST_ROW_COUNT: i64 = 29;

pub const NATIVE_MOVEMENT_COST_ROW_SIZE: i64 = 20;

/// Go：`native_movement_cost.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeMovementCostRowJSON {
    /// Go 標籤：`json:"selector"`
    pub selector: i64,
    /// Go 標籤：`json:"raw"`
    pub raw: String,
}

/// Go：`native_movement_cost.go:20`，`func LoadNativeMovementCostRows(path string) ([][]byte, error)`
pub fn load_native_movement_cost_rows(path: &str) -> Result<Vec<Vec<u8>>> {
    todo!("battle/native_movement_cost.go:LoadNativeMovementCostRows")
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
    todo!("battle/native_movement_cost.go:NativeRelocationDestinationAllowed")
}
