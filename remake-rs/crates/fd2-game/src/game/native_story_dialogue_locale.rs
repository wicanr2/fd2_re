//! Go 來源：`remake/cmd/fd2/native_story_dialogue_locale.go`（221 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;
use crate::go_x_image_font::AlphaImage;

const LOCALIZED_DIALOGUE_SCALE: f64 = 0.78;

/// Reflow only display text. The source and native speaker/control operands remain authoritative.
pub(crate) fn localized_native_dialogue_layout(
    text: &str,
    source: &campaign::NativeDialogueLayout,
    font: Option<&mut Font>,
) -> Result<campaign::NativeDialogueLayout> {
    let font = font
        .filter(|font| font.sf.is_some())
        .ok_or_else(|| game_err!("localized native dialogue font is unavailable"))?;
    if text.trim().is_empty() {
        return Err(game_err!("localized native dialogue text is empty"));
    }
    let (limit, _) = campaign::native_dialogue_line_glyph_limit(&source.control);
    let (_, _, max_width, _, visible_rows) = campaign::native_story_dialogue_text_geometry(&source.control)?;
    let mut width = |line: String| {
        if line.chars().count() > limit as usize { f64::INFINITY } else { font.width(&line, LOCALIZED_DIALOGUE_SCALE) }
    };
    let rows = wrap_text_by_width(text, max_width as f64, &mut width);
    if rows.iter().any(|line| line.is_empty() || width(line.clone()) > max_width as f64) {
        return Err(game_err!("localized native dialogue text does not fit {}", source.control));
    }
    let layout = campaign::NativeDialogueLayout {
        source_dat: source.source_dat.clone(),
        string_index: source.string_index,
        utterance: source.utterance,
        control: source.control.clone(),
        operand: source.operand,
        pages: rows.chunks(visible_rows as usize).map(<[String]>::to_vec).collect(),
        ..Default::default()
    };
    layout.validate()?;
    Ok(layout)
}

pub(crate) fn compose_localized_native_dialogue_progressive_frames(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    font: Option<&mut Font>,
    layout: &campaign::NativeDialogueLayout,
    page: usize,
) -> Result<Vec<Vec<u8>>> {
    let font = font.ok_or_else(|| game_err!("localized native dialogue font is unavailable"))?;
    let rows = layout.pages.get(page).ok_or_else(|| game_err!("localized native dialogue page is unavailable"))?;
    let mut base = campaign::compose_native_story_dialogue_base_frame(background, dialogue_cells, portrait, layout)?;
    let (x, y, width, line_step, _) = campaign::native_story_dialogue_text_geometry(&layout.control)?;
    let mut frames = vec![base.clone()];
    for (index, row) in rows.iter().enumerate() {
        for end in row.char_indices().skip(1).map(|(offset, _)| offset).chain(std::iter::once(row.len())) {
            let mut frame = base.clone();
            draw_indexed_localized_text(
                &mut frame,
                Some(font),
                &row[..end],
                x,
                y + index as i64 * line_step,
                width,
                line_step,
                LOCALIZED_DIALOGUE_SCALE,
                0xcd,
                0x4c,
            )?;
            frames.push(frame);
        }
        base.clone_from(frames.last().unwrap());
    }
    Ok(frames)
}

