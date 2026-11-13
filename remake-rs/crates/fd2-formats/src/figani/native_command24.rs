//! `0x276EC` 指令 24 與 28..31 的動態 FIGANI 契約。移植自 Go `native_command24.go`。

use super::{Animation, FiganiError, msg};

/// 保存 `0x276EC` 指令 24 與 28..31 共用的動態 FIGANI 契約。和只限 selector32 的指令 24 排程不同，
/// 它不指派 actor selector：呼叫端必須提供 actor raw +7 資源並保留其出處。Go 的 nil slice 與空
/// slice 在這裡都是空 `Vec`。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandDerivedStrikeSchedule {
    pub command_id: i64,
    pub target_start: i64,
    pub actor_impact_frame: i64,
    pub target_impact_frame: i64,
    pub target_impact_frames: Vec<i64>,
    pub damage_denominator: i64,
    pub audio_resource: i64,
    pub actor_sample: i64,
    pub actor_cues: Vec<NativeCommandFrameCue>,
    pub target_cues: Vec<NativeCommandFrameCue>,
    pub target_sample: i64,
    pub frame_delays: Vec<i64>,
    pub uses_target_base: bool,
    pub prelude_mode: u8,
    pub uses_bg_transition: bool,
    pub shake_offsets: [i64; 6],
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandFrameCue {
    pub frame: i64,
    pub sample: i64,
}

/// 只取出直接消費的標頭與 marker 欄位。selector19 的指令 28 來源是 header0 加八個目標命中；
/// 既有 header1 strike 契約保留一個目標命中。
pub fn build_native_command_derived_strike_schedule(
    animation: Option<&Animation>,
    command_id: i64,
) -> Result<NativeCommandDerivedStrikeSchedule, FiganiError> {
    let Some(animation) = animation.filter(|a| {
        (a.header_byte1 == 1 || (command_id == 28 && a.header_byte1 == 0))
            && a.header_byte2 > 0
            && (a.header_byte2 as usize) < a.frames.len()
            && (1..=6).contains(&a.header_byte4)
    }) else {
        return Err(msg(format!("figani: derived-strike header unavailable for command {command_id}")));
    };
    let (denominator, uses_target_base, prelude_mode, uses_transition) = match command_id {
        24 | 29 | 30 | 31 => (1, false, 1u8, true),
        28 => (8, true, 0u8, false),
        _ => return Err(msg(format!("figani: derived-strike command unavailable id={command_id}"))),
    };
    let target_start = animation.header_byte2 as i64;
    let (mut actor_impact, mut target_impact) = (-1i64, -1i64);
    let mut delays = Vec::with_capacity(animation.frames.len());
    let (mut actor_cues, mut target_cues, mut target_impacts) = (Vec::new(), Vec::new(), Vec::new());
    for (i, frame) in animation.frames.iter().enumerate() {
        let i = i as i64;
        if frame.delay <= 0 || frame.delay > 0xff {
            return Err(msg(format!("figani: derived-strike frame {i} delay={}", frame.delay)));
        }
        delays.push(frame.delay);
        if i < target_start && frame.raw_byte5 != 0 {
            actor_cues.push(NativeCommandFrameCue { frame: i, sample: frame.raw_byte5 as i64 });
        }
        if command_id == 28 && i >= target_start && frame.raw_byte5 != 0 {
            target_cues.push(NativeCommandFrameCue { frame: i, sample: frame.raw_byte5 as i64 });
        }
        if frame.raw_byte4 != 1 {
            continue;
        }
        if i < target_start {
            if actor_impact >= 0 {
                return Err(FiganiError::Static("figani: derived-strike has multiple actor impact markers"));
            }
            actor_impact = i;
        } else {
            if target_impact >= 0 && command_id != 28 {
                return Err(FiganiError::Static("figani: derived-strike has multiple target impact markers"));
            }
            if target_impact < 0 {
                target_impact = i;
            }
            if command_id == 28 {
                target_impacts.push(i);
                if target_impacts.len() > 8 {
                    return Err(FiganiError::Static("figani: command28 impact count exceeds denominator"));
                }
            }
        }
    }
    if actor_impact < 0 || target_impact < 0 {
        return Err(FiganiError::Static("figani: derived-strike impact markers unavailable"));
    }
    Ok(NativeCommandDerivedStrikeSchedule {
        command_id,
        target_start,
        actor_impact_frame: actor_impact,
        target_impact_frame: target_impact,
        target_impact_frames: target_impacts,
        damage_denominator: denominator,
        audio_resource: 47 + animation.header_byte4 as i64,
        actor_sample: animation.frames[actor_impact as usize].raw_byte5 as i64,
        actor_cues,
        target_cues,
        target_sample: animation.frames[target_impact as usize].raw_byte5 as i64,
        frame_delays: delays,
        uses_target_base,
        prelude_mode,
        uses_bg_transition: uses_transition,
        shake_offsets: [0, 4, 9, 14, 18, 14],
    })
}

