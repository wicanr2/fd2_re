//! Go 來源：`remake/cmd/fd2/native_command3_presentation_test.go`（玩家正式指令入口）與測試 helper
//! `nativeCommand32PresentationFixture`（`native_command32_presentation_test.go:25`）、
//! `ordinaryCommand0HDTestGame`（`modern_hd_command0_test.go:124`）。

use super::*;
use crate::game::native_map_frame_input::native_map_frame_input_test::complete_native_map_frame_fixture;
use crate::test_support;

/// 第一章 map0：`(0,0)` 我方施放者、`(1,0)` 敵方目標，原版 HUD／範圍模式就緒。缺分離素材包或原版地圖時回傳
/// `None`。Go 另設 `FD2_ORIGINAL_*` 指到原版檔；Rust 的指令演出只讀分離素材包。HD 面板（Go
/// `maybeAttachCommandPanelHDTest`）不在這裡。
fn ordinary_command_test_game() -> Option<(Game, UnitIdx, UnitIdx)> {
    test_support::input("FD2_ASSET_PACK", "surfaces/BG_000/resource.json")?;
    let (scene, flash) = (
        test_support::remake_asset("data/native_command_scene.json")?,
        test_support::remake_asset("data/native_command_palette_flash.json")?,
    );
    let mut g = Game {
        rng: Some(battle::GoMathRand::new(32)),
        native_ui_palette: load_native_ui_palette(),
        shot_path: String::from("command-test"),
        virtual_now: VIRTUAL_CLOCK_ORIGIN,
        ..Default::default()
    };
    g.locale_id = String::from("zh-Hant");
    g.locale_catalog = load_official_locale("zh-Hant").expect("locale");
    g.locale_content = load_official_locale_content("zh-Hant").expect("content");
    g.locale_entities = load_official_locale_entities("zh-Hant").expect("entities");
    g.load_map("assets/maps/map0").unwrap();
    if g.native_map_assets.is_none() {
        crate::test_support::skip("原版地圖素材（FD2_FLAME2_DIR）");
        return None;
    }
    g.reset_battle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json");
    let st = g.st.as_mut().unwrap();
    assert!(g.load_err.is_empty() && st.units.len() >= 5, "fixture err={} units={}", g.load_err, st.units.len());
    let (actor, target) = (0, 4);
    assert!(st.units[actor].set_map_placement(0, 0, 3) && st.units[target].set_map_placement(1, 0, 1));
    st.units[actor].materialize_native_map_presentation().unwrap();
    st.units[target].materialize_native_map_presentation().unwrap();
    {
        let a = &mut st.units[actor];
        (a.camp, a.hp, a.max_hp, a.mp) = (battle::OWN, 100, 100, 80);
        (a.class_id, a.battle_fig, a.has_battle_fig) = (5, 9, true);
        (a.native_record_class, a.has_native_record_class) = (19, true);
        let t = &mut st.units[target];
        (t.camp, t.hp, t.max_hp) = (battle::ENEMY, 100, 100);
        (t.class_id, t.battle_fig, t.has_battle_fig) = (5, 0, true);
        (t.native_record_class, t.has_native_record_class) = (2, true);
    }
    let mut book: Vec<battle::NativeCommandRecord> = (0..battle::NATIVE_COMMAND_RECORD_COUNT)
        .map(|id| battle::NativeCommandRecord { id, ..Default::default() })
        .collect();
    book[0] = battle::NativeCommandRecord {
        id: 0,
        damage: 50,
        hit: 100,
        selection_mode: 5,
        effect_mode: 0,
        mp_cost: 2,
        target_code: 0,
    };
    st.native_command_book = book;
    st.native_command_resistances = BTreeMap::from([(2, 10)]);
    st.native_composition_event_bytes = vec![0; (st.w * st.h) as usize];
    st.materialize_native_map_view_state(battle::NativeMapViewState {
        camera_x: 0,
        camera_y: 0,
        cursor_x: 1,
        cursor_y: 0,
        visible_cursor_x: 1,
        visible_cursor_y: 0,
    })
    .unwrap();
    assert!(st.materialize_native_map_hud_state(1, 1, 1) && st.materialize_native_map_range_mode(1));
    (g.cur_x, g.cur_y) = (1, 0);
    g.compose_native_map_frame().unwrap();
    g.native_rng_state = 7;
    g.native_command_scene = Some(battle::load_native_command_scene_table(&scene).expect("scene"));
    g.native_command_palette_flash = Some(battle::load_native_command_palette_flash_table(&flash).expect("flash"));
    Some((g, actor, target))
}

