//! Go 來源：`remake/cmd/fd2/native_preparation_input.go`（194 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativePreparationInput 是鍵盤與決定性玩家路徑共用的整備輸入事件。
/// 它只描述單次按鍵邊界，不直接攜帶節點、名冊或交易結果。
///
/// Go：`native_preparation_input.go:7`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativePreparationInput {
    pub(crate) left: bool,
    pub(crate) right: bool,
    pub(crate) up: bool,
    pub(crate) down: bool,
    pub(crate) enter: bool,
    pub(crate) escape: bool,
}

/// Go `native_preparation_input.go:59` 的區域閉包 `leavePreparation`。
fn leave_preparation(g: &mut Game, outcome: &str) {
    // Go 直接解參考 `g.camp`（`handleNativePreparationInput` 開頭已確認它存在）。
    if !g.camp.as_mut().expect("leavePreparation: g.camp is nil").advance(outcome).is_empty() {
        g.native_town_hub_return = outcome == "cancel";
        g.enter_node();
    }
}

impl Game {
    /// Go `native_preparation_input.go:116` 的區域閉包 `closeThen`：收合失敗時同步執行同一個延續。
    ///
    /// Go 傳同一個 `after` 給收合函式，失敗時再自己呼叫；Rust 的延續移入收合函式就被丟棄，所以收
    /// 建立延續的 `make`，失敗時重建一份再呼叫。`make` 只建立閉包、不改狀態，結果與 Go 相同。
    fn close_preparation_prompt_then(&mut self, make: impl Fn() -> Box<dyn FnOnce(&mut Game) + Send>) {
        if !self.begin_native_preparation_prompt_closing(Some(make())) {
            make()(self);
        }
    }

    /// handleNativePreparationInput 擁有 preparation prompt、選人與最終確認三態。
    /// 所有 campaign mutation 都延後到既有 indexed closing／restore continuation；
    /// 深層畫面不可用時則沿既有失敗即關閉後備邊界同步執行相同 continuation。
    ///
    /// Go：`native_preparation_input.go:17`，`func (g *Game) handleNativePreparationInput(input nativePreparationInput) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn handle_native_preparation_input(&mut self, input: NativePreparationInput) -> bool {
        let Some(camp) = self.camp.as_ref() else {
            return false;
        };
        let Some(n) = camp.node() else {
            return false;
        };
        if n.r#type != "preparation" {
            return false;
        }
        let town_backed = !n.cancel.is_empty();
        if self.native_class_ui_blocks_input() || self.native_town_departure.is_some() {
            return true;
        }
        if self.prep_record_slots {
            todo!(
                "R8: native_preparation_input.go:29（獨立整備的記錄槽：restartPreparationSelection、composeNativePreparationRecordSlotsFrame）"
            );
        }

        if self.prep_required_missing.is_some() {
            if input.enter || input.escape {
                todo!("R8: native_preparation_input.go:68（必要角色提示：beginNativePreparationRequiredClosing）");
            }
            return true;
        }

        if self.prep_confirm {
            if input.left || input.right {
                self.prep_confirm_sel ^= 1;
                self.reset_native_class_ui_pulse();
            }
            if input.escape || input.enter {
                todo!(
                    "R8: native_preparation_input.go:96（出擊名單確認：beginNativePreparationConfirmationClosing、confirmPreparationDeparture）"
                );
            }
            return true;
        }

        if !self.prep_selecting {
            if input.left || input.right {
                self.prep_confirm_sel ^= 1;
                self.reset_native_class_ui_pulse();
            }
            if input.escape {
                if town_backed {
                    self.close_preparation_prompt_then(|| {
                        Box::new(|g: &mut Game| {
                            g.prep_prompt_source = Vec::new();
                            leave_preparation(g, "cancel");
                        })
                    });
                } else {
                    todo!("R8: native_preparation_input.go:132（獨立整備 ESC：restartPreparationSelection）");
                }
                return true;
            }
            if input.enter {
                if !town_backed {
                    todo!(
                        "R8: native_preparation_input.go:137（獨立整備：beginNativePreparationRecordSlots、restartPreparationSelection）"
                    );
                } else if self.prep_confirm_sel != 0 {
                    self.close_preparation_prompt_then(|| {
                        Box::new(|g: &mut Game| {
                            g.prep_prompt_source = Vec::new();
                            leave_preparation(g, "cancel");
                        })
                    });
                } else {
                    todo!(
                        "R8: native_preparation_input.go:151（出發：acceptTownDeparturePrompt、finishNativePreparationDeparture）"
                    );
                }
            }
            return true;
        }

        let count = self.prep_ids.len() as i64;
        let move_preparation = |g: &mut Game, scan_code: u8| {
            if let Ok(next) = fdother::move_native_preparation_roster_cursor(g.prep_sel, count, scan_code) {
                g.prep_sel = next;
            }
        };
        if input.left {
            move_preparation(self, 0x4b);
        }
        if input.right {
            move_preparation(self, 0x4d);
        }
        if input.up {
            move_preparation(self, 0x48);
        }
        if input.down {
            move_preparation(self, 0x50);
        }
        if input.enter {
            todo!("R8: main.go:togglePreparationSelection（選人）");
        }
        if input.escape {
            if town_backed {
                leave_preparation(self, "cancel");
            } else {
                todo!("R8: main.go:restartPreparationSelection（獨立整備 ESC）");
            }
        }
        true
    }
}

#[cfg(test)]
mod native_preparation_input_test;
