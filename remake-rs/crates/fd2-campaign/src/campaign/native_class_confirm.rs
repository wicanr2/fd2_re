//! Go 來源：`remake/internal/campaign/native_class_confirm.go`（807 行）。

use super::*;

/// NativePreparationPortraitLayout exposes the same fixed caller offset and
/// right-to-left rule used by ComposeNativePreparationConfirmationDialogue.
///
/// Go：`native_class_confirm.go:16`，`func NativePreparationPortraitLayout(portrait dato.Frame) (image.Rectangle, bool, error)`
pub fn native_preparation_portrait_layout(portrait: &dato::Frame) -> Result<(go_image::Rectangle, bool)> {
    if portrait.width != 80 || portrait.height != 80 || portrait.pixels.len() != 80 * 80 {
        return Err(campaign_err!("campaign: preparation portrait shape unavailable"));
    }
    let offset = native_facility_portrait_offset_inner(0x4b);
    let (mut x, y) = (offset % 320, offset / 320);
    let mirror = offset == NATIVE_LOWER_PORTRAIT_RIGHT_EDGE;
    if mirror {
        x -= portrait.width - 1;
    }
    // Go `image.Rect` 會把 min／max 排序；寬高為正，兩角已經有序。
    Ok((
        go_image::Rectangle {
            min: go_image::Point { x, y },
            max: go_image::Point { x: x + portrait.width, y: y + portrait.height },
        },
        mirror,
    ))
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
    let words = native_death_reward_abandon_words(strings, item)?;
    native_battle_response_word_frames(question, font, &words)
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
    let words = native_death_reward_abandon_words(strings, item)?;
    let mut frame = dialogue.to_vec();
    let mut frame = compose_native_church_words_at(
        &mut frame,
        font,
        &words,
        NATIVE_BATTLE_END_QUESTION_Y * 320 + NATIVE_BATTLE_END_QUESTION_X,
    )?;
    blit_native_dialogue_portrait_at(&mut frame, portrait, native_facility_portrait_offset_inner(0x4b))?;
    Ok(frame)
}

/// nativeDeathRewardAbandonWords reproduces the caller-owned [0x53AD9]
/// substitution before FDTXT#0x1B2: FFFC is the reward item name, not spacing.
///
/// Go：`native_class_confirm.go:157`，`func nativeDeathRewardAbandonWords(strings *fdtxt.Strings, item int) ([]uint16, error)`
pub(crate) fn native_death_reward_abandon_words(strings: Option<&fdtxt::Strings>, item: i64) -> Result<Vec<u16>> {
    let strings = match strings {
        Some(s) if (0..=0xff).contains(&item) => s,
        _ => return Err(campaign_err!("campaign: invalid death-reward abandon source")),
    };
    let words = strings.words(NATIVE_DEATH_REWARD_ABANDON_INDEX)?;
    let name = strings.words(item + 0xb5)?;
    Ok(expand_fffc(&words, &name))
}

