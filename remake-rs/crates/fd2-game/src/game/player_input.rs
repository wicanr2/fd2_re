//! Go 來源：`remake/cmd/fd2/player_input.go`（378 行，Go v.1.1.15 `7c710b34`）。R5 階段 4（r5i）移植。
//!
//! 輸入對應層：
//!
//! ```text
//! 實體輸入 ──► PlayerInput ──► 邏輯按鍵 ──► 遊戲各處（Game::tick_input）
//!   鍵盤：ControlKeyTable 把實體鍵換成邏輯鍵。
//!   滑鼠、手把：每次 Update 前 step() 取樣一次，換成「虛擬按住」的邏輯鍵。
//!          滑鼠另外產生 PointerTick（指向哪裡、這一刻有沒有點），由各畫面自己換算成格子或列。
//! ```
//!
//! # 與 Go 的對應
//!
//! Go 的遊戲程式呼叫 `keyJustPressed`／`keyPressDuration`／`appendJustPressedKeys`，三者在 `activeInput` 設定時即時查
//! `inpututil` 再經對應表換算。Rust 的遊戲程式讀 [`TickInput`]，所以由 [`PlayerInput::logical_tick`] 在每次 `Update`
//! 開頭把整份實體 [`TickInput`] 換成邏輯的那一份（[`Key::ALL`] 每個鍵的 `duration`、「任意鍵」），實體的那一份留在
//! `TickInput::physical` 給讀實體鍵的地方（Alt+Enter、秘密商店、Ctrl+F12）。兩者對每個鍵的回答相同：Go 在同一次
//! `Update` 內每次查詢都得到同一個值（`inpututil` 每次 `Update` 前更新一次），唯一的例外是 `suppressVirtual` 之後的
//! 查詢，Rust 由 [`Game::suppress_virtual_key`] 重算 `tick_input`。
//!
//! `activeInput` 只在正式執行（Go `main`，Rust [`PlayerRunner`]）設定；測試、黃金畫格與重播驅動端直接設 `tick_input`，
//! 等同 Go 的 `activeInput == nil`（三個讀法直接回 `inpututil` 的值）。
//!
//! 修飾鍵組合（Alt+Enter 全螢幕、修飾鍵秘密商店、Ctrl+F12 除錯 HUD）一律讀實體鍵，不經這一層。

use super::*;
use crate::go_ebiten_keys::{self as ek, EbitenKey};

/// [`Key`]（`fd2-game` 列出的 33 個 Ebiten 鍵）的 Ebiten 數值。Rust 新增（Go 直接用 `ebiten.Key`）。
pub(crate) fn ebiten_key_of(k: Key) -> EbitenKey {
    match k {
        Key::A => ek::KEY_A,
        Key::Alt => ek::KEY_ALT,
        Key::AltLeft => ek::KEY_ALT_LEFT,
        Key::AltRight => ek::KEY_ALT_RIGHT,
        Key::ArrowDown => ek::KEY_ARROW_DOWN,
        Key::ArrowLeft => ek::KEY_ARROW_LEFT,
        Key::ArrowRight => ek::KEY_ARROW_RIGHT,
        Key::ArrowUp => ek::KEY_ARROW_UP,
        Key::Backspace => ek::KEY_BACKSPACE,
        Key::Control => ek::KEY_CONTROL,
        Key::D => ek::KEY_D,
        Key::Enter => ek::KEY_ENTER,
        Key::Escape => ek::KEY_ESCAPE,
        Key::F1 => ek::KEY_F1,
        Key::F10 => ek::KEY_F10,
        Key::F11 => ek::KEY_F11,
        Key::F12 => ek::KEY_F12,
        Key::F2 => ek::KEY_F2,
        Key::F3 => ek::KEY_F3,
        Key::F4 => ek::KEY_F4,
        Key::F5 => ek::KEY_F5,
        Key::F6 => ek::KEY_F6,
        Key::F7 => ek::KEY_F7,
        Key::F8 => ek::KEY_F8,
        Key::F9 => ek::KEY_F9,
        Key::Home => ek::KEY_HOME,
        Key::Numpad5 => ek::KEY_NUMPAD5,
        Key::S => ek::KEY_S,
        Key::Shift => ek::KEY_SHIFT,
        Key::Space => ek::KEY_SPACE,
        Key::Tab => ek::KEY_TAB,
        Key::W => ek::KEY_W,
        Key::Z => ek::KEY_Z,
    }
}

