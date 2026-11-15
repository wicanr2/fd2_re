//! [`EbitenInputState`]（Ebiten v2.6.6 `inpututil` 的每幀狀態，R5 階段 4 r5w）。

use super::*;

fn raw(keys: &[Key]) -> RawInputState {
    RawInputState { keys: keys.iter().copied().collect(), ..Default::default() }
}

/// 按住三幀再放開：只有第一幀「剛按下」、持續時間逐幀加 1、放開那一幀「剛放開」，之後全空
/// （`inpututil.go` 的 `keyDurations` 與 `prevKeyDurations`）。
#[test]
fn key_press_duration_and_edges_follow_inpututil() {
    let mut s = EbitenInputState::default();
    for tick in 1..=3 {
        let input = s.update(&raw(&[Key::Enter]));
        assert_eq!(input.is_just_pressed(Key::Enter), tick == 1);
        assert!(input.is_pressed(Key::Enter) && !input.is_just_released(Key::Enter));
        assert_eq!(input.key_press_duration(Key::Enter), tick);
    }
    let input = s.update(&raw(&[]));
    assert!(input.is_just_released(Key::Enter) && !input.is_pressed(Key::Enter));
    assert_eq!(input.key_press_duration(Key::Enter), 0);
    let input = s.update(&raw(&[]));
    assert_eq!(input, TickInput::default());
}

/// 虛擬鍵 `KeyAlt`：左右任一個按住就是按住（`input.go:418`）；呼叫端給的 `Alt` 本身不算。
#[test]
fn alt_is_derived_from_either_side() {
    let mut s = EbitenInputState::default();
    let input = s.update(&raw(&[Key::AltRight]));
    assert!(input.is_pressed(Key::Alt) && input.is_just_pressed(Key::Alt) && input.is_pressed(Key::AltRight));
    let input = s.update(&raw(&[Key::AltLeft]));
    assert!(input.is_pressed(Key::Alt) && !input.is_just_pressed(Key::Alt), "換手時 KeyAlt 持續按住");
    assert!(input.is_just_pressed(Key::AltLeft) && input.is_just_released(Key::AltRight));
    let input = s.update(&raw(&[Key::Alt]));
    assert!(!input.is_pressed(Key::Alt) && input.is_just_released(Key::Alt));
}

/// `AppendJustPressedKeys` 含 [`Key`] 沒有列出的鍵；按住不放的下一幀不再算。
#[test]
fn other_keys_count_as_any_key_only_on_first_tick() {
    let mut s = EbitenInputState::default();
    let held = RawInputState { other_keys: [7].into(), ..Default::default() };
    let input = s.update(&held);
    assert!(input.any_just_pressed() && input.pressed.is_empty());
    assert!(!s.update(&held).any_just_pressed());
    assert!(!s.update(&RawInputState::default()).any_just_pressed());
    assert!(s.update(&held).any_just_pressed(), "放開後再按又算一次");
}

/// 滑鼠左鍵的 `IsMouseButtonJustPressed`／`JustReleased`／`IsMouseButtonPressed` 與游標座標原樣傳出。
#[test]
fn mouse_left_edges_follow_inpututil() {
    let mut s = EbitenInputState::default();
    let down = RawInputState { mouse_left: true, cursor: (5, -3), ..Default::default() };
    let input = s.update(&down);
    assert!(input.mouse_left_held && input.mouse_left_just_pressed && !input.mouse_left_just_released);
    assert_eq!(input.cursor, (5, -3));
    let input = s.update(&down);
    assert!(input.mouse_left_held && !input.mouse_left_just_pressed);
    let input = s.update(&RawInputState::default());
    assert!(!input.mouse_left_held && input.mouse_left_just_released);
    assert!(!s.update(&RawInputState::default()).mouse_left_just_released);
    assert!(!input.any_just_pressed(), "滑鼠不算 AppendJustPressedKeys");
}

/// `AppendJustPressedTouchIDs`：持續時間 1 的 ID 依遞增排序，位置是同一幀的 `TouchPosition`。
#[test]
fn just_pressed_touches_sorted_by_id() {
    let mut s = EbitenInputState::default();
    let input = s.update(&RawInputState { touches: vec![(9, 1, 2)], ..Default::default() });
    assert_eq!(input.just_pressed_touches, vec![(1, 2)]);
    let input = s.update(&RawInputState { touches: vec![(9, 5, 6), (3, 7, 8), (4, -2, 900)], ..Default::default() });
    assert_eq!(input.just_pressed_touches, vec![(7, 8), (-2, 900)], "ID 9 已按住第二幀");
    let input = s.update(&RawInputState::default());
    assert!(input.just_pressed_touches.is_empty());
    let input = s.update(&RawInputState { touches: vec![(9, 0, 0)], ..Default::default() });
    assert_eq!(input.just_pressed_touches, vec![(0, 0)], "放開後同一個 ID 重新開始");
}
