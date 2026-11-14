//! Go `remake/internal/campaign/native_story_dialogue_test.go` 的逐條移植。
//!
//! Go 測試以相對路徑找輸入：`../../assets` 是 `FD2_REMAKE_ASSETS`，`../../../extracted/raw` 是
//! `FD2_EXTRACTED_RAW_DIR`，`../../../docs` 由 `docs_file` 定位，`../../../org_game/.../FLAME2` 是
//! `FD2_FLAME2_DIR`。Go 的 `t.Skip` 對應 `test_support` 的略過（`FD2_REQUIRE_ORIGINAL=1` 時失敗）。
//!
//! 多數 Go 測試各自重複一段 glyph_map 讀取與「依 FDTXT 原始控制字重建版面」的比對；Rust 把共同
//! 部分抽成本檔的輔助函式，每個測試的輸入、範圍、遞迴與總數仍逐一照 Go。

use super::*;
use crate::test_support::{docs_file, input, remake_asset};

/// Go `fmt.Sscanf(key, "%d", &index)` 成功時的值：略過前導空白，可帶正負號，至少一位十進位數字，
/// 之後的字元不影響成功。
fn sscanf_decimal(key: &str) -> Option<i64> {
    let body = key.trim_start_matches([' ', '\t', '\r']);
    let bytes = body.as_bytes();
    let mut end = usize::from(matches!(bytes.first(), Some(b'+' | b'-')));
    let digits_from = end;
    while end < bytes.len() && bytes[end].is_ascii_digit() {
        end += 1;
    }
    if end == digits_from {
        return None;
    }
    body[..end].parse().ok()
}

/// Go `loadNativeDialogueGlyphMap`：`docs/data/glyph_map.json` 的字模索引 → 字元。
fn load_native_dialogue_glyph_map() -> Option<BTreeMap<u16, String>> {
    let path = docs_file("data/glyph_map.json")?;
    let raw = std::fs::read(&path).unwrap_or_else(|e| panic!("讀 {path}：{e}"));
    let encoded: serde_json::Map<String, serde_json::Value> = serde_json::from_slice(&raw).unwrap();
    let mut glyphs = BTreeMap::new();
    for (key, value) in &encoded {
        if key == "_comment" {
            continue;
        }
        let text = value.as_str().unwrap_or_else(|| panic!("glyph map {key} 不是字串"));
        let index = sscanf_decimal(key);
        let Some(index) = index.filter(|i| (0..=0xffff).contains(i)) else {
            panic!("invalid glyph map key {}", battle::go_quote(key));
        };
        glyphs.insert(index as u16, text.to_string());
    }
    Some(glyphs)
}

/// Go `loadCanonicalGlyphIndex`：`assets/fonts/unicode_to_glyph.json` 的字元 → 正規字模。
fn load_canonical_glyph_index() -> Option<BTreeMap<String, i64>> {
    let path = remake_asset("fonts/unicode_to_glyph.json")?;
    let raw = std::fs::read(&path).unwrap_or_else(|e| panic!("讀 {path}：{e}"));
    let encoded: serde_json::Map<String, serde_json::Value> = serde_json::from_slice(&raw).unwrap();
    let mut index = BTreeMap::new();
    for (key, value) in &encoded {
        if key == "_comment" {
            continue;
        }
        index.insert(key.clone(), value.as_i64().unwrap_or_else(|| panic!("glyph {key} 不是整數")));
    }
    Some(index)
}

