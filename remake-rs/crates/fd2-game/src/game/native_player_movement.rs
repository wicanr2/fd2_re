//! Go 來源：`remake/cmd/fd2/native_player_movement.go`（175 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// Go：`native_player_movement.go:11`，`func (g *Game) prepareNativePlayerMovement(u *battle.Unit) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn prepare_native_player_movement(&mut self, u: UnitIdx) -> Result<()> {
        let ready = self.st.as_ref().is_some_and(|st| st.native_tile_blit_modes.len() as i64 == st.w * st.h);
        if !ready || self.native_ui_palette.len() != 256 {
            return Err(game_err!("native player movement: renderer provenance unavailable"));
        }
        let plan = self.st.as_ref().unwrap().plan_native_player_movement(Some(u))?;
        let assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let record = battle::native_battle_panel_record_for_unit(Some(&self.st.as_ref().unwrap().units[u]))?;
        let mut pixels = vec![0u8; 320 * 200];
        let mut collected: Vec<ModernBattlePanelText> = Vec::new();
        let collect = self.font.as_ref().is_some_and(|f| f.sf.is_some()) && self.locale_entities.is_some();
        self.render_battle_panel_text(&assets, &record, &mut pixels, 0, 0, collect.then_some(&mut collected))?;
        let (x, y) = battle::native_battle_panel_origin(&record, 0, 0)?;
        let mut panel = indexed_frame(149, 42, &vec![0u8; 149 * 42], self.native_ui_palette.clone());
        for row in 0..42i64 {
            let src = ((y + row) * 320 + x) as usize;
            panel.pix[(row * 149) as usize..(row * 149 + 149) as usize].copy_from_slice(&pixels[src..src + 149]);
        }
        let source_image = Arc::new(picture_from_indexed(&panel));
        let mut panel_x = 9;
        if self.st.as_ref().unwrap().native_map_view_state.visible_cursor_x < 7 {
            panel_x = 160;
        }
        let hd = self.freeze_modern_hd_movement_panel(
            Some(u),
            Some(&plan),
            Some(&source_image),
            &record,
            &panel.pix,
            panel_x,
            &collected,
        )?;
        if self.modern_hd_interface_enabled() && hd.is_none() {
            return Err(game_err!("modern HD movement panel: verified name/font unavailable"));
        }
        (self.native_move_panel, self.native_move_panel_x, self.movement_panel_hd) = (Some(source_image), panel_x, hd);
        self.reach = plan.reach.clone();
        go_copy(&mut self.st.as_mut().unwrap().native_tile_blit_modes, &plan.field);
        self.native_move_plan = Some(plan);
        Ok(())
    }

    /// Go：`native_player_movement.go:64`，`func (g *Game) clearNativePlayerMovement()`
    pub(crate) fn clear_native_player_movement(&mut self) {
        if self.native_move_plan.is_some() {
            self.reset_native_target_field();
        }
        (self.native_move_plan, self.native_move_panel) = (None, None);
        (self.movement_panel_hd, self.movement_panel_draw) = (None, None);
    }

    /// Go：`native_player_movement.go:72`，`func (g *Game) restorePlayerMovementCursor()`（`r8`，隨 `ringInput` 的
    /// ESC 分支補移植）。
    pub(crate) fn restore_player_movement_cursor(&mut self) {
        if let Some(st) = self.st.as_mut().filter(|st| st.has_native_map_view_state) {
            while st.native_map_view_state.cursor_x != self.sel_orig_x {
                let delta = if st.native_map_view_state.cursor_x > self.sel_orig_x { -1 } else { 1 };
                let (moved, ok) = st.move_native_map_cursor(delta, 0);
                if !ok || !moved {
                    break;
                }
            }
            while st.native_map_view_state.cursor_y != self.sel_orig_y {
                let delta = if st.native_map_view_state.cursor_y > self.sel_orig_y { -1 } else { 1 };
                let (moved, ok) = st.move_native_map_cursor(0, delta);
                if !ok || !moved {
                    break;
                }
            }
            self.sync_native_map_view();
            return;
        }
        (self.cur_x, self.cur_y) = (self.sel_orig_x, self.sel_orig_y);
    }

    /// beginPlayerAttackTargeting 是指令環攻擊收合後的共同入口（0x18f22 分支）：
    /// 標記射程、提示選目標。重播測試也走這裡，不另抄一份。
    ///
    /// Go：`native_player_movement.go:122`，`func (g *Game) beginPlayerAttackTargeting()`
    pub(crate) fn begin_player_attack_targeting(&mut self) {
        self.mark_native_player_attack_field();
        if let (message, true) = self.locale_message("battle.attack.choose_target", &[]) {
            self.msg = message;
        }
    }

    /// 0x18F76以targetCode0進0x115B6。未通過確認仍留在目標選擇，不能提交待機。
    ///
    /// Go：`native_player_movement.go:130`，`func (g *Game) nativePhysicalAttackConfirmationAllowed() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_physical_attack_confirmation_allowed(&mut self) -> bool {
        let Some(st) = self.st.as_ref() else {
            return false;
        };
        if self.sel.is_none()
            || self.cur_x < 0
            || self.cur_x >= st.w
            || self.cur_y < 0
            || self.cur_y >= st.h
            || st.native_tile_blit_modes.len() as i64 != st.w * st.h
        {
            return false;
        }
        let result = battle::native_cursor_confirmation_allowed(
            battle::Cell { x: self.cur_x, y: self.cur_y },
            st.native_tile_blit_modes[(self.cur_y * st.w + self.cur_x) as usize],
            st.native_map_range_mode,
            0,
            &st.units,
        );
        match result {
            Ok(allowed) => allowed,
            Err(err) => {
                self.load_err = err.message().to_string();
                false
            }
        }
    }

    /// markNativePlayerAttackField 在指令環選攻擊收合後，把 0x18f6a 那次 0x14818 的
    /// 標記寫進 NativeTileBlitModes；選目標期間整幀就會照原版染出武器射程。舊版
    /// JSON 戰場沒有原版物品表時維持不標記。
    ///
    /// Go：`native_player_movement.go:165`，`func (g *Game) markNativePlayerAttackField()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn mark_native_player_attack_field(&mut self) {
        let Some(st) = self.st.as_ref() else {
            return;
        };
        let Some(sel) = self.sel else {
            return;
        };
        if st.native_tile_blit_modes.len() as i64 != st.w * st.h {
            return;
        }
        let Ok(field) = st.native_player_attack_target_field(Some(sel)) else {
            return;
        };
        go_copy(&mut self.st.as_mut().unwrap().native_tile_blit_modes, &field);
    }
}

/// Go 內建 `copy(dst, src)`：複製兩者較短的長度，回傳複製的元素數。
pub(crate) fn go_copy<T: Copy>(dst: &mut [T], src: &[T]) -> usize {
    let n = dst.len().min(src.len());
    dst[..n].copy_from_slice(&src[..n]);
    n
}

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_player_movement.go:100`，`func (g *Game) drawNativeMovementPanel(screen *ebiten.Image)`
    pub(crate) fn draw_native_movement_panel(&mut self, screen: &mut Picture) {
        todo!("r5:native_player_movement.go:drawNativeMovementPanel")
    }
}
