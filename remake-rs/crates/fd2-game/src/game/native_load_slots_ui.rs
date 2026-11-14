//! Go 來源：`remake/cmd/fd2/native_load_slots_ui.go`（360 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 7 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! F 組另外移植 `nativeHotelNodeForTown`（`restoreNativeChapterSlot` 與 `loadJSONGameFromSlot` 的酒店
//! 分支呼叫它）。`drawNativeLoadSlots`（畫面）與 `parseNativeLoadSlotShotState`（截圖模式）不在 R4。

use super::*;

/// Go：`native_load_slots_ui.go:19`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeLoadSlotsUIAssets {
    pub(crate) hd: Option<ModernHDTitleLoadSlots>,
    pub(crate) dialogue_box: fdother::LMI1Entry,
    pub(crate) strings: Option<fdtxt::Strings>,
    pub(crate) font: Option<fdtxt::Font>,
    pub(crate) palette: ColorPalette,
}

/// Go：`native_load_slots_ui.go:32`，`func loadNativeLoadSlotsUIAssets() (*nativeLoadSlotsUIAssets, error)`
pub(crate) fn load_native_load_slots_ui_assets() -> Result<Option<NativeLoadSlotsUIAssets>> {
    let dialogue_box = fdother::load_separated_load_slots_frame(separated_asset_path("ui"))?;
    let strings = fdtxt::load_separated_resource(separated_asset_path("text"), 0)?;
    let font = fdtxt::load_separated_font(separated_asset_path("fonts"))?;
    let (dac, mut palette) = load_native_battle_palette()?;
    // Go `palette[0] = color.NRGBA{A: 0xff}`：不透明黑。
    palette[0] = ColorRgba { r: 0, g: 0, b: 0, a: 0xff };
    let mut assets =
        NativeLoadSlotsUIAssets { hd: None, dialogue_box, strings: Some(strings), font: Some(font), palette };
    // Older isolated native packs can omit the shared dialogue cells.
    // Native loading remains available; entering HD preflights this descriptor.
    assets.hd = freeze_modern_hd_title_load_slots(Some(&assets), &dac).unwrap_or(None);
    Ok(Some(assets))
}

/// Go：`native_load_slots_ui.go:64`，`func nativeLoadSlotMetadata() ([4]campaign.NativeLoadSlot, bool)`
pub(crate) fn native_load_slot_metadata() -> ([campaign::NativeLoadSlot; 4], bool) {
    let native_save_path = std::env::var("FD2_NATIVE_SAVE").unwrap_or_default();
    if !native_save_path.is_empty() {
        return native_load_slot_metadata_from_fd2_sav(&native_save_path);
    }
    let mut slots = [campaign::NativeLoadSlot::default(); 4];
    for (slot, entry) in slots.iter_mut().enumerate() {
        let raw = match std::fs::read(save_slot_path(slot as i64)) {
            Ok(raw) => raw,
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => {
                entry.empty = true;
                continue;
            }
            Err(_) => return ([campaign::NativeLoadSlot::default(); 4], false),
        };
        let data = match go_json_unmarshal_zero::<SaveData>(&raw) {
            Ok(data) if data.chapter > 0 && data.chapter <= 0xff => data,
            // Remake JSON is not the native FD2.SAV ABI. Refuse values that
            // cannot map to 0x30437's raw chapter byte.
            _ => return ([campaign::NativeLoadSlot::default(); 4], false),
        };
        entry.chapter = data.chapter as u8;
    }
    (slots, true)
}

/// Go：`native_load_slots_ui.go:90`，`func nativeLoadSlotMetadataFromFD2SAV( path string, ) ([4]campaign.NativeLoadSlot, bool)`
pub(crate) fn native_load_slot_metadata_from_fd2_sav(path: &str) -> ([campaign::NativeLoadSlot; 4], bool) {
    let slots = [campaign::NativeLoadSlot::default(); 4];
    let Ok(stored) = std::fs::read(path) else {
        return (slots, false);
    };
    let Ok(plain) = fdsave::decode(&stored) else {
        return (slots, false);
    };
    let mut slots = slots;
    for (slot, entry) in slots.iter_mut().enumerate() {
        let Ok(metadata) = fdsave::read_verified_metadata(&plain, slot) else {
            return ([campaign::NativeLoadSlot::default(); 4], false);
        };
        if metadata.chapter == 0xff {
            entry.empty = true;
            continue;
        }
        entry.chapter = metadata.chapter;
    }
    (slots, true)
}

/// Go：`native_load_slots_ui.go:116`，`func nativeLoadSlotConfirmable(slot int) (confirmable, native bool)`
pub(crate) fn native_load_slot_confirmable(slot: i64) -> (bool, bool) {
    if slot < 0 || slot >= fdsave::SLOT_COUNT as i64 {
        return (false, false);
    }
    let (slots, ok) = native_load_slot_metadata();
    let native = !std::env::var("FD2_NATIVE_SAVE").unwrap_or_default().is_empty();
    if !ok || slots[slot as usize].empty {
        return (false, native);
    }
    (true, native)
}

