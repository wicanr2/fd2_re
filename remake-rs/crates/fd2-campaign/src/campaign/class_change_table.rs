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
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ClassChangeCurrent {
    /// Go 標籤：`json:"portrait"`
    pub portrait: i64,
    /// Go 標籤：`json:"default_target"`
    pub default_target: i64,
    /// Go 標籤：`json:"item_id"`
    pub item_id: i64,
    /// Go 標籤：`json:"optional_target,omitempty"`
    pub optional_target: Option<i64>,
    /// Go 標籤：`json:"special_item,omitempty"`
    pub special_item: i64,
    /// Go 標籤：`json:"special_target,omitempty"`
    pub special_target: Option<i64>,
}

/// Go：`class_change_table.go:27`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ClassChangeTarget {
    /// Go 標籤：`json:"portrait"`
    pub portrait: i64,
    /// Go 標籤：`json:"class_id"`
    pub class_id: i64,
    /// Go 標籤：`json:"mobility_increment"`
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
    todo!("campaign/class_change_table.go:LoadClassChangeTable")
}

/// Go：`class_change_table.go:88`，`func inventoryIndex(u *battle.Unit, itemID int) int`
pub(crate) fn inventory_index(u: Option<&Unit>, item_id: i64) -> i64 {
    todo!("campaign/class_change_table.go:inventoryIndex")
}

/// NativeClassChangeTarget mirrors 0x31793's overwrite order. Every eligible
/// unit receives exactly one target: default first, the table-item target
/// replaces it when the item is present, and portrait 9's item-0x5a target
/// replaces both. The original UI then asks the player to choose a character
/// and confirm; it does not expose a target-branch selector.
///
/// Go：`class_change_table.go:105`，`func NativeClassChangeTarget(u *battle.Unit, t ClassChangeTable) (ClassChangeBranch, bool)`
pub fn native_class_change_target(u: Option<&Unit>, t: &ClassChangeTable) -> (ClassChangeBranch, bool) {
    todo!("campaign/class_change_table.go:NativeClassChangeTarget")
}

/// LoadClassChangeGrowth reads docs/data/exe_tables/growth.json. Rows 32..67
/// are the promoted rows and map linearly to target portraits 0x20..0x43;
/// the implemented target table uses the first 34 of them (0x20..0x41).
///
/// Go：`class_change_table.go:148`，`func LoadClassChangeGrowth(path string) (map[int]ClassChangeGrowth, error)`
pub fn load_class_change_growth(path: &str) -> Result<BTreeMap<i64, ClassChangeGrowth>> {
    todo!("campaign/class_change_table.go:LoadClassChangeGrowth")
}
