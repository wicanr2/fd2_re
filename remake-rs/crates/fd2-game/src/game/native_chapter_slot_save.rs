//! Go 來源：`remake/cmd/fd2/native_chapter_slot_save.go`（212 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    pub(crate) fn reorder_preparation_roster(&mut self, order: Vec<i64>) -> Result<()> {
        if let Some(baseline) = &self.native_chapter_slot_baseline {
            let table =
                campaign::load_native_join_constructor_table(&asset_path("assets/data/native_join_constructor.json"))?;
            let item_rows =
                battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))?;
            let mut built = campaign::build_native_chapter_slot(
                baseline,
                &self.party_roster,
                &self.party_join_order,
                u8::try_from(self.handler_chapter).map_err(|_| game_err!("整備章節超出原版範圍"))?,
                u32::try_from(self.gold).map_err(|_| game_err!("金幣超出原版範圍"))?,
                &table,
                &item_rows,
            )?;
            let count = usize::from(built.metadata[1]);
            let mut records = BTreeMap::new();
            let (slots, remainder) = built.roster.as_chunks_mut::<{ fdsave::UNIT_SIZE }>();
            if !remainder.is_empty() {
                return Err(game_err!("整備原版名冊記錄尺寸無效"));
            }
            for record in slots.iter().take(count) {
                if records.insert(i64::from(record[8]), *record).is_some() {
                    return Err(game_err!("整備名冊有重複角色"));
                }
            }
            if records.len() != order.len() {
                return Err(game_err!("整備前後名冊人數不符"));
            }
            for (id, record) in order.iter().zip(slots) {
                let unit = self
                    .party_roster
                    .get(id)
                    .filter(|unit| unit.has_native_identity)
                    .ok_or_else(|| game_err!("整備角色缺少原版身分"))?;
                let original =
                    records.remove(&unit.native_identity).ok_or_else(|| game_err!("找不到整備角色的原版存檔記錄"))?;
                *record = original;
            }
            let plain = fdsave::write_slot(&self.native_chapter_slot_plain, baseline.slot, &built)?;
            self.native_chapter_slot_baseline = Some(fdsave::inspect_chapter_slot(&plain, baseline.slot)?);
        }
        self.party_join_order = order;
        Ok(())
    }

    /// saveNativeChapterSlot 是酒店存檔（0x30012）的原版槽寫回：以標題 LOAD 讀進來的
    /// FD2.SAV plaintext 為底；正常 START 首次保存按已確認政策建立新名冊。
    /// 把目前持續隊伍與金幣寫進 slot，重算校驗和後原子取代
    /// FD2_NATIVE_SAVE。寫成功後同一份 plaintext 成為下一次存檔的基底。
    ///
    /// Go：`native_chapter_slot_save.go:17`，`func (g *Game) saveNativeChapterSlot(slot int) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn save_native_chapter_slot(&mut self, slot: i64) -> Result<()> {
        if self.native_end_turn_recovery.is_some() {
            return Err(game_err!("原版章節槽存檔：END 回復尚未發布完成"));
        }
        let path = native_current_save_path();
        if path.is_empty() {
            return Err(game_err!("原版章節槽存檔：找不到 FD2.SAV 路徑"));
        }
        if self.handler_chapter < 0
            || self.handler_chapter > 0xfe
            || self.gold < 0
            || self.gold as u64 > u64::from(u32::MAX)
        {
            return Err(game_err!("原版章節槽存檔：章節或金幣超出 metadata 範圍"));
        }
        // READY：fd2_ch01_flow_20261017.json start_save_contract。START 不還原舊槽隊伍。
        // 新名冊沿本引擎既有零基底；原版保留值的差異已逐筆揭露並經使用者確認。
        // 只建立本地候選，磁碟取代成功前不發布基底。
        let mut base_plain = self.native_chapter_slot_plain.clone();
        let mut baseline = self.native_chapter_slot_baseline.clone();
        if baseline.is_none() && self.native_chapter_slot_started {
            let stored = std::fs::read(&path)
                .map_err(|err| GameError::new(err.to_string()).wrap("原版章節槽存檔：讀取容器："))?;
            base_plain = fdsave::decode(&stored).map_err(|err| GameError::from(err).wrap("原版章節槽存檔：容器："))?;
            let old = fdsave_slot(slot)
                .and_then(|s| fdsave::read_slot(&base_plain, s).map_err(GameError::from))
                .map_err(|err| err.wrap("原版章節槽存檔：目標槽："))?;
            let mut fresh = fdsave::ChapterSlotSnapshot {
                slot: slot as usize,
                verified: fdsave::VerifiedMetadata::default(),
                metadata: [0; fdsave::METADATA_SIZE],
                records: [fdsave::PersistentRecord::default(); fdsave::ROSTER_UNITS],
                source_save_sha256: String::new(),
            };
            // Go `copy(fresh.Metadata[:], old.Metadata)`：複製兩者較短的長度。
            let n = fresh.metadata.len().min(old.metadata.len());
            fresh.metadata[..n].copy_from_slice(&old.metadata[..n]);
            fresh.metadata[..10].fill(0);
            let options = self.current_native_system_options();
            options.validate().map_err(|err| GameError::from(err).wrap("原版章節槽存檔：系統選項："))?;
            fresh.metadata[6] = options.raw_51aab;
            fresh.metadata[7] = options.raw_53af9;
            fresh.metadata[8] = options.raw_51e61;
            fresh.metadata[9] = options.raw_51e62;
            baseline = Some(fresh);
        }
        let Some(baseline) = baseline.filter(|_| base_plain.len() == fdsave::FILE_SIZE) else {
            return Err(game_err!("原版章節槽存檔：缺少 START 或四槽 LOAD 的基底"));
        };
        let join_table =
            campaign::load_native_join_constructor_table(&asset_path("assets/data/native_join_constructor.json"))
                .map_err(|err| GameError::from(err).wrap("原版章節槽存檔：JOIN constructor："))?;
        let item_rows =
            battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))
                .map_err(|err| GameError::from(err).wrap("原版章節槽存檔：item rows："))?;
        let built = campaign::build_native_chapter_slot(
            &baseline,
            &self.party_roster,
            &self.party_join_order,
            self.handler_chapter as u8,
            self.gold as u32,
            &join_table,
            &item_rows,
        )
        .map_err(|err| GameError::from(err).wrap("原版章節槽存檔："))?;
        let plain = fdsave_slot(slot)
            .and_then(|s| fdsave::write_slot(&base_plain, s, &built).map_err(GameError::from))
            .map_err(|err| err.wrap("原版章節槽存檔："))?;
        let stored = fdsave::encode(&plain).map_err(|err| GameError::from(err).wrap("原版章節槽存檔："))?;
        self.replace_native_save_with_balance(&path, &stored, slot)?;
        let snapshot = fdsave_slot(slot)
            .and_then(|s| fdsave::inspect_chapter_slot(&plain, s).map_err(GameError::from))
            .map_err(|err| err.wrap("原版章節槽存檔：回讀："))?;
        self.native_chapter_slot_plain = plain;
        self.native_chapter_slot_baseline = Some(snapshot);
        self.native_chapter_slot_started = false;
        Ok(())
    }
}
