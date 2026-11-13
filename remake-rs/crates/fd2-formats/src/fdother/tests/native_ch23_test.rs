//! 移植自 `native_ch23_test.go`。

use super::{TempDir, original, pack, skip};
use crate::fdother::*;
use std::cell::{Cell, RefCell};
use std::path::PathBuf;

const STAGE: usize = (NATIVE_CH23_STAGE_STRIDE * NATIVE_CH23_STAGE_HEIGHT) as usize;
const STRIDE: usize = NATIVE_CH23_STAGE_STRIDE as usize;

fn cells(buf: &mut [u8]) -> &[Cell<u8>] {
    Cell::from_mut(buf).as_slice_of_cells()
}

// Go: TestDecodeAndBlitNativeCh23StageFromPlayerArchive
#[test]
fn decode_and_blit_native_ch23_stage_from_player_archive() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    let frame = decode_native_ch23_stage(&path).unwrap();
    let mut staging = vec![0u8; STAGE];
    blit_native_ch23_stage(&frame, &mut staging).unwrap();
    // Go 版在分離面缺少時 t.Skipf；這裡同樣略過，`FD2_REQUIRE_ORIGINAL=1` 時失敗。
    let Some(root) = pack("surfaces", "FDOTHER_042/resource.json") else { return };
    let separated = load_separated_native_ch23_stage(&root).unwrap();
    let (want_indexed, want_mask) = frame.indexed_layers().unwrap();
    assert!(
        separated.x == frame.x
            && separated.y == frame.y
            && separated.width == frame.width
            && separated.height == frame.height
            && separated.indexed == want_indexed
            && separated.mask == want_mask,
        "separated ch23 stage differs from the original archive"
    );
}

// Go: TestSeparatedNativeCh23StageFailsClosedWithoutPack
#[test]
fn separated_native_ch23_stage_fails_closed_without_pack() {
    let dir = TempDir::new("ch23");
    assert!(load_separated_native_ch23_stage(dir.path()).is_err(), "missing separated ch23 stage was accepted");
}

// Go: TestBlitNativeCh23StageRejectsWrongSurfaceWithoutMutation
#[test]
fn blit_native_ch23_stage_rejects_wrong_surface_without_mutation() {
    let frame = Frame::raw(0, 0, NATIVE_CH23_STAGE_WIDTH, NATIVE_CH23_STAGE_HEIGHT, vec![0, 0, 0, 0]);
    let mut staging = vec![0u8; STAGE - 1];
    staging[0] = 7;
    assert!(blit_native_ch23_stage(&frame, &mut staging).is_err(), "wrong staging surface accepted");
    assert_eq!(staging[0], 7, "rejected staging surface mutated");
}

// Go: TestBlitNativeCh23StageRejectsMalformedRLEWithoutMutation
#[test]
fn blit_native_ch23_stage_rejects_malformed_rle_without_mutation() {
    let frame = Frame::raw(0, 0, NATIVE_CH23_STAGE_WIDTH, NATIVE_CH23_STAGE_HEIGHT, vec![0, 0, 0, 0]);
    let mut staging = vec![0u8; STAGE];
    staging[0] = 13;
    staging[STAGE - 1] = 17;
    assert!(blit_native_ch23_stage(&frame, &mut staging).is_err(), "malformed ch23 RLE accepted");
    assert!(staging[0] == 13 && staging[STAGE - 1] == 17, "malformed RLE partially mutated staging");
}

// Go: TestRotateNativeCh23RowsWrapsBottomRowsToTop
#[test]
fn rotate_native_ch23_rows_wraps_bottom_rows_to_top() {
    let mut buf = vec![0u8; STAGE];
    for row in 0..NATIVE_CH23_STAGE_HEIGHT as usize {
        buf[row * STRIDE] = row as u8;
    }
    rotate_native_ch23_rows(&mut buf, 2).unwrap();
    let got = [buf[0], buf[STRIDE], buf[2 * STRIDE], buf[3 * STRIDE]];
    assert_eq!(got, [190, 191, 0, 1], "rotated rows");
}

// Go: TestRotateNativeCh23RowsRejectsInvalidWithoutMutation
#[test]
fn rotate_native_ch23_rows_rejects_invalid_without_mutation() {
    let mut buf = vec![0u8; STAGE];
    buf[0] = 7;
    buf[STAGE - 1] = 9;
    let want = buf.clone();
    assert!(rotate_native_ch23_rows(&mut buf, NATIVE_CH23_STAGE_HEIGHT + 1).is_err(), "invalid latch accepted");
    assert_eq!(buf, want, "buffer mutated");
}

fn entry(dac: &[u8], index: usize) -> [u8; 3] {
    [dac[index * 3], dac[index * 3 + 1], dac[index * 3 + 2]]
}

