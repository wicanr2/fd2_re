//! Go 來源：`remake/cmd/fd2/ai_spell_consumer_test.go`（3 個測試，逐條移植，期望值不改）。

use super::*;
use crate::game::save::save_test::attach_official_locale;

fn unit(f: impl FnOnce(&mut battle::Unit)) -> battle::Unit {
    let mut u = battle::Unit::default();
    f(&mut u);
    u
}

fn state(w: i64, units: Vec<battle::Unit>, spell: battle::Spell) -> battle::State {
    let mut st = battle::State::default();
    (st.w, st.h, st.units, st.spell_book) = (w, 1, units, vec![spell]);
    st
}

fn heal_spell() -> battle::Spell {
    battle::Spell { id: 13, name: "回復".into(), dmg: 70, dist: 3, mp: 5, target: 1, ..Default::default() }
}

fn heal_caster() -> battle::Unit {
    unit(|u| {
        (u.camp, u.x, u.y, u.on_field) = (battle::ENEMY, 0, 0, true);
        (u.hp, u.max_hp, u.mp, u.max_mp, *u.spells) = (40, 40, 10, 10, vec![13]);
    })
}

#[test]
fn test_ai_step_consumes_editable_heal_spell_through_production_loop() {
    let ally = unit(|u| {
        (u.camp, u.x, u.y, u.on_field) = (battle::ENEMY, 1, 0, true);
        (u.hp, u.max_hp, u.mp, u.max_mp, u.acted) = (10, 30, 0, 0, true);
    });
    let mut g = Game {
        st: Some(state(3, vec![heal_caster(), ally], heal_spell())),
        ai_busy: true,
        rng: Some(battle::GoMathRand::new(7)),
        ..Default::default()
    };
    attach_official_locale(&mut g);

    g.ai_step();
    let st = g.st.as_ref().unwrap();
    let (caster, ally) = (&st.units[0], &st.units[1]);
    assert!(
        g.load_err.is_empty() && caster.hp == 40 && ally.hp > 10 && caster.mp == 5 && caster.acted,
        "heal spell was not consumed: err={:?} caster={caster:?} ally={ally:?}",
        g.load_err
    );
    assert!(
        st.turn == 0 && g.ai_busy,
        "first spell action advanced turn prematurely: turn={} ai={}",
        st.turn,
        g.ai_busy
    );
    g.ai_step();
    let st = g.st.as_ref().unwrap();
    assert!(
        g.load_err.is_empty() && st.turn == 1 && !g.ai_busy,
        "spell loop did not finish turn: turn={} ai={} err={:?}",
        st.turn,
        g.ai_busy,
        g.load_err
    );
}

#[test]
fn test_ai_step_consumes_editable_attack_spell_and_moves_into_range() {
    let caster = unit(|u| {
        (u.camp, u.x, u.y, u.on_field) = (battle::ENEMY, 0, 0, true);
        (u.hp, u.max_hp, u.mp, u.max_mp, u.mv, *u.spells) = (40, 40, 10, 10, 1, vec![0]);
    });
    let target = unit(|u| {
        (u.camp, u.x, u.y, u.on_field) = (battle::OWN, 2, 0, true);
        (u.hp, u.max_hp, u.ap, u.dp) = (30, 30, 1, 1);
    });
    let spell = battle::Spell {
        id: 0,
        name: "火炎".into(),
        dmg: 50,
        hit: 100,
        dist: 1,
        mp: 5,
        target: 0,
        ..Default::default()
    };
    let mut g = Game {
        m: Some(MapData { w: 3, h: 1, tile_w: 24, tile_h: 24, tiles: vec![0, 0, 0], ..Default::default() }),
        st: Some(state(3, vec![caster, target], spell)),
        ai_busy: true,
        rng: Some(battle::GoMathRand::new(11)),
        ..Default::default()
    };
    attach_official_locale(&mut g);

    g.ai_step();
    let walk_ok =
        g.walk.as_ref().is_some_and(|w| w.path.len() == 2 && w.path.last() == Some(&battle::Cell { x: 1, y: 0 }));
    assert!(
        g.load_err.is_empty() && walk_ok,
        "attack spell did not choose cast-range path: walk={:?} err={:?}",
        g.walk.as_ref().map(|w| &w.path),
        g.load_err
    );
    let mut step = 0;
    while g.walk.is_some() && step < 8 {
        g.step_battle_walk();
        step += 1;
    }
    let st = g.st.as_ref().unwrap();
    let (caster, target) = (&st.units[0], &st.units[1]);
    assert!(
        g.load_err.is_empty() && g.walk.is_none() && target.hp < 30 && caster.acted,
        "attack spell was not consumed after walk: caster={caster:?} target={target:?} err={:?}",
        g.load_err
    );
    g.ai_step();
    let st = g.st.as_ref().unwrap();
    assert!(
        g.load_err.is_empty() && st.turn == 1 && !g.ai_busy,
        "attack spell loop did not finish turn: turn={} ai={} err={:?}",
        st.turn,
        g.ai_busy,
        g.load_err
    );
}

#[test]
fn test_ai_step_stops_spell_without_rng_before_mutation() {
    let ally = unit(|u| {
        (u.camp, u.x, u.y, u.on_field) = (battle::ENEMY, 1, 0, true);
        (u.hp, u.max_hp, u.acted) = (10, 30, true);
    });
    // planner 可以建立合法法術 route，但 production executor 在缺少決定性
    // （deterministic）RNG 邊界時，必須在 CastArea 改寫 MP 或 HP 前拒絕。
    let mut g =
        Game { st: Some(state(2, vec![heal_caster(), ally], heal_spell())), ai_busy: true, ..Default::default() };
    attach_official_locale(&mut g);
    g.ai_step();
    let st = g.st.as_ref().unwrap();
    let (caster, ally) = (&st.units[0], &st.units[1]);
    assert!(
        !g.load_err.is_empty() && !g.ai_busy && !caster.acted && caster.mp == 10 && ally.hp == 10,
        "spell without RNG was not fail-closed: err={:?} ai={} acted={} mp={} allyHP={}",
        g.load_err,
        g.ai_busy,
        caster.acted,
        caster.mp,
        ally.hp
    );
}
