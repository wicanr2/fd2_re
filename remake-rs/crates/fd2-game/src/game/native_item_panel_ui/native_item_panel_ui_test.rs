//! Go 來源：`remake/cmd/fd2/native_item_panel_ui_test.go` 的傳送物品測試、`native_command23_presentation_test.go`，
//! 以及兩者共用的 `native_unit_present_test.go` 測具。
//!
//! 測具全用合成素材（Go `completeNativeMapFrameFixture`），只讀受版控的 `remake/assets` 資料表。

use super::*;
use crate::game::native_map_frame_input::native_map_frame_input_test::complete_native_map_frame_fixture;
use crate::test_support;

/// 子行程裡跑，設 `FD2_MUTE=1`（Go `t.Setenv("FD2_MUTE", "1")`）。
macro_rules! child {
    ($name:literal) => {
        if f_child_env(module_path!(), $name, |_| vec![("FD2_MUTE", "1".into())]).is_none() {
            return;
        }
    };
}

/// 20×15 合成地圖，三個單位：(0,0)、(5,5)、(6,5)。
///
/// Go：`native_unit_present_test.go:13` `completeNativeUnitPresentGame`
fn complete_native_unit_present_game() -> Game {
    let (mut assets, mut field, mut state) = complete_native_map_frame_fixture();
    (field.w, field.h) = (20, 15);
    field.tiles = vec![0; 20 * 15];
    field.native_tile_blit_modes = vec![0xff; 20 * 15];
    (state.w, state.h) = (field.w, field.h);
    state.native_tile_blit_modes = field.native_tile_blit_modes.clone();
    state
        .materialize_native_map_view_state(battle::NativeMapViewState {
            camera_x: 4,
            camera_y: 3,
            cursor_x: 10,
            cursor_y: 7,
            visible_cursor_x: 6,
            visible_cursor_y: 4,
        })
        .expect("view state");
    for i in 0..2 {
        let mut clone = state.units[0].clone();
        (clone.map_selector_slot, clone.has_map_selector_slot) = (0, false);
        (clone.native_map_presentation, clone.has_native_map_presentation) = (Default::default(), false);
        (clone.x, clone.y) = (5 + i, 5);
        state.append_native_map_selector_batch(vec![clone]).expect("selector batch");
    }
    for (table, lut) in assets.luts.iter_mut().enumerate() {
        for (value, entry) in lut.iter_mut().enumerate() {
            *entry = ((value + table + 1) & 0xff) as u8;
        }
    }
    let mut g = Game {
        native_map_assets: Some(assets),
        m: Some(field),
        st: Some(state),
        cam_x: 4.0 * 24.0,
        cam_y: 3.0 * 24.0,
        // Go 測具沒給時鐘時走 `time.Now`；Rust 的地圖時鐘要有虛擬時間來源。
        shot_path: String::from("relocation-test"),
        virtual_now: VIRTUAL_CLOCK_ORIGIN,
        ..Default::default()
    };
    g.locale_id = String::from("zh-Hant");
    g.locale_catalog = load_official_locale("zh-Hant").expect("locale");
    g.locale_content = load_official_locale_content("zh-Hant").expect("content");
    g.locale_entities = load_official_locale_entities("zh-Hant").expect("entities");
    g.compose_native_map_frame().expect("compose");
    g
}

/// Go：`native_unit_present_test.go:69` `stepNativeUnitPresentNow`
fn step_native_unit_present_now(g: &mut Game) {
    let j = g.native_unit_present.as_mut().unwrap();
    (j.drawn, j.wait) = (true, 0);
    g.step_native_unit_present();
}

fn unit(g: &Game, index: UnitIdx) -> &battle::Unit {
    &g.st.as_ref().unwrap().units[index]
}

fn unit_mut(g: &mut Game, index: UnitIdx) -> &mut battle::Unit {
    &mut g.st.as_mut().unwrap().units[index]
}