/// Go `decodeOriginalNativeDialogueLayouts`：依原始 FDTXT 控制字重建每句的版面。
fn decode_original_native_dialogue_layouts(
    source: &str,
    string_index: i64,
    words: &[u16],
    glyphs: &BTreeMap<u16, String>,
    canonical: &BTreeMap<String, i64>,
) -> Result<Vec<NativeDialogueLayout>> {
    let is_speaker = |word: u16| (0xffec..=0xffef).contains(&word);
    let mut layouts: Vec<NativeDialogueLayout> = Vec::new();
    let mut cursor = 0usize;
    while cursor < words.len() {
        if !is_speaker(words[cursor]) {
            cursor += 1;
            continue;
        }
        if cursor + 1 >= words.len() {
            return Err(campaign_err!("speaker control at end of FDTXT string"));
        }
        let mut layout = NativeDialogueLayout {
            source_dat: source.to_string(),
            string_index,
            utterance: layouts.len() as i64,
            control: format!("{:04X}", words[cursor]),
            operand: words[cursor + 1] as i64,
            ..Default::default()
        };
        cursor += 2;
        let mut pages: Vec<Vec<String>> = Vec::new();
        let mut glyph_pages: Vec<Vec<Vec<String>>> = Vec::new();
        let mut id_pages: Vec<Vec<Vec<i64>>> = Vec::new();
        let mut page: Vec<String> = Vec::new();
        let mut glyph_page: Vec<Vec<String>> = Vec::new();
        let mut id_page: Vec<Vec<i64>> = Vec::new();
        let mut row: Vec<String> = Vec::new();
        let mut id_row: Vec<i64> = Vec::new();
        let mut non_canonical = false;
        macro_rules! flush_row {
            () => {
                if !row.is_empty() {
                    page.push(row.concat());
                    glyph_page.push(std::mem::take(&mut row));
                    id_page.push(std::mem::take(&mut id_row));
                }
            };
        }
        macro_rules! flush_page {
            () => {
                flush_row!();
                if !page.is_empty() {
                    pages.push(std::mem::take(&mut page));
                    glyph_pages.push(std::mem::take(&mut glyph_page));
                    id_pages.push(std::mem::take(&mut id_page));
                }
            };
        }
        while cursor < words.len() && !is_speaker(words[cursor]) {
            let word = words[cursor];
            cursor += 1;
            match word {
                fdtxt::STRING_END => cursor = words.len(),
                0xfffe => {
                    flush_row!();
                }
                0xfffd => {
                    flush_page!();
                }
                _ => {
                    let Some(text) = glyphs.get(&word) else {
                        return Err(campaign_err!("FDTXT glyph {:#x} is absent from glyph_map", word));
                    };
                    row.push(text.clone());
                    id_row.push(word as i64);
                    if canonical.get(text) != Some(&(word as i64)) {
                        non_canonical = true;
                    }
                }
            }
        }
        flush_page!();
        layout.pages = pages;
        let multi_rune = glyph_pages.iter().flatten().flatten().any(|token| token.chars().count() != 1);
        if multi_rune {
            layout.glyph_pages = glyph_pages;
        }
        if non_canonical {
            layout.glyph_ids = id_pages;
        }
        layout.validate()?;
        layouts.push(layout);
    }
    Ok(layouts)
}

/// 讀 `extracted/raw/FDTXT/<source>.bin` 並解析；缺檔時略過（Go `t.Skip`）。
fn load_fdtxt(source: &str) -> Option<fdtxt::Strings> {
    let path = input("FD2_EXTRACTED_RAW_DIR", &format!("FDTXT/{source}.bin"))?;
    let raw = std::fs::read(&path).unwrap();
    Some(fdtxt::parse(&raw).unwrap())
}

/// `CompileHandlerBinding("../../assets/cutscenes/bindings/<name>.json")`，要求沒有問題。
fn compile_binding(name: &str) -> Option<Vec<Beat>> {
    let path = remake_asset(&format!("cutscenes/bindings/{name}.json"))?;
    let (beats, issues) =
        compile_handler_binding(&path).unwrap_or_else(|err| panic!("compile {name} binding: issues=[] err={err}"));
    assert!(issues.is_empty(), "compile {name} binding: issues={issues:?} err=<nil>");
    Some(beats)
}

