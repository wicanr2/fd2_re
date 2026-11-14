//! Go 來源：`remake/cmd/fd2/native_town_ui_test.go`（R4 E 組，期望值不改）。
//!
//! 未移植：`TestNativeTownShotStateIsStrictAndTownOnly`（`:188`，受測的 `parseNativeTownShotState`／
//! `setNativeTownShotState` 是截圖模式的 `r8` 函式）。
//!
//! Go 以 `t.Setenv("FD2_ASSET_PACK", …)` 指向分離素材包；Rust 由 `tools/rust_game_test.sh` 對整個測試行程設定
//! 同一個變數（`FD2_ORIGINAL_*` 只影響 Go 的封存檔退路，Rust 的載入器只讀分離素材包）。

use super::*;
use crate::test_support;

/// `FD2_ASSET_PACK` 存在才回傳 `true`；要求原版輸入時缺檔即失敗。
fn asset_pack_ready(rel: &str) -> bool {
    test_support::input("FD2_ASSET_PACK", rel).is_some()
}

fn town_campaign(
    variant: i64,
    gate: Option<campaign::NativeTownSecretGate>,
    secret: campaign::Node,
) -> campaign::Campaign {
    let mut nodes = BTreeMap::new();
    nodes.insert(
        String::from("town"),
        campaign::Node {
            r#type: String::from("town"),
            native_town_variant: Some(variant),
            native_secret_gate: gate,
            ..Default::default()
        },
    );
    nodes.insert(String::from("secret"), secret);
    campaign::Campaign { start: String::from("town"), nodes, ..Default::default() }
}

/// Go：`native_town_ui_test.go:13` `TestNativeTownAssetsDoNotRequireFDOTHERArchive`
#[test]
fn test_native_town_assets_do_not_require_fdother_archive() {
    if !asset_pack_ready("ui/fdother_010_town_label/resource.json") {
        return;
    }
    let assets = load_native_town_ui_assets().expect("loadNativeTownUIAssets").unwrap();
    assert!(assets.scene.is_some(), "separated town scene was not preflighted");
}