fn native_damage_command_active(g: &Game) -> bool {
    g.native_cmd0_presentation.is_some()
        || g.native_cmd2_presentation.is_some()
        || g.native_cmd3_presentation.is_some()
        || g.native_cmd5_presentation.is_some()
        || g.native_cmd6_presentation.is_some()
        || g.native_cmd7_presentation.is_some()
        || g.native_cmd8_presentation.is_some()
}

fn present_and_step_native_damage_command(g: &mut Game, screen: &mut Picture) {
    if g.native_cmd2_presentation.is_some() {
        g.present_native_command2(Some(screen));
        g.step_native_command2_presentation();
    } else if g.native_cmd3_presentation.is_some() {
        g.present_native_command3(Some(screen));
        g.step_native_command3_presentation();
    } else if g.native_cmd5_presentation.is_some() {
        g.present_native_indexed_damage_command(Some(screen));
        g.step_native_command5_presentation();
    } else if g.native_cmd7_presentation.is_some() {
        g.present_native_command7(Some(screen));
        g.step_native_command7_presentation();
    } else if g.native_cmd8_presentation.is_some() {
        g.present_native_command8(Some(screen));
        g.step_native_command8_presentation();
    }
}

/// Go：`native_command3_presentation_test.go:12` `TestNativePlayerFormalCommandEntryAndPresentation`。Go 測指令
/// 3、4、5、7；Rust 另加 1、2、8，三者走同一個玩家入口與建立函式。
#[test]
fn test_native_player_formal_command_entry_and_presentation() {
    if test_support::input("FD2_ASSET_PACK", "surfaces/BG_000/resource.json").is_none() {
        return;
    }
    let Some(_dir) = f_child_env(module_path!(), "test_native_player_formal_command_entry_and_presentation", |_| {
        vec![("FD2_MUTE", "1".into())]
    }) else {
        return;
    };
    let (Some(spells), Some(resistances)) =
        (test_support::remake_asset("spells.json"), test_support::remake_asset("data/native_command_resistances.json"))
    else {
        return;
    };
    let book = battle::load_native_command_records(&spells).expect("spells");
    let resistances = battle::load_native_command_resistances(&resistances).expect("resistances");
    for command in [1, 2, 3, 4, 5, 7, 8] {
        for side in [0u8, 2] {
            let label = format!("command{command}-side{side}");
            let Some((mut g, actor, target)) = ordinary_command_test_game() else { return };
            let st = g.st.as_mut().unwrap();
            st.native_command_book = book.clone();
            st.native_command_resistances = resistances.clone();
            let a = &mut st.units[actor];
            (a.mp, a.max_mp, a.native_record_byte6) = (200, 200, side);
            if command == 4 || command == 5 {
                (a.battle_fig, a.native_record_byte8) = (14, 14);
                (a.native_identity, a.has_native_identity) = (14, true);
            }
            (st.units[target].hp, st.units[target].max_hp) = (900, 900);
            let (tx, ty) = (st.units[target].x, st.units[target].y);
            (g.sel, g.cur_x, g.cur_y) = (Some(actor), tx, ty);
            g.begin_native_command_target_selection(command).unwrap_or_else(|err| panic!("{label}: {err}"));
            g.confirm();
            assert!(native_damage_command_active(&g), "{label}: formal command did not enter presentation: {}", g.msg);
            let mut screen = Picture::new_image(640, 400);
            for _ in 0..5000 {
                if !native_damage_command_active(&g) {
                    break;
                }
                present_and_step_native_damage_command(&mut g, &mut screen);
            }
            let a = &g.st.as_ref().unwrap().units[actor];
            assert!(
                !native_damage_command_active(&g)
                    && a.mp == 200 - book[command as usize].mp_cost
                    && a.acted
                    && !g.native_command0_targeting
                    && g.sel.is_none()
                    && g.load_err.is_empty(),
                "{label}: formal command did not complete and restore cursor: MP{} acted{} error{} msg{}",
                a.mp,
                a.acted,
                g.load_err,
                g.msg
            );
        }
    }
}

