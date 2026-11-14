//! Go 來源：`remake/internal/campaign/equipment.go`（271 行）。

use super::*;

/// ItemStats is the editable combat contribution copied from the EXE item table.
///
/// Go：`equipment.go:12`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ItemStats {
    /// Go 標籤：`json:"type"`
    pub r#type: i64,
    /// Go 標籤：`json:"ap"`
    pub ap: i64,
    /// Go 標籤：`json:"hit"`
    pub hit: i64,
    /// Go 標籤：`json:"dp"`
    pub dp: i64,
    /// Go 標籤：`json:"ev"`
    pub ev: i64,
    /// Go 標籤：`json:"mv,omitempty"`
    pub mv: i64,
    /// Go 標籤：`json:"range_min,omitempty"`
    pub min: i64,
    /// Go 標籤：`json:"range_max,omitempty"`
    pub max: i64,
}

/// Go：`equipment.go:23`，`func LoadItemStats(path string) (map[int]ItemStats, error)`
pub fn load_item_stats(path: &str) -> Result<BTreeMap<i64, ItemStats>> {
    todo!("campaign/equipment.go:LoadItemStats")
}

/// RecomputeEquipment is the editable/normalized equipment projection. It is
/// not byte-equivalent to 0x1b750 or 0x1145a: it also owns MV and attack range,
/// accepts a typed map, and skips missing rows. Raw native transactions use
/// battle.ApplyNativeRuntimeEquipmentRecalc or ApplyNativeEquipmentRecalc.
///
/// Go：`equipment.go:51`，`func RecomputeEquipment(u *battle.Unit, stats map[int]ItemStats)`
pub fn recompute_equipment(u: Option<&mut Unit>, stats: &BTreeMap<i64, ItemStats>) {
    todo!("campaign/equipment.go:RecomputeEquipment")
}

/// ApplyEquippedAttackRange sets only the normalized attack range from the
/// equipped weapon's item.json range (doc32 weapon_range.json), the same rule
/// RecomputeEquipment applies after a shop purchase. Persistent 0x50-byte
/// records carry no range field, so a roster restored from FD2.SAV (LOAD or
/// CONTINUE) would otherwise fall back to the default 1 and a spear/bow unit
/// could not attack at distance 2 as it does in the original. AP/DP/HIT/EV stay
/// as the raw record wrote them; only AtkMin/AtkMax and their base are touched.
///
/// Go：`equipment.go:92`，`func ApplyEquippedAttackRange(u *battle.Unit, stats map[int]ItemStats)`
pub fn apply_equipped_attack_range(u: Option<&mut Unit>, stats: &BTreeMap<i64, ItemStats>) {
    todo!("campaign/equipment.go:ApplyEquippedAttackRange")
}

/// RecomputeAfterClassChange is the normalized counterpart of the proven
/// 0x31602 -> 0x1b750 handoff. It preserves editable campaign fields and avoids
/// double-counting gear, but it is not evidence for the raw transient modifiers
/// or the x87 rounding performed by 0x1b750.
///
/// Go：`equipment.go:119`，`func RecomputeAfterClassChange(u *battle.Unit, stats map[int]ItemStats)`
pub fn recompute_after_class_change(u: Option<&mut Unit>, stats: &BTreeMap<i64, ItemStats>) {
    todo!("campaign/equipment.go:RecomputeAfterClassChange")
}

/// InitializeEquipmentBase converts an authored effective stat line into the
/// persistent base expected by 0x1145a by subtracting the source's equipped
/// inventory contributions once. This is a normalized compatibility path, not
/// the raw eight-cell 0x1145a transaction. Subsequent saves carry
/// EquipmentBaseSet and never repeat this conversion.
///
/// Go：`equipment.go:154`，`func InitializeEquipmentBase(u *battle.Unit, stats map[int]ItemStats)`
pub fn initialize_equipment_base(u: Option<&mut Unit>, stats: &BTreeMap<i64, ItemStats>) {
    todo!("campaign/equipment.go:InitializeEquipmentBase")
}

/// EquipItem applies the original 0x1c142 rule: weapon IDs below 0x80 replace
/// an equipped weapon, while IDs >=0x80 replace an equipped armour/accessory.
///
/// Go：`equipment.go:180`，`func EquipItem(u *battle.Unit, slot int, stats map[int]ItemStats) error`
pub fn equip_item(u: Option<&mut Unit>, slot: i64, stats: &BTreeMap<i64, ItemStats>) -> Result<()> {
    todo!("campaign/equipment.go:EquipItem")
}

/// EquipNativeCompactSlot applies 0x1c142 through the compact projection used
/// by the remake, but only after proving that projection still matches the
/// native eight raw cells consumed by 0x1b722/0x184c0. This prevents a raw slot
/// index from being passed accidentally to EquipItem when editable data
/// contains holes.
///
/// Go：`equipment.go:220`，`func EquipNativeCompactSlot( u *battle.Unit, compactSlot int, stats map[int]ItemStats, ) error`
pub fn equip_native_compact_slot(
    u: Option<&mut Unit>,
    compact_slot: i64,
    stats: &BTreeMap<i64, ItemStats>,
) -> Result<()> {
    todo!("campaign/equipment.go:EquipNativeCompactSlot")
}
