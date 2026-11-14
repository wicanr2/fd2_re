//! Go 來源：`remake/internal/campaign/native_shop_purchase_message.go`（228 行）。

use super::*;

/// Go：`native_shop_purchase_message.go:13`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeShopPurchaseMessage(pub u8);

/// Go 原式：`iota`
pub const NATIVE_SHOP_PURCHASE_QUESTION: NativeShopPurchaseMessage = NativeShopPurchaseMessage(0);

pub const NATIVE_SHOP_PURCHASE_INSUFFICIENT_GOLD: NativeShopPurchaseMessage = NativeShopPurchaseMessage(1);

pub const NATIVE_SHOP_PURCHASE_NO_ELIGIBLE_RECIPIENT: NativeShopPurchaseMessage = NativeShopPurchaseMessage(2);

pub const NATIVE_SHOP_PURCHASE_EQUIP_QUESTION: NativeShopPurchaseMessage = NativeShopPurchaseMessage(3);

/// These are the four six-variant word tables copied by 0x2f0c5..0x2f0f4.
/// Even variants contain the native placeholder index 1 because only the odd
/// facility variants enter these shop message branches.
///
/// Go：`native_shop_purchase_message.go:25`
pub(crate) static NATIVE_SHOP_PURCHASE_TEXT: [[i64; 6]; 4] =
    [[1, 502, 1, 439, 1, 439], [1, 504, 1, 438, 1, 438], [1, 505, 1, 437, 1, 437], [1, 507, 1, 507, 1, 507]];

/// This is word_5265f, also consumed by the transfer caller. Purchase selects
/// it only after the chosen recipient's exact eight native slots are full.
///
/// Go：`native_shop_purchase_message.go:34`
pub(crate) static NATIVE_SHOP_RECIPIENT_FULL_TEXT: [i64; 6] = [1, 506, 1, 506, 506, 506];

/// Go：`native_shop_purchase_message.go:36`，`func NativeShopPurchaseTextIndex( message NativeShopPurchaseMessage, hubVariant int, ) (int, bool)`
pub fn native_shop_purchase_text_index(message: NativeShopPurchaseMessage, hub_variant: i64) -> (i64, bool) {
    if message > NATIVE_SHOP_PURCHASE_EQUIP_QUESTION
        || hub_variant < 0
        || hub_variant >= NATIVE_SHOP_PURCHASE_TEXT[0].len() as i64
    {
        return (0, false);
    }
    (NATIVE_SHOP_PURCHASE_TEXT[message.0 as usize][hub_variant as usize], true)
}

/// ComposeNativeShopPurchaseMessage reproduces the stable target after
/// 0x1956b and 0x15f84 for one purchase branch. FFFC expands the selected
/// item name (FDTXT itemID+181) and FFFA expands its decimal price.
///
/// Go：`native_shop_purchase_message.go:50`，`func ComposeNativeShopPurchaseMessage( source []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, portraitID int, strings *fdtxt.Strings, font *fdtxt.Font, message NativeShopPurchaseMessage, hubVariant, itemID, price int, ) ([]byte, error)`
pub fn compose_native_shop_purchase_message(
    source: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    portrait_id: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    message: NativeShopPurchaseMessage,
    hub_variant: i64,
    item_id: i64,
    price: i64,
) -> Result<Vec<u8>> {
    if source.len() as i64 != NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
        || dialogue_cells.len() <= 17
        || strings.is_none()
        || font.is_none()
        || item_id < 0
        || price < 0
    {
        return Err(campaign_err!("campaign: native shop purchase message state is invalid"));
    }
    if message == NATIVE_SHOP_PURCHASE_INSUFFICIENT_GOLD {
        return Err(campaign_err!("campaign: insufficient-gold feedback requires an open purchase confirmation"));
    }
    let (text_index, ok) = native_shop_purchase_text_index(message, hub_variant);
    if !ok {
        return Err(campaign_err!("campaign: native shop purchase message variant is invalid"));
    }
    compose_native_shop_item_price_message(
        source,
        dialogue_cells,
        portrait,
        portrait_id,
        strings,
        font,
        text_index,
        item_id,
        price,
    )
}

/// Go：`native_shop_purchase_message.go:80`，`func composeNativeShopItemPriceMessage( source []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, portraitID int, strings *fdtxt.Strings, font *fdtxt.Font, textIndex, itemID, price int, ) ([]byte, error)`
pub(crate) fn compose_native_shop_item_price_message(
    source: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    portrait_id: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    text_index: i64,
    item_id: i64,
    price: i64,
) -> Result<Vec<u8>> {
    let (strings, font) = match (strings, font) {
        (Some(s), Some(f))
            if source.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
                && dialogue_cells.len() > 17
                && text_index >= 0
                && item_id >= 0
                && price >= 0 =>
        {
            (s, f)
        }
        _ => return Err(campaign_err!("campaign: native shop item/price message state is invalid")),
    };
    let mut frame = compose_native_church_dialogue_overlay_at(
        source,
        dialogue_cells,
        portrait,
        native_facility_portrait_offset_inner(portrait_id),
    )?;
    let words = strings.words(text_index)?;
    let name = strings.words(item_id + 181)?;
    let price_words = go_decimal_words(price);
    let mut expanded = Vec::with_capacity(words.len() + name.len() + price_words.len());
    for &word in &words {
        match word {
            0xfffc => expanded.extend_from_slice(&name),
            0xfffa => expanded.extend_from_slice(&price_words),
            0xfffe => expanded.push(word),
            _ => {
                if word >= fdtxt::CONTROL_MIN {
                    return Err(campaign_err!(
                        "campaign: unsupported shop purchase control {}",
                        battle::go_sharp_x(word as i64)
                    ));
                }
                expanded.push(word);
            }
        }
    }
    compose_native_church_words_at(&mut frame, font, &expanded, NATIVE_SHOP_TEXT_OFFSET)
}

