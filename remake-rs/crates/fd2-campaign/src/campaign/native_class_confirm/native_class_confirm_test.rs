//! 移植自 `remake/internal/campaign/native_class_confirm_test.go`。

use super::*;
use crate::campaign::native_class_list::native_class_list_test::{native_class_list_font, native_class_list_strings};

fn one_pixel_portrait(v: u8) -> dato::Frame {
    dato::Frame { width: 1, height: 1, pixels: vec![v] }
}

fn visible_glyphs(words: &[u16]) -> usize {
    words.iter().filter(|&&w| w < fdtxt::CONTROL_MIN).count()
}

/// Go：`TestNativeTreasureResponseExpandsItemNameBeforePublishing`
#[test]
fn native_treasure_response_expands_item_name_before_publishing() {
    let question = vec![0u8; 320 * 200];
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    for hidden in [false, true] {
        let frames = native_treasure_item_response_frames(&question, Some(&strings), Some(&font), 0xc0, hidden);
        let frames = frames.unwrap_or_else(|e| panic!("hidden={hidden} err={e}"));
        assert_eq!(frames.len(), 4, "hidden={hidden} frames");
        assert!(
            frames[3][146 * 320 + 99 + 3 * 16] == 205 && question[146 * 320 + 99] == 0,
            "name expansion changed placement or source"
        );
    }
}

/// Go：`TestNativeBattleEndTurnFramesDoNotAliasAndRedrawPortrait`
#[test]
fn native_battle_end_turn_frames_do_not_alias_and_redraw_portrait() {
    let dialogue = vec![0u8; 320 * 200];
    let portrait = dato::Frame { width: 3, height: 1, pixels: vec![1, 2, 3] };
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let question = compose_native_battle_end_turn_question(&dialogue, &portrait, &strings, &font).unwrap();
    let question_before = question.clone();
    let mut accepted = compose_native_battle_end_turn_response(&question, Some(&strings), Some(&font), true).unwrap();
    let mut canceled = compose_native_battle_end_turn_response(&question, Some(&strings), Some(&font), false).unwrap();
    assert!(question == question_before, "預先合成YES／NO回覆污染了問句畫面");
    (accepted[0], canceled[1]) = (7, 8);
    assert!(
        !(question[0] == 7 || question[1] == 8 || dialogue[0] == 7 || dialogue[1] == 8),
        "question／accepted／canceled／dialogue仍共用底層像素"
    );
    assert_eq!(question[0x9017], 1, "0x16559(0) right-edge pixel");
    assert_eq!(&question[0x9017 - 2..0x9017 + 1], &[3, 2, 1], "0x4E8E1由右往左寫入");
    assert_eq!(dialogue[0x9017], 0, "問句合成污染dialogue portrait pixel");
}

/// Go：`TestNativeBattleEndTurnResponsePublishesOneFramePerGlyph`
#[test]
fn native_battle_end_turn_response_publishes_one_frame_per_glyph() {
    let question = vec![0u8; 320 * 200];
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    for accepted in [true, false] {
        let mut frames =
            native_battle_end_turn_response_frames(&question, Some(&strings), Some(&font), accepted).unwrap();
        let index = if accepted { NATIVE_BATTLE_END_ACCEPTED_INDEX } else { NATIVE_BATTLE_END_CANCELED_INDEX };
        let words = strings.words(index).unwrap();
        assert_eq!(frames.len(), visible_glyphs(&words), "accepted={accepted} frames vs visible glyphs");
        for i in 1..frames.len() {
            assert!(frames[i - 1] != frames[i], "accepted={accepted} frame {i} did not add a glyph");
        }
        if frames.len() > 1 {
            let before = frames[0].clone();
            let last = frames.len() - 1;
            frames[last][0] = 99;
            assert!(frames[0] == before, "accepted={accepted} progressive frames alias");
        }
    }
}