/// Ebiten 數值 → [`Key`]（不在 [`Key`] 的鍵回 `None`）。
pub(crate) fn game_key_of(k: EbitenKey) -> Option<Key> {
    Key::ALL.into_iter().find(|&key| ebiten_key_of(key) == k)
}

/// 實體鍵 `k` 本幀的按住時間（Go `inpututil.KeyPressDuration(k)`）。[`Key`] 列出的鍵讀 `press_duration`，其他 Ebiten 鍵讀
/// `other_press_duration`（外殼以 Ebiten 數值當編號）。`ControlLeft`／`ShiftLeft` 等左右鍵外殼併進 `Control`／`Shift`，
/// 這裡回 0；它們不能指定給動作（[`bindable_key`]），也不是遊戲讀的邏輯鍵，只會以恆等出現在對應表裡。
pub(crate) fn physical_key_duration(t: &TickInput, k: EbitenKey) -> i64 {
    match game_key_of(k) {
        Some(key) => t.key_press_duration(key),
        None if (0..=ek::KEY_MAX).contains(&k) => t.other_press_duration.get(&(k as u32)).copied().unwrap_or(0),
        None => 0,
    }
}

/// 本幀剛按下的實體鍵，依 Ebiten 數值遞增（Go `inpututil.AppendJustPressedKeys(nil)`）。外殼給的非 Ebiten 鍵不列入。
pub(crate) fn physical_just_pressed_keys(t: &TickInput) -> Vec<EbitenKey> {
    let mut keys: Vec<EbitenKey> = t.pressed.iter().map(|&k| ebiten_key_of(k)).collect();
    keys.extend(
        t.other_press_duration
            .iter()
            .filter(|&(&code, &d)| d == 1 && code <= ek::KEY_MAX as u32)
            .map(|(&code, _)| code as EbitenKey),
    );
    keys.sort_unstable();
    keys.dedup();
    keys
}

/// `produce[實體鍵] = 邏輯鍵`（-1 表示這個實體鍵不產生任何邏輯鍵）。
/// 規則：所有動作的邏輯鍵先設為不產生，再把每個動作目前的鍵盤鍵指向該動作的邏輯鍵；其餘鍵恆等。
/// 預設綁定下每個動作的鍵盤鍵就是自己的邏輯鍵，所以整張表是恆等。
///
/// Go：`player_input.go:54`，`type controlKeyTable struct`
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct ControlKeyTable {
    pub(crate) produce: Vec<EbitenKey>,
    /// 邏輯鍵 → 會產生它的實體鍵
    pub(crate) physical: Vec<Vec<EbitenKey>>,
}

impl ControlKeyTable {
    /// Go：`player_input.go:59`，`func newControlKeyTable(b controlBindings) *controlKeyTable`
    pub(crate) fn new(b: &ControlBindings) -> ControlKeyTable {
        let n = (ek::KEY_MAX + 1) as usize;
        let mut produce: Vec<EbitenKey> = (0..n as EbitenKey).collect();
        for action in &CONTROL_ACTIONS {
            produce[action.logical as usize] = KEY_UNBOUND;
        }
        for (i, action) in CONTROL_ACTIONS.iter().enumerate() {
            let k = b.keyboard[i];
            if (0..=ek::KEY_MAX).contains(&k) {
                produce[k as usize] = action.logical;
            }
        }
        let mut physical = vec![Vec::new(); n];
        for (k, &logical) in produce.iter().enumerate() {
            if logical >= 0 {
                physical[logical as usize].push(k as EbitenKey);
            }
        }
        ControlKeyTable { produce, physical }
    }
}