fn flash_table() -> Option<battle::NativeCommandPaletteFlashTable> {
    let path = test_support::remake_asset("data/native_command_palette_flash.json")?;
    Some(battle::load_native_command_palette_flash_table(&path).expect("flash"))
}

/// 兩名我方：施放者 (0,0)、目標 (1,0)；地形碼全為 2。
///
/// Go：`native_command23_presentation_test.go:10` `nativeCommand23TestGame`
fn command23_test_game() -> Option<(Game, UnitIdx, UnitIdx)> {
    let spells = test_support::remake_asset("spells.json")?;
    let flash = flash_table()?;
    let mut g = complete_native_unit_present_game();
    let st = g.st.as_mut().unwrap();
    st.units.truncate(2);
    for (i, unit) in st.units.iter_mut().enumerate() {
        (unit.camp, unit.on_field) = (battle::OWN, true);
        (unit.native_identity, unit.has_native_identity) = (7 + i as i64, true);
        (unit.native_record_class, unit.has_native_record_class) = (1, true);
        (unit.native_record_race, unit.has_native_record_race) = (1, true);
        (unit.native_record_byte5, unit.has_native_record_byte5) = (0, true);
        (unit.native_record_byte6, unit.has_native_record_byte6) = (0, true);
        unit.inventory_slots = vec![255; 8].into();
        unit.native_inventory_flags = vec![128; 8].into();
        assert!(unit.set_native_map_coordinates_raw(i as i64, 0), "raw map coordinates unavailable");
    }
    (st.units[0].mp, st.units[0].max_mp) = (30, 40);
    st.native_command_book = battle::load_native_command_records(&spells).expect("spells");
    g.native_command_book = st.native_command_book.clone();
    g.native_command_palette_flash = Some(flash);
    let st = g.st.as_mut().unwrap();
    st.native_composition_event_bytes = vec![0; (st.w * st.h) as usize];
    st.native_terrain_move_codes = vec![2; (st.w * st.h) as usize];
    st.materialize_native_map_view_state(battle::NativeMapViewState::default()).expect("view state");
    g.native_map_dac = g.native_map_assets.as_ref().unwrap().palette_dac.clone();
    (g.sel, g.moved, g.cur_x, g.cur_y) = (Some(0), true, 0, 0);
    (g.cam_x, g.cam_y) = (0.0, 0.0);
    g.compose_native_map_frame().expect("compose");
    Some((g, 0, 1))
}

