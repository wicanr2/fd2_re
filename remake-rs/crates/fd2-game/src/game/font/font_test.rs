//! Go 來源：`remake/cmd/fd2/text_wrap_segments_test.go`、`font_test.go`、`font_scale_test.go`。
//!
//! 需要真字型量寬的 Go 測試（`TestLocalizedDialogueWrapKeepsEnglishWordsWhole`、
//! `TestFontUserScaleChangesMeasuredWidth`、`TestLocaleDialogueCorpusNeverSplitsWords`）列在
//! `PORTING.md`「R5 測試清單」：量寬由外殼的 `TextMeasure` 提供。

use super::*;

/// Go：`text_wrap_segments_test.go:10` `TestTextSegmentsKeepsLatinWordsWhole`
#[test]
fn test_text_segments_keeps_latin_words_whole() {
    let cases: [(&str, &[&str]); 5] = [
        ("Indexed text", &["Indexed ", "text"]),
        ("翻訳された会話", &["翻", "訳", "さ", "れ", "た", "会", "話"]),
        ("Indexed 日本語 text", &["Indexed ", "日", "本", "語 ", "text"]),
        ("HP 120/240", &["HP ", "120/240"]),
        ("やあ、Zephyr。", &["や", "あ", "、", "Zephyr", "。"]),
    ];
    for (text, want) in cases {
        let got = text_segments(text);
        assert_eq!(got.concat(), text, "{text:?} 切完接不回原文");
        assert_eq!(got, want, "{text:?}");
    }
}

fn rune_width(s: String) -> f64 {
    s.chars().count() as f64
}

/// Go：`text_wrap_segments_test.go:38` `TestWrapTextByWidthDoesNotSplitLatinWords`
#[test]
fn test_wrap_text_by_width_does_not_split_latin_words() {
    let source = "alpha beta gamma delta";
    let got = wrap_text_by_width(source, 12.0, &mut rune_width);
    for line in &got {
        for word in line.split_whitespace() {
            assert!(source.contains(word), "行 {line:?} 裡的 {word:?} 不是完整單字");
        }
    }
    assert_eq!(got.concat(), source, "斷行後文字改變");

    let got = wrap_text_by_width("一二三四五六", 2.0, &mut rune_width);
    assert_eq!(got.len(), 3, "中文兩字一行應切成 3 行：{got:?}");

    let got = wrap_text_by_width("supercalifragilistic", 6.0, &mut rune_width);
    for line in &got {
        assert!(rune_width(line.clone()) <= 6.0, "硬拆之後 {line:?} 仍超過寬度");
    }
    assert_eq!(got.concat(), "supercalifragilistic", "硬拆後文字改變");
}

/// Go：`font_test.go:9` `TestFontWrapPreservesCJKTextWithinWidth`
#[test]
fn test_font_wrap_preserves_cjk_text_within_width() {
    let text = "未能以天空之鑰開啟通往空中大陸的道路，炎龍騎士團的遠征在巨塔前告終。";
    let mut width = |line: String| (line.chars().count() * 18) as f64;
    let lines = wrap_text_by_width(text, 180.0, &mut width);
    assert!(lines.len() >= 2 && lines.concat() == text, "wrapped ending lost text: {lines:?}");
    for line in &lines {
        let got = width(line.clone());
        assert!(got <= 180.0, "wrapped line width {got:.1} exceeds limit: {line:?}");
    }
}

/// Go：`font_scale_test.go:34` `TestFontUserScaleMultipliesAndClamps`（nil 接收者分支不移植）
#[test]
fn test_font_user_scale_multiplies_and_clamps() {
    let mut f = Font { base: 18.0, ..Default::default() };
    assert_eq!(f.effective_scale(1.2), 1.2, "沒設倍率時應原樣回傳");
    f.set_user_scale(0.8);
    assert!((f.effective_scale(1.0) - 0.8).abs() <= 1e-9);
    assert!((f.effective_scale(1.25) - 1.0).abs() <= 1e-9, "倍率要乘在呼叫端的 scale 上");
    f.set_user_scale(0.1);
    assert_eq!(f.user_scale, 0.6, "過小的倍率沒有被夾住");
    f.set_user_scale(9.0);
    assert_eq!(f.user_scale, 1.6, "過大的倍率沒有被夾住");
}

/// 固定寬度的量測器：每個字元 `size / 2` 像素、ascent 為 `size * 0.8` 取整。
struct FixedMeasure;

