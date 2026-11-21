//! Go 來源：`remake/cmd/fd2/player_input_bindings.go`（400 行，Go v.1.1.15 `7c710b34`）。R5 階段 4（r5i）移植。
//!
//! 按鍵設定面板的綁定資料與 `controls-settings.json`。遊戲各處讀的是「邏輯按鍵」：原版操作本來就用的 Enter、Esc、
//! 方向鍵與 F 鍵，值沿用 `ebiten.Key`（Rust 是 [`EbitenKey`]）。每個可設定的動作（[`ControlAction`]）對應一個邏輯按鍵，
//! 再各有一個鍵盤鍵與一個手把按鈕。玩家沒改過的動作，鍵盤鍵就是邏輯按鍵本身，所以預設對應是恆等，鍵盤行為與加入
//! 這一層之前相同。對應表的規則見 [`ControlKeyTable::new`]（`player_input.rs`）。
//!
//! 這是重製端的播放器設定，不是原版行為；不寫戰役存檔，也不改遊戲規則。
//!
//! JSON 讀寫用 serde_json（`rust-port-no-go-stdlib`）：寫出的位元組與 Go `json.MarshalIndent(..., "", " ")` 相同
//! （欄位順序、鍵名排序、一個空白縮排、結尾換行），兩版的設定檔互通。讀檔只比接受或拒收；Go `encoding/json` 的欄位名稱
//! 不分大小寫與重複鍵取最後一個，serde 分大小寫、重複鍵拒收（KNOWN_DIFFERENCES `controls-settings-json-decoder`）。

use super::*;
use crate::go_ebiten_keys::{self as ek, EbitenKey};

/// Go：`player_input_bindings.go:23`，`type controlActionGroup int`。
pub(crate) type ControlActionGroup = usize;
/// 移動與游標
pub(crate) const CONTROL_GROUP_MOVEMENT: ControlActionGroup = 0;
/// 確認與取消
pub(crate) const CONTROL_GROUP_CONFIRM: ControlActionGroup = 1;
/// 系統功能鍵
pub(crate) const CONTROL_GROUP_SYSTEM: ControlActionGroup = 2;
pub(crate) const CONTROL_GROUP_COUNT: ControlActionGroup = 3;

/// Xbox 配置（標準手把配置）的實體按鈕；[`PAD_NONE`] 表示未設定。
///
/// Go：`player_input_bindings.go:33`，`type controlPadButton int`。
pub(crate) type ControlPadButton = i32;
pub(crate) const PAD_NONE: ControlPadButton = -1;
pub(crate) const PAD_A: ControlPadButton = 0;
pub(crate) const PAD_B: ControlPadButton = 1;
pub(crate) const PAD_X: ControlPadButton = 2;
pub(crate) const PAD_Y: ControlPadButton = 3;
pub(crate) const PAD_LB: ControlPadButton = 4;
pub(crate) const PAD_RB: ControlPadButton = 5;
pub(crate) const PAD_LT: ControlPadButton = 6;
pub(crate) const PAD_RT: ControlPadButton = 7;
pub(crate) const PAD_BACK: ControlPadButton = 8;
pub(crate) const PAD_START: ControlPadButton = 9;
pub(crate) const PAD_L3: ControlPadButton = 10;
pub(crate) const PAD_R3: ControlPadButton = 11;
pub(crate) const PAD_UP: ControlPadButton = 12;
pub(crate) const PAD_DOWN: ControlPadButton = 13;
pub(crate) const PAD_LEFT: ControlPadButton = 14;
pub(crate) const PAD_RIGHT: ControlPadButton = 15;
pub(crate) const PAD_BUTTON_COUNT: usize = 16;

