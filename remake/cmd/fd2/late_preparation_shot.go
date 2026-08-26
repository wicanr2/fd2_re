package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"
)

const reviewedLateChapterSlotSaveSHA256 = "f46d9c54d3037f84f05d72714569c282e63f39bf125251a9cf5cd9593ff3241f"

// prepareReviewedLatePreparationShot owns the bounded screenshot route from
// the reviewed native slot through the production title LOAD and preparation
// owners. It never runs without explicit screenshot mode and exact provenance.
func (g *Game) prepareReviewedLatePreparationShot() error {
	if g == nil || g.shotPath == "" || os.Getenv("FD2_SHOT_LATE_PREPARATION") != "1" {
		return fmt.Errorf("晚期整備截圖需要 FD2_SHOT 與 FD2_SHOT_LATE_PREPARATION=1")
	}
	path := os.Getenv("FD2_NATIVE_SAVE")
	stored, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("晚期整備截圖讀取原版槽：%w", err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(stored)); got != reviewedLateChapterSlotSaveSHA256 {
		return fmt.Errorf("晚期整備截圖存檔 SHA-256=%s，不是已審查來源", got)
	}
	if g.camp == nil || g.camp.C == nil {
		return fmt.Errorf("晚期整備截圖的正式戰役圖尚未載入")
	}

	g.titlePhase, g.titleSlotSel = "loadslots", 0
	if !g.confirmTitleLoadSlot(0) {
		return fmt.Errorf("晚期整備截圖的正式 LOAD 被拒絕：%s", g.msg)
	}
	if g.camp.NodeID() != "preparation_ch30" || g.gold != 60 ||
		len(g.partyRoster) != 29 || len(g.partyJoinOrder) != 29 {
		return fmt.Errorf("晚期整備截圖 LOAD 邊界不符：node=%q gold=%d roster=%d order=%d",
			g.camp.NodeID(), g.gold, len(g.partyRoster), len(g.partyJoinOrder))
	}
	if g.acceptTownDeparturePrompt() || !g.prepSelecting || g.prepSel != 0 ||
		g.prepLimit != 19 || g.preparationSelected() != 0 {
		return fmt.Errorf("晚期整備截圖選人邊界不符：selecting=%v cursor=%d limit=%d selected=%d",
			g.prepSelecting, g.prepSel, g.prepLimit, g.preparationSelected())
	}
	if raw := os.Getenv("FD2_SHOT_LATE_PREPARATION_PHASE"); raw != "" {
		phase, err := strconv.Atoi(raw)
		if err != nil || phase < 0 || phase > 2 {
			return fmt.Errorf("晚期整備截圖相位 %q 不在 0..2", raw)
		}
		g.prepIdleCycle = phase
	}
	return nil
}
