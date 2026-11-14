//! Rust 補充（R4 E 組）：`draw_modern_battle_panel_ink` 的字級搜尋（Go `modern_hd_battle_panel.go:224`）。
//! Go 的對應測試都以真 TTF 繪字並斷言畫面（列在 R5 清單），這裡以假量測器驗證搜尋與兩個錯誤。

use super::*;

/// 每個字寬 `size / 2`、高 `size`，原點在基線左上。
struct FixedMeasure;

impl TextMeasure for FixedMeasure {
    fn ascent(&self, face: &FontFace) -> Option<f64> {
        Some(face.size)
    }
    fn bound_string(&self, face: &FontFace, s: &str) -> Rectangle {
        let w = s.chars().count() as i64 * face.size as i64 / 2;
        rgba_rect(0, -(face.size as i64), w, 0)
    }
    fn has_glyph(&self, _: &SfntFont, _: char) -> bool {
        true
    }
}

fn measured_font() -> Font {
    Font {
        sf: Some(Arc::new(SfntFont { data: Vec::new() })),
        base: 18.0,
        measure: TextMeasurer(Some(Arc::new(FixedMeasure))),
        ..Default::default()
    }
}

#[test]
fn panel_ink_picks_largest_fitting_size_without_drawing() {
    let mut f = measured_font();
    let mut canvas = new_rgba_canvas(894, 252);
    let before = canvas.pix.clone();
    // 30×6 寬 180、高 54 的框：字級 52 時兩字寬 52、高 52，加 2 恰好不超框。
    let rect = rgba_rect(0, 0, 180, 54);
    draw_modern_battle_panel_ink(&mut canvas, &mut f, "HP", rect, 96, ColorRgba::default()).unwrap();
    assert!(f.faces.contains_key(&52) && !f.faces.contains_key(&51), "應停在第一個不超框的字級 52");
    assert_eq!(canvas.pix, before, "R4 不畫字");
}

#[test]
fn panel_ink_reports_go_errors() {
    let mut canvas = new_rgba_canvas(894, 252);
    let mut f = measured_font();
    let err = draw_modern_battle_panel_ink(
        &mut canvas,
        &mut f,
        "很長的名字",
        rgba_rect(0, 0, 30, 20),
        48,
        ColorRgba::default(),
    )
    .unwrap_err();
    assert_eq!(err.message(), "modern HD panel: text exceeds safe rectangle: \"很長的名字\"");
    let mut no_ttf = Font { base: 18.0, ..Default::default() };
    let err = draw_modern_battle_panel_ink(
        &mut canvas,
        &mut no_ttf,
        "HP",
        rgba_rect(0, 0, 180, 54),
        48,
        ColorRgba::default(),
    )
    .unwrap_err();
    assert_eq!(err.message(), "modern HD panel: font unavailable");
}
