//! Ebiten v2.6.6 的鍵盤鍵編號與鍵名（`ebiten.Key` 的數值、`Key.String`／`MarshalText` 與 `UnmarshalText`）。
//!
//! R5 階段 4（r5i）新增，供按鍵設定（`player_input_bindings.rs`）使用：
//!
//! - `controls-settings.json` 以 Ebiten 的鍵名記錄玩家指定的鍵（Go `Key.MarshalText`／`UnmarshalText`），兩版的設定檔
//!   要能互通，所以鍵名與別名照 Ebiten。
//! - 面板指定鍵時取「本幀剛按下的第一個鍵」（Go `inpututil.AppendJustPressedKeys` 依 `ebiten.Key` 數值遞增），
//!   所以數值順序也照 Ebiten。
//!
//! 數值是 `internal/ui/keys.go` 的 `iota` 順序（`KeyA = 0` 到 `KeyReserved3`），`ebiten.KeyAlt`／`KeyControl`／
//! `KeyShift`／`KeyMeta` 是 `KeyReserved0..3`（左右任一個的虛擬鍵），`KeyMax = KeyMeta`。鍵名是 `keys.go` 的
//! `String()`，別名是 `keyNameToKeyCode`（比對前先 `strings.ToLower`）。這是 Ebiten 函式庫的資料，不是 Go 標準函式庫，
//! 性質同教訓 `rust-port-no-go-stdlib` 的具名例外（以 Go 套件命名、保留授權聲明）；是否登記由使用者決定。
//!
//! 表格轉錄自 github.com/hajimehoshi/ebiten/v2 v2.6.6 的 `keys.go` 與 `internal/ui/keys.go`
//! （兩者皆由 genkeys.go 產生）。原始碼授權：
//!
//! Copyright 2013 The Ebitengine Authors
//!
//! Licensed under the Apache License, Version 2.0 (the "License");
//! you may not use this file except in compliance with the License.
//! You may obtain a copy of the License at
//!
//!     http://www.apache.org/licenses/LICENSE-2.0
//!
//! Unless required by applicable law or agreed to in writing, software
//! distributed under the License is distributed on an "AS IS" BASIS,
//! WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//! See the License for the specific language governing permissions and
//! limitations under the License.

/// Go `ebiten.Key` 的數值（Go 的 `int`）。`-1` 是 `cmd/fd2` 的 `keyUnbound`。
pub type EbitenKey = i32;

/// 依數值排列的鍵名（Go `Key.String()`）；索引就是 `ebiten.Key` 的值。
pub const KEY_NAMES: [&str; 109] = [
    "A",
    "B",
    "C",
    "D",
    "E",
    "F",
    "G",
    "H",
    "I",
    "J",
    "K",
    "L",
    "M",
    "N",
    "O",
    "P",
    "Q",
    "R",
    "S",
    "T",
    "U",
    "V",
    "W",
    "X",
    "Y",
    "Z",
    "AltLeft",
    "AltRight",
    "ArrowDown",
    "ArrowLeft",
    "ArrowRight",
    "ArrowUp",
    "Backquote",
    "Backslash",
    "Backspace",
    "BracketLeft",
    "BracketRight",
    "CapsLock",
    "Comma",
    "ContextMenu",
    "ControlLeft",
    "ControlRight",
    "Delete",
    "Digit0",
    "Digit1",
    "Digit2",
    "Digit3",
    "Digit4",
    "Digit5",
    "Digit6",
    "Digit7",
    "Digit8",
    "Digit9",
    "End",
    "Enter",
    "Equal",
    "Escape",
    "F1",
    "F2",
    "F3",
    "F4",
    "F5",
    "F6",
    "F7",
    "F8",
    "F9",
    "F10",
    "F11",
    "F12",
    "Home",
    "Insert",
    "MetaLeft",
    "MetaRight",
    "Minus",
    "NumLock",
    "Numpad0",
    "Numpad1",
    "Numpad2",
    "Numpad3",
    "Numpad4",
    "Numpad5",
    "Numpad6",
    "Numpad7",
    "Numpad8",
    "Numpad9",
    "NumpadAdd",
    "NumpadDecimal",
    "NumpadDivide",
    "NumpadEnter",
    "NumpadEqual",
    "NumpadMultiply",
    "NumpadSubtract",
    "PageDown",
    "PageUp",
    "Pause",
    "Period",
    "PrintScreen",
    "Quote",
    "ScrollLock",
    "Semicolon",
    "ShiftLeft",
    "ShiftRight",
    "Slash",
    "Space",
    "Tab",
    "Alt",
    "Control",
    "Shift",
    "Meta",
];

