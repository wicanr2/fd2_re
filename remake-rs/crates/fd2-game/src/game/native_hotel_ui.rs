//! Go 來源：`remake/cmd/fd2/native_hotel_ui.go`（284 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! E 組另補移植 `currentNativeHotelInput`（`r8`，`campInput` 的酒店分支呼叫），改收本幀輸入。

use super::*;

pub(crate) const NATIVE_HOTEL_PORTRAIT_ID: i64 = 0x81;

pub(crate) const NATIVE_HOTEL_PROMPT_TEXT: i64 = 0x249;

pub(crate) const NATIVE_HOTEL_SAVED_TEXT: i64 = 0x294;

pub(crate) const NATIVE_HOTEL_RESOURCE_ID: i64 = 13;

pub(crate) const NATIVE_HOTEL_SERVICE_SAVE: i64 = 1;

pub(crate) const NATIVE_HOTEL_SERVICE_LOAD: i64 = 2;

pub(crate) const NATIVE_HOTEL_SERVICE_EXIT: i64 = 3;

/// Go：`native_hotel_ui.go:30`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeHotelUIAssets {
    pub(crate) assets: Option<campaign::NativeShopAssets>,
    pub(crate) portraits: Vec<dato::Frame>,
    pub(crate) slots_box: fdother::LMI1Entry,
}

/// Go：`native_hotel_ui.go:36`，`func loadNativeHotelUIAssets(shared *nativeClassUIAssets) (*nativeHotelUIAssets, error)`
pub(crate) fn load_native_hotel_ui_assets(shared: Option<&NativeClassUIAssets>) -> Result<Option<NativeHotelUIAssets>> {
    let Some((shared, strings, font)) =
        shared.filter(|s| s.dialogue.len() > 17).and_then(|s| Some((s, s.strings.as_ref()?, s.font.as_ref()?)))
    else {
        return Err(game_err!("native hotel UI: shared facility assets unavailable"));
    };
    let assets = campaign::load_separated_native_shop_assets(&separated_asset_path(""), NATIVE_HOTEL_RESOURCE_ID)?;
    let portraits = load_native_separated_portrait(NATIVE_HOTEL_PORTRAIT_ID)?;
    if portraits.is_empty() {
        return Err(game_err!("native hotel UI: DATO 0x81 has no frames"));
    }
    let slots_box = fdother::load_separated_load_slots_frame(separated_asset_path("ui"))?;
    // 預驗服務後提示，避免還原資料後才發現1DE／1DF／24A文字或字模不足。
    for text in [NATIVE_HOTEL_PROMPT_TEXT, 0x24a, NATIVE_HOTEL_SAVED_TEXT, 0x1de, 0x1df] {
        let mut frame = campaign::compose_native_church_dialogue_overlay_at(
            &assets.background,
            &shared.dialogue,
            &portraits[0],
            campaign::native_facility_portrait_offset(NATIVE_HOTEL_PORTRAIT_ID),
        )?;
        campaign::compose_native_church_text_at(&mut frame, strings, font, text, campaign::NATIVE_SHOP_TEXT_OFFSET)?;
    }
    Ok(Some(NativeHotelUIAssets { assets: Some(assets), portraits, slots_box }))
}

impl Game {
    /// setupNativeHotel 進酒店節點時切到原生介面；素材不齊就維持舊版面（回 false）。
    ///
    /// Go：`native_hotel_ui.go:69`，`func (g *Game) setupNativeHotel() bool`
    pub(crate) fn setup_native_hotel(&mut self) -> bool {
        // Go 直接解參考 `g.camp`（呼叫端 `enterNode` 已確認 campaign 存在）。
        let Some(n) = self.camp.as_ref().expect("setupNativeHotel: g.camp is nil").node() else {
            return false;
        };
        if n.r#type != "hotel" || self.native_hotel_ui.is_none() || self.native_class_ui.is_none() {
            return false;
        }
        self.native_hotel_mode = String::from("menu");
        self.native_hotel_slot_sel = 0;
        self.hotel_sel = 0;
        self.reset_native_shop_ui_pulse();
        true
    }
}

/// nativeHotelInput 是酒店原生介面的輸入；與 0x2D7BD 相同：左右換圖示、enter 選、
/// esc 離開（0x2FEA0）。存檔列表上下選槽、enter 寫檔、esc 回選單；「記錄儲存完畢」
/// 任一鍵回選單。
///
/// Go：`native_hotel_ui.go:169`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeHotelInput {
    pub(crate) delta: i64,
    pub(crate) up: bool,
    pub(crate) down: bool,
    pub(crate) enter: bool,
    pub(crate) esc: bool,
}

