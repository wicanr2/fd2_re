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

/// RGBA 畫面（每像素 R、G、B、A 各一位元組，**預乘 alpha**，依列存放，列距 `width * 4`）。
///
/// 取代 Go 狀態裡的 `*ebiten.Image`、`*image.NRGBA` 與 `image.Image`。Go 以指標共用同一張圖，
/// Rust 的欄位是 `Option<Arc<Picture>>`；要改寫內容的函式收 `&mut Picture`。
///
/// 像素和 Ebiten `ReadPixels` 的輸出相同（R5 階段 1 定案，R4 時寫成不預乘）：`*image.NRGBA` 在 Go 都是
/// 建好就交給 `ebiten.NewImageFromImage`，Rust 直接存那一步之後的預乘值。繪圖操作（Ebiten 語意子集）在
/// `canvas.rs`。
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
    #[default]
    /// Go `ebiten.KeyA`。
    A,
    /// Go `ebiten.KeyAlt`。
    Alt,
    /// Go `ebiten.KeyAltLeft`。
    AltLeft,
    /// Go `ebiten.KeyAltRight`。
    AltRight,
    /// Go `ebiten.KeyArrowDown`。
    ArrowDown,
    /// Go `ebiten.KeyArrowLeft`。
    ArrowLeft,
    /// Go `ebiten.KeyArrowRight`。
    ArrowRight,
    /// Go `ebiten.KeyArrowUp`。
    ArrowUp,
    /// Go `ebiten.KeyBackspace`。
    Backspace,
    /// Go `ebiten.KeyControl`。
    Control,
    /// Go `ebiten.KeyD`。
    D,
    /// Go `ebiten.KeyEnter`。
    Enter,
    /// Go `ebiten.KeyEscape`。
    Escape,
    /// Go `ebiten.KeyF1`。
    F1,
    /// Go `ebiten.KeyF10`。
    F10,
    /// Go `ebiten.KeyF11`。
    F11,
    /// Go `ebiten.KeyF12`。
    F12,
    /// Go `ebiten.KeyF2`。
    F2,
    /// Go `ebiten.KeyF3`。
    F3,
    /// Go `ebiten.KeyF4`。
    F4,
    /// Go `ebiten.KeyF5`。
    F5,
    /// Go `ebiten.KeyF6`。
    F6,
    /// Go `ebiten.KeyF7`。
    F7,
    /// Go `ebiten.KeyF8`。
    F8,
    /// Go `ebiten.KeyF9`。
    F9,
    /// Go `ebiten.KeyHome`。
    Home,
    /// Go `ebiten.KeyNumpad5`。
    Numpad5,
    /// Go `ebiten.KeyS`。
    S,
    /// Go `ebiten.KeyShift`。
    Shift,
    /// Go `ebiten.KeySpace`。
    Space,
    /// Go `ebiten.KeyTab`。
    Tab,
    /// Go `ebiten.KeyW`。
    W,
    /// Go `ebiten.KeyZ`。
    Z,
}