impl Game {
    /// confirmTitleLoadSlot 是標題四槽 selector 的正式確認擁有者。原版槽只有在
    /// checksum、typed restore 與戰間節點進入全部成功後才離開 loadslots；失敗會
    /// 保留選槽畫面，避免看似進入遊戲卻留下部分還原狀態。
    ///
    /// Go：`native_load_slots_ui.go:130`，`func (g *Game) confirmTitleLoadSlot(selected int) bool`
    pub(crate) fn confirm_title_load_slot(&mut self, selected: i64) -> bool {
        let native_save = std::env::var("FD2_NATIVE_SAVE").unwrap_or_default();
        if native_save.is_empty() {
            // JSON stores node boundaries and a zero-based runtime chapter. Its
            // first chapter cannot use the native text-frame metadata projection.
            // Share F9's actual restore result, including its rejection path.
            if selected < 0 || selected >= fdsave::SLOT_COUNT as i64 || !self.load_json_game_from_slot(selected, false)
            {
                return false;
            }
            self.title_phase = String::new();
            return true;
        }
        let (confirmable, native) = native_load_slot_confirmable(selected);
        if !confirmable {
            if native {
                self.msg = "無法讀取原版存檔槽".to_string();
            } else {
                self.msg = "空的存檔槽".to_string();
            }
            return false;
        }
        if native {
            if let Err(err) = self.load_native_game_from_slot(&native_save, selected) {
                self.msg = err.message().to_string();
                return false;
            }
        } else {
            // Go 157 行：FD2_NATIVE_SAVE 非空時 native 恆為真，這個分支實際走不到；照 Go 保留。
            self.load_game_from_slot(selected);
        }
        self.title_phase = String::new();
        true
    }

    /// loadNativeGameFromSlot owns the original four-slot LOAD transaction. It is
    /// deliberately separate from title CONTINUE/current-snapshot restore.
    ///
    /// Go：`native_load_slots_ui.go:166`，`func (g *Game) loadNativeGameFromSlot(path string, slot int) error`
    pub(crate) fn load_native_game_from_slot(&mut self, path: &str, slot: i64) -> Result<()> {
        self.restore_native_chapter_slot(path, slot, false)
    }
}

/// Go：`native_load_slots_ui.go:170`
/// Go `errors.New` 的哨兵錯誤；`errors.Is` 以 [`GameError::is`] 判斷。
pub(crate) static ERR_NATIVE_HOTEL_PREPARATION_SLOT: GameError = GameError::sentinel("本章不能由商店內讀取！");

/// nativeHotelNodeForTown只沿已編寫的城鎮選項找酒店，不猜節點名稱或建立返回邊。
///
/// Go：`native_load_slots_ui.go:173`，`func nativeHotelNodeForTown(graph *campaign.Campaign, town string) (string, error)`
/// （F 組補移植，原標 `r8`）
pub(crate) fn native_hotel_node_for_town(graph: Option<&campaign::Campaign>, town: &str) -> Result<String> {
    let Some(town_node) = graph.and_then(|g| g.nodes.get(town)).filter(|n| n.r#type == "town") else {
        return Err(game_err!("酒店讀檔：存檔沒有城鎮返回入口"));
    };
    let graph = graph.expect("上一行已確認 graph 非 nil");
    let mut found = String::new();
    for option in &town_node.options {
        let Some(node) = graph.nodes.get(&option.to) else {
            continue;
        };
        if node.r#type != "hotel" || node.next != town {
            continue;
        }
        if !found.is_empty() {
            return Err(game_err!("酒店讀檔：城鎮酒店入口不唯一"));
        }
        found = option.to.clone();
    }
    if found.is_empty() {
        return Err(game_err!("酒店讀檔：城鎮未編寫酒店返回邊"));
    }
    Ok(found)
}

