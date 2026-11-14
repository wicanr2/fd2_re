//! Go 來源：`remake/cmd/fd2/native_level_up_dialogue.go`（373 行）。
//!
//! 骨架：Go 檔的 11 個函式中產生 9 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const NATIVE_LEVEL_UP_EXP_TEXT_INDEX: i64 = 0x1e8;

pub(crate) const NATIVE_LEVEL_UP_TEXT_INDEX: i64 = 0x1e9;

/// AP、DP、DX、MaxHP、MaxMP 依序 0x1EA..0x1EE
pub(crate) const NATIVE_LEVEL_UP_GROWTH_TEXT_BASE: i64 = 0x1ea;

pub(crate) const NATIVE_LEVEL_UP_LEARN_TEXT_INDEX: i64 = 0x24b;

/// FFFC = 0x1B9 + command id（0x1E389）
pub(crate) const NATIVE_LEVEL_UP_LEARN_NAME_BASE: i64 = 0x1b9;

/// 0x1E504 push 0xb：11 個 BIOS tick（18.2 Hz）≈ 0.6 s；重製端以 60 Hz 幀近似。
pub(crate) const NATIVE_LEVEL_UP_CLOSE_FRAMES: i64 = 36;

/// Go：`native_level_up_dialogue.go:32`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct PendingNativeLevelUp {
    pub(crate) unit: Option<UnitIdx>,
    pub(crate) exp: i64,
    pub(crate) ups: Vec<battle::LevelUpEvent>,
}

/// Go：`native_level_up_dialogue.go:38`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeLevelUpDialogueState {
    pub(crate) unit: Option<UnitIdx>,
    pub(crate) hd_world: Option<ModernHDMapFrame>,
    pub(crate) hd_coverage: BTreeMap<campaign::NativeDialogueFramePhase, Vec<Vec<bool>>>,
    /// awaitLastKey：最後一頁也要等鍵（原版最後一欄的 FFFD）；沒有升級時只有一頁一行、
    /// 沒有 FFFD，直接進計時。
    pub(crate) await_last_key: bool,
    pub(crate) close_timer: i64,
    /// 0x1E5C0(n) 的 n 個 BIOS tick 換成 60 Hz 幀
    pub(crate) close_frames: i64,
    pub(crate) closing: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    /// queueNativeLevelUpDialogue 在原生 caller 收下經驗時排一則。0x29F72的+6==2
    /// 是累計writer閘門，不是0x1E292的consumer閘門；敵人也可能承接前值（#81）。
    ///
    /// Go：`native_level_up_dialogue.go:53`，`func (g *Game) queueNativeLevelUpDialogue(unit *battle.Unit, exp int, ups []battle.LevelUpEvent)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn queue_native_level_up_dialogue(
        &mut self,
        unit: Option<UnitIdx>,
        exp: i64,
        ups: &mut [battle::LevelUpEvent],
    ) {
        todo!("native_level_up_dialogue.go:queueNativeLevelUpDialogue")
    }

    /// Go：`native_level_up_dialogue.go:60`，`func (g *Game) runPendingNativeLevelUpDialogues(then func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn run_pending_native_level_up_dialogues(
        &mut self,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        todo!("native_level_up_dialogue.go:runPendingNativeLevelUpDialogues")
    }
}

/// expandNativeMessageWords 把 FDTXT_000 一條字串展開成原始 glyph word：FFFA 代入十進位數字、
/// FFFC 代入名字，FFFD 只回報等鍵。0x15F84 的單行訊息（升級、中毒扣血）共用。
///
/// Go：`native_level_up_dialogue.go:74`，`func expandNativeMessageWords(strings interface { Words(int) ([]uint16, error) }, index int, number int, name []uint16) ([]uint16, bool, error)`
/// Go 參數是匿名介面 `interface{ Words(int) ([]uint16, error) }`，正式程式只傳 `*fdtxt.Strings`。
pub(crate) fn expand_native_message_words(
    strings: &fdtxt::Strings,
    index: i64,
    number: i64,
    name: &[u16],
) -> Result<(Vec<u16>, bool)> {
    todo!("native_level_up_dialogue.go:expandNativeMessageWords")
}