/// 一個邏輯幀的輸入：本幀按下、放開與按住的鍵。取代 Go 的 `inpututil.IsKeyJustPressed`、
/// `inpututil.IsKeyJustReleased` 與 `ebiten.IsKeyPressed`。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TickInput {
    pub pressed: BTreeSet<Key>,
    pub released: BTreeSet<Key>,
    pub held: BTreeSet<Key>,
    /// R4 A 組新增（共用變更需求）：Go `inpututil.KeyPressDuration(k)`，按住中的鍵已按住的邏輯幀數
    /// （按下那一幀是 1）；沒列出的鍵是 0。`keyRepeat` 的連發節奏依這個值。
    pub press_duration: std::collections::BTreeMap<Key, i64>,
    /// R5 階段 4（r5w）：本幀剛按下的鍵裡有 [`Key`] 沒有列出的 Ebiten 鍵（例如字母 Q）。Go
    /// `inpututil.AppendJustPressedKeys(nil)` 會把它們算進「任意鍵」。
    pub other_just_pressed: bool,
    /// R5 階段 4（r5w）：Go `inpututil.AppendJustPressedTouchIDs(nil)` 每個觸控點的 `ebiten.TouchPosition`
    /// （Ebiten 邏輯座標，依觸控 ID 遞增）。座標可能落在 640×400 之外，由 `player_touch_coordinates` 篩選。
    pub just_pressed_touches: Vec<(i64, i64)>,
    /// R5 階段 4（r5w）：Go `ebiten.AppendTouchIDs(nil)` 與 `TouchPosition`：按住中的全部觸控點 `(id, x, y)`，依 ID 遞增。
    /// 只給外框 UI（工具列、海報滑桿，R6）用；遊戲內只讀 [`TickInput::just_pressed_touches`]。
    pub touches: Vec<(u64, i64, i64)>,
    /// R5 階段 4（r5w）：Go `ebiten.CursorPosition()`（Ebiten 邏輯座標）。Go 只在外框 UI（工具列、海報、速度選單、
    /// 數值編輯器，R6）讀滑鼠；遊戲內的滑鼠語意等 Go 版定案後再移植（使用者 2026-11-16）。
    pub cursor: (i64, i64),
    /// R5 階段 4（r5w）：Go `ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)`。
    pub mouse_left_held: bool,
    /// R5 階段 4（r5w）：Go `inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)`。
    pub mouse_left_just_pressed: bool,
    /// R5 階段 4（r5w）：Go `inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)`。
    pub mouse_left_just_released: bool,
    /// R5 階段 4（r5w）：Go `ebiten.AppendInputChars(nil)`（數值編輯器的文字輸入，R6）。
    pub input_chars: Vec<char>,
    /// R5 階段 4（r5i）：[`Key`] 沒有列出、按住中的鍵的按住時間，鍵是 [`RawInputState::other_keys`] 的編號（外殼以
    /// Ebiten 的 `ebiten.Key` 數值當編號，見 `go_ebiten_keys.rs`）。按鍵設定面板與自訂綁定（例如把確認改成 Q）讀它。
    pub other_press_duration: std::collections::BTreeMap<u32, i64>,
    /// R5 階段 4（r5i）：輸入對應層（`player_input.rs`）打開時，本幀的實體輸入；這時本結構的鍵是邏輯鍵（Go
    /// `keyJustPressed` 等）。`None` 表示沒有對應層（測試、重播，Go `activeInput == nil`），鍵就是實體鍵。
    pub physical: Option<Box<TickInput>>,
}

impl TickInput {
    /// R5 階段 4（r5i）：實體輸入（Go 直接查 `inpututil`／`ebiten.IsKeyPressed` 的地方：Alt+Enter、修飾鍵秘密商店、
    /// Ctrl+F12）。沒有輸入對應層時就是自己。
    pub fn physical(&self) -> &TickInput {
        self.physical.as_deref().unwrap_or(self)
    }

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

    /// Go `inpututil.KeyPressDuration(k)`（R4 A 組新增）。
    pub fn key_press_duration(&self, k: Key) -> i64 {
        self.press_duration.get(&k).copied().unwrap_or(0)
    }

    /// Go `len(inpututil.AppendJustPressedKeys(nil)) > 0`（R4 A 組新增）：本幀有任何鍵按下。
    /// R5 階段 4 起含 [`TickInput::other_just_pressed`]。
    pub fn any_just_pressed(&self) -> bool {
        !self.pressed.is_empty() || self.other_just_pressed
    }
}

impl Key {
    /// R5 階段 4（r5w）：全部 33 個鍵，依宣告順序。
    pub const ALL: [Key; 33] = [
        Key::A,
        Key::Alt,
        Key::AltLeft,
        Key::AltRight,
        Key::ArrowDown,
        Key::ArrowLeft,
        Key::ArrowRight,
        Key::ArrowUp,
        Key::Backspace,
        Key::Control,
        Key::D,
        Key::Enter,
        Key::Escape,
        Key::F1,
        Key::F10,
        Key::F11,
        Key::F12,
        Key::F2,
        Key::F3,
        Key::F4,
        Key::F5,
        Key::F6,
        Key::F7,
        Key::F8,
        Key::F9,
        Key::Home,
        Key::Numpad5,
        Key::S,
        Key::Shift,
        Key::Space,
        Key::Tab,
        Key::W,
        Key::Z,
    ];
}

