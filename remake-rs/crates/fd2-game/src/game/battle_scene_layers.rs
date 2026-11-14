//! Go 來源：`remake/cmd/fd2/battle_scene_layers.go`（36 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// battleSceneLayer 是全螢幕戰鬥演出的三個圖層。
///
/// Go：`battle_scene_layers.go:4`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct BattleSceneLayer(pub(crate) i64);

/// 守方 figure（待機幀循環）
///
/// Go 原式：`iota`
pub(crate) const BATTLE_LAYER_DEFENDER_FIGURE: BattleSceneLayer = BattleSceneLayer(0);

/// 我方腳下的 TAI 台座
///
/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const BATTLE_LAYER_OWN_PEDESTAL: BattleSceneLayer = BattleSceneLayer(1);

/// 攻方 figure（攻擊幀序）
///
/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const BATTLE_LAYER_ATTACKER_FIGURE: BattleSceneLayer = BattleSceneLayer(2);
