//! Go 來源：`remake/cmd/fd2/native_death_reward_message.go`（180 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// pendingNativeDeathRewardMessage 是 0x1AA1D 物品型態 0／1 成功臂要播的訊息：
/// 0x1956B(killer+7) 開對話格 → 0x15F84 寫 0x1B0／0x1B3（FFFC 物品名／FFFA 金額）→
/// 0x16559(0) 蓋 DATO 第 0 幀 → 0x16C57(0) 等一個鍵 → 0x196CB 五幀關框並復原。
/// 第七章 r3 seq 389：原版 attack_result 的 checkpoint 就停在這個等鍵處（ui=dialogue），
/// 驅動端接著送 enter（finish-dialogue）才回到游標。
///
/// Go：`native_death_reward_message.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct PendingNativeDeathRewardMessage {
    pub(crate) killer: Option<UnitIdx>,
    pub(crate) kind: i64,
    pub(crate) value: i64,
}

/// Go：`native_death_reward_message.go:22`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeDeathRewardMessageState {
    pub(crate) killer: Option<UnitIdx>,
    /// 這次行動的單位：0x1AA1D 在 0x13512 設 +5 bit7 之前，底圖上它還沒變灰
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) await_ack: bool,
    pub(crate) r#final: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) portrait: dato::Frame,
    /// 0x16559 等待期間店主式眨眼／嘴型各幀，重播端各出一張變體
    pub(crate) portraits: Vec<dato::Frame>,
    pub(crate) kind: i64,
    pub(crate) value: i64,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_world: Option<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) opening: Option<NativeClassUIJob>,
    pub(crate) closing: Option<NativeClassUIJob>,
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    pub(crate) chrome_err: Option<GameError>,
}

