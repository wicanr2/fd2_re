//! 呈現回報（R4 階段 3，Rust 新增，沒有單一 Go 檔對應）。
//!
//! Go 的演出工作在 `Draw` 裡設 `drawn = true`，下一次 `Update` 才前進（120「畫過才推進」）。`fd2-game`
//! 把 `Draw` 拆成兩半：畫像素是外殼（`fd2-app`，R5）的事；`Draw` 裡改流程狀態的部分集中在本模組的
//! [`Game::acknowledge_present`]。
//!
//! # 契約
//!
//! - 外殼每畫完一幀呼叫一次 `acknowledge_present`；無畫面驅動端在 Go 呼叫 `g.Draw(screen)` 的地方直接呼叫。
//! - 它照 Go `Draw`（`main.go:9355`）的分支優先序選出本幀的呈現擁有者，做那個擁有者的 `draw*` 函式裡的
//!   狀態改寫：前置條件成立時設 `drawn`／`presented` 旗標，不成立時走 Go `Draw` 的失敗路徑
//!   （`fail*`／`cancel*`／寫 `loadErr`）。回傳本幀的擁有者，外殼據此畫對應的畫面。
//! - 前置條件只看狀態（幀索引、緩衝長度、調色盤長度），與 Go `draw*` 的 `return false` 條件逐條相同；
//!   Go 只在 GPU 上才會失敗的情況不存在於忠實模式。
//! - 不畫像素、不建 GPU 貼圖；唯一的例外是 Go 快取在演出工作裡、而 `step*` 以「有沒有快取」判斷流程的
//!   `NativePhysicalBodyPlayback::image`（`native_physical_body.go:433`、`:484`），這裡照 Go 建立同一份畫面。
//! - HD（`modern_*`）分支是 R6；第一章沒有執行到、Go 失敗路徑的函式還沒移植的分支是 R8。這些分支在擁有者存在時
//!   停在 `todo!`，不以猜測的旗標推進流程。
//! - Go `Draw` 尾段（地圖整幀、對話框、指令環、系統資訊等，`main.go:9799`..`10188`）的狀態改寫與合成交錯（原生整幀
//!   重建 `nativeMapVGA`、各設施與對白呈現設 `drawn`、`markActionOverlayDrawn` 等）。R5 階段 3 起這一段直接呼叫
//!   R5 移植的 [`Game::draw_field`]，畫到丟棄的 640×400 畫布：狀態改寫與 `Draw` 是同一份程式，像素不用。
//!
//! R5 階段 3 起，章重播的 `ack_presents` 照 Go 呼叫 [`Game::draw`]（Go 在城鎮出發、有原生場景的攻擊演出、END 回收
//! 三處呼叫 `g.Draw` 到暫存影像），不再經本模組；本模組留給不需要像素的呼叫端與單元測試。
//!
//! 逐項對應 Go 行號見 `PORTING.md`「R4 階段 3：呈現回報」。

use super::*;

/// 本幀的呈現擁有者（Go `Draw` 的 if 鏈，`main.go:9395`..`9797`；其後是地圖整幀）。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum PresentOwner {
    StartupBlocked,
    NativeDefeat,
    Title,
    NativeEnding,
    NativeCh20SkyKey,
    NativeCh23Loop,
    Native2189A,
    NativeUnitPresent,
    NativeCh28PostPresent,
    TransitionReveal,
    NoMap,
    IndexedTransition,
    NativeHealPresentation,
    NativeModifierPresentation,
    NativeAICommandModifier,
    NativeCommandPresentation,
    NativeAIItemPresentation,
    NativeTownDeparture,
    NativePaletteRamp,
    NativePalettePulse,
    SpawnIntroTransition,
    NativeTurnStagingIndexed,
    NativeFieldEvent61,
    NativeEndTurnRecovery,
    NativeAIIdleRecovery,
    AttackScene,
    Map,
}

fn vga_and_palette_ready(vga: &[u8], palette: &[ColorRgba]) -> bool {
    vga.len() as i64 == indexedmap::NATIVE_MAP_VGA_SIZE && palette.len() == 256
}

