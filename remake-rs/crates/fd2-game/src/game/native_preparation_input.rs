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
            if town_backed {
                self.load_err = "城鎮整備不可使用記錄槽".into();
            } else if input.escape {
                self.restart_preparation_selection();
            } else {
                let movement = i64::from(input.down) - i64::from(input.up);
                self.prep_record_slot_sel = (self.prep_record_slot_sel + movement).clamp(0, 3);
                if !self.compose_native_preparation_record_slots().1 {
                    self.load_err = "缺少整備記錄槽畫面".into();
                } else if input.enter {
                    if !native_current_save_path().is_empty()
                        && self.native_chapter_slot_baseline.is_none()
                        && !self.native_chapter_slot_started
                    {
                        self.load_err = "缺少整備記錄槽基底".into();
                    } else if let Err(err) = self.save_game_to_slot(self.prep_record_slot_sel) {
                        self.load_err = format!("整備記錄存檔：{}", err.message());
                    }
                }
            }
            return true;
        }

        if self.prep_required_missing.is_some() {
            if input.enter || input.escape {
                let after = Box::new(move |game: &mut Game| {
                    game.prep_required_missing = None;
                    if town_backed {
                        leave_preparation(game, "cancel");
                    } else {
                        game.prep_ids = game.party_join_order.iter().copied().skip(1).collect();
                        game.prep_sel = 0;
                        game.restart_preparation_selection();
                    }
                });
                if let Err(err) = self.begin_native_preparation_required(Some(after)) {
                    self.load_err = err.message().into();
                }
            }
            return true;
        }

        if self.prep_confirm {
            if input.left || input.right {
                self.prep_confirm_sel ^= 1;
                self.reset_native_class_ui_pulse();
            }
            if input.escape || input.enter {
                let confirmed = !input.escape && self.prep_confirm_sel == 0;
                let after = move |game: &mut Game| {
                    if confirmed {
                        game.finish_native_preparation_departure(Box::new(|game| leave_preparation(game, "confirm")));
                    } else if town_backed {
                        leave_preparation(game, "cancel");
                    } else {
                        game.restart_preparation_selection();
                    }
                };
                if self.begin_native_preparation_confirmation(Some(Box::new(after))).is_err() {
                    after(self);
                }
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
                    self.close_preparation_prompt_then(|| Box::new(Game::restart_preparation_selection));
                }
                return true;
            }
            if input.enter {
                if !town_backed {
                    self.close_preparation_prompt_then(|| {
                        Box::new(|game| {
                            if game.prep_confirm_sel == 0 {
                                game.prep_record_slot_sel = 0;
                                if game.compose_native_preparation_record_slots().1 {
                                    game.prep_record_slots = true;
                                } else {
                                    game.load_err = "缺少整備記錄槽畫面".into();
                                }
                            } else {
                                game.restart_preparation_selection();
                            }
                        })
                    });
                } else if self.prep_confirm_sel != 0 {
                    self.close_preparation_prompt_then(|| {
                        Box::new(|g: &mut Game| {
                            g.prep_prompt_source = Vec::new();
                            leave_preparation(g, "cancel");
                        })
                    });
                } else {
                    self.close_preparation_prompt_then(|| {
                        Box::new(|game| {
                            if game.prep_ids.len() as i64 > game.prep_limit {
                                game.restart_preparation_selection();
                            } else {
                                game.finish_native_preparation_departure(Box::new(|game| {
                                    game.prep_prompt_source.clear();
                                    leave_preparation(game, "confirm");
                                }));
                            }
                        })
                    });
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
        if input.enter
            && let Err(err) = self.toggle_preparation_selection()
        {
            self.load_err = err.message().into();
        }
        if input.escape {
            if town_backed {
                leave_preparation(self, "cancel");
            } else {
                self.restart_preparation_selection();
            }
        }
        true
    }

    pub(crate) fn restart_preparation_selection(&mut self) {
        self.prep_record_slots = false;
        self.prep_selecting = true;
        self.prep_confirm = false;
        self.prep_required_missing = None;
        self.prep_confirm_sel = 0;
        self.native_class_ui_job = None;
        self.reset_native_class_ui_pulse();
        self.party_deploy.clear();
    }

    fn toggle_preparation_selection(&mut self) -> Result<()> {
        let Some(&id) = self.prep_ids.get(self.prep_sel as usize) else {
            return Ok(());
        };
        let selected = self.party_deploy.entry(id).or_default();
        *selected = !*selected;
        let selected = *selected;
        if self.party_deploy.values().filter(|selected| **selected).count() as i64 == self.prep_limit {
            let node = self.camp.as_ref().and_then(|camp| camp.node()).ok_or_else(|| game_err!("缺少整備節點"))?;
            let mut required_missing = None;
            for &required in &node.required_party_identities {
                let mut found = false;
                for id in self.prep_ids.iter().filter(|id| self.party_deploy.get(id) == Some(&true)) {
                    let unit = self
                        .party_roster
                        .get(id)
                        .filter(|unit| unit.has_native_record_byte8)
                        .ok_or_else(|| game_err!("整備角色缺少身分原始記錄"))?;
                    found |= i64::from(unit.native_record_byte8) == required;
                }
                if !found {
                    required_missing = Some(required);
                    break;
                }
            }
            let mut order = self.preparation_packed_order()?;
            if required_missing.is_none() {
                self.front_preparation_order(&mut order, &node.party_front_identities)?;
            }
            self.reorder_preparation_roster(order)?;
            self.prep_selecting = false;
            self.prep_required_missing = required_missing;
            if required_missing.is_some() {
                self.begin_native_preparation_required(None)?;
            } else {
                self.prep_confirm = true;
                self.prep_confirm_sel = 0;
                self.begin_native_preparation_confirmation(None)?;
            }
        } else if selected && self.prep_sel + 1 < self.prep_ids.len() as i64 {
            self.prep_sel += 1;
        }
        Ok(())
    }

    fn preparation_packed_order(&self) -> Result<Vec<i64>> {
        let Some((&leader, rest)) = self.party_join_order.split_first() else {
            return Err(game_err!("整備缺少固定隊長"));
        };
        let mut order = vec![leader];
        for selected in [true, false] {
            order.extend(
                rest.iter().copied().filter(|id| self.party_deploy.get(id).copied().unwrap_or(false) == selected),
            );
        }
        Ok(order)
    }

    fn front_preparation_order(&self, order: &mut [i64], identities: &[i64]) -> Result<()> {
        let mut seen = BTreeSet::new();
        for &identity in identities {
            if !(1..=31).contains(&identity) || !seen.insert(identity) {
                return Err(game_err!("整備前置角色身分無效"));
            }
            let mut found = None;
            for (index, id) in order.iter().enumerate().skip(1) {
                let unit = self
                    .party_roster
                    .get(id)
                    .filter(|unit| unit.has_native_identity)
                    .ok_or_else(|| game_err!("整備角色缺少原版身分"))?;
                if unit.native_identity == identity {
                    if found.is_some() || self.party_deploy.get(id) != Some(&true) {
                        return Err(game_err!("整備前置角色重複或未選入"));
                    }
                    found = Some(index);
                }
            }
            let index = found.ok_or_else(|| game_err!("找不到整備前置角色"))?;
            order[1..=index].rotate_right(1);
        }
        Ok(())
    }
}

#[cfg(test)]
mod native_preparation_input_test;
