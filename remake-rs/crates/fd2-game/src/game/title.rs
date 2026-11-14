//! Go 來源：`remake/cmd/fd2/title.go`（890 行）。
//!
//! title.go — 開頭動畫與主選單。目前接入已驗證的發行商畫面、AFM／FDOTHER
//! 資源、個別選單輸入與交錯排程；兩段 wipe、logo 揭示與精確 DAC 時序仍未達 E2。
//! ① 魔王立繪 320×735(FDOTHER #0x45-0x49 五幀直疊)由下往上垂直捲動(視窗 200 高,
//!
//!     src y=535→0,原版 0x1fa85;任意鍵跳過)。淡入目前仍用 ColorScale 作 E1
//!     可玩近似；原版 0x1f525 是65次六位元 DAC 寫入，尚未接到 title indexed path。
//!
//! ② 抹除轉場 → 標題畫面(FDOTHER #7 sub0,FLAME DRAGON logo,palette=FDOTHER #8)
//!   - 三選單項 START/LOAD/CONTINUE(#7 sub1-6 未選/選中素材)。
//!
//! ③ 選單:↑↓ wrap+游標音、Enter 確認(選中項閃 4 次,0x1fe2c);無 ESC 分支(doc23 §3)。
//!
//! 骨架：Go 檔的 22 個函式中產生 13 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`title.go:27`
#[derive(Debug, Clone, Default)]
pub(crate) struct TitleAssets {
    /// 320×735 立繪
    pub(crate) scroll: Option<Arc<Picture>>,
    /// 同 indexed pixels，FDOTHER #102 短暫 DAC
    pub(crate) scroll_pulse: Option<Arc<Picture>>,
    /// 320×200 標題畫面
    pub(crate) title: Option<Arc<Picture>>,
    /// FDOTHER #74 + palette #76 漢堂發行商畫面
    pub(crate) publisher: Option<Arc<Picture>>,
    /// sub_286BD phase 40→0，FDOTHER #69..73/#101
    pub(crate) red_fade_frames: [Option<Arc<Picture>>; 20],
    /// sub_286BD phase 0→40，FDOTHER #7/#8
    pub(crate) title_fade_frame: [Option<Arc<Picture>>; 20],
    pub(crate) hd_items: [[ModernHDTitleMenuItem; 2]; 3],
    /// START/LOAD/CONTINUE ×(未選/選中)
    pub(crate) items: [[Option<Arc<Picture>>; 2]; 3],
    /// 靜態幕:0=守護者(FDOTHER#100)、1=浮空城(#75)
    pub(crate) cut_static: [Option<Arc<Picture>>; 2],
    /// 離線分離後的 ANI indexed frame／DAC snapshots
    pub(crate) ani_clips: BTreeMap<i64, afm::Clip>,
}

impl Game {
    /// applyTitleMenuEvent 是標題主選單輸入的單一具型別擁有者；Ebiten 鍵盤與
    /// 決定性玩家路徑測試都必須經過此接縫。
    ///
    /// Go：`title.go:51`，`func (g *Game) applyTitleMenuEvent(event TitleMenuEvent) bool`
    pub(crate) fn apply_title_menu_event(&mut self, event: TitleMenuEvent) -> bool {
        todo!("title.go:applyTitleMenuEvent")
    }
}

/// 開場過場腳本把 sub_1F894 已證實的 esi=535→0 捲動與 AFM／FDOTHER 插播
/// 投影成60Hz typed steps。原始位址、caller 與直接指令見
/// docs/data/ida/fd2_title_scroll_schedule_ida.txt；這裡只擁有 runtime 排程。
///
/// Go：`title.go:114`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct CutStep {
    /// "publisher"／"afm"／"static"／"scroll"／"hold"／palette steps
    pub(crate) kind: String,
    /// afm:ANI.DAT index；static:cutStatic index
    pub(crate) res: i64,
    /// afm:每幀；其他:整個 step 的60Hz幀數
    pub(crate) tick: i64,
    /// afm 是否可由任意鍵中斷
    pub(crate) skip: bool,
    /// scroll 的原版 esi 起點
    pub(crate) scroll_from: i64,
    /// scroll 的原版 esi 終點
    pub(crate) scroll_to: i64,
}

/// Go：`title.go:123`
pub(crate) static CUT_SCRIPT: LazyLock<Vec<CutStep>> = LazyLock::new(|| {
    vec![
        CutStep { kind: String::from("publisher"), tick: 119, ..Default::default() },
        CutStep { kind: String::from("afm"), res: 3, tick: 5, skip: true, ..Default::default() },
        CutStep { kind: String::from("scroll"), tick: 153, scroll_from: 535, scroll_to: 450, ..Default::default() },
        CutStep { kind: String::from("static"), res: 0, tick: 45, ..Default::default() },
        CutStep { kind: String::from("scroll"), tick: 216, scroll_from: 450, scroll_to: 330, ..Default::default() },
        CutStep { kind: String::from("afm"), res: 4, tick: 5, ..Default::default() },
        CutStep { kind: String::from("afm"), res: 5, tick: 3, ..Default::default() },
        CutStep { kind: String::from("scroll"), tick: 216, scroll_from: 330, scroll_to: 210, ..Default::default() },
        CutStep { kind: String::from("afm"), res: 6, tick: 5, ..Default::default() },
        CutStep { kind: String::from("afm"), res: 7, tick: 3, ..Default::default() },
        CutStep { kind: String::from("scroll"), tick: 180, scroll_from: 210, scroll_to: 110, ..Default::default() },
        CutStep { kind: String::from("afm"), res: 8, tick: 5, ..Default::default() },
        CutStep { kind: String::from("scroll"), tick: 153, scroll_from: 110, scroll_to: 25, ..Default::default() },
        CutStep { kind: String::from("afm"), res: 0, tick: 1, ..Default::default() },
        CutStep { kind: String::from("scroll"), tick: 27, scroll_from: 25, scroll_to: 10, ..Default::default() },
        CutStep { kind: String::from("static"), res: 1, tick: 60, ..Default::default() },
        CutStep { kind: String::from("scroll"), tick: 18, scroll_from: 10, scroll_to: 0, ..Default::default() },
        CutStep { kind: String::from("hold"), tick: 60, ..Default::default() },
        CutStep { kind: String::from("redfade"), tick: 20, ..Default::default() },
        CutStep { kind: String::from("redhold"), tick: 6, ..Default::default() },
        CutStep { kind: String::from("afm"), res: 1, tick: 1, skip: true, ..Default::default() },
        CutStep { kind: String::from("titlefade"), tick: 20, ..Default::default() },
    ]
});

