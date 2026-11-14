//! Go 來源：`remake/internal/campaign/native_class_confirm.go`（807 行）。

use super::*;

/// NativePreparationPortraitLayout exposes the same fixed caller offset and
/// right-to-left rule used by ComposeNativePreparationConfirmationDialogue.
///
/// Go：`native_class_confirm.go:16`，`func NativePreparationPortraitLayout(portrait dato.Frame) (image.Rectangle, bool, error)`
pub fn native_preparation_portrait_layout(portrait: &dato::Frame) -> Result<(go_image::Rectangle, bool)> {
    todo!("campaign/native_class_confirm.go:NativePreparationPortraitLayout")
}

pub(crate) const NATIVE_CONFIRM_BASE_X: i64 = 248;

pub(crate) const NATIVE_CONFIRM_Y: i64 = 168;

pub(crate) const NATIVE_PREPARATION_QUESTION_INDEX: i64 = 658;

pub(crate) const NATIVE_PREPARATION_QUESTION_X: i64 = 95;

pub(crate) const NATIVE_PREPARATION_QUESTION_Y: i64 = 119;

pub(crate) const NATIVE_DEPARTURE_QUESTION_INDEX: i64 = 513;

pub(crate) const NATIVE_DEPARTURE_QUESTION_X: i64 = 95;

pub(crate) const NATIVE_RECORD_QUESTION_INDEX: i64 = 410;

pub(crate) const NATIVE_RECORD_QUESTION_X: i64 = 100;

pub(crate) const NATIVE_BATTLE_END_QUESTION_INDEX: i64 = 0x1a3;

pub(crate) const NATIVE_BATTLE_END_ACCEPTED_INDEX: i64 = 0x1a4;

pub(crate) const NATIVE_BATTLE_END_CANCELED_INDEX: i64 = 0x19c;

pub(crate) const NATIVE_BATTLE_EXIT_QUESTION_INDEX: i64 = 0x19f;

pub(crate) const NATIVE_BATTLE_EXIT_ACCEPTED_INDEX: i64 = 0x1a0;

pub(crate) const NATIVE_BATTLE_SAVE_QUESTION_INDEX: i64 = 0x19a;

pub(crate) const NATIVE_BATTLE_SAVE_ACCEPTED_INDEX: i64 = 0x19b;

pub(crate) const NATIVE_BATTLE_LOAD_QUESTION_INDEX: i64 = 0x19d;

pub(crate) const NATIVE_BATTLE_LOAD_ACCEPTED_INDEX: i64 = 0x19e;

pub(crate) const NATIVE_BATTLE_MARCH_QUESTION_INDEX: i64 = 0x1a1;

pub(crate) const NATIVE_BATTLE_MARCH_ACCEPTED_INDEX: i64 = 0x1a2;

pub(crate) const NATIVE_DEATH_REWARD_DISCARD_INDEX: i64 = 0x1b1;

pub(crate) const NATIVE_DEATH_REWARD_ABANDON_INDEX: i64 = 0x1b2;

pub(crate) const NATIVE_BATTLE_END_QUESTION_X: i64 = 99;

pub(crate) const NATIVE_BATTLE_END_QUESTION_Y: i64 = 127;

pub(crate) const NATIVE_BATTLE_END_RESPONSE_Y: i64 = 146;

/// ComposeNativeBattleEndTurnQuestion 重現 END caller 透過0x15f84把
/// FDTXT#0x1a3寫到畫面座標(99,127)；輸入必須先是0x1956b(0x4b)完成的
/// DATO#75對話框畫面。
///
/// Go：`native_class_confirm.go:60`，`func ComposeNativeBattleEndTurnQuestion( dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_battle_end_turn_question(
    dialogue: &[u8],
    portrait: &dato::Frame,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
) -> Result<Vec<u8>> {
    compose_native_battle_system_question(dialogue, portrait, strings, font, NATIVE_BATTLE_END_QUESTION_INDEX)
}

