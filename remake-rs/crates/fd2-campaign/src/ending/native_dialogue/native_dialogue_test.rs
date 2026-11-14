//! 移植自 `remake/internal/ending/native_dialogue_test.go`。
//!
//! 第一個測試呼叫 B 組的 `campaign::native_class_list_{opening,closing}_frames`。

use super::*;

#[test]
fn test_native_dialogue_uses_six_band_opening_and_five_band_closing_restore() {
    let background = vec![0u8; ENDING_DIALOGUE_BYTES as usize];
    let mut base = vec![0u8; ENDING_DIALOGUE_BYTES as usize];
    for y in 112..198 {
        for x in 5..315 {
            base[y * 320 + x] = (y - 111) as u8;
        }
    }
    let opening = compose_native_dialogue_opening_frames(&background, &base).unwrap();
    assert_eq!(opening.len(), 6, "opening frames");
    for (i, y) in [177, 164, 151, 138, 125, 112].into_iter().enumerate() {
        assert_eq!(opening[i][y * 320 + 5], 1, "opening frame{i} did not start at y={y}");
    }
    let closing = compose_native_dialogue_closing_frames(&background, &base).unwrap();
    assert!(
        closing.len() == 6 && closing[5] == background,
        "closing frames={} or source restore mismatch",
        closing.len()
    );
    for (i, y) in [125, 138, 151, 164, 177].into_iter().enumerate() {
        assert_eq!(closing[i][y * 320 + 5], 1, "closing frame{i} did not start at y={y}");
    }
}

#[test]
fn test_native_dialogue_progressive_frames_allow_four_rows_and_publish_one_glyph() {
    let mut font_raw = vec![0u8; 2 * fdtxt::GLYPH_BYTES];
    for row in 0..fdtxt::GLYPH_HEIGHT {
        font_raw[row * 2] = 0x80;
        font_raw[fdtxt::GLYPH_BYTES + row * 2] = 0x40;
    }
    let font = fdtxt::parse_font(&font_raw).unwrap();
    let pages = vec![vec!["甲".to_string(), "乙".into(), "甲".into(), "乙".into()]];
    let glyphs = BTreeMap::from([("甲".to_string(), 0), ("乙".to_string(), 1)]);
    let frames = compose_native_dialogue_progressive_frames(
        &vec![0u8; ENDING_DIALOGUE_BYTES as usize],
        Some(&font),
        Some(&glyphs),
        &pages,
        0,
    )
    .unwrap();
    assert_eq!(frames.len(), 5, "progressive frames, want frame0 plus four glyphs");
    for i in 1..frames.len() {
        assert_ne!(frames[i - 1], frames[i], "glyph frame{i} did not change indexed output");
    }
    assert_eq!(
        frames[4][(ENDING_DIALOGUE_TEXT_OFFSET + 3 * ENDING_DIALOGUE_LINE_STEP + 1) as usize],
        0xcd,
        "fourth ending row was not rendered at the proven line step"
    );
}

#[test]
fn test_native_dialogue_base_uses_right_edge_portrait_and_fails_closed() {
    let background = vec![0u8; ENDING_DIALOGUE_BYTES as usize];
    let cells: Vec<fdother::RawCell> =
        (0..18).map(|i| fdother::RawCell { width: 1, height: 1, pixels: vec![(i + 1) as u8] }).collect();
    let portrait = dato::Frame { width: 2, height: 1, pixels: vec![0x21, 0x22] };
    let base = compose_native_dialogue_base(&background, &cells, &portrait).unwrap();
    let anchor = ENDING_DIALOGUE_PORTRAIT_ANCHOR as usize;
    assert!(
        base[anchor] == 0x21 && base[anchor - 1] == 0x22,
        "right-edge portrait bytes={:#x}/{:#x}",
        base[anchor],
        base[anchor - 1]
    );
    assert!(
        compose_native_dialogue_base(&background, &cells[..17], &portrait).is_err(),
        "incomplete FDOTHER #5 cells were accepted"
    );
}

#[test]
fn test_native_dialogue_playback_preserves_block_page_utterance_and_closing_gates() {
    let frame = |value: u8| {
        let mut out = vec![0u8; ENDING_DIALOGUE_BYTES as usize];
        out[0] = value;
        out
    };
    let block = |base: u8, pages: usize| {
        let opening: Vec<Vec<u8>> = (0..6).map(|i| frame(base + i)).collect();
        let closing: Vec<Vec<u8>> = (0..6).map(|i| frame(base + 20 + i)).collect();
        let page_frames: Vec<Vec<Vec<u8>>> =
            (0..pages as u8).map(|i| vec![frame(base + 40 + i), frame(base + 50 + i)]).collect();
        let mouth: Vec<Vec<u8>> = (0..pages as u8).map(|i| frame(base + 60 + i)).collect();
        NativeDialogueBlockFrames {
            opening,
            utterances: vec![
                NativeDialogueUtteranceFrames { pages: page_frames, mouth_open: mouth },
                NativeDialogueUtteranceFrames {
                    pages: vec![vec![frame(base + 70)]],
                    mouth_open: vec![frame(base + 71)],
                },
            ],
            closing,
        }
    };
    let mut p = new_native_dialogue_playback(&[block(1, 2), block(100, 1)]).unwrap();
    for i in 0..6u8 {
        assert_eq!(p.current_frame(false)[0], 1 + i, "opening frame{i}");
        p.step();
    }
    while !p.waiting() {
        p.step();
    }
    assert!(p.current_frame(true)[0] == 61 && p.confirm(), "first page did not expose mouth frame or accept confirm");
    while !p.waiting() {
        p.step();
    }
    assert!(p.page == 1 && p.confirm(), "second page state={p:?}");
    while !p.waiting() {
        p.step();
    }
    assert!(
        p.utterance == 1 && p.confirm() && p.phase == NATIVE_DIALOGUE_CLOSING,
        "second utterance did not enter closing"
    );
    for _ in 0..6 {
        p.step();
    }
    assert!(p.block == 1 && p.phase == NATIVE_DIALOGUE_OPENING, "next block state block={} phase={}", p.block, p.phase);
    while !p.waiting() {
        p.step();
    }
    assert!(p.confirm(), "final block first utterance did not confirm");
    while !p.waiting() {
        p.step();
    }
    assert!(p.confirm(), "final block second utterance did not confirm");
    while !p.done() {
        p.step();
    }
    assert!(p.current_frame(false).is_empty(), "completed playback retained a visible frame");
}
