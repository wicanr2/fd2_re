//! Go `player_input_test.go`（Go v.1.1.15 `7c710b34`）的 14 項測試，逐條移植。
//!
//! Go 以全域 `activeInput` 讓 `keyJustPressed` 等讀對應層，測試裡沒有實體鍵盤（`inpututil` 全是 0）。Rust 沒有全域，
//! 直接以空的實體輸入（`TickInput::default()`）查 [`PlayerInput`]：`dur`／`just`／`any_keys` 對應 Go 的三個讀法，
//! `repeat` 是 `keyRepeat` 讀邏輯 [`TickInput`]。

use super::*;
use crate::go_ebiten_keys as ek;

fn none() -> TickInput {
    TickInput::default()
}

fn dur(input: &PlayerInput, k: EbitenKey) -> i64 {
    input.duration(&none(), k)
}

fn just(input: &PlayerInput, k: EbitenKey) -> bool {
    dur(input, k) == 1
}

fn any_keys(input: &PlayerInput) -> Vec<EbitenKey> {
    input.append_just_pressed(&none(), Vec::new())
}

fn repeat(input: &PlayerInput, k: Key) -> bool {
    key_repeat(&input.logical_tick(none()), k)
}

fn active(b: ControlBindings) -> PlayerInput {
    PlayerInput::new(b, ControlsLoadStatus::Ok)
}

fn focused() -> InputDeviceSnapshot {
    InputDeviceSnapshot { focused: true, ..Default::default() }
}

fn pad(buttons: &[ControlPadButton]) -> InputDeviceSnapshot {
    let mut snap = focused();
    let mut p = GamepadSnapshot { name: String::from("test pad"), ..Default::default() };
    for &b in buttons {
        p.buttons[b as usize] = true;
    }
    snap.pads = vec![p];
    snap
}

fn accept_below_toolbar(x: i64, y: i64) -> (i64, i64, bool) {
    if y < 24 || !(0..LOGICAL_W).contains(&x) || y >= LOGICAL_H {
        return (0, 0, false);
    }
    (x, y, true)
}

fn idx(id: &str) -> usize {
    control_action_index(id) as usize
}

/// Go：`player_input_test.go:44` `TestControlKeyTableDefaultIsIdentity`
#[test]
fn test_control_key_table_default_is_identity() {
    let table = ControlKeyTable::new(&default_control_bindings());
    for k in 0..=ek::KEY_MAX {
        assert!(table.produce[k as usize] == k && table.physical[k as usize] == vec![k], "default table at {k}");
    }
    // 每個動作預設綁自己的邏輯鍵，鍵與手把各不重複。
    let b = default_control_bindings();
    for (i, action) in CONTROL_ACTIONS.iter().enumerate() {
        assert!(
            b.keyboard[i] == action.logical && b.keyboard_owner(action.logical) == i as i64,
            "{} keyboard",
            action.id
        );
        if action.default_pad != PAD_NONE {
            assert_eq!(b.gamepad_owner(action.default_pad), i as i64, "{} gamepad", action.id);
        }
    }
    let want: BTreeMap<&str, ControlPadButton> = [
        ("up", PAD_UP),
        ("down", PAD_DOWN),
        ("left", PAD_LEFT),
        ("right", PAD_RIGHT),
        ("confirm", PAD_A),
        ("cancel", PAD_B),
        ("space", PAD_X),
        ("controls", PAD_START),
    ]
    .into();
    for (i, action) in CONTROL_ACTIONS.iter().enumerate() {
        let expected = want.get(action.id).copied().unwrap_or(PAD_NONE);
        assert_eq!(b.gamepad[i], expected, "{} default gamepad", action.id);
    }
    for unused in [PAD_Y, PAD_LB, PAD_RB, PAD_LT, PAD_RT, PAD_BACK] {
        assert!(b.gamepad_owner(unused) < 0, "shoulder/select button {unused} bound by default");
    }
}

