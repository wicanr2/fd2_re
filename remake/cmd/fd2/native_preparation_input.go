package main

import "github.com/wicanr2/fd2_re/remake/internal/fdother"

// nativePreparationInput 是鍵盤與決定性玩家路徑共用的整備輸入事件。
// 它只描述單次按鍵邊界，不直接攜帶節點、名冊或交易結果。
type nativePreparationInput struct {
	left, right bool
	up, down    bool
	enter       bool
	escape      bool
}

// handleNativePreparationInput 擁有 preparation prompt、選人與最終確認三態。
// 所有 campaign mutation 都延後到既有 indexed closing／restore continuation；
// 深層畫面不可用時則沿既有失敗即關閉後備邊界同步執行相同 continuation。
func (g *Game) handleNativePreparationInput(input nativePreparationInput) bool {
	if g == nil || g.camp == nil {
		return false
	}
	n := g.camp.Node()
	if n == nil || n.Type != "preparation" {
		return false
	}
	if g.nativeClassUIBlocksInput() {
		return true
	}
	townBacked := n.Cancel != ""
	if g.prepRecordSlots {
		if townBacked {
			g.loadErr = "preparation record slots have a town owner"
			return true
		}
		if input.escape {
			g.restartPreparationSelection()
			return true
		}
		if input.up && g.prepRecordSlotSel > 0 {
			g.prepRecordSlotSel--
		}
		if input.down && g.prepRecordSlotSel < 3 {
			g.prepRecordSlotSel++
		}
		if _, ok := g.composeNativePreparationRecordSlotsFrame(); !ok {
			g.loadErr = "preparation record slot source unavailable"
			return true
		}
		if input.enter {
			if nativeCurrentSavePath() != "" && g.nativeChapterSlotBaseline == nil {
				g.loadErr = "preparation record save baseline unavailable"
				return true
			}
			if err := g.saveGameToSlot(g.prepRecordSlotSel); err != nil {
				g.loadErr = "preparation record save failed: " + err.Error()
			}
		}
		return true
	}
	leavePreparation := func(outcome string) {
		if g.camp.Advance(outcome) != "" {
			g.nativeTownHubReturn = outcome == "cancel"
			g.enterNode()
		}
	}

	if g.prepRequiredMissing != nil {
		if input.enter || input.escape {
			if !g.beginNativePreparationRequiredClosing(func() {
				g.prepRequiredMissing = nil
				leavePreparation("cancel")
			}) {
				g.loadErr = "preparation required-character closing assets unavailable"
			}
		}
		return true
	}

	if g.prepConfirm {
		closeThen := func(after func()) {
			if !g.beginNativePreparationConfirmationClosing(after) {
				after()
			}
		}
		if input.left || input.right {
			g.prepConfirmSel ^= 1
			g.resetNativeClassUIPulse()
		}
		if input.escape {
			if townBacked {
				closeThen(func() { leavePreparation("cancel") })
			} else {
				closeThen(g.restartPreparationSelection)
			}
		}
		if input.enter {
			if g.prepConfirmSel == 0 {
				g.confirmPreparationDeparture()
			} else if townBacked {
				closeThen(func() { leavePreparation("cancel") })
			} else {
				closeThen(g.restartPreparationSelection)
			}
		}
		return true
	}

	if !g.prepSelecting {
		closeThen := func(after func()) {
			if !g.beginNativePreparationPromptClosing(after) {
				after()
			}
		}
		if input.left || input.right {
			g.prepConfirmSel ^= 1
			g.resetNativeClassUIPulse()
		}
		if input.escape {
			if townBacked {
				closeThen(func() {
					g.prepPromptSource = nil
					leavePreparation("cancel")
				})
			} else {
				closeThen(g.restartPreparationSelection)
			}
			return true
		}
		if input.enter {
			if !townBacked {
				closeThen(func() {
					if g.prepConfirmSel == 0 {
						g.beginNativePreparationRecordSlots()
					} else {
						g.restartPreparationSelection()
					}
				})
			} else if g.prepConfirmSel != 0 {
				closeThen(func() {
					g.prepPromptSource = nil
					leavePreparation("cancel")
				})
			} else {
				closeThen(func() {
					if g.acceptTownDeparturePrompt() {
						g.prepPromptSource = nil
						leavePreparation("confirm")
					}
				})
			}
		}
		return true
	}

	movePreparation := func(scanCode byte) {
		if next, err := fdother.MoveNativePreparationRosterCursor(
			g.prepSel, len(g.prepIDs), scanCode,
		); err == nil {
			g.prepSel = next
		}
	}
	if input.left {
		movePreparation(0x4b)
	}
	if input.right {
		movePreparation(0x4d)
	}
	if input.up {
		movePreparation(0x48)
	}
	if input.down {
		movePreparation(0x50)
	}
	if input.enter {
		g.togglePreparationSelection()
	}
	if input.escape {
		if townBacked {
			leavePreparation("cancel")
		} else {
			g.restartPreparationSelection()
		}
	}
	return true
}