/// `controls-settings.json` 與面板使用的按鈕名稱。
///
/// Go：`player_input_bindings.go:77`，`var padButtonIDs`。Go 的 `padStandardButtons`（對到 Ebiten 標準手把配置）在
/// 外殼：`fd2-app/src/input.rs` 把 Bevy 的 `GamepadButton` 對到同一組按鈕。
pub(crate) const PAD_BUTTON_IDS: [&str; PAD_BUTTON_COUNT] = [
    "a",
    "b",
    "x",
    "y",
    "lb",
    "rb",
    "lt",
    "rt",
    "back",
    "start",
    "l3",
    "r3",
    "dpad_up",
    "dpad_down",
    "dpad_left",
    "dpad_right",
];

/// Go：`player_input_bindings.go:83`，`func parsePadButton(id string) (controlPadButton, bool)`
pub(crate) fn parse_pad_button(id: &str) -> (ControlPadButton, bool) {
    match PAD_BUTTON_IDS.iter().position(|name| *name == id) {
        Some(button) => (button as ControlPadButton, true),
        None => (PAD_NONE, false),
    }
}

/// 面板的一列。`logical` 是遊戲程式讀的鍵；`default_pad` 是手把預設。
///
/// Go：`player_input_bindings.go:93`，`type controlAction struct`
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct ControlAction {
    pub(crate) id: &'static str,
    pub(crate) group: ControlActionGroup,
    pub(crate) logical: EbitenKey,
    pub(crate) default_pad: ControlPadButton,
}

const fn action(
    id: &'static str,
    group: ControlActionGroup,
    logical: EbitenKey,
    default_pad: ControlPadButton,
) -> ControlAction {
    ControlAction { id, group, logical, default_pad }
}

/// `CONTROL_ACTIONS` 的順序就是面板的順序。手把預設依使用者 2026-11-16 定案：十字鍵與左搖桿
/// 是方向鍵、A＝Enter、B＝Esc、X＝Space、Start＝開設定面板，肩鍵與 Select 不綁。
///
/// Go：`player_input_bindings.go:102`，`var controlActions`
pub(crate) const CONTROL_ACTIONS: [ControlAction; 23] = [
    action("up", CONTROL_GROUP_MOVEMENT, ek::KEY_ARROW_UP, PAD_UP),
    action("down", CONTROL_GROUP_MOVEMENT, ek::KEY_ARROW_DOWN, PAD_DOWN),
    action("left", CONTROL_GROUP_MOVEMENT, ek::KEY_ARROW_LEFT, PAD_LEFT),
    action("right", CONTROL_GROUP_MOVEMENT, ek::KEY_ARROW_RIGHT, PAD_RIGHT),
    action("confirm", CONTROL_GROUP_CONFIRM, ek::KEY_ENTER, PAD_A),
    action("cancel", CONTROL_GROUP_CONFIRM, ek::KEY_ESCAPE, PAD_B),
    action("space", CONTROL_GROUP_CONFIRM, ek::KEY_SPACE, PAD_X),
    action("back", CONTROL_GROUP_CONFIRM, ek::KEY_BACKSPACE, PAD_NONE),
    action("next_unit", CONTROL_GROUP_CONFIRM, ek::KEY_Z, PAD_NONE),
    action("inspect", CONTROL_GROUP_CONFIRM, ek::KEY_HOME, PAD_NONE),
    action("end_turn", CONTROL_GROUP_CONFIRM, ek::KEY_TAB, PAD_NONE),
    action("help", CONTROL_GROUP_SYSTEM, ek::KEY_F1, PAD_NONE),
    action("theme", CONTROL_GROUP_SYSTEM, ek::KEY_F2, PAD_NONE),
    action("audio", CONTROL_GROUP_SYSTEM, ek::KEY_F3, PAD_NONE),
    action("locale", CONTROL_GROUP_SYSTEM, ek::KEY_F4, PAD_NONE),
    action("quick_save", CONTROL_GROUP_SYSTEM, ek::KEY_F5, PAD_NONE),
    action("music", CONTROL_GROUP_SYSTEM, ek::KEY_F6, PAD_NONE),
    action("font", CONTROL_GROUP_SYSTEM, ek::KEY_F7, PAD_NONE),
    action("speed", CONTROL_GROUP_SYSTEM, ek::KEY_F8, PAD_NONE),
    action("quick_load", CONTROL_GROUP_SYSTEM, ek::KEY_F9, PAD_NONE),
    action("poster", CONTROL_GROUP_SYSTEM, ek::KEY_F10, PAD_NONE),
    action("interface", CONTROL_GROUP_SYSTEM, ek::KEY_F11, PAD_NONE),
    action("controls", CONTROL_GROUP_SYSTEM, ek::KEY_F12, PAD_START),
];