/// 這一次 Update 的滑鼠指向，座標是 640×400 遊戲畫布。只有滑鼠真的移動或按下左鍵的那一刻 `active` 才為 true；
/// 靜止的滑鼠不影響游標與選單。
///
/// Go：`player_input.go:82`，`type pointerTick struct`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct PointerTick {
    pub(crate) active: bool,
    pub(crate) clicked: bool,
    pub(crate) x: i64,
    pub(crate) y: i64,
}

/// 一次取樣的滑鼠與手把狀態；測試用假的快照驅動 [`PlayerInput::step`]。
///
/// Go：`player_input.go:88`，`type inputDeviceSnapshot struct`
#[derive(Debug, Clone, Default, PartialEq)]
pub struct InputDeviceSnapshot {
    pub focused: bool,
    pub cursor_x: i64,
    pub cursor_y: i64,
    pub left: bool,
    pub right: bool,
    pub left_just: bool,
    pub right_just: bool,
    pub wheel_y: f64,
    pub pads: Vec<GamepadSnapshot>,
}

/// Go：`player_input.go:97`，`type gamepadSnapshot struct`。`id` 是 Rust 新增：Go 的面板直接以 Ebiten 的手把 ID 查
/// `inpututil`，Rust 由 runner 依 `id` 記每支手把的按住時間（[`DeviceInputState`]）。
#[derive(Debug, Clone, Default, PartialEq)]
pub struct GamepadSnapshot {
    pub id: u64,
    pub name: String,
    pub buttons: [bool; PAD_BUTTON_COUNT],
    pub stick_x: f64,
    pub stick_y: f64,
}

/// Go：`player_input.go:103`，`type playerInput struct`
#[derive(Debug, Clone, PartialEq)]
pub(crate) struct PlayerInput {
    pub(crate) bindings: ControlBindings,
    pub(crate) status: ControlsLoadStatus,
    pub(crate) table: ControlKeyTable,

    /// 虛擬按住時間（滑鼠鍵、手把）
    pub(crate) held: BTreeMap<EbitenKey, i64>,
    /// 只維持這一刻的按下（滾輪）
    pub(crate) pulse: BTreeSet<EbitenKey>,
    /// 這一刻作廢的虛擬按下（點到不能選的項目）
    pub(crate) suppressed: BTreeSet<EbitenKey>,

    pub(crate) cursor_x: i64,
    pub(crate) cursor_y: i64,
    pub(crate) cursor_known: bool,
    /// 從遊戲區開始按下的滑鼠鍵
    pub(crate) mouse_left: bool,
    pub(crate) mouse_right: bool,
    pub(crate) wheel: f64,
    /// 失焦、斷線或關面板後，要等手把全部放開、搖桿回中央才再讀
    pub(crate) pad_latched: bool,
    pub(crate) pad_count: i64,
    pub(crate) pad_names: Vec<String>,
}

impl PlayerInput {
    /// Go：`player_input.go:121`，`func newPlayerInput(b controlBindings, status controlsLoadStatus) *playerInput`
    pub(crate) fn new(b: ControlBindings, status: ControlsLoadStatus) -> PlayerInput {
        let mut input = PlayerInput {
            table: ControlKeyTable::new(&b),
            bindings: b,
            status,
            held: BTreeMap::new(),
            pulse: BTreeSet::new(),
            suppressed: BTreeSet::new(),
            cursor_x: 0,
            cursor_y: 0,
            cursor_known: false,
            mouse_left: false,
            mouse_right: false,
            wheel: 0.0,
            pad_latched: false,
            pad_count: 0,
            pad_names: Vec::new(),
        };
        let b = input.bindings.clone();
        input.set_bindings(b);
        input
    }

    /// Go：`player_input.go:127`，`func (in *playerInput) setBindings(b controlBindings)`
    pub(crate) fn set_bindings(&mut self, b: ControlBindings) {
        self.bindings = b;
        self.bindings.deadzone = normalize_stick_deadzone(self.bindings.deadzone);
        self.table = ControlKeyTable::new(&self.bindings);
    }