/// 收集 dialog 拍的原生版面，依 FDTXT 字串索引分組。`recursive` 照 Go 是否走進 then／else。
fn collect_by_string(beats: &[Beat], recursive: bool, out: &mut BTreeMap<i64, Vec<NativeDialogueLayout>>) {
    for beat in beats {
        if beat.op == "dialog"
            && let Some(layout) = &beat.native_dialogue
        {
            out.entry(layout.string_index).or_default().push(layout.clone());
        }
        if recursive {
            collect_by_string(&beat.then, recursive, out);
            collect_by_string(&beat.r#else, recursive, out);
        }
    }
}

/// 頂層 dialog 拍的原生版面，依出現順序。
fn collect_flat(beats: &[Beat]) -> Vec<NativeDialogueLayout> {
    beats.iter().filter(|beat| beat.op == "dialog").filter_map(|beat| beat.native_dialogue.clone()).collect()
}

/// 綁定檔的原生版面依字串索引比對原始控制字；回傳全部 utterance 數。
fn check_by_string(binding: &str, source: &str, indices: &[i64], recursive: bool) -> Option<usize> {
    let strings_table = load_fdtxt(source)?;
    let glyphs = load_native_dialogue_glyph_map()?;
    let canonical = load_canonical_glyph_index()?;
    let beats = compile_binding(binding)?;
    let mut got = BTreeMap::new();
    collect_by_string(&beats, recursive, &mut got);
    let mut total = 0;
    for &string_index in indices {
        let words = strings_table.words(string_index).unwrap();
        let want = decode_original_native_dialogue_layouts(source, string_index, &words, &glyphs, &canonical).unwrap();
        let layouts = got.get(&string_index).cloned().unwrap_or_default();
        assert_eq!(
            layouts.len(),
            want.len(),
            "{source} index{string_index} layouts={} want={}",
            layouts.len(),
            want.len()
        );
        for (i, w) in want.iter().enumerate() {
            assert_eq!(&layouts[i], w, "{source} index{string_index} utterance{i}");
        }
        total += want.len();
    }
    Some(total)
}

/// 綁定檔頂層全部 dialog 拍的原生版面比對單一字串索引；回傳拍數。
fn check_flat(binding: &str, source: &str, string_index: i64) -> Option<(usize, usize)> {
    let strings_table = load_fdtxt(source)?;
    let glyphs = load_native_dialogue_glyph_map()?;
    let canonical = load_canonical_glyph_index()?;
    let beats = compile_binding(binding)?;
    let got = collect_flat(&beats);
    let words = strings_table.words(string_index).unwrap();
    let want = decode_original_native_dialogue_layouts(source, string_index, &words, &glyphs, &canonical).unwrap();
    assert_eq!(got, want, "{source} index{string_index} layouts={} want={}", got.len(), want.len());
    Some((got.len(), want.len()))
}

/// Go：`TestCh24NativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch24_native_dialogue_layouts_match_original_control_words() {
    check_by_string("ch24_post", "FDTXT_025", &[6, 7], false);
}

/// Go：`TestCh01PreNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch01_pre_native_dialogue_layouts_match_original_control_words() {
    check_by_string("ch01_pre", "FDTXT_002", &[0, 1, 2, 3], false);
}

/// Go：`TestCh00PreNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch00_pre_native_dialogue_layouts_match_original_control_words() {
    let Some(glyphs) = load_native_dialogue_glyph_map() else { return };
    let Some(canonical) = load_canonical_glyph_index() else { return };
    let Some(beats) = compile_binding("ch00_pre") else { return };
    let mut got: BTreeMap<(String, i64), Vec<NativeDialogueLayout>> = BTreeMap::new();
    for beat in &beats {
        if beat.op == "dialog"
            && let Some(layout) = &beat.native_dialogue
        {
            got.entry((layout.source_dat.clone(), layout.string_index)).or_default().push(layout.clone());
        }
    }
    let mut total = 0;
    for (source, count) in [("FDTXT_033", 6), ("FDTXT_032", 10), ("FDTXT_001", 3)] {
        let Some(strings_table) = load_fdtxt(source) else { return };
        for string_index in 0..count {
            let words = strings_table.words(string_index).unwrap();
            let want =
                decode_original_native_dialogue_layouts(source, string_index, &words, &glyphs, &canonical).unwrap();
            let layouts = got.get(&(source.to_string(), string_index)).cloned().unwrap_or_default();
            assert_eq!(
                layouts.len(),
                want.len(),
                "{source} index{string_index} layouts={}, want {}",
                layouts.len(),
                want.len()
            );
            for (i, w) in want.iter().enumerate() {
                assert_eq!(&layouts[i], w, "{source} index{string_index} utterance{i}");
            }
            total += want.len();
        }
    }
    assert_eq!(total, 97, "ch00 native dialogue utterances={total}, want 97");
}

