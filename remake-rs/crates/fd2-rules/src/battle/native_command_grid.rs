//! Go 來源：`remake/internal/battle/native_command_grid.go`（66 行）。

use super::*;

/// NativeCommandGridCell is one recovered 0x1ceed command-menu cell in the
/// original 320×200 coordinate space. The caller supplies command IDs in the
/// exact 0x1c269 order; this type intentionally contains no effect semantics.
///
/// Go：`native_command_grid.go:6`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandGridCell {
    pub command_id: i64,
    pub column: i64,
    pub row: i64,
    pub x: i64,
    pub y: i64,
    pub selected: bool,
}

pub const NATIVE_COMMAND_GRID_X: i64 = 0x12;

pub const NATIVE_COMMAND_GRID_Y: i64 = 0x67;

pub const NATIVE_COMMAND_GRID_COLUMN_W: i64 = 0x64;

pub const NATIVE_COMMAND_GRID_ROW_H: i64 = 0x16;

pub const NATIVE_COMMAND_GRID_ROWS: i64 = 4;

/// NativeCommandGrid mirrors 0x1ceed's i/4 column and i%4 row placement.
///
/// Go：`native_command_grid.go:24`，`func NativeCommandGrid(ids []int, selected int) []NativeCommandGridCell`
pub fn native_command_grid(ids: &[i64], selected: i64) -> Vec<NativeCommandGridCell> {
    todo!("battle/native_command_grid.go:NativeCommandGrid")
}

/// NativeCommandGridMove implements 0x1d51d navigation. It returns the prior
/// index for an invalid horizontal move; up/down wrap only at the list ends.
///
/// Go：`native_command_grid.go:41`，`func NativeCommandGridMove(index, count, direction int) int`
pub fn native_command_grid_move(index: i64, count: i64, direction: i64) -> i64 {
    todo!("battle/native_command_grid.go:NativeCommandGridMove")
}