/// Go：`native_story_dialogue_locale.go:160`，`func drawIndexedLocalizedText( dst []byte, displayFont *Font, text string, x, y, maxWidth, height int, scale float64, foreground, shadow byte, ) error`
///
/// 非繁中語系把 TTF 字形點陣化寫進索引色畫格：`font.Drawer` 畫進 alpha 遮罩，A≥32 的像素先在右下一格寫陰影、
/// 再寫前景。點陣化照 `crate::go_x_image_font`（字形位置、kern、dot 與 Go 相同，覆蓋率不同，KNOWN_DIFFERENCES
/// `ttf-raster-coverage`）。`displayFont.indexedHD` 有值時只記錄 HD 文字列（與 Go 相同，不畫像素）。
pub(crate) fn draw_indexed_localized_text(
    dst: &mut [u8],
    display_font: Option<&mut Font>,
    text: &str,
    x: i64,
    y: i64,
    max_width: i64,
    height: i64,
    scale: f64,
    foreground: u8,
    shadow: u8,
) -> Result<()> {
    let Some(display_font) = display_font else {
        return Err(game_err!("localized native dialogue draw bounds are invalid"));
    };
    if dst.len() != 320 * 200 || text.is_empty() || x < 0 || y < 0 || x + max_width > 320 || y + height > 200 {
        return Err(game_err!("localized native dialogue draw bounds are invalid"));
    }
    if scale <= 0.0 || display_font.sf.is_none() || display_font.width(text, scale) > max_width as f64 {
        return Err(game_err!("localized native dialogue line {} exceeds safe width", battle::go_quote(text)));
    }
    let measure = display_font.measure.0.clone();
    let sf = display_font.sf.clone();
    for r in text.chars() {
        // Go `displayFont.sf.GlyphIndex`；`sf` 為 nil 時 Go 會在上面的 `Width` 回 0 後於這裡解參考 panic。
        let sf = sf.as_ref().expect("drawIndexedLocalizedText: displayFont.sf is nil");
        let measure = measure.as_ref().expect("Width 已經過 faceFor，量測器已注入");
        if !measure.has_glyph(sf, r) {
            return Err(game_err!("localized native dialogue font lacks {}", go_quote_rune(r)));
        }
    }
    // Go 的文字列存 `font: displayFont`（指標）；Rust 存複本，複本不帶 HD 擷取狀態（只用到字型本身）。
    let run_font = display_font.indexed_hd.is_some().then(|| Font { indexed_hd: None, ..display_font.clone() });
    let px = (display_font.base * display_font.effective_scale(scale)).round() as i64;
    if let (Some(run_font), Some(capture)) = (run_font, display_font.indexed_hd.as_mut()) {
        let clip = rect(x * 2, y * 2, (x + max_width) * 2, (y + height) * 2);
        let pixels = (px * 2) as f64;
        for (offset, index) in [(1i64, shadow), (0, foreground)] {
            let color = modern_hd_native_text_color(&capture.palette, index);
            capture.runs.push(ModernHDTextRun {
                font: Some(run_font.clone()),
                text: text.to_string(),
                x: (x * 2) as f64 + offset as f64 / 3.0,
                y: (y * 2) as f64 + offset as f64 / 3.0,
                pixels,
                color,
                clip,
                ..Default::default()
            });
        }
        return Ok(());
    }
    let (face, ascent) = display_font.face_for(px);
    let Some(face) = face else {
        return Err(game_err!("localized native dialogue font face is unavailable"));
    };
    let mut mask = AlphaImage::new(max_width, height);
    // Go `fixed.P(0, int(ascent))`。
    let dot = (0, ((ascent as i64) << 6) as i32);
    measure.as_ref().expect("face_for 成功時已注入 TextMeasure").draw_alpha(&face, &mut mask, dot, text);
    let alpha = |px: i64, py: i64| mask.pix[(py * max_width + px) as usize];
    for py in 0..height {
        for px in 0..max_width {
            if alpha(px, py) < 32 {
                continue;
            }
            if px + 1 < max_width && py + 1 < height {
                dst[((y + py + 1) * 320 + x + px + 1) as usize] = shadow;
            }
        }
    }
    for py in 0..height {
        for px in 0..max_width {
            if alpha(px, py) >= 32 {
                dst[((y + py) * 320 + x + px) as usize] = foreground;
            }
        }
    }
    Ok(())
}

/// Go `fmt` 的 `%q` 用在 rune（`strconv.QuoteRune`）：單引號包住，跳脫規則同 `battle::go_quote`。
fn go_quote_rune(r: char) -> String {
    let quoted = battle::go_quote(&r.to_string());
    let inner = &quoted[1..quoted.len() - 1];
    match r {
        '\'' => String::from("'\\''"),
        '"' => String::from("'\"'"),
        _ => format!("'{inner}'"),
    }
}