impl TextMeasure for FixedMeasure {
    fn ascent(&self, face: &FontFace) -> Option<f64> {
        Some((face.size * 0.8).round())
    }
    fn bound_string(&self, face: &FontFace, s: &str) -> Rectangle {
        let w = s.chars().count() as i64 * (face.size as i64 / 2);
        Rectangle { min: Point { x: 0, y: -(face.size as i64) }, max: Point { x: w, y: 0 } }
    }
    fn has_glyph(&self, _font: &SfntFont, r: char) -> bool {
        r != '\u{fffd}'
    }
}

#[test]
fn font_width_uses_injected_measure_and_caches_faces() {
    let mut f = Font {
        sf: Some(Arc::new(SfntFont { data: b"OTTO".to_vec() })),
        base: 18.0,
        measure: TextMeasurer(Some(Arc::new(FixedMeasure))),
        ..Default::default()
    };
    assert_eq!(f.width("abcd", 1.0), 36.0);
    f.set_user_scale(0.6);
    // round(18 × 0.6 × 0.5) = 5 → 夾到 6 px
    assert_eq!(f.width("ab", 0.5), 6.0);
    assert_eq!(f.face_for(2).1, 5.0);
    assert!(f.faces.contains_key(&6) && f.faces.contains_key(&18));
    // 沒有字型：照 Go 回 0，不呼叫量測器。
    let mut none = Font { base: 18.0, ..Default::default() };
    assert_eq!(none.width("abc", 1.0), 0.0);
    assert!(none.wrap("a b", 1.0, 10.0) == ["a b"]);
}

#[test]
fn rune_breakable_follows_go_script_tables() {
    for r in ['漢', 'あ', 'ア', '한', '、', 'Ａ', '〇', '々'] {
        assert!(rune_breakable(r as i32), "{r:?}");
    }
    // U+3006 不在 Han（Go 表的 0x3005..0x3007 stride 2），但落在全形標點區 0x3000..0x303F。
    assert!(rune_breakable(0x3006));
    for r in ['A', '1', ' ', 'é'] {
        assert!(!rune_breakable(r as i32), "{r:?}");
    }
    assert!(!rune_breakable(-1));
}

// ---- R5 r5f：真 TTF（`NotoSansCJK-Regular.ttc`，Go 黃金同一檔） ----

use crate::go_x_image_font::{Rect26_6, XImageFace, XImageFont};
use crate::test_support::{NOTO_SANS_CJK_REGULAR_PATH, cjk_font_available};

fn real_font() -> Option<Font> {
    if !cjk_font_available() {
        return None;
    }
    Some(load_font().expect("Go fontPaths 的系統 Noto CJK 已在，load_font 應成功"))
}

fn real_x_image_font() -> Option<XImageFont> {
    if !cjk_font_available() {
        return None;
    }
    Some(XImageFont::parse(Arc::new(std::fs::read(NOTO_SANS_CJK_REGULAR_PATH).unwrap())).expect("解析 TTC"))
}

/// 期望值是 2026-11-15 以 `fd2-go-test-local`（x/image v0.12.0）對同一檔列印的 `Face.Metrics`、`Kern`、
/// `GlyphBounds`（26.6，`r5f` 探針，見 PORTING.md「R5 階段 2：r5f」）。
#[test]
fn x_image_metrics_kern_and_bounds_match_go_probe() {
    let Some(font) = real_x_image_font() else { return };
    assert_eq!(font.units_per_em(), 1000);
    assert_eq!(font.num_glyphs(), 65535);
    for (px, height, ascent, descent) in [(6, 9, 7, 2), (18, 27, 21, 6), (22, 32, 26, 7), (54, 79, 63, 16)] {
        let m = XImageFace::new(&font, px as f64, 72.0).metrics();
        assert_eq!((m.height, m.ascent, m.descent), (height * 64, ascent * 64, descent * 64), "px {px}");
    }
    let face = XImageFace::new(&font, 22.0, 72.0);
    assert_eq!(face.kern('"', ','), -128);
    assert_eq!(face.kern('"', 'A'), -64);
    assert_eq!(face.kern('\'', ','), -128);
    for (a, b) in [('索', '爾'), ('攻', '擊'), ('１', '２'), ('H', 'P'), ('，', '。')] {
        assert_eq!(face.kern(a, b), 0, "{a}{b}");
    }
    let r = |min_x, min_y, max_x, max_y| Rect26_6 { min_x, min_y, max_x, max_y };
    for (c, want, adv) in [
        ('索', r(62, -1184, 1325, 115), 1408),
        ('爾', r(46, -1098, 1359, 113), 1408),
        ('A', r(6, -1032, 850, 0), 832),
        ('1', r(124, -1032, 690, 0), 768),
        ('，', r(190, -331, 465, 151), 1408),
    ] {
        assert_eq!(face.glyph_bounds(c), (want, adv, true), "{c}");
    }
    assert_eq!(font.fractional_points(), 0, "字型單位出現小數座標，x/image 會取整");
}

