//! Go 來源：`remake/internal/campaign/class_change_table.go`（179 行）。

use super::*;

/// ClassChangeTable is the editable projection of the native 0x31793 and
/// 0x4e48d tables. Maps are keyed by the byte value used by the EXE.
///
/// Go：`class_change_table.go:13`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ClassChangeTable {
    pub current: BTreeMap<i64, ClassChangeCurrent>,
    pub targets: BTreeMap<i64, ClassChangeTarget>,
}

/// Go：`class_change_table.go:18`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct ClassChangeCurrent {
    /// Go 標籤：`json:"portrait"`
    #[serde(rename = "portrait")]
    pub portrait: i64,
    /// Go 標籤：`json:"default_target"`
    #[serde(rename = "default_target")]
    pub default_target: i64,
    /// Go 標籤：`json:"item_id"`
    #[serde(rename = "item_id")]
    pub item_id: i64,
    /// Go 標籤：`json:"optional_target,omitempty"`
    #[serde(rename = "optional_target", skip_serializing_if = "battle::go_is_zero")]
    pub optional_target: Option<i64>,
    /// Go 標籤：`json:"special_item,omitempty"`
    #[serde(rename = "special_item", skip_serializing_if = "battle::go_is_zero")]
    pub special_item: i64,
    /// Go 標籤：`json:"special_target,omitempty"`
    #[serde(rename = "special_target", skip_serializing_if = "battle::go_is_zero")]
    pub special_target: Option<i64>,
}

/// Go：`class_change_table.go:27`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct ClassChangeTarget {
    /// Go 標籤：`json:"portrait"`
    pub portrait: i64,
    /// Go 標籤：`json:"class_id"`
    #[serde(rename = "class_id")]
    pub class_id: i64,
    /// Go 標籤：`json:"mobility_increment"`
    #[serde(rename = "mobility_increment")]
    pub mobility_increment: i64,
}

/// ClassChangeBranch is the single target selected by native 0x31793 for one
/// church candidate. Despite the historical name, the native UI does not show
/// these as player-selectable branches.
///
/// Go：`class_change_table.go:36`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ClassChangeBranch {
    pub branch: String,
    pub portrait: i64,
    pub class_id: i64,
    pub mobility_increment: i64,
    pub required_item_id: i64,
    pub inventory_index: i64,
}

/// Go：`class_change_table.go:45`，`func LoadClassChangeTable(path string) (ClassChangeTable, error)`
pub fn load_class_change_table(path: &str) -> Result<ClassChangeTable> {
    /// Go 的匿名 wire 結構（`class_change_table.go:48`）。
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Wire {
        #[serde(rename = "current_portraits", deserialize_with = "battle::null_as_default")]
        current: Vec<ClassChangeCurrent>,
        #[serde(rename = "target_portraits", deserialize_with = "battle::null_as_default")]
        targets: Vec<ClassChangeTarget>,
    }
    let raw = battle::read_file(path)?;
    let wire: Wire = battle::decode_json(&raw)?;
    if wire.current.len() != 0x12 || wire.targets.len() != 0x22 {
        return Err(campaign_err!(
            "class change table lengths current={} targets={}",
            wire.current.len(),
            wire.targets.len()
        ));
    }
    let mut t = ClassChangeTable::default();
    for (i, row) in wire.current.into_iter().enumerate() {
        let i = i as i64;
        if row.portrait != i || row.default_target != i + 0x20 {
            return Err(campaign_err!("class change current row {i} is not indexed"));
        }
        if row.item_id < 0 || row.item_id > 0xff {
            return Err(campaign_err!("class change current item row {i} out of byte range"));
        }
        if let Some(opt) = row.optional_target
            && (row.item_id == 0xff || !(0..=0xff).contains(&opt))
        {
            return Err(campaign_err!("class change optional branch row {i} is invalid"));
        }
        if let Some(special) = row.special_target
            && (!(0..=0xff).contains(&special) || !(0..=0xff).contains(&row.special_item))
        {
            return Err(campaign_err!("class change special branch row {i} is invalid"));
        }
        t.current.insert(i, row);
    }
    for (i, row) in wire.targets.into_iter().enumerate() {
        let i = i as i64;
        if row.portrait != i + 0x20 {
            return Err(campaign_err!("class change target row {} is not indexed", i + 0x20));
        }
        if !(0..=0xff).contains(&row.class_id) || !(0..=0xff).contains(&row.mobility_increment) {
            return Err(campaign_err!("class change target row {} has non-byte class metadata", i + 0x20));
        }
        t.targets.insert(row.portrait, row);
    }
    Ok(t)
}