/// ComposeNativeBattleExitQuestion 重現 sub_19DF7 selector3 在相同原版對話座標
/// 寫入的 FDTXT#0x19f 問句。
///
/// Go：`native_class_confirm.go:70`，`func ComposeNativeBattleExitQuestion( dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_battle_exit_question(
    dialogue: &[u8],
    portrait: &dato::Frame,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
) -> Result<Vec<u8>> {
    compose_native_battle_system_question(dialogue, portrait, strings, font, NATIVE_BATTLE_EXIT_QUESTION_INDEX)
}

/// ComposeNativeBattleCurrentSaveQuestion reproduces sub_19DF7 selector1's
/// FDTXT#0x19a question at the shared native battle-system dialogue position.
///
/// Go：`native_class_confirm.go:80`，`func ComposeNativeBattleCurrentSaveQuestion( dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_battle_current_save_question(
    dialogue: &[u8],
    portrait: &dato::Frame,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
) -> Result<Vec<u8>> {
    compose_native_battle_system_question(dialogue, portrait, strings, font, NATIVE_BATTLE_SAVE_QUESTION_INDEX)
}

/// ComposeNativeBattleCurrentLoadQuestion reproduces sub_19DF7 selector2's
/// FDTXT#0x19d question at the shared native battle-system dialogue position.
///
/// Go：`native_class_confirm.go:90`，`func ComposeNativeBattleCurrentLoadQuestion( dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_battle_current_load_question(
    dialogue: &[u8],
    portrait: &dato::Frame,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
) -> Result<Vec<u8>> {
    compose_native_battle_system_question(dialogue, portrait, strings, font, NATIVE_BATTLE_LOAD_QUESTION_INDEX)
}

/// ComposeNativeBattleGroupMarchQuestion 重現 sub_16F55 selector1 的
/// FDTXT#0x1a1 問句。
///
/// Go：`native_class_confirm.go:100`，`func ComposeNativeBattleGroupMarchQuestion( dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_battle_group_march_question(
    dialogue: &[u8],
    portrait: &dato::Frame,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
) -> Result<Vec<u8>> {
    compose_native_battle_system_question(dialogue, portrait, strings, font, NATIVE_BATTLE_MARCH_QUESTION_INDEX)
}

/// ComposeNativeDeathRewardDiscardQuestion reproduces sub_1AA1D's full
/// inventory FDTXT#0x1B1 question at (99,127). Despite the historical
/// worklist wording, the original text and following selector ask which item
/// on the killer should be discarded; no companion selector is involved.
///
/// Go：`native_class_confirm.go:112`，`func ComposeNativeDeathRewardDiscardQuestion( dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_death_reward_discard_question(
    dialogue: &[u8],
    portrait: &dato::Frame,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
) -> Result<Vec<u8>> {
    compose_native_battle_system_question(dialogue, portrait, strings, font, NATIVE_DEATH_REWARD_DISCARD_INDEX)
}

/// NativeDeathRewardAbandonResponseFrames reproduces the NO/Escape branch
/// that writes FDTXT#0x1B2 below the still-visible question at (99,146).
///
/// Go：`native_class_confirm.go:122`，`func NativeDeathRewardAbandonResponseFrames( question []byte, strings *fdtxt.Strings, font *fdtxt.Font, item int, ) ([][]byte, error)`
pub fn native_death_reward_abandon_response_frames(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: &fdtxt::Font,
    item: i64,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativeDeathRewardAbandonResponseFrames")
}

/// ComposeNativeDeathRewardSelectorCancel reproduces 0x1AB1F: after the item
/// selector closes, the original opens a fresh dialogue and writes the same
/// FDTXT#0x1B2 at the question position (99,127).
///
/// Go：`native_class_confirm.go:135`，`func ComposeNativeDeathRewardSelectorCancel( dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, item int, ) ([]byte, error)`
pub fn compose_native_death_reward_selector_cancel(
    dialogue: &[u8],
    portrait: &dato::Frame,
    strings: Option<&fdtxt::Strings>,
    font: &fdtxt::Font,
    item: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:ComposeNativeDeathRewardSelectorCancel")
}

