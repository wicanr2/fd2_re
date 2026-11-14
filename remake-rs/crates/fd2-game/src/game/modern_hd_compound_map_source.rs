//! Go 來源：`remake/cmd/fd2/modern_hd_compound_map_source.go`（192 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Selection callbacks become private constant writes before a scene is sealed.
/// Two different backgrounds reject destination-dependent operations.
///
/// Go：`modern_hd_compound_map_source.go:16`，`func freezeCompoundMapInput(in indexedmap.NativeFrameInput) (indexedmap.NativeFrameInput, error)`
pub(crate) fn freeze_compound_map_input(
    r#in: &indexedmap::NativeFrameInput<'_>,
) -> Result<indexedmap::NativeFrameInput<'static>> {
    todo!("modern_hd_compound_map_source.go:freezeCompoundMapInput")
}

impl ModernHDCompoundMapScene {
    /// Go：`modern_hd_compound_map_source.go:64`，`func (s *modernHDCompoundMapScene) digest() ([32]byte, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn digest(&mut self) -> Result<[u8; 32]> {
        todo!("modern_hd_compound_map_source.go:modernHDCompoundMapScene.digest")
    }

    /// Go：`modern_hd_compound_map_source.go:95`，`func (s *modernHDCompoundMapScene) seal()`
    pub(crate) fn seal(&mut self) {
        todo!("modern_hd_compound_map_source.go:modernHDCompoundMapScene.seal")
    }
}

impl Game {
    /// The post-transaction producer composes from the same stage seed. It does not
    /// advance the native stage clock while assembling private command frames.
    ///
    /// Go：`modern_hd_compound_map_source.go:135`，`func (g *Game) newModernHDCompoundMapScene(in indexedmap.NativeFrameInput, work, vga []byte, palette color.Palette, inputErr error) *modernHDCompoundMapScene`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn new_modern_hd_compound_map_scene(
        &mut self,
        r#in: &indexedmap::NativeFrameInput<'_>,
        work: &[u8],
        vga: &[u8],
        palette: &ColorPalette,
        input_err: Option<&GameError>,
    ) -> Option<ModernHDCompoundMapScene> {
        todo!("modern_hd_compound_map_source.go:newModernHDCompoundMapScene")
    }
}

impl ModernHDCompoundMapScene {
    /// Go：`modern_hd_compound_map_source.go:155`，`func (s *modernHDCompoundMapScene) validate(g *Game) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn validate(&mut self, g: &Game) -> Result<()> {
        todo!("modern_hd_compound_map_source.go:modernHDCompoundMapScene.validate")
    }
}
