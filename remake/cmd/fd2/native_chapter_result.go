package main

import "github.com/wicanr2/fd2_re/remake/internal/battle"

// nativeChapterResultJob 保留原版同步處理器在非同步介面裡的擁有權：
// 當前列對白完全收框之後，才讀取下一列的raw條件。
type nativeChapterResultJob struct {
	nextRule int
	code     int
	matched  []string
}

func (g *Game) beginNativeChapterResult() {
	if g.nativeChapterResult != nil || g.battleEvent != nil || g.loadErr != "" {
		return
	}
	code, err := battle.NativeDefaultResultCode(g.st)
	if err != nil {
		g.failNativeChapterResult(err.Error())
		return
	}
	g.nativeChapterResult = &nativeChapterResultJob{code: code}
	g.advanceNativeChapterResult()
}

func (g *Game) failNativeChapterResult(message string) {
	g.loadErr = "原生章節結果：" + message
	g.startupBlocked = true
}

func (g *Game) advanceNativeChapterResult() {
	j := g.nativeChapterResult
	if j == nil || g.sc == nil || g.st == nil || g.loadErr != "" {
		return
	}
	for j.nextRule < len(g.sc.NativeResultRules) {
		rule := g.sc.NativeResultRules[j.nextRule]
		j.nextRule++
		matched, err := rule.Match(g.st)
		if err != nil {
			g.failNativeChapterResult(rule.ID + "：" + err.Error())
			return
		}
		if !matched {
			continue
		}
		if !g.nativeBattleDialogueAvailable() {
			g.failNativeChapterResult(rule.ID + "：缺少原生戰場對白狀態")
			return
		}
		j.code = rule.Code
		j.matched = append(j.matched, rule.ID)
		g.startBattleEvent(rule.Actions, g.advanceNativeChapterResult)
		return
	}
	// 對白與所有條件都完成後才把pending交給外層結果／敗北擁有者。
	g.nativeResultMatchedRules = append([]string(nil), j.matched...)
	g.nativeChapterResult = nil
	switch j.code {
	case 0:
		g.result = ""
	case 1:
		g.result = "lose"
	case 2:
		g.result = "win"
	}
	g.beginNativeDefeatIfConfigured()
}
