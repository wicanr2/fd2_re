//! Go `image.Point`、`image.Rectangle`、`image.RGBA` 的同形值型別。沒有對應的 Go 檔。
//!
//! `campaign` 與 `ending` 的公開結構與回傳值直接使用這幾個 Go 標準函式庫型別（座標、視窗、
//! 對話框範圍與合成後的 RGBA 影像）。這裡只定義同樣的欄位，讓骨架簽名可以照 Go 寫；不移植
//! Go `image` 套件。階段 2 需要的方法（例如 `Dx`／`Dy`／`Empty`）由用到的組依 Go 定義補上，並在
//! PORTING.md 記錄。

/// Go `image.Point`：`X`、`Y` 為 Go `int`。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Point {
    pub x: i64,
    pub y: i64,
}

/// Go `image.Rectangle`：`Min` 含、`Max` 不含（Go 的半開區間慣例）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Rectangle {
    pub min: Point,
    pub max: Point,
}

/// Go `image.RGBA`：`Pix` 依列存放 R、G、B、A 各一位元組，`Stride` 是相鄰兩列的位元組距離，
/// `Rect` 是影像範圍。只有 `ending` 的合成器輸出用到（`IndexedCompositor.RGBA`）。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Rgba {
    pub pix: Vec<u8>,
    pub stride: i64,
    pub rect: Rectangle,
}
