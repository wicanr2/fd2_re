//! Go 來源：`remake/cmd/fd2/native_event_dialogue_input_test.go`（R4 B 組，期望值不改）。

use super::*;

/// Go：`native_event_dialogue_input_test.go:5` `TestNativeBattleDialogueAdvanceInput`
#[test]
fn test_native_battle_dialogue_advance_input() {
    let tests = [
        ("none", false, false, false),
        ("enter_or_space", true, false, true),
        ("escape", false, true, true),
        ("both", true, true, true),
    ];
    for (name, enter_or_space, escape, want) in tests {
        let got = native_battle_dialogue_advance_input(enter_or_space, escape);
        assert_eq!(got, want, "{name}: nativeBattleDialogueAdvanceInput({enter_or_space}, {escape})");
    }
}