/// R5 階段 4（r5w）：一個邏輯幀開始時、已換成邏輯按鍵的輸入狀態，交給 [`EbitenInputState::update`]。
///
/// 這是 `fd2-game` 與外殼之間的輸入介面：外殼先把實體輸入（鍵盤、滑鼠、觸控，之後的手把與自訂綁定）對應成
/// 下列邏輯狀態，`fd2-game` 只看這一層，不知道實體裝置。
///
/// - `keys`：按住中的邏輯鍵。`Control`／`Shift` 代表左右任一個（[`Key`] 沒有分左右）；`AltLeft`／`AltRight` 分開給，
///   虛擬鍵 `Alt` 由 [`EbitenInputState`] 照 Ebiten `isKeyPressed` 算出。
/// - `other_keys`：按住中、[`Key`] 沒有列出的鍵，以外殼自訂的編號區分（只用來算「剛按下」）。
/// - `touches`：按住中的觸控點 `(id, x, y)`，座標已換成 Ebiten 邏輯座標（`int(...)` 向零截斷）。
/// - `cursor`、`mouse_left`：滑鼠游標的 Ebiten 邏輯座標與左鍵是否按住。
/// - `input_chars`：本幀輸入的文字（Go `ebiten.AppendInputChars`）。
///
/// R5 階段 4（r5i）：`other_keys` 的編號由外殼定為 Ebiten 的 `ebiten.Key` 數值（Ebiten 沒有的鍵用大於 `KeyMax` 的編號）；
/// `devices` 是滑鼠右鍵、滾輪、焦點與手把（`player_input.rs` 的 [`RawDevices`]）。
#[derive(Debug, Clone, Default, PartialEq)]
pub struct RawInputState {
    pub keys: BTreeSet<Key>,
    pub other_keys: BTreeSet<u32>,
    pub touches: Vec<(u64, i64, i64)>,
    pub cursor: (i64, i64),
    pub mouse_left: bool,
    pub input_chars: Vec<char>,
    pub devices: super::player_input::RawDevices,
}

/// R5 階段 4（r5w）：Ebiten v2.6.6 `inpututil` 的每幀狀態（`inpututil/inpututil.go` 的 `inputState.update`，
/// 由 `hooks.AppendHookOnBeforeUpdate` 在每次 `Update` 之前執行一次）。
///
/// - 每個邏輯幀呼叫一次 [`EbitenInputState::update`]：按住的鍵持續時間加 1，放開的歸 0；上一幀的持續時間另存。
/// - `IsKeyJustPressed` 是持續時間等於 1；`IsKeyJustReleased` 是上一幀大於 0 而本幀等於 0；`IsKeyPressed` 是本幀按住；
///   `KeyPressDuration` 是本幀的持續時間。
/// - Ebiten 的 `KeyMax` 是 `KeyMeta`，迴圈涵蓋虛擬鍵 `KeyAlt`／`KeyControl`／`KeyShift`；它們的按住狀態是左右任一個
///   （`input.go:409` `isKeyPressed`）。
/// - 觸控：`AppendJustPressedTouchIDs` 是持續時間等於 1 的 ID，依 ID 遞增排序；位置是同一幀的 `TouchPosition`。
/// - 同一個顯示幀跑多次 `Update` 時，Ebiten 每次都讀同一份實體狀態（`internal/ui/context.go:136`），外殼照樣傳入
///   同一份 [`RawInputState`]。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct EbitenInputState {
    durations: std::collections::BTreeMap<Key, i64>,
    other_durations: std::collections::BTreeMap<u32, i64>,
    touch_durations: std::collections::BTreeMap<u64, i64>,
    mouse_left_duration: i64,
}

