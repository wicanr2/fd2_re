package main

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

const chapterGuideRows = 7

var chapterGuideDetails = image.Rect(424, 44, 574, 69)
var chapterGuidePrevious = image.Rect(66, 324, 166, 350)
var chapterGuideNext = image.Rect(172, 324, 272, 350)
var chapterGuideClose = image.Rect(474, 324, 574, 350)

func (g *Game) chapterToolbarKind() string {
	if g.titlePhase != "" || g.camp == nil || g.camp.C == nil || g.loadErr != "" {
		return ""
	}
	node := g.camp.Node()
	if node == nil {
		return ""
	}
	if node.Type == "battle" && g.st != nil && g.sc != nil {
		return "battle"
	}
	if node.Type == "town" && node.NativeSecretGate != nil && g.fade == nil && g.nativeTownDeparture == nil {
		shop := g.camp.C.Nodes[node.NativeSecretGate.To]
		if shop != nil && shop.Type == "shop" {
			return "town"
		}
	}
	return ""
}

func (p *modernHDPresentation) chapterGuideLines() []string {
	sc := p.sc
	if sc == nil {
		return nil
	}
	fired := make(map[int]bool)
	for _, event := range sc.FiredEvents() {
		fired[event.Index] = true
	}
	rows := make([]string, 0, len(sc.Events))
	for index, event := range sc.Events {
		var kind string
		switch event.Trigger {
		case "on_battle_start":
			kind = "戰鬥開始"
		case "on_turn_end":
			kind = "回合事件"
		case "on_unit_death":
			kind = "擊倒事件"
		default:
			kind = "戰場事件"
		}
		var details []string
		groups := make(map[int]bool)
		if p.chapterGuideDetail && event.When != nil {
			if event.When.Turn > 0 {
				details = append(details, fmt.Sprintf("第 %d 回合", event.When.Turn))
			}
			if event.When.UnitDead != "" {
				details = append(details, event.When.UnitDead+"倒下")
			}
		}
		for _, action := range event.Do {
			if action.Type == "spawn_group" {
				kind = "增援登場"
				for _, group := range action.Groups {
					groups[group] = true
				}
			}
			if p.chapterGuideDetail {
				if action.Grid != nil {
					details = append(details, fmt.Sprintf("地圖格 (%d,%d)", action.Grid[0], action.Grid[1]))
				}
				if action.Type == "join_party" {
					details = append(details, "角色加入")
				}
			}
		}
		status := "待觸發"
		if fired[index] {
			status = "已觸發"
		} else if !event.Once {
			status = "可重複"
		}
		rows = append(rows, fmt.Sprintf("%s　%s", status, kind))
		for _, detail := range details {
			rows = append(rows, "　"+detail)
		}
		if p.chapterGuideDetail && p.st != nil {
			positions := make(map[string]bool)
			for _, units := range [][]*battle.Unit{p.st.Units, p.st.Roster} {
				for _, unit := range units {
					if unit != nil && groups[unit.Group] && unit.HasNativePositionRecord {
						pos := unit.NativePositionRecord
						line := fmt.Sprintf("　%s　預定登場格 (%d,%d)", unit.ClsName, pos.XWord, pos.YWord)
						if !positions[line] {
							rows = append(rows, line)
							positions[line] = true
						}
					}
				}
			}
		}
	}
	if p.chapterGuideDetail && p.st != nil {
		cells := make([]battle.Cell, 0, len(p.st.Treasures))
		for cell := range p.st.Treasures {
			cells = append(cells, cell)
		}
		sort.Slice(cells, func(i, j int) bool {
			return cells[i].Y < cells[j].Y || cells[i].Y == cells[j].Y && cells[i].X < cells[j].X
		})
		for _, cell := range cells {
			reward := p.st.Treasures[cell]
			var name string
			switch reward.Kind {
			case "gold":
				name = fmt.Sprintf("%d 金幣", reward.Value)
			case "item":
				if p.localeEntities != nil {
					name, _ = p.localeEntities.ItemName(reward.Value)
				}
			}
			if name != "" {
				rows = append(rows, fmt.Sprintf("寶物 (%d,%d)　%s", cell.X, cell.Y, name))
			}
		}
	}
	return rows
}

func (p *modernHDPresentation) updateChapterGuide() {
	if p.chapterToolbarKind() != "battle" || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.chapterGuideOpen = false
		return
	}
	var x, y int
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y = ebiten.CursorPosition()
	} else if taps := inpututil.AppendJustPressedTouchIDs(nil); len(taps) > 0 {
		x, y = ebiten.TouchPosition(taps[0])
	} else {
		return
	}
	point := image.Pt(x-p.posterOffset(), y-p.toolbarOffset())
	switch {
	case point.In(chapterGuideClose):
		p.chapterGuideOpen = false
	case point.In(chapterGuideDetails):
		p.chapterGuideDetail = !p.chapterGuideDetail
		p.chapterGuidePage = 0
	case point.In(chapterGuidePrevious) && p.chapterGuidePage > 0:
		p.chapterGuidePage--
	case point.In(chapterGuideNext) && (p.chapterGuidePage+1)*chapterGuideRows < len(p.chapterGuideLines()):
		p.chapterGuidePage++
	}
}

func (p *modernHDPresentation) drawChapterGuide(screen ebiten.FinalScreen, geometry ebiten.GeoM) {
	if !p.chapterGuideOpen || p.font == nil || p.sc == nil {
		return
	}
	if p.chapterGuideSurface == nil {
		p.chapterGuideSurface = ebiten.NewImage(logicalW*3, logicalH*3)
	}
	s := p.chapterGuideSurface
	s.Clear()
	vector.DrawFilledRect(s, 144, 114, 1632, 984, color.RGBA{17, 24, 37, 255}, false)
	ink := color.RGBA{245, 245, 245, 255}
	p.font.Draw(s, fmt.Sprintf("第 %d 章　關卡事件", p.sc.Chapter), 204, 126, 2.5, ink)
	detail := "顯示細節"
	if p.chapterGuideDetail {
		detail = "收起細節"
	}
	p.drawCompactToolbarButton(s, chapterGuideDetails, detail, p.chapterGuideDetail)
	goal := "目標：擊倒所有已上場的敵軍"
	if p.sc.NativeResultInitialCode != nil {
		goal = "目標：完成本關指定事件"
	}
	p.font.Draw(s, goal, 204, 266, 2, ink)
	hint := "提前清場可能略過未觸發的事件"
	if len(p.sc.NativeResultRules) > 0 {
		hint = "本關另有特殊結算條件"
	}
	p.font.Draw(s, hint, 204, 332, 2, ink)
	rows := p.chapterGuideLines()
	start := min(len(rows), p.chapterGuidePage*chapterGuideRows)
	for i, row := range rows[start:min(start+chapterGuideRows, len(rows))] {
		runes := []rune(row)
		if len(runes) > 36 {
			row = string(runes[:35]) + "…"
		}
		p.font.Draw(s, row, 204, float64(408+i*74), 2, ink)
	}
	if start > 0 {
		p.drawCompactToolbarButton(s, chapterGuidePrevious, "上一頁", false)
	}
	if start+chapterGuideRows < len(rows) {
		p.drawCompactToolbarButton(s, chapterGuideNext, "下一頁", false)
	}
	p.drawCompactToolbarButton(s, chapterGuideClose, "關閉", false)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/3, 1.0/3)
	op.GeoM.Concat(geometry)
	screen.DrawImage(s, op)
}