/// Go：`player_input_bindings.go:128`，`func controlActionIndex(id string) int`
pub(crate) fn control_action_index(id: &str) -> i64 {
    CONTROL_ACTIONS.iter().position(|a| a.id == id).map_or(-1, |i| i as i64)
}

/// Go：`player_input_bindings.go:138`..`141`（百分比）。
pub(crate) const DEFAULT_STICK_DEADZONE: i64 = 20;
pub(crate) const MIN_STICK_DEADZONE: i64 = 5;
pub(crate) const MAX_STICK_DEADZONE: i64 = 60;
pub(crate) const STICK_DEADZONE_STEP: i64 = 5;

/// Go：`player_input_bindings.go:144`，`const keyUnbound = ebiten.Key(-1)`
pub(crate) const KEY_UNBOUND: EbitenKey = -1;

/// 每個動作目前的鍵盤鍵與手把按鈕（索引同 [`CONTROL_ACTIONS`]）。
///
/// Go：`player_input_bindings.go:147`，`type controlBindings struct`。Go 的切片在 `setKeyboard` 等方法裡就地修改
/// （值接收者共用底層陣列）；Rust 收 `&mut self`。
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct ControlBindings {
    pub(crate) keyboard: Vec<EbitenKey>,
    pub(crate) gamepad: Vec<ControlPadButton>,
    pub(crate) deadzone: i64,
}

impl Default for ControlBindings {
    fn default() -> Self {
        default_control_bindings()
    }
}

/// Go：`player_input_bindings.go:153`，`func defaultControlBindings() controlBindings`
pub(crate) fn default_control_bindings() -> ControlBindings {
    ControlBindings {
        keyboard: CONTROL_ACTIONS.iter().map(|a| a.logical).collect(),
        gamepad: CONTROL_ACTIONS.iter().map(|a| a.default_pad).collect(),
        deadzone: DEFAULT_STICK_DEADZONE,
    }
}

/// 擋掉修飾鍵（Shift／Ctrl／Alt／Meta 留給秘密商店、Alt+Enter 與 Ctrl+F12）與 Esc（設定面板的取消鍵）。
/// Esc 仍是「取消」的預設鍵，只能用恢復預設取回。
///
/// Go：`player_input_bindings.go:175`，`func bindableKey(k ebiten.Key) bool`
pub(crate) fn bindable_key(k: EbitenKey) -> bool {
    if !(0..=ek::KEY_MAX).contains(&k) {
        return false;
    }
    !matches!(
        k,
        ek::KEY_SHIFT
            | ek::KEY_SHIFT_LEFT
            | ek::KEY_SHIFT_RIGHT
            | ek::KEY_CONTROL
            | ek::KEY_CONTROL_LEFT
            | ek::KEY_CONTROL_RIGHT
            | ek::KEY_ALT
            | ek::KEY_ALT_LEFT
            | ek::KEY_ALT_RIGHT
            | ek::KEY_META
            | ek::KEY_META_LEFT
            | ek::KEY_META_RIGHT
            | ek::KEY_ESCAPE
    )
}

impl ControlBindings {
    /// Go：`player_input_bindings.go:189`，`func (b controlBindings) keyboardOwner(k ebiten.Key) int`
    pub(crate) fn keyboard_owner(&self, k: EbitenKey) -> i64 {
        self.keyboard.iter().position(|&bound| bound == k && k != KEY_UNBOUND).map_or(-1, |i| i as i64)
    }