/// Go：`native_class_confirm.go:180`，`func composeNativeBattleSystemQuestion( dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, index int, ) ([]byte, error)`
pub(crate) fn compose_native_battle_system_question(
    dialogue: &[u8],
    portrait: &dato::Frame,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    index: i64,
) -> Result<Vec<u8>> {
    let mut frame = dialogue.to_vec();
    let mut frame = compose_native_church_text_at(
        &mut frame,
        strings,
        font,
        index,
        NATIVE_BATTLE_END_QUESTION_Y * 320 + NATIVE_BATTLE_END_QUESTION_X,
    )?;
    // 0x17204緊接著呼叫0x16559(0)，所以frame0在文字後再寫一次。
    blit_native_dialogue_portrait_at(&mut frame, portrait, native_facility_portrait_offset_inner(0x4b))?;
    Ok(frame)
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
    let mut frames = native_battle_end_turn_response_frames(question, strings, font, accepted)?;
    // nativeBattleResponseWordFrames 保證至少一格。
    Ok(frames.pop().unwrap_or_default())
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
    let index = if accepted { NATIVE_BATTLE_END_ACCEPTED_INDEX } else { NATIVE_BATTLE_END_CANCELED_INDEX };
    native_battle_system_response_frames(question, strings, font, index)
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
    let index = if accepted { NATIVE_BATTLE_EXIT_ACCEPTED_INDEX } else { NATIVE_BATTLE_END_CANCELED_INDEX };
    native_battle_system_response_frames(question, strings, font, index)
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
    let index = if accepted { NATIVE_BATTLE_SAVE_ACCEPTED_INDEX } else { NATIVE_BATTLE_END_CANCELED_INDEX };
    native_battle_system_response_frames(question, strings, font, index)
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
    let index = if accepted { NATIVE_BATTLE_LOAD_ACCEPTED_INDEX } else { NATIVE_BATTLE_END_CANCELED_INDEX };
    native_battle_system_response_frames(question, strings, font, index)
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
    let index = if accepted { NATIVE_BATTLE_MARCH_ACCEPTED_INDEX } else { NATIVE_BATTLE_END_CANCELED_INDEX };
    native_battle_system_response_frames(question, strings, font, index)
}

/// Go：`native_class_confirm.go:270`，`func nativeBattleSystemResponseFrames( question []byte, strings *fdtxt.Strings, font *fdtxt.Font, index int, ) ([][]byte, error)`
pub(crate) fn native_battle_system_response_frames(
    question: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    index: i64,
) -> Result<Vec<Vec<u8>>> {
    let (Some(strings), Some(font)) = (strings, font) else {
        return Err(campaign_err!("campaign: 原版戰鬥結束回覆資產無效"));
    };
    if question.len() != 320 * 200 {
        return Err(campaign_err!("campaign: 原版戰鬥結束回覆資產無效"));
    }
    let words = strings.words(index)?;
    native_battle_response_word_frames(question, font, &words)
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
    let index = if hidden { 0x1ac } else { 0x1a5 };
    compose_native_battle_system_question(dialogue, portrait, strings, font, index)
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
    let (strings, font) = match (strings, font) {
        (Some(s), Some(f)) if question.len() == 320 * 200 && (0..=255).contains(&item) => (s, f),
        _ => return Err(campaign_err!("campaign: invalid treasure response source")),
    };
    let index = if hidden { 0x1ad } else { 0x1a6 };
    let words = strings.words(index)?;
    let name = strings.words(item + 0xb5)?;
    native_battle_response_word_frames(question, font, &expand_fffc(&words, &name))
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
    let (strings, font) = match (strings, font) {
        (Some(s), Some(f)) if question.len() == 320 * 200 && (0..=0xffff).contains(&value) => (s, f),
        _ => return Err(campaign_err!("campaign: invalid treasure gold response source")),
    };
    let mut index = if value == 0 { 0x1ab } else { 0x1aa };
    if hidden {
        index += 4;
    }
    let words = strings.words(index)?;
    let digits = go_decimal_words(value);
    let mut expanded = Vec::with_capacity(words.len() + digits.len());
    for &word in &words {
        if word == 0xfffa {
            expanded.extend_from_slice(&digits);
        } else {
            expanded.push(word);
        }
    }
    native_battle_response_word_frames(question, font, &expanded)
}

