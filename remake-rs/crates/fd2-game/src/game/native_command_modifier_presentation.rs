//! Go 來源：`remake/cmd/fd2/native_command_modifier_presentation.go`（138 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command_modifier_presentation.go:13`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommandModifierPresentationJob {
    pub(crate) command_id: i64,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palettes: Vec<ColorPalette>,
    pub(crate) phase: i64,
    pub(crate) drawn: bool,
    pub(crate) transaction: Option<Box<dyn FnMut(&mut Game) -> Result<()> + Send>>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd_frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) unit_present: Option<NativeUnitPresentJob>,
}

impl Game {
    /// Go：`native_command_modifier_presentation.go:95`，`func (g *Game) stepNativeCommandModifierPresentation()`
    pub(crate) fn step_native_command_modifier_presentation(&mut self) {
        let Some(j) = self.native_modifier_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        j.phase += 1;
        if j.phase < j.palettes.len() as i64 {
            return;
        }
        // Go 在工作仍掛在 `g.nativeModifierPresentation` 時呼叫交易，之後才清掉；nil func 解參考 panic。
        // Go 的 `then := j.then` 讀的是同一個工作，交易改不到它，所以先取出等價。
        let mut transaction = j.transaction.take().expect("Go：j.transaction 為 nil 時呼叫 panic");
        let then = j.then.take();
        let err = transaction(self);
        self.native_modifier_presentation = None;
        if let Err(err) = err {
            self.msg = format!("原始指令 modifier transaction 失敗: {}", err.message());
            return;
        }
        if let Some(then) = then {
            then(self);
        }
    }
}