impl EbitenInputState {
    /// 推進一個邏輯幀，回傳本幀的 [`TickInput`]。
    pub fn update(&mut self, raw: &RawInputState) -> TickInput {
        let mut held = raw.keys.clone();
        held.remove(&Key::Alt);
        if raw.keys.contains(&Key::AltLeft) || raw.keys.contains(&Key::AltRight) {
            held.insert(Key::Alt);
        }
        let mut input = TickInput::default();
        for k in Key::ALL {
            let prev = self.durations.get(&k).copied().unwrap_or(0);
            let now = if held.contains(&k) { prev + 1 } else { 0 };
            if now > 0 {
                self.durations.insert(k, now);
                input.held.insert(k);
                input.press_duration.insert(k, now);
                if now == 1 {
                    input.pressed.insert(k);
                }
            } else {
                self.durations.remove(&k);
                if prev > 0 {
                    input.released.insert(k);
                }
            }
        }
        let mut others = std::collections::BTreeMap::new();
        for &code in &raw.other_keys {
            let now = self.other_durations.get(&code).copied().unwrap_or(0) + 1;
            input.other_just_pressed |= now == 1;
            others.insert(code, now);
        }
        input.other_press_duration = others.clone();
        self.other_durations = others;
        let mut touches = std::collections::BTreeMap::new();
        for &(id, x, y) in &raw.touches {
            let now = self.touch_durations.get(&id).copied().unwrap_or(0) + 1;
            touches.insert(id, (now, x, y));
        }
        self.touch_durations = touches.iter().map(|(&id, &(d, _, _))| (id, d)).collect();
        input.just_pressed_touches = touches.values().filter(|(d, _, _)| *d == 1).map(|&(_, x, y)| (x, y)).collect();
        input.touches = touches.iter().map(|(&id, &(_, x, y))| (id, x, y)).collect();
        // 滑鼠左鍵（`inpututil.go` 的 `mouseButtonDurations`）。
        let prev = self.mouse_left_duration;
        self.mouse_left_duration = if raw.mouse_left { prev + 1 } else { 0 };
        input.mouse_left_held = raw.mouse_left;
        input.mouse_left_just_pressed = self.mouse_left_duration == 1;
        input.mouse_left_just_released = prev > 0 && self.mouse_left_duration == 0;
        input.cursor = raw.cursor;
        input.input_chars = raw.input_chars.clone();
        input
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
    /// R4 A 組新增（共用變更需求）：要播放的編碼後音訊位元組（分離素材包的 OGG 子資源、WAV 檔或
    /// BGM 檔內容）。Go 把解碼後的 PCM 交給 `audio.NewPlayerFromBytes`；Rust 由外殼解碼。
    pub encoded: Vec<u8>,
    /// R4 A 組新增（共用變更需求）：Go `audio.NewInfiniteLoop`（BGM loop count 0）。
    pub looping: bool,
    /// R5 階段 4 r5o 新增（共用變更）：這個播放請求的序號，建立時由 [`next_voice_serial`] 配發，從 1 起
    /// 遞增。Go 以 `*audio.Player` 指標分辨「同一個播放器」與「同內容的新播放器」（例如停掉同一首曲子
    /// 再重播）；外殼以序號做同樣的分辨。`Default` 的 0 只出現在測試替身。
    pub serial: u64,
}

impl AudioVoice {
    /// Go `IsPlaying()`。
    pub fn is_playing(&self) -> bool {
        self.playing && !self.closed
    }
}

/// R5 階段 4 r5o 新增：配發下一個 [`AudioVoice::serial`]（行程內唯一，不進存檔與檢查點）。
pub fn next_voice_serial() -> u64 {
    static NEXT: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(1);
    NEXT.fetch_add(1, std::sync::atomic::Ordering::Relaxed)
}

/// R5 階段 4 r5o 新增：[`AudioVoice`] 在 `Game` 裡的位置，見 `Game::audio_voices_mut`。
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum VoiceSlot {
    /// Go `g.bgm`（BGM，可能循環）。
    Bgm,
    /// Go `g.titleANI1Voice`（標題 ANI #1 專用音效）。
    TitleAni1,
    /// Go `g.nativePhysicalVoice`（物理演出的同通道音效）。
    NativePhysical,
    /// Go `g.sfxVoices`（疊播的一次性音效）。
    Sfx,
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

#[cfg(test)]
#[path = "support/support_test.rs"]
mod support_test;

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
