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
        if self.title_phase != "menu" {
            return false;
        }
        let mut menu = TitleMenuState { selection: self.title_sel, flash_ticks: self.title_flash };
        if event == TITLE_MENU_UP || event == TITLE_MENU_DOWN {
            let sample = self.sfx_title_move.clone();
            self.play_raw(&sample);
        } else if event == TITLE_MENU_CONFIRM {
            let sample = self.sfx_title_confirm.clone();
            self.play_raw(&sample);
        }
        let action = menu.step(event);
        (self.title_sel, self.title_flash) = (menu.selection, menu.flash_ticks);
        match action {
            TITLE_MENU_LOAD_SLOTS => {
                self.title_slot_sel = 0;
                self.title_phase = String::from("loadslots");
            }
            TITLE_MENU_CONTINUE => {
                // Go：`g.loadNativeContinueFromCurrentSnapshot(os.Getenv("FD2_NATIVE_SAVE"))`；第一章重播走
                // START，`native_continue_handoff.go` 的這個函式是 `r8`。
                todo!("R8: native_continue_handoff.go:loadNativeContinueFromCurrentSnapshot（標題 CONTINUE）")
            }
            TITLE_MENU_START => {
                self.balance = None;
                self.balance_ui = BalanceEditor::default();
                if self.title_needs_new_campaign {
                    // Go：`g.restartCampaignFromTitle()`（`native_defeat.go`，`r8`：戰敗後回標題才會設這個旗標）。
                    todo!("R8: native_defeat.go:restartCampaignFromTitle（戰敗後重新開始）")
                }
                self.native_chapter_slot_started = true;
                self.title_phase.clear();
            }
            _ => {}
        }
        true
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

/// Rust 新增：Go `ebiten.NewImageFromImage(*image.Paletted)` 的 `Arc` 版本。
fn picture_of(frame: &IndexedFrame) -> Option<Arc<Picture>> {
    Some(Arc::new(picture_from_indexed(frame)))
}

/// Go：`title.go:150`，`func titleLogoTransitionIndex() int`（`r8`：`trySkipTitleCutStep` 需要它，
/// 隨本檔一起移植）。
pub(crate) fn title_logo_transition_index() -> i64 {
    CUT_SCRIPT.iter().position(|step| step.kind == "redfade").unwrap_or(CUT_SCRIPT.len()) as i64
}

/// Go：`title.go:157`，`func loadTitleAssets() (*titleAssets, error)`
pub(crate) fn load_title_assets() -> Result<Option<TitleAssets>> {
    let mut t = TitleAssets::default();
    let pack_root = separated_asset_path("");
    let publisher = load_separated_title_publisher(&pack_root).map_err(|e| e.wrap("標題發行商分離素材："))?;
    t.publisher = publisher.as_deref().and_then(picture_of);
    load_separated_title_fdother(&pack_root, Some(&mut t))?;
    for resource in [0, 1, 3, 4, 5, 6, 7, 8] {
        let clip = afm::load_separated_resource(join_path(&[&pack_root, "animations"]), resource)
            .map_err(|e| GameError::from(e).wrap(format!("標題 ANI #{resource} 分離素材：")))?;
        t.ani_clips.insert(resource, clip);
    }
    Ok(Some(t))
}

