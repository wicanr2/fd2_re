//! Go `remake/cmd/fd2/chapter_parity_hp_lock_test.go`：`parityUpdateGame` 與修改路徑的鎖血政策。
//! 只供明示修改路徑的章重播，正式 Game 不含鎖血入口。契約：`docs/data/ida/fd2_ch27_flow_20261016.json#resample_hp_lock_policy`。

use super::*;

/// Go `chapter_parity_hp_lock_test.go:66`：`parityUpdateGame`。沒有鎖血政策時就是 `g.Update()`；
/// Rust 在 `update` 之前把虛擬時鐘推進一個邏輯幀（Ebiten 主迴圈的一次 `Update`）。
///
/// 鎖血與指令傷害演出：原版 oracle 每一步寫回鎖血，原版演出不核對中間 HP；重製端的分段扣血
/// （`apply_native_command_damage_stage`）會核對目標仍是上一段的 HP。演出進行中先不寫回，演出
/// 收尾後的下一次寫回再復原。演出把鎖血單位打到 0 時，下一次寫回因綁定失效而失敗，不會默默蓋過。
/// Go 版同一測具沒有這段，鎖血章遇到敵方法術打中鎖血單位時會同樣失敗。
pub(crate) fn parity_update_game(d: &mut Driver) -> Result<()> {
    let apply = |d: &mut Driver| -> Result<()> {
        match d.hp_policy.as_ref() {
            Some(policy) if !native_command_damage_presentation_active(&d.g) => {
                policy.apply(&mut d.g).map_err(GameError::new)
            }
            _ => Ok(()),
        }
    };
    apply(d)?;
    d.g.advance_virtual_frame(&mut d.clock);
    d.g.update()?;
    apply(d)
}

/// 有分段發布 HP 的指令演出進行中（與 `Game::update` 擋住相機同一組工作）。
fn native_command_damage_presentation_active(g: &Game) -> bool {
    g.native_heal_presentation.is_some()
        || g.native_modifier_presentation.is_some()
        || g.native_ai_command_modifier.is_some()
        || g.native_ai_item_presentation.is_some()
        || g.native_cmd0_presentation.is_some()
        || g.native_cmd1_presentation.is_some()
        || g.native_cmd2_presentation.is_some()
        || g.native_cmd3_presentation.is_some()
        || g.native_cmd5_presentation.is_some()
        || g.native_cmd6_presentation.is_some()
        || g.native_cmd7_presentation.is_some()
        || g.native_cmd8_presentation.is_some()
        || g.native_cmd9_player.is_some()
        || g.native_cmd9_ai_presentation.is_some()
        || g.native_cmd1012.is_some()
        || g.native_cmd24_presentation.is_some()
        || g.native_cmd29_presentation.is_some()
        || g.native_cmd32_presentation.is_some()
        || g.native_cmd33_presentation.is_some()
        || g.native_cmd34_presentation.is_some()
        || g.native_cmd35_presentation.is_some()
}

/// Go `chapter_parity_hp_lock_test.go:13`：`parityHPLockTarget`（計畫與驅動端寫進 actions 的綁定）。
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
pub(crate) struct ParityHPLockTarget {
    pub(crate) index: i64,
    pub(crate) camp: i64,
    pub(crate) identity: i64,
    #[serde(default)]
    pub(crate) hp: Option<i64>,
}

/// Go `chapter_parity_hp_lock_test.go:19`：`parityHPLockCell`（原版 oracle 回報的實際綁定）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
pub(crate) struct ParityHPLockCell {
    pub(crate) index: i64,
    pub(crate) camp: i64,
    pub(crate) identity: i64,
    pub(crate) hp: i64,
}

/// Go `chapter_parity_hp_lock_test.go:25`：`parityHPLockPolicy`。
#[derive(Debug, Clone, Default)]
pub(crate) struct ParityHPLockPolicy {
    pub(crate) cells: Vec<ParityHPLockCell>,
}

/// Go `chapter_parity_hp_lock_test.go:29`：`parityHPLockUnit`。沿 NativeItemPanelRecordForUnit 的既有
/// player-persistent +8 相容契約。
pub(crate) fn parity_hp_lock_unit(g: &Game, cell: ParityHPLockCell) -> std::result::Result<UnitIdx, String> {
    let Some(st) = g.st.as_ref() else {
        return Err(String::from("鎖血索引越界"));
    };
    if cell.index < 0 || cell.index as usize >= st.units.len() {
        return Err(String::from("鎖血索引越界"));
    }
    let u = &st.units[cell.index as usize];
    let (mut identity, mut has_identity) =
        (u.native_identity, u.has_native_identity && (0..=255).contains(&u.native_identity));
    if u.has_native_record_byte8 {
        (identity, has_identity) = (i64::from(u.native_record_byte8), true);
    }
    if !u.on_field
        || u.hp <= 0
        || u.hp > 32767
        || !u.has_native_record_byte5
        || u.native_record_byte5 & 1 != 0
        || !u.has_native_record_byte6
        || i64::from(u.native_record_byte6) != cell.camp
        || !has_identity
        || identity != cell.identity
    {
        return Err(format!("鎖血record{}缺少相同活躍來源", cell.index));
    }
    Ok(cell.index as UnitIdx)
}

impl ParityHPLockPolicy {
    /// Go `chapter_parity_hp_lock_test.go:51`：`apply`。先核對全部綁定再寫，不寫半套。
    pub(crate) fn apply(&self, g: &mut Game) -> std::result::Result<(), String> {
        let units =
            self.cells.iter().map(|&cell| parity_hp_lock_unit(g, cell)).collect::<std::result::Result<Vec<_>, _>>()?;
        let Some(st) = g.st.as_mut() else {
            return Ok(()); // 沒有綁定時 Go 的迴圈不碰戰場（戰前也成立）。
        };
        for (cell, index) in self.cells.iter().zip(units) {
            st.units[index].hp = cell.hp;
        }
        Ok(())
    }
}

/// Go `chapter_parity_hp_lock_test.go:126`：`parityPreservesEnemy`。清敵明示保留的索引必須正好是全部 camp 0
/// 鎖血綁定；回傳 `index` 是否在保留集合內。
pub(crate) fn parity_preserves_enemy(policy: Option<&ParityHPLockPolicy>, indices: &[i64], index: i64) -> bool {
    let locked: BTreeSet<i64> =
        policy.map(|p| p.cells.iter().filter(|c| c.camp == 0).map(|c| c.index).collect()).unwrap_or_default();
    let mut want = BTreeSet::new();
    for &i in indices {
        assert!(want.insert(i) && locked.contains(&i), "清敵保留索引缺少鎖血綁定");
    }
    assert!(want == locked, "清敵未明示保留全部敵方鎖");
    want.contains(&index)
}
