//! 移植自 `remake/internal/battle/native_command_damage_test.go`。

use super::*;
use crate::test_support::{remake_asset, temp_dir};

/// Go：`TestResolveNativeCommandDamageMatchesRecoveredFormula`
#[test]
fn resolve_native_command_damage_matches_recovered_formula() {
    let first = fdother::native_rng_step(7);
    let second = fdother::native_rng_step(first);
    let base = 50 * 7 / 10;
    let want = base * 9 / 10 + (second % 100) as i64 * base / 1000;
    let got = resolve_native_command_damage(50, 100, 7, 7);
    assert!(
        matches!(got, Ok((d, state)) if d.hit && d.damage == want && state == second),
        "got {got:?}, want hit damage={want} state={second:#x}"
    );
}

/// Go：`TestNativeCommandDamageMissDoesNotConsumeVarianceOrMutate`
#[test]
fn native_command_damage_miss_does_not_consume_variance_or_mutate() {
    let mut target = Unit { hp: 40, max_hp: 40, ..Default::default() };
    let got = apply_native_command_damage(Some(&mut target), 50, 0, 10, 1);
    assert!(
        matches!(got, Ok((d, state)) if !d.hit && d.damage == 0 && state == fdother::native_rng_step(1))
            && target.hp == 40,
        "got={got:?} hp={}",
        target.hp
    );
}

/// Go：`TestApplyNativeCommandDamageClampsHP`
#[test]
fn apply_native_command_damage_clamps_hp() {
    let mut target = Unit { hp: 1, max_hp: 100, ..Default::default() };
    let got = apply_native_command_damage(Some(&mut target), 100, 100, 10, 2);
    assert!(
        matches!(got, Ok((d, _)) if d.hit && (90..=99).contains(&d.damage)) && target.hp == 0,
        "got={got:?} hp={}",
        target.hp
    );
}

/// Go：`TestLoadNativeCommandResistancesRejectsInvalidEditableData`
#[test]
fn load_native_command_resistances_rejects_invalid_editable_data() {
    let path = temp_dir("resist").join("resist.json");
    let path_s = path.to_string_lossy().into_owned();
    std::fs::write(&path, br#"[{"cls":5,"resist_raw":7}]"#).unwrap();
    let table = load_native_command_resistances(&path_s);
    assert!(matches!(&table, Ok(t) if t.get(&5) == Some(&7)), "table={table:?}");
    std::fs::write(&path, br#"[{"cls":5,"resist_raw":11}]"#).unwrap();
    assert!(load_native_command_resistances(&path_s).is_err(), "invalid raw multiplier must fail closed");
}

/// Go：`TestLoadNativeCommandResistancesRuntimeExport`
#[test]
fn load_native_command_resistances_runtime_export() {
    let Some(path) = remake_asset("data/native_command_resistances.json") else { return };
    let table = load_native_command_resistances(&path).unwrap();
    let get = |k: i64| table.get(&k).copied().unwrap_or(0);
    assert!(
        table.len() == 26 && get(5) == 7 && get(26) == 4,
        "unexpected recovered table: classes={} mage={} unknown={}",
        table.len(),
        get(5),
        get(26)
    );
}
