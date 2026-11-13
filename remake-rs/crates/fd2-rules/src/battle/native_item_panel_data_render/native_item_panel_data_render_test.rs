//! 移植自 `remake/internal/battle/native_item_panel_data_render_test.go`。

use super::*;
use crate::test_support::{input, put16, remake_asset};

const RS: usize = NATIVE_RECORD_SIZE as usize;
const PANEL: usize = NATIVE_ITEM_PANEL_BYTES as usize;

/// Go：`nativeItemPanelTestStrings`
pub(crate) fn native_item_panel_test_strings(word: u16) -> fdtxt::Strings {
    const COUNT: usize = 500;
    let mut data = vec![0u8; COUNT * 2 + COUNT * 4];
    for index in 0..COUNT {
        let offset = COUNT * 2 + index * 4;
        put16(&mut data, index * 2, offset as u16);
        put16(&mut data, offset, word);
        put16(&mut data, offset + 2, fdtxt::STRING_END);
    }
    fdtxt::parse(&data).unwrap()
}

/// Go：`nativeItemPanelTestAssets`
pub(crate) fn native_item_panel_test_assets(text_word: u16) -> NativeItemPanelDataAssets {
    let raw: BTreeMap<i64, fdother::RawCell> =
        (0..=92).map(|index| (index, fdother::RawCell { width: 1, height: 1, pixels: vec![index as u8] })).collect();
    let frames: BTreeMap<i64, fdother::Frame> =
        (0..=140).map(|index| (index, fdother::Frame::raw(0, 0, 1, 1, vec![1, 0, 1, 0, 0, index as u8]))).collect();
    let mut font_data = vec![0u8; fdtxt::GLYPH_BYTES];
    font_data[0] = 0x80;
    NativeItemPanelDataAssets {
        raw_cells: Some(raw),
        frames: Some(frames),
        strings: Some(native_item_panel_test_strings(text_word)),
        font: Some(fdtxt::parse_font(&font_data).unwrap()),
        ..Default::default()
    }
}

/// Go：`TestRenderNativeItemPanelDataMatches17FC0Subpasses`
#[test]
fn render_native_item_panel_data_matches17_fc0_subpasses() {
    let assets = native_item_panel_test_assets(0);
    let mut record = vec![0u8; RS];
    record[6] = 0; // base icon selects entry 54
    record[37] = 1;
    put16(&mut record, 64, 5);
    put16(&mut record, 66, 10);
    put16(&mut record, 68, 0);
    put16(&mut record, 70, 10);
    put16(&mut record, 72, 12);
    record[34] = 1;

    let mut dst = vec![0u8; PANEL];
    render_native_item_panel_data(&assets, &record, &mut dst).unwrap();
    let at = |x: usize, y: usize| dst[y * 320 + x];
    // HP bar: base, 50 middle cells, then end. MP zero path starts at x+1.
    assert!(
        at(198, 33) == 23 && at(199, 33) == 24 && at(249, 33) == 25,
        "HP bar={}/{}/{}",
        at(198, 33),
        at(199, 33),
        at(249, 33)
    );
    assert!(
        at(198, 52) == 0 && at(199, 52) == 29 && at(300, 52) == 30,
        "MP zero bar={}/{}/{}",
        at(198, 52),
        at(199, 52),
        at(300, 52)
    );
    // The first compared number is unequal and therefore uses base 42.
    assert!(at(267, 41) == 42 && at(279, 41) == 47, "compared number={}/{}", at(267, 41), at(279, 41));
    // record+34 switches the +72 value to colour base 119.
    assert!(at(157, 67) == 119 && at(169, 67) == 121, "alternate number={}/{}", at(157, 67), at(169, 67));
    // Three FDTXT strings use foreground 205; base/flag icons follow after.
    assert!(at(99, 13) == 205 && at(211, 13) == 205 && at(251, 13) == 205, "text foreground");
    assert!(
        at(101, 30) == 54 && at(194, 68) == 55 && at(229, 68) == 0,
        "icons={}/{}/{}",
        at(101, 30),
        at(194, 68),
        at(229, 68)
    );
}

/// Go：`TestRenderNativeItemPanelDataFailsAtomicallyOnTextControl`
#[test]
fn render_native_item_panel_data_fails_atomically_on_text_control() {
    let assets = native_item_panel_test_assets(0xfffe);
    let mut record = vec![0u8; RS];
    put16(&mut record, 66, 1);
    put16(&mut record, 70, 1);
    let mut dst = vec![0x44u8; PANEL];
    let before = dst.clone();
    assert!(
        render_native_item_panel_data(&assets, &record, &mut dst).is_err(),
        "control-bearing item text was accepted"
    );
    assert!(dst == before, "failed dynamic overlay mutated destination");
}

