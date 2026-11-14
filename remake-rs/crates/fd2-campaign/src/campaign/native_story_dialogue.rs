//! Go 來源：`remake/internal/campaign/native_story_dialogue.go`（412 行）。

use super::*;

/// NativeStoryDialoguePortraitRect 回傳原版故事對話頭像實際覆蓋矩形。
/// 下框 0x9017 是 0x4E8E1 的右緣錨點，不能直接當成左上角。
///
/// Go：`native_story_dialogue.go:15`，`func NativeStoryDialoguePortraitRect(control string) (image.Rectangle, error)`
pub fn native_story_dialogue_portrait_rect(control: &str) -> Result<go_image::Rectangle> {
    todo!("campaign/native_story_dialogue.go:NativeStoryDialoguePortraitRect")
}

pub(crate) const NATIVE_STORY_UPPER_FRAME_Y: i64 = 2;

pub(crate) const NATIVE_STORY_LOWER_FRAME_Y: i64 = 112;

pub(crate) const NATIVE_STORY_UPPER_PORTRAIT: i64 = 0x728;

pub(crate) const NATIVE_STORY_LOWER_PORTRAIT: i64 = 0x9017;

pub(crate) const NATIVE_STORY_UPPER_TEXT: i64 = 0xb4f;

pub(crate) const NATIVE_STORY_LOWER_TEXT: i64 = 0x951f;

pub(crate) const NATIVE_STORY_GLYPH_STEP: i64 = 16;

pub(crate) const NATIVE_STORY_LINE_STEP: i64 = 19;

pub(crate) const NATIVE_STORY_VISIBLE_ROWS: i64 = 3;

pub(crate) const NATIVE_STORY_SCROLL_FRAMES: i64 = 10;

/// NativeStoryDialogueTextGeometry 回傳原版索引畫面的文字起點與安全窗口。
/// 多語 renderer 可以替換字形像素，但不可更動 control 所決定的畫面座標。
///
/// Go：`native_story_dialogue.go:43`，`func NativeStoryDialogueTextGeometry(control string) (x, y, width, lineStep, visibleRows int, err error)`
/// Go 具名回傳值依序為：x、y、width、lineStep、visibleRows、err。
pub fn native_story_dialogue_text_geometry(control: &str) -> Result<(i64, i64, i64, i64, i64)> {
    todo!("campaign/native_story_dialogue.go:NativeStoryDialogueTextGeometry")
}

/// Go：`native_story_dialogue.go:56`
pub(crate) static NATIVE_STORY_OPENING_GRID_SIZES: [[i64; 2]; 5] = [[4, 2], [8, 3], [12, 4], [16, 5], [19, 5]];

/// Go：`native_story_dialogue.go:57`
pub(crate) static NATIVE_STORY_CLOSING_GRID_SIZES: [[i64; 2]; 4] = [[16, 5], [12, 4], [8, 3], [4, 2]];

/// ComposeNativeStoryDialogueWaitArrow 使用已載入 FDFIELD 的 0x16C57 非零
/// composition 分支。只供故事地圖的 FFFD 等待頁；不推測無地圖 caller。
///
/// Go：`native_story_dialogue.go:61`，`func ComposeNativeStoryDialogueWaitArrow(stable []byte, cells []fdother.RawCell, layout *NativeDialogueLayout, phase int) ([]byte, error)`
pub fn compose_native_story_dialogue_wait_arrow(
    stable: &[u8],
    cells: &[fdother::RawCell],
    layout: &NativeDialogueLayout,
    phase: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_story_dialogue.go:ComposeNativeStoryDialogueWaitArrow")
}

/// ComposeNativeStoryDialogueOpeningFrames 保存 sub_165AC 五次 sub_168B6 的
/// columns/rows 順序。這些幀只有 FDOTHER #5 格網；portrait 與文字由後續
/// progressive frame0 接手，避免猜測中間 portrait 時序。
///
/// Go：`native_story_dialogue.go:82`，`func ComposeNativeStoryDialogueOpeningFrames( background []byte, dialogueCells []fdother.RawCell, layout *NativeDialogueLayout, ) ([][]byte, error)`
pub fn compose_native_story_dialogue_opening_frames(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    layout: &NativeDialogueLayout,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_story_dialogue.go:ComposeNativeStoryDialogueOpeningFrames")
}

