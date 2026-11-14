//! Go 來源：`remake/cmd/fd2/balance_save_test.go` 裡直接呼叫 F 組函式的兩個測試。
//!
//! 同檔的 `TestBalanceAuditSelectionDoesNotNormalizeOrChangeState`、
//! `TestBalanceWarningRequiresThreeSeparateClicksEveryOpening` 測的是 A 組 `player_balance.go`。

use super::*;

/// Go：`balance_save_test.go:54`
#[test]
fn balance_native_sidecar_keeps_slots_independent_and_original_bytes() {
    let dir = crate::test_support::temp_dir("balance-sidecar");
    let path = dir.join("FD2.SAV").to_string_lossy().into_owned();
    let plain = vec![0x5a; fdsave::FILE_SIZE];
    let stored = fdsave::encode(&plain).unwrap();
    std::fs::write(&path, &stored).unwrap();
    let mut g = Game::default();
    g.replace_native_save_with_balance(&path, &stored, 0).unwrap();
    if file_exists_balance(&format!("{path}.fd2-options.json")) {
        panic!("default created sidecar");
    }
    if std::fs::read(&path).unwrap() != stored {
        panic!("default changed native bytes");
    }
    let mut balance = battle::new_balance_options();
    balance.enemy_percent = 200;
    g.balance = Some(balance);
    g.replace_native_save_with_balance(&path, &stored, 0).unwrap();
    let one = load_balance_saved_state(&path, &plain, 0);
    if !one.as_ref().is_ok_and(|s| s.options.as_ref().is_some_and(|o| o.enemy_percent == 200)) {
        panic!("slot1 options absent {:?}", one.err());
    }
    let two = load_balance_saved_state(&path, &plain, 1);
    if !two.as_ref().is_ok_and(|s| s.options.is_none()) {
        panic!("slot1 leaked into slot2");
    }
    let mut changed = plain.clone();
    let (a, _) = fdsave::slot_bounds(1).unwrap();
    changed[a] = changed[a].wrapping_add(1);
    let stored2 = fdsave::encode(&changed).unwrap();
    g.balance.as_mut().unwrap().enemy_percent = 50;
    g.replace_native_save_with_balance(&path, &stored2, 1).unwrap();
    let one = load_balance_saved_state(&path, &changed, 0).unwrap_or_default();
    let two = load_balance_saved_state(&path, &changed, 1).unwrap_or_default();
    // Go 對 nil 的 Options 取欄位會 panic；這裡同樣以 unwrap 表達。
    if one.options.unwrap().enemy_percent != 200 || two.options.unwrap().enemy_percent != 50 {
        panic!("saving another slot invalidated options");
    }
    changed[a] = changed[a].wrapping_add(1);
    let absent = load_balance_saved_state(&path, &changed, 1).unwrap_or_default();
    if absent.options.is_some() {
        panic!("unrelated old save inherited options");
    }
}

/// Go：`balance_save_test.go:110`
#[test]
fn balance_native_current_restores_enemy_baseline_without_scaling_twice() {
    let dir = crate::test_support::temp_dir("balance-current");
    let path = dir.join("FD2.SAV").to_string_lossy().into_owned();
    let plain = vec![0u8; fdsave::FILE_SIZE];
    let stored = fdsave::encode(&plain).unwrap();
    std::fs::write(&path, &stored).unwrap();
    let base = battle::BalanceStats { hp: 100, mp: 20, ap: 50, dp: 40, dx: 10 };
    let enemy = |base: Option<battle::BalanceStats>, percent: i64| {
        let mut u = battle::Unit::default();
        (u.camp, u.hp, u.max_hp, u.mp, u.max_mp, u.ap, u.dp, u.dx) = (battle::ENEMY, 80, 200, 20, 40, 100, 80, 20);
        u.enemy_balance_base = base;
        u.enemy_balance_percent = percent;
        u
    };
    let u = enemy(Some(base), 200);
    let mut st = battle::State::default();
    st.units = vec![u];
    let mut balance = battle::new_balance_options();
    balance.enemy_percent = 200;
    let mut g = Game { st: Some(st), balance: Some(balance), ..Default::default() };
    g.replace_native_save_with_balance(&path, &stored, -1).unwrap();
    let saved = load_balance_saved_state(&path, &plain, -1).unwrap();
    let loaded = enemy(None, 0);
    let mut cold_st = battle::State::default();
    cold_st.units = vec![loaded];
    let mut cold = Game { st: Some(cold_st), ..Default::default() };
    cold.restore_balance_saved_state(saved);
    cold.sync_balance_state();
    let unit = |g: &Game| g.st.as_ref().unwrap().units[0].clone();
    let l = unit(&cold);
    if l.max_hp != 200 || l.hp != 80 || l.ap != 100 {
        panic!("native current strength compounded");
    }
    cold.balance = None;
    cold.sync_balance_state();
    let l = unit(&cold);
    if l.max_hp != 100 || l.hp != 40 || l.ap != 50 {
        panic!("native current baseline was not restored");
    }
    cold.restore_balance_saved_state(BalanceSavedState::default());
    if cold.balance.is_some() {
        panic!("old save inherited prior profile");
    }
}

/// F 組補充：`balanceSaveKey` 的大小與 slot 檢查（完整的 Go 黃金在 `save/golden_save_test.rs`）。
#[test]
fn balance_save_key_rejects_size_and_slot() {
    assert_eq!(balance_save_key(&[0; 5], 0).unwrap_err().message(), "能力設定存檔大小不符");
    let plain = vec![0u8; fdsave::FILE_SIZE];
    assert_eq!(balance_save_key(&plain, 4).unwrap_err().message(), "fdsave: slot 4 outside 0..3");
    assert_eq!(balance_save_key(&plain, -2).unwrap_err().message(), "fdsave: slot -2 outside 0..3");
    assert!(balance_save_key(&plain, -1).unwrap().starts_with("current:"));
    assert!(balance_save_key(&plain, 3).unwrap().starts_with("slot3:"));
}