    /// Go：`player_input_bindings.go:198`，`func (b controlBindings) gamepadOwner(button controlPadButton) int`
    pub(crate) fn gamepad_owner(&self, button: ControlPadButton) -> i64 {
        self.gamepad.iter().position(|&bound| bound == button && button != PAD_NONE).map_or(-1, |i| i as i64)
    }

    /// 指定一個動作的鍵盤鍵；同一鍵原本屬於別的動作時，那個動作改成未設定。是否先提示由呼叫端決定（面板先問，
    /// 讀檔直接套用）。
    ///
    /// Go：`player_input_bindings.go:209`，`func (b controlBindings) setKeyboard(action int, k ebiten.Key)`
    pub(crate) fn set_keyboard(&mut self, action: usize, k: EbitenKey) {
        let owner = self.keyboard_owner(k);
        if owner >= 0 && owner as usize != action {
            self.keyboard[owner as usize] = KEY_UNBOUND;
        }
        self.keyboard[action] = k;
    }

    /// Go：`player_input_bindings.go:216`，`func (b controlBindings) setGamepad(action int, button controlPadButton)`
    pub(crate) fn set_gamepad(&mut self, action: usize, button: ControlPadButton) {
        let owner = self.gamepad_owner(button);
        if owner >= 0 && owner as usize != action {
            self.gamepad[owner as usize] = PAD_NONE;
        }
        self.gamepad[action] = button;
    }

    /// 清除該列的鍵盤與手把。
    ///
    /// Go：`player_input_bindings.go:224`，`func (b controlBindings) clearAction(action int)`
    pub(crate) fn clear_action(&mut self, action: usize) {
        (self.keyboard[action], self.gamepad[action]) = (KEY_UNBOUND, PAD_NONE);
    }

    /// 讓該列回到預設；預設鍵若被別列占用，別列改成未設定。
    ///
    /// Go：`player_input_bindings.go:229`，`func (b controlBindings) resetAction(action int)`
    pub(crate) fn reset_action(&mut self, action: usize) {
        self.set_keyboard(action, CONTROL_ACTIONS[action].logical);
        let pad = CONTROL_ACTIONS[action].default_pad;
        if pad != PAD_NONE {
            self.set_gamepad(action, pad);
        } else {
            self.gamepad[action] = PAD_NONE;
        }
    }

    /// 回報恢復預設會從哪一列拿走鍵（-1 表示不會）。
    ///
    /// Go：`player_input_bindings.go:239`，`func (b controlBindings) resetConflict(action int) int`
    pub(crate) fn reset_conflict(&self, action: usize) -> i64 {
        let owner = self.keyboard_owner(CONTROL_ACTIONS[action].logical);
        if owner >= 0 && owner as usize != action {
            return owner;
        }
        let pad = CONTROL_ACTIONS[action].default_pad;
        if pad != PAD_NONE {
            let owner = self.gamepad_owner(pad);
            if owner >= 0 && owner as usize != action {
                return owner;
            }
        }
        -1
    }
}

/// Go：`player_input_bindings.go:251`，`func normalizeStickDeadzone(v int) int`
pub(crate) fn normalize_stick_deadzone(v: i64) -> i64 {
    if !(MIN_STICK_DEADZONE..=MAX_STICK_DEADZONE).contains(&v) {
        return DEFAULT_STICK_DEADZONE;
    }
    v - v % STICK_DEADZONE_STEP
}

// ---- controls-settings.json ----