/// Go：`native_class_confirm.go:353`，`func nativeBattleResponseWordFrames(question []byte, font *fdtxt.Font, words []uint16) ([][]byte, error)`
pub(crate) fn native_battle_response_word_frames(
    question: &[u8],
    font: &fdtxt::Font,
    words: &[u16],
) -> Result<Vec<Vec<u8>>> {
    let mut frame = question.to_vec();
    let mut frames = Vec::with_capacity(words.len());
    let (mut line, mut column) = (0i64, 0i64);
    let style = fdtxt::NativeGlyphStyle { foreground: 205, shadow: 76, background: 74 };
    for &word in words {
        if word == 0xfffe {
            line += 1;
            column = 0;
            continue;
        }
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!("campaign: 不支援的戰鬥結束回覆控制碼 {}", battle::go_sharp_x(word as i64)));
        }
        let offset = NATIVE_BATTLE_END_RESPONSE_Y * 320
            + NATIVE_BATTLE_END_QUESTION_X
            + line * 19 * 320
            + column * fdtxt::GLYPH_WIDTH as i64;
        font.blit_native_glyph(&mut frame, 320, offset, word as i64, style)?;
        frames.push(frame.clone());
        column += 1;
    }
    if frames.is_empty() {
        return Err(campaign_err!("campaign: 原版戰鬥結束回覆沒有可見字形"));
    }
    Ok(frames)
}

/// NativeClassConfirmationOpeningFrames reproduces 0x19953's four opening
/// presents with FDOTHER#2 raw cells 16/17.
///
/// Go：`native_class_confirm.go:383`，`func NativeClassConfirmationOpeningFrames(background []byte, cells []fdother.RawCell) ([][]byte, error)`
pub fn native_class_confirmation_opening_frames(background: &[u8], cells: &[fdother::RawCell]) -> Result<Vec<Vec<u8>>> {
    validate_native_class_confirmation_assets(background, cells)?;
    [4, 8, 12, 16].iter().map(|&spread| native_confirmation_spread_frame(background, cells, spread)).collect()
}