impl Game {
    /// runPendingNativeDeathRewardMessages 在行動收尾播第一則待播訊息；資產不齊就失敗即
    /// 關閉（loadErr），不用重製端的 g.msg 文字條替代。
    ///
    /// Go：`native_death_reward_message.go:64`，`func (g *Game) runPendingNativeDeathRewardMessages(actor *battle.Unit, then func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn run_pending_native_death_reward_messages(
        &mut self,
        actor: UnitIdx,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        if self.pending_native_reward_msgs.is_empty() || self.native_system_end_turn_ui.is_some() {
            return false;
        }
        let pending = self.pending_native_reward_msgs.remove(0);
        if let Err(err) = self.begin_native_death_reward_message(pending, actor, then) {
            self.load_err = format!("native death reward message: {}", err.message());
        }
        true
    }

    /// composeNativeMapFrameBeforeActed 以「行動單位的 +5 bit7 還沒設」的狀態組整幀：0x18890 的
    /// handler 裡 0x1AA1D 先播訊息，0x13512 之後才設 bit7；重製端的結算一開始就設了，組底圖時
    /// 暫時拿掉（第七章 r6 seq 633：原版訊息底圖上攻擊者沒變灰）。
    ///
    /// Go：`native_death_reward_message.go:79`，`func (g *Game) composeNativeMapFrameBeforeActed(actor *battle.Unit) error`
    pub(crate) fn compose_native_map_frame_before_acted(&mut self, actor: Option<UnitIdx>) -> Result<()> {
        let cleared = match (actor, self.st.as_mut()) {
            (Some(actor), Some(st)) => {
                let unit = &mut st.units[actor];
                let cleared = unit.has_native_record_byte5 && unit.native_record_byte5 & 0x80 != 0;
                if cleared {
                    unit.native_record_byte5 &= !0x80;
                }
                cleared.then_some(actor)
            }
            _ => None,
        };
        let result = self.compose_native_map_frame();
        // Go 的 defer：`actor.NativeRecordByte5 |= 0x80`（Go 的 actor 指標不受 g.st 替換影響；合成期間 st 的單位索引不變）。
        if let (Some(actor), Some(st)) = (cleared, self.st.as_mut()) {
            st.units[actor].native_record_byte5 |= 0x80;
        }
        result
    }

    /// Go：`native_death_reward_message.go:87`，`func (g *Game) beginNativeDeathRewardMessage(pending pendingNativeDeathRewardMessage, actor *battle.Unit, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_death_reward_message(
        &mut self,
        pending: PendingNativeDeathRewardMessage,
        actor: UnitIdx,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        let killer_fig = pending.killer.and_then(|killer| {
            let unit = &self.st.as_ref().expect("Go：killer 指向 g.st.Units").units[killer];
            unit.has_battle_fig.then_some(unit.battle_fig)
        });
        let Some(killer_fig) = killer_fig else {
            return Err(game_err!("killer DATO selector is unavailable"));
        };
        if self.native_preparation_ui.is_none() || self.native_class_ui.is_none() {
            return Err(game_err!("dialogue assets are unavailable"));
        }
        let portraits = match load_native_separated_portrait(killer_fig) {
            Ok(portraits) if !portraits.is_empty() => portraits,
            _ => return Err(game_err!("killer portrait is unavailable")),
        };
        self.compose_native_map_frame_before_acted(Some(actor))?;
        let ui = self.native_preparation_ui.as_ref().unwrap();
        let source = self.native_map_vga.clone();
        let dialogue =
            campaign::compose_native_preparation_confirmation_dialogue(&source, &ui.dialogue, &portraits[0])?;
        let r#final = campaign::compose_native_death_reward_message_frame(
            &dialogue,
            ui.status.strings.as_ref(),
            ui.status.font.as_ref(),
            pending.kind,
            pending.value,
        )?;
        let opening = campaign::native_class_list_opening_frames(&source, &r#final)?;
        let mut message = NativeDeathRewardMessageState {
            killer: pending.killer,
            actor: Some(actor),
            r#final: r#final.clone(),
            then,
            portrait: portraits[0].clone(),
            portraits,
            kind: pending.kind,
            value: pending.value,
            ..Default::default()
        };
        let class_palette = &self.native_class_ui.as_ref().unwrap().palette;
        match freeze_modern_hd_dialogue_chrome(&ui.dialogue, class_palette) {
            Ok(chrome) => message.chrome = chrome,
            Err(err) => message.chrome_err = Some(err),
        }
        // Go 在這裡以 `freezeNativeCommand9MapInput`／`newModernHDCompoundMapScene` 凍結 HD 地圖場景（錯誤只存進
        // `hdScene`，只有 HD 繪製讀）；Rust 留到 R6，`hd_scene` 是 `None`（PORTING.md「D 組：HD 預凍結延後」）。
        if self.modern_story_portraits.is_some() {
            todo!("R6: native_death_reward_message.go:128 prepareModernHDRewardWorld（HD 主題）")
        }
        self.native_system_end_turn_ui = Some(Box::new(NativeSystemEndTurnUIState {
            source,
            dialogue: r#final,
            reward_message: Some(message),
            ..Default::default()
        }));
        self.native_system_end_turn_confirm = false;
        self.ring = false;
        self.msg = String::new();
        self.reset_native_class_ui_pulse();
        // Go 讓 `message.opening` 指向同一個 `nativeClassUIJob`；那個欄位只給 HD 繪製比對（R6），Rust 不另存複本。
        self.native_class_ui_job = Some(NativeClassUIJob {
            frames: opening,
            after: Some(Box::new(|g: &mut Game| {
                if let Some(message) =
                    g.native_system_end_turn_ui.as_mut().and_then(|state| state.reward_message.as_mut())
                {
                    message.await_ack = true;
                }
            })),
            ..Default::default()
        });
        Ok(())
    }

    /// acknowledgeNativeDeathRewardMessage 是 0x16C57(0) 收到鍵之後的 0x196CB 關框；
    /// 型態 1 的金額在關框之後才加進 [0x53BF3]，關完才回到行動收尾（then）。
    ///
    /// Go：`native_death_reward_message.go:151`，`func (g *Game) acknowledgeNativeDeathRewardMessage()`
    pub(crate) fn acknowledge_native_death_reward_message(&mut self) {
        let Some(state) = self.native_system_end_turn_ui.as_mut() else {
            return;
        };
        let Some(message) = state.reward_message.as_mut().filter(|m| m.await_ack) else {
            return;
        };
        message.await_ack = false;
        let closing = campaign::native_class_list_closing_frames(&state.source, &message.r#final);
        let closing = match closing {
            Ok(closing) if closing.len() == 5 => closing,
            _ => {
                self.load_err = String::from("native death reward message: dialogue close frames unavailable");
                return;
            }
        };
        let restore = state.source.clone();
        // Go 的 `message.closing` 同樣只給 HD 繪製（見 begin）；不另存。
        self.native_class_ui_job = Some(NativeClassUIJob {
            frames: closing,
            restore,
            redraw_native_map_after_restore: true,
            after: Some(Box::new(|g: &mut Game| {
                // Go 的閉包捕捉 `message` 指標；Rust 在關框結束時才從 `nativeSystemEndTurnUI` 取出同一則訊息。
                let state = g.native_system_end_turn_ui.take();
                g.native_system_end_turn_delay = 0;
                let Some(mut message) = state.and_then(|state| state.reward_message) else {
                    return;
                };
                if message.kind == 1 {
                    // 0x1ABF8..0x1ABFD：關框之後才 `add [0x53BF3], [0x53AE1]`。
                    g.gold += message.value;
                }
                if let Some(then) = message.then.take() {
                    then(g);
                }
            })),
            ..Default::default()
        });
    }

    /// Go：`native_death_reward_message.go:177`，`func (g *Game) nativeDeathRewardMessageAwaitingKey() bool`
    pub(crate) fn native_death_reward_message_awaiting_key(&self) -> bool {
        self.native_system_end_turn_ui
            .as_ref()
            .and_then(|state| state.reward_message.as_ref())
            .is_some_and(|message| message.await_ack)
    }
}
