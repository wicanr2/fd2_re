//! 輸入對應（R5 階段 4，r5w）：實體輸入 → 邏輯按鍵 → [`RawInputState`]，再由 `fd2-game` 的
//! [`fd2_game::game::EbitenInputState`] 組成每個邏輯幀的 `TickInput`。
//!
//! 分兩層，之後加手把與自訂綁定只改這一層，不改 `fd2-game`（使用者 2026-11-16）：
//!
//! 1. [`PhysicalSnapshot`]：本顯示幀的實體輸入（Bevy 的按住中按鍵、滑鼠左鍵與游標、觸控點、輸入文字），座標是視窗實體像素。
//! 2. [`Bindings`]：實體輸入 → 邏輯鍵（[`GameKey`]）的對應表。預設表 [`Bindings::ebiten`] 照 Ebiten 的實體鍵名
//!    一對一對應；沒有對應的鍵算成 `other_keys`（Go `inpututil.AppendJustPressedKeys` 的「任意鍵」）。滑鼠與觸控的座標
//!    依 [`Letterbox::to_logical`] 換成 Ebiten 邏輯座標。
//!
//! 對照 Go 實際讀的輸入見 `fd2-game/PORTING.md`「R5 階段 4：r5w」的輸入對照表。

use std::collections::{BTreeSet, HashMap};

use bevy::input::keyboard::KeyCode;
use fd2_game::game::{Key as GameKey, RawInputState};

use crate::video::Letterbox;

/// 一個實體輸入。之後的手把按鍵加在這裡（例如 `GamepadButton(..)`），對應表與 `fd2-game` 不變。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum PhysicalInput {
    Keyboard(KeyCode),
}

/// 本顯示幀的實體輸入（視窗實體像素）。
#[derive(Debug, Clone, Default)]
pub struct PhysicalSnapshot {
    pub held: Vec<PhysicalInput>,
    pub mouse_left: bool,
    pub cursor: Option<(f32, f32)>,
    pub touches: Vec<(u64, f32, f32)>,
    pub chars: Vec<char>,
}

/// 實體輸入 → 邏輯鍵的對應表。
#[derive(Debug, Clone)]
pub struct Bindings {
    map: HashMap<PhysicalInput, GameKey>,
}

/// Ebiten 鍵與 Bevy `KeyCode` 的一對一對應（兩邊都是實體鍵位，以美式鍵盤命名）。Ebiten 的 `KeyControl`／`KeyShift`
/// 是左右任一個，`fd2-game` 的 [`GameKey`] 沒有分左右，所以左右都對到同一個邏輯鍵；`KeyAlt` 由 `fd2-game` 從
/// `AltLeft`／`AltRight` 算出，沒有實體鍵。其餘 Bevy 鍵（含 `NumpadEnter`、Meta）在 Go 沒有讀，算成 `other_keys`。
const EBITEN_KEYS: [(KeyCode, GameKey); 34] = [
    (KeyCode::KeyA, GameKey::A),
    (KeyCode::KeyD, GameKey::D),
    (KeyCode::KeyS, GameKey::S),
    (KeyCode::KeyW, GameKey::W),
    (KeyCode::KeyZ, GameKey::Z),
    (KeyCode::AltLeft, GameKey::AltLeft),
    (KeyCode::AltRight, GameKey::AltRight),
    (KeyCode::ControlLeft, GameKey::Control),
    (KeyCode::ControlRight, GameKey::Control),
    (KeyCode::ShiftLeft, GameKey::Shift),
    (KeyCode::ShiftRight, GameKey::Shift),
    (KeyCode::ArrowDown, GameKey::ArrowDown),
    (KeyCode::ArrowLeft, GameKey::ArrowLeft),
    (KeyCode::ArrowRight, GameKey::ArrowRight),
    (KeyCode::ArrowUp, GameKey::ArrowUp),
    (KeyCode::Backspace, GameKey::Backspace),
    (KeyCode::Enter, GameKey::Enter),
    (KeyCode::Escape, GameKey::Escape),
    (KeyCode::Space, GameKey::Space),
    (KeyCode::Tab, GameKey::Tab),
    (KeyCode::Home, GameKey::Home),
    (KeyCode::Numpad5, GameKey::Numpad5),
    (KeyCode::F1, GameKey::F1),
    (KeyCode::F2, GameKey::F2),
    (KeyCode::F3, GameKey::F3),
    (KeyCode::F4, GameKey::F4),
    (KeyCode::F5, GameKey::F5),
    (KeyCode::F6, GameKey::F6),
    (KeyCode::F7, GameKey::F7),
    (KeyCode::F8, GameKey::F8),
    (KeyCode::F9, GameKey::F9),
    (KeyCode::F10, GameKey::F10),
    (KeyCode::F11, GameKey::F11),
    (KeyCode::F12, GameKey::F12),
];

