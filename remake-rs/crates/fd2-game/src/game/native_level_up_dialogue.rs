//! Go 來源：`remake/cmd/fd2/native_level_up_dialogue.go`（373 行）。
//!
//! 骨架：Go 檔的 11 個函式中產生 9 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const NATIVE_LEVEL_UP_EXP_TEXT_INDEX: i64 = 0x1e8;

pub(crate) const NATIVE_LEVEL_UP_TEXT_INDEX: i64 = 0x1e9;

/// AP、DP、DX、MaxHP、MaxMP 依序 0x1EA..0x1EE
pub(crate) const NATIVE_LEVEL_UP_GROWTH_TEXT_BASE: i64 = 0x1ea;

pub(crate) const NATIVE_LEVEL_UP_LEARN_TEXT_INDEX: i64 = 0x24b;

/// FFFC = 0x1B9 + command id（0x1E389）
pub(crate) const NATIVE_LEVEL_UP_LEARN_NAME_BASE: i64 = 0x1b9;

/// 0x1E504 push 0xb：11 個 BIOS tick（18.2 Hz）≈ 0.6 s；重製端以 60 Hz 幀近似。
pub(crate) const NATIVE_LEVEL_UP_CLOSE_FRAMES: i64 = 36;

/// Go：`native_level_up_dialogue.go:32`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct PendingNativeLevelUp {
    pub(crate) unit: Option<UnitIdx>,
    pub(crate) exp: i64,
    pub(crate) ups: Vec<battle::LevelUpEvent>,
}

/// Go：`native_level_up_dialogue.go:38`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeLevelUpDialogueState {
    pub(crate) unit: Option<UnitIdx>,
    pub(crate) hd_world: Option<ModernHDMapFrame>,
    pub(crate) hd_coverage: BTreeMap<campaign::NativeDialogueFramePhase, Vec<Vec<bool>>>,
    /// awaitLastKey：最後一頁也要等鍵（原版最後一欄的 FFFD）；沒有升級時只有一頁一行、
    /// 沒有 FFFD，直接進計時。
    pub(crate) await_last_key: bool,
    pub(crate) close_timer: i64,
    /// 0x1E5C0(n) 的 n 個 BIOS tick 換成 60 Hz 幀
    pub(crate) close_frames: i64,
    pub(crate) closing: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl NativeLevelUpDialogueState {
    /// 保存第一次 HD 繪製的背景，後續原生幀仍照常提交。
    pub(crate) fn freeze_hd_world<'a>(
        &'a mut self,
        frame: &ModernHDMapFrame,
        cells: &[fdother::RawCell],
        control: &str,
    ) -> Result<&'a ModernHDMapFrame> {
        if self.hd_world.is_none() {
            let mut coverage = BTreeMap::new();
            for (phase, count) in [
                (campaign::NATIVE_DIALOGUE_OPENING, 5),
                (campaign::NATIVE_DIALOGUE_STEADY, 1),
                (campaign::NATIVE_DIALOGUE_CLOSING, 5),
            ] {
                let masks = (0..count)
                    .map(|index| campaign::native_story_dialogue_coverage(cells, control, phase, index))
                    .collect::<std::result::Result<Vec<_>, _>>()?;
                coverage.insert(phase, masks);
            }
            self.hd_coverage = coverage;
            self.hd_world = Some(frame.clone());
        }
        Ok(self.hd_world.as_ref().unwrap())
    }
}

impl Game {
    /// Go：`modern_hd_timed_dialogue.go:13`，`modernHDTimedDialogueEligible`。
    pub(crate) fn modern_hd_timed_dialogue_eligible(&self) -> bool {
        self.native_level_up_dialogue.is_some()
            && !self.story_bg
            && self.walk.is_none()
            && self.banner_t <= 0
            && self.native_dialogue_layout.is_some()
            && self.native_class_ui.is_some()
            && self.native_dialogue_closing.len() == 5
            && matches!(self.dialog.as_slice(), [line] if line.native_dialogue.as_ref().is_some_and(|layout| {
                layout.control == "FFEC" && layout.has_motion_target_y && layout.motion_target_y == 0
            }))
    }

