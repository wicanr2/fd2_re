//! `fd2-game` 的支援型別：取代 Go 版 `cmd/fd2` 依賴的 Ebiten、牆鐘、音訊與 `error`。沒有對應的 Go 檔。
//!
//! 對應規則見 `docs/goal/120-goal-rust-r4-game-state-split-20261114.md`「階段 1 骨架的對應規則」：
//!
//! | Go | 本檔 |
//! |---|---|
//! | `*ebiten.Image`、`*image.NRGBA`、`image.Image` | [`Picture`]（RGBA 像素），欄位是 `Option<Arc<Picture>>` |
//! | `*image.Paletted` | [`IndexedFrame`]（索引色像素與調色盤） |
//! | `color.RGBA`、`color.NRGBA`、`color.Palette` | [`ColorRgba`]、[`ColorNrgba`]、[`ColorPalette`] |
//! | `inpututil`、`ebiten.IsKeyPressed` | 每個邏輯幀一份 [`TickInput`]；[`Key`] 只列 Go 用到的鍵 |
//! | `time.Time`、`gameNow()` | [`GameInstant`]（虛擬時鐘的奈秒） |
//! | `time.Duration` | [`GoDuration`]（`i64` 奈秒） |
//! | `*audio.Player`、`*audio.Context` | [`AudioVoice`]（播放請求與狀態）、[`AudioContext`] |
//! | `font.Face`、`*sfnt.Font` | [`FontFace`]、[`SfntFont`]（TTF 繪字由 `fd2-app` 負責） |
//! | `any`（只用來比對擁有者是否相同） | [`OwnerToken`] |
//! | `error` | [`GameError`]，`(T, error)` 是 [`Result<T>`] |
//!
//! 這些型別只定義資料與 Go 呼叫端需要的最少操作；轉成 GPU 貼圖、真正播放聲音、讀實體鍵盤
//! 都是 `fd2-app`（R5）的事。

use std::borrow::Cow;
use std::collections::BTreeSet;
use std::fmt;
use std::sync::Arc;

use fd2_campaign::campaign::CampaignError;
use fd2_rules::battle::BattleError;

// ---------------- 畫面 ----------------

/// RGBA 畫面（每像素 R、G、B、A 各一位元組，不預乘 alpha，依列存放，列距 `width * 4`）。
///
/// 取代 Go 狀態裡的 `*ebiten.Image`、`*image.NRGBA` 與 `image.Image`。Go 以指標共用同一張圖，
/// Rust 的欄位是 `Option<Arc<Picture>>`；要改寫內容的函式收 `&mut Picture`。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Picture {
    pub width: i64,
    pub height: i64,
    pub pix: Vec<u8>,
}

/// 索引色畫面：每像素一個調色盤索引，調色盤另存。取代 Go 的 `*image.Paletted`，也用來保存
/// 演出工作的 VGA 畫格（320×200）。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct IndexedFrame {
    pub width: i64,
    pub height: i64,
    pub pix: Vec<u8>,
    pub palette: ColorPalette,
}

/// Go `color.RGBA`（預乘 alpha）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct ColorRgba {
    pub r: u8,
    pub g: u8,
    pub b: u8,
    pub a: u8,
}

/// Go `color.NRGBA`（不預乘 alpha）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct ColorNrgba {
    pub r: u8,
    pub g: u8,
    pub b: u8,
    pub a: u8,
}

/// Go `color.Palette`。Go 的元素是 `color.Color` 介面；`cmd/fd2` 放進去的都是不透明的 RGB，
/// 所以 Rust 直接存 [`ColorRgba`]。
pub type ColorPalette = Vec<ColorRgba>;

// ---------------- 輸入 ----------------

/// Go 版用到的 Ebiten 鍵（由產生器從 `cmd/fd2` 正式程式的 `ebiten.Key*` 用法列出）。實體鍵與
/// 觸控對應由 `fd2-app` 決定；`fd2-game` 只看邏輯鍵。
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Default)]
pub enum Key {
{{KEYS}}
}

