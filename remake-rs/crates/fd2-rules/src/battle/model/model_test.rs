//! 移植自 `remake/internal/battle/model_test.go`。

use super::*;
use crate::test_support::{remake_asset, temp_dir};

fn write_units(tag: &str, json: &str) -> String {
    let path = temp_dir(tag).join("units.json");
    std::fs::write(&path, json).unwrap();
    path.to_string_lossy().into_owned()
}

/// Go：`TestLoadSerial0`——驗證序章 units.json 正確載入(M1-8 headless 回歸雛形)。
///
/// Go 讀 `remake/assets/map0_units.json`；這個檔由 `tools/` 從原版產生、不入庫（`remake/.gitignore`），
/// 所以只在掛上產生過素材的 `FD2_REMAKE_ASSETS` 時才跑得到。
#[test]
fn load_serial0() {
    let Some(path) = remake_asset("map0_units.json") else { return };
    let mut st = load(&path).unwrap();
    assert!(st.w == 24 && st.h == 24, "size = {}x{}, want 24x24", st.w, st.h);
    assert_eq!(st.units.len(), 30, "units");
    let (own, enemy) = (st.alive_count(OWN), st.alive_count(ENEMY));
    assert!(own >= 1 && enemy >= 1, "缺陣營:own={own} enemy={enemy}");
    assert_eq!(st.turn, 1, "初始回合");
    for u in &st.units {
        assert!(u.hp > 0 && u.max_hp > 0, "{} 單位 HP 異常:{}/{}", u.camp, u.hp, u.max_hp);
        assert!(u.mv > 0, "{} 單位移動力 = {}", u.camp, u.mv);
        // 註:own 不再自動塞部署格(部署格保留給 scenario spawn_party 主角隊,
        // 見 Load 內註解);units.json 的 own 沿用檔案座標,不驗部署格。
    }
    // UnitAt + Alive
    let (x, y) = (st.units[0].x, st.units[0].y);
    assert!(st.unit_at(x, y).is_some(), "UnitAt({x},{y}) = nil");
    st.units[0].hp = 0;
    assert_ne!(st.unit_at(x, y), Some(0), "陣亡單位不應被 UnitAt 回傳");
}