/// Go：`native_command23_presentation_test.go:56` `TestNativeCommand23FormalConfirmCancelAndFullPresentation`
#[test]
fn test_native_command23_formal_confirm_cancel_and_full_presentation() {
    child!("test_native_command23_formal_confirm_cancel_and_full_presentation");
    let Some((mut g, actor, target)) = command23_test_game() else { return };
    g.begin_native_command_target_selection(23).unwrap();
    g.confirm();
    assert!(!g.native_item_relocating && unit(&g, actor).mp == 30, "spell23 accepted its caster despite inner radius1");
    (g.cur_x, g.cur_y) = (unit(&g, target).x, unit(&g, target).y);
    g.confirm();
    assert!(
        g.native_command23_relocating
            && g.native_item_relocating
            && !g.native_command0_targeting
            && unit(&g, actor).mp == 30,
        "spell23 did not enter its independent destination stage: {}",
        g.msg
    );
    assert!(
        g.cancel_native_item_target_modal() && g.native_command_open && !g.item_open && !g.native_command23_relocating,
        "spell23 destination cancel did not return to command grid"
    );
    g.begin_native_command_target_selection(23).unwrap();
    g.confirm();
    (g.cur_x, g.cur_y) = (unit(&g, actor).x, unit(&g, actor).y);
    g.confirm();
    assert!(
        g.native_modifier_presentation.is_none() && unit(&g, target).x == 1 && unit(&g, actor).mp == 30,
        "spell23 accepted occupied destination"
    );
    (g.cur_x, g.cur_y) = (8, 7);
    g.confirm();
    assert!(
        g.native_modifier_presentation.is_some() && unit(&g, target).x == 1 && unit(&g, actor).mp == 30,
        "spell23 did not preflight complete palette/disappear/appear chain: {}",
        g.msg
    );
    let mut screen = Picture::new_image(640, 400);
    for _ in 0..20 {
        if g.native_modifier_presentation.is_none() {
            break;
        }
        g.present_native_command_modifier(Some(&mut screen));
        g.step_native_command_modifier_presentation();
    }
    let (mut stages, mut saw_off_map) = (0, false);
    for _ in 0..120 {
        if g.native_unit_present.is_none() {
            break;
        }
        g.draw_native_unit_present(Some(&mut screen));
        step_native_unit_present_now(&mut g);
        stages += 1;
        let t = unit(&g, target);
        saw_off_map |= t.x == 255 && t.y == 255;
    }
    let (a, t) = (unit(&g, actor), unit(&g, target));
    assert!(
        g.native_modifier_presentation.is_none()
            && g.native_unit_present.is_none()
            && stages == 96
            && saw_off_map
            && (t.x, t.y) == (8, 7)
            && a.mp == 10
            && a.acted
            && !g.native_command23_relocating
            && !g.native_item_relocating
            && g.sel.is_none()
            && g.load_err.is_empty(),
        "spell23 incomplete: stages{stages} offmap{saw_off_map} MP{} acted{} target({},{}) error{}",
        a.mp,
        a.acted,
        t.x,
        t.y,
        g.load_err
    );
}

/// Go：`native_command23_presentation_test.go:107` `TestNativeCommand23InsufficientMPOrMissingArtIsAtomic`
#[test]
fn test_native_command23_insufficient_mp_or_missing_art_is_atomic() {
    for missing in [false, true] {
        let Some((mut g, actor, target)) = command23_test_game() else { return };
        g.begin_native_command_target_selection(23).unwrap();
        (g.cur_x, g.cur_y) = (unit(&g, target).x, unit(&g, target).y);
        g.confirm();
        if missing {
            g.native_map_assets.as_mut().unwrap().fdother6[0x72].pixels = Vec::new();
        } else {
            unit_mut(&mut g, actor).mp = 19;
        }
        let before_mp = unit(&g, actor).mp;
        (g.cur_x, g.cur_y) = (8, 7);
        g.confirm();
        let (a, t) = (unit(&g, actor), unit(&g, target));
        assert!(
            g.native_modifier_presentation.is_none()
                && g.native_unit_present.is_none()
                && a.mp == before_mp
                && !a.acted
                && (t.x, t.y) == (1, 0)
                && g.native_command23_relocating,
            "rejected spell23 changed transaction state (missing={missing}): {}",
            g.msg
        );
    }
}

