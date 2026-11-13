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
    todo!("battle/native_physical_scene.go:SelectNativePhysicalScene")
}