/// nativeDeathRewardAbandonWords reproduces the caller-owned [0x53AD9]
/// substitution before FDTXT#0x1B2: FFFC is the reward item name, not spacing.
///
/// Go：`native_class_confirm.go:157`，`func nativeDeathRewardAbandonWords(strings *fdtxt.Strings, item int) ([]uint16, error)`
pub(crate) fn native_death_reward_abandon_words(strings: Option<&fdtxt::Strings>, item: i64) -> Result<Vec<u16>> {
    todo!("campaign/native_class_confirm.go:nativeDeathRewardAbandonWords")
}

/// Go：`native_class_confirm.go:180`，`func composeNativeBattleSystemQuestion( dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, index int, ) ([]byte, error)`
pub(crate) fn compose_native_battle_system_question(
    dialogue: &[u8],
    portrait: &dato::Frame,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    index: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:composeNativeBattleSystemQuestion")
}

/// ComposeNativeBattleEndTurnResponse 在(99,146)套用分支專屬回覆：
/// YES使用FDTXT#0x1a4，NO／Escape使用FDTXT#0x19c。
///
/// Go：`native_class_confirm.go:200`，`func ComposeNativeBattleEndTurnResponse( question []byte, strings *fdtxt.Strings, font *fdtxt.Font, accepted bool, ) ([]byte, error)`
pub fn compose_native_battle_end_turn_response(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    accepted: bool,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:ComposeNativeBattleEndTurnResponse")
}

/// NativeBattleEndTurnResponseFrames 保留0x15F84在END分支回覆期間直接寫入VGA的
/// 可見過程：每個普通字形發布一個畫格；FFFE只把目的位置移到下一行。
///
/// Go：`native_class_confirm.go:212`，`func NativeBattleEndTurnResponseFrames( question []byte, strings *fdtxt.Strings, font *fdtxt.Font, accepted bool, ) ([][]byte, error)`
pub fn native_battle_end_turn_response_frames(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    accepted: bool,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativeBattleEndTurnResponseFrames")
}

/// NativeBattleExitResponseFrames 保留 selector3 逐字發布的 FDTXT#0x1a0
/// 接受回覆；取消則與 END 共用 FDTXT#0x19c。
///
/// Go：`native_class_confirm.go:224`，`func NativeBattleExitResponseFrames( question []byte, strings *fdtxt.Strings, font *fdtxt.Font, accepted bool, ) ([][]byte, error)`
pub fn native_battle_exit_response_frames(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    accepted: bool,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativeBattleExitResponseFrames")
}

/// NativeBattleCurrentSaveResponseFrames uses FDTXT#0x19b only after the
/// native FD2.SAV replacement succeeded. Cancellation shares FDTXT#0x19c.
///
/// Go：`native_class_confirm.go:236`，`func NativeBattleCurrentSaveResponseFrames( question []byte, strings *fdtxt.Strings, font *fdtxt.Font, accepted bool, ) ([][]byte, error)`
pub fn native_battle_current_save_response_frames(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    accepted: bool,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativeBattleCurrentSaveResponseFrames")
}

/// NativeBattleCurrentLoadResponseFrames uses FDTXT#0x19e only for the
/// accepted restore. Cancellation shares FDTXT#0x19c.
///
/// Go：`native_class_confirm.go:248`，`func NativeBattleCurrentLoadResponseFrames( question []byte, strings *fdtxt.Strings, font *fdtxt.Font, accepted bool, ) ([][]byte, error)`
pub fn native_battle_current_load_response_frames(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    accepted: bool,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativeBattleCurrentLoadResponseFrames")
}

