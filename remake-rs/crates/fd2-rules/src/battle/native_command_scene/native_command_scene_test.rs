//! 移植自 `remake/internal/battle/native_command_scene_test.go`。

use super::*;
use crate::test_support::remake_asset;

/// Go：`TestLoadNativeCommandSceneTableMatches52363`
#[test]
fn load_native_command_scene_table_matches52363() {
    let Some(path) = remake_asset("data/native_command_scene.json") else { return };
    let table = load_native_command_scene_table(&path).unwrap();
    for chapter in 0..NATIVE_COMMAND_SCENE_CHAPTER_COUNT {
        let got = table.initial_background(chapter).unwrap();
        let want = if [24, 25, 26, 28, 29].contains(&chapter) { 0 } else { 3 };
        assert_eq!(got, want, "chapter {chapter} selector");
    }
    assert!(table.initial_background(30).is_err(), "chapter beyond fixed table accepted");
}

/// Go：`TestNativeCommandBackgroundGateMatches1F183`
#[test]
fn native_command_background_gate_matches1_f183() {
    let mut unit = Unit {
        battle_fig: 1,
        has_battle_fig: true,
        native_record_race: 4,
        has_native_record_race: true,
        native_record_class: 0,
        has_native_record_class: true,
        ..Default::default()
    };
    let got = native_command_background_gate(Some(&unit));
    assert!(matches!(got, Ok(true)), "race4 gate={got:?}");
    (unit.native_record_race, unit.native_record_class) = (0, 19);
    let got = native_command_background_gate(Some(&unit));
    assert!(matches!(got, Ok(true)), "class19 gate={got:?}");
    unit.battle_fig = 28;
    let got = native_command_background_gate(Some(&unit));
    assert!(matches!(got, Ok(false)), "selector28 gate={got:?}");
    unit.has_native_record_race = false;
    assert!(native_command_background_gate(Some(&unit)).is_err(), "missing raw race provenance accepted");
}