// Go: TestApplyNativeDACPaletteCycleE0EFWritesOnlyNativeRange
#[test]
fn apply_native_dac_palette_cycle_e0ef_writes_only_native_range() {
    let mut dac = vec![63u8; 256 * 3];
    apply_native_dac_palette_cycle_e0ef(&mut dac, 0).unwrap();
    assert!(dac[0] == 63 && dac[0xdf * 3] == 63 && dac[0xf0 * 3] == 63, "palette cycle modified outside 0xe0..0xef");
    assert_eq!(entry(&dac, 0xe0), [0x0e, 0x15, 0x26], "palette phase0 entry");
    apply_native_dac_palette_cycle_e0ef(&mut dac, 1).unwrap();
    assert_eq!(entry(&dac, 0xe0), [0x0d, 0x14, 0x25], "palette phase1 entry");
}

/// Go 測試讀 `../../../docs/data/ida/fd2_palette_cycle_table_20261001.json`。容器內 `remake-rs` 掛在
/// `/src`，所以工作區的 `docs/` 要另外掛到 `/docs`；也可用 `FD2_DOCS_DIR` 指定。
fn palette_cycle_evidence() -> Option<PathBuf> {
    let docs = std::env::var_os("FD2_DOCS_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|| PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../../../docs"));
    let path = docs.join("data/ida/fd2_palette_cycle_table_20261001.json");
    if !path.is_file() {
        skip(&format!("IDA 證據 {}", path.display()));
        return None;
    }
    Some(path)
}

// Go: TestNativeDACPaletteCycleMatchesAllOriginalWindows
// 使用獨立IDA證據fixture逐byte驗證全部窗口，避免只驗phase0/1的首色而漏掉後半段錯位。
#[test]
fn native_dac_palette_cycle_matches_all_original_windows() {
    let Some(path) = palette_cycle_evidence() else { return };
    let evidence: serde_json::Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    assert_eq!(
        evidence["sha256"], "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f",
        "IDA證據的原版雜湊不符"
    );
    let hex = evidence["raw_table"]["hex"].as_str().unwrap();
    let raw: Vec<u8> = (0..hex.len()).step_by(2).map(|i| u8::from_str_radix(&hex[i..i + 2], 16).unwrap()).collect();
    assert_eq!(raw.len(), 93, "IDA原始表形態不符");
    for phase in 0..16usize {
        let mut dac = vec![63u8; 256 * 3];
        apply_native_dac_palette_cycle_e0ef(&mut dac, phase as i64).unwrap();
        assert_eq!(&dac[0xe0 * 3..0xf0 * 3], &raw[phase * 3..phase * 3 + 48], "phase{phase}不等於原版IDA窗口");
        assert!(
            dac[..0xe0 * 3].iter().all(|&b| b == 63) && dac[0xf0 * 3..].iter().all(|&b| b == 63),
            "phase{phase}改動窗口外DAC"
        );
    }
}

// Go: TestApplyNativeDACPaletteCycleE0EFRejectsInvalidAtomically
#[test]
fn apply_native_dac_palette_cycle_e0ef_rejects_invalid_atomically() {
    let mut dac = vec![0u8; 256 * 3];
    dac[0xe0 * 3] = 11;
    let want = dac.clone();
    assert!(apply_native_dac_palette_cycle_e0ef(&mut dac, 16).is_err(), "invalid palette phase accepted");
    assert_eq!(dac, want, "palette mutated");
}

// Go: TestAdvanceNativeDACPaletteCycleE0EFMatchesUnsignedTickGate
#[test]
fn advance_native_dac_palette_cycle_e0ef_matches_unsigned_tick_gate() {
    let mut dac = vec![0u8; 256 * 3];
    let (phase, tick, advanced) = advance_native_dac_palette_cycle_e0ef(&mut dac, 0, 0, 1).unwrap();
    assert!(!advanced && phase == 0 && tick == 0, "sub-threshold state=({phase},{tick},{advanced})");
    let (phase, tick, advanced) = advance_native_dac_palette_cycle_e0ef(&mut dac, phase, tick, 2).unwrap();
    assert!(advanced && phase == 1 && tick == 2, "accepted state=({phase},{tick},{advanced})");
    assert_eq!(entry(&dac, 0xe0), [0x0d, 0x14, 0x25], "phase1 entry");
    let (phase, tick, advanced) = advance_native_dac_palette_cycle_e0ef(&mut dac, 15, 0x7fff, -0x7fff).unwrap();
    assert!(advanced && phase == 0 && tick == -0x7fff, "wrapped state=({phase},{tick},{advanced})");
}

// Go: TestAdvanceNativeDACPaletteCycleE0EFRejectsAtomically
#[test]
fn advance_native_dac_palette_cycle_e0ef_rejects_atomically() {
    let mut dac = vec![0u8; 256 * 3];
    dac[0xe0 * 3] = 0x3f;
    let want = dac.clone();
    assert!(advance_native_dac_palette_cycle_e0ef(&mut dac, 16, 0, 2).is_err(), "invalid phase accepted");
    assert_eq!(dac, want, "rejected cycle mutated DAC");
}

fn spec(phase: &str, repeat: i64, stages: std::ops::RangeInclusive<i64>, palette: bool) -> NativeCh23LoopSpec {
    NativeCh23LoopSpec { phase: phase.to_string(), repeat, stage_values: stages.collect(), palette }
}

