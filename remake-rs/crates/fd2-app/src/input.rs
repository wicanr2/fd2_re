//! 輸入對應（R5 階段 4，r5w；r5i 加滑鼠右鍵、滾輪、焦點與手把）：實體輸入 → [`RawInputState`]，再由 `fd2-game` 的
//! [`fd2_game::game::EbitenInputState`] 組成每個邏輯幀的 `TickInput`，正式執行時再經 `fd2-game` 的輸入對應層
//! （`player_input.rs`，Go v.1.1.15 的 `player_input.go`）換成邏輯鍵。
//!
//! 分兩層：
//!
//! 1. [`PhysicalSnapshot`]：本顯示幀的實體輸入（Bevy 的按住中按鍵、滑鼠左右鍵與游標、滾輪、觸控點、輸入文字、
//!    視窗焦點、手把），座標是視窗實體像素。
//! 2. [`Bindings`]：實體鍵 → Ebiten 鍵的對應表。[`fd2_game::game::Key`] 列出的 33 個鍵放進 `keys`，其餘 Ebiten 鍵以
//!    Ebiten 的 `ebiten.Key` 數值放進 `other_keys`（玩家可以把動作改綁到 Q、PageUp 這類鍵；設定面板讀 Delete、R 等），
//!    Ebiten 沒有的 Bevy 鍵（多媒體鍵、`Fn` 等）用大於 `KeyMax` 的編號，只算「任意鍵」。玩家的自訂綁定在 `fd2-game`
//!    （兩版的 `controls-settings.json` 互通），不在這裡。
//!
//! 對照 Go 實際讀的輸入見 `fd2-game/PORTING.md`「R5 階段 4：r5w」的輸入對照表與「R5 階段 4：r5i」。

use std::collections::{BTreeSet, HashMap};

use bevy::input::gamepad::GamepadButton;
use bevy::input::keyboard::KeyCode;
use fd2_game::game::{GamepadSnapshot, Key as GameKey, RawDevices, RawInputState};
use fd2_game::go_ebiten_keys as ek;

use crate::video::Letterbox;

/// 一個實體輸入。
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
    /// r5i：滑鼠右鍵、本顯示幀累積的滾輪格數（上為正）、視窗焦點、手把（Ebiten 標準配置的按鈕與方向）。
    pub devices: RawDevices,
}

/// 實體鍵 → Ebiten 鍵的對應表。
#[derive(Debug, Clone)]
pub struct Bindings {
    map: HashMap<PhysicalInput, GameKey>,
    /// [`GameKey`] 沒有列出的 Ebiten 鍵：Bevy 鍵 → `ebiten.Key` 數值。
    others: HashMap<KeyCode, u32>,
}

/// Ebiten 鍵與 Bevy `KeyCode` 的一對一對應（兩邊都是實體鍵位，以美式鍵盤命名）。Ebiten 的 `KeyControl`／`KeyShift`
/// 是左右任一個，`fd2-game` 的 [`GameKey`] 沒有分左右，所以左右都對到同一個邏輯鍵；`KeyAlt` 由 `fd2-game` 從
/// `AltLeft`／`AltRight` 算出，沒有實體鍵。其餘鍵見 [`ebiten_other_code`]。
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

/// [`EBITEN_KEYS`] 以外、Ebiten 也有的實體鍵：Bevy 的 `KeyCode` 名稱是 W3C `KeyboardEvent.code`，Ebiten 的鍵名也是
/// （字母去掉 `Key` 前綴、`SuperLeft`／`SuperRight` 是 Ebiten 的 `MetaLeft`／`MetaRight`）。Ebiten 的虛擬鍵
/// （`Alt`、`Control`、`Shift`、`Meta`）不是實體鍵，不對應。
fn ebiten_other_code(code: KeyCode) -> Option<u32> {
    let name = format!("{code:?}");
    let name = match code {
        KeyCode::SuperLeft => String::from("MetaLeft"),
        KeyCode::SuperRight => String::from("MetaRight"),
        _ => name.strip_prefix("Key").filter(|rest| rest.len() == 1).map(str::to_string).unwrap_or(name),
    };
    let k = ek::KEY_NAMES.iter().position(|n| *n == name)? as ek::EbitenKey;
    (k < ek::KEY_ALT).then_some(k as u32)
}

