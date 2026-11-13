//! 移植自 `remake/internal/battle/native_command_test.go`（PORTING.md「延後的測試」）。
//!
//! 兩個 `GainExp` 測試要等 E 組 `growth.rs`（以及它呼叫的 B 組函式）完成，先標 `#[ignore]`；其餘
//! 只用到 A 組的測試現在就跑。

use super::*;
use crate::test_support::{docs_file, temp_dir};

/// Go：`TestNativeCommandMaskConstructorAndDynamicFifthByte`
#[test]
fn native_command_mask_constructor_and_dynamic_fifth_byte() {
    let mut u = Unit::default();
    u.set_initial_command_mask(&[0x81, 0x01, 0x00, 0x80]).unwrap();
    assert_eq!(u.native_command_ids(), [0, 7, 8, 31], "initial IDs");
    assert!(
        u.enable_native_command(32) && u.enable_native_command(39) && !u.enable_native_command(40),
        "fifth-byte command bounds failed: {:?}",
        u.native_command_mask
    );
    assert_eq!(u.native_command_ids(), [0, 7, 8, 31, 32, 39], "expanded IDs");
}

/// Go：`TestStateGainExpLearnsExactNativeCommandAtLevel`
#[test]
fn state_gain_exp_learns_exact_native_command_at_level() {
    let u = Unit {
        camp: OWN,
        name: "索爾".into(),
        cls_name: "劍聖".into(),
        portrait: 32,
        battle_fig: 32,
        has_battle_fig: true,
        lv: 3,
        ap: 6,
        dp: 4,
        dx: 2,
        hp: 42,
        max_hp: 42,
        ..Default::default()
    };
    let mut st = State {
        command_learn: Some(BTreeMap::from([(4, vec![CommandLearnEntry { required_level: 4, command_id: 24 }])])),
        command_learn_selectors: Some(BTreeMap::from([(32, 4)])),
        ..Default::default()
    };
    // Go 傳入不在 `State.Units` 的單位；Rust 先 push 進 `units`（battle 模組說明第 6 點）。
    st.units.push(u);
    let events = st.gain_exp(Some(0), 100.0, &mut GoMathRand::new(1));
    assert!(events.len() == 1 && events[0].learned_command_ids == [24], "level-up learning events={events:?}");
    assert_eq!(st.units[0].native_command_ids(), [24], "native command mask after level");
}

/// Go：`TestStateGainExpDoesNotUsePortraitAsLearnRow`
#[test]
fn state_gain_exp_does_not_use_portrait_as_learn_row() {
    let u = Unit {
        camp: OWN,
        portrait: 4,
        battle_fig: 32,
        has_battle_fig: true,
        lv: 3,
        max_hp: 1,
        hp: 1,
        ..Default::default()
    };
    let mut st = State {
        command_learn: Some(BTreeMap::from([(4, vec![CommandLearnEntry { required_level: 4, command_id: 24 }])])),
        command_learn_selectors: Some(BTreeMap::from([(32, 0xff)])),
        ..Default::default()
    };
    st.units.push(u);
    let _ = st.gain_exp(Some(0), 100.0, &mut GoMathRand::new(1));
    assert!(st.units[0].native_command_ids().is_empty(), "portrait was incorrectly used as learn row");
}

/// Go：`TestLoadCommandLearnUsesDenseNativeRows`
#[test]
fn load_command_learn_uses_dense_native_rows() {
    let Some(path) = docs_file("data/exe_tables/command_learn.json") else { return };
    let table = load_command_learn(&path).unwrap();
    let got = &table[&0];
    assert!(
        table.len() == 20
            && got[..2]
                == [
                    CommandLearnEntry { required_level: 5, command_id: 17 },
                    CommandLearnEntry { required_level: 9, command_id: 1 }
                ],
        "native learn table={got:?}"
    );
}

/// Go：`TestLoadCommandLearnSelectorsUsesRawBattleFigRows`
#[test]
fn load_command_learn_selectors_uses_raw_battle_fig_rows() {
    let Some(path) = docs_file("data/exe_tables/growth.json") else { return };
    let selectors = load_command_learn_selectors(&path).unwrap();
    assert_eq!(selectors.get(&32), Some(&4), "selector32 learn_idx");
    assert!(!selectors.contains_key(&4), "selector4 with raw learn_idx FF was materialized");
}

/// Go：`TestNativeCommandMaskRejectsMalformedSourceWithoutMutation`
#[test]
fn native_command_mask_rejects_malformed_source_without_mutation() {
    let mut u = Unit { native_command_mask: [1, 2, 3, 4, 5], ..Default::default() };
    assert!(u.set_initial_command_mask(&[1, 2, 3]).is_err(), "short source mask accepted");
    assert_eq!(u.native_command_mask, [1, 2, 3, 4, 5], "malformed source mutated mask");
}

/// Go：`TestLoadMaterializesFDFIELDInitialCommandMask`
#[test]
fn load_materializes_fdfield_initial_command_mask() {
    let path = temp_dir("command-mask").join("units.json");
    std::fs::write(
        &path,
        r#"{"map":0,"w":1,"h":1,"units":[{"camp":"enemy","hp":1,"map_selector_key":0,"native_record_byte5":128,"native_record_byte34":130,"native_record_byte35":18,"native_record_byte36":7,"initial_command_mask":[1,0,0,128]}]}"#,
    )
    .unwrap();
    let path_str = path.to_str().unwrap();
    let st = load(path_str).unwrap();
    let u = &st.units[0];
    assert_eq!(u.native_command_ids(), [0, 31], "loaded IDs");
    assert!(
        u.has_native_record_byte5
            && u.native_record_byte5 == 0x80
            && u.has_native_record_byte6
            && u.native_record_byte6 == 0,
        "loaded raw record bytes={:#x}/{} {:#x}/{}",
        u.native_record_byte5,
        u.has_native_record_byte5,
        u.native_record_byte6,
        u.has_native_record_byte6
    );
    assert!(
        u.has_native_record_byte34
            && u.native_record_byte34 == 0x82
            && u.has_native_record_byte35
            && u.native_record_byte35 == 0x12
            && u.has_native_record_byte36
            && u.native_record_byte36 == 7,
        "loaded native mode bytes"
    );
    std::fs::write(&path, r#"{"map":0,"w":1,"h":1,"units":[{"camp":"enemy","hp":1,"initial_command_mask":[1,2,3]}]}"#)
        .unwrap();
    assert!(load(path_str).is_err(), "malformed initial command mask loaded");
}