/// 一個邏輯幀的輸入：本幀按下、放開與按住的鍵。取代 Go 的 `inpututil.IsKeyJustPressed`、
/// `inpututil.IsKeyJustReleased` 與 `ebiten.IsKeyPressed`。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TickInput {
    pub pressed: BTreeSet<Key>,
    pub released: BTreeSet<Key>,
    pub held: BTreeSet<Key>,
}

impl TickInput {
    /// Go `inpututil.IsKeyJustPressed(k)`。
    pub fn is_just_pressed(&self, k: Key) -> bool {
        self.pressed.contains(&k)
    }

    /// Go `inpututil.IsKeyJustReleased(k)`。
    pub fn is_just_released(&self, k: Key) -> bool {
        self.released.contains(&k)
    }

    /// Go `ebiten.IsKeyPressed(k)`。
    pub fn is_pressed(&self, k: Key) -> bool {
        self.held.contains(&k)
    }
}

// ---------------- 時間 ----------------

/// 虛擬時鐘上的時間點（奈秒）。取代 Go 的 `time.Time` 與 `gameNow()`：每個邏輯幀前進固定步長
/// 再乘遊戲速度，所以給定輸入序列就有固定結果。零值對應 Go 的 `time.Time{}`（`IsZero` 為真）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct GameInstant(pub i64);

impl GameInstant {
    /// Go `t.IsZero()`。
    pub fn is_zero(self) -> bool {
        self.0 == 0
    }
}

impl std::ops::Sub for GameInstant {
    type Output = GoDuration;

    /// Go `t.Sub(u)`（虛擬時鐘不會接近 `i64` 上限，不需要 Go 的飽和處理）。
    fn sub(self, earlier: GameInstant) -> GoDuration {
        GoDuration(self.0.wrapping_sub(earlier.0))
    }
}

impl std::ops::Add<GoDuration> for GameInstant {
    type Output = GameInstant;

    /// Go `t.Add(d)`。
    fn add(self, d: GoDuration) -> GameInstant {
        GameInstant(self.0.wrapping_add(d.0))
    }
}

/// Go `time.Duration`（`i64` 奈秒）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct GoDuration(pub i64);

impl GoDuration {
    pub const NANOSECOND: GoDuration = GoDuration(1);
    pub const MICROSECOND: GoDuration = GoDuration(1_000);
    pub const MILLISECOND: GoDuration = GoDuration(1_000_000);
    pub const SECOND: GoDuration = GoDuration(1_000_000_000);

    /// Go `d.Seconds()`。
    pub fn seconds(self) -> f64 {
        let sec = self.0 / 1_000_000_000;
        let nsec = self.0 % 1_000_000_000;
        sec as f64 + nsec as f64 / 1e9
    }

    /// Go `d.Milliseconds()`。
    pub fn milliseconds(self) -> i64 {
        self.0 / 1_000_000
    }
}

// ---------------- 音訊 ----------------

/// 一個音效或音樂播放器的請求與狀態。取代 Go 的 `*audio.Player`（與只由它實作的 `sfxVoice`
/// 介面）：`fd2-game` 只記錄要播什麼、是否仍在播放，實際播放與回報播放結束由 `fd2-app` 做。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct AudioVoice {
    /// 播放來源（素材路徑或音樂曲目鍵），與 Go 建立播放器時讀的檔案相同。
    pub source: String,
    /// 外殼回報的播放狀態（Go `IsPlaying()`）。
    pub playing: bool,
    /// Go 已呼叫 `Close()`。
    pub closed: bool,
}

impl AudioVoice {
    /// Go `IsPlaying()`。
    pub fn is_playing(&self) -> bool {
        self.playing && !self.closed
    }
}

/// Go `*audio.Context`：只保存取樣率。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct AudioContext {
    pub sample_rate: i64,
}

// ---------------- 字型 ----------------

/// Go `*sfnt.Font`：TTF／OTF 原始位元組。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SfntFont {
    pub data: Vec<u8>,
}

/// Go `font.Face`：字型與大小。繪字（`text.Draw`）是 `fd2-app` 的事。
#[derive(Debug, Clone, Default, PartialEq)]
pub struct FontFace {
    pub font: Option<Arc<SfntFont>>,
    pub size: f64,
    pub dpi: f64,
}

// ---------------- 其他 ----------------

