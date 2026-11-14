//! Go 來源：`remake/internal/campaign/separated_shop_assets.go`（228 行）。

use super::*;

pub(crate) const SEPARATED_FDOTHER_SIZE: i64 = 3382481;

pub(crate) const SEPARATED_FDOTHERMD5: &str = "22f56e5027edc7c766ad34ca4e5aca93";

pub(crate) const SEPARATED_FDOTHERSHA256: &str = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce";

/// Go：`separated_shop_assets.go:21`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct SeparatedShopEntry {
    /// Go 標籤：`json:"index"`
    pub index: i64,
    /// Go 標籤：`json:"role"`
    pub role: String,
    /// Go 標籤：`json:"codec"`
    pub codec: String,
    /// Go 標籤：`json:"width"`
    pub width: i64,
    /// Go 標籤：`json:"height"`
    pub height: i64,
    /// Go 標籤：`json:"frame"`
    pub frame: String,
    /// Go 標籤：`json:"mask"`
    pub mask: String,
}

/// Go：`separated_shop_assets.go:31`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct SeparatedShopSuccess {
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
    /// Go 標籤：`json:"pre_ticks"`
    pub pre_ticks: i64,
    /// Go 標籤：`json:"ticks_per_frame"`
    pub ticks_per_frame: i64,
    /// Go 標籤：`json:"post_ticks"`
    pub post_ticks: i64,
    /// Go 標籤：`json:"restore_portrait"`
    pub restore_portrait: bool,
    /// Go 標籤：`json:"frames"`
    pub frames: Vec<SeparatedShopEntry>,
}

/// Go：`separated_shop_assets.go:41`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct SeparatedShopDocument {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"kind"`
    pub kind: String,
    /// Go 標籤：`json:"asset_id"`
    pub asset_id: String,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"evidence"`
    pub evidence: String,
    /// Go 標籤：`json:"container_kind"`
    pub container_kind: String,
    /// Go 標籤：`json:"entry_count"`
    pub entry_count: i64,
    /// Go 標籤：`json:"source"`
    pub source: SeparatedTownLabelDocumentSource,
    /// Go 標籤：`json:"entries"`
    pub entries: Vec<SeparatedShopEntry>,
    /// Go 標籤：`json:"success"`
    pub success: SeparatedShopSuccess,
}

/// Go：`separated_shop_assets.go:61`，`func readSeparatedIndexed(path string, width, height int) ([]byte, error)`
pub(crate) fn read_separated_indexed(path: &str, width: i64, height: i64) -> Result<Vec<u8>> {
    todo!("campaign/separated_shop_assets.go:readSeparatedIndexed")
}

/// Go：`separated_shop_assets.go:78`，`func readSeparatedBinaryMask(path string, width, height int) ([]byte, error)`
pub(crate) fn read_separated_binary_mask(path: &str, width: i64, height: i64) -> Result<Vec<u8>> {
    todo!("campaign/separated_shop_assets.go:readSeparatedBinaryMask")
}

/// LoadSeparatedNativeShopAssets loads one caller-proven shop resource without
/// consulting FDOTHER.DAT. Production callers must preflight all three variants.
///
/// Go：`separated_shop_assets.go:103`，`func LoadSeparatedNativeShopAssets(packRoot string, resource int) (*NativeShopAssets, error)`
pub fn load_separated_native_shop_assets(pack_root: &str, resource: i64) -> Result<NativeShopAssets> {
    todo!("campaign/separated_shop_assets.go:LoadSeparatedNativeShopAssets")
}