/// NativeBattleGroupMarchResponseFrames 使用 selector1 接受文字 FDTXT#0x1a2；
/// 取消仍共用 FDTXT#0x19c。
///
/// Go：`native_class_confirm.go:260`，`func NativeBattleGroupMarchResponseFrames( question []byte, strings *fdtxt.Strings, font *fdtxt.Font, accepted bool, ) ([][]byte, error)`
pub fn native_battle_group_march_response_frames(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    accepted: bool,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativeBattleGroupMarchResponseFrames")
}

/// Go：`native_class_confirm.go:270`，`func nativeBattleSystemResponseFrames( question []byte, strings *fdtxt.Strings, font *fdtxt.Font, index int, ) ([][]byte, error)`
pub(crate) fn native_battle_system_response_frames(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    index: i64,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:nativeBattleSystemResponseFrames")
}

/// ComposeNativeTreasureItemQuestion 消費 0x190AC 的可見／隱藏寶物問句。
///
/// Go：`native_class_confirm.go:284`，`func ComposeNativeTreasureItemQuestion(dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, hidden bool) ([]byte, error)`
pub fn compose_native_treasure_item_question(
    dialogue: &[u8],
    portrait: &dato::Frame,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    hidden: bool,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:ComposeNativeTreasureItemQuestion")
}

/// NativeTreasureItemResponseFrames 依 0x191DD 的 item+0xB5 展開 FFFC 名稱。
///
/// Go：`native_class_confirm.go:293`，`func NativeTreasureItemResponseFrames(question []byte, strings *fdtxt.Strings, font *fdtxt.Font, item int, hidden bool) ([][]byte, error)`
pub fn native_treasure_item_response_frames(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    item: i64,
    hidden: bool,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativeTreasureItemResponseFrames")
}

/// NativeTreasureGoldResponseFrames 是 0x190AC 金錢分支（寶物列 +0x53 == 1）的回覆：金額非 0 時
/// 0x19477 先把金額存進 [0x53AE1]，顯示 0x1AA（格子旗標 0x20 成立）／0x1AE（隱藏），FFFA 展開成
/// 十進位；金額為 0 顯示 0x1AB／0x1AF。加金在 0x196CB 關框之後（0x194E8），不在這裡。
///
/// Go：`native_class_confirm.go:323`，`func NativeTreasureGoldResponseFrames(question []byte, strings *fdtxt.Strings, font *fdtxt.Font, value int, hidden bool) ([][]byte, error)`
pub fn native_treasure_gold_response_frames(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    value: i64,
    hidden: bool,
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativeTreasureGoldResponseFrames")
}

/// Go：`native_class_confirm.go:353`，`func nativeBattleResponseWordFrames(question []byte, font *fdtxt.Font, words []uint16) ([][]byte, error)`
pub(crate) fn native_battle_response_word_frames(
    question: &[u8],
    font: &fdtxt::Font,
    words: &[u16],
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:nativeBattleResponseWordFrames")
}

/// NativeClassConfirmationOpeningFrames reproduces 0x19953's four opening
/// presents with FDOTHER#2 raw cells 16/17.
///
/// Go：`native_class_confirm.go:383`，`func NativeClassConfirmationOpeningFrames(background []byte, cells []fdother.RawCell) ([][]byte, error)`
pub fn native_class_confirmation_opening_frames(background: &[u8], cells: &[fdother::RawCell]) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativeClassConfirmationOpeningFrames")
}

/// NativeClassConfirmationClosingFrames reproduces 0x197e5's four presented
/// positions. The caller owns the post-close background restore.
///
/// Go：`native_class_confirm.go:403`，`func NativeClassConfirmationClosingFrames(background []byte, cells []fdother.RawCell) ([][]byte, error)`
pub fn native_class_confirmation_closing_frames(background: &[u8], cells: &[fdother::RawCell]) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativeClassConfirmationClosingFrames")
}

