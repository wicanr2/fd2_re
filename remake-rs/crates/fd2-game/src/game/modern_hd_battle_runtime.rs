//! Go 來源：`remake/cmd/fd2/modern_hd_battle_runtime.go`（343 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_battle_runtime.go:14`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDBattleSource {
    pub(crate) panel_text: Option<ModernBattlePanelText>,
    pub(crate) archive: String,
    pub(crate) frame: figani::Frame,
    pub(crate) resource: i64,
    pub(crate) index: i64,
    pub(crate) dx: i64,
    pub(crate) dy: i64,
    pub(crate) fill: i64,
}

/// Go：`modern_hd_battle_runtime.go:25`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDBattleSources {
    pub(crate) base: Vec<u8>,
    pub(crate) layers: Vec<ModernHDBattleSource>,
    pub(crate) base_scenery: Vec<ModernHDBattleSource>,
}

/// Go：`modern_hd_battle_runtime.go:32`，`func modernHDBattleBodySources(base []byte, attack, idle figani.Frame, side byte, present battlepresent.NativePhysicalBodyPresent, attackResource, attackFrame, idleResource, idleFrame int) *modernHDBattleSources`
pub(crate) fn modern_hd_battle_body_sources(
    base: &mut [u8],
    attack: &figani::Frame,
    idle: &figani::Frame,
    side: u8,
    present: battlepresent::NativePhysicalBodyPresent,
    attack_resource: i64,
    attack_frame: i64,
    idle_resource: i64,
    idle_frame: i64,
) -> Option<ModernHDBattleSources> {
    todo!("modern_hd_battle_runtime.go:modernHDBattleBodySources")
}

impl Game {
    /// Go：`modern_hd_battle_runtime.go:54`，`func (g *Game) modernHDBattleWanted(scene *nativePhysicalScene) bool`
    pub(crate) fn modern_hd_battle_wanted(&self, scene: Option<&NativePhysicalScene>) -> bool {
        todo!("modern_hd_battle_runtime.go:modernHDBattleWanted")
    }

    /// Go：`modern_hd_battle_runtime.go:85`，`func (g *Game) syncModernHDBattleOwner()`
    pub(crate) fn sync_modern_hd_battle_owner(&mut self) {
        todo!("modern_hd_battle_runtime.go:syncModernHDBattleOwner")
    }

    /// Go：`modern_hd_battle_runtime.go:312`，`func (g *Game) drawModernHDBattlePrelude(screen *ebiten.Image, scene *nativePhysicalScene) (bool, error)`
    pub(crate) fn draw_modern_hd_battle_prelude(
        &mut self,
        screen: &mut Picture,
        scene: &mut NativePhysicalScene,
    ) -> Result<bool> {
        todo!("modern_hd_battle_runtime.go:drawModernHDBattlePrelude")
    }

    /// Go：`modern_hd_battle_runtime.go:331`，`func (g *Game) drawModernHDBattleBody(screen *ebiten.Image, scene *nativePhysicalScene, job *nativePhysicalBodyJob) (bool, error)`
    pub(crate) fn draw_modern_hd_battle_body(
        &mut self,
        screen: &mut Picture,
        scene: &mut NativePhysicalScene,
        job: &mut NativePhysicalBodyJob,
    ) -> Result<bool> {
        todo!("modern_hd_battle_runtime.go:drawModernHDBattleBody")
    }
}
