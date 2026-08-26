package main

import "github.com/wicanr2/fd2_re/remake/internal/campaign"

// nativeStoryInput 是鍵盤與決定性玩家路徑共用的故事輸入事件。
// 它只保存單次確認鍵邊界，不提供直接清除對白或改寫節點的捷徑。
type nativeStoryInput struct {
	enter bool
}

// handleNativeStoryInput 擁有一般 story 與處理器 cutscene 的確認鍵消費。
// 原生對話必須完成逐字與已證實收框後，才可交還既有 beat continuation。
func (g *Game) handleNativeStoryInput(n *campaign.Node, input nativeStoryInput) bool {
	if g == nil || g.camp == nil || n == nil {
		return false
	}
	switch n.Type {
	case "story":
		if input.enter && g.fade == nil && len(g.storyWalks) == 0 {
			if g.dlgAdvance() && len(g.dialog) == 0 {
				g.advanceStoryNode(n)
			}
		}
		return true
	case "cutscene":
		if g.approximatePostbattle {
			if input.enter {
				g.continueApproximatePostbattle()
			}
			return true
		}
		if input.enter && len(g.dialog) > 0 && g.nativeDialogueClosingLive {
			return true
		}
		if input.enter && len(g.dialog) > 0 {
			current := g.dialog[len(g.dialog)-1]
			if current.NativeDialogue != nil && g.dlgPage+1 >= dlgPageCount(current) &&
				g.dlgPage >= 0 && g.dlgPage < len(g.nativeDialogueProgressive) &&
				len(g.nativeDialogueProgressive[g.dlgPage]) > 0 &&
				g.nativeDialogueProgress >= len(g.nativeDialogueProgressive[g.dlgPage])-1 {
				if !g.beginNativeStoryDialogueClosing() {
					g.loadErr = "native story dialogue: verified closing frames are unavailable"
				}
				return true
			}
			if g.dlgAdvance() && len(g.dialog) == 0 {
				g.beatAdvance()
			}
		}
		return true
	default:
		return false
	}
}