/// 取代只用來比對「是不是同一個擁有者」的 Go `any` 欄位（Go 以介面值的指標相等判斷）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct OwnerToken(pub u64);

// ---------------- 錯誤 ----------------

/// Go `error` 的 Rust 對應，形狀同 `fd2_rules::battle::BattleError`：訊息即 Go `err.Error()` 的完整
/// 文字，`%w` 包裝以 [`GameError::wrap`] 串接前綴。
///
/// `cmd/fd2` 有兩個以 `errors.Is` 判斷的哨兵錯誤（`errAttackPresentationYield`、
/// `errNativeHotelPreparationSlot`），所以另外記住錯誤源自哪個哨兵：`wrap` 保留來源，
/// [`GameError::is`] 對應 Go `errors.Is`。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct GameError {
    message: Cow<'static, str>,
    sentinel: Option<&'static str>,
}

impl GameError {
    /// 以完整訊息建立錯誤，對應 Go `errors.New(msg)` 或已格式化的 `fmt.Errorf`。
    pub fn new(message: impl Into<String>) -> Self {
        Self { message: Cow::Owned(message.into()), sentinel: None }
    }

    /// 套件層的哨兵錯誤（Go `var errX = errors.New(msg)`）。
    pub const fn sentinel(message: &'static str) -> Self {
        Self { message: Cow::Borrowed(message), sentinel: Some(message) }
    }

    /// 對應 Go `fmt.Errorf("<prefix>%w", err)`：回傳 `prefix` 接上原訊息的新錯誤，保留哨兵來源。
    pub fn wrap(self, prefix: impl fmt::Display) -> Self {
        Self { message: Cow::Owned(format!("{prefix}{}", self.message)), sentinel: self.sentinel }
    }

    /// Go `err.Error()`。
    pub fn message(&self) -> &str {
        &self.message
    }

    /// Go `errors.Is(err, target)`，`target` 是哨兵錯誤。
    pub fn is(&self, target: &GameError) -> bool {
        target.sentinel.is_some() && self.sentinel == target.sentinel
    }
}

impl fmt::Display for GameError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.message)
    }
}

impl std::error::Error for GameError {}

/// `fd2-game` 的結果型別；`(T, error)` 對應 `Result<T>`。
pub type Result<T, E = GameError> = std::result::Result<T, E>;

/// 以 `format!` 語法建立 [`GameError`]，對應不含 `%w` 的 `fmt.Errorf`。
///
/// Go 的 `%d`、`%v`、`%s` 對整數與字串和 Rust `{}` 相同；`%q`、`%x`、`%#x`、`%02x` 與結構的
/// `%v` 要逐處對照 Go 的輸出格式（`fd2_rules::battle::go_quote`、`go_sharp_x`）。
#[macro_export]
macro_rules! game_err {
    ($($arg:tt)*) => {
        $crate::game::GameError::new(format!($($arg)*))
    };
}

impl From<BattleError> for GameError {
    /// 規則層錯誤原樣轉成訊息；對應 Go 直接回傳 `battle` 的 `error`。
    fn from(err: BattleError) -> Self {
        Self::new(err.message())
    }
}

impl From<CampaignError> for GameError {
    /// 戰役層錯誤原樣轉成訊息；對應 Go 直接回傳 `campaign`／`ending` 的 `error`。
    fn from(err: CampaignError) -> Self {
        Self::new(err.message())
    }
}

macro_rules! from_format_error {
    ($($t:ty),* $(,)?) => {
        $(
            impl From<$t> for GameError {
                /// 格式層錯誤原樣轉成訊息；對應 Go 直接回傳下層 `error`。
                fn from(err: $t) -> Self {
                    Self::new(err.to_string())
                }
            }
        )*
    };
}

from_format_error!(
    fd2_formats::afm::AfmError,
    fd2_formats::archive::ArchiveError,
    fd2_formats::dato::DatoError,
    fd2_formats::fdicon::FdiconError,
    fd2_formats::fdother::FdotherError,
    fd2_formats::fdother::NativeFlowError,
    fd2_formats::fdtxt::FdtxtError,
    fd2_formats::figani::FiganiError,
    fd2_formats::save::SaveError,
);