/// Go：`TestNativeDeathRewardUsesDiscardAndAbandonIndices`
#[test]
fn native_death_reward_uses_discard_and_abandon_indices() {
    let dialogue = vec![0u8; 320 * 200];
    let portrait = one_pixel_portrait(9);
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let question = compose_native_death_reward_discard_question(&dialogue, &portrait, &strings, &font).unwrap();
    let end_question = compose_native_battle_end_turn_question(&dialogue, &portrait, &strings, &font).unwrap();
    assert!(question != end_question, "death reward question did not use FDTXT#0x1B1");
    let response = native_death_reward_abandon_response_frames(&question, Some(&strings), &font, 0xd3);
    let response = response.unwrap_or_else(|e| panic!("FDTXT#0x1B2 response err={e}"));
    assert!(!response.is_empty(), "FDTXT#0x1B2 response frames=0");
    let fresh = compose_native_death_reward_selector_cancel(&dialogue, &portrait, Some(&strings), &font, 0xd3).unwrap();
    assert!(fresh != dialogue, "selector cancel did not compose FDTXT#0x1B2");
    let words = native_death_reward_abandon_words(Some(&strings), 0xd3).unwrap();
    assert_eq!(response.len(), visible_glyphs(&words), "FDTXT#0x1B2 frames vs visible glyphs");
    assert!(dialogue == vec![0u8; 320 * 200], "death reward compositors mutated source dialogue");
}

/// Go：`TestNativeBattleExitUsesSelector3TextIndices`
#[test]
fn native_battle_exit_uses_selector3_text_indices() {
    let dialogue = vec![0u8; 320 * 200];
    let portrait = one_pixel_portrait(9);
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let exit_question = compose_native_battle_exit_question(&dialogue, &portrait, &strings, &font).unwrap();
    let end_question = compose_native_battle_end_turn_question(&dialogue, &portrait, &strings, &font).unwrap();
    assert!(exit_question != end_question, "離場問句誤用了 END 的 FDTXT#0x1a3");
    let accepted = native_battle_exit_response_frames(&exit_question, Some(&strings), Some(&font), true).unwrap();
    assert_eq!(accepted.len(), 3, "離場 YES frames want 3 from FDTXT#0x1a0");
    let canceled = native_battle_exit_response_frames(&exit_question, Some(&strings), Some(&font), false).unwrap();
    assert_eq!(canceled.len(), 1, "離場 NO frames want 1 from shared FDTXT#0x19c");
}

/// Go：`TestNativeBattleGroupMarchUsesSelector1TextIndices`
#[test]
fn native_battle_group_march_uses_selector1_text_indices() {
    let dialogue = vec![0u8; 320 * 200];
    let portrait = one_pixel_portrait(9);
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let question = compose_native_battle_group_march_question(&dialogue, &portrait, &strings, &font).unwrap();
    let exit_question = compose_native_battle_exit_question(&dialogue, &portrait, &strings, &font).unwrap();
    assert!(question != exit_question, "全軍移動問句誤用了離場 FDTXT#0x19f");
    let accepted = native_battle_group_march_response_frames(&question, Some(&strings), Some(&font), true).unwrap();
    assert_eq!(accepted.len(), 2, "全軍移動 YES frames want 2 from FDTXT#0x1a2");
    let canceled = native_battle_group_march_response_frames(&question, Some(&strings), Some(&font), false).unwrap();
    assert_eq!(canceled.len(), 1, "全軍移動 NO frames want 1 from FDTXT#0x19c");
}

