//! Go 來源：`remake/cmd/fd2/native_ai_item_presentation_test.go`。

use std::sync::Arc;
use std::sync::atomic::{AtomicUsize, Ordering};

use super::*;

/// Go 兩個原版道具測試共用的第一章 map0 場景：只留兩名單位，`(0,0)` 使用者、`(1,0)` 目標。
/// 缺原版地圖素材時回傳 `None`（`FD2_REQUIRE_ORIGINAL=1` 下失敗）。
fn item_fixture(actor_camp: battle::Camp) -> Option<Game> {
    let mut g = Game {
        rng: Some(battle::GoMathRand::new(13)),
        native_ui_palette: load_native_ui_palette(),
        shot_path: String::from("item-test"),
        virtual_now: VIRTUAL_CLOCK_ORIGIN,
        ..Default::default()
    };
    g.load_map("assets/maps/map0").unwrap();
    if g.native_map_assets.is_none() {
        crate::test_support::skip("原版地圖素材（FD2_FLAME2_DIR）");
        return None;
    }
    g.reset_battle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json");
    let st = g.st.as_mut().unwrap();
    assert!(g.load_err.is_empty() && st.units.len() >= 6, "fixture err={} units={}", g.load_err, st.units.len());
    let (mut actor, mut target) = (st.units[4].clone(), st.units[5].clone());
    assert!(actor.set_map_placement(0, 0, 3) && target.set_map_placement(1, 0, 3));
    actor.materialize_native_map_presentation().unwrap();
    target.materialize_native_map_presentation().unwrap();
    let target_camp = if actor_camp == battle::ENEMY { battle::OWN } else { battle::ENEMY };
    (actor.camp, actor.hp, actor.max_hp, actor.acted) = (actor_camp, 80, 80, false);
    (target.camp, target.hp, target.max_hp) = (target_camp, 80, 80);
    for unit in [&mut actor, &mut target] {
        unit.native_record_byte5 = 0;
        unit.native_record_byte6 = u8::from(unit.camp == battle::ENEMY);
        (unit.has_native_record_byte5, unit.has_native_record_byte6) = (true, true);
        unit.inventory_slots = vec![0xff; 8].into();
        unit.native_inventory_flags = vec![0x80; 8].into();
    }
    st.units = vec![actor, target];
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
    g.native_item_effect_rows =
        battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json")).unwrap();
    g.native_command_book = battle::load_native_command_records(&asset_path("assets/spells.json")).unwrap();
    g.native_command_resistances =
        battle::load_native_command_resistances(&asset_path("assets/data/native_command_resistances.json")).unwrap();
    g.compose_native_map_frame().unwrap();
    Some(g)
}

fn item_plan(item_id: i64) -> battle::AIPlan {
    battle::AIPlan {
        u: Some(0),
        target: Some(1),
        native_action_kind: battle::NATIVE_AI_ACTION_ITEM,
        native_item_slot: 0,
        native_item_id: item_id,
        native_item_target_indices: vec![1],
        ..Default::default()
    }
}

/// 每格延遲改 1 後逐格 Draw／Step 到演出結束；`check` 在每步之後檢查交易沒有提早寫回。
fn run_item_presentation(g: &mut Game, mut check: impl FnMut(&Game, i64)) {
    for frame in &mut g.native_ai_item_presentation.as_mut().unwrap().frames {
        frame.delay = 1;
    }
    let mut screen = Picture::new_image(640, 400);
    let mut steps = 0;
    while g.native_ai_item_presentation.is_some() && steps < 128 {
        assert!(g.draw_native_ai_item_presentation(&mut screen), "draw failed at {steps}");
        g.step_native_ai_item_presentation();
        check(g, steps);
        if let Some(job) = g.native_ai_item_presentation.as_mut().filter(|job| job.holding) {
            job.hold = 0;
        }
        steps += 1;
    }
}

fn record_hp(after: &[u8], record: usize) -> i64 {
    let at = record * 80 + 0x40;
    i64::from(i16::from_le_bytes([after[at], after[at + 1]]))
}