impl Bindings {
    /// 預設對應表：Ebiten 的實體鍵名（Go 版的實際按鍵）。
    pub fn ebiten() -> Bindings {
        let map = EBITEN_KEYS.iter().map(|&(code, key)| (PhysicalInput::Keyboard(code), key)).collect();
        Bindings { map }
    }

    /// 邏輯鍵對應的第一個鍵盤按鍵（腳本輸入注入用；`Control`／`Shift` 取左鍵）。
    pub fn keyboard_code_for(&self, key: GameKey) -> Option<KeyCode> {
        EBITEN_KEYS
            .iter()
            .find(|(code, k)| *k == key && self.map.get(&PhysicalInput::Keyboard(*code)) == Some(&key))
            .map(|(code, _)| *code)
    }

    /// 實體輸入 → 邏輯狀態。`lb` 是本幀畫格在視窗中的位置（滑鼠與觸控座標換算用）。
    pub fn logical_state(&self, snap: &PhysicalSnapshot, lb: Letterbox) -> RawInputState {
        let mut keys = BTreeSet::new();
        let mut other_keys = BTreeSet::new();
        for input in &snap.held {
            match self.map.get(input) {
                Some(&key) => {
                    keys.insert(key);
                }
                None => match input {
                    PhysicalInput::Keyboard(KeyCode::Unidentified(_)) => {}
                    PhysicalInput::Keyboard(code) => {
                        other_keys.insert(other_key_id(*code));
                    }
                },
            }
        }
        let cursor = snap.cursor.map_or((0, 0), |(x, y)| lb.to_logical(x, y));
        let touches = snap
            .touches
            .iter()
            .map(|&(id, x, y)| {
                let (lx, ly) = lb.to_logical(x, y);
                (id, lx, ly)
            })
            .collect();
        RawInputState {
            keys,
            other_keys,
            touches,
            cursor,
            mouse_left: snap.mouse_left,
            input_chars: snap.chars.clone(),
        }
    }
}

/// 沒有對應的鍵的穩定編號（FNV-1a over `KeyCode` 的 Debug 名稱；只用來區分不同的鍵）。
fn other_key_id(code: KeyCode) -> u32 {
    let mut h: u32 = 0x811c_9dc5;
    for b in format!("{code:?}").bytes() {
        h ^= b as u32;
        h = h.wrapping_mul(0x0100_0193);
    }
    h
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::video::letterbox;

    #[test]
    fn every_game_key_except_alt_has_a_keyboard_binding() {
        let b = Bindings::ebiten();
        for key in GameKey::ALL {
            assert_eq!(b.keyboard_code_for(key).is_some(), key != GameKey::Alt, "{key:?}");
        }
        assert_eq!(b.keyboard_code_for(GameKey::Control), Some(KeyCode::ControlLeft));
    }

    #[test]
    fn logical_state_maps_keys_others_and_pointer() {
        let b = Bindings::ebiten();
        let snap = PhysicalSnapshot {
            held: vec![
                PhysicalInput::Keyboard(KeyCode::ShiftRight),
                PhysicalInput::Keyboard(KeyCode::KeyQ),
                PhysicalInput::Keyboard(KeyCode::NumpadEnter),
                PhysicalInput::Keyboard(KeyCode::Enter),
            ],
            mouse_left: true,
            cursor: Some((112.0, 27.0)),
            touches: vec![(3, 1389.0, 824.0)],
            chars: vec!['q'],
        };
        let raw = b.logical_state(&snap, letterbox(1500, 850));
        assert_eq!(raw.keys, [GameKey::Shift, GameKey::Enter].into());
        assert_eq!(raw.other_keys.len(), 2, "Q 與 NumpadEnter（Ebiten KeyNumpadEnter 與 KeyEnter 不同）");
        assert_eq!(raw.cursor, (1, 1));
        assert_eq!(raw.touches, vec![(3, 639, 399)]);
        assert!(raw.mouse_left);
        assert_eq!(raw.input_chars, vec!['q']);
    }
}
