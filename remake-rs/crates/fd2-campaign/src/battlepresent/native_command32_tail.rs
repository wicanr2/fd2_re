//! 移植自 Go `battlepresent/native_command32_tail.go`。

use super::{GoResult, Result, at, err, go_sharp_x, len64};
use crate::indexedmap::{self, NativeCommandHealTailTarget, NativeCommandTailLayer};
use fd2_formats::fdother::{LMI1Entry, NativeAIItemType21TailSchedule, NativeCommand32TailSchedule};
use fd2_rules::battle::{self, NativePresentationDigit};
use std::collections::BTreeSet;

/// Go：`native_command32_tail.go:12`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand32TailResult {
    pub target_record: i64,
    pub hit: bool,
    pub damage: i64,
}

/// Go：`native_command32_tail.go:18`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommand32TailFrames {
    pub effect: Vec<Vec<u8>>,
    pub toggle: Vec<Vec<u8>>,
    pub result: Vec<Vec<u8>>,
    pub queue: Vec<NativePresentationDigit>,
    pub effect_sources: Vec<Vec<NativeCommandTailLayer>>,
    pub toggle_sources: Vec<Vec<NativeCommandTailLayer>>,
    pub result_sources: Vec<Vec<NativeCommandTailLayer>>,
}

/// Go：`native_command32_tail.go:26`（型別別名）。
pub type NativeAIItemType21TailResult = NativeCommand32TailResult;
/// Go：`native_command32_tail.go:27`（型別別名）。
pub type NativeAIItemType21TailFrames = NativeCommand32TailFrames;

/// BuildNativeCommand32TailFrames prebuilds the framebuffer-only 0x2111A
/// sequence. The caller owns Draw timing, sample9 and the HP/RNG transaction.
///
/// Go：`native_command32_tail.go:31`
#[allow(clippy::too_many_arguments)]
pub fn build_native_command32_tail_frames(
    baseline_work: &[u8],
    baseline_vga: &[u8],
    post_transaction_work: &[u8],
    post_transaction_vga: &[u8],
    effect: &[LMI1Entry],
    digits: &[LMI1Entry],
    targets: &[NativeCommandHealTailTarget],
    results: &[NativeCommand32TailResult],
    camera_x: i64,
    camera_y: i64,
    schedule: &NativeCommand32TailSchedule,
) -> Result<NativeCommand32TailFrames> {
    if schedule.effect_resource != 6
        || schedule.effect_start != 0x40
        || schedule.effect_frames != 10
        || schedule.toggle_a != 0x4a
        || schedule.toggle_b != 0x4b
        || schedule.toggle_pairs != 4
        || schedule.digit_bias != 0x5e
        || schedule.digit_frames != 22
    {
        return Err(err("battlepresent: native command32 tail contract unavailable"));
    }
    build_native1_cac7_tail_frames(
        baseline_work,
        baseline_vga,
        post_transaction_work,
        post_transaction_vga,
        effect,
        digits,
        targets,
        results,
        camera_x,
        camera_y,
        TailContract {
            effect_start: schedule.effect_start,
            effect_frames: schedule.effect_frames,
            toggle_a: schedule.toggle_a,
            toggle_b: schedule.toggle_b,
            toggle_pairs: schedule.toggle_pairs,
            digit_bias: schedule.digit_bias,
            digit_frames: schedule.digit_frames,
            digit_vertical: schedule.digit_vertical,
        },
        "command32",
    )
}

/// Go：`native_command32_tail.go:50`
#[allow(clippy::too_many_arguments)]
pub fn build_native_ai_item_type21_tail_frames(
    baseline_work: &[u8],
    baseline_vga: &[u8],
    post_transaction_work: &[u8],
    post_transaction_vga: &[u8],
    effect: &[LMI1Entry],
    digits: &[LMI1Entry],
    targets: &[NativeCommandHealTailTarget],
    results: &[NativeAIItemType21TailResult],
    camera_x: i64,
    camera_y: i64,
    schedule: &NativeAIItemType21TailSchedule,
) -> Result<NativeAIItemType21TailFrames> {
    if schedule.effect_resource != 6
        || (schedule.command_id != 1 && schedule.command_id != 6 && schedule.command_id != 7)
        || schedule.toggle_a != 0x4a
        || schedule.toggle_b != 0x4b
        || schedule.toggle_pairs != 4
        || schedule.digit_bias != 0x5e
        || schedule.digit_frames != 22
    {
        return Err(err("battlepresent: native AI item type21 tail contract unavailable"));
    }
    build_native1_cac7_tail_frames(
        baseline_work,
        baseline_vga,
        post_transaction_work,
        post_transaction_vga,
        effect,
        digits,
        targets,
        results,
        camera_x,
        camera_y,
        TailContract {
            effect_start: schedule.effect_start,
            effect_frames: schedule.effect_frames,
            toggle_a: schedule.toggle_a,
            toggle_b: schedule.toggle_b,
            toggle_pairs: schedule.toggle_pairs,
            digit_bias: schedule.digit_bias,
            digit_frames: schedule.digit_frames,
            digit_vertical: schedule.digit_vertical,
        },
        "AI item type21",
    )
}

