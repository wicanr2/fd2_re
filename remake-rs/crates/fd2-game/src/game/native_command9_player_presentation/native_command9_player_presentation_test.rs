//! D 組：第 9 指令玩家演出的測試。Go 來源：`native_command9_player_presentation_test.go`；夾具在
//! `native_command1_presentation_test.rs`。

use super::*;
use crate::game::native_command1_presentation::native_command1_presentation_test::{
    ACTOR, TARGET, callback_counter, command_record, command_test_state, count, resist, unit,
};
use crate::test_support;

/// Go：`native_command9_player_presentation_test.go:11` `nativeCommand9PlayerTestPlan`
fn test_plan() -> (battle::State, battle::NativeCommandDamagePlan) {
    let state = command_test_state(command_record(9, 999, 50, 3, 0, 30), (battle::OWN, 40), (battle::ENEMY, 403));
    let plan = state.plan_native_command_damage(Some(ACTOR), Some(TARGET), 9, &resist(), 3).expect("plan");
    (state, plan)
}

fn baseline_work() -> Vec<u8> {
    vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize]
}

fn baseline_vga() -> Vec<u8> {
    vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize]
}

/// Go：`native_command9_player_presentation_test.go:28` `TestNativeCommand9PlayerPublishesOnlyAtRecoveredBoundaries`
#[test]
fn test_native_command9_player_publishes_only_at_recovered_boundaries() {
    let (state, plan) = test_plan();
    let (counter, then) = callback_counter();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd9_player = Some(NativeCommand9PlayerJob {
        actor: Some(ACTOR),
        target: Some(TARGET),
        plan: Some(plan.clone()),
        palettes: vec![ColorPalette::new(); 1],
        effect_frames: vec![Vec::new(); 1],
        result_frames: vec![Vec::new(); 1],
        baseline_work: baseline_work(),
        baseline_vga: baseline_vga(),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: target.hp,
        rng_before: plan.rng_before,
        then: Some(then),
        ..Default::default()
    });
    g.step_native_command9_player_presentation();
    assert!(
        unit(&g, ACTOR).mp == plan.mp_before && unit(&g, TARGET).hp == plan.results[0].hp_before,
        "未 Draw 即發布交易"
    );
    g.native_cmd9_player.as_mut().expect("job").drawn = true;
    g.step_native_command9_player_presentation();
    assert!(
        unit(&g, ACTOR).mp == plan.mp_before && unit(&g, TARGET).hp == plan.results[0].hp_before,
        "palette 後過早發布交易"
    );
    g.native_cmd9_player.as_mut().expect("job").drawn = true;
    g.step_native_command9_player_presentation();
    assert!(
        unit(&g, ACTOR).mp == plan.mp_after
            && unit(&g, TARGET).hp == plan.results[0].hp_after
            && !unit(&g, ACTOR).acted,
        "effect 邊界發布錯誤"
    );
    g.native_cmd9_player.as_mut().expect("job").drawn = true;
    g.step_native_command9_player_presentation();
    g.native_cmd9_player.as_mut().expect("job").hold = 0;
    g.step_native_command9_player_presentation();
    assert!(
        g.native_cmd9_player.is_none()
            && unit(&g, ACTOR).acted
            && g.native_rng_state == plan.rng_after
            && count(&counter) == 1,
        "result hold 後未完成交易"
    );
}

/// Go：`native_command9_player_presentation_test.go:62` `TestNativeCommand9PlayerRollsBackLatePublishFailure`
#[test]
fn test_native_command9_player_rolls_back_late_publish_failure() {
    let (state, plan) = test_plan();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd9_player = Some(NativeCommand9PlayerJob {
        actor: Some(ACTOR),
        target: Some(TARGET),
        plan: Some(plan.clone()),
        phase: NATIVE_COMMAND9_PLAYER_EFFECT,
        effect_frames: vec![Vec::new(); 1],
        baseline_work: baseline_work(),
        baseline_vga: baseline_vga(),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: target.hp,
        rng_before: plan.rng_before,
        drawn: true,
        ..Default::default()
    });
    g.st.as_mut().expect("st").units[TARGET].hp -= 1;
    g.step_native_command9_player_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        g.native_cmd9_player.is_none()
            && a.mp == plan.mp_before
            && !a.acted
            && t.hp == plan.results[0].hp_before
            && g.native_rng_state == plan.rng_before
            && !g.load_err.is_empty(),
        "晚期失敗未完整回復"
    );
}

/// Go：`modern_hd_player9_map_test.go:40` `player9MapTestGame`：map0 的指令測具，指令 9 的紀錄改成必中或必不中。
fn player9_test_game(hit: bool) -> Option<(Game, UnitIdx, UnitIdx)> {
    let (mut g, actor, target) =
        crate::game::native_command_target_ui::native_command_target_ui_test::ordinary_command_test_game()?;
    g.st.as_mut().unwrap().native_command_book[9] = battle::NativeCommandRecord {
        id: 9,
        damage: 999,
        hit: if hit { 100 } else { 0 },
        selection_mode: 3,
        effect_mode: 0,
        mp_cost: 30,
        target_code: 0,
    };
    Some((g, actor, target))
}

