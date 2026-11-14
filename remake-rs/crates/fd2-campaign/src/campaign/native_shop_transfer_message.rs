//! Go 來源：`remake/internal/campaign/native_shop_transfer_message.go`（63 行）。

use super::*;

/// Go：`native_shop_transfer_message.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeShopTransferMessage(pub u8);

/// Go 原式：`iota`
pub const NATIVE_SHOP_TRANSFER_DESTINATION_PROMPT: NativeShopTransferMessage = NativeShopTransferMessage(0);

pub const NATIVE_SHOP_TRANSFER_EMPTY_SOURCE: NativeShopTransferMessage = NativeShopTransferMessage(1);

pub const NATIVE_SHOP_TRANSFER_SOURCE_PROMPT: NativeShopTransferMessage = NativeShopTransferMessage(2);

/// Go：`native_shop_transfer_message.go:19`
pub(crate) static NATIVE_SHOP_TRANSFER_TEXT: [i64; 3] = [510, 511, 512];

/// ComposeNativeShopTransferMessage reproduces the three fixed FDTXT messages
/// owned by 0x2f8ea. Empty-source text expands FFFC with the selected actor;
/// the two prompts contain no dynamic operand.
///
/// Go：`native_shop_transfer_message.go:24`，`func ComposeNativeShopTransferMessage( source []byte, dialogueCells []fdother.RawCell, portrait dato.Frame, portraitID int, strings *fdtxt.Strings, font *fdtxt.Font, message NativeShopTransferMessage, nameTextIndex int, ) ([]byte, error)`
pub fn compose_native_shop_transfer_message(
    source: &[u8],
    dialogue_cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    portrait_id: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    message: NativeShopTransferMessage,
    name_text_index: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_shop_transfer_message.go:ComposeNativeShopTransferMessage")
}