/// Go：`font_scale_test.go:99` `TestFontUserScaleChangesMeasuredWidth`（Go 沒字型時略過；Rust 照
/// `FD2_REQUIRE_ORIGINAL` 決定略過或失敗）。
#[test]
fn test_font_user_scale_changes_measured_width() {
    let Some(mut f) = real_font() else { return };
    const SAMPLE: &str = "The bandits split up and head for the village.";
    let base = f.width(SAMPLE, 1.0);
    assert!(base > 0.0, "量到的寬度是 {base:.1}");
    f.set_user_scale(0.8);
    let smaller = f.width(SAMPLE, 1.0);
    f.set_user_scale(1.25);
    let larger = f.width(SAMPLE, 1.0);
    assert!(smaller < base && base < larger, "字級沒有改變寬度：0.8→{smaller:.1}、1.0→{base:.1}、1.25→{larger:.1}");
    assert!(smaller <= base * 0.92, "0.8 只省了 {:.1}%，太少", (1.0 - smaller / base) * 100.0);
}

/// Go：`native_story_dialogue_locale_test.go:97` `TestDrawIndexedLocalizedLineUsesNativePaletteAndRejectsMissingGlyph`。
/// `drawIndexedLocalizedLine` 是 `r8`，它只以 `localizedNativeDialogueFontScale`（0.78）與 0xcd／0x4c 呼叫
/// `drawIndexedLocalizedText`，這裡直接展開。
#[test]
fn test_draw_indexed_localized_line_uses_native_palette_and_rejects_missing_glyph() {
    let Some(mut font) = real_font() else { return };
    let line = |frame: &mut Vec<u8>, font: &mut Font, text: &str| {
        draw_indexed_localized_text(frame, Some(font), text, 16, 16, 224, 19, 0.78, 0xcd, 0x4c)
    };
    let mut frame = vec![0u8; 320 * 200];
    line(&mut frame, &mut font, "Test日本語").unwrap();
    let foreground = frame.iter().filter(|&&p| p == 0xcd).count();
    let shadow = frame.iter().filter(|&&p| p == 0x4c).count();
    assert!(foreground > 0 && shadow > 0, "indexed ink missing: foreground={foreground} shadow={shadow}");
    let err = line(&mut frame, &mut font, "\u{10ffff}").expect_err("missing glyph was accepted");
    assert_eq!(err.message(), "localized native dialogue font lacks '\u{10ffff}'");
}

/// Go：`native_shop_locale_test.go:74` `TestOfficialBattleNamesFitNativePanelRectangle`。
#[test]
fn test_official_battle_names_fit_native_panel_rectangle() {
    let Some(mut display_font) = real_font() else { return };
    if crate::test_support::remake_asset("locales").is_none() {
        return;
    }
    for locale_id in ["zh-Hans", "ja", "en"] {
        let catalog = load_official_locale_entities(locale_id).unwrap().unwrap();
        let mut count = 0;
        for raw_id in 0..=138 {
            let Ok(name) = catalog.battle_name(raw_id) else { continue };
            count += 1;
            assert!(
                display_font.width(&name, LOCALIZED_BATTLE_PANEL_NAME_SCALE)
                    <= LOCALIZED_BATTLE_PANEL_NAME_WIDTH as f64,
                "{locale_id} battle name {raw_id} {name:?} exceeds panel rectangle"
            );
            draw_indexed_localized_text(
                &mut vec![0u8; 320 * 200],
                Some(&mut display_font),
                &name,
                5,
                4,
                LOCALIZED_BATTLE_PANEL_NAME_WIDTH,
                LOCALIZED_BATTLE_PANEL_NAME_HEIGHT,
                LOCALIZED_BATTLE_PANEL_NAME_SCALE,
                0xcd,
                0x4c,
            )
            .unwrap_or_else(|e| panic!("{locale_id} battle name {raw_id} {name:?}: {}", e.message()));
        }
        assert_eq!(count, 94, "{locale_id} battle name count");
    }
}