/// ComposeNativeClassConfirmationFrame draws FDTXT #594 with its verified
/// FFFC dynamic-name substitution, then the normal/pulsed choice cells
/// 48/49 and 51/52 used by 0x19953's input loop.
///
/// Go：`native_class_confirm.go:424`，`func ComposeNativeClassConfirmationFrame( background []byte, cells []fdother.RawCell, strings *fdtxt.Strings, font *fdtxt.Font, nameTextIndex, selected, pulse int, ) ([]byte, error)`
pub fn compose_native_class_confirmation_frame(
    background: &[u8],
    cells: &[fdother::RawCell],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    name_text_index: i64,
    selected: i64,
    pulse: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:ComposeNativeClassConfirmationFrame")
}

/// ComposeNativeConfirmationChoices applies 0x19953's stable choice cells over
/// a caller-owned question frame.
///
/// Go：`native_class_confirm.go:446`，`func ComposeNativeConfirmationChoices( question []byte, cells []fdother.RawCell, selected, pulse int, ) ([]byte, error)`
pub fn compose_native_confirmation_choices(
    question: &[u8],
    cells: &[fdother::RawCell],
    selected: i64,
    pulse: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:ComposeNativeConfirmationChoices")
}

/// ComposeNativePreparationConfirmationQuestion reproduces 0x31d3c's
/// 0x15f84 call: FDTXT index 0x292 is drawn at framebuffer address 0xa951f,
/// which is screen coordinate (95,119), before 0x19953 owns the two choices.
///
/// Go：`native_class_confirm.go:474`，`func ComposeNativePreparationConfirmationQuestion( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_preparation_confirmation_question(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
) -> Result<Vec<u8>> {
    compose_native_preparation_question(
        background,
        dialogue_cells,
        portrait,
        strings,
        font,
        NATIVE_PREPARATION_QUESTION_INDEX,
        NATIVE_PREPARATION_QUESTION_X,
    )
}

/// ComposeNativePreparationDepartureQuestion reproduces 0x2d0d1's
/// town-backed FDTXT index 0x201 prompt at 0xa951f.
///
/// Go：`native_class_confirm.go:489`，`func ComposeNativePreparationDepartureQuestion( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_preparation_departure_question(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
) -> Result<Vec<u8>> {
    compose_native_preparation_question(
        background,
        dialogue_cells,
        portrait,
        strings,
        font,
        NATIVE_DEPARTURE_QUESTION_INDEX,
        NATIVE_DEPARTURE_QUESTION_X,
    )
}

/// ComposeNativePreparationRecordQuestion reproduces 0x2cc29's standalone
/// FDTXT index 0x19a prompt at 0xa9524.
///
/// Go：`native_class_confirm.go:504`，`func ComposeNativePreparationRecordQuestion( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_preparation_record_question(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
) -> Result<Vec<u8>> {
    compose_native_preparation_question(
        background,
        dialogue_cells,
        portrait,
        strings,
        font,
        NATIVE_RECORD_QUESTION_INDEX,
        NATIVE_RECORD_QUESTION_X,
    )
}

/// ComposeNativePreparationRequiredMessage消費31DBE的共享文字0x291，
/// 31E1B寫入的名字索引是原生identity+1。此訊息沒有YES／NO選項。
///
/// Go：`native_class_confirm.go:519`，`func ComposeNativePreparationRequiredMessage( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, identity int, ) ([]byte, error)`
pub fn compose_native_preparation_required_message(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    identity: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:ComposeNativePreparationRequiredMessage")
}

/// Go：`native_class_confirm.go:561`，`func composeNativePreparationQuestion( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, textIndex, textX int, ) ([]byte, error)`
pub(crate) fn compose_native_preparation_question(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    text_index: i64,
    text_x: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:composeNativePreparationQuestion")
}

/// ComposeNativePreparationConfirmationDialogue returns 0x1956b(0x4b)'s
/// stable dialogue target before 0x15f84 writes FDTXT index 0x292.
///
/// Go：`native_class_confirm.go:607`，`func ComposeNativePreparationConfirmationDialogue( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, ) ([]byte, error)`
pub fn compose_native_preparation_confirmation_dialogue(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
) -> Result<Vec<u8>> {
    compose_native_church_dialogue_overlay_at(
        background,
        dialogue_cells,
        portrait,
        native_facility_portrait_offset_inner(0x4b),
    )
}