impl Game {
    /// Go `main.go:9356`..`9372`：每幀開頭清本幀呈現旗標（A 組表第 1 項），`beginModernHDText`（第 3 項，
    /// 忠實模式只清 `modernHDText`）。
    pub(crate) fn begin_present_frame(&mut self) {
        self.native_map_presented = false;
        self.posters.native_dialogue_visible = false;
        self.movement_panel_draw = None;
        self.native_interface_hud_visible = false;
        self.modern_opening_view = None;
        self.modern_hd_overlay = None;
        self.window_dialogue_border = None;
        self.modern_hd_scene = None;
        self.preparation_roster_hd = None;
        self.facility_roster_hd = None;
        self.facility_list_roster_hd = None;
        self.modern_hd_panel_portraits = Vec::new();
        self.modern_hd_battle_visible = false;
        self.sync_modern_hd_battle_owner();
        self.modern_hd_world_visible = false;
        self.modern_hd_timed_dialogue_visible = false;
        if self.banner_t <= 0
            && let Some(renderer) = &mut self.modern_hd_map_gpu
        {
            renderer.clear_banner();
        }
        // Go `defer g.beginModernHDText(screen)()`：忠實模式回傳的收尾是空函式。
        self.modern_hd_text = None;
        if self.modern_hd_interface_enabled() && self.font.is_some() {
            todo!("R6: modern_hd_text.go:92 beginModernHDText（HD 文字框與字型綁定）")
        }
    }

    /// Go `main.go:9376`..`9394` 的第二個 `defer`：每條 return 路徑都會執行（A 組表第 4、5 項）。
    pub(crate) fn finish_present_frame(&mut self) {
        if let Some(job) = self.native_turn_staging.as_mut()
            && !job.indexed
            && job.phase == NATIVE_TURN_STAGING_FLASH
        {
            job.drawn = true;
        }
        if self.help_visible && self.modern_hd_text.is_some() {
            todo!("R6: main.go:9390 clearModernHDText（說明畫面的 HD 文字層）")
        }
    }

    /// 呈現回報：外殼畫完一幀（或無畫面驅動端在 Go 呼叫 `g.Draw` 的地方）呼叫。見模組文件的契約。
    pub(crate) fn acknowledge_present(&mut self) -> PresentOwner {
        self.begin_present_frame();
        let owner = self.acknowledge_present_owner();
        self.finish_present_frame();
        owner
    }