/// Go：`TestNativeBattleCurrentSaveUsesSelector1TextIndices`
#[test]
fn native_battle_current_save_uses_selector1_text_indices() {
    let dialogue = vec![0u8; 320 * 200];
    let portrait = one_pixel_portrait(9);
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let question = compose_native_battle_current_save_question(&dialogue, &portrait, &strings, &font).unwrap();
    let exit_question = compose_native_battle_exit_question(&dialogue, &portrait, &strings, &font).unwrap();
    assert!(question != exit_question, "目前戰況存檔問句誤用了離場 FDTXT#0x19f");
    let accepted = native_battle_current_save_response_frames(&question, Some(&strings), Some(&font), true).unwrap();
    assert_eq!(accepted.len(), 2, "目前戰況存檔 YES frames want 2 from FDTXT#0x19b");
    let canceled = native_battle_current_save_response_frames(&question, Some(&strings), Some(&font), false).unwrap();
    assert_eq!(canceled.len(), 1, "目前戰況存檔 NO frames want 1 from FDTXT#0x19c");
}

/// Go：`TestNativeBattleCurrentLoadUsesSelector2TextIndices`
#[test]
fn native_battle_current_load_uses_selector2_text_indices() {
    let dialogue = vec![0u8; 320 * 200];
    let portrait = one_pixel_portrait(9);
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let question = compose_native_battle_current_load_question(&dialogue, &portrait, &strings, &font).unwrap();
    let save_question = compose_native_battle_current_save_question(&dialogue, &portrait, &strings, &font).unwrap();
    assert!(question != save_question, "目前戰況讀檔問句誤用了存檔 FDTXT#0x19a");
    let accepted = native_battle_current_load_response_frames(&question, Some(&strings), Some(&font), true).unwrap();
    assert!(!accepted.is_empty(), "目前戰況讀檔 FDTXT#0x19e 沒有可見接受回覆");
    let canceled = native_battle_current_load_response_frames(&question, Some(&strings), Some(&font), false).unwrap();
    assert_eq!(canceled.len(), 1, "目前戰況讀檔 NO frames want 1 from FDTXT#0x19c");
}

/// Go：`nativeClassConfirmCells`
pub(crate) fn native_class_confirm_cells() -> Vec<fdother::RawCell> {
    let mut cells = vec![fdother::RawCell::default(); 53];
    for index in [16usize, 17, 48, 49, 51, 52] {
        let height = if index == 16 || index == 17 { 20 } else { 16 };
        cells[index] = fdother::RawCell { width: 24, height, pixels: vec![index as u8; (24 * height) as usize] };
    }
    cells
}

/// Go：`TestNativeClassConfirmationOpenCloseCoordinates`
#[test]
fn native_class_confirmation_open_close_coordinates() {
    let background = vec![0u8; 320 * 200];
    let cells = native_class_confirm_cells();
    let open = native_class_confirmation_opening_frames(&background, &cells).unwrap();
    assert_eq!(open.len(), 4, "open frames");
    for (i, spread) in [4usize, 8, 12, 16].into_iter().enumerate() {
        assert!(
            open[i][168 * 320 + 248 - spread] == 16 && open[i][168 * 320 + 248 + spread] == 17,
            "open frame{i} spread{spread} mismatch"
        );
    }
    let close_frames = native_class_confirmation_closing_frames(&background, &cells).unwrap();
    for (i, spread) in [12usize, 8, 4, 0].into_iter().enumerate() {
        if spread == 0 {
            assert_eq!(
                close_frames[i][168 * 320 + 248],
                17,
                "close frame{i} overlap did not preserve second-cell overwrite"
            );
            continue;
        }
        assert!(
            close_frames[i][168 * 320 + 248 - spread] == 16 && close_frames[i][168 * 320 + 248 + spread] == 17,
            "close frame{i} spread{spread} mismatch"
        );
    }
}

/// Go：`TestComposeNativeClassConfirmationExpandsNameAndSelection`
#[test]
fn compose_native_class_confirmation_expands_name_and_selection() {
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let frame = compose_native_class_confirmation_frame(
        &vec![0u8; 320 * 200],
        &native_class_confirm_cells(),
        Some(&strings),
        Some(&font),
        1,
        1,
        1,
    )
    .unwrap();
    assert_eq!(frame[119 * 320 + 12], 205, "dynamic name foreground");
    assert_eq!(frame[168 * 320 + 232], 48, "option0 cell");
    assert_eq!(frame[168 * 320 + 264], 52, "selected option1 pulse cell");
}