/// Go：`native_command6_presentation_test.go:163` `TestPlayerNativeCommand6CursorCentersThroughConfirm`：游標格的
/// 佔用者不成為傷害目標；MP、HP、亂數終值與 Go 相同。
#[test]
fn test_player_native_command6_cursor_centers_through_confirm() {
    if test_support::input("FD2_ASSET_PACK", "animations/FDOTHER_032/animation.json").is_none() {
        return;
    }
    let Some(_dir) = f_child_env(module_path!(), "test_player_native_command6_cursor_centers_through_confirm", |_| {
        vec![("FD2_MUTE", "1".into())]
    }) else {
        return;
    };
    let (Some(spells), Some(scene), Some(flash)) = (
        test_support::remake_asset("spells.json"),
        test_support::remake_asset("data/native_command_scene.json"),
        test_support::remake_asset("data/native_command_palette_flash.json"),
    ) else {
        return;
    };
    let book = battle::load_native_command_records(&spells).expect("spells");
    for side in [0u8, 1] {
        for empty in [false, true] {
            let label = format!("side{side}/empty{empty}");
            let (mut assets, mut field, mut state) = complete_native_map_frame_fixture();
            (field.w, field.h, field.tiles) = (30, 30, vec![0; 900]);
            (state.w, state.h) = (30, 30);
            (state.native_tile_blit_modes, state.native_composition_event_bytes) = (vec![0; 900], vec![0; 900]);
            let mut actor = state.units[0].clone();
            (actor.camp, actor.on_field, actor.x, actor.y) = (battle::OWN, true, 20, 22);
            (actor.hp, actor.max_hp, actor.mp, actor.max_mp) = (331, 331, 1101, 1101);
            actor.native_record_byte6 = side;
            (actor.native_record_byte8, actor.has_native_record_byte8) = (0, true);
            (actor.inventory_slots, actor.native_inventory_flags) = (vec![0; 8].into(), vec![0; 8].into());
            actor.native_command_mask = [0x40, 0, 0, 0, 0];
            let mut center = actor.clone();
            (center.x, center.y, center.hp, center.max_hp) = (22, 20, 877, 877);
            let mut target = battle::Unit::default();
            (target.camp, target.x, target.y, target.hp, target.max_hp, target.class_id, target.on_field) =
                (battle::ENEMY, 22, 18, 83, 83, 4, true);
            (target.battle_fig, target.has_battle_fig, target.has_native_record_byte5) = (0, true, true);
            (target.native_record_byte6, target.has_native_record_byte6) = (1, true);
            (target.native_record_byte8, target.has_native_record_byte8) = (1, true);
            (target.native_record_race, target.has_native_record_race) = (1, true);
            (target.native_record_class, target.has_native_record_class) = (4, true);
            (target.inventory_slots, target.native_inventory_flags) = (vec![0; 8].into(), vec![0; 8].into());
            state.units = vec![actor, center, target];
            state.native_command_book = book.clone();
            state.native_command_resistances = BTreeMap::from([(4, 4)]);
            while assets.luts.len() <= 15 {
                assets.luts.push((0..=255).collect());
            }
            let mut g = Game {
                m: Some(field),
                st: Some(state),
                sel: Some(0),
                cur_x: 22,
                cur_y: 20,
                native_command0_targeting: true,
                native_command_target_id: 6,
                native_map_assets: Some(assets),
                native_ui_palette: load_native_ui_palette(),
                native_command_scene: Some(battle::load_native_command_scene_table(&scene).expect("scene")),
                native_command_palette_flash: Some(
                    battle::load_native_command_palette_flash_table(&flash).expect("flash"),
                ),
                native_rng_state: 3473,
                ..Default::default()
            };
            g.locale_id = String::from("zh-Hant");
            g.locale_catalog = load_official_locale("zh-Hant").expect("locale");
            g.locale_entities = load_official_locale_entities("zh-Hant").expect("entities");
            if empty {
                (g.cur_x, g.cur_y) = (21, 19);
            }
            let candidates = g.native_command_target_units_for(6);
            assert!(matches!(&candidates, Ok(list) if list.is_empty()), "{label}: initial candidates={candidates:?}");
            g.materialize_native_command_target_field(&book[6]).expect("materialize");
            g.confirm();
            let snap = |g: &Game| {
                let units = &g.st.as_ref().unwrap().units;
                (units[0].mp, units[0].acted, units[2].hp, units[1].hp, g.native_rng_state)
            };
            assert_eq!(snap(&g), (1101, false, 83, 877, 3473), "{label}: confirm crossed prebuild boundary");
            let job = g.native_cmd6_presentation.as_ref().unwrap_or_else(|| panic!("{label}: no command6: {}", g.msg));
            let results = &job.plan.as_ref().unwrap().results;
            assert!(
                results.len() == 1 && results[0].target == Some(2),
                "{label}: cursor occupant became damage target"
            );
            g.step_native_command6_presentation();
            assert_eq!(snap(&g), (1101, false, 83, 877, 3473), "{label}: undrawn frame published state");
            let mut screen = Picture::new_image(640, 400);
            for _ in 0..256 {
                if g.native_cmd6_presentation.is_none() {
                    break;
                }
                g.present_native_command6(Some(&mut screen));
                g.step_native_command6_presentation();
            }
            assert!(g.native_cmd6_presentation.is_none(), "{label}: command6 did not finish: {}", g.load_err);
            assert_eq!(snap(&g), (1071, true, 4, 877, 33552), "{label}: transaction err={}", g.load_err);
            let st = g.st.as_ref().unwrap();
            assert!(!g.native_command0_targeting && g.sel.is_none() && st.native_map_range_mode == 1, "{label}: modal");
        }
    }
}