/// Go `buildNative1CAC7TailFrames` 的八個純量參數；Rust 收成一個結構以免參數過長，值與順序不變。
struct TailContract {
    effect_start: i64,
    effect_frames: i64,
    toggle_a: i64,
    toggle_b: i64,
    toggle_pairs: i64,
    digit_bias: i64,
    digit_frames: i64,
    digit_vertical: [i64; 25],
}

/// Go：`native_command32_tail.go:69`
#[allow(clippy::too_many_arguments)]
fn build_native1_cac7_tail_frames(
    baseline_work: &[u8],
    baseline_vga: &[u8],
    post_transaction_work: &[u8],
    post_transaction_vga: &[u8],
    effect: &[LMI1Entry],
    digits: &[LMI1Entry],
    targets: &[NativeCommandHealTailTarget],
    results: &[NativeCommand32TailResult],
    camera_x: i64,
    camera_y: i64,
    c: TailContract,
    owner: &str,
) -> Result<NativeCommand32TailFrames> {
    if targets.is_empty()
        || results.len() != targets.len()
        || c.effect_start < 0
        || c.effect_frames <= 0
        || c.effect_start.wrapping_add(c.effect_frames) > len64(effect)
        || c.toggle_a < 0
        || c.toggle_b < 0
        || c.toggle_a >= len64(effect)
        || c.toggle_b >= len64(effect)
        || c.toggle_pairs != 4
        || c.digit_bias != 0x5e
        || c.digit_frames != 22
        || len64(digits) <= c.digit_bias + 9
    {
        return Err(err(format!("battlepresent: native 0x1cac7 {owner} contract unavailable")));
    }
    let mut by_record = BTreeSet::new();
    let mut visible_effect = Vec::with_capacity(targets.len());
    for (index, target) in targets.iter().enumerate() {
        if by_record.contains(&target.record_index) || results[index].target_record != target.record_index {
            return Err(err(format!("battlepresent: command32 target {index} provenance mismatch")));
        }
        by_record.insert(target.record_index);
        if target.x >= camera_x - 1 && target.x <= camera_x + 12 && target.y >= camera_y - 1 && target.y <= camera_y + 8
        {
            visible_effect.push(*target);
        }
    }

    let mut out = NativeCommand32TailFrames {
        effect: Vec::with_capacity(at(c.effect_frames)),
        toggle: Vec::with_capacity(at(c.toggle_pairs * 2)),
        result: Vec::with_capacity(at(c.digit_frames)),
        ..Default::default()
    };
    let compose = |snapshot: &[u8], initial_vga: &[u8], descriptor: i64, visible: &[NativeCommandHealTailTarget]| {
        let (mut work, mut vga) = (snapshot.to_vec(), initial_vga.to_vec());
        let layers = indexedmap::compose_native_command_heal_effect_frame_with_sources(
            &mut work,
            &mut vga,
            snapshot,
            &effect[at(descriptor)],
            visible,
            camera_x,
            camera_y,
            6,
            descriptor,
        )?;
        Ok::<_, indexedmap::Error>((vga, layers))
    };
    for frame in 0..c.effect_frames {
        let (vga, layers) = compose(baseline_work, baseline_vga, c.effect_start + frame, &visible_effect)
            .wrap(format!("battlepresent: command32 effect {frame}: "))?;
        out.effect.push(vga);
        out.effect_sources.push(layers);
    }
    for _ in 0..c.toggle_pairs {
        for descriptor in [c.toggle_a, c.toggle_b] {
            let (vga, layers) = compose(baseline_work, baseline_vga, descriptor, &visible_effect)
                .wrap(format!("battlepresent: command32 toggle {}: ", go_sharp_x(descriptor)))?;
            out.toggle.push(vga);
            out.toggle_sources.push(layers);
        }
    }
    for (index, result) in results.iter().enumerate() {
        let target = &targets[index];
        let in_camera =
            target.x >= camera_x && target.x < camera_x + 12 && target.y >= camera_y - 1 && target.y <= camera_y + 7;
        if result.hit {
            out.queue = battle::append_native_presentation_digits(
                &out.queue,
                result.damage,
                c.digit_bias,
                result.target_record,
                in_camera,
            )
            .go()?;
        } else if in_camera {
            out.queue = battle::append_native_presentation_miss(&out.queue, result.target_record);
        }
    }
    for frame in 0..c.digit_frames {
        let (mut work, mut vga) = (post_transaction_work.to_vec(), post_transaction_vga.to_vec());
        let layers = indexedmap::compose_native_command_heal_digit_frame_with_sources(
            &mut work,
            &mut vga,
            post_transaction_work,
            digits,
            &out.queue,
            targets,
            camera_x,
            camera_y,
            c.digit_vertical,
            frame,
        )
        .wrap(format!("battlepresent: command32 result {frame}: "))?;
        out.result.push(vga);
        out.result_sources.push(layers);
    }
    Ok(out)
}

#[cfg(test)]
mod native_command32_tail_test;
