//! 移植自 `native_command10_12_test.go`。

use crate::fdother::*;
use std::path::PathBuf;

// Go: TestBuildNativeCommand10To12SchedulePreservesRawTables
#[test]
fn build_native_command10_to12_schedule_preserves_raw_tables() {
    for id in 10..=12 {
        let s = build_native_command10_to12_schedule(id).unwrap();
        assert!(
            s.sound_resource == 80
                && s.x_origins == [128, 0, -128]
                && s.y_origins == [128, 0, 128]
                && s.sampling_increments == [131, 128, 125]
                && s.surface_cycle == [0, 1, 2, 3]
                && s.frames == 60
                && s.delay_ms == 10
                && s.main_sample == 13
                && s.main_sample_frames == [0, 6, 12, 18, 24, 30, 36, 42]
                && s.result_resource == 5
                && s.result_digit_bias == 94
                && s.result_miss_descriptors == [116, 117, 118, 118]
                && s.result_frames == 22
                && s.result_hold_ms == 500,
            "command {id} schedule={s:?}"
        );
    }
    let s10 = build_native_command10_to12_schedule(10).unwrap();
    let s11 = build_native_command10_to12_schedule(11).unwrap();
    let s12 = build_native_command10_to12_schedule(12).unwrap();
    assert!(!s10.prelude.enabled);
    assert_eq!(
        s11.prelude,
        NativeCommand10To12Prelude { enabled: true, initial_radius: 15, radius_step: 10, repeat: 10, sample: 2 }
    );
    assert_eq!(
        s12.prelude,
        NativeCommand10To12Prelude { enabled: true, initial_radius: 30, radius_step: 16, repeat: 10, sample: 2 }
    );
    assert!(build_native_command10_to12_schedule(9).is_err(), "command9 accepted");
    assert!(build_native_command10_to12_schedule(13).is_err(), "command13 accepted");
}

/// 原版 FDOTHER.DAT：`FD2_FLAME2_DIR`；Go 測試用 `../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT`，
/// 缺檔時略過。`FD2_REQUIRE_ORIGINAL=1` 時缺檔即失敗。
fn original_fdother() -> Option<PathBuf> {
    let path = match std::env::var_os("FD2_FLAME2_DIR") {
        Some(dir) => PathBuf::from(dir).join("FDOTHER.DAT"),
        None => PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"),
    };
    if path.is_file() {
        return Some(path);
    }
    assert!(std::env::var_os("FD2_REQUIRE_ORIGINAL").is_none(), "缺少原版 FDOTHER.DAT");
    eprintln!("略過：沒有玩家提供的 FDOTHER.DAT");
    None
}

// Go: TestNativeCommand10To12OriginalSoundSelectors
#[test]
fn native_command10_to12_original_sound_selectors() {
    let Some(path) = original_fdother() else { return };
    for selector in [2, 13] {
        let raw = crate::archive::read_nested_resource(&path, NATIVE_COMMAND10_TO12_SOUND_RESOURCE as usize, selector)
            .unwrap_or_else(|e| panic!("FDOTHER #80 selector {selector}: {e}"));
        assert!(!raw.is_empty(), "FDOTHER #80 selector {selector} len=0");
    }
}