/// ComposeNativeConfirmationPortrait reproduces the lower-frame mouth write
/// in sub_19953 without changing the caller-owned question or source buffer.
///
/// Go：`native_class_confirm.go:619`，`func ComposeNativeConfirmationPortrait(frame []byte, portrait dato.Frame) ([]byte, error)`
pub fn compose_native_confirmation_portrait(frame: &[u8], portrait: &dato::Frame) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:ComposeNativeConfirmationPortrait")
}

/// ComposeNativePreparationConfirmationFrame adds the stable 0x19953 choice
/// state to the caller-owned preparation screen. Opening/closing presentation
/// remains caller-owned and is not implied by this compositor.
///
/// Go：`native_class_confirm.go:633`，`func ComposeNativePreparationConfirmationFrame( background []byte, cells []fdother.RawCell, dialogueCells []fdother.RawCell, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, selected, pulse int, ) ([]byte, error)`
pub fn compose_native_preparation_confirmation_frame(
    background: &[u8],
    cells: &[fdother::RawCell],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    selected: i64,
    pulse: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:ComposeNativePreparationConfirmationFrame")
}

/// NativePreparationConfirmationOpeningFrames preserves the caller sequence:
/// 0x1956b reveals the dialogue in six bands, then 0x19953 presents four
/// widening choice frames over the completed question.
///
/// Go：`native_class_confirm.go:654`，`func NativePreparationConfirmationOpeningFrames( source, dialogue, question []byte, cells []fdother.RawCell, ) ([][]byte, error)`
pub fn native_preparation_confirmation_opening_frames(
    source: &[u8],
    dialogue: &[u8],
    question: &[u8],
    cells: &[fdother::RawCell],
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativePreparationConfirmationOpeningFrames")
}

/// NativePreparationConfirmationClosingFrames preserves 0x197e5's four
/// choice frames followed by 0x2d31b's five dialogue bands. The caller still
/// owns the final untouched source presentation and continuation.
///
/// Go：`native_class_confirm.go:672`，`func NativePreparationConfirmationClosingFrames( source, dialogue, question []byte, cells []fdother.RawCell, ) ([][]byte, error)`
pub fn native_preparation_confirmation_closing_frames(
    source: &[u8],
    dialogue: &[u8],
    question: &[u8],
    cells: &[fdother::RawCell],
) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_confirm.go:NativePreparationConfirmationClosingFrames")
}

/// ComposeNativeReviveConfirmationQuestion reproduces FDTXT590 with the
/// selected actor (-4/FFFC) and fee (-6/FFFA) dynamic substitutions.
///
/// Go：`native_class_confirm.go:689`，`func ComposeNativeReviveConfirmationQuestion( background []byte, strings *fdtxt.Strings, font *fdtxt.Font, nameTextIndex, fee int, ) ([]byte, error)`
pub fn compose_native_revive_confirmation_question(
    background: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    name_text_index: i64,
    fee: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:ComposeNativeReviveConfirmationQuestion")
}

/// ComposeNativeClassConfirmationQuestion reproduces the class caller's
/// FDTXT#594 draw before 0x19953 starts its choice-cell opening.
///
/// Go：`native_class_confirm.go:752`，`func ComposeNativeClassConfirmationQuestion( background []byte, strings *fdtxt.Strings, font *fdtxt.Font, nameTextIndex int, ) ([]byte, error)`
pub fn compose_native_class_confirmation_question(
    background: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    name_text_index: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_confirm.go:ComposeNativeClassConfirmationQuestion")
}

/// Go：`native_class_confirm.go:793`，`func validateNativeClassConfirmationAssets(background []byte, cells []fdother.RawCell) error`
pub(crate) fn validate_native_class_confirmation_assets(background: &[u8], cells: &[fdother::RawCell]) -> Result<()> {
    todo!("campaign/native_class_confirm.go:validateNativeClassConfirmationAssets")
}