/// Go `ebiten.KeyMax`（`KeyMeta`）。
pub const KEY_MAX: EbitenKey = 108;

/// 依鍵名取數值（只收 [`KEY_NAMES`] 的正式名稱，大小寫照原樣）。表格內部用，常數由它在編譯期算出。
const fn code(name: &str) -> EbitenKey {
    let mut i = 0;
    while i < KEY_NAMES.len() {
        if const_eq(KEY_NAMES[i], name) {
            return i as EbitenKey;
        }
        i += 1;
    }
    panic!("unknown Ebiten key name")
}

const fn const_eq(a: &str, b: &str) -> bool {
    let (a, b) = (a.as_bytes(), b.as_bytes());
    if a.len() != b.len() {
        return false;
    }
    let mut i = 0;
    while i < a.len() {
        if a[i] != b[i] {
            return false;
        }
        i += 1;
    }
    true
}

pub const KEY_A: EbitenKey = code("A");
pub const KEY_D: EbitenKey = code("D");
pub const KEY_Q: EbitenKey = code("Q");
pub const KEY_R: EbitenKey = code("R");
pub const KEY_S: EbitenKey = code("S");
pub const KEY_W: EbitenKey = code("W");
pub const KEY_Z: EbitenKey = code("Z");
pub const KEY_ALT_LEFT: EbitenKey = code("AltLeft");
pub const KEY_ALT_RIGHT: EbitenKey = code("AltRight");
pub const KEY_ARROW_DOWN: EbitenKey = code("ArrowDown");
pub const KEY_ARROW_LEFT: EbitenKey = code("ArrowLeft");
pub const KEY_ARROW_RIGHT: EbitenKey = code("ArrowRight");
pub const KEY_ARROW_UP: EbitenKey = code("ArrowUp");
pub const KEY_BACKSPACE: EbitenKey = code("Backspace");
pub const KEY_CONTROL_LEFT: EbitenKey = code("ControlLeft");
pub const KEY_CONTROL_RIGHT: EbitenKey = code("ControlRight");
pub const KEY_DELETE: EbitenKey = code("Delete");
pub const KEY_DIGIT1: EbitenKey = code("Digit1");
pub const KEY_ENTER: EbitenKey = code("Enter");
pub const KEY_ESCAPE: EbitenKey = code("Escape");
pub const KEY_F1: EbitenKey = code("F1");
pub const KEY_F2: EbitenKey = code("F2");
pub const KEY_F3: EbitenKey = code("F3");
pub const KEY_F4: EbitenKey = code("F4");
pub const KEY_F5: EbitenKey = code("F5");
pub const KEY_F6: EbitenKey = code("F6");
pub const KEY_F7: EbitenKey = code("F7");
pub const KEY_F8: EbitenKey = code("F8");
pub const KEY_F9: EbitenKey = code("F9");
pub const KEY_F10: EbitenKey = code("F10");
pub const KEY_F11: EbitenKey = code("F11");
pub const KEY_F12: EbitenKey = code("F12");
pub const KEY_HOME: EbitenKey = code("Home");
pub const KEY_META_LEFT: EbitenKey = code("MetaLeft");
pub const KEY_META_RIGHT: EbitenKey = code("MetaRight");
pub const KEY_NUMPAD5: EbitenKey = code("Numpad5");
pub const KEY_NUMPAD_ENTER: EbitenKey = code("NumpadEnter");
pub const KEY_PAGE_DOWN: EbitenKey = code("PageDown");
pub const KEY_PAGE_UP: EbitenKey = code("PageUp");
pub const KEY_SHIFT_LEFT: EbitenKey = code("ShiftLeft");
pub const KEY_SHIFT_RIGHT: EbitenKey = code("ShiftRight");
pub const KEY_SPACE: EbitenKey = code("Space");
pub const KEY_TAB: EbitenKey = code("Tab");
pub const KEY_ALT: EbitenKey = code("Alt");
pub const KEY_CONTROL: EbitenKey = code("Control");
pub const KEY_SHIFT: EbitenKey = code("Shift");
pub const KEY_META: EbitenKey = code("Meta");

