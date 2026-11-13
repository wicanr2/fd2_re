//! Go 來源：`remake/internal/battle/native_ch28_post_prelude.go`（136 行）。

use super::*;

/// Go：`native_ch28_post_prelude.go:8`
pub(crate) static NATIVE_CH28_POST_FRONTIERS: LazyLock<BTreeSet<i64>> =
    LazyLock::new(|| BTreeSet::from([76, 78, 80, 82, 84, 86, 87, 88, 90, 93]));

/// ValidateNativeCh28PostFrontier accepts only the materialization prefixes
/// produced by the recovered map28 chain: 20 persistent records, group8's 56
/// rows, groups4..7 in two-row order, then group1's three rows. Event72 may
/// append groups2/3 atomically between complete groups. Group9 stays pending.
/// READY: docs/data/ida/fd2_ch29_flow_20261017.json.
///
/// Go：`native_ch28_post_prelude.go:17`，`func ValidateNativeCh28PostFrontier(st *State) error`
pub fn validate_native_ch28_post_frontier(st: Option<&State>) -> Result<()> {
    let Some(st) = st else {
        return Err(battle_err!("ch28 post: battle state unavailable"));
    };
    let n = st.units.len();
    if !NATIVE_CH28_POST_FRONTIERS.contains(&(n as i64)) {
        return Err(battle_err!("ch28 post: runtime frontier {n} is not recovered"));
    }
    if n < 76 {
        return Err(battle_err!("ch28 post: base frontier unavailable"));
    }
    // Go 的 `st.Units[index] == nil` 檢查在 Rust 不存在：`units` 存單位本體。
    for index in 20..76 {
        if st.units[index].group != 8 {
            return Err(battle_err!("ch28 post: slot{index} is not the recovered group8 prefix"));
        }
    }
    let want_tail = [4, 4, 5, 5, 6, 6, 7, 7, 1, 1, 1];
    let mut tail: Vec<i64> = st.units[76..].iter().map(|u| u.group).collect();
    if let Some(offset) = tail.iter().position(|&g| g == 2 || g == 3) {
        if ![0, 2, 4, 6, 8, 11].contains(&offset) {
            return Err(battle_err!("ch28 post: event72 splits a dynamic group at slot{}", 76 + offset));
        }
        if tail.len() - offset < 6 {
            return Err(battle_err!("ch28 post: event72 append is incomplete"));
        }
        for (i, want) in [2, 2, 2, 3, 3, 3].into_iter().enumerate() {
            if tail[offset + i] != want {
                return Err(battle_err!("ch28 post: event72 append order disagrees at slot{}", 76 + offset + i));
            }
        }
        tail.drain(offset..offset + 6);
    }
    if ![0, 2, 4, 6, 8, 11].contains(&tail.len()) {
        return Err(battle_err!("ch28 post: incomplete dynamic group prefix"));
    }
    for (offset, &group) in tail.iter().enumerate() {
        if group != want_tail[offset] {
            return Err(battle_err!("ch28 post: slot{} breaks the recovered dynamic group order", 76 + offset));
        }
    }
    let pending_group9 = st.roster.iter().filter(|u| u.group == 9).count();
    if pending_group9 != 1 {
        return Err(battle_err!("ch28 post: pending group9 rows={pending_group9} want 1"));
    }
    Ok(())
}

/// ApplyNativeCh28PostRawPrelude reproduces only the proven raw writers before
/// sub_1DB65: clear word +0x40 from slot20 through the current tail, then set
/// slot20 raw +7/+8 to 0x7e. The 0x1DB65 indexed presenter is a separate gate;
/// callers must not treat this transaction as the complete postbattle beat.
///
/// Go：`native_ch28_post_prelude.go:82`，`func ApplyNativeCh28PostRawPrelude(st *State) error`
pub fn apply_native_ch28_post_raw_prelude(st: Option<&mut State>) -> Result<()> {
    validate_native_ch28_post_frontier(st.as_deref())?;
    let Some(st) = st else { unreachable!("驗證已拒收 nil 狀態") };
    for (index, unit) in st.units.iter().enumerate() {
        if !unit.has_native_map_presentation || !unit.has_native_record_byte5 || unit.hp < 0 || unit.hp > 0xffff {
            return Err(battle_err!("ch28 post: slot{index} lacks presenter raw-field provenance"));
        }
    }
    if st.has_native_runtime_unit_projection {
        if st.native_runtime_records.len() != st.units.len() {
            return Err(battle_err!("ch28 post: complete 0x50 runtime projection is inconsistent"));
        }
        for (index, unit) in st.units.iter().enumerate() {
            let record = &st.native_runtime_records[index].raw;
            if record[0] != unit.native_map_presentation.x
                || record[1] != unit.native_map_presentation.y
                || record[3] != unit.native_map_presentation.pose
                || record[5] != unit.native_record_byte5
                || u16::from_le_bytes([record[0x40], record[0x41]]) != unit.hp as u16
            {
                return Err(battle_err!("ch28 post: slot{index} raw projection disagrees with typed presenter fields"));
            }
        }
    } else if !st.native_runtime_records.is_empty() {
        return Err(battle_err!("ch28 post: unproven partial 0x50 runtime projection"));
    }

    // Go 以修改後的複本換掉每個元素（長度與順序不變）；Rust 就地修改，見 `battle` 模組文件。
    for (index, unit) in st.units.iter_mut().enumerate() {
        if index >= 20 {
            unit.hp = 0;
        }
        if index == 20 {
            unit.battle_fig = 0x7e;
            unit.has_battle_fig = true;
            unit.native_record_byte8 = 0x7e;
            unit.has_native_record_byte8 = true;
        }
    }
    if st.has_native_runtime_unit_projection {
        for record in st.native_runtime_records[20..].iter_mut() {
            record.raw[0x40..0x42].copy_from_slice(&[0, 0]);
        }
        st.native_runtime_records[20].raw[7] = 0x7e;
        st.native_runtime_records[20].raw[8] = 0x7e;
    }
    Ok(())
}

#[cfg(test)]
mod native_ch28_post_prelude_test;
