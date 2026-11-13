//! Go `native_command4_test.go`。

use super::*;
use crate::archive::read_nested_resource;

pub(super) fn command4_test_animation() -> Animation {
    test_animation(14, 0)
}

// Go: TestNativeCommand4ScheduleAndMarkers
#[test]
fn native_command4_schedule_and_markers() {
    let s = build_native_command4_presentation_schedule(1, Some(&command4_test_animation())).unwrap();
    assert!(
        s.command_id == 4
            && s.effect_resource == 22
            && s.sound_resource == 85
            && s.front_frames == 2
            && s.target_frames == 12
            && s.tail_frames == 8
            && s.marker_counter == 3,
        "command4 schedule={s:?}"
    );
    let mut state = new_native_command5_state_for_schedule(0x1234, s);
    for _ in 0..s.front_frames {
        state = plan_native_command5_draw_frame(state, s, 1).unwrap().next;
    }
    let (frames, _) = build_native_command5_target_sequence(state, s, 1).unwrap();
    let stages = frames.iter().filter(|f| f.hp_stage != 0).count();
    assert!(frames.len() == 12 && stages == 6, "frames={} stages={stages}", frames.len());
}

// Go: TestOriginalFDOTHERCommand4ResourcesMatchRecoveredSignatures（Go 版缺檔時直接失敗、不略過）
#[test]
fn original_fdother_command4_resources_match_recovered_signatures() {
    let Some(path) = original("FDOTHER.DAT") else { return };
    for (side, resource) in [(1u8, 22), (0, 23)] {
        let effect = decode_resource(&path, resource).unwrap();
        let s = build_native_command4_presentation_schedule(side, Some(&effect)).unwrap();
        assert_eq!(s.effect_resource, resource);
    }
    for sample in 0..=1 {
        let raw = read_nested_resource(&path, 85, sample).unwrap();
        assert!(!raw.is_empty(), "#85 sub{sample}");
    }
}
