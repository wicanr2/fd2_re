//! Go 來源：`remake/internal/campaign/native_story_dialogue.go`（412 行）。

use super::*;

/// NativeStoryDialoguePortraitRect 回傳原版故事對話頭像實際覆蓋矩形。
/// 下框 0x9017 是 0x4E8E1 的右緣錨點，不能直接當成左上角。
///
/// Go：`native_story_dialogue.go:15`，`func NativeStoryDialoguePortraitRect(control string) (image.Rectangle, error)`
pub fn native_story_dialogue_portrait_rect(control: &str) -> Result<go_image::Rectangle> {
    match control {
        "FFEF" | "FFED" => Ok(go_rect(
            NATIVE_STORY_UPPER_PORTRAIT % 320,
            NATIVE_STORY_UPPER_PORTRAIT / 320,
            NATIVE_STORY_UPPER_PORTRAIT % 320 + 80,
            NATIVE_STORY_UPPER_PORTRAIT / 320 + 80,
        )),
        "FFEE" | "FFEC" => {
            let (right, y) = (NATIVE_STORY_LOWER_PORTRAIT % 320, NATIVE_STORY_LOWER_PORTRAIT / 320);
            Ok(go_rect(right - 79, y, right + 1, y + 80))
        }
        _ => Err(campaign_err!("campaign: native story dialogue control {} is unsupported", battle::go_quote(control))),
    }
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
    let (limit, ok) = native_dialogue_line_glyph_limit_inner(control);
    if !ok {
        return Err(campaign_err!(
            "campaign: native story dialogue control {} is unsupported",
            battle::go_quote(control)
        ));
    }
    let mut offset = NATIVE_STORY_LOWER_TEXT;
    if control == "FFEF" || control == "FFED" {
        offset = NATIVE_STORY_UPPER_TEXT;
    }
    Ok((offset % 320, offset / 320, limit * NATIVE_STORY_GLYPH_STEP, NATIVE_STORY_LINE_STEP, NATIVE_STORY_VISIBLE_ROWS))
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
    if stable.len() != 320 * 200 || cells.len() < 20 || !(0..=1).contains(&phase) {
        return Err(campaign_err!("campaign: native story wait arrow assets are invalid"));
    }
    layout.validate()?;
    let mut base = NATIVE_STORY_LOWER_TEXT;
    if layout.control == "FFEF" || layout.control == "FFED" {
        base = NATIVE_STORY_UPPER_TEXT;
    }
    let mut frame = stable.to_vec();
    cells[(18 + phase) as usize].blit_opaque_at_offset(&mut frame, 320, base + 0x47a0 + 0x640)?;
    Ok(frame)
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
    if background.len() != 320 * 200 || dialogue_cells.len() <= 17 {
        return Err(campaign_err!("campaign: native story dialogue opening assets are invalid"));
    }
    layout.validate().map_err(|err| err.wrap("campaign: "))?;
    let mut frame_y = NATIVE_STORY_LOWER_FRAME_Y;
    if layout.control == "FFEF" || layout.control == "FFED" {
        frame_y = NATIVE_STORY_UPPER_FRAME_Y;
    }
    let mut frames = Vec::with_capacity(NATIVE_STORY_OPENING_GRID_SIZES.len());
    for size in &NATIVE_STORY_OPENING_GRID_SIZES {
        let mut frame = background.to_vec();
        let placements = fdother::plan_native_dialogue_frame_grid(320, 5, frame_y, size[0], size[1])?;
        for placement in &placements {
            dialogue_cells[placement.resource_index as usize].blit_opaque_at_offset(
                &mut frame,
                320,
                placement.destination_byte,
            )?;
        }
        frames.push(frame);
    }
    Ok(frames)
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
    if background.len() != 320 * 200 || dialogue_cells.len() <= 17 {
        return Err(campaign_err!("campaign: native story dialogue closing assets are invalid"));
    }
    layout.validate().map_err(|err| err.wrap("campaign: "))?;
    let mut frame_y = NATIVE_STORY_LOWER_FRAME_Y;
    if layout.control == "FFEF" || layout.control == "FFED" {
        frame_y = NATIVE_STORY_UPPER_FRAME_Y;
    }
    let mut frames = Vec::with_capacity(NATIVE_STORY_CLOSING_GRID_SIZES.len() + 1);
    for size in &NATIVE_STORY_CLOSING_GRID_SIZES {
        let mut frame = background.to_vec();
        let placements = fdother::plan_native_dialogue_frame_grid(320, 5, frame_y, size[0], size[1])?;
        for placement in &placements {
            dialogue_cells[placement.resource_index as usize].blit_opaque_at_offset(
                &mut frame,
                320,
                placement.destination_byte,
            )?;
        }
        frames.push(frame);
    }
    frames.push(background.to_vec());
    if motion_target_y == 0 {
        return Ok(frames);
    }
    // Go `native_story_dialogue.go:154` 的 `len(dialogueCells) == 0` 分支在前面 `<= 17` 的檢查之後
    // 不可能成立，保留只為對照。
    if dialogue_cells.is_empty() {
        return Err(campaign_err!("campaign: native story dialogue closing motion has no cells"));
    }
    // 可見游標**可以是負的**。原版那三組視圖全域裡只有鏡頭與絕對游標被寫入端夾住；
    // 可見游標是自由的 dword，走行步進在鏡頭還能捲時照樣遞減，收據量到過 -1
    // （docs/data/ui-traces/fd2-story-pan-cursor-20260909.json）。滑動終點因此可能
    // 落在畫面外——那是合法狀態，不是錯誤。先前在這裡失敗即關閉，戰鬥中途只要
    // 有單位往左走得夠遠、接著觸發對白，整場就停在這裡。
    let total = visible_cursor_x.wrapping_add(visible_cursor_y);
    if total == 0 {
        return Ok(frames);
    }
    let (cursor_x, cursor_y) = (24 * visible_cursor_x + 4, 24 * visible_cursor_y + 4);
    // Go 的 `for step := 0; step <= total; step++`：total 為負時一次都不跑。
    let mut step = 0;
    while step <= total {
        let x = 5 - ((5 - cursor_x) * step) / total;
        let y = motion_target_y - ((motion_target_y - cursor_y) * step) / total;
        let mut frame = background.to_vec();
        dialogue_cells[0].blit_at_clipped(&mut frame, 320, x, y)?;
        frames.push(frame);
        step += 1;
    }
    Ok(frames)
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
    let frames = compose_native_story_dialogue_progressive_frames(
        background,
        dialogue_cells,
        portrait,
        font,
        Some(glyph_index),
        layout,
        page,
    )?;
    Ok(frames.last().cloned().unwrap_or_default())
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
    if stable_page.len() != 320 * 200 {
        return Err(campaign_err!("campaign: native story dialogue mouth base is invalid"));
    }
    layout.validate().map_err(|err| err.wrap("campaign: "))?;
    let mut portrait_offset = NATIVE_STORY_LOWER_PORTRAIT;
    if layout.control == "FFEF" || layout.control == "FFED" {
        portrait_offset = NATIVE_STORY_UPPER_PORTRAIT;
    }
    let mut frame = stable_page.to_vec();
    blit_native_dialogue_portrait_at(&mut frame, portrait, portrait_offset)?;
    Ok(frame)
}

