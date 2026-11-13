//! 指令 32..35 共用的 `0x27FC9` 索引色演出。移植自 Go `native_command_compound_presentation.go`。

use super::{Animation, FiganiError, msg};

/// 保存指令 32..35 分派到各自尾段之前共用的 `0x27FC9` 索引色演出。調色盤三元組是
/// `0x5254F..0x5255A` 的原始位元組。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCompoundPresentationSchedule {
    pub command_id: i64,
    pub effect_resource: i64,
    pub sound_resource: i64,
    pub actor_slide_frames: i64,
    pub actor_slide_step_x: i64,
    pub effect_slide_frames: i64,
    pub effect_slide_step_x: i64,
    pub prelude_hold_ticks: i64,
    pub main_delay_ticks: i64,
    pub overlay_first_frame: bool,
    pub main_frame_count: i64,
    pub mixer_sample2_frame: i64,
    pub sample1_frame: i64,
    pub tail_enabled: bool,
    pub tail_frames: i64,
    pub palette_rgb: [u8; 3],
}

pub fn build_native_compound_presentation_schedule(
    command_id: i64,
    effect: Option<&Animation>,
) -> Result<NativeCompoundPresentationSchedule, FiganiError> {
    let Some(effect) = effect.filter(|e| (32..=35).contains(&command_id) && e.frames.len() >= 2) else {
        return Err(msg(format!("figani: compound presentation unavailable id={command_id}")));
    };
    let colors: [[u8; 3]; 4] = [[63, 63, 63], [51, 57, 63], [53, 0, 0], [53, 58, 9]];
    let mut s = NativeCompoundPresentationSchedule {
        command_id,
        effect_resource: command_id + 33,
        sound_resource: command_id + 59,
        actor_slide_frames: 8,
        actor_slide_step_x: 20,
        effect_slide_frames: 9,
        effect_slide_step_x: 30,
        prelude_hold_ticks: 6,
        main_delay_ticks: 2,
        main_frame_count: effect.frames.len() as i64 - 1,
        mixer_sample2_frame: -1,
        sample1_frame: -1,
        palette_rgb: colors[(command_id - 32) as usize],
        ..Default::default()
    };
    s.overlay_first_frame = command_id == 33 || command_id == 34;
    if command_id == 32 || command_id == 35 {
        s.mixer_sample2_frame = 1;
        s.tail_enabled = true;
        s.tail_frames = 11;
    }
    if command_id == 34 {
        s.sample1_frame = 2;
    }
    if command_id == 33 {
        s.sample1_frame = 6;
    }
    Ok(s)
}