/// Go：`title.go:178`，`func loadSeparatedTitleFDOTHER(packRoot string, target *titleAssets) error`
pub(crate) fn load_separated_title_fdother(pack_root: &str, target: Option<&mut TitleAssets>) -> Result<()> {
    let Some(target) = target else {
        return Err(GameError::new("標題素材 target 不可為 nil"));
    };
    let (surface_root, palette_root) = (join_path(&[pack_root, "surfaces"]), join_path(&[pack_root, "palette"]));
    let (scroll_dac, scroll_palette) = fdother::load_separated_fdother_palette(&palette_root, 101)
        .map_err(|e| GameError::from(e).wrap("標題捲動調色盤："))?;
    let scroll_palette = color_palette_from_nrgba(&scroll_palette);
    let mut scroll_pixels = vec![0u8; 320 * 735];
    for (index, resource) in [69i64, 70, 71, 72, 73].into_iter().enumerate() {
        let mut frame = fdother::load_separated_single_frame(&surface_root, "FDOTHER.DAT", resource)
            .map_err(|e| GameError::from(e).wrap(format!("標題捲動 FDOTHER #{resource}：")))?;
        if frame.width != 320 || frame.height != 147 {
            return Err(game_err!("標題捲動 FDOTHER #{resource} 幾何={}x{}", frame.width, frame.height));
        }
        frame.y = index as i64 * 147;
        frame.blit(&mut scroll_pixels, 320, -1)?;
    }
    target.scroll = picture_of(&indexed_frame(320, 735, &scroll_pixels, scroll_palette));
    let (_, pulse_palette) = fdother::load_separated_fdother_palette(&palette_root, 102)
        .map_err(|e| GameError::from(e).wrap("標題捲動短暫調色盤："))?;
    target.scroll_pulse =
        picture_of(&indexed_frame(320, 735, &scroll_pixels, color_palette_from_nrgba(&pulse_palette)));

    let (title_dac, title_palette) = fdother::load_separated_fdother_palette(&palette_root, 8)
        .map_err(|e| GameError::from(e).wrap("標題主畫面調色盤："))?;
    let title_palette = color_palette_from_nrgba(&title_palette);
    let title_frame = fdother::load_separated_nested_single_frame(&surface_root, 7, 0)
        .map_err(|e| GameError::from(e).wrap("標題主畫面："))?;
    if title_frame.width != 320 || title_frame.height != 200 {
        return Err(game_err!("標題主畫面幾何={}x{}", title_frame.width, title_frame.height));
    }
    let mut title_pixels = vec![0u8; 320 * 200];
    title_frame.blit(&mut title_pixels, 320, -1)?;
    target.title = picture_of(&indexed_frame(320, 200, &title_pixels, title_palette.clone()));

    let want_menu_geometry: [(i64, i64); 6] = [(61, 7), (61, 7), (62, 7), (62, 7), (62, 8), (62, 8)];
    for nested in 1..=6i64 {
        let frame = fdother::load_separated_nested_single_frame(&surface_root, 7, nested)
            .map_err(|e| GameError::from(e).wrap(format!("標題選單 FDOTHER #7/{nested}：")))?;
        let want = want_menu_geometry[(nested - 1) as usize];
        if frame.width != want.0 || frame.height != want.1 {
            return Err(game_err!("標題選單 FDOTHER #7/{nested} 幾何={}x{}", frame.width, frame.height));
        }
        let (item, state) = (((nested - 1) / 2) as usize, ((nested - 1) % 2) as usize);
        target.items[item][state] = masked_rgba(&frame, &title_palette);
        let hd = freeze_modern_hd_title_menu_item(
            &frame,
            &title_dac,
            &title_palette,
            item as i64,
            state as i64,
            target.items[item][state].as_ref(),
        )?;
        target.hd_items[item][state] = hd.unwrap_or_default();
    }

    for (slot, (resource, palette)) in [(100i64, 99i64), (75, 76)].into_iter().enumerate() {
        let frame = fdother::load_separated_single_frame(&surface_root, "FDOTHER.DAT", resource)
            .map_err(|e| GameError::from(e).wrap(format!("標題靜態幕 FDOTHER #{resource}：")))?;
        if frame.width != 320 || frame.height != 200 {
            return Err(game_err!("標題靜態幕 FDOTHER #{resource} 幾何={}x{}", frame.width, frame.height));
        }
        let (_, palette) = fdother::load_separated_fdother_palette(&palette_root, palette)?;
        let mut pixels = vec![0u8; 320 * 200];
        frame.blit(&mut pixels, 320, -1)?;
        target.cut_static[slot] = picture_of(&indexed_frame(320, 200, &pixels, color_palette_from_nrgba(&palette)));
    }

    let (red, title) =
        build_title_palette_transitions(&scroll_pixels[..320 * 200], &scroll_dac, &title_pixels, &title_dac)?;
    for index in 0..red.len() {
        target.red_fade_frames[index] = red[index].as_deref().and_then(picture_of);
        target.title_fade_frame[index] = title[index].as_deref().and_then(picture_of);
    }
    Ok(())
}

