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
    todo!("campaign/native_church_scene.go:ComposeNativeChurchScene")
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
    todo!("campaign/native_church_scene.go:ComposeNativeChurchSceneBase")
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
    todo!("campaign/native_church_scene.go:ComposeNativeChurchDialogueOverlayAt")
}

/// Go：`native_church_scene.go:122`，`func blitNativeDialoguePortraitAt(frame []byte, portrait dato.Frame, offset int) error`
pub(crate) fn blit_native_dialogue_portrait_at(frame: &mut [u8], portrait: &dato::Frame, offset: i64) -> Result<()> {
    todo!("campaign/native_church_scene.go:blitNativeDialoguePortraitAt")
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
    todo!("campaign/native_church_scene.go:ComposeNativeChurchTextAt")
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
    todo!("campaign/native_church_scene.go:ComposeNativeChurchTextWithNameAt")
}

/// Go：`native_church_scene.go:182`，`func composeNativeChurchWordsAt( frame []byte, font *fdtxt.Font, words []uint16, baseOffset int, ) ([]byte, error)`
pub(crate) fn compose_native_church_words_at(
    frame: &mut [u8],
    font: &fdtxt::Font,
    words: &[u16],
    base_offset: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_church_scene.go:composeNativeChurchWordsAt")
}
