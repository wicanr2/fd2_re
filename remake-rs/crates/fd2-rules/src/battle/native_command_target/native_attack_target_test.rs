//! 移植自 `remake/internal/battle/native_attack_target_test.go`。

use super::*;

fn enemy_at(x: i64) -> Unit {
    Unit { camp: ENEMY, x, y: 0, hp: 10, max_hp: 10, on_field: true, ..Default::default() }
}

/// Go：`TestNativeAttackCandidatesAppliesInnerRadiusMarker`
#[test]
fn native_attack_candidates_applies_inner_radius_marker() {
    let units = vec![enemy_at(0), enemy_at(1), enemy_at(2)];
    let got = native_attack_candidates(5, 1, Cell { x: 0, y: 0 }, 3, 2, 0, &[0; 5], &units);
    assert!(matches!(&got, Ok(t) if t.len() == 1 && t[0] == 2), "inner-radius candidates={got:?}, want only slot2");
}

/// Go：`TestNativeAttackCandidatesCrossSkipsInnerRadius`
#[test]
fn native_attack_candidates_cross_skips_inner_radius() {
    let units = vec![enemy_at(0), enemy_at(1)];
    let got = native_attack_candidates(3, 1, Cell { x: 0, y: 0 }, 0x11, 99, 0, &[0; 3], &units);
    assert!(matches!(&got, Ok(t) if t.len() == 2), "cross candidates={got:?}, want both cells");
}

/// Go：`TestNativeAttackCandidatesRejectsNegativeInnerRadius`
#[test]
fn native_attack_candidates_rejects_negative_inner_radius() {
    assert!(
        native_attack_candidates(1, 1, Cell::default(), 1, -1, 0, &[0], &[]).is_err(),
        "negative inner radius unexpectedly accepted"
    );
}
