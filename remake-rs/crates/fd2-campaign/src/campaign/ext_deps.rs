//! `campaign` 正式程式用到的外部 crate 項目清單。沒有對應的 Go 檔。
//!
//! 每一項都在編譯期確認 `fd2-rules`／`fd2-formats` 已有公開的 Rust 對應；階段 2 移植時直接用
//! 這些路徑。註解是 Go 名稱（2026-11-14 以 `go/types` 從 `remake/internal/campaign` 正式程式列出）。

#[rustfmt::skip]
#[allow(unused_imports)]
mod battle_items {
    use fd2_rules::battle::{
        ALLY, // const battle.Ally
        apply_native_equipment_recalc, // func battle.ApplyNativeEquipmentRecalc
        Camp, // type battle.Camp
        ENEMY, // const battle.Enemy
        load, // func battle.Load
        materialize_native_map_selector_slots, // func battle.MaterializeNativeMapSelectorSlots
        NativeChestControl, // type battle.NativeChestControl
        NativeFacilityPriceMode, // type battle.NativeFacilityPriceMode
        NativeFieldEvent, // type battle.NativeFieldEvent
        NativeFieldEventRule, // type battle.NativeFieldEventRule
        NativeFieldUnitControl, // type battle.NativeFieldUnitControl
        NATIVE_ITEM_EFFECT_ROW_SIZE, // const battle.NativeItemEffectRowSize
        NativeItemPanelDataAssets, // type battle.NativeItemPanelDataAssets
        native_item_panel_record_for_unit, // func battle.NativeItemPanelRecordForUnit
        NativeMapPresentationState, // type battle.NativeMapPresentationState
        NativeMapViewState, // type battle.NativeMapViewState
        NativeRuntimeRecordState, // type battle.NativeRuntimeRecordState
        NativeTurnEventControl, // type battle.NativeTurnEventControl
        OWN, // const battle.Own
        remove_native_compact_inventory, // func battle.RemoveNativeCompactInventory
        render_native_facility_item_rows, // func battle.RenderNativeFacilityItemRows
        render_native_facility_item_rows_without_names, // func battle.RenderNativeFacilityItemRowsWithoutNames
        render_native_facility_number, // func battle.RenderNativeFacilityNumber
        Scenario, // type battle.Scenario
        State, // type battle.State
        Unit, // type battle.Unit
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod dato_items {
    use fd2_formats::dato::{
        Frame, // type dato.Frame
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod fdicon_items {
    use fd2_formats::fdicon::{
        Bank, // type fdicon.Bank
        decode_file, // func fdicon.DecodeFile
        load_separated_bank, // func fdicon.LoadSeparatedBank
        NativeBinaryTickState, // type fdicon.NativeBinaryTickState
        NativeMapSpriteCycleState, // type fdicon.NativeMapSpriteCycleState
        NativeSelectorCache, // type fdicon.NativeSelectorCache
        SEPARATED_SPRITE_COUNT, // const fdicon.SeparatedSpriteCount
        Sprite, // type fdicon.Sprite
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod fdother_items {
    use fd2_formats::fdother::{
        Frame, // type fdother.Frame
        LMI1Entry, // type fdother.LMI1Entry
        load_separated_single_frame, // func fdother.LoadSeparatedSingleFrame
        NATIVE_MAP_VIEWPORT_COLUMNS, // const fdother.NativeMapViewportColumns
        NATIVE_MAP_VIEWPORT_ROWS, // const fdother.NativeMapViewportRows
        NATIVE_SYSTEM_INFO_BYTES, // const fdother.NativeSystemInfoBytes
        NativeTerrainPhaseState, // type fdother.NativeTerrainPhaseState
        parse_opaque_run_cell, // func fdother.ParseOpaqueRunCell
        parse_raw_cell, // func fdother.ParseRawCell
        parse_single_frame, // func fdother.ParseSingleFrame
        plan_native_dialogue_frame_grid, // func fdother.PlanNativeDialogueFrameGrid
        RawCell, // type fdother.RawCell
        read_resource, // func fdother.ReadResource
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod fdtxt_items {
    use fd2_formats::fdtxt::{
        CONTROL_MIN, // const fdtxt.ControlMin
        Font, // type fdtxt.Font
        GLYPH_WIDTH, // const fdtxt.GlyphWidth
        NativeGlyphStyle, // type fdtxt.NativeGlyphStyle
        Strings, // type fdtxt.Strings
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod save_items {
    use fd2_formats::save::{
        ChapterSlotSnapshot, // type fdsave.ChapterSlotSnapshot
        ContinueRuntimeInput, // type fdsave.ContinueRuntimeInput
        ContinueRuntimeRecord, // type fdsave.ContinueRuntimeRecord
        CurrentSnapshot, // type fdsave.CurrentSnapshot
        PersistentRecord, // type fdsave.PersistentRecord
        ROSTER_SIZE, // const fdsave.RosterSize
        ROSTER_UNITS, // const fdsave.RosterUnits
        Slot, // type fdsave.Slot
        SLOT_COUNT, // const fdsave.SlotCount
    };
}

// 方法（掃描 Rust 原始碼時確認存在；`use` 無法引入方法）：
// - battle.State.BindNativeFutureItemRows -> battle::State::bind_native_future_item_rows
// - battle.State.MaterializeNativeMapHUDState -> battle::State::materialize_native_map_hud_state
// - battle.State.MaterializeNativeMapRangeMode -> battle::State::materialize_native_map_range_mode
// - battle.State.MaterializeNativeMapViewState -> battle::State::materialize_native_map_view_state
// - battle.Unit.AddInventoryItem -> battle::Unit::add_inventory_item
// - battle.Unit.RemoveInventoryIndex -> battle::Unit::remove_inventory_index
// - battle.Unit.RestoreNativeHP -> battle::Unit::restore_native_hp
// - dato.Frame.BlitAtOffset -> dato::Frame::blit_at_offset
// - dato.Frame.BlitRightToLeftAtOffset -> dato::Frame::blit_right_to_left_at_offset
// - fdicon.Sprite.BlitAt -> fdicon::Sprite::blit_at
// - fdother.Frame.Blit -> fdother::Frame::blit
// - fdother.Frame.BlitAt -> fdother::Frame::blit_at
// - fdother.LMI1Entry.BlitAt -> fdother::LMI1Entry::blit_at
// - fdother.LMI1Entry.BlitOpaqueAt -> fdother::LMI1Entry::blit_opaque_at
// - fdother.RawCell.BlitAt -> fdother::RawCell::blit_at
// - fdother.RawCell.BlitAtClipped -> fdother::RawCell::blit_at_clipped
// - fdother.RawCell.BlitOpaqueAtOffset -> fdother::RawCell::blit_opaque_at_offset
// - fdsave.ChapterSlotSnapshot.ActiveRecords -> save::ChapterSlotSnapshot::active_records
// - fdsave.ContinueRuntimeInput.ValidatedForRuntimeBridge -> save::ContinueRuntimeInput::validated_for_runtime_bridge
// - fdsave.PersistentRecord.View -> save::PersistentRecord::view
// - fdtxt.Font.BlitNativeGlyph -> fdtxt::Font::blit_native_glyph
// - fdtxt.Font.GlyphCount -> fdtxt::Font::glyph_count
// - fdtxt.Strings.Words -> fdtxt::Strings::words

// 找不到 Rust 對應（缺口，見 PORTING.md）：

#[cfg(test)]
#[rustfmt::skip]
#[allow(unused_imports)]
mod battle_test_items {
    use fd2_rules::battle::{
        load_native_item_effect_row_prefix, // func battle.LoadNativeItemEffectRowPrefix
        load_native_item_panel_data_assets_archive, // func battle.LoadNativeItemPanelDataAssetsArchive
        load_scenario, // func battle.LoadScenario
        native_ai_scoring_records, // func battle.NativeAIScoringRecords
        NATIVE_FACILITY_FULL_PRICE, // const battle.NativeFacilityFullPrice
        NATIVE_FACILITY_THREE_QUARTER_PRICE, // const battle.NativeFacilityThreeQuarterPrice
    };
}

#[cfg(test)]
#[rustfmt::skip]
#[allow(unused_imports)]
mod dato_test_items {
    use fd2_formats::dato::{
        decode_resource, // func dato.DecodeResource
    };
}

#[cfg(test)]
#[rustfmt::skip]
#[allow(unused_imports)]
mod fdicon_test_items {
    use fd2_formats::fdicon::{
        NATIVE_SIZE, // const fdicon.NativeSize
    };
}

#[cfg(test)]
#[rustfmt::skip]
#[allow(unused_imports)]
mod fdother_test_items {
    use fd2_formats::fdother::{
        decode_raw_cell_resource, // func fdother.DecodeRawCellResource
        parse_lmi1, // func fdother.ParseLMI1
        parse_lmi1_frame_entry, // func fdother.ParseLMI1FrameEntry
        parse_lmi1_opaque_entry, // func fdother.ParseLMI1OpaqueEntry
        parse_lmi1_raw_entry, // func fdother.ParseLMI1RawEntry
        parse_vga_palette, // func fdother.ParseVGAPalette
    };
}

#[cfg(test)]
#[rustfmt::skip]
#[allow(unused_imports)]
mod fdtxt_test_items {
    use fd2_formats::fdtxt::{
        GLYPH_BYTES, // const fdtxt.GlyphBytes
        GLYPH_HEIGHT, // const fdtxt.GlyphHeight
        parse, // func fdtxt.Parse
        parse_font, // func fdtxt.ParseFont
        STRING_END, // const fdtxt.StringEnd
    };
}

#[cfg(test)]
#[rustfmt::skip]
#[allow(unused_imports)]
mod save_test_items {
    use fd2_formats::save::{
        build_continue_runtime_input, // func fdsave.BuildContinueRuntimeInput
        ContinueRuntimeContext, // type fdsave.ContinueRuntimeContext
        CURRENT_FIELD_CONTROL_SIZE, // const fdsave.CurrentFieldControlSize
        CURRENT_FIELD_CONTROL_UNIT_OFFSET, // const fdsave.CurrentFieldControlUnitOffset
        CurrentRuntimeHeader, // type fdsave.CurrentRuntimeHeader
        decode, // func fdsave.Decode
        inspect_current_snapshot, // func fdsave.InspectCurrentSnapshot
        UNIT_SIZE, // const fdsave.UnitSize
        VerifiedMetadata, // type fdsave.VerifiedMetadata
    };
}

// 以上是 Go 測試檔另外用到的外部項目（正式程式已用到的不重列）；`cfg(test)` 下編譯檢查。

// 方法（掃描 Rust 原始碼時確認存在；`use` 無法引入方法）：
// - fdicon.Bank.SpriteFor -> fdicon::Bank::sprite_for
// - fdother.Frame.IndexedLayers -> fdother::Frame::indexed_layers
// - fdsave.CurrentSnapshot.ActivePersistentRecords -> save::CurrentSnapshot::active_persistent_records

// 找不到 Rust 對應（缺口，見 PORTING.md）：
