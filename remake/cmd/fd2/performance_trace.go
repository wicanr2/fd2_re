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
	var rendered, reused, mapRendered, mapReused uint64
	var mapRaster, battleRaster [2]int
	var posterRendered, posterReused uint64
	quality := "unknown"
	var graphics ebiten.DebugInfo
	ebiten.ReadDebugInfo(&graphics)
	if p.modernHDPresentation != nil && p.Game != nil {
		quality = p.Game.playerHDQuality.name()
		posterRendered, posterReused = p.poster.cleanRendered, p.poster.cleanReused
	}
	if p.modernHDPresentation != nil && p.Game != nil && p.modernHDBattleGPU != nil {
		rendered, reused = p.modernHDBattleGPU.rendered, p.modernHDBattleGPU.reused
		if world := p.modernHDBattleGPU.World; world != nil {
			battleRaster = [2]int{world.Bounds().Dx(), world.Bounds().Dy()}
		}
	}
	if p.modernHDPresentation != nil && p.Game != nil && p.modernHDMapGPU != nil {
		mapRendered, mapReused = p.modernHDMapGPU.rendered, p.modernHDMapGPU.reused
		if world := p.modernHDMapGPU.World; world != nil {
			mapRaster = [2]int{world.Bounds().Dx(), world.Bounds().Dy()}
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
		BattleRendered  uint64         `json:"battle_gpu_rendered_total"`
		BattleReused    uint64         `json:"battle_gpu_reused_total"`
		MapRendered     uint64         `json:"map_gpu_rendered_total"`
		MapReused       uint64         `json:"map_gpu_reused_total"`
		GraphicsLibrary string         `json:"graphics_library"`
		TextureQuality  string         `json:"hd_texture_quality"`
		MapRaster       [2]int         `json:"map_raster_size"`
		BattleRaster    [2]int         `json:"battle_raster_size"`
		PosterRendered  uint64         `json:"poster_background_rendered_total"`
		PosterReused    uint64         `json:"poster_background_reused_total"`
	}{now.Sub(p.started).Seconds(), interval, ebiten.ActualFPS(), ebiten.ActualTPS(),
		p.updates, p.draws, p.finals, p.scenes, final, cpuFinished, rendered, reused, mapRendered, mapReused, graphics.GraphicsLibrary.String(), quality, mapRaster, battleRaster, posterRendered, posterReused})
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
	g := p.modernHDPresentation.Game
	profileScene := g.atk != nil
	if os.Getenv("FD2_PERFORMANCE_CPU_PROFILE_SCENE") == "world" {
		profileScene = g.modernHDWorldVisible && g.atk == nil
	}
	if !p.cpuAttempted && profileScene {
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