    /// Go：`player_input.go:133`，`func (in *playerInput) keyboardDuration(k ebiten.Key) int`
    pub(crate) fn keyboard_duration(&self, physical: &TickInput, k: EbitenKey) -> i64 {
        if !(0..=ek::KEY_MAX).contains(&k) {
            return physical_key_duration(physical, k);
        }
        self.table.physical[k as usize].iter().map(|&p| physical_key_duration(physical, p)).max().unwrap_or(0).max(0)
    }

    /// Go：`player_input.go:146`，`func (in *playerInput) virtualDuration(k ebiten.Key) int`
    pub(crate) fn virtual_duration(&self, k: EbitenKey) -> i64 {
        if self.suppressed.contains(&k) {
            return 0;
        }
        if self.pulse.contains(&k) {
            return 1;
        }
        self.held.get(&k).copied().unwrap_or(0)
    }

    /// Go：`player_input.go:156`，`func (in *playerInput) duration(k ebiten.Key) int`
    pub(crate) fn duration(&self, physical: &TickInput, k: EbitenKey) -> i64 {
        self.keyboard_duration(physical, k).max(self.virtual_duration(k))
    }

    /// Go：`player_input.go:164`，`func (in *playerInput) appendJustPressed(keys []ebiten.Key) []ebiten.Key`
    pub(crate) fn append_just_pressed(&self, physical: &TickInput, mut keys: Vec<EbitenKey>) -> Vec<EbitenKey> {
        let mut seen = BTreeSet::new();
        for p in physical_just_pressed_keys(physical) {
            let logical = if (0..=ek::KEY_MAX).contains(&p) { self.table.produce[p as usize] } else { p };
            if logical >= 0 && !seen.contains(&logical) && self.duration(physical, logical) == 1 {
                seen.insert(logical);
                keys.push(logical);
            }
        }
        let mut virtual_keys: Vec<EbitenKey> = self.held.keys().copied().chain(self.pulse.iter().copied()).collect();
        virtual_keys.sort_unstable();
        for k in virtual_keys {
            if !seen.contains(&k) && self.virtual_duration(k) == 1 && self.duration(physical, k) == 1 {
                seen.insert(k);
                keys.push(k);
            }
        }
        keys
    }

    /// 作廢這一刻由滑鼠或手把產生的按下；鍵盤照常。
    ///
    /// Go：`player_input.go:194`，`func (in *playerInput) suppressVirtual(k ebiten.Key)`
    pub(crate) fn suppress_virtual(&mut self, k: EbitenKey) {
        self.suppressed.insert(k);
    }

    /// Rust 新增：Go 三個讀法（`keyJustPressed`、`keyPressDuration`、`appendJustPressedKeys`）對這一刻的回答，組成遊戲讀的
    /// [`TickInput`]。`press_duration`／`held`／`pressed` 是 [`Key::ALL`] 每個鍵的 `duration`；`any_just_pressed()` 與
    /// `appendJustPressed` 非空相同：[`Key`] 以外的邏輯鍵（例如沒有指定給動作的 Q）由 `other_just_pressed` 表示。
    /// 遊戲不讀邏輯的「剛放開」，`released` 沿用實體的（只有 Go 的除錯觀測器讀）。滑鼠、觸控與文字照實體的。
    pub(crate) fn logical_tick(&self, physical: TickInput) -> TickInput {
        let mut out = TickInput {
            released: physical.released.clone(),
            other_press_duration: physical.other_press_duration.clone(),
            just_pressed_touches: physical.just_pressed_touches.clone(),
            touches: physical.touches.clone(),
            cursor: physical.cursor,
            mouse_left_held: physical.mouse_left_held,
            mouse_left_just_pressed: physical.mouse_left_just_pressed,
            mouse_left_just_released: physical.mouse_left_just_released,
            input_chars: physical.input_chars.clone(),
            ..Default::default()
        };
        for k in Key::ALL {
            let d = self.duration(&physical, ebiten_key_of(k));
            if d > 0 {
                out.held.insert(k);
                out.press_duration.insert(k, d);
                if d == 1 {
                    out.pressed.insert(k);
                }
            }
        }
        let keys = self.append_just_pressed(&physical, Vec::new());
        out.other_just_pressed = keys.iter().any(|&k| game_key_of(k).is_none())
            || physical.other_press_duration.iter().any(|(&code, &d)| d == 1 && code > ek::KEY_MAX as u32);
        out.physical = Some(Box::new(physical));
        out
    }