/// NativeClassConfirmationClosingFrames reproduces 0x197e5's four presented
/// positions. The caller owns the post-close background restore.
///
/// Go：`native_class_confirm.go:403`，`func NativeClassConfirmationClosingFrames(background []byte, cells []fdother.RawCell) ([][]byte, error)`
pub fn native_class_confirmation_closing_frames(background: &[u8], cells: &[fdother::RawCell]) -> Result<Vec<Vec<u8>>> {
    validate_native_class_confirmation_assets(background, cells)?;
    [12, 8, 4, 0].iter().map(|&spread| native_confirmation_spread_frame(background, cells, spread)).collect()
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
    validate_native_class_confirmation_assets(background, cells)?;
    if !(0..=1).contains(&selected) || !(0..=1).contains(&pulse) || strings.is_none() || font.is_none() {
        return Err(campaign_err!("campaign: invalid native class confirmation state"));
    }
    let frame = compose_native_class_confirmation_question(background, strings, font, name_text_index)?;
    compose_native_confirmation_choices(&frame, cells, selected, pulse)
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
    validate_native_class_confirmation_assets(question, cells)?;
    if !(0..=1).contains(&selected) || !(0..=1).contains(&pulse) {
        return Err(campaign_err!("campaign: invalid native confirmation choice state"));
    }
    let mut frame = question.to_vec();
    for (option, (base, dx)) in [(48usize, -16i64), (51, 16)].into_iter().enumerate() {
        let mut index = base;
        if option as i64 == selected {
            index += pulse as usize;
        }
        cells[index].blit_at(&mut frame, 320, NATIVE_CONFIRM_BASE_X + dx, NATIVE_CONFIRM_Y)?;
    }
    Ok(frame)
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
    let (strings, font) = match (strings, font) {
        (Some(s), Some(f)) if (1..=31).contains(&identity) => (s, f),
        _ => return Err(campaign_err!("campaign: invalid preparation required-character source")),
    };
    let mut frame = compose_native_preparation_confirmation_dialogue(background, dialogue_cells, portrait)?;
    let words = strings.words(0x291)?;
    let name = strings.words(identity + 1)?;
    let style = fdtxt::NativeGlyphStyle { foreground: 205, shadow: 76, background: 0 };
    for (i, &word) in expand_fffc(&words, &name).iter().enumerate() {
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!(
                "campaign: unsupported required-character control {}",
                battle::go_sharp_x(word as i64)
            ));
        }
        font.blit_native_glyph(
            &mut frame,
            320,
            NATIVE_PREPARATION_QUESTION_Y * 320 + NATIVE_PREPARATION_QUESTION_X + i as i64 * fdtxt::GLYPH_WIDTH as i64,
            word as i64,
            style,
        )?;
    }
    blit_native_dialogue_portrait_at(&mut frame, portrait, native_facility_portrait_offset_inner(0x4b))?;
    Ok(frame)
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
    let (strings, font) = match (strings, font) {
        (Some(s), Some(f)) if background.len() == 320 * 200 => (s, f),
        _ => return Err(campaign_err!("campaign: native preparation confirmation assets are unavailable")),
    };
    let mut frame = compose_native_preparation_confirmation_dialogue(background, dialogue_cells, portrait)?;
    let question = strings.words(text_index)?;
    let style = fdtxt::NativeGlyphStyle { foreground: 205, shadow: 76, background: 0 };
    for (i, &word) in question.iter().enumerate() {
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!(
                "campaign: unsupported preparation confirmation control {}",
                battle::go_sharp_x(word as i64)
            ));
        }
        font.blit_native_glyph(
            &mut frame,
            320,
            NATIVE_PREPARATION_QUESTION_Y * 320 + text_x + i as i64 * fdtxt::GLYPH_WIDTH as i64,
            word as i64,
            style,
        )?;
    }
    // 0x31d70 immediately calls 0x16559(0), so DATO frame zero is written
    // once more after the text layout and owns every overlapping pixel.
    blit_native_dialogue_portrait_at(&mut frame, portrait, native_facility_portrait_offset_inner(0x4b))?;
    Ok(frame)
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
    if frame.len() != 320 * 200 {
        return Err(campaign_err!("campaign: confirmation portrait source is incomplete"));
    }
    let mut result = frame.to_vec();
    blit_native_dialogue_portrait_at(&mut result, portrait, native_facility_portrait_offset_inner(0x4b))?;
    Ok(result)
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
    let question =
        compose_native_preparation_confirmation_question(background, dialogue_cells, portrait, strings, font)?;
    compose_native_confirmation_choices(&question, cells, selected, pulse)
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
    let mut dialogue_frames = native_class_list_opening_frames(source, dialogue)?;
    let choice_frames = native_class_confirmation_opening_frames(question, cells)?;
    dialogue_frames.extend(choice_frames);
    Ok(dialogue_frames)
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
    let mut choice_frames = native_class_confirmation_closing_frames(question, cells)?;
    let dialogue_frames = native_class_list_closing_frames(source, dialogue)?;
    choice_frames.extend(dialogue_frames);
    Ok(choice_frames)
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
    let (strings, font) = match (strings, font) {
        (Some(s), Some(f)) if background.len() == 320 * 200 && fee >= 0 => (s, f),
        _ => return Err(campaign_err!("campaign: native revive confirmation assets/state are invalid")),
    };
    let question = strings.words(590)?;
    let name = strings.words(name_text_index)?;
    let fee_words = go_decimal_words(fee);
    let mut expanded = Vec::with_capacity(question.len() + name.len() + fee_words.len());
    for &word in &question {
        match word {
            0xfffc => expanded.extend_from_slice(&name),
            0xfffa => expanded.extend_from_slice(&fee_words),
            0xfffe => expanded.push(word),
            _ => {
                if word >= fdtxt::CONTROL_MIN {
                    return Err(campaign_err!(
                        "campaign: unsupported revive confirmation control {}",
                        battle::go_sharp_x(word as i64)
                    ));
                }
                expanded.push(word);
            }
        }
    }
    let mut frame = background.to_vec();
    let style = fdtxt::NativeGlyphStyle { foreground: 205, shadow: 76, background: 0 };
    let (mut line, mut column) = (0i64, 0i64);
    for &word in &expanded {
        if word == 0xfffe {
            line += 1;
            column = 0;
            continue;
        }
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!(
                "campaign: dynamic revive text contains control {}",
                battle::go_sharp_x(word as i64)
            ));
        }
        font.blit_native_glyph(
            &mut frame,
            320,
            (119 + line * 19) * 320 + 12 + column * fdtxt::GLYPH_WIDTH as i64,
            word as i64,
            style,
        )?;
        column += 1;
    }
    Ok(frame)
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
    let (strings, font) = match (strings, font) {
        (Some(s), Some(f)) if background.len() == 320 * 200 => (s, f),
        _ => return Err(campaign_err!("campaign: native class confirmation question assets are unavailable")),
    };
    let question = strings.words(594)?;
    let name = strings.words(name_text_index)?;
    let mut expanded = Vec::with_capacity(name.len() + question.len());
    for &word in &question {
        if word == 0xfffc {
            expanded.extend_from_slice(&name);
            continue;
        }
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!(
                "campaign: unsupported class confirmation control {}",
                battle::go_sharp_x(word as i64)
            ));
        }
        expanded.push(word);
    }
    let mut frame = background.to_vec();
    let style = fdtxt::NativeGlyphStyle { foreground: 205, shadow: 76, background: 0 };
    for (i, &word) in expanded.iter().enumerate() {
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!(
                "campaign: dynamic class name contains control {}",
                battle::go_sharp_x(word as i64)
            ));
        }
        font.blit_native_glyph(
            &mut frame,
            320,
            119 * 320 + 12 + i as i64 * fdtxt::GLYPH_WIDTH as i64,
            word as i64,
            style,
        )?;
    }
    Ok(frame)
}