/// Go：`title.go:282`，`func maskedRGBA(frame fdother.Frame, palette color.Palette) *image.NRGBA`
///
/// Go 把 `palette[i].RGBA()`（預乘值）的高 8 位元直接寫進 NRGBA 的像素；Rust 照寫同樣的位元組
/// （調色盤是不透明色，預乘與否相同）。
pub(crate) fn masked_rgba(frame: &fdother::Frame, palette: &ColorPalette) -> Option<Arc<Picture>> {
    let (w, h) = (frame.width.max(0), frame.height.max(0));
    let mut pix = vec![0u8; (w * h * 4) as usize];
    for (index, &mask) in frame.mask.iter().enumerate() {
        if mask == 0 {
            continue;
        }
        // Go 的 palette 索引越界會 panic；Rust 同樣以索引存取。
        let c = palette[frame.indexed[index] as usize];
        pix[index * 4..index * 4 + 4].copy_from_slice(&[c.r, c.g, c.b, c.a]);
    }
    Some(Arc::new(Picture { width: w, height: h, pix }))
}

/// loadSeparatedTitlePublisher 是正式 runtime owner；只消費分離 surface與DAC，
/// 不在失敗後回退 FDOTHER.DAT。
///
/// Go：`title.go:333`，`func loadSeparatedTitlePublisher(packRoot string) (*image.Paletted, error)`
pub(crate) fn load_separated_title_publisher(pack_root: &str) -> Result<Option<Arc<IndexedFrame>>> {
    let frame = fdother::load_separated_single_frame(join_path(&[pack_root, "surfaces"]), "FDOTHER.DAT", 74)?;
    if frame.width != 320 || frame.height != 200 {
        return Err(game_err!("title publisher geometry={}x{}, want 320x200", frame.width, frame.height));
    }
    let mut pixels = vec![0u8; 320 * 200];
    frame.blit(&mut pixels, 320, -1)?;
    let (_, palette) = fdother::load_separated_fdother_palette(join_path(&[pack_root, "palette"]), 76)?;
    Ok(Some(Arc::new(indexed_frame(320, 200, &pixels, color_palette_from_nrgba(&palette)))))
}

/// Go：`title.go:354`，`func nativeTitlePaletted(pixels, dac []byte) (*image.Paletted, error)`
pub(crate) fn native_title_paletted(pixels: &[u8], dac: &[u8]) -> Result<Option<Arc<IndexedFrame>>> {
    if pixels.len() != 320 * 200 {
        return Err(game_err!("title indexed framebuffer={}, want 64000", pixels.len()));
    }
    let palette = fdother::parse_vga_palette(dac)?;
    Ok(Some(Arc::new(indexed_frame(320, 200, pixels, color_palette_from_nrgba(&palette)))))
}

