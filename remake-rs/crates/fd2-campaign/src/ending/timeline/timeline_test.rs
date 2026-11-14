//! 移植自 `remake/internal/ending/timeline_test.go`。

use super::*;
use crate::test_support::{docs_file, input, remake_asset};

pub(crate) fn native_2bce5() -> Option<Timeline> {
    let path = remake_asset("endings/native_2bce5.json")?;
    Some(load_timeline(&path).unwrap())
}

#[test]
fn test_native2_bce5_timeline_is_recovered_but_not_playable() {
    let Some(timeline) = native_2bce5() else { return };
    assert!(
        timeline.native_handler == "0x2bce5"
            && timeline.resource.archive == "FDOTHER.DAT"
            && timeline.resource.index == 0x36,
        "timeline header = {timeline:?}"
    );
    assert!(!timeline.ready(), "opaque native ending timeline must remain fail-closed");
    assert_eq!(timeline.segments.len(), 18, "segment count");
    let s = &timeline.segments;
    assert_eq!(s[17].op, "native_post_composite_opaque", "post-composite gate = {:?}", s[17]);
    assert!(
        s[15].first_frame_formula == "(i%4)+1"
            && s[15].second_frame_formula == "(i%4)+5"
            && s[16].first_frame_formula == "(i%4)+1"
            && s[16].second_frame_formula == "(i%4)+5",
        "composite frame formulas = {:?} / {:?}",
        s[15],
        s[16]
    );
    assert!(
        s[12].op == "blit_frame_sequence"
            && s[15].op == "native_composite_loop_opaque"
            && s[16].op == "native_composite_loop_baseline",
        "frame schedule landmarks = {s:?}"
    );
    assert!(
        s[0].frame == Some(0) && s[0].target == "offscreen" && s[0].stride == 320 && s[0].transparent == Some(-1),
        "first native blit = {:?}",
        s[0]
    );
    let ani = &s[3];
    assert!(
        ani.op == "ani_play"
            && ani.ani_resource == Some(2)
            && ani.frame_delay_ms == 100
            && ani.skippable == Some(false),
        "ANI prefix = {ani:?}"
    );
    let blocks = &s[9].else_dialogue;
    assert!(
        blocks.len() == 5
            && blocks[0].portrait_id == 37
            && blocks[0].source_dat == "FDTXT_031"
            && blocks[0].script == "ch31.json"
            && blocks[0].string_index == 2
            && blocks[0].scene_index == 1
            && blocks[0].line == 0
            && blocks[0].count == 1
            && blocks[4].string_index == 6
            && blocks[4].line == 4
            && blocks[4].count == 1,
        "first ending dialogue branch = {blocks:?}"
    );
    let got: Vec<i64> = blocks.iter().map(|b| b.native_utterances[0].operand).collect();
    assert_eq!(got, [37, 21, 26, 105, 32], "first ending native speakers");
    let got = &blocks[0].native_utterances[0];
    assert!(
        got.control_source == "caller_2c39b" && got.control.is_empty() && got.pages.len() == 1,
        "paged ending utterance = {got:?}"
    );
    let got = &blocks[3].native_utterances[0].pages;
    assert!(got.len() == 1 && got[0].len() == 2, "four-row ending page = {got:?}");
    let blocks = &s[9].then_dialogue;
    assert!(
        blocks.len() == 1
            && blocks[0].portrait_id == 4
            && blocks[0].source_dat == "FDTXT_027"
            && blocks[0].script == "ch27.json"
            && blocks[0].string_index == 17
            && blocks[0].scene_index == 3
            && blocks[0].line == 1
            && blocks[0].count == 1
            && blocks[0].native_utterances[0].control_source == "caller_2c39b"
            && blocks[0].native_utterances[0].operand == 4,
        "first bad-ending dialogue branch = {blocks:?}"
    );
    let blocks = &s[13].else_dialogue;
    assert!(
        blocks.len() == 1
            && blocks[0].portrait_id == 45
            && blocks[0].source_dat == "FDTXT_031"
            && blocks[0].script == "ch31.json"
            && blocks[0].string_index == 7
            && blocks[0].scene_index == 1
            && blocks[0].line == 5
            && blocks[0].count == 1
            && blocks[0].native_utterances[0].operand == 45,
        "second ending dialogue branch = {blocks:?}"
    );
    let blocks = &s[13].then_dialogue;
    assert!(
        blocks.len() == 3
            && blocks[0].string_index == 18
            && blocks[0].line == 2
            && blocks[2].string_index == 20
            && blocks[2].line == 4,
        "second bad-ending dialogue branch = {blocks:?}"
    );
}

