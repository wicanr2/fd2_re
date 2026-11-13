//! 移植自 `remake/internal/battle/native_ch28_post_prelude_test.go`。

use super::*;
use crate::test_support::put16;

fn native_ch28_post_prelude_fixture(frontier: usize) -> State {
    let units: Vec<Unit> = (0..frontier)
        .map(|index| {
            let group = if (20..76).contains(&index) {
                8
            } else if index >= 76 {
                [4, 4, 5, 5, 6, 6, 7, 7, 1, 1, 1][index - 76]
            } else {
                0
            };
            Unit {
                group,
                hp: 0x1234,
                native_map_presentation: NativeMapPresentationState {
                    x: index as u8,
                    y: 2,
                    pose: 3,
                    ..Default::default()
                },
                has_native_map_presentation: true,
                native_record_byte5: 0,
                has_native_record_byte5: true,
                ..Default::default()
            }
        })
        .collect();
    let records: Vec<NativeRuntimeRecordState> = units
        .iter()
        .map(|u| {
            let mut r = NativeRuntimeRecordState::default();
            r.raw[0] = u.native_map_presentation.x;
            r.raw[1] = u.native_map_presentation.y;
            r.raw[3] = u.native_map_presentation.pose;
            r.raw[5] = u.native_record_byte5;
            put16(&mut r.raw, 0x40, u.hp as u16);
            r
        })
        .collect();
    State {
        units,
        roster: vec![
            Unit { group: 2, ..Default::default() },
            Unit { group: 3, ..Default::default() },
            Unit { group: 9, ..Default::default() },
        ],
        native_runtime_records: records,
        has_native_runtime_unit_projection: true,
        ..Default::default()
    }
}

/// Go：`TestNativeCh28PostFrontiersFollowRecoveredAppendPrefixes`
#[test]
fn native_ch28_post_frontiers_follow_recovered_append_prefixes() {
    for frontier in [76, 78, 80, 82, 84, 87] {
        let st = native_ch28_post_prelude_fixture(frontier);
        let got = validate_native_ch28_post_frontier(Some(&st));
        assert!(got.is_ok(), "frontier {frontier}: {got:?}");
    }
    let mut invalid = native_ch28_post_prelude_fixture(82);
    invalid.units[80].group = 7;
    assert!(validate_native_ch28_post_frontier(Some(&invalid)).is_err(), "out-of-order dynamic group was accepted");
}

/// Go：`TestNativeCh28PostAdmitsAtomicCaptainAppendBetweenCompleteGroups`
#[test]
fn native_ch28_post_admits_atomic_captain_append_between_complete_groups() {
    let main = [4, 4, 5, 5, 6, 6, 7, 7, 1, 1, 1];
    let captain = [2, 2, 2, 3, 3, 3];
    let fixture = |groups: &[i64]| {
        let mut st = native_ch28_post_prelude_fixture(76);
        for &group in groups {
            let mut unit = st.units[75].clone();
            unit.group = group;
            st.units.push(unit);
            let record = st.native_runtime_records[75];
            st.native_runtime_records.push(record);
        }
        st
    };
    for length in [0, 2, 4, 6, 8, 11] {
        for insertion in [0, 2, 4, 6, 8, 11] {
            if insertion > length {
                continue;
            }
            let mut groups = main[..insertion].to_vec();
            groups.extend_from_slice(&captain);
            groups.extend_from_slice(&main[insertion..length]);
            let mut st = fixture(&groups);
            let got = apply_native_ch28_post_raw_prelude(Some(&mut st));
            assert!(got.is_ok(), "主鏈{length}／插入{insertion}：{got:?}");
            assert!(
                st.units[19].hp == 0x1234 && st.units.last().unwrap().hp == 0,
                "原生清除未保留持續隊伍或漏掉追加名冊"
            );
        }
    }
    for groups in [
        &[4, 2, 2, 2, 3, 3, 3, 4][..], // 隊長鏈不能拆開兩筆主群組。
        &[2, 2, 3, 2, 3, 3],           // 數量正確但六筆順序錯誤。
        &[2, 2, 2, 3],                 // 只有部分隊長鏈。
        &[2, 2, 2, 3, 3, 3, 2, 2],     // 不能再接第二次隊長鏈。
    ] {
        let mut st = fixture(groups);
        let (before_units, before_raw) = (st.units.clone(), st.native_runtime_records.clone());
        assert!(apply_native_ch28_post_raw_prelude(Some(&mut st)).is_err(), "錯誤增援順序通過：{groups:?}");
        assert!(st.units == before_units && st.native_runtime_records == before_raw, "拒收增援順序時改寫了狀態");
    }
}