fn record(id: i64, selection_mode: i64, mp_cost: i64, target_code: i64) -> battle::NativeCommandRecord {
    battle::NativeCommandRecord { id, selection_mode, mp_cost, target_code, ..Default::default() }
}

/// Go：`native_action_overlay_test.go:755` `TestPlayerNativeCommand13RunsCursorTransactionThroughConfirm`
#[test]
fn test_player_native_command13_runs_cursor_transaction_through_confirm() {
    let (mut assets, mut field, mut state) = complete_native_map_frame_fixture();
    for lut in &mut assets.luts {
        for (index, value) in lut.iter_mut().enumerate() {
            *value = index as u8;
        }
    }
    let mut book: Vec<battle::NativeCommandRecord> = (0..36).map(|id| record(id, 0, 0, 0)).collect();
    book[13] = battle::NativeCommandRecord { damage: 70, ..record(13, 1, 3, 1) };
    {
        let actor = &mut state.units[0];
        (actor.camp, actor.on_field, actor.hp, actor.max_hp, actor.mp) = (battle::OWN, true, 40, 100, 5);
        (actor.native_record_word42, actor.has_native_record_word42) = (100, true);
    }
    state.native_command_book = book;
    state.native_composition_event_bytes = vec![0; (state.w * state.h) as usize];
    (state.native_tile_blit_modes[0], field.native_tile_blit_modes[0]) = (0, 0);
    state.materialize_native_map_range_mode(3);
    let dac = assets.palette_dac.clone();
    let mut g = Game {
        native_map_assets: Some(assets),
        native_map_dac: dac,
        m: Some(field),
        st: Some(state),
        sel: Some(0),
        native_command0_targeting: true,
        native_command_target_id: 13,
        rng: Some(battle::GoMathRand::new(2)),
        ..Default::default()
    };
    let actor = |g: &Game| {
        let unit = &g.st.as_ref().unwrap().units[0];
        (unit.mp, unit.hp, unit.acted)
    };
    g.confirm();
    assert!(g.native_heal_presentation.is_none() && actor(&g) == (5, 40, false), "missing indexed baseline");
    g.compose_native_map_frame_at_with_palette_cycle(VIRTUAL_CLOCK_ORIGIN, false, true).expect("compose");
    let digits = std::mem::take(&mut g.native_map_assets.as_mut().unwrap().command_heal_digits);
    g.confirm();
    assert!(g.native_heal_presentation.is_none() && actor(&g) == (5, 40, false), "missing FDOTHER#5");
    g.native_map_assets.as_mut().unwrap().command_heal_digits = digits;
    g.confirm();
    assert!(g.native_heal_presentation.is_some(), "command13 did not start: {}", g.msg);
    assert_eq!(actor(&g), (5, 40, false), "command13 did not stop at presentation boundary");
    for _ in 0..256 {
        let Some(job) = g.native_heal_presentation.as_mut() else { break };
        if [
            NATIVE_COMMAND_HEAL_FRAMES,
            NATIVE_COMMAND_HEAL_EFFECT_FRAMES,
            NATIVE_COMMAND_HEAL_MASK_FRAMES,
            NATIVE_COMMAND_HEAL_DIGIT_FRAMES,
        ]
        .contains(&job.phase)
        {
            job.drawn = true;
        }
        g.step_native_command_heal_presentation();
    }
    assert!(g.native_heal_presentation.is_none(), "command13 presentation did not complete");
    assert_eq!(actor(&g), (2, 100, true), "player command13 transaction");
    let st = g.st.as_ref().unwrap();
    assert!(!g.native_command0_targeting && g.sel.is_none() && st.native_map_range_mode == 1, "cleanup");
    assert_eq!(g.msg, "原始指令 13：回復 60");
}

