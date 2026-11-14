//! `ending` 正式程式用到的外部項目清單（`fd2-rules`、`fd2-formats` 與本 crate 的 `campaign`）。沒有對應的 Go 檔。
//!
//! 每一項都在編譯期確認已有公開的 Rust 對應；階段 2 移植時直接用這些路徑。註解是 Go 名稱
//! （2026-11-14 以 `go/types` 從 `remake/internal/ending` 正式程式列出）。

#[rustfmt::skip]
#[allow(unused_imports)]
mod afm_items {
    use fd2_formats::afm::{
        Clip, // type afm.Clip
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod battle_items {
    use fd2_rules::battle::{
        apply_native_runtime_equipment_recalc, // func battle.ApplyNativeRuntimeEquipmentRecalc
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod campaign_items {
    use crate::campaign::{
        native_class_list_closing_frames, // func campaign.NativeClassListClosingFrames
        native_class_list_opening_frames, // func campaign.NativeClassListOpeningFrames
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod dato_items {
    use fd2_formats::dato::{
        Frame, // type dato.Frame
        load_separated_resource, // func dato.LoadSeparatedResource
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod fdicon_items {
    use fd2_formats::fdicon::{
        load_separated_bank, // func fdicon.LoadSeparatedBank
        NativeSelectorCache, // type fdicon.NativeSelectorCache
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod fdother_items {
    use fd2_formats::fdother::{
        apply_vga_palette_delta, // func fdother.ApplyVGAPaletteDelta
        decode_archive_single_frame, // func fdother.DecodeArchiveSingleFrame
        decode_lmi1_raw_entry, // func fdother.DecodeLMI1RawEntry
        decode_resource, // func fdother.DecodeResource
        Frame, // type fdother.Frame
        load_separated_fdother_palette, // func fdother.LoadSeparatedFDOTHERPalette
        load_separated_item_panel_entries, // func fdother.LoadSeparatedItemPanelEntries
        load_separated_single_frame, // func fdother.LoadSeparatedSingleFrame
        parse_vga_palette, // func fdother.ParseVGAPalette
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
        load_separated_font, // func fdtxt.LoadSeparatedFont
        load_separated_resource, // func fdtxt.LoadSeparatedResource
        NativeGlyphStyle, // type fdtxt.NativeGlyphStyle
        parse, // func fdtxt.Parse
        parse_font, // func fdtxt.ParseFont
        Strings, // type fdtxt.Strings
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod figani_items {
    use fd2_formats::figani::{
        Animation, // type figani.Animation
        Frame, // type figani.Frame
        load_separated_archive_resource, // func figani.LoadSeparatedArchiveResource
        load_separated_resource, // func figani.LoadSeparatedResource
        NativeScheduler, // type figani.NativeScheduler
    };
}

#[rustfmt::skip]
#[allow(unused_imports)]
mod save_items {
    use fd2_formats::save::{
        CURRENT_FIELD_CONTROL_UNIT_OFFSET, // const fdsave.CurrentFieldControlUnitOffset
        CURRENT_FIELD_CONTROL_UNIT_SIZE, // const fdsave.CurrentFieldControlUnitSize
        PersistentRecord, // type fdsave.PersistentRecord
        UNIT_SIZE, // const fdsave.UnitSize
    };
}

// 方法（掃描 Rust 原始碼時確認存在；`use` 無法引入方法）：
// - dato.Frame.BlitAtOffset -> dato::Frame::blit_at_offset
// - dato.Frame.BlitRightToLeftAtOffset -> dato::Frame::blit_right_to_left_at_offset
// - fdicon.NativeSelectorCache.SlotFor -> fdicon::NativeSelectorCache::slot_for
// - fdother.Frame.Blit -> fdother::Frame::blit
// - fdother.Frame.BlitAt -> fdother::Frame::blit_at
// - fdother.RawCell.BlitOpaqueAtOffset -> fdother::RawCell::blit_opaque_at_offset
// - fdtxt.Font.BlitNativeGlyph -> fdtxt::Font::blit_native_glyph
// - fdtxt.Font.GlyphCount -> fdtxt::Font::glyph_count
// - fdtxt.Strings.Words -> fdtxt::Strings::words
// - figani.Frame.BlitAt -> figani::Frame::blit_at
// - figani.Frame.BlitAtBase -> figani::Frame::blit_at_base
// - figani.Frame.BlitTranslated -> figani::Frame::blit_translated
// - figani.NativeScheduler.Step -> figani::NativeScheduler::step

// 找不到 Rust 對應（缺口，見 PORTING.md）：

#[cfg(test)]
#[rustfmt::skip]
#[allow(unused_imports)]
mod afm_test_items {
    use fd2_formats::afm::{
        decode_resource, // func afm.DecodeResource
    };
}

#[cfg(test)]
#[rustfmt::skip]
#[allow(unused_imports)]
mod battle_test_items {
    use fd2_rules::battle::{
        load_native_item_effect_row_prefix, // func battle.LoadNativeItemEffectRowPrefix
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
mod fdtxt_test_items {
    use fd2_formats::fdtxt::{
        GLYPH_BYTES, // const fdtxt.GlyphBytes
        GLYPH_HEIGHT, // const fdtxt.GlyphHeight
    };
}

// 以上是 Go 測試檔另外用到的外部項目（正式程式已用到的不重列）；`cfg(test)` 下編譯檢查。

// 方法（掃描 Rust 原始碼時確認存在；`use` 無法引入方法）：
// - fdtxt.Font.GlyphBit -> fdtxt::Font::glyph_bit

// 找不到 Rust 對應（缺口，見 PORTING.md）：