/// Ebiten 沒有的 Bevy 鍵的穩定編號（FNV-1a over `KeyCode` 的 Debug 名稱，最高位元設 1，不與 Ebiten 數值重疊）。
fn non_ebiten_key_id(code: KeyCode) -> u32 {
    let mut h: u32 = 0x811c_9dc5;
    for b in format!("{code:?}").bytes() {
        h ^= b as u32;
        h = h.wrapping_mul(0x0100_0193);
    }
    0x8000_0000 | h
}

/// Bevy 手把按鈕 → Go `padStandardButtons` 的按鈕順序（`PAD_BUTTON_IDS`：a、b、x、y、lb、rb、lt、rt、back、start、l3、
/// r3、十字鍵上下左右）。Ebiten 標準配置的 FrontTop 是 Bevy 的 `LeftTrigger`／`RightTrigger`（肩鍵），FrontBottom 是
/// `LeftTrigger2`／`RightTrigger2`（扳機）；中央的 Guide（Bevy `Mode`）Go 不開放。
pub const PAD_BUTTONS: [GamepadButton; 16] = [
    GamepadButton::South,
    GamepadButton::East,
    GamepadButton::West,
    GamepadButton::North,
    GamepadButton::LeftTrigger,
    GamepadButton::RightTrigger,
    GamepadButton::LeftTrigger2,
    GamepadButton::RightTrigger2,
    GamepadButton::Select,
    GamepadButton::Start,
    GamepadButton::LeftThumb,
    GamepadButton::RightThumb,
    GamepadButton::DPadUp,
    GamepadButton::DPadDown,
    GamepadButton::DPadLeft,
    GamepadButton::DPadRight,
];

/// 一支手把的狀態轉成 `fd2-game` 的快照：按鈕依 [`PAD_BUTTONS`]，左搖桿換成 Ebiten 標準配置的方向（Bevy 的 Y 軸上為正，
/// Ebiten 下為正）。
pub fn gamepad_snapshot(
    id: u64,
    name: &str,
    pressed: impl Fn(GamepadButton) -> bool,
    stick: (f32, f32),
) -> GamepadSnapshot {
    let mut pad = GamepadSnapshot {
        id,
        name: name.to_string(),
        stick_x: stick.0 as f64,
        stick_y: -(stick.1 as f64),
        ..Default::default()
    };
    for (i, &button) in PAD_BUTTONS.iter().enumerate() {
        pad.buttons[i] = pressed(button);
    }
    pad
}

impl Bindings {
    /// 預設對應表：Ebiten 的實體鍵名（Go 版的實際按鍵）。
    pub fn ebiten() -> Bindings {
        let map = EBITEN_KEYS.iter().map(|&(code, key)| (PhysicalInput::Keyboard(code), key)).collect();
        Bindings { map, others: HashMap::new() }
    }

    /// 邏輯鍵對應的第一個鍵盤按鍵（腳本輸入注入用；`Control`／`Shift` 取左鍵）。
    pub fn keyboard_code_for(&self, key: GameKey) -> Option<KeyCode> {
        EBITEN_KEYS
            .iter()
            .find(|(code, k)| *k == key && self.map.get(&PhysicalInput::Keyboard(*code)) == Some(&key))
            .map(|(code, _)| *code)
    }

    fn other_code(&mut self, code: KeyCode) -> u32 {
        *self.others.entry(code).or_insert_with(|| ebiten_other_code(code).unwrap_or_else(|| non_ebiten_key_id(code)))
    }

