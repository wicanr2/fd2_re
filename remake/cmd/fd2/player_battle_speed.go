package main

import "time"

// Zero keeps old settings and F8 behaviour: follow the general game speed.
func normalizeBattleAnimationSpeed(speed int) int {
	if speed < 0 || speed > 4 {
		return 0
	}
	return speed
}

func (g *Game) battleAnimationActive() bool {
	return g != nil && (g.atk != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.nativeAICommandModifier != nil || g.nativeAIItemPresentation != nil ||
		g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil ||
		g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9Player != nil ||
		g.nativeCmd9AIPresentation != nil || g.nativeCmd1012 != nil || g.nativeCmd24Presentation != nil ||
		g.nativeCmd29Presentation != nil || g.nativeCmd32Presentation != nil || g.nativeCmd33Presentation != nil ||
		g.nativeCmd34Presentation != nil || g.nativeCmd35Presentation != nil ||
		g.nativeCh28PostPresent != nil && (g.nativeCh28PostPresent.kind == "action" || g.nativeCh28PostPresent.kind == "status"))
}

func (g *Game) effectivePlayerSpeed() int {
	if g == nil {
		return 1
	}
	if speed := normalizeBattleAnimationSpeed(g.playerBattleSpeed); speed != 0 && g.battleAnimationActive() {
		return speed
	}
	return normalizePlayerSpeed(g.playerSpeed)
}

func (g *Game) syncPlayerClockSpeedAt(now time.Time) {
	speed := g.effectivePlayerSpeed()
	if g.playerClock.speed != speed {
		g.playerClock.sample(now)
		g.playerClock.speed = speed
	}
}

func (g *Game) setBattleAnimationSpeed(speed int) {
	g.playerBattleSpeed = normalizeBattleAnimationSpeed(speed)
	g.syncPlayerClockSpeedAt(time.Now())
	configured := loadSettings()
	configured.BattleAnimationSpeed = g.playerBattleSpeed
	saveSettings(configured)
}

func (g *Game) chooseGeneralSpeed(speed int) {
	g.setPlayerSpeedAt(speed, time.Now())
	configured := loadSettings()
	configured.GameSpeed = g.playerSpeed
	saveSettings(configured)
}