/// Go：`native_item_panel_ui_test.go:443` `TestNativeRelocationUsesSeparateDestinationCursorAndMode6Legality`
#[test]
fn test_native_relocation_uses_separate_destination_cursor_and_mode6_legality() {
    child!("test_native_relocation_uses_separate_destination_cursor_and_mode6_legality");
    let (Some(flash), Some(rows)) = (flash_table(), test_support::remake_asset("data/native_item_effect_rows.json"))
    else {
        return;
    };
    let mut g = complete_native_unit_present_game();
    let (actor, target) = (0, 1);
    let a = unit_mut(&mut g, actor);
    (a.x, a.y, a.on_field, a.camp) = (0, 0, true, battle::OWN);
    (a.native_identity, a.has_native_identity) = (24, true);
    (a.native_record_byte5, a.has_native_record_byte5) = (0, true);
    (a.mp, a.max_mp) = (30, 40);
    (a.inventory, a.equipped) = (vec![101].into(), vec![false].into());
    a.inventory_slots = vec![101, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff].into();
    a.native_inventory_flags = vec![0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80].into();
    let t = unit_mut(&mut g, target);
    (t.x, t.y, t.on_field, t.camp) = (1, 0, true, battle::OWN);
    assert!(t.set_native_map_coordinates_raw(1, 0), "target raw map coordinates unavailable");
    (t.native_identity, t.has_native_identity) = (1, true);
    (t.native_record_class, t.has_native_record_class) = (1, true);
    (t.native_record_byte5, t.has_native_record_byte5) = (0, true);
    t.inventory_slots = vec![0xff; 8].into();
    t.native_inventory_flags = vec![0x80; 8].into();
    let st = g.st.as_mut().unwrap();
    st.units.truncate(2);
    let mut book: Vec<battle::NativeCommandRecord> = (0..battle::NATIVE_COMMAND_RECORD_COUNT)
        .map(|id| battle::NativeCommandRecord { id, ..Default::default() })
        .collect();
    book[23].mp_cost = 20;
    let cells = (st.w * st.h) as usize;
    st.native_composition_event_bytes = vec![0; cells];
    st.native_terrain_move_codes = vec![2; cells];
    st.native_tile_blit_modes[1] = 0;
    st.materialize_native_map_view_state(battle::NativeMapViewState {
        cursor_x: 1,
        visible_cursor_x: 1,
        ..Default::default()
    })
    .expect("view state");
    g.m.as_mut().unwrap().native_tile_blit_modes[1] = 0;
    (g.sel, g.moved, g.item_open, g.native_command_book) = (Some(actor), true, true, book);
    g.native_map_dac = g.native_map_assets.as_ref().unwrap().palette_dac.clone();
    (g.cam_x, g.cam_y) = (0.0, 0.0);
    g.compose_native_map_frame().expect("compose");
    g.native_command_palette_flash = Some(flash);
    g.native_item_effect_rows = battle::load_native_item_effect_row_prefix(&rows).expect("item rows");

    assert!(matches!(g.begin_native_target_item(0, 101), Ok(true)), "targeting");
    let want_first =
        i64::from(g.native_item_effect_rows[(101 * battle::NATIVE_ITEM_EFFECT_ROW_SIZE + 0x12) as usize]) + 2;
    assert_eq!(g.st.as_ref().unwrap().native_map_range_mode, want_first, "type23 first-stage selector");
    assert!(
        matches!(g.apply_native_target_item(target), Ok(false)) && g.native_item_relocating && !g.native_item_targeting,
        "target stage"
    );
    let all_ff = vec![0xffu8; cells];
    let field_reset = |g: &Game| {
        let st = g.st.as_ref().unwrap();
        st.native_map_range_mode == 1 && st.native_tile_blit_modes == all_ff
    };
    assert!(field_reset(&g), "destination stage selector/grid");
    assert!(
        g.cancel_native_item_target_modal()
            && g.item_open
            && !g.native_item_targeting
            && !g.native_item_relocating
            && field_reset(&g),
        "destination cancel did not return to item panel"
    );
    assert!(matches!(g.begin_native_target_item(0, 101), Ok(true)), "retargeting");
    assert!(matches!(g.apply_native_target_item(target), Ok(false)) && g.native_item_relocating, "second target stage");
    let destinations = g.native_relocation_destinations();
    assert!(
        destinations.contains(&battle::Cell { x: 2, y: 2 }) && !destinations.contains(&battle::Cell { x: 0, y: 0 }),
        "mode6 destinations={destinations:?}"
    );
    let intro = std::mem::take(&mut g.native_map_assets.as_mut().unwrap().fdother6[0x72].pixels);
    let applied = g.apply_native_relocation_destination(8, 7);
    let (a, t) = (unit(&g, actor), unit(&g, target));
    assert!(
        applied.is_err()
            && g.native_modifier_presentation.is_none()
            && g.native_unit_present.is_none()
            && (t.x, t.y) == (1, 0)
            && a.mp == 30
            && !a.acted,
        "missing renderer crossed atomic boundary: {applied:?}"
    );
    g.native_map_assets.as_mut().unwrap().fdother6[0x72].pixels = intro;
    assert!(matches!(g.apply_native_relocation_destination(8, 7), Ok(true)), "destination applied");
    let (a, t) = (unit(&g, actor), unit(&g, target));
    assert!(
        g.native_modifier_presentation.is_some() && (t.x, t.y) == (1, 0) && a.mp == 30 && !a.acted,
        "relocation crossed palette boundary"
    );
    for phase in 0..8 {
        assert!(
            g.present_native_command_modifier(Some(&mut Picture::new_image(640, 400))),
            "command23 palette phase {phase} did not draw"
        );
        g.step_native_command_modifier_presentation();
    }
    let (a, t) = (unit(&g, actor), unit(&g, target));
    assert!(
        g.native_modifier_presentation.is_none()
            && g.native_unit_present.is_some()
            && (t.x, t.y) == (1, 0)
            && a.mp == 30,
        "command23 did not enter disappear presentation atomically"
    );
    let mut saw_off_map = false;
    for _ in 0..120 {
        if g.native_unit_present.is_none() {
            break;
        }
        step_native_unit_present_now(&mut g);
        let t = unit(&g, target);
        saw_off_map |= t.x == 0xff && t.y == 0xff;
    }
    let (a, t) = (unit(&g, actor), unit(&g, target));
    assert!(
        g.native_unit_present.is_none()
            && saw_off_map
            && (t.x, t.y) == (8, 7)
            && a.mp == 10
            && a.inventory[..] == [101]
            && a.acted
            && g.sel.is_none()
            && !g.native_item_relocating
            && field_reset(&g),
        "relocation actor mp={} acted={} target=({},{}) err={}",
        a.mp,
        a.acted,
        t.x,
        t.y,
        g.load_err
    );
}

