//! Go 來源：`remake/internal/campaign/native_shop_sell_message.go`（80 行）。

use super::*;

/// Go：`native_shop_sell_message.go:11`
pub(crate) static NATIVE_SHOP_SELL_QUESTION_TEXT: [i64; 6] = [508, 508, 508, 659, 508, 508];

/// Go：`native_shop_sell_message.go:12`
pub(crate) static NATIVE_SHOP_SELL_EMPTY_TEXT: [i64; 6] = [509, 509, 509, 509, 509, 509];

/// Go：`native_shop_sell_message.go:14`，`func ComposeNativeShopSellQuestion( source []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, portraitID int, strings *fdtxt.Strings, font *fdtxt.Font, choices []fdother.RawCell, hubVariant, itemID, salePrice, selected, pulse int, ) ([]byte, error)`
pub fn compose_native_shop_sell_question(
    source: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    portrait_id: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    choices: &[fdother::RawCell],
    hub_variant: i64,
    item_id: i64,
    sale_price: i64,
    selected: i64,
    pulse: i64,
) -> Result<Vec<u8>> {
    let question = compose_native_shop_sell_question_base(
        source,
        dialogue_cells,
        portrait,
        portrait_id,
        strings,
        font,
        hub_variant,
        item_id,
        sale_price,
    )?;
    compose_native_confirmation_choices(&question, choices, selected, pulse)
}

/// Go：`native_shop_sell_message.go:39`，`func ComposeNativeShopSellQuestionBase( source []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, portraitID int, strings *fdtxt.Strings, font *fdtxt.Font, hubVariant, itemID, salePrice int, ) ([]byte, error)`
pub fn compose_native_shop_sell_question_base(
    source: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    portrait_id: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    hub_variant: i64,
    item_id: i64,
    sale_price: i64,
) -> Result<Vec<u8>> {
    if hub_variant < 0 || hub_variant >= NATIVE_SHOP_SELL_QUESTION_TEXT.len() as i64 {
        return Err(campaign_err!("campaign: native shop sell question variant is invalid"));
    }
    compose_native_shop_item_price_message(
        source,
        dialogue_cells,
        portrait,
        portrait_id,
        strings,
        font,
        NATIVE_SHOP_SELL_QUESTION_TEXT[hub_variant as usize],
        item_id,
        sale_price,
    )
}

/// Go：`native_shop_sell_message.go:57`，`func ComposeNativeShopSellEmpty( source []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, portraitID int, strings *fdtxt.Strings, font *fdtxt.Font, hubVariant, nameTextIndex int, ) ([]byte, error)`
pub fn compose_native_shop_sell_empty(
    source: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    portrait_id: i64,
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    hub_variant: i64,
    name_text_index: i64,
) -> Result<Vec<u8>> {
    if hub_variant < 0 || hub_variant >= NATIVE_SHOP_SELL_EMPTY_TEXT.len() as i64 {
        return Err(campaign_err!("campaign: native shop sell-empty variant is invalid"));
    }
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
        NATIVE_SHOP_SELL_EMPTY_TEXT[hub_variant as usize],
        name_text_index,
        NATIVE_SHOP_TEXT_OFFSET,
    )
}
