//! Go 來源：`remake/internal/battle/balance.go`（198 行）。

use super::*;

/// BalanceOptions is an optional, progress-local remake extension. Nil preserves
/// the original rules. Keys are persistent record identities, never selectors.
///
/// Go：`balance.go:7`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct BalanceOptions {
    /// Go 標籤：`json:"growth"`
    pub growth: GrowthRates,
    /// Go 標籤：`json:"characters,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "BTreeMap::is_empty")]
    pub characters: BTreeMap<i64, GrowthRates>,
    /// Go 標籤：`json:"skill_hit,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "BTreeMap::is_empty")]
    pub skill_hit: BTreeMap<i64, BTreeMap<i64, i64>>,
    /// Go 標籤：`json:"enemy_percent"`
    pub enemy_percent: i64,
}

/// Go：`balance.go:14`
///
/// Go 沒有 JSON 標籤，欄位名即 Go 欄位名（`HP`、`MP`、`AP`、`DP`、`DX`）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default, rename_all = "UPPERCASE")]
pub struct GrowthRates {
    pub hp: i64,
    pub mp: i64,
    pub ap: i64,
    pub dp: i64,
    pub dx: i64,
}

/// Go：`balance.go:16`，`func OriginalGrowthRates() GrowthRates`
pub fn original_growth_rates() -> GrowthRates {
    GrowthRates { hp: 100, mp: 100, ap: 100, dp: 100, dx: 100 }
}

/// Go：`balance.go:17`，`func NewBalanceOptions() *BalanceOptions`
pub fn new_balance_options() -> BalanceOptions {
    BalanceOptions { growth: original_growth_rates(), enemy_percent: 100, ..Default::default() }
}

impl BalanceOptions {
    /// Go：`balance.go:21`，`func (o *BalanceOptions) Validate() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn validate(&self) -> Result<()> {
        let check = |r: &GrowthRates| [r.hp, r.mp, r.ap, r.dp, r.dx].iter().all(|n| (0..=500).contains(n));
        if !check(&self.growth) || self.enemy_percent < 10 || self.enemy_percent > 500 {
            return Err(battle_err!("能力設定倍率超出範圍"));
        }
        for (id, r) in &self.characters {
            if *id < 0 || *id > 31 || !check(r) {
                return Err(battle_err!("成長設定角色或倍率無效"));
            }
        }
        // Go 以 map 迭代順序檢查；兩種錯誤同時存在時 Go 報哪一個隨順序而定，Rust 依鍵的升冪。
        for (id, rows) in &self.skill_hit {
            if *id < 0 || *id > 31 {
                return Err(battle_err!("命中設定角色無效"));
            }
            for (cmd, n) in rows {
                if !native_command_has_hit_roll(*cmd) || *n < 0 || *n > 100 {
                    return Err(battle_err!("招式沒有可調命中骰或命中率無效"));
                }
            }
        }
        Ok(())
    }

    /// Go：`balance.go:54`，`func (o *BalanceOptions) IsOriginal() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn is_original(&self) -> bool {
        self.growth == original_growth_rates()
            && self.characters.is_empty()
            && self.skill_hit.is_empty()
            && self.enemy_percent == 100
    }
}

/// Go：`balance.go:58`，`func BalanceIdentity(u *Unit) (int, bool)`
pub fn balance_identity(u: Option<&Unit>) -> (i64, bool) {
    match u {
        Some(u) if u.camp == OWN && u.has_native_identity && (0..=31).contains(&u.native_identity) => {
            (u.native_identity, true)
        }
        _ => (0, false),
    }
}

/// Go：`balance.go:65`，`func NativeCommandHasHitRoll(id int) bool`
pub fn native_command_has_hit_roll(id: i64) -> bool {
    (0..=12).contains(&id) || matches!(id, 22 | 26 | 27 | 32)
}

impl State {
    /// Go：`balance.go:69`，`func (s *State) CommandHit(actor *Unit, commandID, original int) int`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn command_hit(&self, actor: Option<UnitIdx>, command_id: i64, original: i64) -> i64 {
        let (id, ok) = balance_identity(actor.map(|i| &self.units[i]));
        let Some(balance) = self.balance.as_ref() else {
            return original;
        };
        if !ok || !native_command_has_hit_roll(command_id) {
            return original;
        }
        if let Some(n) = balance.skill_hit.get(&id).and_then(|rows| rows.get(&command_id)) {
            return *n;
        }
        original
    }

    /// Go：`balance.go:80`，`func (s *State) balanceGrowthRow(u *Unit, row GrowthRow) GrowthRow`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn balance_growth_row(&self, u: Option<UnitIdx>, row: GrowthRow) -> GrowthRow {
        let mut row = row;
        let Some(balance) = self.balance.as_ref() else {
            return row;
        };
        let (id, ok) = balance_identity(u.map(|i| &self.units[i]));
        if !ok {
            return row;
        }
        let mut r = balance.growth;
        if let Some(own) = balance.characters.get(&id) {
            r = *own;
        }
        row.balance_rates = Some(r);
        row
    }
}