/// Go：`native_hotel_ui.go:174`，`func currentNativeHotelInput(enter bool) nativeHotelInput`（`r8`，E 組隨
/// `campInput` 補移植）
///
/// 簽名改寫：Go 讀全域鍵盤狀態，Rust 讀本幀輸入 `input`。
pub(crate) fn current_native_hotel_input(input: &TickInput, enter: bool) -> NativeHotelInput {
    let mut r#in = NativeHotelInput { enter, esc: input.is_just_pressed(Key::Escape), ..Default::default() };
    if input.is_just_pressed(Key::ArrowLeft) {
        r#in.delta = -1;
    }
    if input.is_just_pressed(Key::ArrowRight) {
        r#in.delta = 1;
    }
    r#in.up = input.is_just_pressed(Key::ArrowUp);
    r#in.down = input.is_just_pressed(Key::ArrowDown);
    r#in
}

impl Game {
    /// Go：`native_hotel_ui.go:187`，`func (g *Game) handleNativeHotelInput(in nativeHotelInput) bool`
    pub(crate) fn handle_native_hotel_input(&mut self, r#in: NativeHotelInput) -> bool {
        match self.native_hotel_mode.as_str() {
            "menu" => {
                if r#in.delta == -1 {
                    self.hotel_sel = (self.hotel_sel + 3) % 4;
                    self.reset_native_shop_ui_pulse();
                }
                if r#in.delta == 1 {
                    self.hotel_sel = (self.hotel_sel + 1) % 4;
                    self.reset_native_shop_ui_pulse();
                }
                if r#in.esc {
                    self.leave_hotel();
                    return true;
                }
                if r#in.enter {
                    match self.hotel_sel {
                        NATIVE_HOTEL_SERVICE_SAVE => {
                            self.native_hotel_mode = String::from("slots");
                            self.native_hotel_slot_sel = 0;
                        }
                        NATIVE_HOTEL_SERVICE_LOAD => {
                            self.native_hotel_mode = String::from("loadslots");
                            self.native_hotel_slot_sel = 0;
                        }
                        NATIVE_HOTEL_SERVICE_EXIT => self.leave_hotel(),
                        0 => {
                            // 0x2FFA5 打聽消息：進本章的傳聞 story 節點（沒有就留在選單）。
                            // Go 直接解參考 `g.camp`。
                            let camp = self.camp.as_mut().expect("handleNativeHotelInput: g.camp is nil");
                            if camp.node().is_some_and(|n| !n.rumor.is_empty()) {
                                self.native_hotel_mode.clear();
                                self.native_hotel_prompt_return = true;
                                camp.advance("rumor");
                                self.enter_node();
                            }
                        }
                        _ => {
                            self.apply_hotel_service_selection(self.hotel_sel as u8);
                        }
                    }
                }
                true
            }
            "slots" | "loadslots" => {
                if r#in.up && self.native_hotel_slot_sel > 0 {
                    self.native_hotel_slot_sel -= 1;
                }
                if r#in.down && self.native_hotel_slot_sel < 3 {
                    self.native_hotel_slot_sel += 1;
                }
                if r#in.esc {
                    self.native_hotel_mode = String::from("menu");
                    self.native_hotel_prompt_return = true;
                    return true;
                }
                if r#in.enter {
                    if self.native_hotel_mode == "loadslots" {
                        todo!("R8: native_hotel_ui.go:confirmNativeHotelLoadSlot（酒店讀檔）");
                    } else if self.save_game_to_slot(self.native_hotel_slot_sel).is_ok() {
                        self.native_hotel_mode = String::from("saved");
                    }
                }
                true
            }
            "saved" | "loaded" => {
                if r#in.enter || r#in.esc {
                    self.native_hotel_mode = String::from("menu");
                    self.native_hotel_prompt_return = true;
                }
                true
            }
            "load_blocked" => {
                if r#in.enter || r#in.esc {
                    self.native_hotel_mode = String::from("loadslots");
                }
                true
            }
            _ => false,
        }
    }
}

#[cfg(test)]
mod native_hotel_ui_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// composeNativeHotelStable 是背景＋對話框＋店主頭像（指定 DATO 幀）＋「有什麼事嗎？」。
    ///
    /// Go：`native_hotel_ui.go:82`，`func (g *Game) composeNativeHotelStable(portraitFrame int) ([]byte, bool)`
    pub(crate) fn compose_native_hotel_stable(&self, portrait_frame: i64) -> (Vec<u8>, bool) {
        todo!("r5:native_hotel_ui.go:composeNativeHotelStable")
    }

    /// composeNativeHotelFrame 依目前模式組出整幀；portraitFrame 是店主頭像的 DATO 幀
    /// （0x16559 等待期間會眨眼，重播對每一幀各出一張）。
    ///
    /// Go：`native_hotel_ui.go:106`，`func (g *Game) composeNativeHotelFrame(portraitFrame int) ([]byte, bool)`
    pub(crate) fn compose_native_hotel_frame(&mut self, portrait_frame: i64) -> (Vec<u8>, bool) {
        todo!("r5:native_hotel_ui.go:composeNativeHotelFrame")
    }

    /// Go：`native_hotel_ui.go:159`，`func (g *Game) drawNativeHotel(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_hotel(&mut self, screen: &mut Picture) -> bool {
        todo!("r5:native_hotel_ui.go:drawNativeHotel")
    }
}
