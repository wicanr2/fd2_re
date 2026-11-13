//! 移植自 Go `preparation_input.go`。

/// `0x19953` 的 raw 回傳契約。Go 版是 `int` 具名型別。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativePreparationInputResult(pub i64);

impl NativePreparationInputResult {
    pub const CONTINUE: NativePreparationInputResult = NativePreparationInputResult(0);
    pub const CONFIRM: NativePreparationInputResult = NativePreparationInputResult(1);
    pub const CANCEL: NativePreparationInputResult = NativePreparationInputResult(-1);
}

/// 套用一個已驗證的 scancode 分支。游標狀態是 raw `[0x53c57]` 的 0/1 值；確認／取消與原版
/// 一樣把它重設為 0。未知按鍵維持等待／繼續。
///
/// Go 版接受 nil 指標並回傳 continue；Rust 以 `None` 保留這條路徑。
pub fn apply_native_preparation_input(scan_code: u8, cursor_state: Option<&mut i64>) -> NativePreparationInputResult {
    let Some(cursor_state) = cursor_state else {
        return NativePreparationInputResult::CONTINUE;
    };
    match scan_code {
        0xe0 | 0x52 | 0x1c | 0x39 => {
            *cursor_state = 0;
            return NativePreparationInputResult::CONFIRM;
        }
        0x01 | 0x53 => {
            *cursor_state = 0;
            return NativePreparationInputResult::CANCEL;
        }
        0x4b => *cursor_state = 0,
        0x4d => *cursor_state = 1,
        _ => {}
    }
    NativePreparationInputResult::CONTINUE
}

#[cfg(test)]
mod tests;
