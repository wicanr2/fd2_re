//! Go 來源：`remake/cmd/fd2/native_map_unit_window.go`（126 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// nativeMapSelectionOverlay 決定這一幀比 0x11CAC 多畫什麼：
    ///   - 玩家選了自己還沒行動的單位、正在挑目的地（移動範圍亮著、還沒移動、指令環
    ///     沒開）：0x18B84 的重繪，多一個單位資訊視窗（0x18C6D）；
    ///   - 指令環開著：0x1741C 把四張 FDOTHER#2 圖示貼在游標四周（穩態用 0x179D5 的
    ///     停留位移，開合中用各自的動畫位移），被選方向依 [0x53C13] 閃爍。
    ///
    /// 回傳 nil 表示照 0x11CAC。
    ///
    /// Go：`native_map_unit_window.go:26`，`func (g *Game) nativeMapSelectionOverlay(state *battle.State) (func([]byte, int, int) error, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// 簽名（C 組）：Go 回傳捕捉 `g`、`state` 的閉包 `func([]byte, int, int) error`。Rust 回傳
    /// [`NativeMapSelectionOverlay`]，記下 Go 閉包在建立時捕捉的值；合成時以
    /// [`NativeMapSelectionOverlay::render`] 借用同一個 `Game` 與 `state` 畫進工作緩衝，組成
    /// `indexedmap::SelectionOverlay`。
    pub(crate) fn native_map_selection_overlay(
        &mut self,
        state: Option<&battle::State>,
    ) -> Result<Option<NativeMapSelectionOverlay>> {
        let Some(state) = state.filter(|st| st.has_native_map_view_state) else {
            return Ok(None);
        };
        if self.ring
            && self.sel.is_some()
            && self.walk.is_none()
            && self.native_action_cells_raw.len() as i64 == NATIVE_ACTION_OVERLAY_CELL_COUNT
        {
            return Ok(Some(self.native_map_action_overlay(state)));
        }
        // Go 讀 `g.sel.Camp`；`g.sel` 指向 `g.st.Units`（不是傳進來的候選狀態）。
        let sel_unit = self.sel.and_then(|sel| self.st.as_ref().map(|st| &st.units[sel]));
        let Some(sel_unit) = sel_unit else {
            return Ok(None);
        };
        if self.moved
            || self.ring
            || self.walk.is_some()
            || self.cast_sp.is_some()
            || self.native_command0_targeting
            || sel_unit.camp != battle::OWN
        {
            return Ok(None);
        }
        // Go `g.reach == nil`：Rust 的 `reach` 不分 nil 與空 map（見 PORTING.md「C 組 KNOWN_DIFFERENCES」）。
        if self.reach.is_empty() {
            return Ok(None);
        }
        let record = battle::native_item_panel_record_for_unit(Some(sel_unit))
            .map_err(|e| GameError::from(e).wrap("native map unit window: "));
        if self.native_map_unit_window_assets.is_none() {
            let assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))
                .map_err(|e| GameError::from(e).wrap("native map unit window: "))?;
            self.native_map_unit_window_assets = Some(assets);
        }
        let record = record?;
        let visible_x = state.native_map_view_state.visible_cursor_x;
        Ok(Some(NativeMapSelectionOverlay::UnitWindow { record, visible_x }))
    }

    /// nativeMapActionOverlay 把指令環貼進工作緩衝，與 drawNativeActionOverlay 同一組
    /// 位移、格子索引與閃爍規則，只是目的地換成 indexed 畫面。
    ///
    /// Go：`native_map_unit_window.go:58`，`func (g *Game) nativeMapActionOverlay(state *battle.State) func([]byte, int, int) error`
    ///
    /// 簽名（C 組）：同 [`Game::native_map_selection_overlay`]，回傳記錄捕捉值的
    /// [`NativeMapSelectionOverlay::Action`]；Go 在建立時就失敗的閉包是 [`NativeMapSelectionOverlay::Failed`]。
    pub(crate) fn native_map_action_overlay(&mut self, state: &battle::State) -> NativeMapSelectionOverlay {
        let overlay = self.native_action_overlay_state();
        let (frame, closing) = self.action_overlay_render_state();
        let mut offsets = match fdother::action_overlay_frame_offsets(frame, closing) {
            Ok(offsets) => offsets,
            Err(err) => return NativeMapSelectionOverlay::Failed(battle::BattleError::from(err)),
        };
        let mut steady = !self.action_overlay_blocks_input();
        if steady && self.native_action_overlay_open_frame_variant {
            offsets = match fdother::action_overlay_frame_offsets(3, false) {
                Ok(offsets) => offsets,
                Err(err) => return NativeMapSelectionOverlay::Failed(battle::BattleError::from(err)),
            };
            steady = false;
        } else if steady {
            offsets = [-0x23a0, 0x378, 0x3a8, 0x2ac0];
        }
        NativeMapSelectionOverlay::Action(NativeMapActionOverlay {
            overlay,
            offsets,
            steady,
            closing,
            view: state.native_map_view_state,
            ring_sel: self.ring_sel,
            blink: self.action_overlay_blink.phase,
        })
    }
}

