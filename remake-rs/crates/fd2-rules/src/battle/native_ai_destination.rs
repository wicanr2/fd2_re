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
    base_flags: &mut [u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<Vec<Cell>> {
    todo!("battle/native_ai_destination.go:NativeAIPhysicalDestinations")
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
    base_flags: &mut [u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<Vec<u8>> {
    todo!("battle/native_ai_destination.go:nativeMovementDestinationField")
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
    base_flags: &mut [u8],
    terrain_move_codes: &[u8],
    cost_row: &[u8],
) -> Result<Vec<u8>> {
    todo!("battle/native_ai_destination.go:nativeMovementBudgetField")
}

/// Go：`native_ai_destination.go:118`
pub(crate) static NATIVE_AI_DESTINATION_NEIGHBOURS: &[[i64; 2]] = &[[1, 0], [-1, 0], [0, 1], [0, -1]];

/// Go：`native_ai_destination.go:125`，`func nativeAIOppositeSelectorGroup(recordByte6 byte, selector int) bool`
pub(crate) fn native_ai_opposite_selector_group(record_byte6: u8, selector: i64) -> bool {
    todo!("battle/native_ai_destination.go:nativeAIOppositeSelectorGroup")
}

/// Go：`native_ai_destination.go:132`，`func nativeAISameSelectorGroup(recordByte6 byte, selector int) bool`
pub(crate) fn native_ai_same_selector_group(record_byte6: u8, selector: i64) -> bool {
    todo!("battle/native_ai_destination.go:nativeAISameSelectorGroup")
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
    todo!("battle/native_ai_destination.go:nativeAIPropagateDestinationBudget")
}