    /// queueNativeLevelUpDialogue 在原生 caller 收下經驗時排一則。0x29F72的+6==2
    /// 是累計writer閘門，不是0x1E292的consumer閘門；敵人也可能承接前值（#81）。
    ///
    /// Go：`native_level_up_dialogue.go:53`，`func (g *Game) queueNativeLevelUpDialogue(unit *battle.Unit, exp int, ups []battle.LevelUpEvent)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn queue_native_level_up_dialogue(
        &mut self,
        unit: Option<UnitIdx>,
        exp: i64,
        ups: &[battle::LevelUpEvent],
    ) {
        let Some(index) = unit else {
            return;
        };
        if exp <= 0 || !self.st.as_ref().expect("Go：unit 指向 g.st.Units").units[index].has_native_record_byte6 {
            return;
        }
        self.pending_native_level_ups.push(PendingNativeLevelUp { unit, exp, ups: ups.to_vec() });
    }

    /// Go：`native_level_up_dialogue.go:60`，`func (g *Game) runPendingNativeLevelUpDialogues(then func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn run_pending_native_level_up_dialogues(
        &mut self,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        if self.pending_native_level_ups.is_empty()
            || self.native_level_up_dialogue.is_some()
            || !self.dialog.is_empty()
        {
            return false;
        }
        let pending = self.pending_native_level_ups.remove(0);
        if let Err(err) = self.begin_native_level_up_dialogue(&pending, then) {
            self.load_err = format!("native level-up dialogue: {}", err.message());
        }
        true
    }
}

/// expandNativeMessageWords 把 FDTXT_000 一條字串展開成原始 glyph word：FFFA 代入十進位數字、
/// FFFC 代入名字，FFFD 只回報等鍵。0x15F84 的單行訊息（升級、中毒扣血）共用。
///
/// Go：`native_level_up_dialogue.go:74`，`func expandNativeMessageWords(strings interface { Words(int) ([]uint16, error) }, index int, number int, name []uint16) ([]uint16, bool, error)`
/// Go 參數是匿名介面 `interface{ Words(int) ([]uint16, error) }`；Rust 對應 [`NativeMessageStrings`]（正式程式只傳 `*fdtxt.Strings`，測試傳假字串表）。
pub(crate) fn expand_native_message_words(
    strings: &dyn NativeMessageStrings,
    index: i64,
    number: i64,
    name: &[u16],
) -> Result<(Vec<u16>, bool)> {
    let words = strings.words(index)?;
    let mut out = Vec::with_capacity(words.len() + 8);
    let mut wait = false;
    for word in words {
        match word {
            0xfffa => out.extend(native_message_decimal_digits(number)),
            0xfffc => out.extend_from_slice(name),
            0xfffd => wait = true,
            0xfffe => {
                return Err(game_err!("message text {} has an unexpected line break", battle::go_sharp_x(index)));
            }
            _ => out.push(word),
        }
    }
    Ok((out, wait))
}

/// Go `for _, digit := range strconv.Itoa(number) { out = append(out, uint16(digit-'0')) }`：負數的 `-` 也照
/// Go 換成 `uint16('-' - '0')`（繞回成 0xfffd）。
fn native_message_decimal_digits(number: i64) -> Vec<u16> {
    number.to_string().chars().map(|digit| (digit as i32 - '0' as i32) as u16).collect()
}

/// nativeMessagePages 把一條 FDTXT_000 字串展開成 0x15F84 會寫的頁：FFFE 換行、FFFD 等鍵後
/// 換頁，FFFA 代入十進位數字。回傳的 lastWait 表示字串以 FFFD 收尾（最後一頁也等鍵）。
///
/// Go：`native_level_up_dialogue.go:104`，`func nativeMessagePages(strings interface { Words(int) ([]uint16, error) }, index int, number int) ([][][]uint16, bool, error)`
/// （`r8`，隨呼叫端 `beginNativeTransientPhases` 補移植）。
pub(crate) fn native_message_pages(
    strings: &dyn NativeMessageStrings,
    index: i64,
    number: i64,
) -> Result<(Vec<Vec<Vec<u16>>>, bool)> {
    let words = strings.words(index)?;
    let mut pages: Vec<Vec<Vec<u16>>> = Vec::new();
    let mut page: Vec<Vec<u16>> = Vec::new();
    let mut line: Vec<u16> = Vec::new();
    let mut last_wait = false;
    for word in words {
        last_wait = false;
        match word {
            0xfffa => line.extend(native_message_decimal_digits(number)),
            0xfffe => page.push(std::mem::take(&mut line)),
            0xfffd => {
                page.push(std::mem::take(&mut line));
                pages.push(std::mem::take(&mut page));
                last_wait = true;
            }
            0xfffc => {
                return Err(game_err!("message text {} needs a name argument", battle::go_sharp_x(index)));
            }
            _ => line.push(word),
        }
    }
    if !line.is_empty() {
        page.push(line);
    }
    if !page.is_empty() {
        pages.push(page);
    }
    if pages.is_empty() {
        return Err(game_err!("message text {} is empty", battle::go_sharp_x(index)));
    }
    Ok((pages, last_wait))
}