    /// 每次 Update 取樣一次。`game_input` 為 false（設定面板、能力設定或工具列吃掉輸入）時，手把照樣計時（關掉面板的
    /// 那一下不會又傳進遊戲），滑鼠鍵不算進遊戲。`accept` 把視窗座標換成遊戲畫布座標，不在遊戲區（工具列、海報側欄、
    /// 下拉選單）回 false。
    ///
    /// Go：`player_input.go:201`，`func (in *playerInput) step(snap inputDeviceSnapshot, gameInput bool, accept func(x, y int) (int, int, bool)) pointerTick`
    pub(crate) fn step(
        &mut self,
        snap: &InputDeviceSnapshot,
        game_input: bool,
        accept: Option<&dyn Fn(i64, i64) -> (i64, i64, bool)>,
    ) -> PointerTick {
        self.pulse.clear();
        self.suppressed.clear();
        let mut held = BTreeSet::new();
        let mut ptr = PointerTick::default();
        if !snap.focused {
            (self.mouse_left, self.mouse_right, self.cursor_known, self.wheel) = (false, false, false, 0.0);
            self.pad_latched = true;
        } else {
            ptr = self.step_mouse(snap, game_input, accept, &mut held);
            self.step_gamepads(snap, &mut held);
        }
        self.held.retain(|k, _| held.contains(k));
        for k in held {
            *self.held.entry(k).or_insert(0) += 1;
        }
        ptr
    }

    /// Go：`player_input.go:228`，`func (in *playerInput) stepMouse(...) pointerTick`
    fn step_mouse(
        &mut self,
        snap: &InputDeviceSnapshot,
        game_input: bool,
        accept: Option<&dyn Fn(i64, i64) -> (i64, i64, bool)>,
        held: &mut BTreeSet<EbitenKey>,
    ) -> PointerTick {
        let mut ptr = PointerTick::default();
        let (mut x, mut y, mut ok) = (0, 0, false);
        if game_input && let Some(accept) = accept {
            (x, y, ok) = accept(snap.cursor_x, snap.cursor_y);
        }
        let moved = self.cursor_known && (snap.cursor_x != self.cursor_x || snap.cursor_y != self.cursor_y);
        (self.cursor_x, self.cursor_y, self.cursor_known) = (snap.cursor_x, snap.cursor_y, true);
        if !snap.left {
            self.mouse_left = false;
        } else if snap.left_just && ok {
            self.mouse_left = true;
            ptr.clicked = true;
        }
        if !snap.right {
            self.mouse_right = false;
        } else if snap.right_just && ok {
            self.mouse_right = true;
        }
        if self.mouse_left {
            held.insert(ek::KEY_ENTER);
        }
        if self.mouse_right {
            held.insert(ek::KEY_ESCAPE);
        }
        // 滾輪一格是一次上／下；觸控板的小數位移累積到一格才算。
        if ok {
            // Go `math.Max(-1.5, math.Min(1.5, in.wheel+snap.wheelY))`；NaN 只會來自外殼，兩邊都不會出現。
            self.wheel = (self.wheel + snap.wheel_y).clamp(-1.5, 1.5);
            if self.wheel >= 1.0 {
                self.pulse.insert(ek::KEY_ARROW_UP);
                self.wheel -= 1.0;
            } else if self.wheel <= -1.0 {
                self.pulse.insert(ek::KEY_ARROW_DOWN);
                self.wheel += 1.0;
            }
        } else {
            self.wheel = 0.0;
        }
        if ok && (moved || ptr.clicked) {
            (ptr.active, ptr.x, ptr.y) = (true, x, y);
        }
        ptr
    }

