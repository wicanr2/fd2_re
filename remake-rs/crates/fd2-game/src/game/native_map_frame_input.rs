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
pub(crate) fn build_native_map_frame_input(
    assets: &mut NativeMapAssets,
    field: Option<&MapData>,
    state: Option<&mut battle::State>,
    runtime: NativeMapFrameRuntime,
) -> Result<indexedmap::NativeFrameInput<'static>> {
    todo!("native_map_frame_input.go:buildNativeMapFrameInput")
}

impl Game {
    /// nativeCh12UpWalkFrame limits the recovered 0x13185 geometry to the reviewed
    /// player caller. Other chapters, AI, directions and story owners stay separate.
    ///
    /// Go：`native_map_frame_input.go:93`，`func (g *Game) nativeCh12UpWalkFrame() (unitY, motion int, ok bool)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ch12_up_walk_frame(&self) -> (i64, i64, bool) {
        todo!("native_map_frame_input.go:nativeCh12UpWalkFrame")
    }
}