/// ComposeNativeShopPurchaseInsufficientGold reproduces the 0x2f2d3 branch:
/// 0x197e5 has already presented all four choice-closing frames. FDTXT 504/438
/// is appended to that caller-owned final framebuffer at literal VGA 0xac44c,
/// which is framebuffer offset (12,157).
///
/// Go：`native_shop_purchase_message.go:142`，`func ComposeNativeShopPurchaseInsufficientGold( postChoiceClose []byte, strings *fdtxt.Strings, font *fdtxt.Font, hubVariant int, ) ([]byte, error)`
pub fn compose_native_shop_purchase_insufficient_gold(
    post_choice_close: &mut [u8],
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    hub_variant: i64,
) -> Result<Vec<u8>> {
    let (strings, font) = match (strings, font) {
        (Some(s), Some(f)) if post_choice_close.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT => (s, f),
        _ => return Err(campaign_err!("campaign: native shop insufficient-gold state is invalid")),
    };
    let (text_index, ok) = native_shop_purchase_text_index(NATIVE_SHOP_PURCHASE_INSUFFICIENT_GOLD, hub_variant);
    if !ok {
        return Err(campaign_err!("campaign: native shop insufficient-gold variant is invalid"));
    }
    // Go 就地改寫呼叫端的 postChoiceClose 後回傳它；Rust 同樣改寫並回傳複本。
    compose_native_church_text_at(post_choice_close, strings, font, text_index, 157 * NATIVE_SHOP_WIDTH + 12)
}

/// ComposeNativeShopPurchaseRecipientFull reproduces the 0x2f36d branch:
/// rebuild the facility dialogue overlay, expand FFFC with unit[+7]+1, wait in
/// mode one, then close back to the purchase product loop. This compositor
/// returns the stable open-message target; wait/close remain caller-owned.
///
/// Go：`native_shop_purchase_message.go:171`，`func ComposeNativeShopPurchaseRecipientFull( source []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, portraitID int, strings *fdtxt.Strings, font *fdtxt.Font, hubVariant, recipientNameTextIndex int, ) ([]byte, error)`
pub fn compose_native_shop_purchase_recipient_full(
    source: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    portrait_id: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    hub_variant: i64,
    recipient_name_text_index: i64,
) -> Result<Vec<u8>> {
    let (strings, font) = match (strings, font) {
        (Some(s), Some(f))
            if source.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
                && dialogue_cells.len() > 17
                && hub_variant >= 0
                && hub_variant < NATIVE_SHOP_RECIPIENT_FULL_TEXT.len() as i64
                && recipient_name_text_index >= 0 =>
        {
            (s, f)
        }
        _ => return Err(campaign_err!("campaign: native shop recipient-full state is invalid")),
    };
    let mut frame = compose_native_church_dialogue_overlay_at(
        source,
        dialogue_cells,
        portrait,
        native_facility_portrait_offset_inner(portrait_id),
    )?;
    compose_native_church_text_with_name_at(
        &mut frame,
        strings,
        font,
        NATIVE_SHOP_RECIPIENT_FULL_TEXT[hub_variant as usize],
        recipient_name_text_index,
        NATIVE_SHOP_TEXT_OFFSET,
    )
}

/// Go：`native_shop_purchase_message.go:203`，`func ComposeNativeShopPurchaseConfirmation( source []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, portraitID int, strings *fdtxt.Strings, font *fdtxt.Font, choices []fdother.RawCell, message NativeShopPurchaseMessage, hubVariant, itemID, price, selected, pulse int, ) ([]byte, error)`
pub fn compose_native_shop_purchase_confirmation(
    source: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    portrait_id: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    choices: &[fdother::RawCell],
    message: NativeShopPurchaseMessage,
    hub_variant: i64,
    item_id: i64,
    price: i64,
    selected: i64,
    pulse: i64,
) -> Result<Vec<u8>> {
    if message != NATIVE_SHOP_PURCHASE_QUESTION && message != NATIVE_SHOP_PURCHASE_EQUIP_QUESTION {
        return Err(campaign_err!("campaign: native shop confirmation message has no yes/no lifecycle"));
    }
    let question = compose_native_shop_purchase_message(
        source,
        dialogue_cells,
        portrait,
        portrait_id,
        strings,
        font,
        message,
        hub_variant,
        item_id,
        price,
    )?;
    compose_native_confirmation_choices(&question, choices, selected, pulse)
}