/// Go：`TestCh00PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch00_post_native_dialogue_layouts_match_original_control_words() {
    if let Some((got, want)) = check_flat("ch00_post", "FDTXT_001", 9) {
        assert_eq!(got, 13, "FDTXT_001 index9 layouts={got} want={want}");
    }
}

/// Go：`TestCh22PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch22_post_native_dialogue_layouts_match_original_control_words() {
    let indices: Vec<i64> = (8..=17).collect();
    if let Some(total) = check_by_string("ch22_post", "FDTXT_023", &indices, true) {
        assert_eq!(total, 56, "ch22_post native dialogue utterances={total}, want 56");
    }
}

/// Go：`TestCh17PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch17_post_native_dialogue_layouts_match_original_control_words() {
    if let Some(total) = check_by_string("ch17_post", "FDTXT_018", &[7, 8, 9, 10], false) {
        assert_eq!(total, 21, "ch17_post native dialogue utterances={total}, want 21");
    }
}

/// Go：`TestCh12PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch12_post_native_dialogue_layouts_match_original_control_words() {
    if let Some((got, want)) = check_flat("ch12_post", "FDTXT_013", 9) {
        assert_eq!(got, 12, "FDTXT_013 index9 layouts={got} want={want}");
    }
}

/// Go：`TestCh19PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch19_post_native_dialogue_layouts_match_original_control_words() {
    let indices: Vec<i64> = (11..=16).collect();
    if let Some(total) = check_by_string("ch19_post", "FDTXT_020", &indices, true) {
        assert_eq!(total, 29, "ch19_post native dialogue utterances={total}, want 29");
    }
}

/// Go：`TestCh07PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch07_post_native_dialogue_layouts_match_original_control_words() {
    if let Some(total) = check_by_string("ch07_post", "FDTXT_008", &[3, 4], false) {
        assert_eq!(total, 8, "ch07_post native dialogue utterances={total}, want 8");
    }
}

/// Go：`TestCh06PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch06_post_native_dialogue_layouts_match_original_control_words() {
    let Some(strings_table) = load_fdtxt("FDTXT_007") else { return };
    let Some(glyphs) = load_native_dialogue_glyph_map() else { return };
    let Some(canonical) = load_canonical_glyph_index() else { return };
    let Some(beats) = compile_binding("ch06_post") else { return };
    let mut got: BTreeMap<i64, Vec<NativeDialogueLayout>> = BTreeMap::new();
    let mut seen: BTreeSet<String> = BTreeSet::new();
    let mut callers: BTreeSet<String> = BTreeSet::new();
    fn collect(
        items: &[Beat],
        got: &mut BTreeMap<i64, Vec<NativeDialogueLayout>>,
        seen: &mut BTreeSet<String>,
        callers: &mut BTreeSet<String>,
    ) {
        for beat in items {
            if beat.op == "dialog"
                && let Some(layout) = &beat.native_dialogue
            {
                let caller = format!("{}#{}", beat.source, layout.string_index);
                callers.insert(caller.clone());
                let key = format!("{caller}#{}", layout.utterance);
                if seen.insert(key) {
                    got.entry(layout.string_index).or_default().push(layout.clone());
                }
            }
            collect(&beat.then, got, seen, callers);
            collect(&beat.r#else, got, seen, callers);
        }
    }
    collect(&beats, &mut got, &mut seen, &mut callers);
    let mut total = 0;
    for string_index in 4..=5 {
        let words = strings_table.words(string_index).unwrap();
        let want =
            decode_original_native_dialogue_layouts("FDTXT_007", string_index, &words, &glyphs, &canonical).unwrap();
        let layouts = got.get(&string_index).cloned().unwrap_or_default();
        assert_eq!(layouts, want, "FDTXT_007 index{string_index} layouts={} want={}", layouts.len(), want.len());
        total += want.len();
    }
    assert!(
        callers.len() == 2 && seen.len() == 12 && total == 12,
        "ch06_post native dialogue callers={} layouts={} utterances={total}, want 2/12/12",
        callers.len(),
        seen.len()
    );
}

