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
    if background.len() as i64 != ENDING_DIALOGUE_BYTES || cells.len() <= 17 {
        return Err(campaign_err!("ending: native dialogue base assets are invalid"));
    }
    let mut frame = background.to_vec();
    let placements = fdother::plan_native_dialogue_frame_grid(ENDING_DIALOGUE_STRIDE, 5, 112, 19, 5)?;
    for placement in &placements {
        // Go 直接索引 `cells[placement.ResourceIndex]`；超出範圍會 panic，Rust 回錯誤。
        let Some(cell) = usize::try_from(placement.resource_index).ok().and_then(|i| cells.get(i)) else {
            return Err(campaign_err!("ending: native dialogue base assets are invalid"));
        };
        cell.blit_opaque_at_offset(&mut frame, ENDING_DIALOGUE_STRIDE, placement.destination_byte)?;
    }
    portrait.blit_right_to_left_at_offset(&mut frame, ENDING_DIALOGUE_STRIDE, ENDING_DIALOGUE_PORTRAIT_ANCHOR)?;
    Ok(frame)
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
    let (Some(font), Some(glyphs)) = (font, glyphs) else {
        return Err(campaign_err!("ending: native dialogue progressive assets or page are invalid"));
    };
    if base.len() as i64 != ENDING_DIALOGUE_BYTES || page < 0 || page >= pages.len() as i64 {
        return Err(campaign_err!("ending: native dialogue progressive assets or page are invalid"));
    }
    let rows = &pages[page as usize];
    if rows.is_empty() || rows.len() > 4 {
        return Err(campaign_err!("ending: native dialogue page {page} has {} rows", rows.len()));
    }
    let mut frame = base.to_vec();
    let mut frames = Vec::with_capacity(1 + ENDING_DIALOGUE_MAXIMUM_GLYPHS as usize * rows.len());
    frames.push(frame.clone());
    let style = fdtxt::NativeGlyphStyle { foreground: 0xcd, shadow: 0x4c, background: 0x4a };
    for (row, text) in rows.iter().enumerate() {
        let runes: Vec<char> = text.chars().collect();
        if runes.is_empty() || runes.len() as i64 > ENDING_DIALOGUE_MAXIMUM_GLYPHS {
            return Err(campaign_err!("ending: native dialogue row {row} has {} glyphs", runes.len()));
        }
        for (column, r) in runes.iter().enumerate() {
            let key = r.to_string();
            let glyph = match glyphs.get(&key) {
                Some(&glyph) if glyph >= 0 && glyph < font.glyph_count() as i64 => glyph,
                _ => {
                    return Err(campaign_err!(
                        "ending: native dialogue glyph {} is unavailable",
                        battle::go_quote(&key)
                    ));
                }
            };
            let destination = ENDING_DIALOGUE_TEXT_OFFSET
                + row as i64 * ENDING_DIALOGUE_LINE_STEP
                + column as i64 * ENDING_DIALOGUE_GLYPH_STEP;
            font.blit_native_glyph(&mut frame, ENDING_DIALOGUE_STRIDE, destination, glyph, style)?;
            frames.push(frame.clone());
        }
    }
    Ok(frames)
}

/// ComposeNativeDialogueMouthFrame 只替換下框頭像；caller仍負責等待輸入時序。
///
/// Go：`native_dialogue.go:84`，`func ComposeNativeDialogueMouthFrame(stable []byte, portrait dato.Frame) ([]byte, error)`
pub fn compose_native_dialogue_mouth_frame(stable: &[u8], portrait: &dato::Frame) -> Result<Vec<u8>> {
    if stable.len() as i64 != ENDING_DIALOGUE_BYTES {
        return Err(campaign_err!("ending: native dialogue mouth base is invalid"));
    }
    let mut frame = stable.to_vec();
    portrait.blit_right_to_left_at_offset(&mut frame, ENDING_DIALOGUE_STRIDE, ENDING_DIALOGUE_PORTRAIT_ANCHOR)?;
    Ok(frame)
}

