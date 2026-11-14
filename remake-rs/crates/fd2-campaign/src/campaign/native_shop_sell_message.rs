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
    todo!("campaign/native_shop_sell_message.go:ComposeNativeShopSellQuestion")
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
    todo!("campaign/native_shop_sell_message.go:ComposeNativeShopSellQuestionBase")
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
    todo!("campaign/native_shop_sell_message.go:ComposeNativeShopSellEmpty")
}