/// 獨立的逐位元組重組：不經原生合成器、`BlitAt` 或 HD 定位，只照圖層來源把非零像素寫回工作緩衝，再取 312×192。
///
/// Go：`modern_hd_player9_map_test.go:53` `independentPlayer9VGA`
fn independent_player9_vga(
    g: &Game,
    j: &NativeCommand9PlayerJob,
    layers: &[indexedmap::NativeCommandTailLayer],
) -> Vec<u8> {
    let mut work = j.baseline_work.clone();
    let base = indexedmap::native_unit_present_strip_layout_for(0, 0, 0, 0).work_offset;
    let assets = g.native_map_assets.as_ref().unwrap();
    for layer in layers {
        let entries = match layer.resource {
            6 => &assets.fdother6,
            5 => &assets.command_heal_digits,
            other => panic!("unexpected source {other}"),
        };
        let entry = &entries[layer.entry as usize];
        for y in 0..entry.height {
            for x in 0..entry.width {
                let value = entry.pixels[(y * entry.width + x) as usize];
                if value == 0 {
                    continue;
                }
                let at = base + (layer.y + y) * fdicon::NATIVE_MAP_STRIDE + layer.x + x;
                assert!(at >= 0 && (at as usize) < work.len(), "source exceeded native work");
                work[at as usize] = value;
            }
        }
    }
    let mut want = j.baseline_vga.clone();
    for y in 0..192i64 {
        for x in 0..312i64 {
            want[(y * 320 + x) as usize] = work[(base + y * fdicon::NATIVE_MAP_STRIDE + x) as usize];
        }
    }
    want
}

/// Go：`modern_hd_player9_map_test.go:90` `TestModernHDPlayer9NativeSourcesMatchFullVGA` 的原版索引色部分。
/// HD 凍結輸入（`hdInput`／`hdInputErr`）屬 R6，不比。
#[test]
fn test_native_command9_player_sources_match_full_vga() {
    if test_support::input("FD2_ASSET_PACK", "surfaces/BG_000/resource.json").is_none() {
        return;
    }
    if f_child_env(module_path!(), "test_native_command9_player_sources_match_full_vga", |_| {
        vec![("FD2_MUTE", "1".into())]
    })
    .is_none()
    {
        return;
    }
    for hit in [false, true] {
        let Some((mut g, actor, target)) = player9_test_game(hit) else { return };
        g.start_native_command9_player_presentation(actor, Some(target), Box::new(|_, _| {})).unwrap();
        let j = g.native_cmd9_player.as_ref().unwrap();
        assert!(
            j.effect_sources.len() == 27
                && j.result_sources.len() == 22
                && j.plan.as_ref().unwrap().results[0].native_command_damage.hit == hit,
            "incomplete native player9 producer"
        );
        for (i, sources) in j.effect_sources.iter().enumerate() {
            let s = sources.first();
            assert!(
                sources.len() == 1
                    && s.is_some_and(|s| s.resource == 6 && s.entry == 87 + i as i64 && s.x == 24 && s.y == -6),
                "wrong original effect origin {i} {sources:?}"
            );
            assert!(independent_player9_vga(&g, j, sources) == j.effect_frames[i], "independent effect {i} differs");
        }
        for (i, sources) in j.result_sources.iter().enumerate() {
            if !hit {
                let entries: Vec<i64> = sources.iter().map(|s| s.entry).collect();
                assert_eq!(entries, [116, 117, 118, 118], "MISS must preserve four ordered glyph writes");
            }
            assert!(independent_player9_vga(&g, j, sources) == j.result_frames[i], "independent result {i} differs");
        }
    }
}

/// Rust 補充：從 map0 指令測具開演出，正式呈現與 stepper 走完四段，命中與不中都只在效果段結束時寫 MP 與 HP，
/// 收尾續行一次並交回基準畫面。
#[test]
fn test_native_command9_player_presents_end_to_end() {
    if test_support::input("FD2_ASSET_PACK", "surfaces/BG_000/resource.json").is_none() {
        return;
    }
    if f_child_env(module_path!(), "test_native_command9_player_presents_end_to_end", |_| {
        vec![("FD2_MUTE", "1".into())]
    })
    .is_none()
    {
        return;
    }
    for hit in [false, true] {
        let Some((mut g, actor, target)) = player9_test_game(hit) else { return };
        let (mp_before, hp_before) = (unit(&g, actor).mp, unit(&g, target).hp);
        let done = std::sync::Arc::new(std::sync::Mutex::new(Vec::new()));
        let sink = done.clone();
        g.start_native_command9_player_presentation(
            actor,
            Some(target),
            Box::new(move |_, results| sink.lock().unwrap().push(results)),
        )
        .unwrap();
        let baseline = g.native_cmd9_player.as_ref().unwrap().baseline_vga.clone();
        let mut screen = Picture::new_image(640, 400);
        let mut phases = std::collections::BTreeSet::new();
        for steps in 0..400 {
            let Some(j) = g.native_cmd9_player.as_ref() else { break };
            phases.insert(j.phase.0);
            if j.phase == NATIVE_COMMAND9_PLAYER_PALETTE || j.phase == NATIVE_COMMAND9_PLAYER_EFFECT {
                assert!(
                    unit(&g, actor).mp == mp_before && unit(&g, target).hp == hp_before,
                    "published early at {steps}"
                );
            }
            assert!(g.present_native_command9_player(Some(&mut screen)), "draw failed at {steps}: {}", g.load_err);
            g.step_native_command9_player_presentation();
        }
        assert!(g.native_cmd9_player.is_none() && g.load_err.is_empty(), "presentation did not finish: {}", g.load_err);
        assert_eq!(phases.len(), 4, "phases={phases:?}");
        let results = done.lock().unwrap();
        assert!(results.len() == 1 && results[0].len() == 1, "callback count");
        assert_eq!(results[0][0].native_command_damage.hit, hit);
        assert_eq!(unit(&g, actor).mp, mp_before - 30, "MP cost");
        assert_eq!(unit(&g, target).hp < hp_before, hit, "target HP after hit={hit}");
        assert_eq!(g.native_map_vga, baseline, "baseline frame not restored");
    }
}