/// ComposeNativeDialogueClosingFrames 重播 sub_2D31B 的五段crop並加入caller source restore。
///
/// Go：`native_dialogue.go:96`，`func ComposeNativeDialogueClosingFrames(background, base []byte) ([][]byte, error)`
pub fn compose_native_dialogue_closing_frames(background: &[u8], base: &[u8]) -> Result<Vec<Vec<u8>>> {
    let mut frames = campaign::native_class_list_closing_frames(background, base)?;
    frames.push(background.to_vec());
    Ok(frames)
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
        f.write_str(match *self {
            NATIVE_DIALOGUE_OPENING => "opening",
            NATIVE_DIALOGUE_PROGRESSIVE => "progressive",
            NATIVE_DIALOGUE_WAITING => "waiting",
            NATIVE_DIALOGUE_CLOSING => "closing",
            NATIVE_DIALOGUE_COMPLETE => "complete",
            _ => "unknown",
        })
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
    if blocks.is_empty() {
        return Err(campaign_err!("ending: native dialogue playback has no blocks"));
    }
    for (block_index, block) in blocks.iter().enumerate() {
        if block.opening.len() != 6 || block.closing.len() != 6 || block.utterances.is_empty() {
            return Err(campaign_err!("ending: native dialogue block {block_index} has incomplete lifecycle frames"));
        }
        for (utterance_index, utterance) in block.utterances.iter().enumerate() {
            if utterance.pages.is_empty() || utterance.mouth_open.len() != utterance.pages.len() {
                return Err(campaign_err!(
                    "ending: native dialogue block {block_index} utterance {utterance_index} has incomplete pages"
                ));
            }
            for (page, frames) in utterance.pages.iter().enumerate() {
                if frames.is_empty() || utterance.mouth_open[page].len() as i64 != ENDING_DIALOGUE_BYTES {
                    return Err(campaign_err!(
                        "ending: native dialogue block {block_index} utterance {utterance_index} page {page} is incomplete"
                    ));
                }
                if frames.iter().any(|frame| frame.len() as i64 != ENDING_DIALOGUE_BYTES) {
                    return Err(campaign_err!(
                        "ending: native dialogue block {block_index} utterance {utterance_index} page {page} has invalid frame"
                    ));
                }
            }
        }
        if block.opening.iter().chain(block.closing.iter()).any(|frame| frame.len() as i64 != ENDING_DIALOGUE_BYTES) {
            return Err(campaign_err!(
                "ending: native dialogue block {block_index} has invalid opening or closing frame"
            ));
        }
    }
    Ok(NativeDialoguePlayback { blocks: blocks.to_vec(), phase: NATIVE_DIALOGUE_OPENING, ..Default::default() })
}

impl NativeDialoguePlayback {
    /// Go：`native_dialogue.go:183`，`func (p *NativeDialoguePlayback) CurrentFrame(mouthOpen bool) []byte`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn current_frame(&self, mouth_open: bool) -> Vec<u8> {
        // Go 回傳共用的切片；Rust 回傳複本（呼叫端只讀）。nil 對應空 Vec。
        if self.phase == NATIVE_DIALOGUE_COMPLETE || self.block < 0 || self.block >= self.blocks.len() as i64 {
            return Vec::new();
        }
        let block = &self.blocks[self.block as usize];
        let (u, p, f) = (self.utterance as usize, self.page as usize, self.frame as usize);
        match self.phase {
            NATIVE_DIALOGUE_OPENING => block.opening[f].clone(),
            NATIVE_DIALOGUE_PROGRESSIVE => block.utterances[u].pages[p][f].clone(),
            NATIVE_DIALOGUE_WAITING => {
                if mouth_open {
                    return block.utterances[u].mouth_open[p].clone();
                }
                let frames = &block.utterances[u].pages[p];
                frames[frames.len() - 1].clone()
            }
            NATIVE_DIALOGUE_CLOSING => block.closing[f].clone(),
            _ => Vec::new(),
        }
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
        if self.done() || self.waiting() {
            return;
        }
        let block = &self.blocks[self.block as usize];
        match self.phase {
            NATIVE_DIALOGUE_OPENING => {
                if self.frame + 1 < block.opening.len() as i64 {
                    self.frame += 1;
                    return;
                }
                (self.phase, self.frame) = (NATIVE_DIALOGUE_PROGRESSIVE, 0);
            }
            NATIVE_DIALOGUE_PROGRESSIVE => {
                let frames = &block.utterances[self.utterance as usize].pages[self.page as usize];
                if self.frame + 1 < frames.len() as i64 {
                    self.frame += 1;
                    return;
                }
                (self.phase, self.frame) = (NATIVE_DIALOGUE_WAITING, 0);
            }
            NATIVE_DIALOGUE_CLOSING => {
                if self.frame + 1 < block.closing.len() as i64 {
                    self.frame += 1;
                    return;
                }
                self.block += 1;
                (self.utterance, self.page, self.frame) = (0, 0, 0);
                self.phase = if self.block >= self.blocks.len() as i64 {
                    NATIVE_DIALOGUE_COMPLETE
                } else {
                    NATIVE_DIALOGUE_OPENING
                };
            }
            _ => {}
        }
    }

    /// Confirm 只在完整頁等待期接受輸入，依序換頁、換句或開始收框。
    ///
    /// Go：`native_dialogue.go:250`，`func (p *NativeDialoguePlayback) Confirm() bool`
    pub fn confirm(&mut self) -> bool {
        if !self.waiting() {
            return false;
        }
        let block = &self.blocks[self.block as usize];
        let utterance = &block.utterances[self.utterance as usize];
        if self.page + 1 < utterance.pages.len() as i64 {
            self.page += 1;
            (self.frame, self.phase) = (0, NATIVE_DIALOGUE_PROGRESSIVE);
            return true;
        }
        if self.utterance + 1 < block.utterances.len() as i64 {
            self.utterance += 1;
            (self.page, self.frame, self.phase) = (0, 0, NATIVE_DIALOGUE_PROGRESSIVE);
            return true;
        }
        (self.frame, self.phase) = (0, NATIVE_DIALOGUE_CLOSING);
        true
    }
}

#[cfg(test)]
mod native_dialogue_test;