/// Go：`class_change_table.go:88`，`func inventoryIndex(u *battle.Unit, itemID int) int`
pub(crate) fn inventory_index(u: Option<&Unit>, item_id: i64) -> i64 {
    let Some(u) = u else {
        return -1;
    };
    if item_id < 0 || item_id == 0xff {
        return -1;
    }
    u.inventory.iter().position(|&id| id == item_id).map_or(-1, |i| i as i64)
}

/// NativeClassChangeTarget mirrors 0x31793's overwrite order. Every eligible
/// unit receives exactly one target: default first, the table-item target
/// replaces it when the item is present, and portrait 9's item-0x5a target
/// replaces both. The original UI then asks the player to choose a character
/// and confirm; it does not expose a target-branch selector.
///
/// Go：`class_change_table.go:105`，`func NativeClassChangeTarget(u *battle.Unit, t ClassChangeTable) (ClassChangeBranch, bool)`
pub fn native_class_change_target(u: Option<&Unit>, t: &ClassChangeTable) -> (ClassChangeBranch, bool) {
    let Some(unit) = u else {
        return (ClassChangeBranch::default(), false);
    };
    let Some(row) = t.current.get(&unit.portrait) else {
        return (ClassChangeBranch::default(), false);
    };
    let resolve = |branch: &str, portrait: i64, item_id: i64, item_index: i64| -> Option<ClassChangeBranch> {
        let target = t.targets.get(&portrait)?;
        Some(ClassChangeBranch {
            branch: branch.to_string(),
            portrait: target.portrait,
            class_id: target.class_id,
            mobility_increment: target.mobility_increment,
            required_item_id: item_id,
            inventory_index: item_index,
        })
    };
    let Some(mut selected) = resolve("default", row.default_target, -1, -1) else {
        return (ClassChangeBranch::default(), false);
    };
    if let Some(opt) = row.optional_target {
        let idx = inventory_index(u, row.item_id);
        if idx >= 0 {
            match resolve("optional", opt, row.item_id, idx) {
                Some(branch) => selected = branch,
                None => return (ClassChangeBranch::default(), false),
            }
        }
    }
    if let Some(special) = row.special_target {
        let idx = inventory_index(u, row.special_item);
        if idx >= 0 {
            match resolve("special", special, row.special_item, idx) {
                Some(branch) => selected = branch,
                None => return (ClassChangeBranch::default(), false),
            }
        }
    }
    (selected, true)
}

/// LoadClassChangeGrowth reads docs/data/exe_tables/growth.json. Rows 32..67
/// are the promoted rows and map linearly to target portraits 0x20..0x43;
/// the implemented target table uses the first 34 of them (0x20..0x41).
///
/// Go：`class_change_table.go:148`，`func LoadClassChangeGrowth(path string) (map[int]ClassChangeGrowth, error)`
pub fn load_class_change_growth(path: &str) -> Result<BTreeMap<i64, ClassChangeGrowth>> {
    /// Go 的匿名列結構（`class_change_table.go:151`）。`AP` 等沒有標籤，Go 以欄位名不分大小寫比對；
    /// 檔案實際用小寫鍵，Rust 認小寫與 Go 欄位名原文兩種寫法。
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Row {
        idx: i64,
        #[serde(alias = "AP")]
        ap: [i64; 2],
        #[serde(alias = "DP")]
        dp: [i64; 2],
        #[serde(alias = "DX")]
        dx: [i64; 2],
        #[serde(alias = "HP")]
        hp: [i64; 2],
        #[serde(alias = "MP")]
        mp: [i64; 2],
    }
    let raw = battle::read_file(path)?;
    let rows: Vec<Row> = battle::decode_json(&raw)?;
    let mut out = BTreeMap::new();
    for row in rows {
        if row.idx < 32 || row.idx >= 68 {
            continue;
        }
        for r in [row.ap, row.dp, row.dx, row.hp, row.mp] {
            if r[0] < 0 || r[1] < r[0] {
                return Err(campaign_err!("class change growth: invalid range at idx {}", row.idx));
            }
        }
        let key = 0x20 + row.idx - 32;
        if out.contains_key(&key) {
            return Err(campaign_err!("class change growth: duplicate idx {}", row.idx));
        }
        out.insert(key, ClassChangeGrowth { ap: row.ap, dp: row.dp, dx: row.dx, hp: row.hp, mp: row.mp });
    }
    if out.len() != 36 {
        return Err(campaign_err!("class change growth rows={}, want 36", out.len()));
    }
    Ok(out)
}

#[cfg(test)]
pub(crate) mod class_change_table_test;
