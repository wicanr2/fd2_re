//! 移植自 Go `battlepresent/native_command32_tail_test.go`。

use super::*;
use crate::indexedmap::{NATIVE_MAP_VGA_SIZE, NATIVE_UNIT_PRESENT_WORK_SIZE};
use fd2_formats::fdother;

fn command32_tail_entries(count: usize) -> Vec<LMI1Entry> {
    (0..count).map(|index| LMI1Entry { width: 1, height: 1, pixels: vec![index as u8] }).collect()
}

fn targets() -> Vec<NativeCommandHealTailTarget> {
    vec![NativeCommandHealTailTarget { record_index: 3, x: 1, y: 1, ..Default::default() }]
}

#[test]
fn test_build_native_command32_tail_frames_preserves_all_phases() {
    let (effect, digits) = (command32_tail_entries(0x4c), command32_tail_entries(0x77));
    let schedule = fdother::build_native_command32_tail_schedule(&effect, &digits).unwrap();
    let work = vec![0u8; NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
    let vga = vec![0u8; NATIVE_MAP_VGA_SIZE as usize];
    let results = [NativeCommand32TailResult { target_record: 3, hit: true, damage: 42 }];
    let frames = build_native_command32_tail_frames(
        &work,
        &vga,
        &work,
        &vga,
        &effect,
        &digits,
        &targets(),
        &results,
        0,
        0,
        &schedule,
    )
    .unwrap();
    assert!(
        frames.effect.len() == 10 && frames.toggle.len() == 8 && frames.result.len() == 22 && frames.queue.len() == 4,
        "command32 tail frames effect={} toggle={} result={} queue={:?}",
        frames.effect.len(),
        frames.toggle.len(),
        frames.result.len(),
        frames.queue
    );
    assert!(
        frames.queue[0].digit == 0
            && frames.queue[2].digit == schedule.digit_bias + 4
            && frames.queue[3].digit == schedule.digit_bias + 2,
        "command32 damage queue={:?}",
        frames.queue
    );
}

#[test]
fn test_build_native_command32_tail_frames_miss_and_failure_are_atomic() {
    let (effect, digits) = (command32_tail_entries(0x4c), command32_tail_entries(0x77));
    let schedule = fdother::build_native_command32_tail_schedule(&effect, &digits).unwrap_or_default();
    let work = vec![0u8; NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
    let vga = vec![0u8; NATIVE_MAP_VGA_SIZE as usize];
    let results = [NativeCommand32TailResult { target_record: 3, ..Default::default() }];
    let frames = build_native_command32_tail_frames(
        &work,
        &vga,
        &work,
        &vga,
        &effect,
        &digits,
        &targets(),
        &results,
        0,
        0,
        &schedule,
    )
    .unwrap();
    let got: Vec<i64> = frames.queue.iter().take(4).map(|d| d.digit).collect();
    assert_eq!(got, [0x74, 0x75, 0x76, 0x76], "command32 miss queue");
    let (before_work, before_vga) = (work.clone(), vga.clone());
    assert!(
        build_native_command32_tail_frames(
            &work,
            &vga,
            &work,
            &vga,
            &effect[..0x4b],
            &digits,
            &targets(),
            &results,
            0,
            0,
            &schedule
        )
        .is_err(),
        "command32 accepted missing toggle descriptor"
    );
    // Go 檢查呼叫端緩衝沒有被改寫；Rust 以唯讀借用傳入，型別已保證，仍照 Go 比對一次。
    assert!(work == before_work && vga == before_vga, "failed command32 tail build mutated caller buffers");
}
