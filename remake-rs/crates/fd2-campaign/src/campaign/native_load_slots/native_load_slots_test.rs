//! 移植自 `remake/internal/campaign/native_load_slots_test.go`。

use super::*;
use crate::test_support::{docs_file, input};

/// Go：`loadNativeLoadSlotTestAssets`。缺原版檔時回 `None`（Go 直接 `t.Fatal`）。
fn load_native_load_slot_test_assets() -> Option<(fdother::LMI1Entry, fdtxt::Strings, fdtxt::Font)> {
    let fdother_path = input("FD2_FLAME2_DIR", "FDOTHER.DAT")?;
    let resource13 = fdother::read_resource(&fdother_path, 13).unwrap();
    let entries = fdother::parse_lmi1(&resource13).unwrap();
    assert!(entries.len() > 16, "FDOTHER#13 entries={}", entries.len());
    let strings = fdtxt::parse(&fdother::read_resource(fdother_path.with_file_name("FDTXT.DAT"), 0).unwrap()).unwrap();
    let font = fdtxt::parse_font(&fdother::read_resource(&fdother_path, 4).unwrap()).unwrap();
    Some((entries[16].clone(), strings, font))
}

const EMPTY: NativeLoadSlot = NativeLoadSlot { empty: true, chapter: 0 };

/// Go：`TestComposeNativeLoadSlotsFrameMatchesEmptyDOSBoxOracle`
#[test]
fn compose_native_load_slots_frame_matches_empty_dosbox_oracle() {
    let Some((dialogue_box, strings, font)) = load_native_load_slot_test_assets() else { return };
    let frame = compose_native_load_slots_frame(
        &vec![0u8; (NATIVE_LOAD_SLOT_WIDTH * NATIVE_LOAD_SLOT_HEIGHT) as usize],
        &dialogue_box,
        Some(&strings),
        Some(&font),
        [EMPTY; 4],
        0,
    )
    .unwrap();
    assert_native_load_frame_matches_oracle(&frame, "load-empty-original-dosbox.png");
}

/// Go：`TestComposeNativeLoadSlotsFrameMatchesModifiedValidDOSBoxOracle`
#[test]
fn compose_native_load_slots_frame_matches_modified_valid_dosbox_oracle() {
    let Some((dialogue_box, strings, font)) = load_native_load_slot_test_assets() else { return };
    let frame = compose_native_load_slots_frame(
        &vec![0u8; (NATIVE_LOAD_SLOT_WIDTH * NATIVE_LOAD_SLOT_HEIGHT) as usize],
        &dialogue_box,
        Some(&strings),
        Some(&font),
        [NativeLoadSlot { empty: false, chapter: 1 }, EMPTY, EMPTY, EMPTY],
        0,
    )
    .unwrap();
    assert_native_load_frame_matches_oracle(&frame, "load-valid-modified-original-dosbox.png");
}

/// 以 `png` crate 展開成 8 位元 RGB(A) 後，依 Go `color.NRGBA.RGBA()` 換成 16 位元預乘值。
/// 這是 Go `image.Decode` + `At().RGBA()` 的測試端對應；只用在受版控的原版擷圖。
fn decode_png_rgba16(path: &str) -> (usize, usize, Vec<[u32; 3]>) {
    let data = std::fs::read(path).unwrap();
    let mut decoder = png::Decoder::new(std::io::Cursor::new(&data));
    decoder.set_transformations(png::Transformations::EXPAND | png::Transformations::STRIP_16);
    let mut reader = decoder.read_info().unwrap();
    let mut buf = vec![0u8; reader.output_buffer_size().unwrap()];
    let info = reader.next_frame(&mut buf).unwrap();
    let (w, h) = (info.width as usize, info.height as usize);
    let channels = info.color_type.samples();
    let mut out = Vec::with_capacity(w * h);
    for y in 0..h {
        let line = &buf[y * info.line_size..];
        for x in 0..w {
            let p = &line[x * channels..x * channels + channels];
            let (r, g, b, a) = match channels {
                1 => (p[0], p[0], p[0], 255),
                2 => (p[0], p[0], p[0], p[1]),
                3 => (p[0], p[1], p[2], 255),
                _ => (p[0], p[1], p[2], p[3]),
            };
            let a16 = a as u32 * 0x101;
            let pre = |v: u8| v as u32 * 0x101 * a16 / 0xffff;
            out.push([pre(r), pre(g), pre(b)]);
        }
    }
    (w, h, out)
}

/// Go：`assertNativeLoadFrameMatchesOracle`
fn assert_native_load_frame_matches_oracle(frame: &[u8], filename: &str) {
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let palette = fdother::parse_vga_palette(&fdother::read_resource(&fdother_path, 0).unwrap()).unwrap();
    let Some(oracle_path) = docs_file(&format!("figures/{filename}")) else { return };
    let (w, h, oracle) = decode_png_rgba16(&oracle_path);
    assert!(w as i64 == NATIVE_LOAD_SLOT_WIDTH && h as i64 == NATIVE_LOAD_SLOT_HEIGHT, "oracle bounds={w}x{h}");
    let mut mismatches = 0;
    let (mut first, mut last) = (None, (-1i64, -1i64));
    for y in 0..h {
        for x in 0..w {
            let (gr, gg, gb, _) = palette[frame[y * w + x] as usize].rgba();
            let want = oracle[y * w + x];
            if [gr, gg, gb] != want {
                mismatches += 1;
                last = (x as i64, y as i64);
                if first.is_none() {
                    first = Some((x, y, [gr >> 8, gg >> 8, gb >> 8], want.map(|v| v >> 8)));
                }
            }
        }
    }
    assert_eq!(mismatches, 0, "RGB mismatches; first={first:?} last={last:?}");
}

/// Go：`TestComposeNativeLoadSlotsFrameRejectsInvalidState`
#[test]
fn compose_native_load_slots_frame_rejects_invalid_state() {
    let Some((dialogue_box, strings, font)) = load_native_load_slot_test_assets() else { return };
    assert!(
        compose_native_load_slots_frame(
            &vec![0u8; (NATIVE_LOAD_SLOT_WIDTH * NATIVE_LOAD_SLOT_HEIGHT) as usize],
            &dialogue_box,
            Some(&strings),
            Some(&font),
            [NativeLoadSlot::default(); 4],
            4,
        )
        .is_err(),
        "invalid selection accepted"
    );
}
