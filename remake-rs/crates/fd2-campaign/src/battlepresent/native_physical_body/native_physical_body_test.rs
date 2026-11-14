//! 移植自 Go `battlepresent/native_physical_body_test.go`。

use super::*;
use crate::battlepresent::test_util::{anim, fig};
use fd2_rules::battle::NativePhysicalRollResult;

fn desc(delay: i64, raw4: u8, raw5: u8) -> Frame {
    Frame { delay, raw_byte4: raw4, raw_byte5: raw5, ..Default::default() }
}

fn strike(roll: NativePhysicalRollResult) -> NativePhysicalStrike {
    NativePhysicalStrike { roll, ..Default::default() }
}

#[test]
fn test_native_physical_body_hp_and_zero_delay() {
    let attack = anim(vec![desc(0, 1, 2), desc(2, 0, 0), desc(1, 1, 3)]);
    let idle = anim(vec![desc(2, 0, 0), desc(3, 0, 0)]);
    let strikes = [
        strike(NativePhysicalRollResult { damage: 7, ..Default::default() }),
        strike(NativePhysicalRollResult { damage: 20, ..Default::default() }),
    ];
    let plan = build_native_physical_body_plan(Some(&attack), Some(&idle), 0, 12, &strikes).unwrap();
    assert!(
        plan.len() == 6
            && plan[0].hp == 9
            && plan[0].presents.is_empty()
            && plan[2].hp == 5
            && plan[3].hp == 0
            && plan[5].hp == 0,
        "HP split / zero delay / overkill: {plan:?}"
    );
    // 零 delay 的命中仍設定phase與fill，直到下個實際present才消費。
    let first = plan[1].presents[0];
    assert!(
        first.dx == 14 && first.dy == 10 && first.opaque_fill == 33 && plan[1].presents[1].opaque_fill == -1,
        "zero-delay hit lost first actual impact: {first:?}"
    );
    // 第一揮三次idle呈現後是frame1/counter1，第二揮不重置。
    assert!(
        plan[4].presents[0].idle_frame == 1
            && plan[4].presents[1].idle_frame == 1
            && plan[5].presents[0].idle_frame == 0,
        "multi-strike idle counter was reset"
    );
}

#[test]
fn test_native_physical_body_miss_and_pulse_gates() {
    let mut attack = Animation {
        header_byte1: 1,
        header_byte2: 1,
        frames: vec![desc(1, 0, 0), desc(2, 1, 4)],
        ..Default::default()
    };
    let idle = anim(vec![desc(1, 0, 0)]);
    for missed in [true, false] {
        let roll = NativePhysicalRollResult { missed, crit: true, status: true, ..Default::default() };
        let p = build_native_physical_body_plan(Some(&attack), Some(&idle), 2, 80, &[strike(roll)]).unwrap();
        assert!(
            p.len() == 1
                && p[0].frame == 1
                && p[0].hp == 80
                && p[0].presents[0].dy == 0
                && p[0].presents[0].status_pulse
                && p[0].presents[1].crit_pulse,
            "header / pulse gate: {p:?}"
        );
        if missed {
            assert!(
                p[0].cue == 0 && p[0].presents[0].dx == 0 && p[0].presents[0].opaque_fill == -1,
                "MISS received displacement or lost cue0"
            );
        } else {
            assert!(
                p[0].cue == 4 && p[0].presents[0].dx == -14 && p[0].presents[0].opaque_fill == 33,
                "nonzero raw side lost signed impact"
            );
        }
    }
    // 分母統計全部descriptors，包含header2之前；raw4=2不走pulse。
    attack.frames[0].raw_byte4 = 1;
    attack.frames[1].raw_byte4 = 2;
    let roll = NativePhysicalRollResult { damage: 9, crit: true, status: true, ..Default::default() };
    let p = build_native_physical_body_plan(Some(&attack), Some(&idle), 0, 80, &[strike(roll)]);
    assert!(
        matches!(&p, Ok(p) if p[0].hp == 76 && !p[0].presents[0].crit_pulse && !p[0].presents[0].status_pulse),
        "raw4 / complete denominator gate changed"
    );
}

#[test]
fn test_native_physical_body_idle_zero_wrap() {
    let attack = anim(vec![desc(255, 0, 0), desc(2, 0, 0)]);
    let idle = anim(vec![desc(0, 0, 0), desc(1, 0, 0)]);
    let p =
        build_native_physical_body_plan(Some(&attack), Some(&idle), 0, 30, &[NativePhysicalStrike::default()]).unwrap();
    assert!(
        p[0].presents[254].idle_frame == 0 && p[1].presents[0].idle_frame == 0 && p[1].presents[1].idle_frame == 1,
        "idle delay0 must advance after 256 presents"
    );
}

#[test]
fn test_native_physical_body_raw_layer_order_and_clipping() {
    let frame = |value: u8| fig(319, 199, 2, 1, vec![value, value], vec![255, 255], 0);
    for side in [0u8, 1, 2] {
        for bit in [0u8, 1] {
            let (mut attack, idle) = (frame(10), frame(20));
            attack.raw_byte7 = bit;
            let out = compose_native_physical_body_present(
                &[0u8; 64000],
                &attack,
                &idle,
                side,
                NativePhysicalBodyPresent { opaque_fill: -1, ..Default::default() },
            )
            .unwrap();
            let want = if (side == 0 && bit == 0) || (side != 0 && bit != 0) { 20 } else { 10 };
            assert!(out[63999] == want && out[63680] == 0, "raw side={side} bit={bit}: edge={}", out[63999]);
        }
    }
}