/// Go：`TestCh15PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch15_post_native_dialogue_layouts_match_original_control_words() {
    if let Some(total) = check_by_string("ch15_post", "FDTXT_016", &[2, 3, 4], true) {
        assert_eq!(total, 23, "ch15_post native dialogue utterances={total}, want 23");
    }
}

/// Go：`TestCh16PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch16_post_native_dialogue_layouts_match_original_control_words() {
    if let Some(total) = check_by_string("ch16_post", "FDTXT_017", &[5, 7, 6, 8], true) {
        assert_eq!(total, 26, "ch16_post native dialogue utterances={total}, want 26");
    }
}

/// Go：`TestCh09PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch09_post_native_dialogue_layouts_match_original_control_words() {
    if let Some(total) = check_by_string("ch09_post", "FDTXT_010", &[4, 5], false) {
        assert_eq!(total, 35, "ch09_post native dialogue utterances={total}, want 35");
    }
}

/// Go：`TestCh05PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch05_post_native_dialogue_layouts_match_original_control_words() {
    if let Some((got, want)) = check_flat("ch05_post", "FDTXT_006", 6) {
        assert_eq!(got, 19, "FDTXT_006 index6 layouts={got} want={want}");
    }
}

/// Go：`TestCh21PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch21_post_native_dialogue_layouts_match_original_control_words() {
    if let Some(total) = check_by_string("ch21_post", "FDTXT_022", &[4, 5, 6], false) {
        assert_eq!(total, 11, "ch21_post native dialogue utterances={total}, want 11");
    }
}

/// Go：`TestCh23PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch23_post_native_dialogue_layouts_match_original_control_words() {
    if let Some(total) = check_by_string("ch23_post", "FDTXT_024", &[2, 3], false) {
        assert_eq!(total, 11, "ch23_post native dialogue utterances={total}, want 11");
    }
}

/// Go：`TestCh20PostGroupedNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch20_post_grouped_native_dialogue_layouts_match_original_control_words() {
    let Some(strings_table) = load_fdtxt("FDTXT_021") else { return };
    let Some(glyphs) = load_native_dialogue_glyph_map() else { return };
    let Some(canonical) = load_canonical_glyph_index() else { return };
    let Some(path) = remake_asset("scenarios/campaign_full.json") else { return };
    let c = load(&path).unwrap();
    let targets = [
        ("story_ch21_post_sky_key_intro", 1, "0x24182", 5),
        ("story_ch21_post_sky_key_crafted", 0, "0x2424b", 7),
        ("story_ch21_post_sky_key_crafted", 2, "0x24286", 8),
        ("story_ch21_post_sky_key_crafted", 4, "0x242c1", 9),
        ("story_ch21_post_sky_key_crafted", 6, "0x24308", 10),
        ("story_ch21_post_sky_key_insufficient", 0, "0x24308", 6),
    ];
    let mut total = 0;
    for (node, beat_index, caller, text_index) in targets {
        let beat = &c.nodes[node].beats[beat_index];
        assert!(
            beat.op == "dialog" && beat.source == caller && beat.native_dialogue.is_none(),
            "{node} beat{beat_index} caller/native shape drifted: {beat:?}"
        );
        let words = strings_table.words(text_index).unwrap();
        let want =
            decode_original_native_dialogue_layouts("FDTXT_021", text_index, &words, &glyphs, &canonical).unwrap();
        assert!(
            beat.native_dialogues == want,
            "{node} beat{beat_index} FDTXT_021 index{text_index} 原生版面與raw control不一致"
        );
        total += want.len();
    }
    assert_eq!(total, 30, "ch20 post native utterances={total}, want 30");
}