/// 對話格 1..19 各為 1×1、色號等於索引。
fn one_pixel_dialogue_cells() -> Vec<fdother::RawCell> {
    let mut dialogue = vec![fdother::RawCell::default(); 20];
    for (index, cell) in dialogue.iter_mut().enumerate().skip(1) {
        *cell = fdother::RawCell { width: 1, height: 1, pixels: vec![index as u8] };
    }
    dialogue
}

/// Go：`TestComposeNativePreparationConfirmationUses31D3CAnchor`
#[test]
fn compose_native_preparation_confirmation_uses_31d3c_anchor() {
    let background = vec![0u8; 320 * 200];
    let dialogue = one_pixel_dialogue_cells();
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let frame = compose_native_preparation_confirmation_frame(
        &background,
        &native_class_confirm_cells(),
        &dialogue,
        &one_pixel_portrait(99),
        Some(&strings),
        Some(&font),
        1,
        1,
    )
    .unwrap();
    assert_eq!(frame[119 * 320 + 95], 205, "question foreground");
    assert_eq!(frame[120 * 320 + 94], 76, "question shadow");
    assert_eq!(frame[0x9017], 99, "DATO#75 default portrait anchor");
    assert_eq!(frame[168 * 320 + 232], 48, "option0 cell");
    assert_eq!(frame[168 * 320 + 264], 52, "selected option1 pulse cell");
    assert_eq!(background[119 * 320 + 95], 0, "input background mutated");
    let dialogue_frame =
        compose_native_preparation_confirmation_dialogue(&background, &dialogue, &one_pixel_portrait(99)).unwrap();
    let question = compose_native_preparation_confirmation_question(
        &background,
        &dialogue,
        &one_pixel_portrait(99),
        Some(&strings),
        Some(&font),
    )
    .unwrap();
    let open = native_preparation_confirmation_opening_frames(
        &background,
        &dialogue_frame,
        &question,
        &native_class_confirm_cells(),
    )
    .unwrap();
    assert_eq!(open.len(), 10, "opening frames");
    let close_frames = native_preparation_confirmation_closing_frames(
        &background,
        &dialogue_frame,
        &question,
        &native_class_confirm_cells(),
    )
    .unwrap();
    assert_eq!(close_frames.len(), 9, "closing frames");
    assert_eq!(
        open[5][112 * 320 + 5],
        dialogue_frame[112 * 320 + 5],
        "sixth dialogue opening frame did not reach stable target"
    );
    assert!(
        open[6][168 * 320 + 244] == 16 && open[9][168 * 320 + 232] == 16,
        "choice opening did not follow the six dialogue frames"
    );
    assert!(
        close_frames[0][168 * 320 + 236] == 16 && close_frames[3][168 * 320 + 248] == 17,
        "choice closing did not precede dialogue closing"
    );
}

/// Go：`TestComposeNativePreparationPromptsUseCallerSpecificFDTXTAnchors`
#[test]
fn compose_native_preparation_prompts_use_caller_specific_fdtxt_anchors() {
    let background = vec![0u8; 320 * 200];
    let dialogue = one_pixel_dialogue_cells();
    let portrait = one_pixel_portrait(99);
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let departure =
        compose_native_preparation_departure_question(&background, &dialogue, &portrait, Some(&strings), Some(&font))
            .unwrap();
    let record =
        compose_native_preparation_record_question(&background, &dialogue, &portrait, Some(&strings), Some(&font))
            .unwrap();
    assert_eq!(departure[119 * 320 + 95], 205, "town departure question did not start at x=95");
    assert!(
        record[119 * 320 + 100] == 205 && record[119 * 320 + 95] != 205,
        "standalone record question did not start at x=100"
    );
}