/// Go：`player_input_test.go:79` `TestControlKeyTableRebindMovesLogicalKey`
#[test]
fn test_control_key_table_rebind_moves_logical_key() {
    let mut b = default_control_bindings();
    let (confirm, next) = (idx("confirm"), idx("next_unit"));
    assert_eq!(b.keyboard_owner(ek::KEY_Z), next as i64, "Z owner");
    b.set_keyboard(confirm, ek::KEY_Z);
    assert!(b.keyboard[confirm] == ek::KEY_Z && b.keyboard[next] == KEY_UNBOUND, "conflict replacement");
    let table = ControlKeyTable::new(&b);
    assert!(
        table.produce[ek::KEY_Z as usize] == ek::KEY_ENTER && table.produce[ek::KEY_ENTER as usize] == KEY_UNBOUND,
        "Z should produce Enter and physical Enter nothing"
    );
    assert!(table.physical[ek::KEY_Z as usize].is_empty() && table.physical[ek::KEY_ENTER as usize] == vec![ek::KEY_Z]);
    // 沒有指定給任何動作的鍵（WASD、Numpad5）維持原版操作。
    for k in [ek::KEY_W, ek::KEY_A, ek::KEY_NUMPAD5, ek::KEY_SHIFT] {
        assert_eq!(table.produce[k as usize], k, "unassigned key changed {k}");
    }
    b.set_keyboard(idx("up"), ek::KEY_W);
    assert_eq!(ControlKeyTable::new(&b).produce[ek::KEY_W as usize], ek::KEY_ARROW_UP, "W assigned to up");
}

/// Go：`player_input_test.go:108` `TestControlBindingsClearResetAndReservedKeys`
#[test]
fn test_control_bindings_clear_reset_and_reserved_keys() {
    let mut b = default_control_bindings();
    let (space, confirm) = (idx("space"), idx("confirm"));
    b.clear_action(space);
    assert!(b.keyboard[space] == KEY_UNBOUND && b.gamepad[space] == PAD_NONE, "clear");
    assert!(b.reset_conflict(space) < 0, "reset of a cleared row has no conflict");
    b.reset_action(space);
    assert_eq!(b, default_control_bindings(), "reset row did not restore defaults");
    b.set_gamepad(space, PAD_A);
    assert!(b.gamepad[confirm] == PAD_NONE && b.gamepad[space] == PAD_A, "gamepad conflict replacement");
    assert_eq!(
        b.reset_conflict(confirm),
        space as i64,
        "reset conflict should name the row holding the default button"
    );
    b.reset_action(confirm);
    assert!(b.gamepad[confirm] == PAD_A && b.gamepad[space] == PAD_NONE, "reset takes the default back");
    for k in [
        ek::KEY_SHIFT,
        ek::KEY_CONTROL_LEFT,
        ek::KEY_ALT_RIGHT,
        ek::KEY_META,
        ek::KEY_ESCAPE,
        KEY_UNBOUND,
        ek::KEY_MAX + 1,
    ] {
        assert!(!bindable_key(k), "reserved key bindable {k}");
    }
    for k in [ek::KEY_ENTER, ek::KEY_Z, ek::KEY_F12, ek::KEY_W, ek::KEY_DIGIT1] {
        assert!(bindable_key(k), "ordinary key not bindable {k}");
    }
}

