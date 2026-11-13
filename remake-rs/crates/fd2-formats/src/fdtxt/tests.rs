//! 逐條移植自 Go `remake/internal/fdtxt/fdtxt_test.go` 與 `native_presenter_test.go`，期望值不改。
//!
//! Go 測試以相對路徑讀 `extracted/raw/` 與 `remake/generated-assets/`；Rust 改由環境變數指定：
//! `FD2_FLAME2_DIR`（原版 FLAME2 目錄）、`FD2_EXTRACTED_RAW_DIR`（`extracted/raw`）、
//! `FD2_SEPARATED_DIR`（`remake/generated-assets/fd2-original-b97caf22`）。缺檔時跳過，
//! 與 Go 的 `t.Skip` 相同；`FD2_REQUIRE_ORIGINAL=1` 時缺檔即失敗。

use super::*;
use std::path::PathBuf;
use std::sync::atomic::{AtomicUsize, Ordering};

/// 與 `archive::tests::original` 相同的慣例，延伸到多個輸入根目錄。
fn input(var: &str, rel: &str) -> Option<PathBuf> {
    let path = std::env::var_os(var).map(|d| PathBuf::from(d).join(rel)).filter(|p| p.exists());
    if path.is_none() {
        assert!(std::env::var_os("FD2_REQUIRE_ORIGINAL").is_none(), "缺少 {var}/{rel}");
        eprintln!("略過：沒有 {var}/{rel}");
    }
    path
}

/// Go 在分離素材載入失敗時以 `t.Skipf` 跳過；要求原版輸入時改為失敗，避免把壞素材當成缺檔。
fn skip_or_fail(what: &str, err: &FdtxtError) {
    assert!(std::env::var_os("FD2_REQUIRE_ORIGINAL").is_none(), "{what}: {err}");
    eprintln!("略過：{what}: {err}");
}

struct TempDir(PathBuf);