/// ComposeNativeStoryDialogueProgressiveFrames 保存 0x15F84 每寫入一個普通
/// glyph 才前進目的位址的發布順序。第0張只有完整框與頭像；後續每張各新增一個
/// glyph。幀數不代表 DOS wall-clock，只是 caller 可決定性消費的順序契約。
/// NativeStoryDialogueGlyphSteps 分辨普通字形與捲動中間幀，供逐字計數器使用。
///
/// Go：`native_story_dialogue.go:227`，`func NativeStoryDialogueGlyphSteps(layout *NativeDialogueLayout, page int) ([]bool, error)`
pub fn native_story_dialogue_glyph_steps(layout: &NativeDialogueLayout, page: i64) -> Result<Vec<bool>> {
    layout.validate()?;
    if page < 0 || page as u64 >= layout.pages.len() as u64 {
        return Err(campaign_err!("native dialogue page unavailable"));
    }
    let mut steps = vec![false];
    let prior_rows: i64 = layout.pages[..page as usize].iter().map(|rows| rows.len() as i64).sum();
    for (row, text) in layout.pages[page as usize].iter().enumerate() {
        if prior_rows + row as i64 >= NATIVE_STORY_VISIBLE_ROWS {
            steps.extend(std::iter::repeat_n(false, NATIVE_STORY_SCROLL_FRAMES as usize));
        }
        let tokens = layout.glyph_tokens(page, row as i64, text)?;
        steps.extend(std::iter::repeat_n(true, tokens.len()));
    }
    Ok(steps)
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
    let (Some(font), Some(glyph_index)) = (font, glyph_index) else {
        return Err(campaign_err!("campaign: native story dialogue assets are invalid"));
    };
    if background.len() != 320 * 200 || dialogue_cells.len() <= 17 {
        return Err(campaign_err!("campaign: native story dialogue assets are invalid"));
    }
    layout.validate().map_err(|err| err.wrap("campaign: "))?;
    if page < 0 || page as u64 >= layout.pages.len() as u64 {
        return Err(campaign_err!("campaign: native story dialogue page {page} is unavailable"));
    }
    if page > 0 {
        // FFFD 只暫停輸入；前文與行號仍由同一句擁有。重建完整前綴，
        // 但只發布上一頁終點起的新畫格，不重新消費前文的字形時鐘。
        let mut prefix = layout.clone();
        prefix.pages = vec![Vec::new()];
        prefix.glyph_pages = Vec::new();
        if !layout.glyph_pages.is_empty() {
            prefix.glyph_pages = vec![Vec::new()];
        }
        prefix.glyph_ids = Vec::new();
        if !layout.glyph_ids.is_empty() {
            prefix.glyph_ids = vec![Vec::new()];
        }
        let mut offset = 0usize;
        for i in 0..=page as usize {
            prefix.pages[0].extend(layout.pages[i].iter().cloned());
            if !prefix.glyph_pages.is_empty() {
                prefix.glyph_pages[0].extend(layout.glyph_pages[i].iter().cloned());
            }
            if !prefix.glyph_ids.is_empty() {
                prefix.glyph_ids[0].extend(layout.glyph_ids[i].iter().cloned());
            }
            if i < page as usize {
                let steps = native_story_dialogue_glyph_steps(layout, i as i64)?;
                offset += steps.len() - 1;
            }
        }
        let frames = compose_native_story_dialogue_progressive_frames(
            background,
            dialogue_cells,
            portrait,
            Some(font),
            Some(glyph_index),
            &prefix,
            0,
        )?;
        return Ok(frames[offset..].to_vec());
    }
    let (line_glyph_limit, ok) = native_dialogue_line_glyph_limit_inner(&layout.control);
    if !ok {
        return Err(campaign_err!(
            "campaign: native story dialogue control {} is unsupported",
            battle::go_quote(&layout.control)
        ));
    }
    let mut frame = compose_native_story_dialogue_base_frame(background, dialogue_cells, portrait, layout)?;
    let mut text_offset = NATIVE_STORY_LOWER_TEXT;
    if layout.control == "FFEF" || layout.control == "FFED" {
        text_offset = NATIVE_STORY_UPPER_TEXT;
    }
    let page_rows = &layout.pages[page as usize];
    let mut frames: Vec<Vec<u8>> =
        Vec::with_capacity(1 + (line_glyph_limit as usize) * page_rows.len() + NATIVE_STORY_SCROLL_FRAMES as usize);
    frames.push(frame.clone());
    let style = fdtxt::NativeGlyphStyle { foreground: 0xcd, shadow: 0x4c, background: 0x4a };
    for (row, text) in page_rows.iter().enumerate() {
        let mut visible_row = row as i64;
        if row as i64 >= NATIVE_STORY_VISIBLE_ROWS {
            // sub_16E24 的完整逐像素時鐘未納入本切片；直接指令只固定
            // 第三列後先捲動19px並把logical line減一。沿用重製既有
            // 10幀近似，且每幀都只改框內三列文字窗口。
            let (text_x, text_y) = (text_offset % 320, text_offset / 320);
            let window_x = text_x - 1; // 包含 0x4EA2A 左下 shadow
            const WINDOW_W: usize = 208;
            const WINDOW_H: i64 = 72;
            // 0x16E24：五次3px再一次4px；每次複製72列，僅清理
            // text base+0x5A00 的208 bytes。來源不是「三行字」矩形。
            let shifts = [3i64, 3, 3, 3, 3, 4];
            let mut pass = 0usize;
            for step in 1..=NATIVE_STORY_SCROLL_FRAMES {
                let mut next = frame.clone();
                while (pass as i64) < (shifts.len() as i64) * step / NATIVE_STORY_SCROLL_FRAMES {
                    for y in 0..WINDOW_H {
                        let dst = ((text_y + y) * 320 + window_x) as usize;
                        let src = dst + (shifts[pass] * 320) as usize;
                        next.copy_within(src..src + WINDOW_W, dst);
                    }
                    let clear_at = (text_offset + 0x5a00) as usize;
                    next[clear_at..clear_at + WINDOW_W].fill(style.background);
                    pass += 1;
                }
                frames.push(next.clone());
                frame = next;
            }
            visible_row = NATIVE_STORY_VISIBLE_ROWS - 1;
        }
        let tokens = layout.glyph_tokens(page, row as i64, text).map_err(|err| err.wrap("campaign: "))?;
        if tokens.is_empty() || tokens.len() as i64 > line_glyph_limit {
            return Err(campaign_err!("campaign: native story dialogue row {row} has {} glyphs", tokens.len()));
        }
        for (column, token) in tokens.iter().enumerate() {
            let found = glyph_index.get(token).copied();
            let mut glyph = found.unwrap_or(0);
            if found.is_some() && !layout.glyph_ids.is_empty() {
                glyph = layout.glyph_ids[page as usize][row][column];
            }
            if found.is_none() || glyph < 0 || glyph as u64 >= font.glyph_count() as u64 {
                return Err(campaign_err!(
                    "campaign: native story dialogue glyph {} is unavailable",
                    battle::go_quote(token)
                ));
            }
            let destination =
                text_offset + visible_row * NATIVE_STORY_LINE_STEP * 320 + column as i64 * NATIVE_STORY_GLYPH_STEP;
            font.blit_native_glyph(&mut frame, 320, destination, glyph, style)?;
            frames.push(frame.clone());
        }
    }
    Ok(frames)
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
    if background.len() != 320 * 200 || dialogue_cells.len() <= 17 {
        return Err(campaign_err!("campaign: native story dialogue base assets are invalid"));
    }
    layout.validate().map_err(|err| err.wrap("campaign: "))?;
    let (mut frame_y, mut portrait_offset) = (NATIVE_STORY_LOWER_FRAME_Y, NATIVE_STORY_LOWER_PORTRAIT);
    if layout.control == "FFEF" || layout.control == "FFED" {
        (frame_y, portrait_offset) = (NATIVE_STORY_UPPER_FRAME_Y, NATIVE_STORY_UPPER_PORTRAIT);
    }
    let mut frame = background.to_vec();
    let placements = fdother::plan_native_dialogue_frame_grid(320, 5, frame_y, 19, 5)?;
    for placement in &placements {
        dialogue_cells[placement.resource_index as usize].blit_opaque_at_offset(
            &mut frame,
            320,
            placement.destination_byte,
        )?;
    }
    blit_native_dialogue_portrait_at(&mut frame, portrait, portrait_offset)?;
    Ok(frame)
}