/// nativeLevelUpDialogueLines 把 0x1E292 會寫的每一行展開成 FDTXT 原始字（FFFA／FFFC 已代入），
/// 依 FFFD 切頁。回傳每頁的行（每行是原始 glyph word 序列），以及最後一行是否以 FFFD 收尾
/// （是：最後一頁也等鍵；否：例如只有經驗一行、或「學會了」收尾，直接進 0x1E5C0 計時）。
///
/// Go：`native_level_up_dialogue.go:151`，`func nativeLevelUpDialogueLines(strings interface { Words(int) ([]uint16, error) }, pending pendingNativeLevelUp) ([][][]uint16, bool, error)`
/// Go 參數是匿名介面 `interface{ Words(int) ([]uint16, error) }`，正式程式只傳 `*fdtxt.Strings`。
pub(crate) fn native_level_up_dialogue_lines(
    strings: &fdtxt::Strings,
    pending: &PendingNativeLevelUp,
) -> Result<(Vec<Vec<Vec<u16>>>, bool)> {
    todo!("native_level_up_dialogue.go:nativeLevelUpDialogueLines")
}

impl Game {
    /// Go：`native_level_up_dialogue.go:207`，`func (g *Game) beginNativeLevelUpDialogue(pending pendingNativeLevelUp, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_level_up_dialogue(
        &mut self,
        pending: &PendingNativeLevelUp,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        todo!("native_level_up_dialogue.go:beginNativeLevelUpDialogue")
    }

    /// beginNativeTimedMessageDialogue 是 0x1956B(unit+7) 開框 → 0x15F84 寫 FDTXT_000 字串（FFFD 等鍵
    /// 切頁）→ 0x1E5C0(n) 計時 → 0x196CB 關框的共用排程；升級對話（0x1E292）與中毒扣血訊息
    /// （0x1A866）都走這條。
    ///
    /// Go：`native_level_up_dialogue.go:228`，`func (g *Game) beginNativeTimedMessageDialogue(unit *battle.Unit, textIndex int, pages [][][]uint16, lastWait bool, closeFrames int, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_timed_message_dialogue(
        &mut self,
        unit: Option<UnitIdx>,
        text_index: i64,
        pages: &[Vec<Vec<u16>>],
        last_wait: bool,
        close_frames: i64,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        todo!("native_level_up_dialogue.go:beginNativeTimedMessageDialogue")
    }

    /// nativeLevelUpDialogueAtWait 回報對話正停在等鍵（頁尾 FFFD，或最後一頁字寫完）。
    ///
    /// Go：`native_level_up_dialogue.go:289`，`func (g *Game) nativeLevelUpDialogueAtWait() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_level_up_dialogue_at_wait(&self) -> bool {
        todo!("native_level_up_dialogue.go:nativeLevelUpDialogueAtWait")
    }

    /// stepNativeLevelUpDialogue 每幀：字寫完且不等鍵的頁（沒有升級的單頁）直接進 0x1E5C0 的
    /// 計時；計時到就 0x196CB 關框。
    ///
    /// Go：`native_level_up_dialogue.go:296`，`func (g *Game) stepNativeLevelUpDialogue(advance bool)`
    pub(crate) fn step_native_level_up_dialogue(&mut self, advance: bool) {
        todo!("native_level_up_dialogue.go:stepNativeLevelUpDialogue")
    }

    /// Go：`native_level_up_dialogue.go:332`，`func (g *Game) finishNativeLevelUpDialogue()`
    pub(crate) fn finish_native_level_up_dialogue(&mut self) {
        todo!("native_level_up_dialogue.go:finishNativeLevelUpDialogue")
    }
}
