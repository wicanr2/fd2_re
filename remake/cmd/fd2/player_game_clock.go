package main

import "time"

type playerGameClock struct {
	wall, game time.Time
	speed      int
}

func normalizePlayerSpeed(speed int) int {
	if speed < 1 || speed > 4 {
		return 1
	}
	return speed
}

func (c *playerGameClock) sample(now time.Time) time.Time {
	if c.wall.IsZero() {
		c.wall, c.game = now, now
		return c.game
	}
	delta := now.Sub(c.wall)
	if delta < 0 {
		return c.game
	}
	c.game = c.game.Add(delta * time.Duration(normalizePlayerSpeed(c.speed)))
	c.wall = now
	return c.game
}

func (g *Game) gameNow() time.Time {
	return g.playerClock.sample(time.Now())
}

func (g *Game) setPlayerSpeedAt(speed int, now time.Time) {
	g.playerClock.sample(now)
	g.playerSpeed = normalizePlayerSpeed(speed)
	g.playerClock.speed = g.playerSpeed
}

func (g *Game) cyclePlayerSpeed() {
	next := normalizePlayerSpeed(g.playerSpeed) + 1
	if next > 4 {
		next = 1
	}
	g.setPlayerSpeedAt(next, time.Now())
	s := loadSettings()
	s.GameSpeed = g.playerSpeed
	saveSettings(s)
}
