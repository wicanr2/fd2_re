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
    ids.iter()
        .enumerate()
        .map(|(i, &id)| {
            let i = i as i64;
            NativeCommandGridCell {
                command_id: id,
                column: i / NATIVE_COMMAND_GRID_ROWS,
                row: i % NATIVE_COMMAND_GRID_ROWS,
                x: NATIVE_COMMAND_GRID_X + NATIVE_COMMAND_GRID_COLUMN_W * (i / NATIVE_COMMAND_GRID_ROWS),
                y: NATIVE_COMMAND_GRID_Y + NATIVE_COMMAND_GRID_ROW_H * (i % NATIVE_COMMAND_GRID_ROWS),
                selected: i == selected,
            }
        })
        .collect()
}

/// NativeCommandGridMove implements 0x1d51d navigation. It returns the prior
/// index for an invalid horizontal move; up/down wrap only at the list ends.
///
/// Go：`native_command_grid.go:41`，`func NativeCommandGridMove(index, count, direction int) int`
pub fn native_command_grid_move(index: i64, count: i64, direction: i64) -> i64 {
    if count <= 0 || index < 0 || index >= count {
        return -1;
    }
    match direction {
        // up
        0 => {
            if index == 0 {
                return count - 1;
            }
            return index - 1;
        }
        // down
        1 => {
            if index == count - 1 {
                return 0;
            }
            return index + 1;
        }
        // left
        2 if index >= NATIVE_COMMAND_GRID_ROWS => return index - NATIVE_COMMAND_GRID_ROWS,
        // right
        3 if index + NATIVE_COMMAND_GRID_ROWS < count => return index + NATIVE_COMMAND_GRID_ROWS,
        _ => {}
    }
    index
}

#[cfg(test)]
mod native_command_grid_test;