/// Go：`title.go:157`，`func loadTitleAssets() (*titleAssets, error)`
pub(crate) fn load_title_assets() -> Result<Option<TitleAssets>> {
    todo!("title.go:loadTitleAssets")
}

/// Go：`title.go:178`，`func loadSeparatedTitleFDOTHER(packRoot string, target *titleAssets) error`
pub(crate) fn load_separated_title_fdother(pack_root: &str, target: Option<&mut TitleAssets>) -> Result<()> {
    todo!("title.go:loadSeparatedTitleFDOTHER")
}

/// Go：`title.go:282`，`func maskedRGBA(frame fdother.Frame, palette color.Palette) *image.NRGBA`
pub(crate) fn masked_rgba(frame: &fdother::Frame, palette: &ColorPalette) -> Option<Arc<Picture>> {
    todo!("title.go:maskedRGBA")
}

/// loadSeparatedTitlePublisher 是正式 runtime owner；只消費分離 surface與DAC，
/// 不在失敗後回退 FDOTHER.DAT。
///
/// Go：`title.go:333`，`func loadSeparatedTitlePublisher(packRoot string) (*image.Paletted, error)`
pub(crate) fn load_separated_title_publisher(pack_root: &str) -> Result<Option<Arc<IndexedFrame>>> {
    todo!("title.go:loadSeparatedTitlePublisher")
}

/// Go：`title.go:354`，`func nativeTitlePaletted(pixels, dac []byte) (*image.Paletted, error)`
pub(crate) fn native_title_paletted(pixels: &[u8], dac: &[u8]) -> Result<Option<Arc<IndexedFrame>>> {
    todo!("title.go:nativeTitlePaletted")
}

/// Go：`title.go:424`，`func buildTitlePaletteTransitions(scrollPixels, scrollDAC, titlePixels, titleDAC []byte) ([20]*image.Paletted, [20]*image.Paletted, error)`
pub(crate) fn build_title_palette_transitions(
    scroll_pixels: &[u8],
    scroll_dac: &[u8],
    title_pixels: &[u8],
    title_dac: &[u8],
) -> Result<([Option<Arc<IndexedFrame>>; 20], [Option<Arc<IndexedFrame>>; 20])> {
    todo!("title.go:buildTitlePaletteTransitions")
}

/// titlePalettePhase 把原版41個8ms DAC相位投影到20個60Hz畫面，保留0與40端點。
///
/// Go：`title.go:470`，`func titlePalettePhase(tick int) int`
pub(crate) fn title_palette_phase(tick: i64) -> i64 {
    todo!("title.go:titlePalettePhase")
}

impl Game {
    /// loadCutClip 將已完整預檢的分離 ANI 資源建立成 Ebiten 影格。
    ///
    /// Go：`title.go:506`，`func (g *Game) loadCutClip(res int) []*ebiten.Image`
    pub(crate) fn load_cut_clip(&mut self, res: i64) -> Vec<Option<Arc<Picture>>> {
        todo!("title.go:loadCutClip")
    }

    /// Go：`title.go:521`，`func (g *Game) enterTitleMenu()`
    pub(crate) fn enter_title_menu(&mut self) {
        todo!("title.go:enterTitleMenu")
    }

    /// cutAdvance 前進到下一步驟(重置該步狀態);全部播完 → 進選單。
    ///
    /// Go：`title.go:532`，`func (g *Game) cutAdvance()`
    pub(crate) fn cut_advance(&mut self) {
        todo!("title.go:cutAdvance")
    }

    /// trySkipTitleCutStep 保存兩個原版 input owner：AFM 第三參數只中斷明確
    /// skippable 的當前幕；sub_1F894 捲動列後的 pending key 則直接進標題揭示。
    ///
    /// Go：`title.go:553`，`func (g *Game) trySkipTitleCutStep(step cutStep, anyKey bool) bool`
    pub(crate) fn try_skip_title_cut_step(&mut self, step: &CutStep, any_key: bool) -> bool {
        todo!("title.go:trySkipTitleCutStep")
    }

    /// titleUpdate 處理開頭動畫/主選單輸入。回傳 true = 仍在 title 流程。
    ///
    /// Go：`title.go:573`，`func (g *Game) titleUpdate() bool`
    pub(crate) fn title_update(&mut self) -> bool {
        todo!("title.go:titleUpdate")
    }
}