/// 只記與預設不同的列；空字串表示玩家清除了該列。之後新增的動作沿用預設。
///
/// Go：`player_input_bindings.go:261`，`type controlsSettingsFile struct`。欄位順序同 Go，`omitempty` 的 map 以
/// `None` 表示（Go 的 nil map）。
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub(crate) struct ControlsSettingsFile {
    #[serde(default)]
    pub(crate) schema_version: i64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub(crate) keyboard: Option<BTreeMap<String, String>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub(crate) gamepad: Option<BTreeMap<String, String>>,
    #[serde(default, rename = "stick_deadzone_percent")]
    pub(crate) stick_deadzone: i64,
}

/// Go：`player_input_bindings.go:268`
pub(crate) const CONTROLS_SETTINGS_SCHEMA: i64 = 1;

/// 與遊戲存檔、`fd2_settings.json` 分開，放在同一個使用者資料目錄。
///
/// Go：`player_input_bindings.go:271`，`func controlsSettingsPath() string`
pub(crate) fn controls_settings_path() -> String {
    #[cfg(test)]
    if let Some(path) = CONTROLS_SETTINGS_PATH_OVERRIDE.with(|p| p.borrow().clone()) {
        return path;
    }
    user_data_path("controls-settings.json")
}

#[cfg(test)]
thread_local! {
    /// 測試用：Go 測試把 `userDataDirCached` 換成暫存目錄；Rust 的測試平行執行，改成每個執行緒自己的設定檔路徑。
    pub(crate) static CONTROLS_SETTINGS_PATH_OVERRIDE: std::cell::RefCell<Option<String>> =
        const { std::cell::RefCell::new(None) };
}

/// Go：`player_input_bindings.go:273`，`type controlsLoadStatus int`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) enum ControlsLoadStatus {
    /// 沒有檔案
    #[default]
    Default,
    Ok,
    /// 有無法辨識的列，已略過
    Partial,
    /// 整份無法讀取，改用預設；下次變更時重寫
    Corrupt,
}

/// Go：`player_input_bindings.go:282`，`func loadControlsSettings() (controlBindings, controlsLoadStatus)`
pub(crate) fn load_controls_settings() -> (ControlBindings, ControlsLoadStatus) {
    load_controls_settings_from(&controls_settings_path())
}

/// Go：`player_input_bindings.go:286`，`func loadControlsSettingsFrom(path string) (controlBindings, controlsLoadStatus)`
pub(crate) fn load_controls_settings_from(path: &str) -> (ControlBindings, ControlsLoadStatus) {
    let mut b = default_control_bindings();
    let raw = match std::fs::read(path) {
        Ok(raw) => raw,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return (b, ControlsLoadStatus::Default),
        Err(_) => return (b, ControlsLoadStatus::Corrupt),
    };
    let file: ControlsSettingsFile = match serde_json::from_slice(&raw) {
        Ok(file) => file,
        Err(_) => return (b, ControlsLoadStatus::Corrupt),
    };
    if file.schema_version != CONTROLS_SETTINGS_SCHEMA {
        return (b, ControlsLoadStatus::Corrupt);
    }
    let mut status = ControlsLoadStatus::Ok;
    // 依面板順序套用，結果與玩家在面板上逐列確認取代相同，不受 JSON 物件順序影響。
    for (i, action) in CONTROL_ACTIONS.iter().enumerate() {
        if let Some(name) = file.keyboard.as_ref().and_then(|m| m.get(action.id)) {
            if name.is_empty() {
                b.keyboard[i] = KEY_UNBOUND;
            } else {
                match ek::key_from_name(name) {
                    Some(k) if bindable_key(k) || k == action.logical => b.set_keyboard(i, k),
                    _ => status = ControlsLoadStatus::Partial,
                }
            }
        }
        if let Some(name) = file.gamepad.as_ref().and_then(|m| m.get(action.id)) {
            if name.is_empty() {
                b.gamepad[i] = PAD_NONE;
            } else {
                let (button, valid) = parse_pad_button(name);
                if valid {
                    b.set_gamepad(i, button);
                } else {
                    status = ControlsLoadStatus::Partial;
                }
            }
        }
    }
    for id in file.keyboard.iter().flat_map(|m| m.keys()).chain(file.gamepad.iter().flat_map(|m| m.keys())) {
        if control_action_index(id) < 0 {
            status = ControlsLoadStatus::Partial;
        }
    }
    if file.stick_deadzone != 0 {
        b.deadzone = normalize_stick_deadzone(file.stick_deadzone);
        if b.deadzone != file.stick_deadzone {
            status = ControlsLoadStatus::Partial;
        }
    }
    (b, status)
}