/// Go：`TestRenderNativeFacilityNumberAccepts1B41DCurrencyWidth`
#[test]
fn render_native_facility_number_accepts1_b41_d_currency_width() {
    let assets = native_item_panel_test_assets(0);
    let mut dst = vec![0u8; PANEL];
    render_native_facility_number(&assets, &mut dst, 10, 20, 12345678, 31, 8).unwrap();
    for (digit, want) in [32u8, 33, 34, 35, 36, 37, 38, 39].into_iter().enumerate() {
        assert_eq!(dst[20 * 320 + 10 + digit * 6], want, "digit {digit}");
    }
}

/// Go：`TestRenderNativeItemPanelRowsMatches184C0GeometryAndSelection`
#[test]
fn render_native_item_panel_rows_matches184_c0_geometry_and_selection() {
    let assets = native_item_panel_test_assets(0);
    let mut record = vec![0u8; RS];
    for slot in 0..8 {
        record[0x0a + slot * 2] = 0x80;
    }
    (record[0x0a], record[0x0b]) = (0, 0);
    let mut effect_rows = vec![0u8; NATIVE_ITEM_EFFECT_ROW_SIZE as usize];
    put16(&mut effect_rows, 1, 12);
    let mut dst = vec![0u8; PANEL];
    render_native_item_panel_rows(&assets, &record, 0, &effect_rows, &mut dst).unwrap();
    assert_eq!(dst[101 * 320 + 13], 59, "category icon");
    assert_eq!(dst[103 * 320 + 42], 201, "selected item text");
    assert_eq!(dst[107 * 320 + 110], 64, "stat icon");
    assert!(
        dst[107 * 320 + 135] == 42 && dst[107 * 320 + 141] == 43 && dst[107 * 320 + 147] == 44,
        "stat number={}/{}/{}",
        dst[107 * 320 + 135],
        dst[107 * 320 + 141],
        dst[107 * 320 + 147]
    );
}

/// Go：`TestNativeItemPanelBaseAndDataWithPlayerAssets`
#[test]
fn native_item_panel_base_and_data_with_player_assets() {
    let (Some(fdother_path), Some(fdtxt_path), Some(_)) = (
        input("FD2_FLAME2_DIR", "FDOTHER.DAT"),
        input("FD2_FLAME2_DIR", "FDTXT.DAT"),
        input("FD2_ASSET_PACK", "portraits/DATO_000_m0.png"),
    ) else {
        return;
    };
    let (fdother_path, fdtxt_path) =
        (fdother_path.to_string_lossy().into_owned(), fdtxt_path.to_string_lossy().into_owned());
    let asset_pack_root = std::env::var("FD2_ASSET_PACK").unwrap();
    let portrait_root = format!("{asset_pack_root}/portraits");
    let mut dst = vec![0u8; PANEL];
    let mut record = vec![0u8; RS];
    for slot in 0..8 {
        record[0x0a + slot * 2] = 0x80;
    }
    (record[0x0a], record[0x0b]) = (0, 0);
    put16(&mut record, 64, 80);
    put16(&mut record, 66, 100);
    put16(&mut record, 68, 20);
    put16(&mut record, 70, 40);
    render_native_item_panel_resources(&asset_pack_root, &portrait_root, &record, &mut dst).unwrap();
    let mut oracle = vec![0u8; PANEL];
    render_native_item_panel_resources_archive(&fdother_path, &fdtxt_path, &portrait_root, &record, &mut oracle)
        .unwrap();
    assert!(dst == oracle, "separated native item panel differs from fixed archive");
    let separated_values = load_native_battle_panel_value_assets(&asset_pack_root).unwrap();
    let archive_values = load_native_battle_panel_value_assets_archive(&fdother_path).unwrap();
    let mut separated_panel = vec![0u8; PANEL];
    let mut archive_panel = vec![0u8; PANEL];
    let values = NativeBattlePanelValues { level: 12, hp: 80, max_hp: 100, mp: 20, max_mp: 40 };
    render_native_battle_panel_values_at(&separated_values, &mut separated_panel, 0, 154, values).unwrap();
    render_native_battle_panel_values_at(&archive_values, &mut archive_panel, 0, 154, values).unwrap();
    assert!(separated_panel == archive_panel, "separated native battle panel value render differs from fixed archive");
    let assets = load_native_item_panel_data_assets(&asset_pack_root).unwrap();
    let Some(rows_path) = remake_asset("data/native_item_effect_rows.json") else { return };
    let item_rows = load_native_item_effect_row_prefix(&rows_path).unwrap();
    render_native_item_panel_rows(&assets, &record, 0, &item_rows, &mut dst).unwrap();
    let nonzero = dst.iter().filter(|&&p| p != 0).count();
    assert!(nonzero >= 1000, "player complete item panel nonzero pixels={nonzero}");
}