    /// 實體輸入 → 邏輯狀態。`lb` 是本幀畫格在視窗中的位置（滑鼠與觸控座標換算用）。
    pub fn logical_state(&mut self, snap: &PhysicalSnapshot, lb: Letterbox) -> RawInputState {
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
                        other_keys.insert(self.other_code(*code));
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
            devices: snap.devices.clone(),
        }
    }
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
        let mut b = Bindings::ebiten();
        let snap = PhysicalSnapshot {
            held: vec![
                PhysicalInput::Keyboard(KeyCode::ShiftRight),
                PhysicalInput::Keyboard(KeyCode::KeyQ),
                PhysicalInput::Keyboard(KeyCode::NumpadEnter),
                PhysicalInput::Keyboard(KeyCode::Enter),
                PhysicalInput::Keyboard(KeyCode::MediaPlayPause),
            ],
            mouse_left: true,
            cursor: Some((112.0, 27.0)),
            touches: vec![(3, 1389.0, 824.0)],
            chars: vec!['q'],
            devices: RawDevices { focused: true, mouse_right: true, wheel_y: 1.0, pads: Vec::new() },
        };
        let raw = b.logical_state(&snap, letterbox(1500, 850));
        assert_eq!(raw.keys, [GameKey::Shift, GameKey::Enter].into());
        // r5i：Ebiten 鍵用 `ebiten.Key` 數值（Q=16、NumpadEnter=88，Ebiten 的 KeyNumpadEnter 與 KeyEnter 不同）；
        // Ebiten 沒有的鍵最高位元是 1。
        let others: Vec<u32> = raw.other_keys.iter().copied().collect();
        assert_eq!(&others[..2], &[ek::KEY_Q as u32, ek::KEY_NUMPAD_ENTER as u32]);
        assert_eq!(others.len(), 3);
        assert!(others[2] & 0x8000_0000 != 0);
        assert_eq!(raw.cursor, (1, 1));
        assert_eq!(raw.touches, vec![(3, 639, 399)]);
        assert!(raw.mouse_left);
        assert_eq!(raw.input_chars, vec!['q']);
        assert!(raw.devices.focused && raw.devices.mouse_right && raw.devices.wheel_y == 1.0);
    }

    /// Bevy 的 W3C 鍵名與 Ebiten 鍵名對得上：Ebiten 的每個實體鍵（虛擬鍵以外）都有一個 Bevy 鍵。
    #[test]
    fn ebiten_physical_keys_all_have_a_bevy_key() {
        let bevy = [
            KeyCode::KeyB,
            KeyCode::Backquote,
            KeyCode::BracketLeft,
            KeyCode::ContextMenu,
            KeyCode::Delete,
            KeyCode::Digit0,
            KeyCode::Digit9,
            KeyCode::End,
            KeyCode::Equal,
            KeyCode::Insert,
            KeyCode::SuperLeft,
            KeyCode::SuperRight,
            KeyCode::NumLock,
            KeyCode::Numpad0,
            KeyCode::NumpadDecimal,
            KeyCode::PageDown,
            KeyCode::PageUp,
            KeyCode::Pause,
            KeyCode::PrintScreen,
            KeyCode::Quote,
            KeyCode::ScrollLock,
            KeyCode::Semicolon,
            KeyCode::Slash,
            KeyCode::CapsLock,
        ];
        let names: Vec<&str> =
            bevy.iter().map(|&c| ek::key_string(ebiten_other_code(c).unwrap() as ek::EbitenKey)).collect();
        assert_eq!(
            names,
            [
                "B",
                "Backquote",
                "BracketLeft",
                "ContextMenu",
                "Delete",
                "Digit0",
                "Digit9",
                "End",
                "Equal",
                "Insert",
                "MetaLeft",
                "MetaRight",
                "NumLock",
                "Numpad0",
                "NumpadDecimal",
                "PageDown",
                "PageUp",
                "Pause",
                "PrintScreen",
                "Quote",
                "ScrollLock",
                "Semicolon",
                "Slash",
                "CapsLock"
            ]
        );
        assert_eq!(ebiten_other_code(KeyCode::Meta), None, "Ebiten 的 Meta 是虛擬鍵");
        assert_eq!(ebiten_other_code(KeyCode::F13), None);
        assert_eq!(ebiten_other_code(KeyCode::IntlBackslash), None);
    }

    #[test]
    fn gamepad_buttons_follow_go_order_and_stick_y_is_flipped() {
        let pad = gamepad_snapshot(9, "pad", |b| matches!(b, GamepadButton::South | GamepadButton::Start), (0.5, 0.25));
        assert_eq!(pad.buttons.iter().filter(|&&p| p).count(), 2);
        assert!(pad.buttons[0] && pad.buttons[9], "a 與 start");
        assert_eq!((pad.stick_x, pad.stick_y, pad.id), (0.5, -0.25, 9));
    }
}