impl ControlBindings {
    /// Go：`player_input_bindings.go:343`，`func (b controlBindings) settingsFile() controlsSettingsFile`
    pub(crate) fn settings_file(&self) -> ControlsSettingsFile {
        let mut file = ControlsSettingsFile {
            schema_version: CONTROLS_SETTINGS_SCHEMA,
            stick_deadzone: self.deadzone,
            ..Default::default()
        };
        for (i, action) in CONTROL_ACTIONS.iter().enumerate() {
            let k = self.keyboard[i];
            if k != action.logical {
                let name = if k != KEY_UNBOUND { ek::key_string(k).to_string() } else { String::new() };
                file.keyboard.get_or_insert_with(BTreeMap::new).insert(action.id.to_string(), name);
            }
            let button = self.gamepad[i];
            if button != action.default_pad {
                let name = if button != PAD_NONE { PAD_BUTTON_IDS[button as usize].to_string() } else { String::new() };
                file.gamepad.get_or_insert_with(BTreeMap::new).insert(action.id.to_string(), name);
            }
        }
        file
    }
}

/// Go：`player_input_bindings.go:371`，`func saveControlsSettings(b controlBindings) error`
pub(crate) fn save_controls_settings(b: &ControlBindings) -> std::io::Result<()> {
    save_controls_settings_to(&controls_settings_path(), b)
}

/// 先寫暫存檔再改名，中途失敗不會留下半份設定。
///
/// Go：`player_input_bindings.go:376`，`func saveControlsSettingsTo(path string, b controlBindings) error`。
/// Go `os.CreateTemp(dir, ".controls-settings-*.json")` 的亂數段，Rust 以行程編號加遞增序號產生、`create_new`
/// 保證不覆蓋既有檔；錯誤文字是 Rust 的（`rust-port-no-go-stdlib`）。
pub(crate) fn save_controls_settings_to(path: &str, b: &ControlBindings) -> std::io::Result<()> {
    use std::io::Write as _;
    use std::sync::atomic::{AtomicU64, Ordering};
    static SERIAL: AtomicU64 = AtomicU64::new(0);
    let mut raw = Vec::new();
    let formatter = serde_json::ser::PrettyFormatter::with_indent(b" ");
    let mut ser = serde_json::Serializer::with_formatter(&mut raw, formatter);
    serde::Serialize::serialize(&b.settings_file(), &mut ser).map_err(std::io::Error::other)?;
    raw.push(b'\n');
    let dir = std::path::Path::new(path).parent().map(|d| d.to_path_buf()).unwrap_or_default();
    std::fs::create_dir_all(if dir.as_os_str().is_empty() { std::path::Path::new(".") } else { &dir })?;
    let temp = loop {
        let name = format!(".controls-settings-{}-{}.json", std::process::id(), SERIAL.fetch_add(1, Ordering::Relaxed));
        let candidate = dir.join(name);
        match std::fs::OpenOptions::new().write(true).create_new(true).open(&candidate) {
            Ok(file) => break (candidate, file),
            Err(err) if err.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(err) => return Err(err),
        }
    };
    let (name, mut file) = temp;
    let write = file.write_all(&raw).and_then(|_| file.sync_all());
    drop(file);
    if let Err(err) = write {
        let _ = std::fs::remove_file(&name);
        return Err(err);
    }
    if let Err(err) = std::fs::rename(&name, path) {
        let _ = std::fs::remove_file(&name);
        return Err(err);
    }
    Ok(())
}
