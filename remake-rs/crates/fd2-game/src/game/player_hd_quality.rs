//! Go 來源：`remake/cmd/fd2/player_hd_quality.go`（117 行）。
//!
//! 骨架：Go 檔的 9 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`player_hd_quality.go:10`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct HdTextureQuality(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const HD_TEXTURE_FULL: HdTextureQuality = HdTextureQuality(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const HD_TEXTURE_BALANCED: HdTextureQuality = HdTextureQuality(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const HD_TEXTURE_LIGHT: HdTextureQuality = HdTextureQuality(2);

/// Go：`player_hd_quality.go:18`，`func parseHDTextureQuality(name string) hdTextureQuality`
pub(crate) fn parse_hd_texture_quality(name: &str) -> HdTextureQuality {
    todo!("player_hd_quality.go:parseHDTextureQuality")
}

impl HdTextureQuality {
    /// Go：`player_hd_quality.go:29`，`func (q hdTextureQuality) name() string`
    /// Go 值接收者：本體對接收者的改寫只影響副本。
    pub(crate) fn name(&self) -> String {
        todo!("player_hd_quality.go:name")
    }
}
