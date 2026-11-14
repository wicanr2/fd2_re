//! Go 來源：`remake/cmd/fd2/native_town_secret_input.go`（52 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! E 組另補移植 `nativeBIOSFunctionScan`（`r8`，`pressedNativeTownSecretScan` 呼叫）。
//!
//! Go 直接查 `ebiten.IsKeyPressed`／`inpututil.IsKeyJustPressed`；Rust 兩個函式改收本幀的
//! [`TickInput`]（120「輸入一律讀 `self.tick_input`」），呼叫端傳 `&self.tick_input` 或
//! `update` 開頭複製的同一份輸入。

use super::*;

/// Go 原式：`iota`
pub(crate) const NATIVE_FUNCTION_SHIFT: i64 = 0;
/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_FUNCTION_CONTROL: i64 = 1;
/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_FUNCTION_ALT: i64 = 2;

/// Go：`native_town_secret_input.go:14`，`func nativeBIOSFunctionScan(modifier, function int) (int, bool)`
/// （`r8`，E 組隨呼叫端補移植）。
pub(crate) fn native_bios_function_scan(modifier: i64, function: i64) -> (i64, bool) {
    if !(NATIVE_FUNCTION_SHIFT..=NATIVE_FUNCTION_ALT).contains(&modifier) || !(1..=10).contains(&function) {
        return (0, false);
    }
    ([0x54, 0x5e, 0x68][modifier as usize] + function - 1, true)
}

/// Go：`native_town_secret_input.go:22`，`func nativeModifierHeld() bool`
///
/// 簽名改寫：Go 讀全域鍵盤狀態，Rust 讀本幀輸入 `input`。
pub(crate) fn native_modifier_held(input: &TickInput) -> bool {
    input.is_pressed(Key::Shift) || input.is_pressed(Key::Control) || input.is_pressed(Key::Alt)
}

/// Go：`native_town_secret_input.go:28`，`func pressedNativeTownSecretScan() (int, bool)`
///
/// 簽名改寫：同 [`native_modifier_held`]，讀本幀輸入 `input`。
pub(crate) fn pressed_native_town_secret_scan(input: &TickInput) -> (i64, bool) {
    let mut modifier: i64 = -1;
    for (index, key) in [Key::Shift, Key::Control, Key::Alt].into_iter().enumerate() {
        if input.is_pressed(key) {
            if modifier != -1 {
                return (0, false);
            }
            modifier = index as i64;
        }
    }
    if modifier == -1 {
        return (0, false);
    }
    for (index, key) in [Key::F1, Key::F2, Key::F3, Key::F4, Key::F5, Key::F6, Key::F7, Key::F8, Key::F9, Key::F10]
        .into_iter()
        .enumerate()
    {
        if input.is_just_pressed(key) {
            return native_bios_function_scan(modifier, index as i64 + 1);
        }
    }
    (0, false)
}

#[cfg(test)]
mod native_town_secret_input_test;