/// Go：`TestLoadKeepsBattleFigSeparateFromLegacyMapFig`
#[test]
fn load_keeps_battle_fig_separate_from_legacy_map_fig() {
    let path = write_units(
        "battlefig",
        r#"{"w":1,"h":1,"units":[{"camp":"own","hp":1,"mp":0,"fig":7,"battle_fig":23,"map_selector_key":2,"map_selector_slot":0,"portrait":23,"x":0,"y":0}]}"#,
    );
    let st = load(&path).unwrap();
    let got = &st.units[0];
    assert!(got.fig == 7 && got.battle_fig == 23, "selectors = map {} battle {}, want 7/23", got.fig, got.battle_fig);
    assert!(got.has_map_selector_slot && got.map_selector_slot == 0, "native map selector={}", got.map_selector_slot);
    assert!(got.has_map_selector_key && got.map_selector_key == 2, "native map key={}", got.map_selector_key);

    std::fs::write(&path, r#"{"w":1,"h":1,"units":[{"camp":"own","hp":1,"mp":0,"fig":7,"portrait":23,"x":0,"y":0}]}"#)
        .unwrap();
    let st = load(&path).unwrap();
    assert_eq!(st.units[0].battle_fig, 7, "legacy BattleFig fallback");
}

/// Go：`TestLoadPreservesNativeConstructorRawTable`
#[test]
fn load_preserves_native_constructor_raw_table() {
    let path = write_units(
        "constructor",
        r#"{"w":1,"h":1,"units":[{"camp":"own","hp":1,"mp":0,"fig":7,"portrait":68,"x":0,"y":0,"native_identity":68,"native_constructor":{"branch":"high_class","index":0,"record":[1,2,3,4,5,6,7,8,9,10]}}]}"#,
    );
    let st = load(&path).unwrap();
    let got = st.units[0].native_constructor.as_ref().expect("native constructor");
    assert!(
        got.branch == "high_class"
            && got.index == 0
            && got.record.len() == 10
            && got.record[0] == 1
            && got.record[9] == 10,
        "native constructor={got:?}"
    );
    assert!(st.units[0].has_native_identity && st.units[0].native_identity == 68, "native identity");
    let u = &st.units[0];
    assert!(
        u.has_native_record_race
            && u.native_record_race == 1
            && u.has_native_record_class
            && u.native_record_class == 2,
        "constructor raw race/class={}/{}",
        u.native_record_race,
        u.native_record_class
    );
    std::fs::write(
        &path,
        r#"{"w":1,"h":1,"units":[{"camp":"own","hp":1,"native_constructor":{"branch":"high_class","index":0,"record":[1]}}]}"#,
    )
    .unwrap();
    assert!(load(&path).is_err(), "malformed native constructor must fail closed");
}

/// Go：`TestLoadRejectsNativeIdentityOutsideByte`
#[test]
fn load_rejects_native_identity_outside_byte() {
    let path = write_units("identity", r#"{"w":1,"h":1,"units":[{"camp":"own","hp":1,"native_identity":256}]}"#);
    assert!(load(&path).is_err(), "native identity outside byte must fail closed");
}

/// Go：`TestLoadKeepsScriptedRawByte8SeparateFromIdentity`
#[test]
fn load_keeps_scripted_raw_byte8_separate_from_identity() {
    let path = write_units(
        "byte8",
        r#"{"w":1,"h":1,"units":[{"camp":"enemy","hp":1,"mp":0,"fig":7,"x":0,"y":0,"native_record_byte8":96,"native_record_byte3d":7,"native_record_death_effect":[2,52,18],"native_source_byte3":8,"native_source_byte20":9,"native_source_byte25":10,"native_position_record":{"x_word":257,"y_word":514,"raw_key":96}}]}"#,
    );
    let st = load(&path).unwrap();
    let unit = &st.units[0];
    assert!(
        unit.has_native_record_byte8
            && unit.native_record_byte8 == 96
            && !unit.has_native_identity
            && unit.has_native_record_byte3_d
            && unit.native_record_byte3_d == 7
            && unit.has_native_record_death_effect
            && unit.native_record_death_effect == [2, 52, 18]
            && unit.native_fdfield_source_byte3 == 8
            && unit.native_fdfield_source_byte20 == 9
            && unit.native_fdfield_source_byte25 == 10
            && unit.has_native_position_record
            && unit.native_position_record == NativePositionRecord { x_word: 257, y_word: 514, raw_key: 96 },
        "scripted raw +8 was misclassified: {unit:?}"
    );
}

/// Go：`TestLoadRejectsMalformedNativeDeathEffectTriple`
#[test]
fn load_rejects_malformed_native_death_effect_triple() {
    let path =
        write_units("death", r#"{"w":1,"h":1,"units":[{"camp":"enemy","hp":1,"native_record_death_effect":[1,2]}]}"#);
    assert!(load(&path).is_err(), "two-byte native death effect was accepted");
}

fn keyed(key: i64) -> Unit {
    Unit { map_selector_key: key, has_map_selector_key: true, ..Default::default() }
}

/// Go：`TestMaterializeNativeMapSelectorSlotsRequiresExplicitKeys`
#[test]
fn materialize_native_map_selector_slots_requires_explicit_keys() {
    let mut units = vec![keyed(2), keyed(0), keyed(2)];
    let mut cache = fdicon::NativeSelectorCache::default();
    materialize_native_map_selector_slots(&mut units, Some(&mut cache)).unwrap();
    for (i, want) in [0, 1, 0].into_iter().enumerate() {
        assert!(
            units[i].has_map_selector_slot && units[i].map_selector_slot == want,
            "unit {i} native slot={}, want {want}",
            units[i].map_selector_slot
        );
    }
    let mut missing = vec![Unit { fig: 99, ..Default::default() }];
    assert!(
        materialize_native_map_selector_slots(&mut missing, Some(&mut fdicon::NativeSelectorCache::default())).is_err(),
        "missing raw key must fail rather than fall back to Fig"
    );
    assert!(!missing[0].has_map_selector_slot, "failed materialization must not mutate slots");
    let mut invalid = vec![keyed(9), keyed(0x100)];
    let mut cache = fdicon::NativeSelectorCache::default();
    assert!(
        materialize_native_map_selector_slots(&mut invalid, Some(&mut cache)).is_err(),
        "invalid raw key must fail"
    );
    assert!(cache.key_for_slot(0).is_err(), "preflight failure must not mutate cache");
}

/// Go：`TestStateNativeMapSelectorCachePreservesConstructionOrder`
#[test]
fn state_native_map_selector_cache_preserves_construction_order() {
    let mut st = State::default();
    st.append_native_map_selector_batch(vec![keyed(9), keyed(4)]).unwrap();
    st.append_native_map_selector_batch(vec![keyed(0), keyed(2), keyed(0)]).unwrap();
    for (i, want) in [0, 1, 2, 3, 2].into_iter().enumerate() {
        assert_eq!(st.units[i].map_selector_slot, want, "construction slot {i}");
    }
    assert!(
        st.append_native_map_selector_batch(vec![Unit { fig: 123, ..Default::default() }]).is_err(),
        "missing raw key must reject whole append"
    );
    assert_eq!(st.units.len(), 5, "failed append mutated unit order");
    assert!(
        st.native_map_selector_cache.as_ref().unwrap().key_for_slot(4).is_err(),
        "failed append mutated native cache"
    );
}

/// Go：`TestNativeMapSpriteKeyFailsClosedAfterLegacyFallback`
#[test]
fn native_map_sprite_key_fails_closed_after_legacy_fallback() {
    let mut st = State::default();
    st.append_native_map_selector_batch_or_legacy(vec![keyed(7)]);
    assert_eq!(st.native_map_sprite_key(Some(0)), (7, true), "native map key");
    let invalid = Unit { fig: 99, ..Default::default() };
    st.append_native_map_selector_batch_or_legacy(vec![invalid.clone()]);
    assert!(st.native_map_selector_error.is_some(), "malformed batch did not record native selector failure");
    assert!(!st.native_map_sprite_key(Some(0)).1, "prior native slot remained enabled after malformed fallback");
    // Go 以指標相等確認是同一個單位；Rust 比較內容。
    assert!(st.units.len() == 2 && st.units[1] == invalid, "legacy fallback did not preserve unit order");
}