// Go: TestRunNativeCh23InitialLoopPreservesRawSchedule
#[test]
fn run_native_ch23_initial_loop_preserves_raw_schedule() {
    let mut buf = vec![0u8; STAGE];
    for row in 0..NATIVE_CH23_STAGE_HEIGHT as usize {
        buf[row * STRIDE] = row as u8;
    }
    let staging = cells(&mut buf);
    let (draws, ticks, palettes) = (Cell::new(0), Cell::new(0), Cell::new(0));
    let first_draw_row = Cell::new(-1i64);
    let latches = RefCell::new(Vec::new());
    let mut hooks = NativeCh23LoopHooks {
        latch: Some(Box::new(|stage| {
            latches.borrow_mut().push(stage);
            Ok(())
        })),
        draw: Some(Box::new(|| {
            if draws.get() == 0 {
                first_draw_row.set(staging[0].get() as i64);
            }
            draws.set(draws.get() + 1);
            Ok(())
        })),
        tick: Some(Box::new(|| {
            ticks.set(ticks.get() + 1);
            Ok(())
        })),
        palette: Some(Box::new(|_| {
            palettes.set(palettes.get() + 1);
            Ok(())
        })),
    };
    run_native_ch23_loop(&spec("initial", 30, 2..=9, false), staging, &[], &mut hooks).unwrap();
    drop(hooks);
    assert!(
        draws.get() == 8 * 30 && ticks.get() == draws.get() && palettes.get() == 0 && first_draw_row.get() == 0,
        "initial schedule draws={} ticks={} palettes={} firstRow={}",
        draws.get(),
        ticks.get(),
        palettes.get(),
        first_draw_row.get()
    );
    assert_eq!(latches.into_inner(), [2, 3, 4, 5, 6, 7, 8, 9], "raw latch stages");
}

// Go: TestRunNativeCh23RequiresRawLatchCallback
#[test]
fn run_native_ch23_requires_raw_latch_callback() {
    let mut buf = vec![0u8; STAGE];
    let mut hooks =
        NativeCh23LoopHooks { draw: Some(Box::new(|| Ok(()))), tick: Some(Box::new(|| Ok(()))), ..Default::default() };
    assert!(
        run_native_ch23_loop(&spec("initial", 30, 2..=9, false), cells(&mut buf), &[], &mut hooks).is_err(),
        "missing raw latch callback accepted"
    );
}

// Go: TestRunNativeCh23PaletteLoopKeepsRawESIOrder
#[test]
fn run_native_ch23_palette_loop_keeps_raw_esi_order() {
    let mut buf = vec![0u8; STAGE];
    let mut dac = vec![0x5au8; 256 * 3];
    let raw_esi = RefCell::new(Vec::new());
    let (draws, ticks) = (Cell::new(0), Cell::new(0));
    let mut hooks = NativeCh23LoopHooks {
        latch: Some(Box::new(|_| Ok(()))),
        palette: Some(Box::new(|value| {
            raw_esi.borrow_mut().push(value);
            Ok(())
        })),
        draw: Some(Box::new(|| {
            draws.set(draws.get() + 1);
            Ok(())
        })),
        tick: Some(Box::new(|| {
            ticks.set(ticks.get() + 1);
            Ok(())
        })),
    };
    run_native_ch23_loop(&spec("palette", 12, 10..=14, true), cells(&mut buf), cells(&mut dac), &mut hooks).unwrap();
    drop(hooks);
    let raw_esi = raw_esi.into_inner();
    assert!(
        raw_esi.len() == 60 && draws.get() == 60 && ticks.get() == 60,
        "palette schedule rawESI={} draws={} ticks={}",
        raw_esi.len(),
        draws.get(),
        ticks.get()
    );
    for (i, &value) in raw_esi.iter().enumerate() {
        assert_eq!(value, i as i64, "raw ESI[{i}]");
    }
}

// Go: TestRunNativeCh23LoopRollsBackOnCallbackFailure
#[test]
fn run_native_ch23_loop_rolls_back_on_callback_failure() {
    let mut buf = vec![0x31u8; STAGE];
    let mut dac = vec![0x42u8; 256 * 3];
    let (before_stage, before_dac) = (buf.clone(), dac.clone());
    {
        let staging = cells(&mut buf);
        let mut hooks = NativeCh23LoopHooks {
            latch: Some(Box::new(|stage| {
                staging[0].set(stage as u8);
                Ok(())
            })),
            palette: Some(Box::new(|value| {
                if value == 7 {
                    return Err(FdotherError::Contract("synthetic raw palette failure".to_string()));
                }
                Ok(())
            })),
            draw: Some(Box::new(|| Ok(()))),
            tick: Some(Box::new(|| Ok(()))),
        };
        let r = run_native_ch23_loop(&spec("palette", 12, 10..=14, true), staging, cells(&mut dac), &mut hooks);
        assert!(r.is_err(), "callback failure accepted");
    }
    assert!(buf == before_stage && dac == before_dac, "callback failure mutated raw buffers");
}
