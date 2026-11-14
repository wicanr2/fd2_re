//! Go 來源：`remake/cmd/fd2/modern_hd_death_stage.go`（151 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// This source belongs to the native 1DB65 producer, including pixel themes.
/// It is separate from a compound scene: the first 25 copies start at VGA zero
/// and A accumulates LMI writes over the final visible pose.
///
/// Go：`modern_hd_death_stage.go:17`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDDeathStageSource {
    pub(crate) stage: Option<NativeCh23AdapterState>,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) signature: [u8; 32],
    pub(crate) sealed: bool,
}

impl ModernHDDeathStageSource {
    /// Go：`modern_hd_death_stage.go:24`，`func (s *modernHDDeathStageSource) seed(work []byte) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// R4 B 組：Go 對 nil 接收者回 nil；Rust 的呼叫端以 `Option` 判斷，`None` 不呼叫（同樣不做事）。
    pub(crate) fn seed(&self, work: &mut [u8]) -> Result<()> {
        let Some(stage) = self.stage.as_ref() else {
            return Err(GameError::new("native death: same-compose stage unavailable"));
        };
        Ok(indexedmap::seed_native_ch23_staging(work, &stage.staging)?)
    }
}