    /// Go：`player_input.go:272`，`func (in *playerInput) stepGamepads(snap inputDeviceSnapshot, held map[ebiten.Key]bool)`
    fn step_gamepads(&mut self, snap: &InputDeviceSnapshot, held: &mut BTreeSet<EbitenKey>) {
        if snap.pads.len() as i64 != self.pad_count {
            self.pad_latched = true; // 接上或拔掉手把：所有按鈕先放開、搖桿回中央才開始讀
        }
        self.pad_count = snap.pads.len() as i64;
        self.pad_names.clear();
        let mut buttons = [false; PAD_BUTTON_COUNT];
        let (mut stick_x, mut stick_y) = (0.0f64, 0.0f64);
        for pad in &snap.pads {
            self.pad_names.push(pad.name.clone());
            for (button, &pressed) in pad.buttons.iter().enumerate() {
                buttons[button] = buttons[button] || pressed;
            }
            if pad.stick_x.abs() > stick_x.abs() {
                stick_x = pad.stick_x;
            }
            if pad.stick_y.abs() > stick_y.abs() {
                stick_y = pad.stick_y;
            }
        }
        let direction = stick_direction(stick_x, stick_y, self.bindings.deadzone);
        if self.pad_latched {
            if buttons.iter().any(|&pressed| pressed) {
                return;
            }
            if direction != KEY_UNBOUND {
                return;
            }
            self.pad_latched = false;
        }
        for (i, action) in CONTROL_ACTIONS.iter().enumerate() {
            let button = self.bindings.gamepad[i];
            if button != PAD_NONE && buttons[button as usize] {
                held.insert(action.logical);
            }
        }
        if direction != KEY_UNBOUND {
            held.insert(direction);
        }
    }
}

/// 只取主軸，避免斜推同時送出兩個方向；死區以內不算。
///
/// Go：`player_input.go:315`，`func stickDirection(x, y float64, deadzone int) ebiten.Key`
pub(crate) fn stick_direction(x: f64, y: f64, deadzone: i64) -> EbitenKey {
    let limit = normalize_stick_deadzone(deadzone) as f64 / 100.0;
    if x.abs() <= limit && y.abs() <= limit {
        return KEY_UNBOUND;
    }
    if x.abs() >= y.abs() {
        if x < 0.0 {
            return ek::KEY_ARROW_LEFT;
        }
        return ek::KEY_ARROW_RIGHT;
    }
    if y < 0.0 {
        return ek::KEY_ARROW_UP;
    }
    ek::KEY_ARROW_DOWN
}

/// R5 階段 4（r5i）：外殼每個顯示幀讀到的滑鼠右鍵、滾輪、視窗焦點與手把（Go `readInputDevices` 讀 Ebiten 的部分；
/// 滑鼠左鍵與游標在 [`RawInputState`] 本體）。手把只收標準配置，按鈕依 [`PAD_BUTTON_IDS`] 的順序，搖桿照 Ebiten 標準
/// 手把配置的方向（右、下為正）。`wheel_y` 是本顯示幀累積的滾輪格數（上為正），外殼只在第一次 `Update` 給值。
#[derive(Debug, Clone, Default, PartialEq)]
pub struct RawDevices {
    pub focused: bool,
    pub mouse_right: bool,
    pub wheel_y: f64,
    pub pads: Vec<GamepadSnapshot>,
}

/// Rust 新增：Go `readInputDevices`（`player_input.go:334`）讀 Ebiten 的部分裡，需要每次 `Update` 計時的狀態：滑鼠右鍵
/// 與每支手把每個按鈕的按住時間（Go `inpututil.IsMouseButtonJustPressed(Right)`、`StandardGamepadButtonPressDuration`）。
/// 滑鼠左鍵的計時沿用 [`EbitenInputState`]。外殼每個顯示幀給一份 [`RawDevices`]，同一顯示幀的每次 `Update` 都傳同一份
/// （滾輪只在第一次 `Update` 有值，同 Ebiten `readInputState` 讀後歸零）。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct DeviceInputState {
    mouse_right_duration: i64,
    pad_durations: BTreeMap<u64, [i64; PAD_BUTTON_COUNT]>,
}

