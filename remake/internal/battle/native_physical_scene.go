package battle

import "errors"

// NativePhysicalSceneSelection 保留正常 sub_28A6C 的雙背景與台座選擇。
// 來源與 READY 規格：fd2_physical_background_selection_20261004.json。
// PrimaryBG／SecondaryBG 在交換前分別對應 IDA LE 0x28CCC／0x28DA2 的來源。
type NativePhysicalSceneSelection struct {
	PrimaryBG              byte
	SecondaryBG            byte
	BaseBG                 byte
	TAI                    byte
	HasSeparateBackgrounds bool
}

// SelectNativePhysicalScene 只消費原始 record 與 control，不改戰鬥狀態。
// 540FF 非零的特殊 caller 另有契約，不能套用這個正常物理入口。
func SelectNativePhysicalScene(initial byte, actor, target *Unit,
	actorControl, targetControl [4]byte, headerByte1 byte,
) (NativePhysicalSceneSelection, error) {
	if initial > 3 || actor == nil || target == nil ||
		!actor.HasNativeRecordByte6 || !target.HasNativeRecordByte6 ||
		actor.BattleFig < 0 || actor.BattleFig > 255 || target.BattleFig < 0 || target.BattleFig > 255 {
		return NativePhysicalSceneSelection{}, errors.New("native physical scene raw input unavailable")
	}
	actorGate, err := NativeCommandBackgroundGate(actor)
	if err != nil {
		return NativePhysicalSceneSelection{}, err
	}
	targetGate, err := NativeCommandBackgroundGate(target)
	if err != nil {
		return NativePhysicalSceneSelection{}, err
	}
	primaryGate, secondaryGate := actorGate, targetGate
	primaryControl, secondaryControl := actorControl, targetControl
	if actor.NativeRecordByte6 != 0 {
		primaryGate, secondaryGate = targetGate, actorGate
		primaryControl, secondaryControl = targetControl, actorControl
	}
	chooseBG := func(gate bool, control [4]byte) byte {
		if gate && initial != 0 {
			return initial
		}
		return control[2]
	}
	selection := NativePhysicalSceneSelection{
		PrimaryBG:              chooseBG(primaryGate, primaryControl),
		SecondaryBG:            chooseBG(secondaryGate, secondaryControl),
		TAI:                    secondaryControl[2],
		HasSeparateBackgrounds: headerByte1 != 0,
	}
	// 0x28C01..0x28C06 在 initial==0 時只改 secondary BG 的 stack local。
	// EAX 仍保留 initial，0x28C41 把它傳給 TAI；不能共用 chooseBG。
	if secondaryGate {
		selection.TAI = initial
	}
	selection.BaseBG = selection.PrimaryBG
	if selection.HasSeparateBackgrounds && actor.NativeRecordByte6 != 0 {
		selection.BaseBG = selection.SecondaryBG
	}
	return selection, nil
}
