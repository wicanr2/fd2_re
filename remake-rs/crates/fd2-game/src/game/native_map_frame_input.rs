//! Go 來源：`remake/cmd/fd2/native_map_frame_input.go`（107 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeMapFrameRuntime is the still-caller-owned raw global subset needed by
/// one steady 0x11cac frame. It deliberately has no conversion from the
/// remake's 640x400 pixel camera or normalized selection/highlight state.
///
/// Go：`native_map_frame_input.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeMapFrameRuntime {
    pub(crate) hud: indexedmap::NativeMapHUDInput,
    pub(crate) chapter_aux_phase: i64,
}

/// buildNativeMapFrameInput joins the all-or-nothing original asset bundle,
/// exact exported FDFIELD cells, battle-local raw roster and explicit runtime
/// globals. Any incomplete provenance rejects the entire frame input.
///
/// Go：`native_map_frame_input.go:24`，`func buildNativeMapFrameInput( assets *nativeMapAssets, field *MapData, state *battle.State, runtime nativeMapFrameRuntime, ) (indexedmap.NativeFrameInput, error)`
///
/// 簽名（C 組）：回傳的輸入借用 `assets` 與 `state`（`fd2-game` 保存資源本身，每次呼叫重組借用的輸入）；
/// Go 的三個指標參數都會檢查 nil，所以都是 `Option`，且都不改寫。
pub(crate) fn build_native_map_frame_input<'a>(
    assets: Option<&'a NativeMapAssets>,
    field: Option<&MapData>,
    state: Option<&'a battle::State>,
    runtime: NativeMapFrameRuntime,
) -> Result<indexedmap::NativeFrameInput<'a>> {
    let (Some(assets), Some(field), Some(state)) =
        (assets.filter(|a| native_map_assets_available(Some(a))), field, state)
    else {
        return Err(game_err!("native map frame: incomplete assets, field, or battle state"));
    };
    if !state.has_native_map_hud_state {
        return Err(game_err!("native map frame: HUD runtime state is not materialized"));
    }
    if field.w <= 0
        || field.h <= 0
        || field.tiles.len() as i64 != field.w * field.h
        || field.native_terrain_control != assets.controls
    {
        return Err(game_err!("native map frame: editable field does not match native FDSHAP controls"));
    }
    if !state.has_native_map_view_state
        || !state.has_native_map_range_mode_state
        || state.native_map_range_mode < 0
        || state.native_map_range_mode > 5
    {
        return Err(game_err!("native map frame: raw runtime globals are outside verified bounds"));
    }
    let view = state.native_map_view_state;
    // 撿走寶箱那一格的圖塊由 0x12263 就地 +1，繪圖端跟著讀那份可變緩衝；可編輯地圖的
    // field.Tiles 是初始值，直接拿去畫會讓箱子永遠關著。
    let (mutable, ok) = state.native_map_draw_tiles();
    let draw_tiles: &[i64] = if ok {
        if mutable.len() != field.tiles.len() {
            return Err(game_err!("native map frame: mutable map buffer does not match the editable field"));
        }
        &mutable
    } else {
        &field.tiles
    };
    let cells = indexedmap::build_native_terrain_cells(draw_tiles, &state.native_tile_blit_modes)
        .map_err(|e| GameError::from(e).wrap("native map frame: terrain cells: "))?;
    let roster = state.native_map_frame_roster().map_err(|e| GameError::from(e).wrap("native map frame: roster: "))?;
    let lut_index = match fdother::native_terrain_lut_index(roster.terrain_phase) {
        Ok(i) if i >= 0 && (i as usize) < assets.luts.len() && assets.luts[i as usize].len() == 256 => i as usize,
        _ => return Err(game_err!("native map frame: terrain phase LUT is incomplete")),
    };
    let frame = indexedmap::FrameInput {
        terrain_bank: assets.terrain.as_ref(),
        range_bank: assets.range.as_ref(),
        unit_bank: assets.units.as_ref(),
        foreground_bank: assets.terrain.as_ref(),
        selector_cache: state.native_map_selector_cache.as_ref(),
        cells,
        controls: &assets.controls,
        lut: &assets.luts[lut_index],
        map_width: field.w,
        camera_x: view.camera_x,
        camera_y: view.camera_y,
        flip: roster.terrain_flip,
        terrain_cycle: roster.cycles.idle,
        idle_cycle: roster.cycles.idle,
        moving_cycle: roster.cycles.moving,
        pixel_shift: roster.unit_pixel_shift,
        range_mode: state.native_map_range_mode,
        cursor_x: view.cursor_x,
        cursor_y: view.cursor_y,
        units: roster.units,
        foreground_units: roster.foreground,
        parallax: assets.parallax.as_ref(),
        chapter_aux: assets.chapter_aux.as_ref(),
        chapter_aux_phase: runtime.chapter_aux_phase,
        ..Default::default()
    };
    let mut hud = runtime.hud;
    hud.display_gate_a = state.native_map_hud_state.display_gate_a != 0;
    hud.display_gate_b = state.native_map_hud_state.display_gate_b != 0;
    hud.anchor_x = state.native_map_hud_state.anchor_x;
    Ok(indexedmap::NativeFrameInput {
        frame,
        hud,
        frames: assets.frames.clone(),
        hud_terrain: assets.terrain.as_ref(),
        hud_units: assets.units.as_ref(),
        hud_cache: state.native_map_selector_cache.as_ref(),
    })
}

impl Game {
    /// nativeCh12UpWalkFrame limits the recovered 0x13185 geometry to the reviewed
    /// player caller. Other chapters, AI, directions and story owners stay separate.
    ///
    /// Go：`native_map_frame_input.go:93`，`func (g *Game) nativeCh12UpWalkFrame() (unitY, motion int, ok bool)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ch12_up_walk_frame(&self) -> (i64, i64, bool) {
        let ok = self.st.as_ref().is_some_and(|st| st.has_native_map_view_state)
            && self.camp.as_ref().is_some_and(|camp| camp.node_id() == "battle_ch12")
            && !self.has_story_native_map_view
            && self.native_map_assets.as_ref().is_some_and(|a| a.map_index == 11);
        let Some(walk) = self.walk.as_ref().filter(|_| ok) else {
            return (0, 0, false);
        };
        if walk.u.is_none()
            || walk.then.is_some()
            || walk.tick < 1
            || walk.tick > NATIVE_MAP_GRID_MOTION_FRAMES
            || walk.seg < 0
            || walk.seg + 1 >= walk.path.len() as i64
        {
            return (0, 0, false);
        }
        let (a, b) = (walk.path[walk.seg as usize], walk.path[walk.seg as usize + 1]);
        if b.x != a.x || b.y != a.y - 1 {
            return (0, 0, false);
        }
        (a.y, walk.tick, true)
    }
}

#[cfg(test)]
#[path = "native_map_frame_input/native_map_frame_input_test.rs"]
pub(crate) mod native_map_frame_input_test;
