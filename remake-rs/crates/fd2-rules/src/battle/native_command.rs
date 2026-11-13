//! Go 來源：`remake/internal/battle/native_command.go`（48 行）。

use super::*;

impl Unit {
    /// SetInitialCommandMask materializes the exact constructor input for native
    /// unit+0x1a..+0x1e.  An absent JSON field is accepted for legacy editable
    /// scenarios and yields a zero mask; any non-empty malformed field is rejected
    /// rather than being truncated into a different command inventory.
    ///
    /// Go：`native_command.go:9`，`func (u *Unit) SetInitialCommandMask(raw []byte) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_initial_command_mask(&mut self, raw: &[u8]) -> Result<()> {
        todo!("battle/native_command.go:Unit.SetInitialCommandMask")
    }

    /// NativeCommandIDs expands the five-byte bitset in exactly the original
    /// byte-major, low-bit-first order used by 0x1c269.
    ///
    /// Go：`native_command.go:24`，`func (u *Unit) NativeCommandIDs() []int`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command_ids(&self) -> Vec<i64> {
        todo!("battle/native_command.go:Unit.NativeCommandIDs")
    }

    /// EnableNativeCommand mirrors the bounded OR write performed by 0x1d7fb.
    /// It only changes the native inventory bit; it does not claim the command is
    /// executable until its record/effect path has separately been recovered.
    ///
    /// Go：`native_command.go:42`，`func (u *Unit) EnableNativeCommand(id int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn enable_native_command(&mut self, id: i64) -> bool {
        todo!("battle/native_command.go:Unit.EnableNativeCommand")
    }
}