/// Go `image.Rect(x0, y0, x1, y1)`：兩軸各自把較小值放到 `Min`。`go_image.rs` 只有欄位，Go
/// `image` 的方法由用到的組各自補；這裡用自由函式，避免和其他組替同一型別補的方法衝突。
pub(crate) fn go_rect(x0: i64, y0: i64, x1: i64, y1: i64) -> go_image::Rectangle {
    let (x0, x1) = if x0 > x1 { (x1, x0) } else { (x0, x1) };
    let (y0, y1) = if y0 > y1 { (y1, y0) } else { (y0, y1) };
    go_image::Rectangle { min: go_image::Point { x: x0, y: y0 }, max: go_image::Point { x: x1, y: y1 } }
}

/// Go `Rectangle.Empty`：`Min.X >= Max.X || Min.Y >= Max.Y`。
pub(crate) fn go_rect_empty(r: go_image::Rectangle) -> bool {
    r.min.x >= r.max.x || r.min.y >= r.max.y
}

/// Go `Rectangle.Union`：一方為空時回傳另一方，否則取兩者的外接矩形。
pub(crate) fn go_rect_union(r: go_image::Rectangle, s: go_image::Rectangle) -> go_image::Rectangle {
    if go_rect_empty(r) {
        return s;
    }
    if go_rect_empty(s) {
        return r;
    }
    go_image::Rectangle {
        min: go_image::Point { x: r.min.x.min(s.min.x), y: r.min.y.min(s.min.y) },
        max: go_image::Point { x: r.max.x.max(s.max.x), y: r.max.y.max(s.max.y) },
    }
}

// 測試：Go `remake/internal/campaign/native_story_dialogue_test.go` 逐條移植。
#[cfg(test)]
mod native_story_dialogue_test;

// 測試：Go `remake/internal/campaign/native_story_portrait_rect_test.go` 逐條移植。
#[cfg(test)]
mod native_story_portrait_rect_test;
