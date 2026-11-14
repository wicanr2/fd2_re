//! Go 來源：`remake/cmd/fd2/native_town_secret_input_test.go`（1 個測試）；另加 Rust 補充：兩個輸入函式改讀
//! `TickInput` 後的行為（Go 讀全域鍵盤，沒有對應測試）。

use super::*;

fn input(held: &[Key], pressed: &[Key]) -> TickInput {
    TickInput { held: held.iter().copied().collect(), pressed: pressed.iter().copied().collect(), ..Default::default() }
}

/// Go：`native_town_secret_input_test.go:5` `TestNativeBIOSFunctionScanRanges`
#[test]
fn test_native_bios_function_scan_ranges() {
    for (modifier, function, want) in [
        (NATIVE_FUNCTION_SHIFT, 1, 0x54),
        (NATIVE_FUNCTION_SHIFT, 10, 0x5d),
        (NATIVE_FUNCTION_CONTROL, 1, 0x5e),
        (NATIVE_FUNCTION_CONTROL, 10, 0x67),
        (NATIVE_FUNCTION_ALT, 1, 0x68),
        (NATIVE_FUNCTION_ALT, 10, 0x71),
    ] {
        let (got, ok) = native_bios_function_scan(modifier, function);
        assert!(ok && got == want, "modifier={modifier} function={function}: got {got:#x},{ok} want {want:#x},true");
    }
    for (modifier, function) in [(-1, 1), (3, 1), (0, 0), (0, 11)] {
        assert!(
            !native_bios_function_scan(modifier, function).1,
            "invalid modifier/function ({modifier},{function}) accepted"
        );
    }
}

/// Rust 補充：修飾鍵只看「按住」，與 Go `ebiten.IsKeyPressed` 相同。
#[test]
fn native_modifier_held_reads_held_keys() {
    assert!(!native_modifier_held(&TickInput::default()));
    assert!(!native_modifier_held(&input(&[], &[Key::Shift])), "本幀按下但未列為按住不算");
    for key in [Key::Shift, Key::Control, Key::Alt] {
        assert!(native_modifier_held(&input(&[key], &[])), "{key:?}");
    }
}

/// Rust 補充：恰好一個修飾鍵按住、F1..F10 本幀按下才回掃描碼；兩個修飾鍵同時按住拒收。
#[test]
fn pressed_native_town_secret_scan_reads_tick_input() {
    assert_eq!(pressed_native_town_secret_scan(&input(&[Key::Alt], &[Key::F10])), (0x71, true));
    assert_eq!(pressed_native_town_secret_scan(&input(&[Key::Shift], &[Key::F1])), (0x54, true));
    assert_eq!(pressed_native_town_secret_scan(&input(&[Key::Shift, Key::Alt], &[Key::F1])), (0, false));
    assert_eq!(pressed_native_town_secret_scan(&input(&[], &[Key::F1])), (0, false));
    assert_eq!(pressed_native_town_secret_scan(&input(&[Key::Control, Key::F2], &[])), (0, false), "F 鍵只按住不算");
}