/// Go `Key.String()`：有效的鍵回傳鍵名，其他值回傳空字串。
pub fn key_string(k: EbitenKey) -> &'static str {
    if (0..=KEY_MAX).contains(&k) { KEY_NAMES[k as usize] } else { "" }
}

/// Go `keyNameToKeyCode` 的舊名別名（v2.1 起不建議使用，但 `UnmarshalText` 仍接受）。
const ALIASES: [(&str, &str); 36] = [
    ("0", "Digit0"),
    ("1", "Digit1"),
    ("2", "Digit2"),
    ("3", "Digit3"),
    ("4", "Digit4"),
    ("5", "Digit5"),
    ("6", "Digit6"),
    ("7", "Digit7"),
    ("8", "Digit8"),
    ("9", "Digit9"),
    ("apostrophe", "Quote"),
    ("down", "ArrowDown"),
    ("graveaccent", "Backquote"),
    ("kp0", "Numpad0"),
    ("kp1", "Numpad1"),
    ("kp2", "Numpad2"),
    ("kp3", "Numpad3"),
    ("kp4", "Numpad4"),
    ("kp5", "Numpad5"),
    ("kp6", "Numpad6"),
    ("kp7", "Numpad7"),
    ("kp8", "Numpad8"),
    ("kp9", "Numpad9"),
    ("kpadd", "NumpadAdd"),
    ("kpdecimal", "NumpadDecimal"),
    ("kpdivide", "NumpadDivide"),
    ("kpenter", "NumpadEnter"),
    ("kpequal", "NumpadEqual"),
    ("kpmultiply", "NumpadMultiply"),
    ("kpsubtract", "NumpadSubtract"),
    ("left", "ArrowLeft"),
    ("leftbracket", "BracketLeft"),
    ("menu", "ContextMenu"),
    ("right", "ArrowRight"),
    ("rightbracket", "BracketRight"),
    ("up", "ArrowUp"),
];

/// Go `(*Key).UnmarshalText` 經 `keyNameToKeyCode(strings.ToLower(name))`：正式鍵名與舊名別名，不分大小寫。
///
/// Go 的 `strings.ToLower` 是 Unicode 小寫；鍵名都是 ASCII，非 ASCII 的輸入兩邊都找不到，所以只做 ASCII 小寫。
pub fn key_from_name(name: &str) -> Option<EbitenKey> {
    if !name.is_ascii() {
        return None;
    }
    let lower = name.to_ascii_lowercase();
    if let Some(i) = KEY_NAMES.iter().position(|n| n.to_ascii_lowercase() == lower) {
        return Some(i as EbitenKey);
    }
    ALIASES
        .iter()
        .find(|(alias, _)| *alias == lower)
        .and_then(|(_, target)| KEY_NAMES.iter().position(|n| n == target))
        .map(|i| i as EbitenKey)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn names_codes_and_aliases_follow_ebiten() {
        assert_eq!(KEY_NAMES.len() as EbitenKey, KEY_MAX + 1);
        assert_eq!((KEY_A, KEY_Z, KEY_ENTER, KEY_ESCAPE, KEY_TAB, KEY_ALT, KEY_META), (0, 25, 54, 56, 104, 105, 108));
        assert_eq!(key_string(KEY_DIGIT1), "Digit1");
        assert_eq!(key_string(-1), "");
        assert_eq!(key_string(KEY_MAX + 1), "");
        assert_eq!(key_from_name("z"), Some(KEY_Z));
        assert_eq!(key_from_name("ENTER"), Some(KEY_ENTER));
        assert_eq!(key_from_name("kp5"), Some(KEY_NUMPAD5));
        assert_eq!(key_from_name("1"), Some(KEY_DIGIT1));
        assert_eq!(key_from_name("return"), None);
        assert_eq!(key_from_name("NoSuchKey"), None);
        assert_eq!(key_from_name("Ｚ"), None);
    }
}