/// Go：`TestNativeCh28PostRawPreludeClearsTailAndPatchesSlot20`
#[test]
fn native_ch28_post_raw_prelude_clears_tail_and_patches_slot20() {
    let mut st = native_ch28_post_prelude_fixture(87);
    st.native_runtime_records[19].raw[0x40] = 0xaa;
    st.units[19].hp = 0x12aa;
    apply_native_ch28_post_raw_prelude(Some(&mut st)).unwrap();
    assert_eq!(st.native_runtime_records[19].raw[0x40], 0xaa, "raw prelude cleared a record before slot20");
    for index in 20..st.native_runtime_records.len() {
        let raw = &st.native_runtime_records[index].raw;
        assert!(raw[0x40] == 0 && raw[0x41] == 0, "slot{index} raw word40 was not cleared");
    }
    let (raw20, u20) = (&st.native_runtime_records[20].raw, &st.units[20]);
    assert!(
        raw20[7] == 0x7e
            && raw20[8] == 0x7e
            && u20.hp == 0
            && st.units[19].hp == 0x12aa
            && u20.has_battle_fig
            && u20.battle_fig == 0x7e
            && u20.has_native_record_byte8
            && u20.native_record_byte8 == 0x7e,
        "slot20 raw patch record={:?} unit={u20:?}",
        &raw20[7..9]
    );
}

/// Go：`TestNativeCh28PostRawPreludeSupportsTypedNormalBattleState`
#[test]
fn native_ch28_post_raw_prelude_supports_typed_normal_battle_state() {
    let mut st = native_ch28_post_prelude_fixture(76);
    st.native_runtime_records.clear();
    st.has_native_runtime_unit_projection = false;
    apply_native_ch28_post_raw_prelude(Some(&mut st)).unwrap();
    assert!(
        st.units[19].hp == 0x1234
            && st.units[20].hp == 0
            && st.units[75].hp == 0
            && st.units[20].battle_fig == 0x7e
            && st.units[20].native_record_byte8 == 0x7e,
        "typed normal-battle transaction mismatch: slot19={:?} slot20={:?} slot75={:?}",
        st.units[19],
        st.units[20],
        st.units[75]
    );
}

/// Go：`TestNativeCh28PostRawPreludeRejectsIncompleteProjectionAtomically`
#[test]
fn native_ch28_post_raw_prelude_rejects_incomplete_projection_atomically() {
    let mut st = native_ch28_post_prelude_fixture(84);
    st.native_runtime_records.truncate(83);
    let (before_units, before_records) = (st.units.clone(), st.native_runtime_records.clone());
    assert!(apply_native_ch28_post_raw_prelude(Some(&mut st)).is_err(), "incomplete raw projection was accepted");
    assert!(
        st.units == before_units && st.native_runtime_records == before_records,
        "failed raw prelude mutated battle state"
    );
}

/// Go：`TestNativeCh28PostRawPreludeRejectsProjectionDisagreementAtomically`
#[test]
fn native_ch28_post_raw_prelude_rejects_projection_disagreement_atomically() {
    let mut st = native_ch28_post_prelude_fixture(76);
    st.native_runtime_records[8].raw[0x40] += 1;
    let (before_units, before_records) = (st.units.clone(), st.native_runtime_records.clone());
    assert!(apply_native_ch28_post_raw_prelude(Some(&mut st)).is_err(), "raw/typed disagreement was accepted");
    assert!(
        st.units == before_units && st.native_runtime_records == before_records,
        "projection disagreement mutated battle state"
    );
}