/// Go：`title.go:424`，`func buildTitlePaletteTransitions(scrollPixels, scrollDAC, titlePixels, titleDAC []byte) ([20]*image.Paletted, [20]*image.Paletted, error)`
///
/// Go 錯誤時回傳已填一半的陣列；Rust 的 `Result` 只帶錯誤（呼叫端在錯誤時都丟棄陣列）。
pub(crate) fn build_title_palette_transitions(
    scroll_pixels: &[u8],
    scroll_dac: &[u8],
    title_pixels: &[u8],
    title_dac: &[u8],
) -> Result<([Option<Arc<IndexedFrame>>; 20], [Option<Arc<IndexedFrame>>; 20])> {
    let mut red_frames: [Option<Arc<IndexedFrame>>; 20] = Default::default();
    let mut title_frames: [Option<Arc<IndexedFrame>>; 20] = Default::default();
    if scroll_pixels.len() != 320 * 200 || title_pixels.len() != 320 * 200 {
        return Err(GameError::new("title transition source geometry mismatch"));
    }
    for tick in 0..20i64 {
        let phase = title_palette_phase(tick);
        let red_dac = fdother::interpolate_native_compound_dac(scroll_dac, 0, 255, 40 - phase, [63, 0, 0])?;
        red_frames[tick as usize] = native_title_paletted(scroll_pixels, &red_dac)?;
        let fade_dac = fdother::interpolate_native_compound_dac(title_dac, 0, 255, phase, [56, 60, 63])?;
        title_frames[tick as usize] = native_title_paletted(title_pixels, &fade_dac)?;
    }
    Ok((red_frames, title_frames))
}

/// titlePalettePhase 把原版41個8ms DAC相位投影到20個60Hz畫面，保留0與40端點。
///
/// Go：`title.go:470`，`func titlePalettePhase(tick int) int`
pub(crate) fn title_palette_phase(tick: i64) -> i64 {
    if tick <= 0 {
        return 0;
    }
    if tick >= 19 {
        return 40;
    }
    (tick * 40 + 9) / 19
}

impl Game {
    /// loadCutClip 將已完整預檢的分離 ANI 資源建立成 Ebiten 影格。
    ///
    /// Go：`title.go:506`，`func (g *Game) loadCutClip(res int) []*ebiten.Image`
    ///
    /// Go 對空影格的 clip 回傳非 nil 的空切片；Rust 的空 `Vec` 與「沒有 clip」相同。分離動畫每個資源都有
    /// 固定的非零影格數（`afm::load_separated_resource` 核對），兩者在正式素材下不會分歧。
    pub(crate) fn load_cut_clip(&mut self, res: i64) -> Vec<Option<Arc<Picture>>> {
        let Some(clip) = self.title_assets.as_ref().and_then(|t| t.ani_clips.get(&res)) else {
            return Vec::new();
        };
        clip.frames
            .iter()
            .map(|f| Some(Arc::new(Picture { width: f.width as i64, height: f.height as i64, pix: f.pix.clone() })))
            .collect()
    }

    /// Go：`title.go:521`，`func (g *Game) enterTitleMenu()`
    pub(crate) fn enter_title_menu(&mut self) {
        self.title_phase = String::from("menu");
        self.title_sel = 0;
        if let Some(assets) = self.title_assets.as_mut() {
            // AFM snapshots只供一次性開場；進選單後釋放約80 MiB的CPU影格，
            // 保留主選單仍需的FDOTHER畫面。
            assets.ani_clips = BTreeMap::new();
        }
    }

    /// cutAdvance 前進到下一步驟(重置該步狀態);全部播完 → 進選單。
    ///
    /// Go：`title.go:532`，`func (g *Game) cutAdvance()`
    pub(crate) fn cut_advance(&mut self) {
        if self.cut_idx >= 0 && self.cut_idx < CUT_SCRIPT.len() as i64 {
            let step = &CUT_SCRIPT[self.cut_idx as usize];
            if step.kind == "afm" && step.res == 1 {
                self.stop_title_ani1_sound();
            }
        }
        self.cut_idx += 1;
        self.cut_cur = Vec::new();
        (self.cut_frame, self.cut_tick) = (0, 0);
        if self.cut_idx >= CUT_SCRIPT.len() as i64 {
            self.enter_title_menu();
            return;
        }
        let next = &CUT_SCRIPT[self.cut_idx as usize];
        if next.kind == "scroll" {
            self.scroll_y = next.scroll_from as f64;
        }
    }

