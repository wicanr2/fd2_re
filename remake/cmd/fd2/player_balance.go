package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

type balanceEditor struct {
	names                            map[int]string
	open                             bool
	ack, tab, person, command, field int
	all                              bool
	draft                            *battle.BalanceOptions
	values                           [5]int
	input, message                   string
}

var playerBalanceButton = image.Rect(564, 3, 632, 21)

func (g *Game) balanceAllowed() bool {
	return g.playerPosterControlsAllowed() && g.titlePhase == "" && g.camp != nil &&
		g.walk == nil && g.nativePlayerStatus == nil && !g.ring && !g.nativeCommandOpen &&
		!g.aiBusy && g.actJob == nil && g.battleEvent == nil && g.camPan == nil && g.nativeTurnStaging == nil && g.nativeLevelUpDialogue == nil && g.nativeSystemEndTurnUI == nil && g.bannerT == 0 && g.actionOverlayPhase == ""
}

func (g *Game) openBalanceEditor() {
	if !g.balanceAllowed() {
		g.msg = "請在操作空檔開啟能力設定"
		return
	}
	g.balanceUI = balanceEditor{open: true, all: true, field: -1, draft: battle.NewBalanceOptions()}
	if g.balance != nil {
		raw, _ := json.Marshal(g.balance)
		_ = json.Unmarshal(raw, g.balanceUI.draft)
	}
	if catalog, err := campaign.LoadNativeCharacterCatalog(assetPath("assets/data/native_character_catalog.json")); err == nil {
		g.balanceUI.names = map[int]string{}
		for _, entry := range catalog.Identities {
			g.balanceUI.names[entry.ID] = entry.Name
		}
	}
	g.loadBalanceValues()
	// The toolbar consumes this click before ordinary Game.Update runs.
	g.recordInputAuditState()
}