impl Game {
    /// restoreNativeChapterSlot共用四槽資料交易。酒店caller先查302BF的gate，
    /// 成功後仍在酒店顯示1DE；標題LOAD則沿原plan進城鎮／整備。
    ///
    /// Go：`native_load_slots_ui.go:196`，`func (g *Game) restoreNativeChapterSlot(path string, slot int, inHotel bool) error`
    pub(crate) fn restore_native_chapter_slot(&mut self, path: &str, slot: i64, in_hotel: bool) -> Result<()> {
        if self.camp.as_ref().is_none_or(|c| c.c.is_none()) {
            return Err(game_err!("原版四槽讀檔：戰役圖尚未載入"));
        }
        let wrap = |err: GameError| err.wrap("原版四槽讀檔：");
        let stored = std::fs::read(path).map_err(|err| wrap(GameError::new(err.to_string())))?;
        let plain = fdsave::decode(&stored).map_err(|err| wrap(err.into()))?;
        let snapshot = fdsave_slot(slot)
            .and_then(|s| fdsave::inspect_chapter_slot(&plain, s).map_err(GameError::from))
            .map_err(wrap)?;
        let mut catalog =
            campaign::load_native_character_catalog(&asset_path("assets/data/native_character_catalog.json"))
                .map_err(|err| wrap(err.into()))?;
        let gates =
            campaign::load_native_intermission_gate_table(&asset_path("assets/data/native_intermission_gate.json"))
                .map_err(|err| wrap(err.into()))?;
        if in_hotel {
            let chapter = snapshot.verified.chapter as usize;
            if chapter >= gates.entries.len() {
                return Err(game_err!("酒店讀檔：原版章節超出gate資料"));
            }
            if gates.entries[chapter] != 0 {
                return Err(ERR_NATIVE_HOTEL_PREPARATION_SLOT.clone());
            }
        }
        let graph = self.camp.as_ref().and_then(|c| c.c.as_ref());
        let mut plan = campaign::build_native_chapter_slot_restore_plan(&snapshot, &mut catalog, &gates, graph)?;
        let mut entry_node = plan.entry_node.clone();
        if in_hotel {
            entry_node = native_hotel_node_for_town(graph, &entry_node)?;
        }
        let options = fdother::NativeSystemOptions {
            raw_53af9: plan.raw53_af9,
            raw_51aab: plan.hud_gate_a,
            raw_51e61: plan.raw51_e61,
            raw_51e62: plan.raw51_e62,
        };
        options.validate().map_err(|err| GameError::from(err).wrap("原版四槽讀檔：系統設定："))?;
        // Go `uint64(plan.Currency) > uint64(^uint(0)>>1)`：64 位元平台的 int 上限。
        if u64::from(plan.currency) > i64::MAX as u64 {
            return Err(game_err!("原版四槽讀檔：金幣 {} 超出目前平台整數範圍", plan.currency));
        }

        let balance_state = load_balance_saved_state(path, &plain, slot)?;
        // Every fallible decode/validation step has completed. Apply the complete
        // transaction at once so malformed native data cannot leave mixed state.
        self.capture_native_map_hud_persistence();
        for unit in plan.party_roster.values_mut() {
            campaign::apply_equipped_attack_range(Some(unit), &self.shop_item_stats);
        }
        let graph = self.camp.as_ref().and_then(|c| c.c.as_ref()).expect("已確認 g.camp.C 非 nil");
        let mut runner = campaign::new_runner(graph);
        runner.cur = entry_node;
        self.camp = Some(runner);
        self.gold = i64::from(plan.currency);
        self.items = Vec::new();
        // Go 讓 g.partyMembers／partyJoinOrder／partyRoster 與 plan 共用同一份 map 與切片；
        // nativeChapterRestore 之後只當「有沒有四槽還原」的旗標讀（main.go:3044、3188、3930），
        // Rust 存複本。
        self.party_members = plan.party_members.clone();
        self.party_join_order = plan.party_join_order.clone();
        self.party_roster = plan.party_roster.clone();
        self.balance = balance_state.options;
        self.balance_ui = BalanceEditor::default();
        self.party_deploy = BTreeMap::new();
        self.handler_chapter = plan.native_chapter_index;
        self.restore_native_map_hud_gate_a(plan.hud_gate_a);
        self.native_system_options = Some(options);
        // Native chapter slots restore persistent records and metadata, not the
        // previous battle array. Drop it before enterNode so its old gate A cannot
        // overwrite the slot byte through the cross-node capture hook.
        self.dialog = Vec::new();
        self.st = None;
        self.sel = None;
        let plan_entry = plan.entry_node.clone();
        let display_chapter = plan.display_chapter;
        self.native_chapter_restore = Some(plan);
        // 酒店存檔（0x30012）要把持續紀錄寫回同一份 FD2.SAV 的同一個槽；讀檔時的
        // plaintext 與槽快照就是那次寫回的 byte 基底（沒有 raw 來源的 byte 照抄）。
        self.native_chapter_slot_plain = plain;
        self.native_chapter_slot_baseline = Some(snapshot);
        self.native_chapter_slot_started = false;
        self.clear_church_transient_state_for_load();
        self.clear_shop_transient_state_for_load();
        self.native_hotel_prompt_return = false;
        self.load_err = String::new();
        self.enter_node();
        if !self.load_err.is_empty() {
            return Err(game_err!("原版四槽讀檔：進入 {} 失敗：{}", plan_entry, self.load_err));
        }
        if in_hotel {
            self.hotel_sel = NATIVE_HOTEL_SERVICE_LOAD;
            self.native_hotel_mode = "loaded".to_string();
            self.native_hotel_prompt_return = true;
        }
        self.msg = format!("已讀取原版槽位{}（第{}章：{}）", slot + 1, display_chapter, plan_entry);
        Ok(())
    }
}

#[cfg(test)]
mod native_load_slots_ui_test;
