//! Go 來源：`remake/internal/campaign/native_persistent_party.go`（198 行）。

use super::*;

pub(crate) const NATIVE_CHARACTER_IDENTITY_COUNT: i64 = 32;

pub(crate) const NATIVE_CHARACTER_CLASS_COUNT: i64 = 29;

pub(crate) const NATIVE_CHARACTER_EXESHA256: &str = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f";

/// NativeCatalogEntry is one editable name attached to an exact native byte
/// value. It deliberately contains no portrait or sprite aliases.
///
/// Go：`native_persistent_party.go:20`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCatalogEntry {
    /// Go 標籤：`json:"id"`
    pub id: i64,
    /// Go 標籤：`json:"name"`
    pub name: String,
}

/// Go：`native_persistent_party.go:25`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCharacterCatalogSource {
    /// Go 標籤：`json:"reference_file"`
    pub reference_file: String,
    /// Go 標籤：`json:"exe_sha256"`
    #[serde(rename = "exe_sha256")]
    pub exesha256: String,
}

/// NativeCharacterCatalog names the two independently sourced byte domains
/// needed to present one persistent roster record.
///
/// Go：`native_persistent_party.go:32`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCharacterCatalog {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"source"`
    pub source: NativeCharacterCatalogSource,
    /// Go 標籤：`json:"identities"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub identities: Vec<NativeCatalogEntry>,
    /// Go 標籤：`json:"classes"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub classes: Vec<NativeCatalogEntry>,
    #[serde(skip)]
    pub(crate) identity_names: [String; 32],
    #[serde(skip)]
    pub(crate) class_names: [String; 29],
}

/// LoadNativeCharacterCatalog requires complete, ordered native domains. A
/// missing, duplicate, reordered, or blank row fails closed.
///
/// Go：`native_persistent_party.go:44`，`func LoadNativeCharacterCatalog(path string) (*NativeCharacterCatalog, error)`
pub fn load_native_character_catalog(path: &str) -> Result<NativeCharacterCatalog> {
    let raw = battle::read_file(path).map_err(|err| CampaignError::from(err).wrap("native character catalog: "))?;
    let mut catalog: NativeCharacterCatalog =
        battle::decode_json(&raw).map_err(|err| CampaignError::from(err).wrap("native character catalog: "))?;
    catalog.validate()?;
    Ok(catalog)
}

impl NativeCharacterCatalog {
    /// Go：`native_persistent_party.go:59`，`func (catalog *NativeCharacterCatalog) validate() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn validate(&mut self) -> Result<()> {
        if self.schema_version != 1 {
            return Err(campaign_err!("native character catalog: schema version={}, want 1", self.schema_version));
        }
        if self.source.reference_file != "docs/data/fd2-reference-files.json"
            || self.source.exesha256 != NATIVE_CHARACTER_EXESHA256
        {
            return Err(campaign_err!("native character catalog: source does not match reference FD2.EXE"));
        }
        if self.identities.len() as i64 != NATIVE_CHARACTER_IDENTITY_COUNT {
            return Err(campaign_err!(
                "native character catalog: identities={}, want {}",
                self.identities.len(),
                NATIVE_CHARACTER_IDENTITY_COUNT
            ));
        }
        if self.classes.len() as i64 != NATIVE_CHARACTER_CLASS_COUNT {
            return Err(campaign_err!(
                "native character catalog: classes={}, want {}",
                self.classes.len(),
                NATIVE_CHARACTER_CLASS_COUNT
            ));
        }
        for (index, entry) in self.identities.iter().enumerate() {
            if entry.id != index as i64 || entry.name.is_empty() {
                return Err(campaign_err!(
                    "native character catalog: identity row {index} has id={} name={}",
                    entry.id,
                    battle::go_quote(&entry.name)
                ));
            }
            self.identity_names[index] = entry.name.clone();
        }
        for (index, entry) in self.classes.iter().enumerate() {
            if entry.id != index as i64 || entry.name.is_empty() {
                return Err(campaign_err!(
                    "native character catalog: class row {index} has id={} name={}",
                    entry.id,
                    battle::go_quote(&entry.name)
                ));
            }
            self.class_names[index] = entry.name.clone();
        }
        Ok(())
    }
}