    /// Go `Draw` 的 if 鏈（`main.go:9395`..`9797`）。
    fn acknowledge_present_owner(&mut self) -> PresentOwner {
        if self.startup_blocked {
            return PresentOwner::StartupBlocked;
        }
        if let Some(job) = self.native_defeat.as_mut() {
            // Go `native_defeat.go:160` drawNativeDefeat：每次都設 drawn（174），有斜坡時一併（176）。
            job.drawn = true;
            if let Some(ramp) = job.ramp.as_mut() {
                ramp.drawn = true;
            }
            return PresentOwner::NativeDefeat;
        }
        if !self.title_phase.is_empty() {
            // Go `title.go:716` drawTitle 沒有流程旗標；`localeMessage` 失敗寫 `loadErr` 與 HD 文字是繪圖側（R5／R6），
            // 截圖模式的 `captureShot` 是 fd2-tools。
            return PresentOwner::Title;
        }
        if self.native_ending.is_some() {
            todo!("R8: main.go:9413 drawNativeEndingPreview（結局）")
        }
        if self.native_ch20_sky_key.is_some() {
            todo!("R8: main.go:9417 drawNativeCh20SkyKey／failNativeCh20SkyKey（第二十章）")
        }
        if self.native_ch23_loop.is_some() {
            todo!("R8: main.go:9428 drawNativeCh23Loop／failNativeCh23Loop（第二十四章）")
        }
        if self.native2189_a.is_some() {
            todo!("R8: main.go:9439 drawNative2189A／failNative2189A（第二十二章）")
        }
        if self.native_unit_present.is_some() {
            todo!("R8: main.go:9450 drawNativeUnitPresent／failNativeUnitPresent")
        }
        if self.native_ch28_post_present.is_some() {
            todo!("R8: main.go:9461 drawNativeCh28PostPresent／failNativeCh28PostPresent")
        }
        if self.transition_reveal.is_some() {
            todo!("R8: main.go:9472 drawNative24B4D／failNative24B4D")
        }
        if self.m.is_none() {
            return PresentOwner::NoMap;
        }
        let hd_map_sprites = self.modern_story_portraits.as_ref().is_some_and(|set| set.map_sprites_hd.len() == 125);
        if let Some(job) = self.indexed_transition.as_mut() {
            // Go `native_indexed_transition.go:331`：長度不對時 Draw 只印除錯字，工作留著。
            if hd_map_sprites {
                todo!("R6: native_indexed_transition.go:336 HD 索引轉場幀")
            }
            if vga_and_palette_ready(&job.vga, &job.palette) {
                job.drawn = true;
            }
            return PresentOwner::IndexedTransition;
        }
        if self.native_heal_presentation.is_some() {
            self.present_native_command_heal(None);
            return PresentOwner::NativeHealPresentation;
        }
        if self.native_modifier_presentation.is_some() {
            todo!("R8: main.go:9510 drawNativeCommandModifierPresentation")
        }
        if self.native_ai_command_modifier.is_some() {
            self.present_native_ai_command_modifier(None);
            return PresentOwner::NativeAICommandModifier;
        }
        if let Some(job) = self.native_cmd0_presentation.as_mut() {
            if !job.present(None) {
                self.fail_native_command0_presentation(&game_err!("指令 0 演出畫面無效"));
            }
            return PresentOwner::NativeCommandPresentation;
        }
        if self.native_cmd6_presentation.is_some()
            || self.native_cmd1_presentation.is_some()
            || self.native_cmd2_presentation.is_some()
            || self.native_cmd3_presentation.is_some()
            || self.native_cmd5_presentation.is_some()
            || self.native_cmd7_presentation.is_some()
            || self.native_cmd8_presentation.is_some()
            || self.native_cmd9_ai_presentation.is_some()
            || self.native_cmd9_player.is_some()
            || self.native_cmd1012.is_some()
            || self.native_cmd24_presentation.is_some()
            || self.native_cmd29_presentation.is_some()
            || self.native_cmd32_presentation.is_some()
            || self.native_cmd33_presentation.is_some()
            || self.native_cmd34_presentation.is_some()
            || self.native_cmd35_presentation.is_some()
        {
            todo!("R8: main.go:9528..9701 指令 0～35 演出的 drawn 條件與 fail 路徑（第一章沒有）")
        }
        if self.native_ai_item_presentation.is_some() {
            if !self.acknowledge_native_ai_item_presentation() {
                self.fail_native_ai_item_presentation(&game_err!("敵方道具演出畫面無效"));
            }
            return PresentOwner::NativeAIItemPresentation;
        }
        let hd_interface = self.modern_hd_interface_enabled();
        if let Some(job) = self.native_town_departure.as_mut() {
            if hd_interface {
                todo!("R6: native_town_departure.go:88 drawModernHDTownDeparture")
            }
            // Go `native_town_departure.go:92`：不檢查範圍就取 `frames[step]`；越界時 Go panic。
            assert!(
                job.step >= 0 && (job.step as usize) < job.frames.len(),
                "native town departure: step out of range"
            );
            job.drawn = true;
            return PresentOwner::NativeTownDeparture;
        }
        let hd_battle =
            self.atk.is_some() && self.modern_hd_battle_wanted(self.atk.as_ref().and_then(|a| a.native_scene.as_ref()));
        if let Some(job) = self.native_palette_ramp.as_mut() {
            // Go `native_palette_ramp.go:164`。
            let hd = job.hd_battle_sources.is_some() && hd_battle;
            if vga_and_palette_ready(&job.vga, &job.palette) {
                if hd {
                    todo!("R6: native_palette_ramp.go:169 HD 物理演出返回 DAC")
                }
                job.drawn = true;
            }
            return PresentOwner::NativePaletteRamp;
        }
        if let Some(job) = self.native_palette_pulse.as_mut() {
            // Go `native_palette_pulse.go:158`。
            if vga_and_palette_ready(&job.vga, &job.palette) {
                job.drawn = true;
            }
            return PresentOwner::NativePalettePulse;
        }
        if let Some(job) = self.spawn_intro_transition.as_mut() {
            // Go `native_spawn_intro.go:360`。
            if vga_and_palette_ready(&job.vga, &job.palette) {
                job.drawn = true;
            }
            return PresentOwner::SpawnIntroTransition;
        }
        if self.native_turn_staging.as_ref().is_some_and(|j| j.indexed) {
            todo!("R8: main.go:9745 drawNativeTurnStaging（indexed；PAN 階段先合成整幀，native_turn_staging.go:486）")
        }
        if self.native_field_event61.is_some() {
            todo!("R8: main.go:9755 drawNativeFieldEvent61")
        }
        if self.native_end_turn_recovery.is_some() {
            // Go `native_end_turn_recovery.go:162`；畫不出來時 Draw 呼叫 cancelNativeEndTurnRecovery（main.go:9768）。
            let palette_ok =
                self.native_map_dac.len() == 256 * 3 && fdother::vga_palette_from_dac(&self.native_map_dac).is_ok();
            let job = self.native_end_turn_recovery.as_mut().unwrap();
            let ok = palette_ok
                && job.frame >= 0
                && (job.frame as usize) < job.frames.len()
                && job.frames[job.frame as usize].len() as i64 == indexedmap::NATIVE_MAP_VGA_SIZE;
            if ok {
                job.drawn = true;
            } else {
                self.cancel_native_end_turn_recovery(&game_err!("native END recovery: Draw unavailable"));
            }
            return PresentOwner::NativeEndTurnRecovery;
        }
        if self.native_ai_idle_recovery.is_some() {
            // Go `native_ai_idle_recovery.go:328`；失敗時 main.go:9779..9783。
            let assets_ok = self.native_map_assets.as_ref().is_some_and(|a| a.palette.len() == 256);
            let job = self.native_ai_idle_recovery.as_mut().unwrap();
            let ok = assets_ok
                && job.frame >= 0
                && (job.frame as usize) < job.frames.len()
                && job.frames[job.frame as usize].len() as i64 == indexedmap::NATIVE_MAP_VGA_SIZE;
            if ok {
                job.drawn = true;
            } else {
                let job = self.native_ai_idle_recovery.take().unwrap();
                let st = self.st.as_mut().unwrap();
                st.native_map_range_mode = job.before_range;
                st.has_native_map_range_mode_state = job.before_has;
                self.load_err = "native AI 0x13fd4 presentation unavailable".to_string();
                self.ai_busy = false;
            }
            return PresentOwner::NativeAIIdleRecovery;
        }
        if self.atk.is_some() {
            self.acknowledge_battle_scene();
            return PresentOwner::AttackScene;
        }
        // Go `main.go:9799`..`10188`：尾段的狀態改寫與合成交錯，照 `Draw` 執行同一份程式（R5 階段 3），像素丟棄。
        let mut scratch = Picture::new_image(LOGICAL_W, LOGICAL_H);
        self.draw_field(&mut scratch);
        PresentOwner::Map
    }