/// ComposeNativeStoryDialogueClosingFrames 重建 sub_16B43 對 sub_165AC 五張
/// framebuffer snapshot 的實際 restore 結果。前四張不是猜測性倒播，而是
/// snapshot[4..1] 所保存的16×5、12×4、8×3、4×2狀態；第五張是原背景。
/// motionTargetY 非零時，再依原函式的整數公式疊上 FDOTHER #5 entry0。
///
/// Go：`native_story_dialogue.go:118`，`func ComposeNativeStoryDialogueClosingFrames( background []byte, dialogueCells []fdother.RawCell, layout *NativeDialogueLayout, motionTargetY, visibleCursorX, visibleCursorY int, ) ([][]byte, error)`
pub fn compose_native_story_dialogue_closing_frames(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    layout: &NativeDialogueLayout,
    motion_target_y: i64,
    visible_cursor_x: i64,
    visible_cursor_y: i64,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_story_dialogue.go:ComposeNativeStoryDialogueClosingFrames")
}

/// ComposeNativeStoryDialoguePage 使用已證實的 FDOTHER #5 格網、DATO 頭像、
/// FDOTHER #4 字模與具型別 FFFE／FFFD 投影，建立一頁穩定的 0x15F84 故事畫面。
/// 開關框插值與逐字時序刻意不納入這個穩定頁 compositor。
///
/// Go：`native_story_dialogue.go:180`，`func ComposeNativeStoryDialoguePage( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, font *fdtxt.Font, glyphIndex map[string]int, layout *NativeDialogueLayout, page int, ) ([]byte, error)`
pub fn compose_native_story_dialogue_page(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    font: Option<&fdtxt::Font>,
    glyph_index: &BTreeMap<String, i64>,
    layout: &NativeDialogueLayout,
    page: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_story_dialogue.go:ComposeNativeStoryDialoguePage")
}

/// ComposeNativeStoryDialogueMouthFrame 對應 sub_16559：保留穩定 indexed page，
/// 再把所選 DATO frame 貼到 caller-owned 上／下框頭像 anchor。caller 必須先驗證
/// DATO 四幀資源邊界，才可傳入 frame3。
///
/// Go：`native_story_dialogue.go:201`，`func ComposeNativeStoryDialogueMouthFrame( stablePage []byte, portrait dato.Frame, layout *NativeDialogueLayout, ) ([]byte, error)`
pub fn compose_native_story_dialogue_mouth_frame(
    stable_page: &[u8],
    portrait: &dato::Frame,
    layout: &NativeDialogueLayout,
) -> Result<Vec<u8>> {
    todo!("campaign/native_story_dialogue.go:ComposeNativeStoryDialogueMouthFrame")
}

/// ComposeNativeStoryDialogueProgressiveFrames 保存 0x15F84 每寫入一個普通
/// glyph 才前進目的位址的發布順序。第0張只有完整框與頭像；後續每張各新增一個
/// glyph。幀數不代表 DOS wall-clock，只是 caller 可決定性消費的順序契約。
/// NativeStoryDialogueGlyphSteps 分辨普通字形與捲動中間幀，供逐字計數器使用。
///
/// Go：`native_story_dialogue.go:227`，`func NativeStoryDialogueGlyphSteps(layout *NativeDialogueLayout, page int) ([]bool, error)`
pub fn native_story_dialogue_glyph_steps(layout: &NativeDialogueLayout, page: i64) -> Result<Vec<bool>> {
    todo!("campaign/native_story_dialogue.go:NativeStoryDialogueGlyphSteps")
}

/// Go：`native_story_dialogue.go:254`，`func ComposeNativeStoryDialogueProgressiveFrames( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, font *fdtxt.Font, glyphIndex map[string]int, layout *NativeDialogueLayout, page int, ) ([][]byte, error)`
pub fn compose_native_story_dialogue_progressive_frames(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    font: Option<&fdtxt::Font>,
    glyph_index: Option<&BTreeMap<String, i64>>,
    layout: &NativeDialogueLayout,
    page: i64,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_story_dialogue.go:ComposeNativeStoryDialogueProgressiveFrames")
}

/// ComposeNativeStoryDialogueBaseFrame 建立完全展開且含閉嘴頭像的索引畫面，
/// 但不寫入文字；原版 FDTXT 與多語 TTF 逐字 renderer 共用此底圖。
///
/// Go：`native_story_dialogue.go:382`，`func ComposeNativeStoryDialogueBaseFrame( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, layout *NativeDialogueLayout, ) ([]byte, error)`
pub fn compose_native_story_dialogue_base_frame(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    layout: &NativeDialogueLayout,
) -> Result<Vec<u8>> {
    todo!("campaign/native_story_dialogue.go:ComposeNativeStoryDialogueBaseFrame")
}