/// Rust 新增：Go `nativeMapSelectionOverlay` 回傳的閉包在建立時捕捉的值（見該函式的簽名說明）。
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum NativeMapSelectionOverlay {
    /// 0x18B84 的單位資訊視窗（0x18C6D）：Go 閉包捕捉 `assets`（`g.nativeMapUnitWindowAssets`）、`record`
    /// 與 `visibleX`。
    UnitWindow { record: Vec<u8>, visible_x: i64 },
    /// 0x1741C 的指令環。
    Action(NativeMapActionOverlay),
    /// Go `nativeMapActionOverlay` 在建立閉包前就失敗時，回傳只會回報該錯誤的閉包。
    Failed(battle::BattleError),
}

/// Rust 新增：Go `nativeMapActionOverlay` 閉包捕捉的值。`cells`（`g.nativeActionCellsRaw`）、
/// `state.Units` 與 `g.nativeMapAssets` 在 Go 也是建立後到呼叫前不變的同一份，Rust 在呼叫時才借用。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct NativeMapActionOverlay {
    pub(crate) overlay: fdother::ActionOverlayState,
    pub(crate) offsets: [i64; 4],
    pub(crate) steady: bool,
    pub(crate) closing: bool,
    pub(crate) view: battle::NativeMapViewState,
    pub(crate) ring_sel: i64,
    pub(crate) blink: i64,
}

impl NativeMapSelectionOverlay {
    /// Go 閉包本體：`func(frame []byte, stride, viewportBase int) error`。`g` 與 `state` 是建立閉包時的
    /// 同一個 `Game` 與候選狀態。
    pub(crate) fn render(
        &self,
        g: &Game,
        state: &battle::State,
        buf: &mut [u8],
        stride: i64,
        viewport_base: i64,
    ) -> std::result::Result<(), battle::BattleError> {
        match self {
            NativeMapSelectionOverlay::UnitWindow { record, visible_x } => {
                let assets = g.native_map_unit_window_assets.as_ref().expect("Go：assets 在建立閉包時已載入");
                battle::render_native_map_unit_window(
                    assets,
                    record,
                    &mut buf[viewport_base as usize..],
                    stride,
                    *visible_x,
                )
            }
            NativeMapSelectionOverlay::Failed(err) => Err(err.clone()),
            NativeMapSelectionOverlay::Action(a) => a.render(g, state, buf, stride),
        }
    }
}

impl NativeMapActionOverlay {
    fn render(
        &self,
        g: &Game,
        state: &battle::State,
        buf: &mut [u8],
        stride: i64,
    ) -> std::result::Result<(), battle::BattleError> {
        let view = self.view;
        let origin = fdother::action_overlay_origin(view.visible_cursor_x, view.visible_cursor_y)?;
        let cells = &g.native_action_cells_raw;
        for (direction, &offset) in self.offsets.iter().enumerate() {
            let direction = direction as i64;
            let mut index = self.overlay.cell_index(direction);
            if self.steady
                && direction == self.ring_sel
                && fdother::action_overlay_accepts_direction(self.overlay.availability, direction)
            {
                index = self.overlay.selected_cell_index(direction, self.blink);
            }
            let index = index?;
            if index < 0 || index >= cells.len() as i64 {
                return Err(battle::BattleError::new(format!("native action overlay cell {index} is absent")));
            }
            let pos = origin + offset;
            if pos < 0 || pos >= buf.len() as i64 {
                return Err(battle::BattleError::new(format!("native action overlay position {pos} is invalid")));
            }
            cells[index as usize].blit_at(buf, stride, pos % stride, pos / stride)?;
        }
        if !self.closing {
            // sub_179D5 (0x17A9C) and sub_1741C (0x17544) redraw the
            // first active raw unit at the cursor after all four icons.
            // Contract: fd2_parity_ring_open_20261005.json, issue #180.
            for unit in &state.units {
                let (entry, ok) = unit.native_unit_layer_entry();
                if !ok {
                    return Err(battle::BattleError::new(
                        "native action overlay: cursor unit provenance is incomplete",
                    ));
                }
                if entry.inactive || entry.x != view.cursor_x || entry.y != view.cursor_y {
                    continue;
                }
                let Some(bank) = g.native_map_assets.as_ref().and_then(|a| a.units.as_ref()) else {
                    return Err(battle::BattleError::new("native action overlay: cursor unit bank is absent"));
                };
                let cycles = state.native_map_cycle_state;
                // Go 把 nil 的 `*NativeSelectorCache` 交給 BlitNativeUnitLayer，由它的開頭檢查回報；Rust 的參數不收
                // nil，這裡回傳同一段錯誤文字（`fdicon/unit_layer.go:41`）。
                let Some(cache) = state.native_map_selector_cache.as_ref() else {
                    return Err(battle::BattleError::new("fdicon: invalid native unit layer"));
                };
                return Ok(bank.blit_native_unit_layer(
                    buf,
                    stride,
                    cache,
                    &[entry],
                    view.camera_x,
                    view.camera_y,
                    12,
                    7,
                    cycles.idle,
                    cycles.moving,
                    state.native_unit_pixel_shift_state.value,
                )?);
            }
        }
        Ok(())
    }
}

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// #181 uses this Draw's admission, matching nativeMapSelectionOverlay's ring
    /// branch. Stale buffers cannot witness presentation. Modern themes retain their
    /// catalog icons. Contract: fd2_parity_ring_open_20261005.json/draw_layer_spec.
    ///
    /// Go：`native_map_unit_window.go:14`，`func (g *Game) nativeActionOverlayInMapFrame(presented bool) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_action_overlay_in_map_frame(&mut self, presented: bool) -> bool {
        todo!("r5:native_map_unit_window.go:nativeActionOverlayInMapFrame")
    }
}
