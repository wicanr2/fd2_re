//! Go 來源：`remake/internal/campaign/native_persistent_party.go`（198 行）。

use super::*;

pub(crate) const NATIVE_CHARACTER_IDENTITY_COUNT: i64 = 32;

pub(crate) const NATIVE_CHARACTER_CLASS_COUNT: i64 = 29;

pub(crate) const NATIVE_CHARACTER_EXESHA256: &str = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f";

/// NativeCatalogEntry is one editable name attached to an exact native byte
/// value. It deliberately contains no portrait or sprite aliases.
///
/// Go：`native_persistent_party.go:20`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCatalogEntry {
    /// Go 標籤：`json:"id"`
    pub id: i64,
    /// Go 標籤：`json:"name"`
    pub name: String,
}

/// Go：`native_persistent_party.go:25`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCharacterCatalogSource {
    /// Go 標籤：`json:"reference_file"`
    pub reference_file: String,
    /// Go 標籤：`json:"exe_sha256"`
    pub exesha256: String,
}

/// NativeCharacterCatalog names the two independently sourced byte domains
/// needed to present one persistent roster record.
///
/// Go：`native_persistent_party.go:32`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCharacterCatalog {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"source"`
    pub source: NativeCharacterCatalogSource,
    /// Go 標籤：`json:"identities"`
    pub identities: Vec<NativeCatalogEntry>,
    /// Go 標籤：`json:"classes"`
    pub classes: Vec<NativeCatalogEntry>,
    pub(crate) identity_names: [String; 32],
    pub(crate) class_names: [String; 29],
}

/// LoadNativeCharacterCatalog requires complete, ordered native domains. A
/// missing, duplicate, reordered, or blank row fails closed.
///
/// Go：`native_persistent_party.go:44`，`func LoadNativeCharacterCatalog(path string) (*NativeCharacterCatalog, error)`
pub fn load_native_character_catalog(path: &str) -> Result<NativeCharacterCatalog> {
    todo!("campaign/native_persistent_party.go:LoadNativeCharacterCatalog")
}

impl NativeCharacterCatalog {
    /// Go：`native_persistent_party.go:59`，`func (catalog *NativeCharacterCatalog) validate() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn validate(&mut self) -> Result<()> {
        todo!("campaign/native_persistent_party.go:NativeCharacterCatalog.validate")
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
    todo!("campaign/native_persistent_party.go:MaterializeNativePersistentPartyRecord")
}
