//! 移植自 Go `battlepresent/native_physical_body.go`。

use super::{GoResult, Result, err, len64};
use fd2_formats::figani::{Animation, Frame};
use fd2_rules::battle::NativePhysicalStrike;

/// NativePhysicalBodyFrame 保留 0x2939D 的 descriptor 邊界，包含零次呈現。
/// 證據與 READY 契約：docs/data/ida/fd2_physical_background_selection_20261004.json。
///
/// Go：`native_physical_body.go:12`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativePhysicalBodyFrame {
    pub strike: i64,
    pub frame: i64,
    pub hp: i64,
    pub update_panel: bool,
    /// -1 表示不呼叫 0x25A96；0 是有效的 miss cue。
    pub cue: i64,
    pub presents: Vec<NativePhysicalBodyPresent>,
}

/// Go：`native_physical_body.go:19`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalBodyPresent {
    pub idle_frame: i64,
    pub dx: i64,
    pub dy: i64,
    /// -1 保留來源；33 是原版 opaque index。
    pub opaque_fill: i64,
    pub status_pulse: bool,
    pub crit_pulse: bool,
}

/// BuildNativePhysicalBodyPlan 只消費已結算的 Roll，不讀取或推進 RNG。
/// idle 與 impact phase 跨同一個 0x2939D 的多揮保留。
///
/// Go：`native_physical_body.go:29`
pub fn build_native_physical_body_plan(
    attack: Option<&Animation>,
    idle: Option<&Animation>,
    raw_side: u8,
    initial_hp: i64,
    strikes: &[NativePhysicalStrike],
) -> Result<Vec<NativePhysicalBodyFrame>> {
    let (Some(attack), Some(idle)) = (attack, idle) else {
        return Err(err("battlepresent: physical body inputs unavailable"));
    };
    if attack.frames.is_empty()
        || idle.frames.is_empty()
        || idle.frames.len() > 255
        || initial_hp < 0
        || strikes.is_empty()
    {
        return Err(err("battlepresent: physical body inputs unavailable"));
    }
    let start = if attack.header_byte1 != 0 { attack.header_byte2 as usize } else { 0 };
    if start > attack.frames.len() {
        return Err(err("battlepresent: physical body header exceeds frames"));
    }
    let mut hits = 0i64;
    for f in &attack.frames {
        if !(0..=255).contains(&f.delay) {
            return Err(err("battlepresent: physical body delay is not a byte"));
        }
        if f.raw_byte4 != 0 {
            hits += 1;
        }
    }
    for f in &idle.frames {
        if !(0..=255).contains(&f.delay) {
            return Err(err("battlepresent: physical idle delay is not a byte"));
        }
    }
    if hits == 0 {
        hits = 1;
    }
    const HORIZONTAL: [i64; 6] = [0, 4, 9, 14, 18, 14];
    const VERTICAL: [i64; 6] = [0, 2, 4, 6, 8, 10];
    let (mut phase, mut fill, mut idle_frame) = (0usize, -1i64, 0usize);
    let mut idle_counter: u8 = 0;
    let mut hp = initial_hp;
    let mut plan = Vec::new();
    for (si, strike) in strikes.iter().enumerate() {
        if strike.roll.damage < 0 || strike.defender_hp < 0 || hp == 0 {
            return Err(err("battlepresent: physical strike has invalid HP or damage"));
        }
        let (before, mut hit_step) = (hp, 0i64);
        for fi in start..attack.frames.len() {
            let f = &attack.frames[fi];
            let mut item =
                NativePhysicalBodyFrame { strike: si as i64, frame: fi as i64, hp, cue: -1, ..Default::default() };
            if f.raw_byte4 != 0 {
                hit_step += 1;
                hp = before.wrapping_sub(strike.roll.damage.wrapping_mul(hit_step) / hits);
                if hp < 0 {
                    hp = 0;
                }
                (item.hp, item.update_panel) = (hp, true);
                if !strike.roll.missed {
                    (phase, fill) = (5, 33);
                }
            }
            if f.raw_byte5 != 0 {
                item.cue = f.raw_byte5 as i64;
                if f.raw_byte4 != 0 && strike.roll.missed {
                    item.cue = 0;
                }
            }
            for _ in 0..f.delay {
                let (mut dx, mut dy) = (HORIZONTAL[phase], VERTICAL[phase]);
                if attack.header_byte1 != 0 {
                    dy = 0;
                }
                if raw_side != 0 {
                    (dx, dy) = (-dx, -dy);
                }
                let pulse = f.raw_byte4 == 1 && hit_step == hits;
                item.presents.push(NativePhysicalBodyPresent {
                    idle_frame: idle_frame as i64,
                    dx,
                    dy,
                    opaque_fill: fill,
                    status_pulse: pulse && strike.roll.status,
                    crit_pulse: pulse && strike.roll.crit,
                });
                idle_counter = idle_counter.wrapping_add(1); // 原版 uint8 increment 後才作等號比較。
                if idle_counter as i64 == idle.frames[idle_frame].delay {
                    idle_counter = 0;
                    idle_frame += 1;
                    if idle_frame == idle.frames.len() {
                        idle_frame = 0;
                    }
                }
                phase = phase.saturating_sub(1);
                fill = -1;
            }
            plan.push(item);
        }
    }
    Ok(plan)
}

/// ComposeNativePhysicalBodyPresent 以有界 viewport 表達原版 stride400 的
/// 40-pixel staging margin；只裁切 viewport 外像素，不遮蔽比較畫布。
///
/// Go：`native_physical_body.go:121`
pub fn compose_native_physical_body_present(
    base: &[u8],
    attack: &Frame,
    idle: &Frame,
    raw_side: u8,
    present: NativePhysicalBodyPresent,
) -> Result<Vec<u8>> {
    if len64(base) != 320 * 200 || (present.opaque_fill != -1 && present.opaque_fill != 33) {
        return Err(err("battlepresent: physical body base or fill is invalid"));
    }
    let mut out = base.to_vec();
    let fill = if present.opaque_fill == 33 { Some(33u8) } else { None };
    let attack_first = (raw_side == 0 && attack.raw_byte7 & 1 == 0) || (raw_side != 0 && attack.raw_byte7 & 1 != 0);
    if attack_first {
        attack.blit_translated(&mut out, 320, 0, 0, None).go()?;
    }
    idle.blit_translated(&mut out, 320, present.dx, present.dy, fill).go()?;
    if !attack_first {
        attack.blit_translated(&mut out, 320, 0, 0, None).go()?;
    }
    Ok(out)
}

#[cfg(test)]
mod native_physical_body_test;