/// Go：`TestCh25PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch25_post_native_dialogue_layouts_match_original_control_words() {
    let indices: Vec<i64> = (5..=11).collect();
    if let Some(total) = check_by_string("ch25_post", "FDTXT_026", &indices, true) {
        assert_eq!(total, 41, "ch25_post native dialogue utterances={total}, want 41");
    }
}

/// Go：`TestCh27PostNativeDialogueLayoutsMatchOriginalControlWords`
#[test]
fn ch27_post_native_dialogue_layouts_match_original_control_words() {
    if let Some((got, want)) = check_flat("ch27_post", "FDTXT_028", 7) {
        assert_eq!(got, 5, "FDTXT_028 index7 layouts={got} want={want}");
    }
}

/// Go：`TestNativeDialogueLayoutUsesControlSpecificOriginalLineLimits`
#[test]
fn native_dialogue_layout_uses_control_specific_original_line_limits() {
    let limits = [("FFEC", 13), ("FFED", 15), ("FFEE", 14), ("FFEF", 14)];
    for (control, limit) in limits {
        let tokens: Vec<String> = vec!["甲".to_string(); limit];
        let mut layout = NativeDialogueLayout {
            source_dat: "FDTXT_TEST".into(),
            control: control.into(),
            pages: vec![vec![tokens.concat()]],
            glyph_pages: vec![vec![tokens.clone()]],
            ..Default::default()
        };
        if let Err(err) = layout.validate() {
            panic!("{control}: exact original limit {limit} rejected: {err}");
        }
        layout.pages[0][0].push('乙');
        layout.glyph_pages[0][0].push("乙".into());
        assert!(layout.validate().is_err(), "control {control} accepted {} glyphs above limit {limit}", limit + 1);
    }
}

/// Go：`TestNativeDialogueLayoutPreservesMultiRuneOriginalGlyphTokens`
#[test]
fn native_dialogue_layout_preserves_multi_rune_original_glyph_tokens() {
    let tokens = ["『", "AS", "R-", "07", "，", "妳", "的", "系", "統", "真", "的", "受", "損"];
    let mut layout = NativeDialogueLayout {
        source_dat: "FDTXT_029".into(),
        string_index: 12,
        control: "FFEF".into(),
        operand: 126,
        pages: vec![vec!["『ASR-07，妳的系統真的受損".into()]],
        glyph_pages: vec![vec![tokens.iter().map(|t| t.to_string()).collect()]],
        ..Default::default()
    };
    if let Err(err) = layout.validate() {
        panic!("13 raw glyphs expanded to 16 Unicode runes were rejected: {err}");
    }
    let got = layout.glyph_tokens(0, 0, &layout.pages[0][0]);
    assert!(matches!(&got, Ok(t) if t.len() == 13), "multi-rune glyph tokens={got:?}");
    layout.glyph_pages[0][0][1] = "A".into();
    assert!(layout.validate().is_err(), "glyph tokens that no longer join to authored text were accepted");
}

