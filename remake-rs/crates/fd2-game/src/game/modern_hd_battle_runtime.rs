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
    base: &[u8],
    attack: &figani::Frame,
    idle: &figani::Frame,
    side: u8,
    present: &battlepresent::NativePhysicalBodyPresent,
    attack_resource: i64,
    attack_frame: i64,
    idle_resource: i64,
    idle_frame: i64,
) -> ModernHDBattleSources {
    let a = ModernHDBattleSource {
        frame: attack.clone(),
        resource: attack_resource,
        index: attack_frame,
        fill: -1,
        ..Default::default()
    };
    let b = ModernHDBattleSource {
        frame: idle.clone(),
        resource: idle_resource,
        index: idle_frame,
        dx: present.dx,
        dy: present.dy,
        fill: present.opaque_fill,
        ..Default::default()
    };
    let layers = if (side == 0 && attack.raw_byte7 & 1 == 0) || (side != 0 && attack.raw_byte7 & 1 != 0) {
        vec![a, b]
    } else {
        vec![b, a]
    };
    ModernHDBattleSources { base: base.to_vec(), layers, ..Default::default() }
}

impl Game {
    /// Go：`modern_hd_battle_runtime.go:54`，`func (g *Game) modernHDBattleWanted(scene *nativePhysicalScene) bool`
    ///
    /// 忠實模式（沒有 HD 美術、HD 介面關閉）走到的分支照 Go 移植；HD 美術開啟後依主題素材判定的
    /// 部分在 R6。
    pub(crate) fn modern_hd_battle_wanted(&self, scene: Option<&NativePhysicalScene>) -> bool {
        let Some(scene) = scene else {
            return false;
        };
        if scene.body_resources.as_ref().is_none_or(|b| b.attack.is_none()) {
            return false;
        }
        if self.modern_hd_interface_enabled() && !scene.hd_panel_text.is_empty() {
            return true;
        }
        match self.modern_story_portraits.as_ref() {
            Some(set) if set.has_hd => {
                todo!("R6: modern_hd_battle_runtime.go:modernHDBattleWanted（HD 美術的背景、台座與動畫素材判定）")
            }
            _ => false,
        }
    }

    /// Go：`modern_hd_battle_runtime.go:85`，`func (g *Game) syncModernHDBattleOwner()`
    ///
    /// `modernHDBattleGPU` 只在 HD 戰鬥畫面建立；忠實模式下是 nil，Go 直接返回。
    pub(crate) fn sync_modern_hd_battle_owner(&mut self) {
        if self.modern_hd_battle_gpu.is_none() {
            return;
        }
        todo!("R6: modern_hd_battle_runtime.go:syncModernHDBattleOwner（HD 戰鬥 GPU 擁有者切換）")
    }

    /// Go：`modern_hd_battle_runtime.go:312`，`func (g *Game) drawModernHDBattlePrelude(screen *ebiten.Image, scene *nativePhysicalScene) (bool, error)`
    pub(crate) fn draw_modern_hd_battle_prelude(
        &mut self,
        screen: &mut Picture,
        scene: &mut NativePhysicalScene,
    ) -> Result<bool> {
        if !self.modern_hd_battle_wanted(Some(scene)) {
            return Ok(false);
        }
        todo!("R6: modern_hd_battle_runtime.go:drawModernHDBattlePrelude（HD 戰鬥前奏繪製）")
    }

    /// Go：`modern_hd_battle_runtime.go:331`，`func (g *Game) drawModernHDBattleBody(screen *ebiten.Image, scene *nativePhysicalScene, job *nativePhysicalBodyJob) (bool, error)`
    pub(crate) fn draw_modern_hd_battle_body(
        &mut self,
        screen: &mut Picture,
        scene: &mut NativePhysicalScene,
        job: &mut NativePhysicalBodyJob,
    ) -> Result<bool> {
        if !self.modern_hd_battle_wanted(Some(scene)) || job.hd_sources.is_none() {
            return Ok(false);
        }
        todo!("R6: modern_hd_battle_runtime.go:drawModernHDBattleBody（HD 戰鬥本體繪製）")
    }
}
