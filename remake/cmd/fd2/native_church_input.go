package main

import "github.com/wicanr2/fd2_re/remake/internal/campaign"

// nativeChurchMenuInput is the single typed consumer input for the verified
// 0x2D7BD selection loop and 0x3072F four-service dispatch.
type nativeChurchMenuInput struct {
	delta  int
	enter  bool
	escape bool
}

type nativeChurchStatusInput struct {
	delta  int
	enter  bool
	escape bool
}

func (g *Game) handleNativeChurchMenuInput(input nativeChurchMenuInput) bool {
	if g.churchMode != "menu" || g.nativeChurchUIBlocksInput() {
		return false
	}
	if input.delta != 0 {
		g.churchSel = campaign.AdvanceNativeChurchServiceSelection(g.churchSel, input.delta)
		g.resetNativeChurchUIPulse()
		return true
	}
	if input.escape {
		if !g.beginNativeChurchMenuClosing(g.leaveChurch) {
			g.leaveChurch()
		}
		return true
	}
	if !input.enter {
		return true
	}

	selected := g.churchSel
	openService := func() {
		switch selected {
		case 0: // 0x2FFA5 caller-owned roster → 0x17AED(actor)
			g.churchMode = "status_roster"
			g.churchIDs = g.churchRosterIDs()
			g.churchSel = 0
			g.churchRosterStart = 0
			g.beginNativeChurchRosterOpening()
		case 1: // 0x2F8EA raw source→destination inventory transfer
			g.churchMode = "transfer_source"
			g.churchIDs = g.churchTransferSourceIDs()
			g.churchSel = 0
			g.churchRosterStart = 0
			g.nativeChurchTextIndex = 512
			g.beginNativeChurchRosterOpening()
		case 2, 3: // 0x30DC3 revive / 0x31385 class-change
			g.churchMode = map[int]string{2: "revive", 3: "class"}[selected]
			g.churchIDs = g.churchCandidates(g.churchMode)
			g.churchSel = 0
			g.churchVerticalStart = 0
			if g.churchMode == "class" {
				g.beginNativeClassListOpening()
			} else if len(g.churchIDs) == 0 {
				g.openNativeChurchReviveEmpty()
			} else {
				g.nativeChurchTextIndex = 589
				g.beginNativeChurchReviveListOpening()
			}
		}
	}
	if !g.beginNativeChurchMenuClosing(openService) {
		openService()
	}
	return true
}

func (g *Game) handleNativeChurchStatusInput(input nativeChurchStatusInput) bool {
	if g.nativeChurchUIBlocksInput() || g.nativeClassUIBlocksInput() {
		return false
	}
	switch g.churchMode {
	case "status_roster":
		listLen := len(g.churchIDs)
		if input.delta != 0 {
			g.churchSel = campaign.AdvanceNativeTwoColumnSelection(
				g.churchSel, listLen, input.delta,
			)
		}
		g.churchRosterStart, _ = campaign.NativeTwoColumnWindow(
			listLen, g.churchSel, g.churchRosterStart,
		)
		if input.escape {
			if !g.beginNativeChurchRosterClosing(g.returnToNativeChurchMenu) {
				g.returnToNativeChurchMenu()
			}
			return true
		}
		if !input.enter {
			return true
		}
		if listLen == 0 || g.churchSel < 0 || g.churchSel >= listLen {
			return true
		}
		id := g.churchIDs[g.churchSel]
		openStatus := func() {
			if !g.beginNativeChurchStatus(id) {
				g.msg = "角色缺少原版 status/command panel provenance"
				g.returnToNativeStatusRoster()
			}
		}
		if !g.beginNativeChurchRosterClosing(openStatus) {
			openStatus()
		}
		return true
	case "status_view", "status_commands":
		if !input.enter && !input.escape {
			return true
		}
		if g.churchMode == "status_view" && len(g.churchCommandPanel) != 0 {
			if !g.beginNativeChurchStatusCommandTransition() {
				g.closeNativeChurchStatus(g.churchStatusPanel)
			}
			return true
		}
		panel := g.churchStatusPanel
		if g.churchMode == "status_commands" {
			panel = g.churchCommandPanel
		}
		g.closeNativeChurchStatus(panel)
		return true
	default:
		return false
	}
}