/// Rust 補充：Go `main.go:5896..5917` 缺原版面板時的後備清單。上下鍵在 0..7 之間移動，Enter 只顯示訊息，
/// 不使用物品也不動道具欄；TTF 繪製沒有字型時不畫。
#[test]
fn test_fallback_item_list_moves_within_eight_cells_and_never_uses_items() {
    if test_support::remake_asset("locales/zh-Hant/pack.json").is_none() {
        return;
    }
    let mut unit = battle::Unit::default();
    (unit.camp, unit.on_field) = (battle::OWN, true);
    *unit.inventory = vec![0x20];
    let mut st = battle::State::default();
    st.units.push(unit);
    let mut g = Game { st: Some(st), sel: Some(0), item_open: true, ..Default::default() };
    crate::game::save::save_test::attach_official_locale(&mut g);
    let press = |g: &mut Game, key: Key| g.tick_input = TickInput { pressed: [key].into(), ..Default::default() };
    press(&mut g, Key::ArrowUp);
    assert!(g.native_item_panel_input(false, false) && g.item_sel == 0, "moved above the first cell");
    for _ in 0..10 {
        press(&mut g, Key::ArrowDown);
        g.native_item_panel_input(false, false);
    }
    assert_eq!(g.item_sel, 7, "fallback list must stop at the eighth cell");
    g.tick_input = TickInput::default();
    g.native_item_panel_input(true, false);
    let (empty, _) = g.locale_message("battle.item.empty_slot", &[]);
    assert_eq!(g.msg, empty);
    g.item_sel = 0;
    g.native_item_panel_input(true, false);
    let (unverified, _) = g.locale_message("battle.item.effect_unverified", &[localization::FormatArg::Int(0x20)]);
    assert!(g.msg == unverified && g.msg.contains("20h"), "msg={:?}", g.msg);
    assert!(g.item_open && *g.st.as_ref().unwrap().units[0].inventory == [0x20], "fallback mutated the inventory");
    let mut screen = Picture::new_image(LOGICAL_W, LOGICAL_H);
    g.draw_item_menu(&mut screen);
}