/// Go：`native_class_confirm.go:793`，`func validateNativeClassConfirmationAssets(background []byte, cells []fdother.RawCell) error`
pub(crate) fn validate_native_class_confirmation_assets(background: &[u8], cells: &[fdother::RawCell]) -> Result<()> {
    if background.len() != 320 * 200 || cells.len() <= 52 {
        return Err(campaign_err!("campaign: native class confirmation assets are unavailable"));
    }
    for index in [16usize, 17, 48, 49, 51, 52] {
        let want_height = if index == 16 || index == 17 { 20 } else { 16 };
        if cells[index].width != 24 || cells[index].height != want_height {
            return Err(campaign_err!(
                "campaign: native class confirmation cell {index} is {}x{}",
                cells[index].width,
                cells[index].height
            ));
        }
    }
    Ok(())
}

/// 0x19953／0x197e5 的一個選項展開畫格：從背景複本貼上 FDOTHER#2 原始格 16／17，左右各偏
/// `spread`。Go 的開啟與收合各寫一份同樣的迴圈（`native_class_confirm.go:390`、`:409`）。
fn native_confirmation_spread_frame(background: &[u8], cells: &[fdother::RawCell], spread: i64) -> Result<Vec<u8>> {
    let mut frame = background.to_vec();
    cells[16].blit_at(&mut frame, 320, NATIVE_CONFIRM_BASE_X - spread, NATIVE_CONFIRM_Y)?;
    cells[17].blit_at(&mut frame, 320, NATIVE_CONFIRM_BASE_X + spread, NATIVE_CONFIRM_Y)?;
    Ok(frame)
}

/// 把 `words` 裡的 FFFC 換成 `name`（原版 `[0x53AD9]` 動態名稱）；其他字照抄。
pub(crate) fn expand_fffc(words: &[u16], name: &[u16]) -> Vec<u16> {
    let mut expanded = Vec::with_capacity(words.len() + name.len());
    for &word in words {
        if word == 0xfffc {
            expanded.extend_from_slice(name);
        } else {
            expanded.push(word);
        }
    }
    expanded
}

/// Go `for _, digit := range strconv.Itoa(v) { append(uint16(digit-'0')) }`：十進位每一位一個字；
/// 負號 '-' 減 '0' 為負，轉 uint16 時繞回（呼叫端都先擋掉負值）。
pub(crate) fn go_decimal_words(v: i64) -> Vec<u16> {
    v.to_string().chars().map(|d| (d as i32 - '0' as i32) as u16).collect()
}

#[cfg(test)]
pub(crate) mod native_class_confirm_test;