/// Go：`native_ai_item_presentation_test.go:347`，`TestExecuteNativeAIItemType21UsesOriginalIndexed1CAC7Tail`
#[test]
fn test_execute_native_ai_item_type21_uses_original_indexed_1cac7_tail() {
    let Some(mut g) = item_fixture(battle::ENEMY) else { return };
    {
        let actor = &mut g.st.as_mut().unwrap().units[0];
        actor.inventory_slots = vec![38, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff].into();
        actor.native_inventory_flags = vec![0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80].into();
    }
    let plan = item_plan(38);
    g.native_rng_state = 7;
    let tx = g.plan_native_ai_item_restore(&plan).unwrap();
    let route = tx.damage_route.expect("type21 走指令傷害路線");
    assert!(route.command_id == 1 && route.presentation == 0x1cac7 && tx.damage.len() == 1, "route={route:?}");
    let (expected_hp, expected_rng) = (record_hp(&tx.after, 1), tx.rng_after);
    let continued = Arc::new(AtomicUsize::new(0));
    let counter = continued.clone();
    g.native_ai_action_after = Some(Box::new(move |_: &mut Game| {
        counter.fetch_add(1, Ordering::SeqCst);
    }));
    g.execute_native_ai_action(&plan).unwrap();
    {
        let job = g.native_ai_item_presentation.as_ref().expect("type21 演出要建立");
        let st = g.st.as_ref().unwrap();
        assert!(
            job.frames.len() == 38
                && job.publish_at == 16
                && st.units[1].hp == 80
                && g.native_rng_state == 7
                && st.units[0].inventory_slots[0] == 38,
            "type21 pre-publish frames={} publish={} hp={} rng={:#x}",
            job.frames.len(),
            job.publish_at,
            st.units[1].hp,
            g.native_rng_state
        );
    }
    run_item_presentation(&mut g, |g, steps| {
        let hp = g.st.as_ref().unwrap().units[1].hp;
        assert!(
            steps >= 15 || (hp == 80 && g.native_rng_state == 7),
            "type21 published before final toggle Draw at step {steps}: hp={hp} rng={:#x}",
            g.native_rng_state
        );
    });
    let st = g.st.as_ref().unwrap();
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        g.native_ai_item_presentation.is_none()
            && target.hp == expected_hp
            && g.native_rng_state == expected_rng
            && actor.inventory_slots[0] == 38
            && actor.native_inventory_flags[0] & 0x80 == 0
            && actor.acted
            && continued.load(Ordering::SeqCst) == 1,
        "AI type21 item incomplete: hp={}/{expected_hp} rng={:#x}/{expected_rng:#x} slot={} flags={:#x} acted={} continued={}",
        target.hp,
        g.native_rng_state,
        actor.inventory_slots[0],
        actor.native_inventory_flags[0],
        actor.acted,
        continued.load(Ordering::SeqCst)
    );
}

/// Go：`native_ai_item_presentation_test.go:215`，`TestPlayerDamageItemsUseOriginalIndexedTails`
#[test]
fn test_player_damage_items_use_original_indexed_tails() {
    for (name, item_id, frames, publish) in [("type24_1CD17", 79, 40, 18), ("type21_1CAC7", 38, 38, 16)] {
        player_damage_item_uses_original_indexed_tail(name, item_id, frames, publish);
    }
}

/// Go：`native_ai_item_presentation_test.go:228`，`testPlayerDamageItemUsesOriginalIndexedTail`
fn player_damage_item_uses_original_indexed_tail(name: &str, item_id: i64, want_frames: usize, want_publish: i64) {
    let Some(mut g) = item_fixture(battle::OWN) else { return };
    {
        let actor = &mut g.st.as_mut().unwrap().units[0];
        actor.inventory = vec![item_id].into();
        actor.equipped = vec![false].into();
        actor.inventory_slots = vec![item_id, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff].into();
        actor.native_inventory_flags = vec![0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80].into();
    }
    (g.sel, g.moved, g.item_open, g.native_rng_state) = (Some(0), true, true, 7);
    assert!(g.begin_native_target_item(0, item_id).unwrap(), "{name}: targeting");
    let tx = g.plan_native_ai_item_restore(&item_plan(item_id)).unwrap();
    let (expected_hp, expected_rng) = (record_hp(&tx.after, 1), tx.rng_after);
    assert!(g.apply_native_target_item(1).unwrap(), "{name}: applied");
    {
        let job = g.native_ai_item_presentation.as_ref().expect("我方傷害物品演出要建立");
        let st = g.st.as_ref().unwrap();
        let actor = &st.units[0];
        assert!(
            job.frames.len() == want_frames
                && job.publish_at == want_publish
                && st.units[1].hp == 80
                && g.native_rng_state == 7
                && !actor.acted
                && actor.native_record_byte5 & 0x80 == 0
                && g.native_item_targeting,
            "{name}: player pre-publish frames={} publish={} rng={:#x} acted={} byte5={:#x} targeting={}",
            job.frames.len(),
            job.publish_at,
            g.native_rng_state,
            actor.acted,
            actor.native_record_byte5,
            g.native_item_targeting
        );
    }
    run_item_presentation(&mut g, |g, steps| {
        let st = g.st.as_ref().unwrap();
        let hp = st.units[1].hp;
        assert!(
            steps >= want_publish - 1 || (hp == 80 && g.native_rng_state == 7),
            "{name}: player transaction published before final blend Draw at step {steps}: hp={hp} rng={:#x}",
            g.native_rng_state
        );
        if g.native_ai_item_presentation.as_ref().is_some_and(|job| job.holding) {
            let actor = &st.units[0];
            assert!(
                !actor.acted && actor.native_record_byte5 & 0x80 == 0,
                "{name}: player action completed during result hold: acted={} byte5={:#x}",
                actor.acted,
                actor.native_record_byte5
            );
        }
    });
    let st = g.st.as_ref().unwrap();
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        g.native_ai_item_presentation.is_none()
            && target.hp == expected_hp
            && g.native_rng_state == expected_rng
            && actor.inventory_slots[0] == item_id
            && actor.native_inventory_flags[0] & 0x80 == 0
            && actor.acted
            && actor.native_record_byte5 & 0x80 != 0
            && !g.native_item_targeting
            && g.sel.is_none(),
        "{name}: player damage item incomplete: hp={}/{expected_hp} rng={:#x}/{expected_rng:#x} slot={} flags={:#x} acted={} byte5={:#x} targeting={} sel={:?}",
        target.hp,
        g.native_rng_state,
        actor.inventory_slots[0],
        actor.native_inventory_flags[0],
        actor.acted,
        actor.native_record_byte5,
        g.native_item_targeting,
        g.sel
    );
}