/// Go：`player_input_test.go:145` `TestControlsSettingsRoundTripCorruptAndPartial`
#[test]
fn test_controls_settings_round_trip_corrupt_and_partial() {
    let dir = crate::test_support::temp_dir("r5i-controls-settings");
    let path = dir.join("controls-settings.json").to_string_lossy().into_owned();
    assert_eq!(load_controls_settings_from(&path), (default_control_bindings(), ControlsLoadStatus::Default));
    let mut b = default_control_bindings();
    b.set_keyboard(idx("confirm"), ek::KEY_Z);
    b.set_gamepad(idx("space"), PAD_Y);
    b.gamepad[idx("controls")] = PAD_NONE;
    b.deadzone = 35;
    save_controls_settings_to(&path, &b).unwrap();
    let raw = std::fs::read(&path).unwrap();
    let file: ControlsSettingsFile = serde_json::from_slice(&raw).unwrap();
    let want_keyboard: BTreeMap<String, String> =
        [("confirm", "Z"), ("next_unit", "")].map(|(k, v)| (k.to_string(), v.to_string())).into();
    let want_pad: BTreeMap<String, String> =
        [("space", "y"), ("controls", "")].map(|(k, v)| (k.to_string(), v.to_string())).into();
    assert!(
        file.keyboard == Some(want_keyboard)
            && file.gamepad == Some(want_pad)
            && file.stick_deadzone == 35
            && file.schema_version == 1,
        "file stores only overrides: {}",
        String::from_utf8_lossy(&raw)
    );
    // Rust 新增：位元組與 Go `json.MarshalIndent(..., "", " ")` 加換行相同（兩版設定檔互通）。
    let go = "{\n \"schema_version\": 1,\n \"keyboard\": {\n  \"confirm\": \"Z\",\n  \"next_unit\": \"\"\n },\n \"gamepad\": {\n  \"controls\": \"\",\n  \"space\": \"y\"\n },\n \"stick_deadzone_percent\": 35\n}\n";
    assert_eq!(String::from_utf8(raw).unwrap(), go);
    let (loaded, status) = load_controls_settings_from(&path);
    assert!(status == ControlsLoadStatus::Ok && loaded == b, "round trip status={status:?}");
    assert_eq!(std::fs::read_dir(&dir).unwrap().count(), 1, "atomic save left temporary files");
    for corrupt in ["{", r#"{"schema_version":2}"#, "[]", ""] {
        std::fs::write(&path, corrupt).unwrap();
        assert_eq!(
            load_controls_settings_from(&path),
            (default_control_bindings(), ControlsLoadStatus::Corrupt),
            "corrupt {corrupt:?}"
        );
    }
    std::fs::write(&path, r#"{"schema_version":1,"keyboard":{"confirm":"NoSuchKey","cancel":"Shift","ghost":"Z","space":"Q"},"gamepad":{"confirm":"turbo"},"stick_deadzone_percent":99}"#).unwrap();
    let (loaded, status) = load_controls_settings_from(&path);
    let mut want = default_control_bindings();
    want.set_keyboard(idx("space"), ek::KEY_Q);
    assert!(status == ControlsLoadStatus::Partial && loaded == want, "partial file: status={status:?} got {loaded:?}");
    let _ = std::fs::remove_dir_all(&dir);
}

/// Go：`player_input_test.go:195` `TestPlayerInputGamepadButtonsRepeatAndDeadzone`
#[test]
fn test_player_input_gamepad_buttons_repeat_and_deadzone() {
    let mut input = active(default_control_bindings());
    input.step(&pad(&[PAD_A]), true, None);
    assert_eq!(dur(&input, ek::KEY_ENTER), 0, "a pad connected with a button held is latched until released");
    input.step(&pad(&[]), true, None);
    input.step(&pad(&[PAD_A]), true, None);
    assert!(just(&input, ek::KEY_ENTER) && dur(&input, ek::KEY_ENTER) == 1, "A should press Enter");
    assert_eq!(any_keys(&input), vec![ek::KEY_ENTER], "any-key list");
    input.step(&pad(&[PAD_A]), true, None);
    assert!(!just(&input, ek::KEY_ENTER) && dur(&input, ek::KEY_ENTER) == 2, "held A is one press");
    input.step(&pad(&[]), true, None);
    assert_eq!(dur(&input, ek::KEY_ENTER), 0, "release");
    for (button, key) in
        [(PAD_B, ek::KEY_ESCAPE), (PAD_X, ek::KEY_SPACE), (PAD_START, ek::KEY_F12), (PAD_UP, ek::KEY_ARROW_UP)]
    {
        input.step(&pad(&[button]), true, None);
        assert!(just(&input, key), "default gamepad mapping {button} {key}");
        input.step(&pad(&[]), true, None);
    }
    for unused in [PAD_Y, PAD_LB, PAD_RB, PAD_BACK] {
        input.step(&pad(&[unused]), true, None);
        assert!(any_keys(&input).is_empty(), "unbound button produced a key {unused}");
        input.step(&pad(&[]), true, None);
    }
    // 十字鍵按住與鍵盤方向鍵同節奏：第 1 格、第 12 格、之後每 5 格。
    let mut repeats = Vec::new();
    for tick in 1..=22 {
        input.step(&pad(&[PAD_DOWN]), true, None);
        if repeat(&input, Key::ArrowDown) {
            repeats.push(tick);
        }
    }
    assert_eq!(repeats, vec![1, 12, 17, 22], "repeat cadence");
    input.step(&pad(&[]), true, None);
    let stick = |x: f64, y: f64| {
        let mut snap = focused();
        snap.pads = vec![GamepadSnapshot { stick_x: x, stick_y: y, ..Default::default() }];
        snap
    };
    input.step(&stick(0.15, -0.19), true, None);
    assert!(any_keys(&input).is_empty(), "inside 20% dead zone");
    input.step(&stick(0.6, 0.3), true, None);
    assert!(just(&input, ek::KEY_ARROW_RIGHT) && dur(&input, ek::KEY_ARROW_DOWN) == 0, "dominant axis right only");
    input.step(&stick(0.0, 0.0), true, None);
    input.step(&stick(-0.2, -0.7), true, None);
    assert!(just(&input, ek::KEY_ARROW_UP), "stick up");
    let mut b = default_control_bindings();
    b.deadzone = 50;
    input.set_bindings(b);
    input.step(&stick(0.0, 0.0), true, None);
    input.step(&stick(0.45, 0.0), true, None);
    assert!(any_keys(&input).is_empty(), "dead zone setting ignored");
}

/// Go：`player_input_test.go:272` `TestPlayerInputRebindingAppliesToGamepad`
#[test]
fn test_player_input_rebinding_applies_to_gamepad() {
    let mut b = default_control_bindings();
    b.set_gamepad(idx("space"), PAD_A); // A 改成空白鍵，確認改成未設定
    let mut input = active(b);
    input.step(&pad(&[]), true, None);
    input.step(&pad(&[PAD_A]), true, None);
    assert!(just(&input, ek::KEY_SPACE) && !just(&input, ek::KEY_ENTER), "rebound A");
}

/// Go：`player_input_test.go:283` `TestPlayerInputFocusLossAndDisconnectReleaseInput`
#[test]
fn test_player_input_focus_loss_and_disconnect_release_input() {
    let mut input = active(default_control_bindings());
    input.step(&pad(&[]), true, None);
    input.step(&pad(&[PAD_A]), true, None);
    assert!(just(&input, ek::KEY_ENTER), "press");
    let mut lost = pad(&[PAD_A]);
    lost.focused = false;
    input.step(&lost, true, None);
    assert_eq!(dur(&input, ek::KEY_ENTER), 0, "focus loss should release");
    input.step(&pad(&[PAD_A]), true, None);
    assert_eq!(dur(&input, ek::KEY_ENTER), 0, "button held across focus loss must be released first");
    let mut held = focused();
    held.pads = vec![GamepadSnapshot { stick_x: 0.9, ..Default::default() }];
    input.step(&held, true, None);
    assert_eq!(dur(&input, ek::KEY_ARROW_RIGHT), 0, "stick must return to centre before reading");
    input.step(&pad(&[]), true, None);
    input.step(&pad(&[PAD_A]), true, None);
    assert!(just(&input, ek::KEY_ENTER), "after release the pad works again");
    input.step(&focused(), true, None); // 拔掉手把
    assert_eq!(dur(&input, ek::KEY_ENTER), 0, "disconnect releases");
    input.step(&pad(&[PAD_A]), true, None);
    assert_eq!(dur(&input, ek::KEY_ENTER), 0, "reconnected pad is latched until released");
}

/// Go：`player_input_test.go:321` `TestPlayerInputMouseButtonsWheelAndPointer`
#[test]
fn test_player_input_mouse_buttons_wheel_and_pointer() {
    let mut input = active(default_control_bindings());
    let mouse =
        |x: i64, y: i64, left: bool, left_just: bool, right: bool, right_just: bool, wheel: f64| InputDeviceSnapshot {
            focused: true,
            cursor_x: x,
            cursor_y: y,
            left,
            left_just,
            right,
            right_just,
            wheel_y: wheel,
            pads: Vec::new(),
        };
    let acc: &dyn Fn(i64, i64) -> (i64, i64, bool) = &accept_below_toolbar;
    let p = input.step(&mouse(100, 100, false, false, false, false, 0.0), true, Some(acc));
    assert!(!p.active, "first sample has no movement");
    let p = input.step(&mouse(100, 100, false, false, false, false, 0.0), true, Some(acc));
    assert!(!p.active, "static mouse must not point");
    let p = input.step(&mouse(130, 90, false, false, false, false, 0.0), true, Some(acc));
    assert!(p.active && !p.clicked && p.x == 130 && p.y == 90, "moved mouse points {p:?}");
    let p = input.step(&mouse(130, 90, true, true, false, false, 0.0), true, Some(acc));
    assert!(p.active && p.clicked && just(&input, ek::KEY_ENTER), "left click = Enter {p:?}");
    input.step(&mouse(130, 90, true, false, false, false, 0.0), true, Some(acc));
    assert!(!just(&input, ek::KEY_ENTER) && dur(&input, ek::KEY_ENTER) == 2, "held left is one press");
    input.step(&mouse(130, 90, false, false, false, false, 0.0), true, Some(acc));
    input.step(&mouse(130, 10, true, true, false, false, 0.0), true, Some(acc));
    assert_eq!(dur(&input, ek::KEY_ENTER), 0, "click on the toolbar must not reach the game");
    input.step(&mouse(130, 90, true, false, false, false, 0.0), true, Some(acc));
    assert_eq!(dur(&input, ek::KEY_ENTER), 0, "a press that started outside stays outside");
    input.step(&mouse(130, 90, false, false, true, true, 0.0), true, Some(acc));
    assert!(just(&input, ek::KEY_ESCAPE), "right click = Esc");
    input.step(&mouse(130, 90, false, false, false, false, 1.0), true, Some(acc));
    assert!(just(&input, ek::KEY_ARROW_UP) && repeat(&input, Key::ArrowUp), "wheel up = one Up");
    input.step(&mouse(130, 90, false, false, false, false, 0.0), true, Some(acc));
    assert_eq!(dur(&input, ek::KEY_ARROW_UP), 0, "wheel pulse lasts one tick");
    input.step(&mouse(130, 90, false, false, false, false, -0.5), true, Some(acc));
    assert_eq!(dur(&input, ek::KEY_ARROW_DOWN), 0, "half notch is not a step");
    input.step(&mouse(130, 90, false, false, false, false, -0.5), true, Some(acc));
    assert!(just(&input, ek::KEY_ARROW_DOWN), "two half notches make one Down");
    input.step(&mouse(130, 90, false, false, false, false, -1.0), true, Some(acc));
    input.step(&mouse(130, 90, false, false, false, false, -1.0), true, Some(acc));
    assert!(just(&input, ek::KEY_ARROW_DOWN), "consecutive wheel notches are separate presses");
    // 設定面板開著（gameInput=false）：點擊不進遊戲。
    let p = input.step(&mouse(200, 200, true, true, false, false, 0.0), false, Some(acc));
    assert!(!p.active && dur(&input, ek::KEY_ENTER) == 0, "panel owns the mouse");
}

/// Go：`player_input_test.go:383` `TestPlayerInputSuppressedClick`
#[test]
fn test_player_input_suppressed_click() {
    let mut input = active(default_control_bindings());
    let acc: &dyn Fn(i64, i64) -> (i64, i64, bool) = &accept_below_toolbar;
    input.step(
        &InputDeviceSnapshot { focused: true, cursor_x: 50, cursor_y: 50, ..Default::default() },
        true,
        Some(acc),
    );
    input.step(
        &InputDeviceSnapshot {
            focused: true,
            cursor_x: 50,
            cursor_y: 50,
            left: true,
            left_just: true,
            ..Default::default()
        },
        true,
        Some(acc),
    );
    input.suppress_virtual(ek::KEY_ENTER);
    assert!(!just(&input, ek::KEY_ENTER) && any_keys(&input).is_empty(), "suppressed click still confirms");
}

/// Go：`player_input_test.go:393` `TestNilActiveInputReadsKeyboardDirectly`。Rust 沒有全域：沒有對應層時 `tick_input`
/// 就是實體輸入（`physical()` 是自己），測試裡沒有鍵盤。
#[test]
fn test_nil_active_input_reads_keyboard_directly() {
    let g = Game::default();
    assert!(g.player_input.is_none() && g.tick_input.physical.is_none(), "tests must not leave the input layer set");
    let input = &g.tick_input;
    assert!(
        !input.is_just_pressed(Key::Enter) && input.key_press_duration(Key::Enter) == 0 && !input.any_just_pressed()
    );
    assert!(std::ptr::eq(input.physical(), input));
}

fn native_pointer_game(camera_x: i64, camera_y: i64) -> Game {
    let mut st = battle::State::default();
    (st.w, st.h, st.has_native_map_view_state) = (40, 30, true);
    st.native_map_view_state = battle::NativeMapViewState {
        camera_x,
        camera_y,
        cursor_x: camera_x + 6,
        cursor_y: camera_y + 4,
        visible_cursor_x: 6,
        visible_cursor_y: 4,
    };
    Game {
        st: Some(st),
        m: Some(MapData { w: 40, h: 30, tile_w: 48, tile_h: 48, ..Default::default() }),
        ..Default::default()
    }
}

/// Go：`player_input_test.go:408` `TestPointerMapCellUsesNativeViewportGeometry`
#[test]
fn test_pointer_map_cell_uses_native_viewport_geometry() {
    let mut g = native_pointer_game(3, 5);
    g.pointer = PointerTick { active: true, clicked: false, x: 8 + 48 * 2 + 5, y: 8 + 48 * 3 + 47 };
    assert_eq!(g.pointer_map_cell(), (5, 8, true), "cell");
    for (x, y) in [(7, 100), (8 + 48 * 13, 100), (100, 8 + 48 * 8)] {
        g.pointer = PointerTick { active: true, clicked: false, x, y };
        assert!(!g.pointer_map_cell().2, "outside the native viewport ({x},{y})");
    }
    g.pointer = PointerTick { x: 100, y: 100, ..Default::default() };
    assert!(!g.pointer_map_cell().2, "inactive pointer");
    let mut legacy = Game {
        m: Some(MapData { w: 20, h: 20, tile_w: 48, tile_h: 48, ..Default::default() }),
        cam_x: 96.0,
        cam_y: 48.0,
        cur_x: 2,
        cur_y: 1,
        pointer: PointerTick { active: true, clicked: false, x: 150, y: 60 },
        ..Default::default()
    };
    assert_eq!(legacy.pointer_map_cell(), (5, 2, true), "legacy touch conversion");
    legacy.follow_pointer_on_map();
    assert_eq!((legacy.cur_x, legacy.cur_y), (5, 2), "cursor follows pointer");
}

/// Go：`player_input_test.go:435` `TestPointerOverlayDirectionAndGate`
#[test]
fn test_pointer_overlay_direction_and_gate() {
    let mut g = native_pointer_game(0, 0);
    let (cx, cy) = (8 + 6 * 48 + 24, 8 + 4 * 48 + 24);
    for (dx, dy, want, ok) in [
        (0, -48, 0, true),
        (-48, 5, 1, true),
        (48, -5, 2, true),
        (3, 48, 3, true),
        (0, 0, 0, false),
        (10, 10, 0, false),
        (200, 0, 0, false),
    ] {
        g.pointer = PointerTick { active: true, clicked: false, x: cx + dx, y: cy + dy };
        let (direction, got) = g.pointer_overlay_direction();
        assert!(got == ok && (!got || direction == want), "overlay direction ({dx},{dy}) {direction} {got}");
    }
    g.ring_sel = 0;
    g.pointer = PointerTick { active: true, clicked: false, x: cx + 48, y: cy };
    g.apply_pointer_to_overlay([0, 0, 1, 0]); // 右邊不可用：原版閘門不改選擇
    assert_eq!(g.ring_sel, 0, "unavailable direction selected");
    g.apply_pointer_to_overlay([0, 0, 0, 0]);
    assert_eq!(g.ring_sel, 2, "available direction");
}

/// Go：`player_input_test.go:459` `TestPointerSelectsTitleMenuSlotsAndTown`。讀檔四格的換列走 `applyTitleSlotEvent`
/// （`r8`），這裡只驗指向的列；選單三項、城鎮三種配置與選項框照 Go。
#[test]
fn test_pointer_selects_title_menu_slots_and_town() {
    let mut assets = TitleAssets::default();
    for item in &mut assets.items {
        item[0] = Some(Arc::new(Picture::new_image(40, 8)));
    }
    let mut g = Game { title_phase: String::from("menu"), title_assets: Some(assets), ..Default::default() };
    let rects = g.title_menu_rects();
    let at = (rects[2].min.x + 3, rects[2].min.y + 3);
    g.pointer = PointerTick { active: true, clicked: false, x: at.0, y: at.1 };
    g.apply_pointer_to_title_menu();
    assert_eq!(g.title_sel, 2, "title menu");
    g.title_flash = 5;
    let at = (rects[0].min.x + 3, rects[0].min.y + 3);
    g.pointer = PointerTick { active: true, clicked: false, x: at.0, y: at.1 };
    g.apply_pointer_to_title_menu();
    assert_eq!(g.title_sel, 2, "confirmation flash must not move the selection");
    let slots = title_slot_rects();
    assert_eq!(pointer_in(&slots, slots[3].min.x + 5, slots[3].min.y + 5), 3, "load slot");
    let mut g = Game { title_phase: String::from("loadslots"), title_slot_sel: 3, ..Default::default() };
    g.pointer = PointerTick { active: true, clicked: false, x: slots[3].min.x + 5, y: slots[3].min.y + 5 };
    g.apply_pointer_to_title_slots(); // 已在指向的列：不換列，不經 r8 的 applyTitleSlotEvent
    assert_eq!(g.title_slot_sel, 3);
    for variant in 0..3 {
        let rects = native_town_selection_rects(variant);
        assert_eq!(rects.len(), 5, "town rects");
        for target in 0..5 {
            let mut g = Game { camp_sel: 0, ..Default::default() };
            let r = rects[target as usize];
            let at = (r.min.x + (r.max.x - r.min.x) / 2, r.min.y + (r.max.y - r.min.y) / 2);
            g.pointer = PointerTick { active: true, clicked: false, x: at.0, y: at.1 };
            g.apply_pointer_to_native_town(variant);
            assert_eq!(g.camp_sel, target, "town {variant} {target}");
        }
    }
    assert!(
        pointer_in(&choice_menu_rects(3), 300, 162 + 2 * 28 + 5) == 2
            && pointer_in(&choice_menu_rects(3), 100, 170) == -1
    );
}

/// Go：`player_input_test.go:505` `TestF12OpensControlsPanelAndKeepsDebugHUD`
#[test]
fn test_f12_opens_controls_panel_and_keeps_debug_hud() {
    let mut input = active(default_control_bindings());
    input.step(&pad(&[]), true, None);
    input.step(&pad(&[PAD_START]), true, None);
    let mut g = Game { tick_input: input.logical_tick(none()), player_input: Some(input), ..Default::default() };
    g.update().unwrap();
    assert!(g.controls.open && !g.debug, "Start → F12 opens the panel, not the debug HUD");
}

/// Rust 新增：預設綁定、沒有滑鼠與手把時，邏輯輸入與實體輸入相同（只多了 `physical`）。第一章重播與外殼驅動不經
/// 這一層也不受影響；這一項保證正式執行（[`PlayerRunner`]）的鍵盤行為與加入這一層之前相同。
#[test]
fn logical_tick_is_identity_without_devices() {
    let mut input = active(default_control_bindings());
    let mut state = EbitenInputState::default();
    let mut rng = battle::GoMathRand::new(5);
    let keys = Key::ALL;
    for tick in 0..2000 {
        let mut raw = RawInputState::default();
        for &k in &keys {
            if rng.intn(5) == 0 {
                raw.keys.insert(k);
            }
        }
        for code in [ek::KEY_Q as u32, ek::KEY_PAGE_UP as u32, 0x8000_0001] {
            if rng.intn(4) == 0 {
                raw.other_keys.insert(code);
            }
        }
        let physical = state.update(&raw);
        input.step(&InputDeviceSnapshot::default(), true, None);
        let mut logical = input.logical_tick(physical.clone());
        assert_eq!(logical.physical.take().map(|p| *p), Some(physical.clone()), "tick {tick}");
        assert_eq!(logical, TickInput { physical: None, ..physical }, "tick {tick}");
    }
}

/// Rust 新增：改綁後，實體鍵經對應表換成邏輯鍵；Alt+Enter 等讀實體鍵的地方不受改綁影響。
#[test]
fn logical_tick_applies_keyboard_rebinding() {
    let mut b = default_control_bindings();
    b.set_keyboard(idx("confirm"), ek::KEY_Q);
    let input = active(b);
    let mut state = EbitenInputState::default();
    let physical = state.update(&RawInputState { other_keys: [ek::KEY_Q as u32].into(), ..Default::default() });
    let logical = input.logical_tick(physical);
    assert!(logical.is_just_pressed(Key::Enter) && logical.any_just_pressed() && !logical.other_just_pressed);
    assert!(!logical.physical().is_just_pressed(Key::Enter) && logical.physical().other_just_pressed);
    let physical = state.update(&RawInputState { keys: [Key::Enter].into(), ..Default::default() });
    let logical = input.logical_tick(physical);
    assert!(!logical.is_pressed(Key::Enter) && !logical.any_just_pressed(), "physical Enter no longer produces Enter");
    assert!(logical.physical().is_just_pressed(Key::Enter), "Alt+Enter still sees the physical key");
}
