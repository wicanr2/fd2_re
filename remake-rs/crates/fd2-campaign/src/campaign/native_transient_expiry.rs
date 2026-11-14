//! Go 來源：`remake/internal/campaign/native_transient_expiry.go`（105 行）。

use super::*;

pub const NATIVE_TRANSIENT_EXPIRY_TEXT_BASE: i64 = 0x1e1;

pub const NATIVE_TRANSIENT_EXPIRY_TEXT_COUNT: i64 = 6;

pub const NATIVE_TRANSIENT_EXPIRY_TEXT_OFFSET: i64 = 0x9f23;

/// ComposeNativeTransientExpiryFrame consumes the exact sub_1A866 expiry
/// inputs without assigning gameplay names to raw +0x22..+0x27. The original
/// DATO +7 selector is resolved by the caller and supplied as a decoded frame.
///
/// Go：`native_transient_expiry.go:22`，`func ComposeNativeTransientExpiryFrame( background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font, counterIndex int, ) ([]byte, error)`
pub fn compose_native_transient_expiry_frame(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    counter_index: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_transient_expiry.go:ComposeNativeTransientExpiryFrame")
}

/// ComposeNativeTransientExpiryDialogue retains the original lower-left DATO
/// placement while allowing the UI adapter to replace only translated text.
///
/// Go：`native_transient_expiry.go:46`，`func ComposeNativeTransientExpiryDialogue(background []byte, dialogueCells []fdother.RawCell, portrait dato.Frame) ([]byte, error)`
pub fn compose_native_transient_expiry_dialogue(
    background: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
) -> Result<Vec<u8>> {
    compose_native_church_dialogue_overlay_at(background, dialogue_cells, portrait, NATIVE_LOWER_PORTRAIT_RIGHT_EDGE)
}

/// 0x1AA1D 物品型態 0／1 的成功訊息：FDTXT_000 0x1B0「從敵人身上，得到 FFFC！」
/// （FFFC = [0x53AD9] = item+0xB5 的物品名）與 0x1B3「從敵人身上，得到 FFFA 元！」，
/// 都以 0x15F84(buffer, index, 0xA9F23, 320, 205, 76, 74, 19, 1) 寫在對話格
/// 文字座標 (99,127)，和 sub_1A866 到期訊息同一個 offset。
pub const NATIVE_DEATH_REWARD_ITEM_TEXT_INDEX: i64 = 0x1b0;

pub const NATIVE_DEATH_REWARD_GOLD_TEXT_INDEX: i64 = 0x1b3;

pub const NATIVE_DEATH_REWARD_ITEM_NAME_BASE: i64 = 0xb5;

/// Go 原式：`NativeTransientExpiryTextOffset`
pub const NATIVE_DEATH_REWARD_MESSAGE_TEXT_OFFS: i64 = 40739;

/// ComposeNativeDeathRewardMessageFrame 在已由 0x1956B(killer+7) 開好的對話格上寫
/// 0x1AA1D 成功訊息。kind 0 是物品（value 為物品 byte），kind 1 是金錢（value 為金額）。
///
/// Go：`native_transient_expiry.go:63`，`func ComposeNativeDeathRewardMessageFrame( dialogue []byte, strings *fdtxt.Strings, font *fdtxt.Font, kind, value int, ) ([]byte, error)`
pub fn compose_native_death_reward_message_frame(
    dialogue: &[u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    kind: i64,
    value: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_transient_expiry.go:ComposeNativeDeathRewardMessageFrame")
}
