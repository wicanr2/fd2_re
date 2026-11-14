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
    /// Go 的匿名列結構（`equipment.go:29`）：`id`、嵌入的 `ItemStats` 與 `range`。嵌入的
    /// `range_min`／`range_max` 讀進來後立刻被 `range` 覆寫，這裡直接不讀。Go 讀 `[2]int` 時
    /// 元素不足補零、多的忽略，Rust 以 `Vec` 讀入後取前兩個，行為相同。
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Row {
        id: i64,
        r#type: i64,
        ap: i64,
        hit: i64,
        dp: i64,
        ev: i64,
        mv: i64,
        #[serde(deserialize_with = "battle::null_as_default")]
        range: Vec<i64>,
    }
    let raw = battle::read_file(path)?;
    let rows: Vec<Row> = battle::decode_json(&raw)?;
    let mut out = BTreeMap::new();
    for row in rows {
        if row.id < 0 || row.id > 0xff {
            return Err(campaign_err!("invalid item id {}", row.id));
        }
        let range = |i: usize| row.range.get(i).copied().unwrap_or(0);
        out.insert(
            row.id,
            ItemStats {
                r#type: row.r#type,
                ap: row.ap,
                hit: row.hit,
                dp: row.dp,
                ev: row.ev,
                mv: row.mv,
                min: range(0),
                max: range(1),
            },
        );
    }
    Ok(out)
}

/// RecomputeEquipment is the editable/normalized equipment projection. It is
/// not byte-equivalent to 0x1b750 or 0x1145a: it also owns MV and attack range,
/// accepts a typed map, and skips missing rows. Raw native transactions use
/// battle.ApplyNativeRuntimeEquipmentRecalc or ApplyNativeEquipmentRecalc.
///
/// Go：`equipment.go:51`，`func RecomputeEquipment(u *battle.Unit, stats map[int]ItemStats)`
pub fn recompute_equipment(u: Option<&mut Unit>, stats: &BTreeMap<i64, ItemStats>) {
    let Some(u) = u else {
        return;
    };
    if !u.equipment_base_set {
        (u.base_ap, u.base_dp, u.base_hit, u.base_ev, u.base_mv) = (u.ap, u.dp, u.hit, u.ev, u.mv);
        (u.base_atk_min, u.base_atk_max, u.equipment_base_set) = (u.atk_min, u.atk_max, true);
    }
    (u.ap, u.dp, u.hit, u.ev, u.mv) = (u.base_ap, u.base_dp, u.base_hit, u.base_ev, u.base_mv);
    (u.atk_min, u.atk_max) = (u.base_atk_min, u.base_atk_max);
    for i in 0..u.equipped.len() {
        if !u.equipped[i] || i >= u.inventory.len() {
            continue;
        }
        let id = u.inventory[i];
        let Some(item) = stats.get(&id) else {
            continue;
        };
        u.ap = u.ap.wrapping_add(item.ap);
        u.dp = u.dp.wrapping_add(item.dp);
        u.hit = u.hit.wrapping_add(item.hit);
        u.ev = u.ev.wrapping_add(item.ev);
        u.mv = u.mv.wrapping_add(item.mv);
        // 射程只從武器取：0x1b83d 找的是 ID < 0x80 的已裝備物品，防具與飾品在
        // item.json 也有 range [1,1]，拿來覆寫會把槍／弓的 2 格蓋回 1。
        if id < 0x80 && item.min > 0 {
            u.atk_min = item.min;
        }
        if id < 0x80 && item.max > 0 {
            u.atk_max = item.max;
        }
    }
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
    let Some(u) = u else {
        return;
    };
    (u.atk_min, u.atk_max) = (0, 0);
    for i in 0..u.equipped.len() {
        if !u.equipped[i] || i >= u.inventory.len() {
            continue;
        }
        let id = u.inventory[i];
        let Some(item) = stats.get(&id) else {
            continue;
        };
        if id < 0x80 && item.min > 0 {
            u.atk_min = item.min;
        }
        if id < 0x80 && item.max > 0 {
            u.atk_max = item.max;
        }
    }
    (u.base_atk_min, u.base_atk_max) = (u.atk_min, u.atk_max);
}

/// RecomputeAfterClassChange is the normalized counterpart of the proven
/// 0x31602 -> 0x1b750 handoff. It preserves editable campaign fields and avoids
/// double-counting gear, but it is not evidence for the raw transient modifiers
/// or the x87 rounding performed by 0x1b750.
///
/// Go：`equipment.go:119`，`func RecomputeAfterClassChange(u *battle.Unit, stats map[int]ItemStats)`
pub fn recompute_after_class_change(u: Option<&mut Unit>, stats: &BTreeMap<i64, ItemStats>) {
    let Some(u) = u else {
        return;
    };
    let (mut base_ap, mut base_dp, mut base_mv) = (u.ap, u.dp, u.mv);
    if u.equipment_base_set {
        for (i, &equipped) in u.equipped.iter().enumerate() {
            if !equipped || i >= u.inventory.len() {
                continue;
            }
            let Some(item) = stats.get(&u.inventory[i]) else {
                continue;
            };
            base_ap = base_ap.wrapping_sub(item.ap);
            base_dp = base_dp.wrapping_sub(item.dp);
            base_mv = base_mv.wrapping_sub(item.mv);
        }
    }
    // +0x3e is the raw DX/HIT/EV base; equipped item +HIT/+EV are then
    // accumulated by RecomputeEquipment.
    (u.hit, u.ev) = (u.dx, u.dx);
    (u.base_ap, u.base_dp) = (base_ap, base_dp);
    (u.base_hit, u.base_ev) = (u.dx, u.dx);
    u.base_mv = base_mv;
    (u.base_atk_min, u.base_atk_max) = (u.atk_min, u.atk_max);
    u.equipment_base_set = true;
    recompute_equipment(Some(u), stats);
}

