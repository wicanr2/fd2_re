//! Go 來源：`remake/cmd/fd2/modern_hd_split_strike.go`（333 行）。
//!
//! 骨架：Go 檔的 14 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// 消費既有24／29 owner的索引及乾淨base，不寫入排程或交易。
///
/// Go：`modern_hd_split_strike.go:107`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct ModernHDSplitStrike {}

/// 衍生打擊的 BG.DAT 背景（與施放者 TAI 平台）HD 來源。原版路徑也先跑它：背景或平台的幾何不合時整個指令拒絕，
/// 和 Go 一樣在消耗亂數之前失敗。
///
/// Go：`modern_hd_split_strike.go:86`，`func modernSplitStrikeScenery(background fdother.Frame, selector int, platform *fdother.Frame) ([]modernHDBattleSource, error)`
pub(crate) fn modern_split_strike_scenery(
    background: &fdother::Frame,
    selector: i64,
    platform: Option<&fdother::Frame>,
) -> Result<Vec<ModernHDBattleSource>> {
    if !modern_battle_backdrop_geometry(background) {
        return Ok(Vec::new());
    }
    let mut frame = modern_hd_battle_surface_frame(background)?;
    (frame.x, frame.y) = (0, 50);
    let mut sources = vec![ModernHDBattleSource {
        archive: String::from("BG.DAT"),
        resource: selector,
        frame,
        fill: -1,
        ..Default::default()
    }];
    if let Some(platform) = platform {
        sources.push(modern_battle_platform_source(platform, selector)?);
    }
    Ok(sources)
}
