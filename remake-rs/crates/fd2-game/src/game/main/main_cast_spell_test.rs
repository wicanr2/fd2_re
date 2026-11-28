//! `confirm` 施法目標分支（Go `main.go:7484`..`7543`）的重製端測試。Go 沒有對應測試；本檔只核對
//! 該分支的閘門與交易次序，不代表原版法術演出。

use super::*;
use crate::game::save::save_test::attach_official_locale;

fn unit(f: impl FnOnce(&mut battle::Unit)) -> battle::Unit {
    let mut u = battle::Unit::default();
    f(&mut u);
    u
}

/// 我方施法者 (0,0)、受傷隊友 (1,0)、敵人 (2,0)；選好單體回復，等待確認目標。
fn cast_game(rng: bool, sealed: bool) -> Game {
    let caster = unit(|u| {
        (u.name, u.camp, u.x, u.y, u.on_field) = ("索爾".into(), battle::OWN, 0, 0, true);
        (u.hp, u.max_hp, u.mp, u.max_mp, u.sealed) = (40, 40, 10, 10, sealed);
    });
    let ally = unit(|u| {
        (u.camp, u.x, u.y, u.on_field, u.hp, u.max_hp) = (battle::OWN, 1, 0, true, 10, 30);
    });
    let enemy = unit(|u| {
        (u.camp, u.x, u.y, u.on_field, u.hp, u.max_hp) = (battle::ENEMY, 2, 0, true, 20, 20);
    });
    let mut st = battle::State::default();
    (st.w, st.h, st.units) = (3, 1, vec![caster, ally, enemy]);
    let spell =
        battle::Spell { id: 13, name: "回復".into(), dmg: 70, dist: 3, mp: 5, target: 1, ..Default::default() };
    let mut g = Game {
        st: Some(st),
        sel: Some(0),
        moved: true,
        cast_sp: Some(spell),
        rng: rng.then(|| battle::GoMathRand::new(7)),
        ..Default::default()
    };
    attach_official_locale(&mut g);
    g
}

fn confirm_at(g: &mut Game, x: i64, y: i64) {
    (g.cur_x, g.cur_y) = (x, y);
    g.confirm();
}

#[test]
fn cast_confirm_rejects_out_of_range_and_wrong_camp_then_heals_ally() {
    let mut g = cast_game(true, false);
    // 施法者自己的格子距離 0，不在 InCastRange；敵人格對回復法術不是合法目標。
    for (x, y) in [(0, 0), (2, 0)] {
        confirm_at(&mut g, x, y);
        let st = g.st.as_ref().unwrap();
        assert!(
            g.cast_sp.is_some() && g.sel == Some(0) && st.units[0].mp == 10 && !st.units[0].acted,
            "({x},{y}) consumed the spell: mp={} acted={} err={:?}",
            st.units[0].mp,
            st.units[0].acted,
            g.load_err
        );
    }
    confirm_at(&mut g, 1, 0);
    let st = g.st.as_ref().unwrap();
    let (caster, ally) = (&st.units[0], &st.units[1]);
    assert!(
        g.load_err.is_empty() && caster.mp == 5 && caster.acted && ally.hp > 10,
        "heal was not settled: err={:?} caster={caster:?} ally={ally:?}",
        g.load_err
    );
    assert!(g.cast_sp.is_none() && g.sel.is_none() && !g.moved && g.atk.is_none());
    assert!(g.msg.contains("索爾") && g.msg.contains("回復"), "msg={:?}", g.msg);
}

#[test]
fn cast_confirm_blocked_or_without_rng_keeps_selection_and_mp() {
    let mut g = cast_game(true, true);
    confirm_at(&mut g, 1, 0);
    let st = g.st.as_ref().unwrap();
    assert!(g.cast_sp.is_some() && st.units[0].mp == 10 && !st.units[0].acted && st.units[1].hp == 10);
    assert!(g.load_err.is_empty() && !g.msg.is_empty(), "blocked msg={:?} err={:?}", g.msg, g.load_err);

    let mut g = cast_game(false, false);
    confirm_at(&mut g, 1, 0);
    let st = g.st.as_ref().unwrap();
    assert!(g.cast_sp.is_some() && st.units[0].mp == 10 && !st.units[0].acted && st.units[1].hp == 10);
    assert!(!g.load_err.is_empty(), "missing rng was not fail-closed");
}