/// BalanceStats stores the unscaled enemy baseline for idempotent application
/// and native current-battle sidecar persistence.
///
/// Go：`balance.go:98`
///
/// Go 沒有 JSON 標籤，欄位名即 Go 欄位名（`HP`、`MP`、`AP`、`DP`、`DX`）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default, rename_all = "UPPERCASE")]
pub struct BalanceStats {
    pub hp: i64,
    pub mp: i64,
    pub ap: i64,
    pub dp: i64,
    pub dx: i64,
}

impl Unit {
    /// Go：`balance.go:100`，`func (u *Unit) BalanceStats() BalanceStats`
    pub fn balance_stats(&self) -> BalanceStats {
        let (mut ap, mut dp) = (self.ap, self.dp);
        if self.equipment_base_set {
            (ap, dp) = (self.base_ap, self.base_dp);
        }
        BalanceStats { hp: self.max_hp, mp: self.max_mp, ap, dp, dx: self.dx }
    }

    /// SetBalanceStats preserves equipment contributions and never heals a unit.
    ///
    /// Go：`balance.go:109`，`func (u *Unit) SetBalanceStats(v BalanceStats) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_balance_stats(&mut self, v: BalanceStats) -> Result<()> {
        let r = 0..=32767;
        if v.hp < 1
            || !r.contains(&v.hp)
            || !r.contains(&v.mp)
            || !r.contains(&v.ap)
            || !r.contains(&v.dp)
            || !r.contains(&v.dx)
        {
            return Err(battle_err!("能力值超出0至32767範圍，生命上限須至少1"));
        }
        let old = self.balance_stats();
        if self.ap + v.ap - old.ap > 32767
            || self.dp + v.dp - old.dp > 32767
            || self.hit + v.dx - old.dx > 32767
            || self.ev + v.dx - old.dx > 32767
        {
            return Err(battle_err!("能力值與裝備加成合計超出32767"));
        }
        (self.max_hp, self.max_mp) = (v.hp, v.mp);
        self.ap += v.ap - old.ap;
        self.dp += v.dp - old.dp;
        if self.equipment_base_set {
            self.base_ap = v.ap;
            self.base_dp = v.dp;
            self.base_hit += v.dx - old.dx;
            self.base_ev += v.dx - old.dx;
        }
        self.hit += v.dx - old.dx;
        self.ev += v.dx - old.dx;
        self.dx = v.dx;
        if self.hp > self.max_hp {
            self.hp = self.max_hp;
        }
        if self.mp > self.max_mp {
            self.mp = self.max_mp;
        }
        if self.has_native_record_word42 {
            self.native_record_word42 = self.max_hp as u16;
        }
        if self.has_native_record_word46 {
            self.native_record_word46 = self.max_mp as u16;
        }
        Ok(())
    }
}

/// Go：`balance.go:144`，`func scaleBalance(n, p int) int`
pub(crate) fn scale_balance(n: i64, p: i64) -> i64 {
    (n.wrapping_mul(p) / 100).clamp(0, 32767)
}

impl State {
    /// Go：`balance.go:155`，`func (s *State) ApplyEnemyStrength()`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn apply_enemy_strength(&mut self) {
        let pct = self.balance.as_ref().map_or(100, |b| b.enemy_percent);
        for u in &mut self.units {
            if u.camp != ENEMY || u.native_constructor_pending {
                continue;
            }
            if pct == 100 && u.enemy_balance_base.is_none() {
                continue;
            }
            if u.enemy_balance_base.is_none() {
                u.enemy_balance_base = Some(u.balance_stats());
                u.enemy_balance_percent = 100;
            }
            if u.enemy_balance_percent == pct {
                continue;
            }
            let base = u.enemy_balance_base.unwrap_or_default();
            let mut v = BalanceStats {
                hp: scale_balance(base.hp, pct),
                mp: scale_balance(base.mp, pct),
                ap: scale_balance(base.ap, pct),
                dp: scale_balance(base.dp, pct),
                dx: scale_balance(base.dx, pct),
            };
            if v.hp < 1 {
                v.hp = 1;
            }
            let (hp, mp, old_hp, old_mp) = (u.hp, u.mp, u.max_hp, u.max_mp);
            if u.set_balance_stats(v).is_err() {
                continue;
            }
            if old_hp > 0 {
                u.hp = hp.wrapping_mul(v.hp) / old_hp;
                if hp > 0 && u.hp == 0 {
                    u.hp = 1;
                }
            }
            if old_mp > 0 {
                u.mp = mp.wrapping_mul(v.mp) / old_mp;
            }
            u.enemy_balance_percent = pct;
        }
    }
}

#[cfg(test)]
mod balance_test;
