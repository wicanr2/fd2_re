//! Go 來源：`remake/internal/ending/native_dialogue.go`（268 行）。

use super::*;

pub(crate) const ENDING_DIALOGUE_STRIDE: i64 = 320;

/// Go 原式：`320 * 200`
pub(crate) const ENDING_DIALOGUE_BYTES: i64 = 64000;

pub(crate) const ENDING_DIALOGUE_PORTRAIT_ANCHOR: i64 = 0x9017;

pub(crate) const ENDING_DIALOGUE_TEXT_OFFSET: i64 = 0x9514;

/// Go 原式：`19 * endingDialogueStride`
pub(crate) const ENDING_DIALOGUE_LINE_STEP: i64 = 6080;

pub(crate) const ENDING_DIALOGUE_GLYPH_STEP: i64 = 16;

pub(crate) const ENDING_DIALOGUE_MAXIMUM_GLYPHS: i64 = 13;

/// ComposeNativeDialogueBase 建立 sub_1956B 的穩定19×5格網與下框DATO頭像。
/// 0x9017是每列最右端，必須使用原版right-to-left寫入方向。
///
/// Go：`native_dialogue.go:25`，`func ComposeNativeDialogueBase(background []byte, cells []fdother.RawCell, portrait dato.Frame) ([]byte, error)`
pub fn compose_native_dialogue_base(
    background: &[u8],
    cells: &[fdother::RawCell],
    portrait: &dato::Frame,
) -> Result<Vec<u8>> {
    todo!("ending/native_dialogue.go:ComposeNativeDialogueBase")
}

/// ComposeNativeDialogueOpeningFrames 重用已證實的 sub_1974C 六段310px crop。
///
/// Go：`native_dialogue.go:46`，`func ComposeNativeDialogueOpeningFrames(background, base []byte) ([][]byte, error)`
pub fn compose_native_dialogue_opening_frames(background: &[u8], base: &[u8]) -> Result<Vec<Vec<u8>>> {
    campaign::native_class_list_opening_frames(background, base)
}

/// ComposeNativeDialogueProgressiveFrames 依 sub_15F84 的目的位址、色碼與
/// ordinary-glyph發布順序，建立單一typed page。base須已含該句speaker frame0。
///
/// Go：`native_dialogue.go:52`，`func ComposeNativeDialogueProgressiveFrames(base []byte, font *fdtxt.Font, glyphs map[string]int, pages [][]string, page int) ([][]byte, error)`
pub fn compose_native_dialogue_progressive_frames(
    base: &[u8],
    font: Option<&fdtxt::Font>,
    glyphs: Option<&BTreeMap<String, i64>>,
    pages: &[Vec<String>],
    page: i64,
) -> Result<Vec<Vec<u8>>> {
    todo!("ending/native_dialogue.go:ComposeNativeDialogueProgressiveFrames")
}

/// ComposeNativeDialogueMouthFrame 只替換下框頭像；caller仍負責等待輸入時序。
///
/// Go：`native_dialogue.go:84`，`func ComposeNativeDialogueMouthFrame(stable []byte, portrait dato.Frame) ([]byte, error)`
pub fn compose_native_dialogue_mouth_frame(stable: &[u8], portrait: &dato::Frame) -> Result<Vec<u8>> {
    todo!("ending/native_dialogue.go:ComposeNativeDialogueMouthFrame")
}

/// ComposeNativeDialogueClosingFrames 重播 sub_2D31B 的五段crop並加入caller source restore。
///
/// Go：`native_dialogue.go:96`，`func ComposeNativeDialogueClosingFrames(background, base []byte) ([][]byte, error)`
pub fn compose_native_dialogue_closing_frames(background: &[u8], base: &[u8]) -> Result<Vec<Vec<u8>>> {
    todo!("ending/native_dialogue.go:ComposeNativeDialogueClosingFrames")
}

/// Go：`native_dialogue.go:105`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeDialogueUtteranceFrames {
    pub pages: Vec<Vec<Vec<u8>>>,
    pub mouth_open: Vec<Vec<u8>>,
}

/// Go：`native_dialogue.go:110`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeDialogueBlockFrames {
    pub opening: Vec<Vec<u8>>,
    pub utterances: Vec<NativeDialogueUtteranceFrames>,
    pub closing: Vec<Vec<u8>>,
}

/// Go：`native_dialogue.go:116`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeDialoguePhase(pub u8);

/// Go 原式：`iota`
pub const NATIVE_DIALOGUE_OPENING: NativeDialoguePhase = NativeDialoguePhase(0);

pub const NATIVE_DIALOGUE_PROGRESSIVE: NativeDialoguePhase = NativeDialoguePhase(1);

pub const NATIVE_DIALOGUE_WAITING: NativeDialoguePhase = NativeDialoguePhase(2);

pub const NATIVE_DIALOGUE_CLOSING: NativeDialoguePhase = NativeDialoguePhase(3);

pub const NATIVE_DIALOGUE_COMPLETE: NativeDialoguePhase = NativeDialoguePhase(4);

impl fmt::Display for NativeDialoguePhase {
    /// Go：`native_dialogue.go:126`，`func (p NativeDialoguePhase) String() string`
    /// Rust 以 `Display` 實作 Go 的 `String()`。
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        todo!("ending/native_dialogue.go:NativeDialoguePhase.String")
    }
}

/// NativeDialoguePlayback 只管理已完整預建的indexed frames；輸入映射與時鐘由caller擁有。
///
/// Go：`native_dialogue.go:144`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeDialoguePlayback {
    pub blocks: Vec<NativeDialogueBlockFrames>,
    pub block: i64,
    pub utterance: i64,
    pub page: i64,
    pub frame: i64,
    pub phase: NativeDialoguePhase,
}

/// Go：`native_dialogue.go:151`，`func NewNativeDialoguePlayback(blocks []NativeDialogueBlockFrames) (*NativeDialoguePlayback, error)`
pub fn new_native_dialogue_playback(blocks: &[NativeDialogueBlockFrames]) -> Result<NativeDialoguePlayback> {
    todo!("ending/native_dialogue.go:NewNativeDialoguePlayback")
}

impl NativeDialoguePlayback {
    /// Go：`native_dialogue.go:183`，`func (p *NativeDialoguePlayback) CurrentFrame(mouthOpen bool) []byte`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn current_frame(&self, mouth_open: bool) -> Vec<u8> {
        todo!("ending/native_dialogue.go:NativeDialoguePlayback.CurrentFrame")
    }

    /// Go：`native_dialogue.go:206`，`func (p *NativeDialoguePlayback) Waiting() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn waiting(&self) -> bool {
        self.phase == NATIVE_DIALOGUE_WAITING
    }

    /// Go：`native_dialogue.go:210`，`func (p *NativeDialoguePlayback) Done() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn done(&self) -> bool {
        self.phase == NATIVE_DIALOGUE_COMPLETE
    }

    /// Step 發布下一張非互動畫面；等待輸入時保持穩定頁。
    ///
    /// Go：`native_dialogue.go:215`，`func (p *NativeDialoguePlayback) Step()`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn step(&mut self) {
        todo!("ending/native_dialogue.go:NativeDialoguePlayback.Step")
    }

    /// Confirm 只在完整頁等待期接受輸入，依序換頁、換句或開始收框。
    ///
    /// Go：`native_dialogue.go:250`，`func (p *NativeDialoguePlayback) Confirm() bool`
    pub fn confirm(&mut self) -> bool {
        todo!("ending/native_dialogue.go:NativeDialoguePlayback.Confirm")
    }
}
