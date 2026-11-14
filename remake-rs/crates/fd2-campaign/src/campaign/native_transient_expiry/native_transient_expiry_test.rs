//! 移植自 `remake/internal/campaign/native_transient_expiry_test.go`。

use super::*;
use crate::campaign::native_church_scene::native_church_scene_test::load_native_church_original_scene_assets;

/// Go：`TestNativeTransientExpiryTextContract`
#[test]
#[allow(clippy::assertions_on_constants)] // Go 在執行期比對常數
fn native_transient_expiry_text_contract() {
    assert!(
        NATIVE_TRANSIENT_EXPIRY_TEXT_BASE == 481
            && NATIVE_TRANSIENT_EXPIRY_TEXT_COUNT == 6
            && NATIVE_TRANSIENT_EXPIRY_TEXT_OFFSET == 0x9f23,
        "base={} count={} offset={:#x}",
        NATIVE_TRANSIENT_EXPIRY_TEXT_BASE,
        NATIVE_TRANSIENT_EXPIRY_TEXT_COUNT,
        NATIVE_TRANSIENT_EXPIRY_TEXT_OFFSET
    );
}

/// Go：`TestNativeTransientExpiryRejectsUnknownCounterBeforeAssets`
#[test]
fn native_transient_expiry_rejects_unknown_counter_before_assets() {
    let r = compose_native_transient_expiry_frame(&[], &[], &dato::Frame::default(), None, None, 6);
    assert!(r.is_err(), "unknown transient counter was accepted");
}

/// Go：`TestNativeDeathRewardMessageTextContract`
#[test]
#[allow(clippy::assertions_on_constants)] // Go 在執行期比對常數
fn native_death_reward_message_text_contract() {
    assert!(
        NATIVE_DEATH_REWARD_ITEM_TEXT_INDEX == 0x1b0
            && NATIVE_DEATH_REWARD_GOLD_TEXT_INDEX == 0x1b3
            && NATIVE_DEATH_REWARD_ITEM_NAME_BASE == 0xb5
            && NATIVE_DEATH_REWARD_MESSAGE_TEXT_OFFS == 0x9f23,
        "item={:#x} gold={:#x} name={:#x} offset={:#x}",
        NATIVE_DEATH_REWARD_ITEM_TEXT_INDEX,
        NATIVE_DEATH_REWARD_GOLD_TEXT_INDEX,
        NATIVE_DEATH_REWARD_ITEM_NAME_BASE,
        NATIVE_DEATH_REWARD_MESSAGE_TEXT_OFFS
    );
    assert!(
        compose_native_death_reward_message_frame(&vec![0u8; 320 * 200], None, None, 2, 1).is_err(),
        "kind 2 has no 0x1AA1D message and must be rejected"
    );
}

/// Go：`TestComposeNativeDeathRewardMessageExpandsItemNameAndGold`
///
/// 0x1B0 的 FFFC 展開成 item+0xB5 的物品名、0x1B3 的 FFFA 展開成金額；兩者都寫在 0x9F23。
#[test]
fn compose_native_death_reward_message_expands_item_name_and_gold() {
    let Some(a) = load_native_church_original_scene_assets() else { return };
    let background = vec![0u8; 320 * 200];
    let dialogue = compose_native_church_dialogue_overlay_at(
        &background,
        &a.dialogue_cells,
        &a.portrait,
        NATIVE_LOWER_PORTRAIT_RIGHT_EDGE,
    )
    .unwrap();
    let item = compose_native_death_reward_message_frame(&dialogue, Some(&a.strings), Some(&a.font), 0, 0x00).unwrap();
    let name = a.strings.words(0xb5).unwrap();
    // 對照：手動展開同一句再用同一個 composer 畫。
    let words = a.strings.words(0x1b0).unwrap_or_default();
    let mut expanded = Vec::new();
    for w in words {
        if w == 0xfffc {
            expanded.extend_from_slice(&name);
        } else {
            expanded.push(w);
        }
    }
    let want = compose_native_church_words_at(&mut dialogue.clone(), &a.font, &expanded, 0x9f23).unwrap();
    assert!(item == want, "item message frame differs from the FFFC-expanded 0x1B0 composition");
    assert!(item != dialogue, "item message wrote nothing");
    let gold = compose_native_death_reward_message_frame(&dialogue, Some(&a.strings), Some(&a.font), 1, 150).unwrap();
    assert!(
        gold != dialogue && gold != item,
        "gold message must differ from both the bare dialogue and the item message"
    );
}