/// MaterializeNativePersistentPartyRecord projects only proven record fields
/// into the existing party Unit. Raw +7 is the player-persistent key passed to
/// 0x11019, so it is preserved as MapSelectorKey; it is not inferred to be a
/// portrait, Fig, or character identity. The projection does not assign those
/// fields, position, deployment state, spells, attack range, or chapter node.
///
/// Go：`native_persistent_party.go:113`，`func MaterializeNativePersistentPartyRecord( record fdsave.PersistentRecord, catalog *NativeCharacterCatalog, ) (battle.Unit, error)`
pub fn materialize_native_persistent_party_record(
    record: save::PersistentRecord,
    catalog: &mut NativeCharacterCatalog,
) -> Result<Unit> {
    catalog.validate()?;
    let view = record.view();
    let identity = i64::from(view.raw_identity);
    let class_id = i64::from(view.class);
    if identity >= catalog.identity_names.len() as i64 {
        return Err(campaign_err!("native persistent party: identity {identity} outside catalog"));
    }
    if class_id >= catalog.class_names.len() as i64 {
        return Err(campaign_err!("native persistent party: class {class_id} outside catalog"));
    }

    // `Unit` 有 fd2-rules 私有欄位，crate 外不能用結構更新語法；從零值逐欄設定，結果與 Go 的
    // 複合字面值相同（未列出的欄位維持零值）。
    let mut unit = Unit::default();
    unit.camp = battle::OWN;
    unit.name = catalog.identity_names[identity as usize].clone();
    unit.cls_name = catalog.class_names[class_id as usize].clone();
    unit.class_id = class_id;
    unit.lv = i64::from(view.level);
    unit.hp = i64::from(view.hp);
    unit.max_hp = i64::from(view.max_hp);
    unit.mp = i64::from(view.mp);
    unit.max_mp = i64::from(view.max_mp);
    unit.ap = i64::from(view.ap);
    unit.dp = i64::from(view.dp);
    unit.hit = i64::from(view.hit);
    unit.ev = i64::from(view.ev);
    unit.mv = i64::from(view.movement);
    (unit.native_identity, unit.has_native_identity) = (identity, true);
    (unit.native_record_byte8, unit.has_native_record_byte8) = (view.raw_identity, true);
    (unit.map_selector_key, unit.has_map_selector_key) = (i64::from(view.raw_presentation_key), true);
    (unit.native_record_race, unit.has_native_record_race) = (view.race, true);
    (unit.native_record_class, unit.has_native_record_class) = (view.class, true);
    (unit.native_record_word42, unit.has_native_record_word42) = (view.max_hp as u16, true);
    (unit.native_record_word46, unit.has_native_record_word46) = (view.max_mp as u16, true);
    unit.native_record_death_effect = [record.raw[0x31], record.raw[0x32], record.raw[0x33]];
    unit.has_native_record_death_effect = true;
    unit.native_command_mask = view.command_mask;
    unit.native_transient = view.transient;
    (unit.native_record_byte5, unit.has_native_record_byte5) = (view.raw_byte5, true);
    (unit.native_record_byte6, unit.has_native_record_byte6) = (view.raw_camp, true);
    (unit.native_record_byte34, unit.has_native_record_byte34) = (view.raw_byte34, true);
    (unit.native_record_byte35, unit.has_native_record_byte35) = (view.raw_byte35, true);
    (unit.native_record_byte36, unit.has_native_record_byte36) = (view.raw_byte36, true);
    unit.base_ap = i64::from(view.base_ap);
    unit.base_dp = i64::from(view.base_dp);
    unit.base_hit = i64::from(view.dx);
    unit.base_ev = i64::from(view.dx);
    unit.base_mv = i64::from(view.movement);
    unit.equipment_base_set = true;
    unit.dx = i64::from(view.dx);
    unit.exp = f64::from(view.experience);
    unit.inventory_slots = vec![0; view.inventory.len()];
    unit.native_inventory_flags = vec![0; view.inventory.len()];
    for (index, cell) in view.inventory.iter().enumerate() {
        unit.inventory_slots[index] = i64::from(cell.item_id);
        unit.native_inventory_flags[index] = i64::from(cell.flags);
        if (cell.flags as i8) < 0 {
            continue;
        }
        unit.inventory.push(i64::from(cell.item_id));
        unit.equipped.push(cell.flags & 0x40 != 0);
    }
    Ok(unit)
}

#[cfg(test)]
mod native_persistent_party_test;