/// Go：`native_town_ui_test.go:32` `TestNativeTownProductionOwnerUsesEditableVariantAndHiddenSelection`
#[test]
fn test_native_town_production_owner_uses_editable_variant_and_hidden_selection() {
    if !asset_pack_ready("ui/fdother_010_town_label/resource.json") {
        return;
    }
    let shared = load_native_class_ui_assets().expect("loadNativeClassUIAssets").unwrap();
    let town = load_native_town_ui_assets().expect("loadNativeTownUIAssets").unwrap();
    let variant = 2;
    let c = town_campaign(
        variant,
        Some(campaign::NativeTownSecretGate { selection: 1, scan_code: 0x5f, to: String::from("secret") }),
        campaign::Node { r#type: String::from("shop"), native_hub_variant: 5, ..Default::default() },
    );
    let mut g = Game {
        camp: Some(campaign::new_runner(&c)),
        camp_sel: 1,
        native_class_ui: Some(shared.clone()),
        native_town_ui: Some(town.clone()),
        ..Default::default()
    };
    let (visible, ok) = g.compose_native_town_frame();
    assert!(ok && visible.len() == 320 * 200, "visible native town frame={} ok={ok}", visible.len());
    g.party_join_order = vec![7];
    let unit = |key: i64| {
        let mut u = battle::Unit::default();
        (u.map_selector_key, u.has_map_selector_key) = (key, true);
        u
    };
    g.party_roster = BTreeMap::from([(7, unit(50))]);
    let mut selected_scene = town.scene.clone().unwrap();
    let units = shared.units.as_ref().unwrap();
    for cycle in 0..selected_scene.pulse.len() {
        selected_scene.pulse[cycle] = units.sprite_for(50, 0, cycle as i64).unwrap().clone();
    }
    let want_selected = campaign::compose_native_town_frame(
        Some(&selected_scene),
        shared.strings.as_ref(),
        shared.font.as_ref(),
        variant,
        g.camp_sel,
        g.native_town_ui_pulse,
    )
    .unwrap();
    let (selected, ok) = g.compose_native_town_frame();
    assert!(
        ok && selected == want_selected && selected != visible,
        "town cursor did not follow the first persistent map selector key"
    );
    g.party_roster.insert(7, unit(999));
    assert!(!g.compose_native_town_frame().1, "out-of-bank persistent map selector key did not fail closed");
    g.party_roster.insert(7, unit(50));
    let camp = g.camp.as_ref().unwrap();
    assert!(
        camp.match_native_town_secret(g.camp_sel, 0x5f) && camp.node_id() == "town",
        "native secret chord did not remain in the town owner"
    );
    g.camp_sel = 5;
    let (hidden, ok) = g.compose_native_town_frame();
    assert!(ok && visible != hidden, "hidden selection did not redraw the native town frame");
    let sel = g.camp_sel;
    let camp = g.camp.as_mut().unwrap();
    assert!(
        camp.confirm_native_town_secret(sel) && camp.node_id() == "secret",
        "confirmed hidden selection did not dispatch variant 5"
    );
}

/// Go：`native_town_ui_test.go:119` `TestNativeTownPulseUsesFourTickSignedDelta`
#[test]
fn test_native_town_pulse_uses_four_tick_signed_delta() {
    let mut g = Game::default();
    for (tick, want) in [(0x7ffe, 0), (0x7fff, 0), (-0x7ffe, 1), (-0x7ffa, 2), (-0x7ff6, 3), (-0x7ff2, 0)] {
        g.step_native_town_ui_pulse_tick(tick);
        assert_eq!(g.native_town_ui_pulse, want, "tick {tick:#x}");
    }
}

/// Go：`native_town_ui_test.go:142` `TestNativeTownSelectionUsesOriginalLeftRightWrap`
#[test]
fn test_native_town_selection_uses_original_left_right_wrap() {
    for (selection, delta, want) in [(0, -1, 4), (4, 1, 0), (2, -1, 1), (2, 1, 3), (5, -1, 4), (5, 1, 0)] {
        let (got, ok) = native_town_move_selection(selection, delta);
        assert!(ok && got == want, "selection={selection} delta={delta} got={got} ok={ok} want={want}");
    }
    for (selection, delta) in [(-1, 1), (6, -1), (0, 0)] {
        assert!(!native_town_move_selection(selection, delta).1, "invalid move ({selection},{delta}) was accepted");
    }
    let mut g = Game {
        camp_sel: 0,
        native_town_ui_pulse: 2,
        native_town_ui_last_tick: 321,
        native_town_ui_has_tick: true,
        ..Default::default()
    };
    assert!(g.move_native_town_selection(1) && g.camp_sel == 1, "runtime move selection={}", g.camp_sel);
    assert!(
        g.native_town_ui_pulse == 2 && g.native_town_ui_last_tick == 321 && g.native_town_ui_has_tick,
        "selection move reset pulse"
    );
}

/// Go：`native_town_ui_test.go:245` `TestNativeTownSecretRevealPreservesPulseState`
#[test]
fn test_native_town_secret_reveal_preserves_pulse_state() {
    let c = town_campaign(
        0,
        Some(campaign::NativeTownSecretGate { selection: 0, scan_code: 0x54, to: String::from("secret") }),
        campaign::Node { r#type: String::from("shop"), ..Default::default() },
    );
    let mut g = Game {
        camp_sel: 0,
        camp: Some(campaign::Runner { cur: String::from("town"), c: Some(c), ..Default::default() }),
        native_town_ui_pulse: 2,
        native_town_ui_last_tick: 456,
        native_town_ui_has_tick: true,
        ..Default::default()
    };
    assert!(g.reveal_native_town_secret(0x54) && g.camp_sel == 5, "secret reveal selection={}", g.camp_sel);
    assert!(
        g.native_town_ui_pulse == 2 && g.native_town_ui_last_tick == 456 && g.native_town_ui_has_tick,
        "secret reveal reset pulse"
    );
    assert!(!g.reveal_native_town_secret(0x54), "hidden selection accepted the normal-selection chord again");
}