/// Go：`native_shop_locale_test.go:109` `TestOfficialCommandNamesFitNativeGridCell`。
#[test]
fn test_official_command_names_fit_native_grid_cell() {
    let Some(display_font) = real_font() else { return };
    if crate::test_support::remake_asset("locales").is_none() {
        return;
    }
    for locale_id in ["zh-Hant", "zh-Hans", "ja", "en"] {
        let catalog = load_official_locale_entities(locale_id).unwrap().unwrap();
        let mut g = Game {
            locale_id: locale_id.to_string(),
            locale_entities: Some(catalog),
            font: Some(display_font.clone()),
            ..Default::default()
        };
        let mut count = 0;
        for command_id in (0..=35).filter(|&id| id != 31) {
            let (name, scale) = g
                .localized_native_command_label(command_id)
                .unwrap_or_else(|e| panic!("{locale_id} command {command_id}: {}", e.message()));
            count += 1;
            // Go 以 `localizedNativeCommandNameWidth`（140）驗格子寬度。
            let width = g.font.as_mut().unwrap().width(&name, scale);
            assert!(
                width <= 140.0 && scale >= LOCALIZED_NATIVE_COMMAND_MIN_SCALE,
                "{locale_id} command {command_id} {name:?} does not fit at scale {scale:.3}"
            );
        }
        assert_eq!(count, 35, "{locale_id} command count");
    }
}

/// `Font::draw` 的位置：字畫在 `(round(x), round(y)+ascent)` 的基線上，覆蓋範圍不超出 `BoundString`
/// 平移後的外框，顏色照 `ColorScale.ScaleWithColor`（不透明處等於 `clr`）。
#[test]
fn font_draw_places_glyphs_inside_bound_string() {
    let Some(mut font) = real_font() else { return };
    let clr = ColorRgba { r: 0xff, g: 0xf0, b: 0xb4, a: 0xff };
    let mut dst = Picture::new_image(640, 400);
    let text = "索爾的攻擊 HP-12";
    let (x, y, scale) = (8.0, 370.0, 1.2);
    assert_eq!(font.draw(&mut dst, text, x, y, scale, clr), y);
    let px = (font.base * font.effective_scale(scale)).round() as i64;
    let (face, asc) = font.face_for(px);
    let face = face.unwrap();
    assert_eq!((px, asc), (22, 26.0));
    let b = font.measure.0.clone().unwrap().bound_string(&face, text);
    let (ox, oy) = (x.round() as i64, y.round() as i64 + asc as i64);
    let (mut min_x, mut min_y, mut max_x, mut max_y) = (i64::MAX, i64::MAX, i64::MIN, i64::MIN);
    let mut opaque = 0;
    for py in 0..400 {
        for qx in 0..640 {
            let p = &dst.pix[((py * 640 + qx) * 4) as usize..][..4];
            if p[3] == 0 {
                continue;
            }
            (min_x, min_y, max_x, max_y) = (min_x.min(qx), min_y.min(py), max_x.max(qx + 1), max_y.max(py + 1));
            if p[3] == 0xff {
                opaque += 1;
                assert_eq!(p, [clr.r, clr.g, clr.b, clr.a]);
            }
        }
    }
    assert!(opaque > 0);
    assert!(min_x >= b.min.x + ox && min_y >= b.min.y + oy && max_x <= b.max.x + ox && max_y <= b.max.y + oy);
}
#[test]
fn localized_native_dialogue_reflows_without_changing_source_identity() {
    if !crate::test_support::cjk_font_available() {
        return;
    }
    let mut font = load_font().unwrap();
    let source = campaign::NativeDialogueLayout {
        source_dat: "FDTXT_033".into(),
        string_index: 12,
        utterance: 1,
        control: "FFEE".into(),
        operand: 48,
        ..Default::default()
    };
    let text = "The captain entered the palace. ".repeat(12);
    let output = localized_native_dialogue_layout(&text, &source, Some(&mut font)).unwrap();
    assert_eq!(output.source_dat, source.source_dat);
    assert_eq!((output.string_index, output.utterance, output.operand), (12, 1, 48));
    assert_eq!(output.control, source.control);
    assert!(output.pages.len() > 1);
    assert_eq!(output.pages.into_iter().flatten().collect::<String>(), text);
}