    /// trySkipTitleCutStep 保存兩個原版 input owner：AFM 第三參數只中斷明確
    /// skippable 的當前幕；sub_1F894 捲動列後的 pending key 則直接進標題揭示。
    ///
    /// Go：`title.go:553`，`func (g *Game) trySkipTitleCutStep(step cutStep, anyKey bool) bool`
    pub(crate) fn try_skip_title_cut_step(&mut self, step: &CutStep, any_key: bool) -> bool {
        if !any_key {
            return false;
        }
        if step.kind == "scroll" {
            // sub_1F894 0x1FC59..0x1FC66：每列後的 pending key 直接進
            // wipe／ANI #1，不是 AFM 第三參數的「只中斷當前幕」。
            self.cut_idx = title_logo_transition_index();
            self.cut_cur = Vec::new();
            (self.cut_frame, self.cut_tick) = (0, 0);
            return true;
        }
        if !step.skip {
            return false;
        }
        self.cut_advance();
        true
    }

    /// Go `len(inpututil.AppendJustPressedKeys(nil)) > 0`：本幀有任何鍵按下。
    fn title_any_key_pressed(&self) -> bool {
        !self.tick_input.pressed.is_empty()
    }

    /// titleUpdate 處理開頭動畫/主選單輸入。回傳 true = 仍在 title 流程。
    ///
    /// Go：`title.go:573`，`func (g *Game) titleUpdate() bool`
    ///
    /// Go 在 `titleAssets` 為 nil 時解參考會 panic；Rust 把缺少的素材視同該幕素材缺漏而略過
    /// （`loadGame` 在素材不全時失敗即關閉，正式路徑不會走到）。
    pub(crate) fn title_update(&mut self) -> bool {
        // Cross-platform projection of the original title caller's BIOS low word.
        // It is sampled throughout the normal title path, not synthesized from the save.
        let now = self.game_now();
        self.native_title_clock.sample(now);
        match self.title_phase.as_str() {
            "cutscene" => {
                if self.cut_idx >= CUT_SCRIPT.len() as i64 {
                    self.enter_title_menu();
                    return true;
                }
                let step = CUT_SCRIPT[self.cut_idx as usize].clone();
                if self.cut_idx == 0 && self.cut_frame == 0 && self.cut_tick == 0 {
                    // 開場/標題曲(RE 確認:boot 0x025db5 play_bgm(18,0),doc12 §15)
                    self.play_bgm("FDMUS_018");
                }
                // 原版第三參數只允許中斷當前 AFM 幕；不可跳過後續不可略過的幕。
                let any_key = self.title_any_key_pressed();
                if self.try_skip_title_cut_step(&step, any_key) {
                    return true;
                }
                let assets = self.title_assets.as_ref();
                match step.kind.as_str() {
                    "publisher" => {
                        if assets.is_none_or(|a| a.publisher.is_none()) {
                            self.cut_advance();
                            return true;
                        }
                        self.tick_cut_step(&step);
                        return true;
                    }
                    "static" => {
                        // FDOTHER 靜態幕:hold step.tick 個 tick；素材缺 → 跳過此幕
                        if assets.is_none_or(|a| a.cut_static[step.res as usize].is_none()) {
                            self.cut_advance();
                            return true;
                        }
                        self.tick_cut_step(&step);
                        return true;
                    }
                    "hold" | "redfade" | "redhold" | "titlefade" => {
                        self.tick_cut_step(&step);
                        return true;
                    }
                    "scroll" => {
                        // 原版 esi 區間；AFM／靜態幕只暫停，不重設位置
                        if assets.is_none_or(|a| a.scroll.is_none()) {
                            self.cut_advance();
                            return true;
                        }
                        self.cut_tick += 1;
                        let progress = self.cut_tick as f64 / step.tick as f64;
                        self.scroll_y = step.scroll_from as f64 + (step.scroll_to - step.scroll_from) as f64 * progress;
                        if self.cut_tick >= step.tick {
                            self.scroll_y = step.scroll_to as f64;
                            self.cut_advance();
                        }
                        return true;
                    }
                    _ => {}
                }
                // AFM 動畫幕
                if self.cut_cur.is_empty() {
                    // 進入此幕:執行期解碼
                    self.cut_cur = self.load_cut_clip(step.res);
                    (self.cut_frame, self.cut_tick) = (0, 0);
                    if self.cut_cur.is_empty() {
                        // 解碼失敗 → 跳過此幕(不整段中止)
                        self.cut_advance();
                        return true;
                    }
                    if step.res == 1 {
                        self.start_title_ani1_sound();
                    }
                }
                self.cut_tick += 1;
                if self.cut_tick >= step.tick {
                    self.cut_tick = 0;
                    self.cut_frame += 1;
                    if self.cut_frame >= self.cut_cur.len() as i64 {
                        // 此幕播完 → 下一步
                        self.cut_advance();
                    }
                }
                true
            }
            "scroll" => {
                if self.scroll_y >= 534.0 {
                    // 開場即配樂；同開場曲(RE 確認,取代舊猜測 FDMUS_004)
                    self.play_bgm("FDMUS_018");
                }
                self.scroll_y -= 1.5; // 捲動速度(原版逐列複製;待 dosbox 錄影校)
                let any_key = self.title_any_key_pressed();
                if self.scroll_y <= 0.0 || any_key {
                    // dosbox 實拍(doc23 §2.4):紅閃→「2」縮入→白閃→選單
                    self.title_phase = String::from("logozoom");
                    self.title_tick = 0;
                }
                true
            }
            "logozoom" => {
                self.title_tick += 1;
                if self.title_tick > 50 || self.title_any_key_pressed() {
                    // 紅閃12+縮放30+白閃8
                    self.enter_title_menu();
                }
                true
            }
            "menu" => {
                let input = &self.tick_input;
                let (up, down) = (input.is_just_pressed(Key::ArrowUp), input.is_just_pressed(Key::ArrowDown));
                let confirm = input.is_just_pressed(Key::Enter) || input.is_just_pressed(Key::Space);
                if up {
                    self.apply_title_menu_event(TITLE_MENU_UP);
                }
                if down {
                    self.apply_title_menu_event(TITLE_MENU_DOWN);
                }
                if confirm {
                    self.apply_title_menu_event(TITLE_MENU_CONFIRM);
                }
                self.apply_title_menu_event(TITLE_MENU_TICK);
                true
            }
            "loadslots" => {
                // Native 0x30550 selector: four bounded slots, no wrap, Enter/Space
                // confirms and Esc cancels back to the title menu.
                // Go 呼叫 `applyTitleSlotEvent`（`title.go:86`，`r8`：第一章重播走 START，沒有進讀檔槽）。
                let input = &self.tick_input;
                let event = (input.is_just_pressed(Key::ArrowUp) && self.title_slot_sel > 0)
                    || (input.is_just_pressed(Key::ArrowDown) && self.title_slot_sel < 3)
                    || input.is_just_pressed(Key::Escape)
                    || input.is_just_pressed(Key::Enter)
                    || input.is_just_pressed(Key::Space);
                if event {
                    todo!("R8: title.go:applyTitleSlotEvent（標題 LOAD 讀檔槽）")
                }
                true
            }
            _ => false,
        }
    }

    /// Go `titleUpdate` 裡 publisher／static／hold／palette 幕共用的 `cutTick++`、滿 `tick` 前進。
    fn tick_cut_step(&mut self, step: &CutStep) {
        self.cut_tick += 1;
        if self.cut_tick >= step.tick {
            self.cut_advance();
        }
    }
}

#[cfg(test)]
#[path = "title/title_test.rs"]
mod title_test;
