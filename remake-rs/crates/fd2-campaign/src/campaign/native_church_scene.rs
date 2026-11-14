//! Go 來源：`remake/internal/campaign/native_church_scene.go`（206 行）。

use super::*;

/// Go 原式：`4*320 + 118`
pub(crate) const NATIVE_CHURCH_PORTRAIT_OFFSET: i64 = 1398;

pub(crate) const NATIVE_LOWER_PORTRAIT_RIGHT_EDGE: i64 = 0x9017;

/// Go 原式：`95*320 + 5`
pub(crate) const NATIVE_CHURCH_DECORATION_OFFSET: i64 = 30405;

/// Go 原式：`99*320 + 16`
pub(crate) const NATIVE_CHURCH_GOLD_OFFSET: i64 = 31696;

/// Go 原式：`119*320 + 12`
pub(crate) const NATIVE_CHURCH_TEXT_OFFSET: i64 = 38092;

/// ComposeNativeChurchScene reproduces the stable 0x3072f indexed scene after
/// sub_1956b's six-frame reveal and before 0x2d669 opens the service cells.
/// Callers must provide the exact mixed-codec resources used by each callee.
///
/// Go：`native_church_scene.go:23`，`func ComposeNativeChurchScene( background []byte, decoration fdother.LMI1Entry, dialogueCells []fdother.RawCell, digitFrames []fdother.Frame, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, gold, textIndex int, ) ([]byte, error)`
pub fn compose_native_church_scene(
    background: &[u8],
    decoration: &fdother::LMI1Entry,
    dialogue_cells: &[fdother::RawCell],
    digit_frames: &[fdother::Frame],
    portrait: &dato::Frame,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    gold: i64,
    text_index: i64,
) -> Result<Vec<u8>> {
    let mut frame =
        compose_native_church_scene_base(background, decoration, dialogue_cells, digit_frames, portrait, gold)?;
    render_native_church_text(&mut frame, strings, font, text_index)
}

/// ComposeNativeChurchSceneBase builds 0x3072f's stable scene before FDTXT585
/// or FDTXT586 is written.
///
/// Go：`native_church_scene.go:44`，`func ComposeNativeChurchSceneBase( background []byte, decoration fdother.LMI1Entry, dialogueCells []fdother.RawCell, digitFrames []fdother.Frame, portrait dato.Frame, gold int, ) ([]byte, error)`
pub fn compose_native_church_scene_base(
    background: &[u8],
    decoration: &fdother::LMI1Entry,
    dialogue_cells: &[fdother::RawCell],
    digit_frames: &[fdother::Frame],
    portrait: &dato::Frame,
    gold: i64,
) -> Result<Vec<u8>> {
    if background.len() != 320 * 200
        || dialogue_cells.len() <= 17
        || digit_frames.len() != 10
        || !(0..=99_999_999).contains(&gold)
    {
        return Err(campaign_err!("campaign: native church scene assets/state are invalid"));
    }
    let mut frame = compose_native_church_dialogue_overlay(background, dialogue_cells, portrait)?;
    decoration.blit_opaque_at(
        &mut frame,
        320,
        NATIVE_CHURCH_DECORATION_OFFSET % 320,
        NATIVE_CHURCH_DECORATION_OFFSET / 320,
        false,
    )?;
    let digits = format!("{gold:08}");
    for (i, d) in digits.bytes().take(8).enumerate() {
        let index = (d - b'0') as usize;
        digit_frames[index].blit_at(&mut frame, 320, NATIVE_CHURCH_GOLD_OFFSET + 6 * i as i64, -1)?;
    }
    Ok(frame)
}

/// ComposeNativeChurchDialogueOverlay reproduces sub_1956b's stable target:
/// the FDOTHER#5 19×5 dialogue grid and DATO#131 frame zero over a caller-owned
/// source snapshot. Text is intentionally excluded because 0x2d31b closes
/// against this buffer after the text was drawn only to VGA.
///
/// Go：`native_church_scene.go:82`，`func ComposeNativeChurchDialogueOverlay( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, ) ([]byte, error)`
pub fn compose_native_church_dialogue_overlay(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
) -> Result<Vec<u8>> {
    compose_native_church_dialogue_overlay_at(background, dialogue_cells, portrait, NATIVE_CHURCH_PORTRAIT_OFFSET)
}

