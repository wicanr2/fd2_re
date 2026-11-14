//! Go 來源：`remake/internal/campaign/native_load_slots.go`（143 行）。

use super::*;

pub const NATIVE_LOAD_SLOT_WIDTH: i64 = 320;

pub const NATIVE_LOAD_SLOT_HEIGHT: i64 = 200;

pub(crate) const NATIVE_LOAD_FRAME_X: i64 = 5;

pub(crate) const NATIVE_LOAD_FRAME_Y: i64 = 112;

pub(crate) const NATIVE_LOAD_TEXT_Y: i64 = 119;

pub(crate) const NATIVE_LOAD_ROW_STRIDE: i64 = 19;

pub(crate) const NATIVE_LOAD_NUMBER_X: i64 = 10;

pub(crate) const NATIVE_LOAD_EMPTY_X: i64 = 88;

pub(crate) const NATIVE_LOAD_CHAPTER_X: i64 = 40;

pub(crate) const NATIVE_LOAD_TITLE_X: i64 = 130;

/// NativeLoadSlot preserves only metadata consumed by the proven 0x30437
/// renderer. Chapter is the raw saved byte; it is not normalized to a remake
/// campaign node.
///
/// Go：`native_load_slots.go:29`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeLoadSlot {
    pub empty: bool,
    pub chapter: u8,
}

/// ComposeNativeLoadSlotsFrame reproduces the stable 0x30437 frame used by
/// 0x30550. It deliberately accepts a caller-owned source snapshot because
/// 0x30550 copies the current VGA page before opening FDOTHER#13 LMI entry16.
///
/// Go：`native_load_slots.go:37`，`func ComposeNativeLoadSlotsFrame( source []byte, dialogueBox fdother.LMI1Entry, strings *fdtxt.Strings, font *fdtxt.Font, slots [4]NativeLoadSlot, selection int, ) ([]byte, error)`
pub fn compose_native_load_slots_frame(
    source: &[u8],
    dialogue_box: &fdother::LMI1Entry,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    slots: [NativeLoadSlot; 4],
    selection: i64,
) -> Result<Vec<u8>> {
    let (Some(strings), Some(font)) = (strings, font) else {
        return Err(campaign_err!("campaign: native load-slot assets/state are invalid"));
    };
    if source.len() as i64 != NATIVE_LOAD_SLOT_WIDTH * NATIVE_LOAD_SLOT_HEIGHT
        || dialogue_box.width != 310
        || dialogue_box.height != 86
        || dialogue_box.pixels.len() as i64 != dialogue_box.width * dialogue_box.height
        || selection < 0
        || selection >= slots.len() as i64
    {
        return Err(campaign_err!("campaign: native load-slot assets/state are invalid"));
    }
    let mut frame = source.to_vec();
    let w = dialogue_box.width as usize;
    for row in 0..dialogue_box.height {
        let y = NATIVE_LOAD_FRAME_Y + row;
        if y >= NATIVE_LOAD_SLOT_HEIGHT {
            break;
        }
        let source_start = row as usize * w;
        let destination_start = (y * NATIVE_LOAD_SLOT_WIDTH + NATIVE_LOAD_FRAME_X) as usize;
        frame[destination_start..destination_start + w]
            .copy_from_slice(&dialogue_box.pixels[source_start..source_start + w]);
    }
    for (row, slot) in slots.iter().enumerate() {
        let row = row as i64;
        let foreground: u8 = if row == selection { 0xc9 } else { 0xcd };
        let style = fdtxt::NativeGlyphStyle { foreground, shadow: 0x4c, background: 0 };
        let y = NATIVE_LOAD_TEXT_Y + row * NATIVE_LOAD_ROW_STRIDE;
        render_native_load_text(&mut frame, strings, font, 0x225, NATIVE_LOAD_NUMBER_X, y, style, row + 1)?;
        if slot.empty {
            render_native_load_text(&mut frame, strings, font, 0x202, NATIVE_LOAD_EMPTY_X, y, style, 0)?;
            continue;
        }
        let chapter = slot.chapter as i64;
        render_native_load_text(&mut frame, strings, font, 0x202 + chapter, NATIVE_LOAD_CHAPTER_X, y, style, 0)?;
        render_native_load_text(&mut frame, strings, font, 0x226 + chapter, NATIVE_LOAD_TITLE_X, y, style, 0)?;
    }
    Ok(frame)
}

/// Go：`native_load_slots.go:106`，`func renderNativeLoadText( frame []byte, strings *fdtxt.Strings, font *fdtxt.Font, textIndex, x, y int, style fdtxt.NativeGlyphStyle, number int, ) error`
pub(crate) fn render_native_load_text(
    frame: &mut [u8],
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    text_index: i64,
    x: i64,
    y: i64,
    style: fdtxt::NativeGlyphStyle,
    number: i64,
) -> Result<()> {
    let words = strings.words(text_index)?;
    let mut expanded: Vec<u16> = Vec::with_capacity(words.len() + 2);
    for &word in &words {
        if word == 0xfffa {
            // Go `uint16(digit-'0')`：負號 '-' 減 '0' 為負值，轉 uint16 時繞回。
            for digit in number.to_string().chars() {
                expanded.push((digit as i32 - '0' as i32) as u16);
            }
            continue;
        }
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!(
                "campaign: unsupported native load-slot control {}",
                battle::go_sharp_x(word as i64)
            ));
        }
        expanded.push(word);
    }
    for (column, &word) in expanded.iter().enumerate() {
        let offset = y * NATIVE_LOAD_SLOT_WIDTH + x + column as i64 * fdtxt::GLYPH_WIDTH as i64;
        font.blit_native_glyph(frame, NATIVE_LOAD_SLOT_WIDTH, offset, word as i64, style)?;
    }
    Ok(())
}

#[cfg(test)]
pub(crate) mod native_load_slots_test;