#[test]
fn test_native2_bce5_dialogue_utterances_match_original_fdtxt_words() {
    let Some(timeline) = native_2bce5() else { return };
    let Some(glyphs) = load_ending_oracle_glyphs() else { return };
    let mut tables = BTreeMap::new();
    for source in ["FDTXT_027", "FDTXT_031"] {
        let Some(path) = input("FD2_EXTRACTED_RAW_DIR", &format!("FDTXT/{source}.bin")) else { return };
        let raw = std::fs::read(path).unwrap();
        tables.insert(source.to_string(), fdtxt::parse(&raw).unwrap());
    }
    for segment_index in [9, 13] {
        let segment = &timeline.segments[segment_index];
        for block in segment.then_dialogue.iter().chain(segment.else_dialogue.iter()) {
            let words = tables[&block.source_dat].words(block.string_index).unwrap();
            let want = decode_ending_oracle_utterances(&words, block.portrait_id, &glyphs)
                .unwrap_or_else(|err| panic!("{} index{}: {err}", block.source_dat, block.string_index));
            assert_eq!(
                block.native_utterances, want,
                "{} index{} native utterances",
                block.source_dat, block.string_index
            );
        }
    }
}

/// Go `loadEndingOracleGlyphs`：`docs/data/glyph_map.json` 的 `"<索引>": "<字>"`。
pub(crate) fn load_ending_oracle_glyphs() -> Option<BTreeMap<u16, String>> {
    let path = docs_file("data/glyph_map.json")?;
    let raw = std::fs::read(path).unwrap();
    let encoded: BTreeMap<String, serde_json::Value> = serde_json::from_slice(&raw).unwrap();
    let mut glyphs = BTreeMap::new();
    for (key, value) in encoded {
        if key == "_comment" {
            continue;
        }
        // Go `fmt.Sscanf(key, "%d", &index)`：讀可選正負號與十進位前綴。
        let digits: String = key
            .chars()
            .enumerate()
            .take_while(|(i, c)| c.is_ascii_digit() || (*i == 0 && (*c == '+' || *c == '-')))
            .map(|(_, c)| c)
            .collect();
        let index: i64 = digits.parse().unwrap_or_else(|_| panic!("invalid glyph map key {key:?}"));
        assert!((0..=0xffff).contains(&index), "invalid glyph map key {key:?}");
        let text = value.as_str().unwrap_or_else(|| panic!("glyph {key} is not a string")).to_string();
        glyphs.insert(index as u16, text);
    }
    Some(glyphs)
}

/// Go `decodeEndingOracleUtterances`。
pub(crate) fn decode_ending_oracle_utterances(
    words: &[u16],
    caller_portrait: i64,
    glyphs: &BTreeMap<u16, String>,
) -> std::result::Result<Vec<NativeDialogueUtterance>, String> {
    let is_speaker = |word: u16| (0xffec..=0xffef).contains(&word);
    let has_speaker = words.iter().any(|&w| is_speaker(w));
    let decode_pages = |body: &[u16]| -> std::result::Result<Vec<Vec<String>>, String> {
        let mut pages = Vec::new();
        let mut page: Vec<String> = Vec::new();
        let mut row = String::new();
        for &word in body {
            match word {
                0xfffe => {
                    if !row.is_empty() {
                        page.push(std::mem::take(&mut row));
                    }
                }
                0xfffd => {
                    if !row.is_empty() {
                        page.push(std::mem::take(&mut row));
                    }
                    if !page.is_empty() {
                        pages.push(std::mem::take(&mut page));
                    }
                }
                _ => match glyphs.get(&word) {
                    Some(text) => row.push_str(text),
                    None => return Err(format!("glyph {word:#x} is absent from glyph_map")),
                },
            }
        }
        if !row.is_empty() {
            page.push(row);
        }
        if !page.is_empty() {
            pages.push(page);
        }
        Ok(pages)
    };
    if !has_speaker {
        let pages = decode_pages(words)?;
        return Ok(vec![NativeDialogueUtterance {
            control_source: "caller_2c39b".into(),
            operand: caller_portrait,
            pages,
            ..Default::default()
        }]);
    }
    let mut utterances = Vec::new();
    let mut cursor = 0;
    while cursor < words.len() {
        if !is_speaker(words[cursor]) {
            cursor += 1;
            continue;
        }
        if cursor + 1 >= words.len() {
            return Err("speaker control at end of string".into());
        }
        let (control, operand) = (words[cursor], words[cursor + 1]);
        cursor += 2;
        let start = cursor;
        while cursor < words.len() && !is_speaker(words[cursor]) {
            cursor += 1;
        }
        let pages = decode_pages(&words[start..cursor])?;
        utterances.push(NativeDialogueUtterance {
            control_source: "fdtxt".into(),
            control: format!("{control:04X}"),
            operand: i64::from(operand),
            pages,
        });
    }
    Ok(utterances)
}