/// Go：`native_action_overlay_test.go:827` `TestPlayerNativeCommand17WaitsForEightPalettePhasesBeforeTransaction`。
/// Go 中途 `t.Setenv("FD2_MUTE", "1")`；Rust 改設 `shot_path`，`os_mute_or_shot` 對兩者同樣成立。
#[test]
fn test_player_native_command17_waits_for_eight_palette_phases_before_transaction() {
    let Some(flash) = test_support::remake_asset("data/native_command_palette_flash.json") else { return };
    let (assets, mut field, mut state) = complete_native_map_frame_fixture();
    let mut book: Vec<battle::NativeCommandRecord> =
        (0..battle::NATIVE_COMMAND_RECORD_COUNT).map(|id| record(id, 0, 0, 0)).collect();
    book[17] = record(17, 1, 99, 1);
    book[18] = record(18, 1, 4, 1);
    {
        let actor = &mut state.units[0];
        (actor.camp, actor.on_field) = (battle::OWN, true);
        (actor.hp, actor.max_hp, actor.mp, actor.ap) = (40, 100, 10, 100);
        (actor.lv, actor.native_record_class, actor.has_native_record_class) = (2, 9, true);
        actor.native_transient = [0; 6];
    }
    state.native_command_book = book;
    state.native_composition_event_bytes = vec![0; (state.w * state.h) as usize];
    (state.native_tile_blit_modes[0], field.native_tile_blit_modes[0]) = (0, 0);
    state.materialize_native_map_range_mode(3);
    let dac = assets.palette_dac.clone();
    let mut g = Game {
        native_map_assets: Some(assets),
        native_map_dac: dac,
        m: Some(field),
        st: Some(state),
        sel: Some(0),
        native_command0_targeting: true,
        native_command_target_id: 17,
        native_command_palette_flash: Some(battle::load_native_command_palette_flash_table(&flash).expect("flash")),
        ..Default::default()
    };
    let actor = |g: &Game| {
        let unit = &g.st.as_ref().unwrap().units[0];
        (unit.mp, unit.ap, unit.acted)
    };
    g.confirm();
    assert!(g.native_modifier_presentation.is_none() && actor(&g) == (10, 100, false), "missing framebuffer");
    g.compose_native_map_frame_at_with_palette_cycle(VIRTUAL_CLOCK_ORIGIN, false, true).expect("compose");
    if !os_mute_or_shot(Some(&g)) {
        g.confirm();
        assert!(g.native_modifier_presentation.is_none() && actor(&g) == (10, 100, false), "missing sample");
    }
    g.shot_path = String::from("command17-test");
    g.confirm();
    assert!(g.native_modifier_presentation.is_some(), "command17 did not start: {}", g.msg);
    assert_eq!(actor(&g), (10, 100, false), "command17 did not stop at palette boundary");
    assert_eq!(g.native_modifier_presentation.as_ref().unwrap().palettes.len(), 8);
    let mut screen = Picture::new_image(640, 400);
    for phase in 0..8 {
        let job = g.native_modifier_presentation.as_ref().expect("job");
        assert!(job.phase == phase && actor(&g) == (10, 100, false), "phase {phase} mutated early");
        let entry = job.palettes[phase as usize][0];
        let black = (entry.r, entry.g, entry.b) == (0, 0, 0);
        assert_eq!(black, phase % 2 == 1, "phase {phase} entry={entry:?}");
        assert!(g.present_native_command_modifier(Some(&mut screen)), "phase {phase} did not draw");
        g.step_native_command_modifier_presentation();
    }
    let unit = &g.st.as_ref().unwrap().units[0];
    assert!(
        g.native_modifier_presentation.is_none()
            && (unit.mp, unit.ap, unit.native_transient[0], unit.acted) == (6, 116, 2, true),
        "command17 transaction mp={} ap={} transient={:?} acted={} msg={}",
        unit.mp,
        unit.ap,
        unit.native_transient,
        unit.acted,
        g.msg
    );
}

