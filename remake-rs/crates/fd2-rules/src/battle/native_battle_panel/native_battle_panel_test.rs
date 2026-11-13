//! 移植自 `remake/internal/battle/native_battle_panel_test.go`。

use super::*;
use crate::battle::native_item_panel_data_render::native_item_panel_data_render_test::native_item_panel_test_assets;

const PANEL: usize = NATIVE_ITEM_PANEL_BYTES as usize;

fn native_battle_panel_test_assets() -> NativeItemPanelDataAssets {
    let mut assets = native_item_panel_test_assets(0);
    assets.battle_panel = fdother::LMI1Entry {
        width: NATIVE_BATTLE_PANEL_WIDTH,
        height: NATIVE_BATTLE_PANEL_HEIGHT,
        pixels: vec![9; (NATIVE_BATTLE_PANEL_WIDTH * NATIVE_BATTLE_PANEL_HEIGHT) as usize],
    };
    assets
}

fn native_battle_panel_test_record(raw_side: u8) -> Vec<u8> {
    native_battle_panel_record_for_unit(Some(&Unit {
        lv: 4,
        hp: 5,
        max_hp: 10,
        mp: 0,
        max_mp: 8,
        native_record_byte6: raw_side,
        has_native_record_byte6: true,
        native_record_byte8: 0,
        has_native_record_byte8: true,
        ..Default::default()
    }))
    .unwrap()
}

/// Go：`TestRenderNativeBattlePanelMatches18C6DBottomGeometry`
#[test]
fn render_native_battle_panel_matches18_c6_d_bottom_geometry() {
    let mut dst = vec![0u8; PANEL];
    render_native_battle_panel(&native_battle_panel_test_assets(), &native_battle_panel_test_record(0), &mut dst, 3, 7)
        .unwrap();
    assert!(
        dst[154 * 320] == 9 && dst[195 * 320 + 148] == 9,
        "149x42 opaque panel was not placed at raw-side zero origin"
    );
    assert!(
        dst[(154 + 22) * 320 + 21] == 23 && dst[(154 + 22) * 320 + 22] == 24 && dst[(154 + 22) * 320 + 72] == 25,
        "HP bar did not use entries 23/24/25 at the native origin"
    );
    assert!(
        dst[(154 + 31) * 320 + 22] == 29 && dst[(154 + 31) * 320 + 123] == 30,
        "zero MP bar did not use entries 29/30"
    );
    assert!(
        dst[(154 + 4) * 320 + 132] == 31 && dst[(154 + 4) * 320 + 138] == 35,
        "two-digit level was not drawn with base31 and six-pixel advance"
    );
    assert_eq!(dst[(154 + 4) * 320 + 5], 205, "raw record+8 FDTXT name was not drawn at panel origin+(5,4)");
}

/// Go：`TestRenderNativeBattlePanelWithoutNameKeepsNativeGeometry`
#[test]
fn render_native_battle_panel_without_name_keeps_native_geometry() {
    let mut assets = native_battle_panel_test_assets();
    assets.strings = None;
    assets.font = None;
    let mut dst = vec![0u8; PANEL];
    let record = native_battle_panel_test_record(0);
    render_native_battle_panel_without_name(&assets, &record, &mut dst, 3, 7).unwrap();
    let origin = native_battle_panel_origin(&record, 3, 7);
    assert!(matches!(origin, Ok((0, 154))), "origin={origin:?}");
    assert!(dst[154 * 320] == 9 && dst[(154 + 22) * 320 + 21] == 23, "name-free panel lost native base or HP geometry");
    assert_ne!(dst[(154 + 4) * 320 + 5], 205, "name-free panel unexpectedly rendered FDTXT text");
}

/// Go：`TestRenderNativeBattlePanelValuesAtReuses18C6DBarAndDigitCore`
#[test]
fn render_native_battle_panel_values_at_reuses18_c6_d_bar_and_digit_core() {
    let mut dst = vec![0u8; PANEL];
    let values = NativeBattlePanelValues { level: 2, hp: 8, max_hp: 28, mp: 0, max_mp: 0 };
    render_native_battle_panel_values_at(&native_battle_panel_test_assets(), &mut dst, 0, 154, values).unwrap();
    assert!(
        dst[(154 + 22) * 320 + 21] == 23 && dst[(154 + 22) * 320 + 22] == 24 && dst[(154 + 22) * 320 + 50] == 25,
        "HP 8/28 did not consume native 23/24/25 cells and +1 endpoint"
    );
    assert!(
        dst[(154 + 4) * 320 + 132] == 31 && dst[(154 + 4) * 320 + 138] == 33,
        "level 02 did not consume native digit frames"
    );
}

/// Go：`TestRenderNativeBattlePanelValuesAtFailsAtomically`
#[test]
fn render_native_battle_panel_values_at_fails_atomically() {
    let mut assets = native_battle_panel_test_assets();
    assets.frames.as_mut().unwrap().remove(&31);
    let mut dst = vec![77u8; PANEL];
    let want = dst.clone();
    let values = NativeBattlePanelValues { level: 2, hp: 8, max_hp: 28, ..Default::default() };
    assert!(
        render_native_battle_panel_values_at(&assets, &mut dst, 0, 154, values).is_err(),
        "missing digit frame was accepted"
    );
    assert!(dst == want, "failed value render mutated its destination");
}

/// Go：`TestRenderNativeBattlePanelUsesTopAndRawChapter24Exception`
#[test]
fn render_native_battle_panel_uses_top_and_raw_chapter24_exception() {
    let assets = native_battle_panel_test_assets();
    let mut top = vec![0u8; PANEL];
    render_native_battle_panel(&assets, &native_battle_panel_test_record(1), &mut top, 17, 23).unwrap();
    assert!(top[4 * 320 + 171] == 9 && top[154 * 320] == 0, "nonzero raw side did not select the top panel");
    let mut exception = vec![0u8; PANEL];
    render_native_battle_panel(&assets, &native_battle_panel_test_record(1), &mut exception, 17, 24).unwrap();
    assert!(
        exception[154 * 320] == 9 && exception[4 * 320 + 171] == 0,
        "raw chapter24 unit17 exception did not select the bottom panel"
    );
}

/// Go：`TestRenderNativeBattlePanelFailsAtomically`
#[test]
fn render_native_battle_panel_fails_atomically() {
    let mut assets = native_battle_panel_test_assets();
    assets.raw_cells.as_mut().unwrap().remove(&23);
    let mut dst = vec![77u8; PANEL];
    let want = dst.clone();
    assert!(
        render_native_battle_panel(&assets, &native_battle_panel_test_record(0), &mut dst, 0, 0).is_err(),
        "missing raw HP cell was accepted"
    );
    assert!(dst == want, "failed panel render mutated its destination");
}

/// Go：`TestNativeBattlePanelRecordRequiresRawSelectors`
#[test]
fn native_battle_panel_record_requires_raw_selectors() {
    let unit = Unit { lv: 1, hp: 1, max_hp: 1, ..Default::default() };
    assert!(native_battle_panel_record_for_unit(Some(&unit)).is_err(), "missing raw +6/+8 provenance was accepted");
}