/// nativeLevelUpDialogueLines 把 0x1E292 會寫的每一行展開成 FDTXT 原始字（FFFA／FFFC 已代入），
/// 依 FFFD 切頁。回傳每頁的行（每行是原始 glyph word 序列），以及最後一行是否以 FFFD 收尾
/// （是：最後一頁也等鍵；否：例如只有經驗一行、或「學會了」收尾，直接進 0x1E5C0 計時）。
///
/// Go：`native_level_up_dialogue.go:151`，`func nativeLevelUpDialogueLines(strings interface { Words(int) ([]uint16, error) }, pending pendingNativeLevelUp) ([][][]uint16, bool, error)`
/// Go 參數是匿名介面 `interface{ Words(int) ([]uint16, error) }`；Rust 對應 [`NativeMessageStrings`]（正式程式只傳 `*fdtxt.Strings`，測試傳假字串表）。
pub(crate) fn native_level_up_dialogue_lines(
    strings: &dyn NativeMessageStrings,
    pending: &PendingNativeLevelUp,
) -> Result<(Vec<Vec<Vec<u16>>>, bool)> {
    let mut pages: Vec<Vec<Vec<u16>>> = Vec::new();
    let mut page: Vec<Vec<u16>> = Vec::new();
    let mut last_wait = false;
    let mut push = |pages: &mut Vec<Vec<Vec<u16>>>, line: Vec<u16>, wait: bool| {
        page.push(line);
        last_wait = wait;
        if wait {
            pages.push(std::mem::take(&mut page));
        }
    };
    let (line, wait) = expand_native_message_words(strings, NATIVE_LEVEL_UP_EXP_TEXT_INDEX, pending.exp, &[])?;
    push(&mut pages, line, wait);
    for up in &pending.ups {
        let (line, wait) = expand_native_message_words(strings, NATIVE_LEVEL_UP_TEXT_INDEX, 0, &[])?;
        push(&mut pages, line, wait);
        for (i, gain) in [up.ap_gain, up.dp_gain, up.dx_gain, up.hp_gain, up.mp_gain].into_iter().enumerate() {
            if gain == 0 {
                continue; // 0x1E560：欄位為零不寫那一行
            }
            let (line, wait) =
                expand_native_message_words(strings, NATIVE_LEVEL_UP_GROWTH_TEXT_BASE + i as i64, gain, &[])?;
            push(&mut pages, line, wait);
        }
        for &command in &up.learned_command_ids {
            let name = strings.words(NATIVE_LEVEL_UP_LEARN_NAME_BASE + command)?;
            let (line, wait) = expand_native_message_words(strings, NATIVE_LEVEL_UP_LEARN_TEXT_INDEX, 0, &name)?;
            push(&mut pages, line, wait);
        }
    }
    drop(push);
    if !page.is_empty() {
        pages.push(page);
    }
    Ok((pages, last_wait))
}