/// InitializeEquipmentBase converts an authored effective stat line into the
/// persistent base expected by 0x1145a by subtracting the source's equipped
/// inventory contributions once. This is a normalized compatibility path, not
/// the raw eight-cell 0x1145a transaction. Subsequent saves carry
/// EquipmentBaseSet and never repeat this conversion.
///
/// Go：`equipment.go:154`，`func InitializeEquipmentBase(u *battle.Unit, stats map[int]ItemStats)`
pub fn initialize_equipment_base(u: Option<&mut Unit>, stats: &BTreeMap<i64, ItemStats>) {
    let Some(u) = u else {
        return;
    };
    if u.equipment_base_set {
        return;
    }
    (u.base_ap, u.base_dp, u.base_hit, u.base_ev, u.base_mv) = (u.ap, u.dp, u.hit, u.ev, u.mv);
    (u.base_atk_min, u.base_atk_max) = (u.atk_min, u.atk_max);
    for i in 0..u.equipped.len() {
        if !u.equipped[i] || i >= u.inventory.len() {
            continue;
        }
        let Some(item) = stats.get(&u.inventory[i]) else {
            continue;
        };
        u.base_ap = u.base_ap.wrapping_sub(item.ap);
        u.base_dp = u.base_dp.wrapping_sub(item.dp);
        u.base_hit = u.base_hit.wrapping_sub(item.hit);
        u.base_ev = u.base_ev.wrapping_sub(item.ev);
        u.base_mv = u.base_mv.wrapping_sub(item.mv);
    }
    u.equipment_base_set = true;
    recompute_equipment(Some(u), stats);
}

/// EquipItem applies the original 0x1c142 rule: weapon IDs below 0x80 replace
/// an equipped weapon, while IDs >=0x80 replace an equipped armour/accessory.
///
/// Go：`equipment.go:180`，`func EquipItem(u *battle.Unit, slot int, stats map[int]ItemStats) error`
pub fn equip_item(u: Option<&mut Unit>, slot: i64, stats: &BTreeMap<i64, ItemStats>) -> Result<()> {
    let Some(u) = u else {
        return Err(campaign_err!("invalid equipment slot"));
    };
    if slot < 0 || slot >= u.inventory.len() as i64 {
        return Err(campaign_err!("invalid equipment slot"));
    }
    let slot = slot as usize;
    match stats.get(&u.inventory[slot]) {
        Some(item) if item.r#type < 0x20 => {}
        _ => return Err(campaign_err!("item is not equipment")),
    }
    while u.equipped.len() < u.inventory.len() {
        u.equipped.push(false);
    }
    let category = u.inventory[slot] < 0x80;
    for i in 0..u.equipped.len() {
        if i != slot && u.equipped[i] && (u.inventory[i] < 0x80) == category {
            u.equipped[i] = false;
        }
    }
    u.equipped[slot] = true;
    if u.inventory_slots.len() == 8 && u.native_inventory_flags.len() == 8 {
        let mut compact = 0;
        for raw_slot in 0..u.inventory_slots.len() {
            if u.inventory_slots[raw_slot] == 0xff {
                continue;
            }
            u.native_inventory_flags[raw_slot] = 0;
            if compact < u.equipped.len() && u.equipped[compact] {
                u.native_inventory_flags[raw_slot] = 0x40;
            }
            compact += 1;
        }
    }
    recompute_equipment(Some(u), stats);
    Ok(())
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
    let Some(u) = u else {
        return Err(campaign_err!("invalid native equipment state"));
    };
    if u.inventory_slots.len() != 8
        || u.native_inventory_flags.len() != 8
        || compact_slot < 0
        || compact_slot >= u.inventory.len() as i64
    {
        return Err(campaign_err!("invalid native equipment state"));
    }
    let mut raw_for_compact: Vec<usize> = Vec::with_capacity(8);
    for (raw_slot, &flag) in u.native_inventory_flags.iter().enumerate() {
        if flag & 0x80 != 0 {
            continue;
        }
        let compact = raw_for_compact.len();
        if compact >= u.inventory.len()
            || u.inventory_slots[raw_slot] != u.inventory[compact]
            || compact >= u.equipped.len()
            || (flag & 0x40 != 0) != u.equipped[compact]
        {
            return Err(campaign_err!("native equipment projection diverges at raw slot {raw_slot}"));
        }
        raw_for_compact.push(raw_slot);
    }
    if raw_for_compact.len() != u.inventory.len() || u.equipped.len() != u.inventory.len() {
        return Err(campaign_err!("native equipment projection length diverges"));
    }

    // Go `staged := *u` 淺複製後把四個切片換成新複本；Rust clone 是深複製，結果相同。
    let mut staged = u.clone();
    equip_item(Some(&mut staged), compact_slot, stats)?;
    // EquipItem also supports normalized units and therefore rebuilds native
    // flags when an eight-cell projection is present. Restore the original
    // reserved cells first: 0x1c142 writes only occupied raw cells and must not
    // erase an ignored cell's 0x80 marker merely because its stale item byte is
    // non-0xff.
    let n = staged.native_inventory_flags.len().min(u.native_inventory_flags.len());
    staged.native_inventory_flags[..n].copy_from_slice(&u.native_inventory_flags[..n]);
    for (compact, &raw_slot) in raw_for_compact.iter().enumerate() {
        staged.native_inventory_flags[raw_slot] = 0;
        if staged.equipped[compact] {
            staged.native_inventory_flags[raw_slot] = 0x40;
        }
    }
    *u = staged;
    Ok(())
}

#[cfg(test)]
pub(crate) mod equipment_test;