/// Go 指令 20～27 測試共用的 map0 場景（`native_ai_command20_22_presentation_test.go:102`..）：只留第 4、5 筆單位，
/// `(0,0)`、`(1,0)` 兩格，雙方 +5／+6 都設為 0。
fn command2022_test_game(seed: i64, book: Vec<battle::NativeCommandRecord>) -> Option<Game> {
    let flash = test_support::remake_asset("data/native_command_palette_flash.json")?;
    let mut g = Game {
        rng: Some(battle::GoMathRand::new(seed)),
        native_ui_palette: load_native_ui_palette(),
        shot_path: String::from("command2022-test"),
        virtual_now: VIRTUAL_CLOCK_ORIGIN,
        ..Default::default()
    };
    g.load_map("assets/maps/map0").unwrap();
    if g.native_map_assets.is_none() {
        crate::test_support::skip("原版地圖素材（FD2_FLAME2_DIR）");
        return None;
    }
    g.reset_battle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json");
    g.native_command_palette_flash = Some(battle::load_native_command_palette_flash_table(&flash).expect("flash"));
    let st = g.st.as_mut().unwrap();
    assert!(g.load_err.is_empty() && st.units.len() >= 6, "fixture err={} units={}", g.load_err, st.units.len());
    let (mut actor, mut target) = (st.units[4].clone(), st.units[5].clone());
    assert!(actor.set_map_placement(0, 0, 3) && target.set_map_placement(1, 0, 3));
    actor.materialize_native_map_presentation().unwrap();
    target.materialize_native_map_presentation().unwrap();
    (actor.camp, actor.hp, actor.max_hp, actor.mp, actor.acted) = (battle::OWN, 100, 100, 10, false);
    (target.camp, target.hp, target.max_hp) = (battle::OWN, 50, 100);
    for unit in [&mut actor, &mut target] {
        (unit.native_record_byte5, unit.native_record_byte6) = (0, 0);
        (unit.has_native_record_byte5, unit.has_native_record_byte6) = (true, true);
    }
    st.units = vec![actor, target];
    st.native_command_book = book;
    st.native_composition_event_bytes = vec![0; (st.w * st.h) as usize];
    st.materialize_native_map_view_state(battle::NativeMapViewState {
        camera_x: 0,
        camera_y: 0,
        cursor_x: 1,
        cursor_y: 0,
        visible_cursor_x: 1,
        visible_cursor_y: 0,
    })
    .unwrap();
    assert!(st.materialize_native_map_hud_state(1, 1, 1) && st.materialize_native_map_range_mode(1));
    Some(g)
}

