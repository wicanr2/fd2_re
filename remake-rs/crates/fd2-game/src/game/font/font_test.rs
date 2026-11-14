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