/// `0x276EC` 使用的固定 selector32 resource98 契約。`target_start` 之前的幀屬於 `sub_2B659` 的
/// actor 階段，其餘屬於逐目標傷害階段。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand24Schedule {
    pub target_start: i64,
    pub actor_impact_frame: i64,
    pub target_impact_frame: i64,
    pub audio_resource: i64,
    pub actor_sample: i64,
    pub target_sample: i64,
    pub shake_offsets: [i64; 6],
}

const NATIVE_COMMAND24_RAW_FRAMES: [[u8; 4]; 15] = [
    [0, 0, 2, 0],
    [0, 0, 2, 0],
    [0, 0, 2, 0],
    [0, 0, 8, 0],
    [1, 3, 2, 0],
    [0, 0, 2, 0],
    [0, 0, 2, 0],
    [0, 0, 2, 0],
    [0, 0, 8, 0],
    [0, 0, 2, 0],
    [1, 2, 2, 0],
    [0, 0, 6, 0],
    [0, 0, 2, 0],
    [0, 0, 2, 0],
    [0, 0, 4, 0],
];

/// Go `%v` 對 `[4]byte` 的格式：`[1 3 2 0]`。
fn go_bytes4(b: [u8; 4]) -> String {
    format!("[{} {} {} {}]", b[0], b[1], b[2], b[3])
}

/// 拒收任何和已還原的預設轉職路徑不同的動畫。標頭 byte4=6 經原始 `0x525D6` 表 30..35 選到
/// FDOTHER #53，不是通用的音效語意 ID。
pub fn build_native_command24_schedule(animation: Option<&Animation>) -> Result<NativeCommand24Schedule, FiganiError> {
    let Some(animation) = animation.filter(|a| {
        a.header_byte1 == 1
            && a.header_byte2 == 9
            && a.header_byte4 == 6
            && a.frames.len() == NATIVE_COMMAND24_RAW_FRAMES.len()
    }) else {
        return Err(FiganiError::Static("figani: command24 resource98 header/frame signature mismatch"));
    };
    for (i, frame) in animation.frames.iter().enumerate() {
        let want = NATIVE_COMMAND24_RAW_FRAMES[i];
        if frame.delay <= 0 || frame.delay > 0xff {
            return Err(msg(format!("figani: command24 frame {i} delay={}", frame.delay)));
        }
        let got = [frame.raw_byte4, frame.raw_byte5, frame.delay as u8, frame.raw_byte7];
        if got != want {
            return Err(msg(format!(
                "figani: command24 frame {i} raw signature={} want={}",
                go_bytes4(got),
                go_bytes4(want)
            )));
        }
    }
    Ok(NativeCommand24Schedule {
        target_start: 9,
        actor_impact_frame: 4,
        target_impact_frame: 10,
        audio_resource: 53,
        actor_sample: 3,
        target_sample: 2,
        shake_offsets: [0, 4, 9, 14, 18, 14],
    })
}