fn run_native_command2022_presentation(g: &mut Game, release_hold: bool) {
    let mut screen = Picture::new_image(640, 400);
    for _ in 0..512 {
        if g.native_ai_command_modifier.is_none() {
            break;
        }
        g.present_native_ai_command_modifier(Some(&mut screen));
        g.step_native_ai_command_modifier_presentation();
        if let Some(job) = g.native_ai_command_modifier.as_mut().filter(|job| release_hold && job.holding) {
            job.hold = 0;
        }
    }
}

/// Go：`native_ai_command20_22_presentation_test.go:167` `TestPlayerNativeCommand20UsesPaletteThenOriginalIndexedTail`
#[test]
fn test_player_native_command20_uses_palette_then_original_indexed_tail() {
    let mut book: Vec<battle::NativeCommandRecord> =
        (0..battle::NATIVE_COMMAND_RECORD_COUNT).map(|id| record(id, 0, 0, 0)).collect();
    book[10] = battle::NativeCommandRecord { damage: 10, ..record(10, 0, 0, 0) };
    book[20] = record(20, 1, 2, 1);
    let Some(mut g) = command2022_test_game(20, book) else { return };
    g.st.as_mut().unwrap().units[1].native_transient = [0, 0, 0, 3, 0, 0];
    g.compose_native_map_frame().unwrap();
    g.native_rng_state = 0;
    let continued = std::sync::Arc::new(std::sync::atomic::AtomicUsize::new(0));
    let count = continued.clone();
    let then = Box::new(move |_: &mut Game, _: &battle::NativeAICommand2022Plan| {
        count.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
    });
    g.start_native_command2022_presentation(0, Some(1), 20, None, then).expect("start");
    let snap = |g: &Game| {
        let units = &g.st.as_ref().unwrap().units;
        (units[1].hp, units[1].native_transient[3], units[0].mp, units[0].acted)
    };
    assert_eq!(snap(&g), (50, 3, 10, false), "player transaction crossed first Draw boundary");
    assert!(g.native_ai_command_modifier.as_ref().unwrap().frames.len() >= 8, "player palette phases unavailable");
    run_native_command2022_presentation(&mut g, true);
    let (hp, transient, mp, acted) = snap(&g);
    assert!(
        hp > 50 && transient == 0 && mp == 8 && acted && continued.load(std::sync::atomic::Ordering::SeqCst) == 1,
        "player owner incomplete hp={hp} transient={transient} mp={mp} acted={acted} err={}",
        g.load_err
    );
}