/// ComposeNativeChurchDialogueOverlayAt is the recovered stable target of
/// 0x1956b shared by shop, church, and other facility variants. The portrait
/// destination is selected by the native resource-ID switch at 0x195fc.
///
/// Go：`native_church_scene.go:95`，`func ComposeNativeChurchDialogueOverlayAt( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, portraitOffset int, ) ([]byte, error)`
pub fn compose_native_church_dialogue_overlay_at(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    portrait_offset: i64,
) -> Result<Vec<u8>> {
    if background.len() != 320 * 200 || dialogue_cells.len() <= 17 {
        return Err(campaign_err!("campaign: native church dialogue overlay assets are invalid"));
    }
    let mut frame = background.to_vec();
    let placements = fdother::plan_native_dialogue_frame_grid(320, 5, 112, 19, 5)?;
    for placement in placements {
        dialogue_cells[placement.resource_index as usize].blit_opaque_at_offset(
            &mut frame,
            320,
            placement.destination_byte,
        )?;
    }
    blit_native_dialogue_portrait_at(&mut frame, portrait, portrait_offset)?;
    Ok(frame)
}

/// Go：`native_church_scene.go:122`，`func blitNativeDialoguePortraitAt(frame []byte, portrait dato.Frame, offset int) error`
pub(crate) fn blit_native_dialogue_portrait_at(frame: &mut [u8], portrait: &dato::Frame, offset: i64) -> Result<()> {
    if offset == NATIVE_LOWER_PORTRAIT_RIGHT_EDGE {
        return Ok(portrait.blit_right_to_left_at_offset(frame, 320, offset)?);
    }
    Ok(portrait.blit_at_offset(frame, 320, offset)?)
}

/// Go：`native_church_scene.go:129`，`func renderNativeChurchText( frame []byte, strings *fdtxt.Strings, font *fdtxt.Font, textIndex int, ) ([]byte, error)`
pub(crate) fn render_native_church_text(
    frame: &mut [u8],
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    text_index: i64,
) -> Result<Vec<u8>> {
    compose_native_church_text_at(frame, strings, font, text_index, NATIVE_CHURCH_TEXT_OFFSET)
}

/// ComposeNativeChurchTextAt applies the proven 0x15f84 church text style at
/// a caller-owned VGA offset. It supports only the verified FFFE soft newline.
///
/// Go：`native_church_scene.go:142`，`func ComposeNativeChurchTextAt( frame []byte, strings *fdtxt.Strings, font *fdtxt.Font, textIndex, baseOffset int, ) ([]byte, error)`
pub fn compose_native_church_text_at(
    frame: &mut [u8],
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    text_index: i64,
    base_offset: i64,
) -> Result<Vec<u8>> {
    let words = strings.words(text_index)?;
    compose_native_church_words_at(frame, font, &words, base_offset)
}

/// ComposeNativeChurchTextWithNameAt expands the proven FFFC actor-name
/// control used by facility feedback strings such as FDTXT506.
///
/// Go：`native_church_scene.go:157`，`func ComposeNativeChurchTextWithNameAt( frame []byte, strings *fdtxt.Strings, font *fdtxt.Font, textIndex, nameTextIndex, baseOffset int, ) ([]byte, error)`
pub fn compose_native_church_text_with_name_at(
    frame: &mut [u8],
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    text_index: i64,
    name_text_index: i64,
    base_offset: i64,
) -> Result<Vec<u8>> {
    let words = strings.words(text_index)?;
    let name = strings.words(name_text_index)?;
    let mut expanded = Vec::with_capacity(words.len() + name.len());
    for &word in &words {
        if word == 0xfffc {
            expanded.extend_from_slice(&name);
            continue;
        }
        expanded.push(word);
    }
    compose_native_church_words_at(frame, font, &expanded, base_offset)
}

/// Go：`native_church_scene.go:182`，`func composeNativeChurchWordsAt( frame []byte, font *fdtxt.Font, words []uint16, baseOffset int, ) ([]byte, error)`
pub(crate) fn compose_native_church_words_at(
    frame: &mut [u8],
    font: &fdtxt::Font,
    words: &[u16],
    base_offset: i64,
) -> Result<Vec<u8>> {
    let (mut line, mut column) = (0i64, 0i64);
    let style = fdtxt::NativeGlyphStyle { foreground: 205, shadow: 76, background: 74 };
    for &word in words {
        if word == 0xfffe {
            line += 1;
            column = 0;
            continue;
        }
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!("campaign: unsupported church text control {}", battle::go_sharp_x(word as i64)));
        }
        let offset = base_offset
            .wrapping_add(line.wrapping_mul(19 * 320))
            .wrapping_add(column.wrapping_mul(fdtxt::GLYPH_WIDTH as i64));
        font.blit_native_glyph(frame, 320, offset, word as i64, style)?;
        column += 1;
    }
    // Go 就地改寫 frame 後回傳同一切片；Rust 改寫呼叫端的 frame 並回傳複本。
    Ok(frame.to_vec())
}

#[cfg(test)]
pub(crate) mod native_church_scene_test;