/// Rust 新增：一次 `Update` 的裝置狀態（[`DeviceInputState::update`] 的結果）。
#[derive(Debug, Clone, Default, PartialEq)]
pub(crate) struct DeviceTick {
    /// Go `readInputDevices()` 的結果。
    pub(crate) snapshot: InputDeviceSnapshot,
    /// 每支手把（依 ID 遞增）每個按鈕的按住時間。
    pub(crate) pad_durations: Vec<(u64, [i64; PAD_BUTTON_COUNT])>,
    /// Go `inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)`。
    pub(crate) right_just: bool,
}

impl DeviceInputState {
    /// 推進一次 `Update`。`physical` 是同一次 `Update` 的實體 [`TickInput`]（滑鼠左鍵的剛按下與游標）。
    pub(crate) fn update(&mut self, raw: &RawDevices, physical: &TickInput) -> DeviceTick {
        self.mouse_right_duration = if raw.mouse_right { self.mouse_right_duration + 1 } else { 0 };
        let mut pads: Vec<&GamepadSnapshot> = raw.pads.iter().collect();
        pads.sort_by_key(|p| p.id);
        let mut durations = BTreeMap::new();
        for pad in &pads {
            let prev = self.pad_durations.get(&pad.id).copied().unwrap_or([0; PAD_BUTTON_COUNT]);
            let mut now = [0; PAD_BUTTON_COUNT];
            for (b, &pressed) in pad.buttons.iter().enumerate() {
                now[b] = if pressed { prev[b] + 1 } else { 0 };
            }
            durations.insert(pad.id, now);
        }
        self.pad_durations = durations;
        let snapshot = InputDeviceSnapshot {
            focused: raw.focused,
            cursor_x: physical.cursor.0,
            cursor_y: physical.cursor.1,
            left: physical.mouse_left_held,
            right: raw.mouse_right,
            left_just: physical.mouse_left_just_pressed,
            right_just: self.mouse_right_duration == 1,
            wheel_y: raw.wheel_y,
            pads: pads.into_iter().cloned().collect(),
        };
        DeviceTick {
            snapshot,
            pad_durations: self.pad_durations.iter().map(|(&id, &d)| (id, d)).collect(),
            right_just: self.mouse_right_duration == 1,
        }
    }
}

/// 指向點落在哪一列（沒有則 -1）。
///
/// Go：`player_input.go:371`，`func pointerIn(rects []image.Rectangle, x, y int) int`
pub(crate) fn pointer_in(rects: &[Rectangle], x: i64, y: i64) -> i64 {
    rects.iter().position(|r| r.min.x <= x && x < r.max.x && r.min.y <= y && y < r.max.y).map_or(-1, |i| i as i64)
}

impl Game {
    /// 在每次 Update 取樣一次（包括設定面板與能力設定開著、遊戲 Update 暫停時）。
    ///
    /// Go：`player_input.go:359`，`func (p *modernHDPresentation) sampleInput(gameInput bool)`。Go 的 `g.input != activeInput`
    /// 檢查（只有正式執行才取樣）在 Rust 由呼叫端 [`PlayerRunner`] 保證。
    pub(crate) fn sample_input(&mut self, game_input: bool, devices: &InputDeviceSnapshot) {
        self.pointer = PointerTick::default();
        let Some(mut input) = self.player_input.take() else {
            return;
        };
        let pointer = {
            let accept = |x: i64, y: i64| self.player_touch_coordinates(x, y);
            input.step(devices, game_input, Some(&accept))
        };
        self.pointer = pointer;
        self.player_input = Some(input);
    }

    /// Go `g.input.suppressVirtual(k)` 之後的 `keyJustPressed` 查詢讀到作廢後的值；Rust 的遊戲讀 `tick_input`，所以作廢後
    /// 由實體的那一份重算。沒有設定輸入對應層（測試、重播）時 Go 的 `g.input` 是 nil，不做事。
    pub(crate) fn suppress_virtual_key(&mut self, k: EbitenKey) {
        let Some(input) = self.player_input.as_mut() else {
            return;
        };
        input.suppress_virtual(k);
        if let Some(physical) = self.tick_input.physical.take() {
            self.tick_input = input.logical_tick(*physical);
        }
    }
}

#[cfg(test)]
#[path = "player_input/player_input_test.rs"]
mod player_input_test;