/// Go：`native_ai_command20_22_presentation_test.go:251` `TestPlayerNativeCommand25UsesOriginalTailWithoutEmptyNumericHold`
#[test]
fn test_player_native_command25_uses_original_tail_without_empty_numeric_hold() {
    let mut book: Vec<battle::NativeCommandRecord> =
        (0..battle::NATIVE_COMMAND_RECORD_COUNT).map(|id| record(id, 0, 0, 0)).collect();
    book[25] = record(25, 1, 2, 1);
    let Some(mut g) = command2022_test_game(25, book) else { return };
    for unit in &mut g.st.as_mut().unwrap().units {
        unit.native_record_byte5 = 0x80;
    }
    g.compose_native_map_frame().unwrap();
    let continued = std::sync::Arc::new(std::sync::atomic::AtomicUsize::new(0));
    let count = continued.clone();
    let then = Box::new(move |_: &mut Game, _: &battle::NativeAICommand2022Plan| {
        count.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
    });
    g.start_native_command2022_presentation(0, Some(1), 25, None, then).expect("start");
    let job = g.native_ai_command_modifier.as_ref().expect("command25 presentation job unavailable");
    let byte5 = |g: &Game| {
        let units = &g.st.as_ref().unwrap().units;
        (units[0].native_record_byte5, units[1].native_record_byte5, units[0].mp, units[0].acted)
    };
    assert!(
        job.skip_hold && job.publish_at == job.end_at,
        "skip={} publish={} end={}",
        job.skip_hold,
        job.publish_at,
        job.end_at
    );
    assert_eq!(byte5(&g), (0x80, 0x80, 10, false), "command25 pre-publish");
    run_native_command2022_presentation(&mut g, false);
    assert!(g.native_ai_command_modifier.is_none(), "command25 did not finish: {}", g.load_err);
    assert_eq!(byte5(&g), (0x80, 0, 8, true), "command25 owner incomplete");
    assert_eq!(continued.load(std::sync::atomic::Ordering::SeqCst), 1);
}

/// Rust 新增（Go 沒有玩家指令 31 的測試）：指令 31 沒有演出，確認即走 0x276EC 結算，與 Go `main.go:8160` 的共用
/// 收尾相同。數值沿用 `native_command24_test.rs` 的 31 號倍率測試（AP×18／10－DP，MP 40→14）。
#[test]
fn test_player_native_command31_settles_on_cursor_confirm() {
    let mut actor = battle::Unit::default();
    (actor.camp, actor.on_field, actor.x, actor.y, actor.ap, actor.mp, actor.hp) =
        (battle::OWN, true, 0, 0, 100, 40, 50);
    let mut target = battle::Unit::default();
    (target.camp, target.on_field, target.x, target.y, target.dp, target.hp) = (battle::ENEMY, true, 1, 0, 20, 300);
    // 0x115B6 的游標閘門要求名冊每筆都有原版 +5。
    (actor.has_native_record_byte5, target.has_native_record_byte5) = (true, true);
    let mut book: Vec<battle::NativeCommandRecord> =
        (0..battle::NATIVE_COMMAND_RECORD_COUNT).map(|id| record(id, 0, 0, 0)).collect();
    book[31] = record(31, 1, 26, 0);
    let mut st = battle::State::default();
    (st.w, st.h, st.units, st.native_command_book) = (2, 1, vec![actor, target], book);
    (st.native_composition_event_bytes, st.native_tile_blit_modes) = (vec![0; 2], vec![0xff, 0]);
    st.materialize_native_map_range_mode(3);
    let mut g = Game {
        st: Some(st),
        sel: Some(0),
        cur_x: 1,
        cur_y: 0,
        native_command0_targeting: true,
        native_command_target_id: 31,
        rng: Some(battle::GoMathRand::new(2)),
        ..Default::default()
    };
    g.confirm();
    let units = &g.st.as_ref().unwrap().units;
    let damage = 300 - units[1].hp;
    assert!(units[0].mp == 14 && units[0].acted && damage > 0, "command31 settle mp={} msg={}", units[0].mp, g.msg);
    assert_eq!(g.msg, format!("原始指令 31：傷害 {damage}"));
    let st = g.st.as_ref().unwrap();
    assert!(!g.native_command0_targeting && g.sel.is_none() && st.native_map_range_mode == 1, "command31 cleanup");
    assert_eq!(st.native_tile_blit_modes, vec![0xff, 0xff]);
}