/// Go：`TestComposeNativeStoryDialoguePageUsesOriginalIndexedAssets`
#[test]
#[ignore = "等 B 合併"]
fn compose_native_story_dialogue_page_uses_original_indexed_assets() {
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let Some(dato_path) = input("FD2_FLAME2_DIR", "DATO.DAT") else { return };
    let resource5 = fdother::read_resource(&fdother_path, 5).unwrap();
    let cells: Vec<fdother::RawCell> =
        (0..=19).map(|index| fdother::parse_lmi1_raw_entry(&resource5, index).unwrap()).collect();
    let font_raw = fdother::read_resource(&fdother_path, 4).unwrap();
    let font = fdtxt::parse_font(&font_raw).unwrap();
    let portraits = dato::decode_resource(&dato_path, 26);
    let portraits = match portraits {
        Ok(p) if p.len() >= 4 => p,
        other => panic!("DATO#26 portrait: {other:?}"),
    };
    let Some(index) = load_canonical_glyph_index() else { return };
    // glyph_map 一字多模：FDTXT_008#2 的「．．」是字模 584（兩點），正規表把「．」查成 347
    // （一點）。GlyphIDs 必須蓋過 token 查表，照原始 word 畫。
    {
        let background = vec![0u8; 320 * 200];
        let mut dots = NativeDialogueLayout {
            source_dat: "FDTXT_008".into(),
            string_index: 2,
            control: "FFED".into(),
            operand: 16,
            pages: vec![vec!["『公主殿下．．".into()]],
            ..Default::default()
        };
        let by_token =
            compose_native_story_dialogue_page(&background, &cells, &portraits[0], Some(&font), &index, &dots, 0)
                .unwrap();
        let dot = index["．"];
        dots.glyph_ids = vec![vec![vec![557, 776, 881, 1104, 559, dot, dot]]];
        let canonical =
            compose_native_story_dialogue_page(&background, &cells, &portraits[0], Some(&font), &index, &dots, 0)
                .unwrap();
        assert!(canonical == by_token, "GlyphIDs 與正規字模相同時畫面卻不同");
        (dots.glyph_ids[0][0][5], dots.glyph_ids[0][0][6]) = (584, 584);
        let raw = compose_native_story_dialogue_page(&background, &cells, &portraits[0], Some(&font), &index, &dots, 0)
            .unwrap();
        assert!(raw != by_token, "GlyphIDs 584 沒有蓋過「．」的正規字模 347");
        dots.glyph_ids[0][0].truncate(6);
        assert!(dots.validate().is_err(), "GlyphIDs 與 token 數不同卻通過驗證");
    }
    let layout = NativeDialogueLayout {
        source_dat: "FDTXT_025".into(),
        string_index: 6,
        utterance: 0,
        control: "FFED".into(),
        operand: 16,
        pages: vec![
            vec![
                "『真是驚險的一戰！約拿老頭".into(),
                "　，你從哪裡找來這些生力軍".into(),
                "　？如果沒有他們，這一戰的".into(),
            ],
            vec!["　結果就會改觀了。』".into()],
        ],
        ..Default::default()
    };
    let background = vec![7u8; 320 * 200];
    let page0 = compose_native_story_dialogue_page(&background, &cells, &portraits[0], Some(&font), &index, &layout, 0)
        .unwrap();
    let page1 = compose_native_story_dialogue_page(&background, &cells, &portraits[0], Some(&font), &index, &layout, 1)
        .unwrap();
    let upper_text = NATIVE_STORY_UPPER_TEXT as usize;
    assert!(
        page0.len() == 320 * 200 && page1.len() == 320 * 200 && page0[0] == 7 && page0[upper_text] != 7,
        "native story page did not preserve source/outside frame or render text"
    );
    assert!(page0 != page1, "two FFFD pages produced an identical indexed frame");
    assert_eq!(background[upper_text], 7, "native story compositor mutated its caller-owned background");
    let page0_before_mouth = page0.clone();
    let mouth = compose_native_story_dialogue_mouth_frame(&page0, &portraits[3], &layout).unwrap();
    assert!(mouth.len() == 320 * 200 && mouth != page0, "DATO frame3 did not produce a distinct indexed mouth frame");
    assert!(page0 == page0_before_mouth, "mouth compositor mutated its caller-owned source");
    let progressive = compose_native_story_dialogue_progressive_frames(
        &background,
        &cells,
        &portraits[0],
        Some(&font),
        Some(&index),
        &layout,
        1,
    )
    .unwrap();
    let mut want_frames = 1 + NATIVE_STORY_SCROLL_FRAMES as usize;
    for row in &layout.pages[1] {
        want_frames += row.chars().count();
    }
    assert_eq!(
        progressive.len(),
        want_frames,
        "progressive frames={}, want frame0 plus {} glyphs",
        progressive.len(),
        want_frames - 1
    );
    assert!(progressive[progressive.len() - 1] == page1, "progressive final frame differs from stable page");
    assert!(
        progressive[0] != progressive[progressive.len() - 1],
        "progressive sequence did not publish any visible glyph"
    );
    assert!(progressive[0] == page0, "FFFD 後清空了前頁內容");
    // The HD text planner must preserve every original glyph and scroll tick,
    // including the retained previous page. Compare visible ink to the existing
    // indexed compositor rather than asserting the planner against itself.
    let blank_font = fdtxt::parse_font(&vec![0u8; font_raw.len()]).unwrap();
    let blank_frames = compose_native_story_dialogue_progressive_frames(
        &background,
        &cells,
        &portraits[0],
        Some(&blank_font),
        Some(&index),
        &layout,
        1,
    )
    .unwrap();
    for (progress, original) in progressive.iter().enumerate() {
        let visible = native_story_dialogue_visible_glyphs(&layout, 1, progress as i64).unwrap();
        let mut ink = blank_frames[progress].clone();
        let (x, y, width, _, _) = native_story_dialogue_text_geometry(&layout.control).unwrap();
        for glyph in &visible {
            let mut sample = vec![0u8; 18 * 17];
            font.blit_native_glyph(
                &mut sample,
                18,
                1,
                index.get(&glyph.token).copied().unwrap_or(0),
                fdtxt::NativeGlyphStyle { foreground: 0xcd, shadow: 0x4c, background: 0 },
            )
            .unwrap();
            for py in 0..17i64 {
                for px in 0..18i64 {
                    let (dx, dy) = (glyph.x - 1 + px, glyph.y + py);
                    if dx < x - 1 || dx >= x + width || dy < y || dy >= y + 72 {
                        continue;
                    }
                    let value = sample[(py * 18 + px) as usize];
                    if value != 0 {
                        ink[(dy * 320 + dx) as usize] = value;
                    }
                }
            }
        }
        for py in y..y + 72 {
            for px in x - 1..x + width {
                let at = (py * 320 + px) as usize;
                let value = original[at];
                assert_eq!(
                    value, ink[at],
                    "HD glyph clock differs at progress {progress} pixel ({px},{py}): {value:x} != {:x}; visible={visible:?}",
                    ink[at]
                );
            }
        }
    }
    for y in 0..2 * NATIVE_STORY_LINE_STEP {
        for x in -1..13 * NATIVE_STORY_GLYPH_STEP {
            let a = (NATIVE_STORY_UPPER_TEXT + y * 320 + x) as usize;
            let b = (NATIVE_STORY_UPPER_TEXT + (y + NATIVE_STORY_LINE_STEP) * 320 + x) as usize;
            assert_eq!(page1[a], page0[b], "後頁未保留前頁後兩行");
        }
    }
    let steps = native_story_dialogue_glyph_steps(&layout, 1);
    assert!(matches!(&steps, Ok(s) if s.len() == progressive.len()), "跨頁普通字形與捲動時間軸不符: {steps:?}");
    let opening = compose_native_story_dialogue_opening_frames(&background, &cells, &layout).unwrap();
    assert_eq!(opening.len(), NATIVE_STORY_OPENING_GRID_SIZES.len(), "opening frames");
    for i in 1..opening.len() {
        assert!(opening[i - 1] != opening[i], "opening grid stage {i} did not expand");
    }
    assert_eq!(
        background[(NATIVE_STORY_UPPER_FRAME_Y * 320 + 5) as usize],
        7,
        "native story opening mutated its caller-owned background"
    );
    let closing = compose_native_story_dialogue_closing_frames(&background, &cells, &layout, 0, 0, 0).unwrap();
    assert!(
        closing.len() == 5
            && closing[0] == opening[3]
            && closing[1] == opening[2]
            && closing[2] == opening[1]
            && closing[3] == opening[0]
            && closing[4] == background,
        "sub_16B43 snapshot restore order differs from 16x5..background"
    );
    let motion = compose_native_story_dialogue_closing_frames(&background, &cells, &layout, 2, 2, 3).unwrap();
    assert!(
        motion.len() == 5 + 2 + 3 + 1 && motion[motion.len() - 1] != background,
        "closing motion frames={} or final cursor overlay is missing",
        motion.len()
    );
    let mut bad = layout.clone();
    bad.pages = vec![vec!["🙂".into()]];
    assert!(
        compose_native_story_dialogue_page(&background, &cells, &portraits[0], Some(&font), &index, &bad, 0).is_err(),
        "unknown editable glyph did not fail closed"
    );
}
