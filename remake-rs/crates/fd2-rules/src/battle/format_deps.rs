//! Go 版 `battle` 用到的格式層項目清單。沒有對應的 Go 檔。
//!
//! 每一項都在編譯期確認 `fd2-formats` 已有公開的 Rust 對應；階段 2 移植時直接用這些路徑。
//! 左欄註解是 Go 名稱（2026-11-14 以 `go/types` 從 `remake/internal/battle` 正式程式列出）。
//! 形狀和 Go 不同、需要呼叫端轉換的項目另見 `PORTING.md`「格式層缺口」。

#[rustfmt::skip]
#[allow(unused_imports)]
use fd2_formats::{
    // dato.Frame、dato.Frame.BlitAt、dato.LoadSeparatedResource
    dato::{Frame as DatoFrame, load_separated_resource as dato_load_separated_resource},
    // fdicon.NativeBinaryTickState、AdvanceNativeBinaryTick
    fdicon::{NativeBinaryTickState, advance_native_binary_tick},
    // fdicon.NativeMapSpriteCycleState、AdvanceNativeMapSpriteCycles
    fdicon::{NativeMapSpriteCycleState, advance_native_map_sprite_cycles},
    // fdicon.NativeForegroundLayerEntry、NativeUnitLayerEntry、NativeMapStride、NativeMapHUDLayoutFor
    fdicon::{NATIVE_MAP_STRIDE, NativeForegroundLayerEntry, NativeMapHUDLayout, NativeUnitLayerEntry, native_map_hud_layout_for},
    // fdicon.NativeSelectorCache（SlotFor／KeyForSlot／Keys／Clone）、NewNativeSelectorCacheFromKeys → NativeSelectorCache::from_keys
    fdicon::NativeSelectorCache,
    // fdicon.Sprite、Sprite.BlitAt
    fdicon::Sprite,
    // fdother.Frame、Frame.BlitAt
    fdother::Frame as FdotherFrame,
    // fdother.LMI1Entry、ParseLMI1、ParseLMI1FrameEntry、ParseLMI1RawEntry、ParseLMI1OpaqueEntry、LMI1Entry.BlitOpaqueAt
    fdother::{LMI1Entry, parse_lmi1, parse_lmi1_frame_entry, parse_lmi1_opaque_entry, parse_lmi1_raw_entry},
    // fdother.RawCell、RawCell.BlitOpaqueAtOffset
    fdother::RawCell,
    // fdother.NativeRNGStep
    fdother::native_rng_step,
    // fdother.ReadResource（Rust 的資源索引是 usize）
    fdother::read_resource,
    // fdother.NativeTerrainPhaseState、AdvanceNativeTerrainPhase
    fdother::{NativeTerrainPhaseState, advance_native_terrain_phase},
    // fdother.NativePhasePreselectionScores、NativePhaseUnitScanPlan、PlanNativePhaseUnitScans
    fdother::{NativePhasePreselectionScores, NativePhaseUnitScanPlan, plan_native_phase_unit_scans},
    // fdother.PlanNativeUnitMode
    fdother::plan_native_unit_mode,
    // fdother.PlanNativeDialogueFrameGrid
    fdother::plan_native_dialogue_frame_grid,
    // fdother.LoadSeparatedItemPanelEntries
    fdother::load_separated_item_panel_entries,
    // fdother.NativeCh28PostPoseFrames、NativeCh28PostOverlayOrigin
    fdother::{NATIVE_CH28_POST_POSE_FRAMES, native_ch28_post_overlay_origin},
    // fdother.NativeBattleEntryStep
    fdother::native_battle_entry_step,
    // fdtxt.Strings（Words）、Font（BlitNativeGlyph）、NativeGlyphStyle、ControlMin
    fdtxt::{CONTROL_MIN, Font, NativeGlyphStyle, Strings},
    // fdtxt.Parse、ParseFont、LoadSeparatedResource、LoadSeparatedFont
    fdtxt::{load_separated_font, load_separated_resource as fdtxt_load_separated_resource, parse, parse_font},
};