func (g *Game) balanceIDs() []int {
	seen := map[int]bool{}
	var ids []int
	for id := range g.partyRoster {
		if id >= 0 && id < 32 {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if g.st != nil {
		for _, u := range g.st.Units {
			if id, ok := battle.BalanceIdentity(u); ok && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	sort.Ints(ids)
	return ids
}

func (g *Game) balanceUnit(id int) *battle.Unit {
	if g.st != nil {
		for _, u := range g.st.Units {
			if key, ok := battle.BalanceIdentity(u); ok && key == id {
				return u
			}
		}
	}
	if u, ok := g.partyRoster[id]; ok {
		return &u
	}
	return nil
}

func (g *Game) balanceSelected() (int, *battle.Unit) {
	ids := g.balanceIDs()
	if len(ids) == 0 {
		return -1, nil
	}
	g.balanceUI.person = (g.balanceUI.person%len(ids) + len(ids)) % len(ids)
	id := ids[g.balanceUI.person]
	return id, g.balanceUnit(id)
}

// A paused editor still reports received input. Inspect wrapped indices without
// changing selection, values, gameplay state or RNG.
func (g *Game) balanceAuditState() map[string]any {
	b := &g.balanceUI
	ids := g.balanceIDs()
	id, command := -1, -1
	if len(ids) != 0 {
		id = ids[(b.person%len(ids)+len(ids))%len(ids)]
		commands := g.balanceCommands(g.balanceUnit(id))
		if len(commands) != 0 {
			command = commands[(b.command%len(commands)+len(commands))%len(commands)]
		}
	}
	return map[string]any{"open": b.open, "ack": b.ack, "tab": b.tab,
		"person_index": b.person, "ids": ids, "identity": id, "command_id": command,
		"field": b.field, "values": b.values, "message": b.message}
}

func balanceRateValues(r battle.GrowthRates) [5]int { return [5]int{r.HP, r.MP, r.AP, r.DP, r.DX} }
func balanceValuesRate(v [5]int) battle.GrowthRates {
	return battle.GrowthRates{HP: v[0], MP: v[1], AP: v[2], DP: v[3], DX: v[4]}
}

func (g *Game) balanceCommands(u *battle.Unit) []int {
	var ids []int
	if u == nil {
		return ids
	}
	for id := 0; id < 36; id++ {
		if u.NativeCommandMask[id/8]&(1<<uint(id%8)) != 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func (g *Game) loadBalanceValues() {
	b := &g.balanceUI
	b.field = -1
	b.input = ""
	b.message = ""
	id, u := g.balanceSelected()
	switch b.tab {
	case 0:
		if u != nil {
			s := u.BalanceStats()
			b.values = [5]int{s.HP, s.MP, s.AP, s.DP, s.DX}
		}
	case 1:
		r := b.draft.Growth
		if !b.all {
			if v, ok := b.draft.Characters[id]; ok {
				r = v
			}
		}
		b.values = balanceRateValues(r)
	case 2:
		ids := g.balanceCommands(u)
		if len(ids) > 0 {
			b.command = (b.command%len(ids) + len(ids)) % len(ids)
			cmd := ids[b.command]
			n := 100
			if cmd < len(g.nativeCommandBook) {
				n = g.nativeCommandBook[cmd].Hit
			}
			if cmd == 22 || cmd == 26 || cmd == 27 {
				n = 50
			}
			if v, ok := b.draft.SkillHit[id][cmd]; ok {
				n = v
			}
			b.values[0] = n
		}
	case 3:
		b.values[0] = b.draft.EnemyPercent
	}
}

func (g *Game) applyBalanceEditor() {
	b := &g.balanceUI
	id, u := g.balanceSelected()
	switch b.tab {
	case 0:
		if u == nil {
			b.message = "目前沒有可編輯的我方角色"
			return
		}
		v := battle.BalanceStats{HP: b.values[0], MP: b.values[1], AP: b.values[2], DP: b.values[3], DX: b.values[4]}
		copy := cloneNativeShopUnit(*u)
		if err := copy.SetBalanceStats(v); err != nil {
			b.message = err.Error()
			return
		}
		if roster, exists := g.partyRoster[id]; exists {
			if err := roster.SetBalanceStats(v); err != nil {
				b.message = err.Error()
				return
			}
			g.partyRoster[id] = roster
		}
		*u = copy
	case 1:
		if b.all {
			b.draft.Growth = balanceValuesRate(b.values)
		} else {
			if id < 0 {
				return
			}
			if b.draft.Characters == nil {
				b.draft.Characters = map[int]battle.GrowthRates{}
			}
			b.draft.Characters[id] = balanceValuesRate(b.values)
		}
	case 2:
		ids := g.balanceCommands(u)
		if len(ids) == 0 {
			return
		}
		cmd := ids[b.command]
		if !battle.NativeCommandHasHitRoll(cmd) {
			b.message = "原本無命中骰，保留原傷害規則"
			return
		}
		if b.draft.SkillHit == nil {
			b.draft.SkillHit = map[int]map[int]int{}
		}
		if b.draft.SkillHit[id] == nil {
			b.draft.SkillHit[id] = map[int]int{}
		}
		b.draft.SkillHit[id][cmd] = b.values[0]
	case 3:
		b.draft.EnemyPercent = b.values[0]
	}
	if err := b.draft.Validate(); err != nil {
		b.message = err.Error()
		return
	}
	raw, _ := json.Marshal(b.draft)
	var options battle.BalanceOptions
	_ = json.Unmarshal(raw, &options)
	g.balance = &options
	if options.IsOriginal() {
		g.balance = nil
	}
	g.syncBalanceState()
	b.message = "已套用；請儲存遊戲以保留這份進度的設定"
}

func (g *Game) syncBalanceState() {
	if g.st != nil {
		g.st.Balance = g.balance
		g.st.ApplyEnemyStrength()
	}
}

func (g *Game) balanceFieldLimit() (int, int) {
	b := &g.balanceUI
	switch b.tab {
	case 0:
		if b.field == 0 {
			return 1, 32767
		}
		return 0, 32767
	case 1:
		return 0, 500
	case 2:
		return 0, 100
	default:
		return 10, 500
	}
}

func (g *Game) setBalanceNumber(value int) {
	b := &g.balanceUI
	if b.field < 0 {
		return
	}
	lo, hi := g.balanceFieldLimit()
	if value < lo {
		value = lo
	}
	if value > hi {
		value = hi
	}
	b.values[b.field] = value
}

func (g *Game) balancePoint(at image.Point) {
	b := &g.balanceUI
	if !b.open {
		return
	}
	if b.ack < 3 {
		if at.In(image.Rect(342, 252, 538, 281)) {
			b.ack++
		}
		if at.In(image.Rect(102, 252, 278, 281)) {
			*b = balanceEditor{}
		}
		return
	}
	if at.In(image.Rect(522, 43, 576, 68)) {
		*b = balanceEditor{}
		return
	}
	for i := 0; i < 4; i++ {
		if at.In(image.Rect(62+i*130, 77, 186+i*130, 100)) {
			b.tab = i
			g.loadBalanceValues()
			return
		}
	}
	if b.tab != 3 && at.In(image.Rect(62, 108, 94, 132)) {
		b.person--
		b.command = 0
		g.loadBalanceValues()
		return
	}
	if b.tab != 3 && at.In(image.Rect(400, 108, 432, 132)) {
		b.person++
		b.command = 0
		g.loadBalanceValues()
		return
	}
	if b.tab == 1 && at.In(image.Rect(440, 108, 576, 132)) {
		b.all = !b.all
		g.loadBalanceValues()
		return
	}
	if b.tab == 2 {
		if at.In(image.Rect(62, 143, 94, 167)) {
			b.command--
			g.loadBalanceValues()
			return
		}
		if at.In(image.Rect(540, 143, 576, 167)) {
			b.command++
			g.loadBalanceValues()
			return
		}
	}
	rows := 5
	start := 145
	if b.tab >= 2 {
		rows = 1
		start = 183
	}
	for i := 0; i < rows; i++ {
		y := start + i*25
		if at.In(image.Rect(390, y-2, 430, y+20)) {
			b.field = i
			b.input = ""
			g.setBalanceNumber(b.values[i] - 1)
			return
		}
		if at.In(image.Rect(530, y-2, 570, y+20)) {
			b.field = i
			b.input = ""
			g.setBalanceNumber(b.values[i] + 1)
			return
		}
		if at.In(image.Rect(438, y-2, 522, y+20)) {
			b.field = i
			b.input = ""
			return
		}
	}
	if at.In(image.Rect(62, 321, 214, 348)) {
		id, u := g.balanceSelected()
		switch b.tab {
		case 0:
			g.loadBalanceValues()
			return
		case 1:
			if b.all {
				b.draft.Growth = battle.OriginalGrowthRates()
			} else {
				delete(b.draft.Characters, id)
			}
		case 2:
			ids := g.balanceCommands(u)
			if len(ids) > 0 {
				delete(b.draft.SkillHit[id], ids[b.command])
				if len(b.draft.SkillHit[id]) == 0 {
					delete(b.draft.SkillHit, id)
				}
			}
		case 3:
			b.draft.EnemyPercent = 100
		}
		// A reset is an explicit apply, so deleting an override restores inheritance.
		raw, _ := json.Marshal(b.draft)
		var o battle.BalanceOptions
		_ = json.Unmarshal(raw, &o)
		g.balance = &o
		if o.IsOriginal() {
			g.balance = nil
		}
		g.syncBalanceState()
		g.loadBalanceValues()
		b.message = "已恢復原版或全體預設；請儲存遊戲"
		return
	}
	if at.In(image.Rect(434, 321, 576, 348)) {
		g.applyBalanceEditor()
	}
}

func (p *modernHDPresentation) updateBalanceEditor() bool {
	g := p.Game
	b := &g.balanceUI
	if !b.open {
		return false
	}
	defer g.recordInputAuditState()
	// Keep wall time from advancing the paused game clock.
	g.playerClock.wall = time.Now()
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		*b = balanceEditor{}
		return true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		g.balancePoint(image.Pt(x-p.posterOffset(), y-p.toolbarOffset()))
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		g.balancePoint(image.Pt(x-p.posterOffset(), y-p.toolbarOffset()))
	}
	if b.open && b.ack == 3 && b.field >= 0 {
		for _, r := range ebiten.AppendInputChars(nil) {
			if r >= '0' && r <= '9' && len(b.input) < 5 {
				b.input += string(r)
				v, _ := strconv.Atoi(b.input)
				g.setBalanceNumber(v)
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			if len(b.input) > 0 {
				b.input = b.input[:len(b.input)-1]
			}
			v, _ := strconv.Atoi(b.input)
			g.setBalanceNumber(v)
		}
		step := 1
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			step = 10
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.setBalanceNumber(b.values[b.field] + step)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.setBalanceNumber(b.values[b.field] - step)
		}
	}
	return true
}

func (p *modernHDPresentation) drawBalanceEditor(screen ebiten.FinalScreen, geometry ebiten.GeoM) {
	b := &p.Game.balanceUI
	if !b.open {
		return
	}
	if p.balanceSurface == nil {
		p.balanceSurface = ebiten.NewImage(1920, 1200)
	}
	s := p.balanceSurface
	s.Fill(color.RGBA{0, 0, 0, 180})
	rect := func(r image.Rectangle, c color.RGBA) {
		vector.DrawFilledRect(s, float32(r.Min.X*3), float32(r.Min.Y*3), float32(r.Dx()*3), float32(r.Dy()*3), c, false)
	}
	panel := image.Rect(44, 34, 596, 384)
	for inset, chrome := range []color.RGBA{
		{13, 18, 29, 255}, {115, 142, 153, 255},
		{247, 228, 176, 255}, {183, 141, 70, 255},
		{69, 49, 25, 255}, {122, 151, 168, 255},
	} {
		rect(panel.Inset(inset), chrome)
	}
	rect(panel.Inset(6), color.RGBA{42, 70, 128, 255})
	headingBottom := 72
	if b.ack < 3 {
		headingBottom = 97
	}
	rect(image.Rect(56, 43, 584, headingBottom), color.RGBA{50, 82, 151, 255})
	rect(image.Rect(62, headingBottom, 578, headingBottom+1), color.RGBA{169, 190, 202, 255})
	rect(image.Rect(62, headingBottom+1, 578, headingBottom+2), color.RGBA{23, 43, 82, 255})
	label := func(caption string, x, y, size int) {
		f := p.Game.font
		if f == nil {
			return
		}
		if h := f.hdFonts[p.Game.localeID]; h != nil {
			f = h
		}
		face, _ := f.faceFor(size)
		if face != nil {
			text.Draw(s, caption, face, x*3, y*3, color.RGBA{245, 237, 216, 255})
		}
	}
	button := func(r image.Rectangle, label string, active bool) {
		p.drawCompactToolbarButton(s, r, label, active)
		light, shade := color.RGBA{156, 182, 198, 255}, color.RGBA{13, 25, 43, 255}
		if active {
			light, shade = color.RGBA{249, 230, 176, 255}, color.RGBA{78, 52, 23, 255}
		}
		rect(image.Rect(r.Min.X+1, r.Min.Y+1, r.Max.X-1, r.Min.Y+2), light)
		rect(image.Rect(r.Min.X+1, r.Min.Y+1, r.Min.X+2, r.Max.Y-1), light)
		rect(image.Rect(r.Min.X+1, r.Max.Y-2, r.Max.X-1, r.Max.Y-1), shade)
		rect(image.Rect(r.Max.X-2, r.Min.Y+1, r.Max.X-1, r.Max.Y-1), shade)
	}
	if b.ack < 3 {
		label("修改前，請先想一想", 78, 82, 57)
		label("調整能力與成長會改變原版平衡，", 78, 126, 42)
		label("也可能降低挑戰與探索的樂趣。", 78, 151, 42)
		label("建議先另存遊戲。每次開啟都需按三次。", 78, 195, 39)
		label("按下「套用」才會修改設定。", 78, 220, 39)
		button(image.Rect(102, 252, 278, 281), "返回遊戲", false)
		button(image.Rect(342, 252, 538, 281), fmt.Sprintf("我了解，繼續（%d／3）", b.ack), true)
	} else {
		label("能力與成長", 63, 63, 51)
		button(image.Rect(522, 43, 576, 68), "關閉", false)
		for i, name := range []string{"當前能力", "升級成長", "特技命中", "敵人強度"} {
			button(image.Rect(62+i*130, 77, 186+i*130, 100), name, b.tab == i)
		}
		id, u := p.Game.balanceSelected()
		name := "目前沒有我方角色"
		if u != nil {
			name = u.Name
			if name == "" {
				if roster, exists := p.Game.partyRoster[id]; exists {
					name = roster.Name
				}
			}
			if name == "" {
				name = b.names[id]
			}
			if name == "" {
				name = fmt.Sprintf("角色 %d", id)
			}
		}
		if b.tab == 3 {
			name = "敵方全體"
		} else if b.tab == 1 && b.all {
			name = "我方全體預設"
		}
		button(image.Rect(62, 108, 94, 132), "‹", false)
		label(name, 108, 125, 42)
		button(image.Rect(400, 108, 432, 132), "›", false)
		if b.tab == 1 {
			scope := "逐人覆寫"
			if b.all {
				scope = "全體預設"
			}
			button(image.Rect(440, 108, 576, 132), scope, b.all)
		}
		rows := 5
		start := 145
		names := []string{"生命上限", "魔力上限", "基礎攻擊", "基礎防禦", "敏捷"}
		if b.tab == 1 {
			names = []string{"生命成長 %", "魔力成長 %", "攻擊成長 %", "防禦成長 %", "敏捷成長 %"}
		}
		canEdit := true
		if b.tab >= 2 {
			rows = 1
			start = 183
		}
		if b.tab == 2 {
			ids := p.Game.balanceCommands(u)
			cmd := -1
			if len(ids) > 0 {
				cmd = ids[b.command]
			}
			caption := "目前沒有已學會特技"
			if cmd >= 0 {
				caption = p.Game.nativeCommandLabels[cmd]
				if caption == "" {
					caption = fmt.Sprintf("特技 %d", cmd)
				}
			}
			button(image.Rect(62, 143, 94, 167), "‹", false)
			label(caption, 108, 160, 42)
			button(image.Rect(540, 143, 576, 167), "›", false)
			canEdit = battle.NativeCommandHasHitRoll(cmd)
			names = []string{"命中率 %"}
			if !canEdit {
				label("原本無命中骰，保留原傷害規則", 78, 222, 39)
			}
		}
		if b.tab == 3 {
			names = []string{"敵人整體強度 %"}
			label("生命、魔力、攻擊、防禦與敏捷", 78, 160, 39)
		}
		if canEdit {
			for i := 0; i < rows; i++ {
				y := start + i*25
				label(names[i], 78, y+14, 39)
				button(image.Rect(390, y-2, 430, y+20), "−", false)
				button(image.Rect(438, y-2, 522, y+20), fmt.Sprint(b.values[i]), b.field == i)
				button(image.Rect(530, y-2, 570, y+20), "＋", false)
			}
		}
		note := "保留裝備加成；不會自動補滿生命與魔力。"
		switch b.tab {
		case 1:
			note = "只影響之後升級；100% 使用原版成長。"
		case 2:
			note = "保留目標限制、免疫與原傷害規則。"
		case 3:
			note = "100% 使用原版強度；保留已受傷比例。"
		}
		label(note, 63, 285, 36)
		label("點數值輸入；上下鍵微調，Shift 每次調整 10。", 63, 306, 33)
		reset := "恢復原版／預設"
		if b.tab == 0 {
			reset = "撤銷本次修改"
		}
		button(image.Rect(62, 321, 214, 348), reset, false)
		button(image.Rect(434, 321, 576, 348), "套用", true)
		label(b.message, 63, 369, 30)
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/3, 1.0/3)
	op.GeoM.Concat(geometry)
	screen.DrawImage(s, op)
}
