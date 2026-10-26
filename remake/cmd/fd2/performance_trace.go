package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// performanceTrace observes the existing player without changing its clock.
// All optional Ebitengine layout methods remain promoted from the player.
type performanceTrace struct {
	*modernHDPresentation
	file                   *os.File
	encoder                *json.Encoder
	started, previous      time.Time
	updates, draws, finals int
	scenes                 map[string]int
	err                    error
	cpuAttempted           bool
	cpu                    *performanceCPUProfile
}

var _ ebiten.Game = (*performanceTrace)(nil)
var _ ebiten.FinalScreenDrawer = (*performanceTrace)(nil)

func openPerformanceTrace(player *modernHDPresentation, path string) (*performanceTrace, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("效能觀測輸出：%w", err)
	}
	now := time.Now()
	return &performanceTrace{modernHDPresentation: player, file: file, encoder: json.NewEncoder(file),
		started: now, previous: now, scenes: make(map[string]int)}, nil
}

func (p *performanceTrace) sample(now time.Time, final bool) {
	interval := now.Sub(p.previous).Seconds()
	if !final && interval < 1 || p.err != nil {
		return
	}
	cpuFinished := false
	if p.cpu != nil {
		select {
		case <-p.cpu.done:
			cpuFinished = true
		default:
		}
	}
	p.err = p.encoder.Encode(struct {
		ElapsedSeconds  float64        `json:"elapsed_seconds"`
		IntervalSeconds float64        `json:"interval_seconds"`
		ActualFPS       float64        `json:"ebiten_actual_fps"`
		ActualTPS       float64        `json:"ebiten_actual_tps"`
		Updates         int            `json:"updates"`
		Draws           int            `json:"draws"`
		FinalDraws      int            `json:"final_draws"`
		Scenes          map[string]int `json:"draw_scenes"`
		Final           bool           `json:"final"`
		CPUFinished     bool           `json:"cpu_profile_finished,omitempty"`
	}{now.Sub(p.started).Seconds(), interval, ebiten.ActualFPS(), ebiten.ActualTPS(),
		p.updates, p.draws, p.finals, p.scenes, final, cpuFinished})
	p.previous = now
	p.updates, p.draws, p.finals = 0, 0, 0
	p.scenes = make(map[string]int)
}

func (p *performanceTrace) Update() error {
	if p.err != nil {
		return fmt.Errorf("效能觀測寫入：%w", p.err)
	}
	err := p.modernHDPresentation.Update()
	p.updates++
	p.sample(time.Now(), false)
	return err
}

func (p *performanceTrace) Draw(screen *ebiten.Image) {
	if !p.cpuAttempted && p.modernHDPresentation.Game.atk != nil {
		p.cpuAttempted = true
		if path := os.Getenv("FD2_PERFORMANCE_CPU_PROFILE"); path != "" {
			p.cpu, p.err = startPerformanceCPUProfile(path, 30*time.Second)
			if p.err != nil {
				return
			}
		}
	}
	p.modernHDPresentation.Draw(screen)
	p.draws++
	g := p.modernHDPresentation.Game
	scene := "world-or-ui"
	if g.atk != nil {
		scene = "physical"
	} else if g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil {
		scene = "split-command"
	} else if g.nativeCmd32Presentation != nil || g.nativeCmd33Presentation != nil ||
		g.nativeCmd34Presentation != nil || g.nativeCmd35Presentation != nil {
		scene = "compound-command"
	}
	p.scenes[scene]++
}

func (p *performanceTrace) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.modernHDPresentation.DrawFinalScreen(screen, offscreen, geometry)
	p.finals++
}

func (p *performanceTrace) Close() error {
	var cpuErr error
	if p.cpu != nil {
		cpuErr = p.cpu.Close()
	}
	p.sample(time.Now(), true)
	err := p.file.Close()
	if p.err != nil {
		return p.err
	}
	if cpuErr != nil {
		return cpuErr
	}
	return err
}