impl TempDir {
    fn new(tag: &str) -> TempDir {
        static NEXT: AtomicUsize = AtomicUsize::new(0);
        let n = NEXT.fetch_add(1, Ordering::Relaxed);
        let dir = std::env::temp_dir().join(format!("fd2-fdtxt-{tag}-{}-{n}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        TempDir(dir)
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
    }
}

/// Go `json.Marshal(separatedResource{...})` 的輸出（欄位順序與 omitempty 相同）。
fn resource_json(resource: i64, raw_size: i64, strings: &str) -> String {
    format!(
        concat!(
            r#"{{"schema_version":1,"kind":"fdtxt_word_table","asset_id":"text/FDTXT_{r:03}","status":"decoded","#,
            r#""evidence":"confirmed","source":{{"file":"FDTXT.DAT","resource":{r},"size":120502,"#,
            r#""md5":"fe5c487ce4313485f1da9d48d35b05f9","#,
            r#""sha256":"a4555f8a0e61e884b4f504d56a8bdde11672583bbbbc6506281ae10dcdfb1f69","raw_size":{s}}},"#,
            r#""strings":{strings}}}"#
        ),
        r = resource,
        s = raw_size,
        strings = strings
    )
}

fn words_to_bytes(words: &[u16]) -> Vec<u8> {
    words.iter().flat_map(|w| w.to_le_bytes()).collect()
}

// Go: TestLoadSeparatedResourceRebuildsLosslessWords
#[test]
fn load_separated_resource_rebuilds_lossless_words() {
    let root = TempDir::new("lossless");
    let strings = concat!(
        r#"[{"string_id":"FDTXT_031/string_0000","source_index":0,"text":"『","#,
        r#""tokens":[{"kind":"control","control":"FFEF"},{"kind":"glyph","glyph_index":557,"text":"『"}]}]"#
    );
    let directory = root.0.join("FDTXT_031");
    std::fs::create_dir_all(&directory).unwrap();
    std::fs::write(directory.join("resource.json"), resource_json(31, 8, strings)).unwrap();
    let strings = load_separated_resource(&root.0, 31).unwrap();
    let words = strings.words(0).unwrap();
    assert_eq!(words_to_bytes(&words), [0xef, 0xff, 0x2d, 0x02], "words={words:?}");
}

// Go: TestLoadSeparatedResourceRejectsStaleUTF8Projection
#[test]
fn load_separated_resource_rejects_stale_utf8_projection() {
    let root = TempDir::new("stale");
    let strings = concat!(
        r#"[{"string_id":"FDTXT_000/string_0000","source_index":0,"text":"過期","#,
        r#""tokens":[{"kind":"glyph","glyph_index":557,"text":"『"}]}]"#
    );
    let directory = root.0.join("FDTXT_000");
    std::fs::create_dir_all(&directory).unwrap();
    std::fs::write(directory.join("resource.json"), resource_json(0, 4, strings)).unwrap();
    assert!(load_separated_resource(&root.0, 0).is_err(), "stale UTF-8 projection was accepted");
}

// Go: TestSeparatedResource31MatchesOriginalWords
#[test]
fn separated_resource_31_matches_original_words() {
    let Some(raw_path) = input("FD2_EXTRACTED_RAW_DIR", "FDTXT/FDTXT_031.bin") else { return };
    let original = parse(&std::fs::read(raw_path).unwrap()).unwrap();
    let Some(text_root) = input("FD2_SEPARATED_DIR", "text") else { return };
    let separated = match load_separated_resource(text_root, 31) {
        Ok(s) => s,
        Err(e) => return skip_or_fail("separated FDTXT_031 is absent", &e),
    };
    assert_eq!(separated.count(), original.count(), "string count");
    for index in 0..original.count() as i64 {
        let want = original.words(index).unwrap();
        let got = separated.words(index).unwrap();
        assert_eq!(words_to_bytes(&got), words_to_bytes(&want), "string {index} words differ");
    }
}

// Go: TestSeparatedFontMatchesOriginalBits
#[test]
fn separated_font_matches_original_bits() {
    let Some(raw_path) = input("FD2_EXTRACTED_RAW_DIR", "FDOTHER/FDOTHER_004.bin") else { return };
    let original = parse_font(&std::fs::read(raw_path).unwrap()).unwrap();
    let Some(font_root) = input("FD2_SEPARATED_DIR", "fonts") else { return };
    let separated = match load_separated_font(font_root) {
        Ok(f) => f,
        Err(e) => return skip_or_fail("separated FDOTHER_004 font is absent", &e),
    };
    assert_eq!(separated.glyph_count(), original.glyph_count(), "glyph count");
    for glyph in 0..original.glyph_count() as i64 {
        for y in 0..GLYPH_HEIGHT as i64 {
            for x in 0..GLYPH_WIDTH as i64 {
                let want = original.glyph_bit(glyph, x, y).unwrap();
                let got = separated.glyph_bit(glyph, x, y).unwrap();
                assert_eq!(got, want, "glyph={glyph} x={x} y={y}");
            }
        }
    }
}

// Go: TestParseRetainsControlWordsAndStopsAtTerminator
#[test]
fn parse_retains_control_words_and_stops_at_terminator() {
    // 兩個字串，位移 4 與 10；第一個在範圍結束前就遇到結尾字。
    let data = [4, 0, 10, 0, 0x12, 0, 0xfe, 0xff, 0xff, 0xff, 0x34, 0, 0xfd, 0xff, 0xff, 0xff];
    let s = parse(&data).unwrap();
    assert_eq!(s.count(), 2);
    assert_eq!(s.words(0).unwrap(), [0x12, 0xfffe]);
    assert_eq!(s.words(1).unwrap(), [0x34, 0xfffd]);
}

// Go: TestFontGlyphBitUsesMSBLeftOrder
#[test]
fn font_glyph_bit_uses_msb_left_order() {
    let mut data = vec![0u8; GLYPH_BYTES];
    data[0] = 0x80;
    data[1] = 0x01;
    let f = parse_font(&data).unwrap();
    for (x, y) in [(0, 0), (15, 0)] {
        assert!(f.glyph_bit(0, x, y).unwrap(), "bit({x},{y})");
    }
    assert!(!f.glyph_bit(0, 1, 0).unwrap(), "unexpected set bit");
}

// Go: TestBlitGlyphLeavesZeroBitsTransparent
#[test]
fn blit_glyph_leaves_zero_bits_transparent() {
    let mut data = vec![0u8; GLYPH_BYTES];
    data[0] = 0x80;
    data[1] = 0x01;
    let f = parse_font(&data).unwrap();
    let mut dst = vec![7u8; 20 * 16];
    f.blit_glyph(&mut dst, 20, 2, 0, 42).unwrap();
    assert!(
        dst[2] == 42 && dst[17] == 42 && dst[3] == 7 && dst[22] == 7,
        "glyph pixels = {} {} {} {}",
        dst[2],
        dst[17],
        dst[3],
        dst[22]
    );
}

// Go: TestBlitNativeGlyphMatchesForegroundShadowAndBackgroundABI
#[test]
fn blit_native_glyph_matches_foreground_shadow_and_background_abi() {
    let mut data = vec![0u8; GLYPH_BYTES];
    data[0] = 0x80; // 左上角的來源位元
    let f = parse_font(&data).unwrap();
    let mut dst = vec![7u8; 20 * 18];
    let base = 21usize;
    f.blit_native_glyph(
        &mut dst,
        20,
        base as i64,
        0,
        NativeGlyphStyle { foreground: 0xcd, shadow: 0x4c, background: 3 },
    )
    .unwrap();
    assert!(
        dst[base] == 0xcd
            && dst[base + 20 - 1] == 0x4c
            && dst[base + 20] == 0x4c
            && dst[base - 1] == 7
            && dst[base + 1] == 3
            && dst[base + 15 * 20 + 15] == 3,
        "native glyph pixels={:#x} {:#x} {:#x} {:#x} {:#x} {:#x}",
        dst[base],
        dst[base + 20 - 1],
        dst[base + 20],
        dst[base - 1],
        dst[base + 1],
        dst[base + 15 * 20 + 15]
    );
}

// Go: TestBlitNativeGlyphAdjacentForegroundIsNotOverwrittenByShadow
#[test]
fn blit_native_glyph_adjacent_foreground_is_not_overwritten_by_shadow() {
    let mut data = vec![0u8; GLYPH_BYTES];
    data[0] = 0xc0; // 相鄰的 x=0 與 x=1 來源位元
    let f = parse_font(&data).unwrap();
    let mut dst = vec![0u8; 20 * 18];
    let base = 21usize;
    f.blit_native_glyph(
        &mut dst,
        20,
        base as i64,
        0,
        NativeGlyphStyle { foreground: 0xcd, shadow: 0x4c, background: 0 },
    )
    .unwrap();
    assert!(
        dst[base] == 0xcd && dst[base + 1] == 0xcd,
        "adjacent foreground overwritten: {:#x} {:#x}",
        dst[base],
        dst[base + 1]
    );
    for pos in [base + 20 - 1, base + 20, base + 20 + 1] {
        assert_eq!(dst[pos], 0x4c, "shadow at {pos}");
    }
}

// Go: TestLocateLogicalUtteranceUsesPhysicalLocator
#[test]
fn locate_logical_utterance_uses_physical_locator() {
    // 每個 FFxx 字都會結束一段文字；第二個字形 557 是原版開引號，所以這裡有兩句可見台詞。
    let s = Strings { words: vec![vec![9, OPENING_GLYPH, 1, 0xfffe, 9, OPENING_GLYPH, 2, 0xffff]] };
    let got = s.locate_logical_utterance(1).unwrap();
    assert_eq!(got, LogicalLocator { raw_string_index: 0, raw_utterance_index: 1 });
}

// Go: TestPlayerFDTXT031Physical44HasStablePayload
#[test]
fn player_fdtxt031_physical44_has_stable_payload() {
    let Some(path) = input("FD2_EXTRACTED_RAW_DIR", "FDTXT/FDTXT_031.bin") else { return };
    let s = parse(&std::fs::read(path).unwrap()).unwrap();
    assert_eq!(s.count(), 46, "physical strings");
    let words = s.words(44).unwrap();
    assert!(
        words.len() == 130 && words[0] == 0x00b5 && words[words.len() - 1] == 0x0248,
        "physical #44 words={} first/last={:#x}/{:#x}",
        words.len(),
        words[0],
        words[words.len() - 1]
    );
}

// Go: TestChapterLoader30UsesFDTXTArchiveResource31
#[test]
fn chapter_loader_30_uses_fdtxt_archive_resource_31() {
    let Some(archive_path) = input("FD2_FLAME2_DIR", "FDTXT.DAT") else { return };
    let entry = crate::archive::read_resource(archive_path, 31).unwrap(); // 0x1088d 讓 chapter 30 加一。
    let Some(raw_path) = input("FD2_EXTRACTED_RAW_DIR", "FDTXT/FDTXT_031.bin") else { return };
    let raw = std::fs::read(raw_path).unwrap();
    assert!(entry == raw, "FDTXT.DAT resource 31 does not equal FDTXT_031: {}/{} bytes", entry.len(), raw.len());
}

// Go: TestNativePresenterValidatesPreservesBackgroundAndDoesNotMutateFont
#[test]
fn native_presenter_validates_preserves_background_and_does_not_mutate_font() {
    let f = parse_font(&[255u8; GLYPH_BYTES]).unwrap();
    let style = NativeGlyphStyle { foreground: 205, shadow: 76, background: 74 };
    let calls = Arc::new(AtomicUsize::new(0));
    let seen = Arc::clone(&calls);
    let adapter = f.with_native_presenter(Some(Arc::new(move |stride, base, index, actual| {
        seen.fetch_add(1, Ordering::SeqCst);
        assert!(stride == 32 && base == 33 && index == 0 && actual == style, "lost original glyph arguments");
        true
    })));
    let mut dst = vec![9u8; 32 * 20];
    adapter.blit_native_glyph(&mut dst, 32, 33, 0, style).unwrap();
    for y in 0..16 {
        for x in 0..16 {
            assert_eq!(dst[33 + y * 32 + x], 74, "background was suppressed");
        }
    }
    assert!(dst[33 + 16 * 32] == 9 && dst[32] == 9, "replacement leaked native shadow");
    assert!(adapter.blit_native_glyph(&mut dst, 32, 33, 1, style).is_err(), "invalid glyph accepted");
    assert!(adapter.blit_native_glyph(&mut dst[..50], 32, 33, 0, style).is_err(), "invalid destination accepted");
    assert_eq!(calls.load(Ordering::SeqCst), 1, "presenter ran before validation");
    let mut original = vec![0u8; dst.len()];
    f.blit_native_glyph(&mut original, 32, 33, 0, style).unwrap();
    assert!(original[33] == 205 && original[33 + 16 * 32] == 76, "adapter mutated original font");
    let fallback = f.with_native_presenter(Some(Arc::new(|_, _, _, _| false)));
    let mut actual = vec![0u8; dst.len()];
    fallback.blit_native_glyph(&mut actual, 32, 33, 0, style).unwrap();
    assert_eq!(original, actual, "fallback changed original bytes");
}

// 以下為 Rust 端補充：錯誤文字與 Go 顏色模型。

#[test]
fn error_text_matches_go() {
    assert_eq!(parse(&[]).unwrap_err().to_string(), "fdtxt: resource shorter than first offset");
    assert_eq!(parse(&[0, 0]).unwrap_err().to_string(), "fdtxt: invalid first offset 0x0");
    assert_eq!(parse(&[6, 0, 2, 0]).unwrap_err().to_string(), "fdtxt: invalid first offset 0x6");
    assert_eq!(parse(&[4, 0, 2, 0]).unwrap_err().to_string(), "fdtxt: invalid offset at string 1");
    let s = parse(&[2, 0]).unwrap();
    assert_eq!(s.words(-1).unwrap_err().to_string(), "fdtxt: string index -1 out of range");
    assert_eq!(s.locate_logical_utterance(0).unwrap_err().to_string(), "fdtxt: logical utterance index 0 out of range");
    assert_eq!(parse_font(&[0; 33]).unwrap_err().to_string(), "fdtxt: invalid 16x16 font length 33");
    assert!(matches!(load_separated_resource("", 0), Err(FdtxtError::InvalidSeparatedRequest)));
    assert!(matches!(load_separated_font(""), Err(FdtxtError::FontRootEmpty)));
}

/// Go `NRGBA.RGBA()` 先乘 alpha：alpha 0 的白點讀成黑，alpha 128 讀成 0x8080。
#[test]
fn decode_png_rgba_premultiplies_like_go() {
    let mut data = Vec::new();
    {
        let mut encoder = png::Encoder::new(&mut data, 3, 1);
        encoder.set_color(png::ColorType::Rgba);
        encoder.set_depth(png::BitDepth::Eight);
        let mut writer = encoder.write_header().unwrap();
        writer.write_image_data(&[255, 255, 255, 0, 255, 255, 255, 128, 10, 20, 30, 255]).unwrap();
    }
    let img = decode_png_rgba(&data).unwrap();
    assert_eq!((img.width, img.height, img.go_type), (3, 1, "*image.NRGBA"));
    assert_eq!(img.rgba(0, 0), [0, 0, 0, 0]);
    assert_eq!(img.rgba(1, 0), [0x8080, 0x8080, 0x8080, 0x8080]);
    assert_eq!(img.rgba(2, 0), [0x0a0a, 0x1414, 0x1e1e, 0xffff]);
}