    /// Go `main.go:11231` drawBattleScene 的流程部分：先走 `drawNativePhysicalPrelude`
    /// （`native_physical_scene.go:372`），沒有原生場景時是重製端近似畫面（R8，與 `draw_battle_scene` 相同）。
    fn acknowledge_battle_scene(&mut self) {
        let wanted = self.modern_hd_battle_wanted(self.atk.as_ref().and_then(|a| a.native_scene.as_ref()));
        let Some(scene) = self.atk.as_mut().and_then(|a| a.native_scene.as_mut()) else {
            todo!(
                "R8: main.go:11236 沒有原生場景的攻擊畫面（drawNativePhysicalBase 失敗寫 loadErr；R5 r5b 依階段 0 覆蓋率改標，開場與第一章沒有走到）"
            )
        };
        if scene.prelude_frame < scene.prelude_images.len() as i64 {
            if wanted {
                todo!("R6: native_physical_scene.go:379 drawModernHDBattlePrelude")
            }
            scene.prelude_drawn = true;
            return;
        }
        if scene.lead_frame < scene.lead_images.len() as i64 {
            if scene.body.is_none() {
                if wanted {
                    todo!("R6: native_physical_scene.go:396 HD 出發與捲動")
                }
                scene.lead_drawn = true;
                return;
            }
        } else if scene.body.is_none() {
            todo!(
                "R8: main.go:11236 原生前奏與本體之後的攻擊畫面（drawNativePhysicalBase；R5 r5b 依階段 0 覆蓋率改標，開場與第一章沒有走到）"
            )
        }
        if let Err(err) = self.acknowledge_native_physical_body(wanted) {
            self.load_err = format!("native physical body: {err}");
        }
    }

    /// Go `native_physical_body.go:414` drawNativePhysicalBody 的流程部分：設 `p.drawn`，並照 Go 建立快取畫面
    /// `p.image`（`step` 依它判斷是否要換幀）。cue-only 項目（像素為空）不設 drawn。
    fn acknowledge_native_physical_body(&mut self, hd_wanted: bool) -> Result<()> {
        let ui_palette = self.native_ui_palette.clone();
        let scene = self.atk.as_mut().and_then(|a| a.native_scene.as_mut()).unwrap();
        let p = scene.body.as_mut().unwrap();
        let mut index = p.index;
        if index == p.jobs.len() as i64 && p.return_wait_millis > 0.0 {
            index = p.last_present; // caller等待保留最後影格，不重播cue或新增present。
        }
        if index < 0 || index >= p.jobs.len() as i64 {
            return Err(game_err!("native physical body has no current present"));
        }
        let job = &p.jobs[index as usize];
        if hd_wanted && job.hd_sources.is_some() {
            todo!("R6: native_physical_body.go:426 drawModernHDBattleBody")
        }
        if p.image.is_none() {
            // R5 r5b：與 `draw_native_physical_body` 共用 `present_image`（palette0 照 Go 預乘、像素照 `copy` 補足 320×200）。
            let Some(img) = job.present_image(&ui_palette)? else {
                return Ok(()); // cue-only項目由Update消費。
            };
            p.image = Some(Arc::new(img));
        }
        p.drawn = true;
        Ok(())
    }
}

#[cfg(test)]
#[path = "present/present_test.rs"]
mod present_test;
