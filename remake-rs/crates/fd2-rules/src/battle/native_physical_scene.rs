//! Go 來源：`remake/internal/battle/native_physical_scene.go`（62 行）。

use super::*;

/// NativePhysicalSceneSelection 保留正常 sub_28A6C 的雙背景與台座選擇。
/// 來源與 READY 規格：fd2_physical_background_selection_20261004.json。
/// PrimaryBG／SecondaryBG 在交換前分別對應 IDA LE 0x28CCC／0x28DA2 的來源。
///
/// Go：`native_physical_scene.go:8`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhysicalSceneSelection {
    pub primary_bg: u8,
    pub secondary_bg: u8,
    pub base_bg: u8,
    pub tai: u8,
    pub has_separate_backgrounds: bool,
}

/// SelectNativePhysicalScene 只消費原始 record 與 control，不改戰鬥狀態。
/// 540FF 非零的特殊 caller 另有契約，不能套用這個正常物理入口。
///
/// Go：`native_physical_scene.go:18`，`func SelectNativePhysicalScene(initial byte, actor, target *Unit, actorControl, targetControl [4]byte, headerByte1 byte, ) (NativePhysicalSceneSelection, error)`
pub fn select_native_physical_scene(
    initial: u8,
    actor: Option<&Unit>,
    target: Option<&Unit>,
    actor_control: [u8; 4],
    target_control: [u8; 4],
    header_byte1: u8,
) -> Result<NativePhysicalSceneSelection> {
    let (Some(actor), Some(target)) = (actor, target) else {
        return Err(battle_err!("native physical scene raw input unavailable"));
    };
    if initial > 3
        || !actor.has_native_record_byte6
        || !target.has_native_record_byte6
        || !(0..=255).contains(&actor.battle_fig)
        || !(0..=255).contains(&target.battle_fig)
    {
        return Err(battle_err!("native physical scene raw input unavailable"));
    }
    let actor_gate = native_command_background_gate(Some(actor))?;
    let target_gate = native_command_background_gate(Some(target))?;
    let (mut primary_gate, mut secondary_gate) = (actor_gate, target_gate);
    let (mut primary_control, mut secondary_control) = (actor_control, target_control);
    if actor.native_record_byte6 != 0 {
        (primary_gate, secondary_gate) = (target_gate, actor_gate);
        (primary_control, secondary_control) = (target_control, actor_control);
    }
    let choose_bg = |gate: bool, control: [u8; 4]| if gate && initial != 0 { initial } else { control[2] };
    let mut selection = NativePhysicalSceneSelection {
        primary_bg: choose_bg(primary_gate, primary_control),
        secondary_bg: choose_bg(secondary_gate, secondary_control),
        base_bg: 0,
        tai: secondary_control[2],
        has_separate_backgrounds: header_byte1 != 0,
    };
    // 0x28C01..0x28C06 在 initial==0 時只改 secondary BG 的 stack local。
    // EAX 仍保留 initial，0x28C41 把它傳給 TAI；不能共用 chooseBG。
    if secondary_gate {
        selection.tai = initial;
    }
    selection.base_bg = selection.primary_bg;
    if selection.has_separate_backgrounds && actor.native_record_byte6 != 0 {
        selection.base_bg = selection.secondary_bg;
    }
    Ok(selection)
}

#[cfg(test)]
mod native_physical_scene_test;
