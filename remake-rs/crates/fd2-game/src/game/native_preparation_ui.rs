//! Go 來源：`remake/cmd/fd2/native_preparation_ui.go`（510 行）。
//!
//! 骨架：Go 檔的 26 個函式中產生 7 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_preparation_ui.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Clone, Default)]
pub(crate) struct NativePreparationUIAssets {
    pub(crate) roster: Option<fdother::NativePreparationAssets>,
    pub(crate) status: battle::NativeItemPanelDataAssets,
    pub(crate) choices: Vec<fdother::RawCell>,
    pub(crate) dialogue: Vec<fdother::RawCell>,
    pub(crate) portrait: dato::Frame,
    pub(crate) portraits: Vec<dato::Frame>,
    pub(crate) slots_box: fdother::LMI1Entry,
}

impl Game {
    /// Go：`native_preparation_ui.go:36`，`func (g *Game) stepNativePreparationUILifecycle(now time.Time)`
    pub(crate) fn step_native_preparation_ui_lifecycle(&mut self, now: GameInstant) {
        todo!("native_preparation_ui.go:stepNativePreparationUILifecycle")
    }
}

/// Go：`native_preparation_ui.go:50`，`func loadNativePreparationUIAssets() (*nativePreparationUIAssets, error)`
pub(crate) fn load_native_preparation_ui_assets() -> Result<Option<NativePreparationUIAssets>> {
    todo!("native_preparation_ui.go:loadNativePreparationUIAssets")
}

impl Game {
    /// Go：`native_preparation_ui.go:187`，`func (g *Game) nativePreparationPromptActive() bool`
    pub(crate) fn native_preparation_prompt_active(&self) -> bool {
        todo!("native_preparation_ui.go:nativePreparationPromptActive")
    }

    /// Go：`native_preparation_ui.go:195`，`func (g *Game) composeNativePreparationPromptDialogue() ([]byte, bool)`
    pub(crate) fn compose_native_preparation_prompt_dialogue(&self) -> (Vec<u8>, bool) {
        todo!("native_preparation_ui.go:composeNativePreparationPromptDialogue")
    }

    /// Go：`native_preparation_ui.go:208`，`func (g *Game) composeNativePreparationPromptQuestion() ([]byte, bool)`
    pub(crate) fn compose_native_preparation_prompt_question(&self) -> (Vec<u8>, bool) {
        todo!("native_preparation_ui.go:composeNativePreparationPromptQuestion")
    }

    /// Go：`native_preparation_ui.go:299`，`func (g *Game) beginNativePreparationPromptOpening() bool`
    pub(crate) fn begin_native_preparation_prompt_opening(&mut self) -> bool {
        todo!("native_preparation_ui.go:beginNativePreparationPromptOpening")
    }

    /// Go：`native_preparation_ui.go:323`，`func (g *Game) beginNativePreparationPromptClosing(after func()) bool`
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_preparation_prompt_closing(
        &mut self,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        todo!("native_preparation_ui.go:beginNativePreparationPromptClosing")
    }
}
