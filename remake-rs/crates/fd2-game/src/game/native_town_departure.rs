//! Go 來源：`remake/cmd/fd2/native_town_departure.go`（111 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_town_departure.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeTownDepartureFrame {
    pub(crate) indexed: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) palette: ColorPalette,
}

/// Go：`native_town_departure.go:20`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeTownDepartureJob {
    pub(crate) frames: Vec<NativeTownDepartureFrame>,
    pub(crate) step: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) source: Option<NativeTownFrameSource>,
}

impl Game {
    /// Go：`native_town_departure.go:68`，`func (g *Game) stepNativeTownDeparture()`
    pub(crate) fn step_native_town_departure(&mut self) {
        todo!("native_town_departure.go:stepNativeTownDeparture")
    }
}