impl Game {
    /// Go：`native_level_up_dialogue.go:207`，`func (g *Game) beginNativeLevelUpDialogue(pending pendingNativeLevelUp, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_level_up_dialogue(
        &mut self,
        pending: &PendingNativeLevelUp,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        let Some(index) = pending.unit else {
            return Err(game_err!("unit DATO selector or map presentation is unavailable"));
        };
        let unit = &self.st.as_ref().expect("Go：pending.unit 指向 g.st.Units").units[index];
        if !unit.has_battle_fig || !unit.has_native_map_presentation {
            return Err(game_err!("unit DATO selector or map presentation is unavailable"));
        }
        let Some(strings) = self.native_preparation_ui.as_ref().and_then(|ui| ui.status.strings.as_ref()) else {
            return Err(game_err!("FDTXT_000 strings are unavailable"));
        };
        let (pages, last_wait) = native_level_up_dialogue_lines(strings, pending)?;
        if std::env::var("FD2_SHOT_AI").is_ok_and(|v| !v.is_empty()) {
            eprintln!(
                "level-up dialogue: unit id={} at ({},{}) exp={} ups={} pages={} lastWait={}",
                unit.native_identity,
                unit.x,
                unit.y,
                pending.exp,
                pending.ups.len(),
                pages.len(),
                last_wait
            );
        }
        self.begin_native_timed_message_dialogue(
            pending.unit,
            NATIVE_LEVEL_UP_EXP_TEXT_INDEX,
            &pages,
            last_wait,
            NATIVE_LEVEL_UP_CLOSE_FRAMES,
            then,
        )
    }

    /// beginNativeTimedMessageDialogue 是 0x1956B(unit+7) 開框 → 0x15F84 寫 FDTXT_000 字串（FFFD 等鍵
    /// 切頁）→ 0x1E5C0(n) 計時 → 0x196CB 關框的共用排程；升級對話（0x1E292）與中毒扣血訊息
    /// （0x1A866）都走這條。
    ///
    /// Go：`native_level_up_dialogue.go:228`，`func (g *Game) beginNativeTimedMessageDialogue(unit *battle.Unit, textIndex int, pages [][][]uint16, lastWait bool, closeFrames int, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_timed_message_dialogue(
        &mut self,
        unit: Option<UnitIdx>,
        text_index: i64,
        pages: &[Vec<Vec<u16>>],
        last_wait: bool,
        close_frames: i64,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        let Some(index) = unit else {
            return Err(game_err!("unit DATO selector or map presentation is unavailable"));
        };
        let u = &self.st.as_ref().expect("Go：unit 指向 g.st.Units").units[index];
        if !u.has_battle_fig || !u.has_native_map_presentation {
            return Err(game_err!("unit DATO selector or map presentation is unavailable"));
        }
        let speaker = u.battle_fig;
        // Go 以 `g.nativeBattleGlyphs == nil` 區分沒載入與空表；Rust 的容器不分（B 組 `story-roster-nil-vs-empty`）。
        if self.locale_id == "zh-Hant" && (self.native_battle_font.is_none() || self.native_battle_glyphs.is_empty()) {
            let (font, glyphs) = load_native_battle_name_assets()?;
            (self.native_battle_font, self.native_battle_glyphs) = (font, glyphs);
        }
        if self.native_battle_glyphs.is_empty() {
            return Err(game_err!("glyph index is unavailable"));
        }
        // 原始 glyph word → 索引表的鍵（故事對白引擎用鍵查 glyph）。
        let mut key_of: BTreeMap<i64, &String> = BTreeMap::new();
        for (key, &glyph) in &self.native_battle_glyphs {
            match key_of.get(&glyph) {
                Some(prev) if key >= *prev => {}
                _ => {
                    key_of.insert(glyph, key);
                }
            }
        }
        let mut layout = battle::NativeDialogueLayout {
            source_dat: String::from("FDTXT_000"),
            string_index: text_index,
            utterance: 0,
            control: String::from("FFEC"),
            operand: 0,
            motion_target_y: 0,
            has_motion_target_y: true,
            ..Default::default()
        };
        for page in pages {
            let (mut texts, mut glyph_rows) = (Vec::new(), Vec::new());
            for line in page {
                let mut tokens = Vec::with_capacity(line.len());
                let mut text = String::new();
                for &word in line {
                    let Some(key) = key_of.get(&(word as i64)) else {
                        return Err(game_err!("glyph {} has no index key", word));
                    };
                    tokens.push((*key).clone());
                    text.push_str(key);
                }
                texts.push(text);
                glyph_rows.push(tokens);
            }
            layout.pages.push(texts);
            layout.glyph_pages.push(glyph_rows);
        }
        // 0x11951 的 0x11CAC 整幀重繪：對話底圖是攻擊者已變灰、屍體已移除的地圖，不是攻擊
        // 演出前留下的那一張。
        self.native_map_vga = Vec::new();
        self.dialog = vec![battle::DialogLine {
            speaker,
            upper: Some(false),
            native_dialogue: Some(layout),
            ..Default::default()
        }];
        (self.dlg_page, self.dlg_scroll_t, self.dlg_shown, self.dlg_phase, self.dlg_t) = (0, 0, DLG_NONE, 0, 0);
        self.native_level_up_dialogue = Some(NativeLevelUpDialogueState {
            unit,
            await_last_key: last_wait,
            close_frames,
            then,
            ..Default::default()
        });
        if let Err(err) = self.start_native_dialogue_frames() {
            self.dialog = Vec::new();
            self.native_level_up_dialogue = None;
            return Err(err);
        }
        Ok(())
    }

    /// nativeLevelUpDialogueAtWait 回報對話正停在等鍵（頁尾 FFFD，或最後一頁字寫完）。
    ///
    /// Go：`native_level_up_dialogue.go:289`，`func (g *Game) nativeLevelUpDialogueAtWait() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_level_up_dialogue_at_wait(&self) -> bool {
        self.native_level_up_dialogue.as_ref().is_some_and(|state| !state.closing && state.close_timer == 0)
            && self.native_story_dialogue_at_input_wait()
    }

    /// stepNativeLevelUpDialogue 每幀：字寫完且不等鍵的頁（沒有升級的單頁）直接進 0x1E5C0 的
    /// 計時；計時到就 0x196CB 關框。
    ///
    /// Go：`native_level_up_dialogue.go:296`，`func (g *Game) stepNativeLevelUpDialogue(advance bool)`
    pub(crate) fn step_native_level_up_dialogue(&mut self, advance: bool) {
        if self.native_level_up_dialogue.is_none() || self.dialog.is_empty() || self.native_dialogue_closing_live {
            return;
        }
        let state = self.native_level_up_dialogue.as_mut().unwrap();
        if state.close_timer > 0 {
            state.close_timer -= 1;
            if advance {
                state.close_timer = 0;
            }
            if state.close_timer == 0 {
                state.closing = true;
                if !self.begin_native_story_dialogue_closing() {
                    self.load_err = String::from("native level-up dialogue: closing frames unavailable");
                }
            }
            return;
        }
        if !self.native_story_dialogue_at_input_wait() {
            return;
        }
        let last = self.dlg_page + 1 >= dlg_page_count(self.dialog.last().unwrap());
        if !last {
            if advance {
                self.dlg_advance();
            }
            return;
        }
        let state = self.native_level_up_dialogue.as_mut().unwrap();
        if !state.await_last_key || advance {
            state.close_timer = state.close_frames;
            if state.close_timer <= 0 {
                state.close_timer = NATIVE_LEVEL_UP_CLOSE_FRAMES;
            }
        }
    }

    /// Go：`native_level_up_dialogue.go:332`，`func (g *Game) finishNativeLevelUpDialogue()`
    pub(crate) fn finish_native_level_up_dialogue(&mut self) {
        let Some(mut state) = self.native_level_up_dialogue.take() else {
            return;
        };
        if let Some(then) = state.then.take() {
            then(self);
        }
    }
}

/// R4 D 組：Go `expandNativeMessageWords`、`nativeMessagePages`、`nativeLevelUpDialogueLines` 的匿名介面參數
/// `interface{ Words(int) ([]uint16, error) }`（簽名變更紀錄；骨架寫成具體的 `*fdtxt.Strings`）。
pub(crate) trait NativeMessageStrings {
    /// Go `Words(index)`。
    fn words(&self, index: i64) -> Result<Vec<u16>>;
}

impl NativeMessageStrings for fdtxt::Strings {
    fn words(&self, index: i64) -> Result<Vec<u16>> {
        Ok(fdtxt::Strings::words(self, index)?)
    }
}

#[cfg(test)]
#[path = "native_level_up_dialogue/native_level_up_dialogue_test.rs"]
mod native_level_up_dialogue_test;
