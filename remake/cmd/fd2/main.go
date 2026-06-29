// 炎龍騎士團2 重製 — Go/Ebiten 垂直切片(MVP)。
//
// 目標:證明「Go/Ebiten 跑得起來,且讀得到我們逆向出的資料」。
// 本切片:載入一張戰場(tileset PNG + 地圖 JSON)→ 用 hi-res 畫布渲染 →
//         方向鍵 / WASD / 觸控移動游標,相機跟隨。桌面 / Web(WASM)/ 手機共用。
//
// 資產(玩家自備原版後由 tools/ 產生,不隨庫散布):
//   assets/tileset.png  一張 24×24 圖塊的網格圖(cols 欄)
//   assets/map.json     {"w","h","tileW","tileH","cols","tiles":[地形索引...]}
//
// 建置:見 remake/README.md(docker golang;WASM / 桌面 / 手機)。
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

const (
	logicalW = 640 // hi-res 內部畫布(CJK/觀感原則:拉畫布、別縮字)
	logicalH = 400
)

// MapData 對應 assets/map.json(由 tools/export_engine_assets.py 產生)。
type MapData struct {
	W     int   `json:"w"`
	H     int   `json:"h"`
	TileW int   `json:"tileW"`
	TileH int   `json:"tileH"`
	Cols  int   `json:"cols"` // tileset 每列圖塊數
	Tiles []int `json:"tiles"`
}

type Game struct {
	m       *MapData
	tileset *ebiten.Image
	tiles   []*ebiten.Image // 切好的圖塊
	st      *battle.State    // 戰鬥狀態(單位)
	sc      *battle.Scenario   // 劇本(事件系統,doc 29)
	dialog  []battle.DialogLine // 待顯示對話(事件產生,含說話者)
	portraits map[int][]*ebiten.Image // DATO 頭像:肖像 id → 4 嘴型幀
	mouthOpen  bool // 嘴型動畫狀態(原版 0x16d00:m0閉/m3開)
	mouthTimer int  // 閉嘴倒數(原版 rand%30+2 tick)
	curX    int
	curY    int
	camX    float64
	camY    float64
	loadErr string
	// 截圖鉤子(FD2_SHOT=path 啟用):第 shotFrame 幀存 PNG 後自動退出(有界,供無人值守驗證)
	frame     int
	shotPath  string
	shotFrame int
	shotTurn  int // 截圖前自動推進到第 N 回合(FD2_SHOT_TURN,驗證增援進場)
	shotCurX  int // 截圖時把游標放這(FD2_SHOT_CUR=x,y)
	shotCurY  int
	shotSel   bool // 截圖前自動選取游標單位(FD2_SHOT_SELECT=1)
	// 選取狀態
	sel    *battle.Unit
	reach  map[battle.Cell]bool
	moved  bool   // 已選單位是否移動完(進入攻擊階段)
	result string // 勝負:""/win/lose
	msg    string // 短訊息(攻擊傷害等)
	// 地圖單位 sprite(FDICON 待機分鏡):fig index → 幀序列
	sprites map[int][]*ebiten.Image
	figani  map[int][]*ebiten.Image // 攻擊全身動畫(FIGANI):fig → 幀序列
	atk     *atkAnim                // 進行中的攻擊演出
	bg      *ebiten.Image           // 戰鬥背景(BG.DAT,by 戰場;map0=BG_004 森林)
	font    *Font                   // 原版點陣中文字型(doc 08)
}

// atkAnim 全螢幕戰鬥演出(對照原版 orig_05:守方左/攻方右土台/斬擊弧/血條/閃紅抽血)。
type atkAnim struct {
	atkFig, defFig   int    // 攻方(右土台)/ 守方(左)FIGANI
	atkName, defName string // 名字(資訊條)
	atkHP, atkMax    int
	atkMP, defMP     int
	atkLV, defLV     int
	defHP0, defHP1   int // 守方攻擊前/後 HP(impact 抽乾動畫)
	defMax           int
	timer, total     int
	terrain          int // 攻擊格地形索引(戰鬥背景 = 戰場地形,跟 FDFIELD 戰場資料有關)
}


// terrainAt 回傳某格的地形索引(戰鬥背景用;越界回 -1)。
func (g *Game) terrainAt(x, y int) int {
	if g.m == nil || x < 0 || y < 0 || x >= g.m.W || y >= g.m.H {
		return -1
	}
	return g.m.Tiles[y*g.m.W+x]
}

// figaniIndex:戰鬥全身動畫 FIGANI index = FDICON組 × 3(恆等,反組譯 0x2884c:unit[+7]×3;doc06)。
// unit[+7]=FDICON組(0x11019 ×12→地圖sprite、0x2884c ×3→FIGANI,同一欄)。我方敵方統一:
// 索爾組0→FIGANI0、亞雷斯組4→12、盜賊組96→288。地圖組=FIGANI/3,不需對應表。
func figaniIndex(fig int) int { return fig * 3 }

// loadSprites 載入 assets/sprites/fig_NNN_fMM.png,按 fig index 分組成幀序列。
func loadSprites() map[int][]*ebiten.Image {
	out := map[int][]*ebiten.Image{}
	files, _ := filepath.Glob("assets/sprites/fig_*_f*.png")
	type fr struct {
		idx, fno int
		img      *ebiten.Image
	}
	var frs []fr
	for _, fp := range files {
		var idx, fno int
		if _, e := fmt.Sscanf(filepath.Base(fp), "fig_%d_f%d.png", &idx, &fno); e != nil {
			continue
		}
		raw, e := os.ReadFile(fp)
		if e != nil {
			continue
		}
		im, _, e := image.Decode(bytes.NewReader(raw))
		if e != nil {
			continue
		}
		frs = append(frs, fr{idx, fno, ebiten.NewImageFromImage(im)})
	}
	sort.Slice(frs, func(i, j int) bool {
		if frs[i].idx != frs[j].idx {
			return frs[i].idx < frs[j].idx
		}
		return frs[i].fno < frs[j].fno
	})
	for _, f := range frs {
		out[f.idx] = append(out[f.idx], f.img)
	}
	return out
}

// loadPortraits 載入 assets/portraits/DATO_NNN_mM.png,按肖像 id 分組成 4 嘴型幀。
func loadPortraits() map[int][]*ebiten.Image {
	out := map[int][]*ebiten.Image{}
	files, _ := filepath.Glob("assets/portraits/DATO_*_m*.png")
	type fr struct {
		id, m int
		img   *ebiten.Image
	}
	var frs []fr
	for _, fp := range files {
		var id, m int
		if _, e := fmt.Sscanf(filepath.Base(fp), "DATO_%d_m%d.png", &id, &m); e != nil {
			continue
		}
		raw, e := os.ReadFile(fp)
		if e != nil {
			continue
		}
		im, _, e := image.Decode(bytes.NewReader(raw))
		if e != nil {
			continue
		}
		frs = append(frs, fr{id, m, ebiten.NewImageFromImage(im)})
	}
	sort.Slice(frs, func(i, j int) bool {
		if frs[i].id != frs[j].id {
			return frs[i].id < frs[j].id
		}
		return frs[i].m < frs[j].m
	})
	for _, f := range frs {
		out[f.id] = append(out[f.id], f.img)
	}
	return out
}

// loadFIGANI 載入 assets/figani/FIGANI_NNN_fNN.png,按 fig id 分組成攻擊全身分鏡。
func loadFIGANI() map[int][]*ebiten.Image {
	out := map[int][]*ebiten.Image{}
	files, _ := filepath.Glob("assets/figani/FIGANI_*_f*.png")
	type fr struct {
		id, f int
		img   *ebiten.Image
	}
	var frs []fr
	for _, fp := range files {
		var id, f int
		if _, e := fmt.Sscanf(filepath.Base(fp), "FIGANI_%d_f%d.png", &id, &f); e != nil {
			continue
		}
		raw, e := os.ReadFile(fp)
		if e != nil {
			continue
		}
		im, _, e := image.Decode(bytes.NewReader(raw))
		if e != nil {
			continue
		}
		frs = append(frs, fr{id, f, ebiten.NewImageFromImage(im)})
	}
	sort.Slice(frs, func(i, j int) bool {
		if frs[i].id != frs[j].id {
			return frs[i].id < frs[j].id
		}
		return frs[i].f < frs[j].f
	})
	for _, f := range frs {
		out[f.id] = append(out[f.id], f.img)
	}
	return out
}

// toFullWidth 把半形標點轉全形(中文排版 + 避開部分 face 缺半形 ASCII glyph)。
func toFullWidth(s string) string {
	r := []rune(s)
	for i, c := range r {
		switch c {
		case ',':
			r[i] = '，'
		case '!':
			r[i] = '！'
		case ':':
			r[i] = '：'
		case '?':
			r[i] = '？'
		case ';':
			r[i] = '；'
		case '.':
			r[i] = '。'
		case '(':
			r[i] = '（'
		case ')':
			r[i] = '）'
		}
	}
	return string(r)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// dirToward 從 (ax,ay) 朝 (tx,ty) 的方向:0下 1左 2上 3右(FDICON 方向幀)。
func dirToward(ax, ay, tx, ty int) int {
	dx, dy := tx-ax, ty-ay
	if absInt(dx) > absInt(dy) {
		if dx > 0 {
			return 3
		}
		return 1
	}
	if dy > 0 {
		return 0
	}
	return 2
}

// confirm 處理 Enter/Space:選取我方單位顯示移動範圍,或移動到可達格 / 原地待機。
func (g *Game) confirm() {
	if g.st == nil {
		return
	}
	cur := battle.Cell{X: g.curX, Y: g.curY}
	if g.sel == nil { // 選我方單位
		u := g.st.UnitAt(g.curX, g.curY)
		if u != nil && u.Camp == battle.Own && !u.Acted {
			g.sel = u
			g.moved = false
			g.reach = g.st.Reachable(u)
		}
		return
	}
	if !g.moved { // 移動階段
		switch {
		case g.curX == g.sel.X && g.curY == g.sel.Y: // 原地 → 不移動,進攻擊/待命階段
			g.moved = true
			g.reach = nil
		case g.reach[cur] && g.st.UnitAt(g.curX, g.curY) == nil: // 移動到可達空格
			g.sel.X, g.sel.Y = g.curX, g.curY
			g.moved = true
			g.reach = nil
		}
		return
	}
	// 攻擊階段:游標在相鄰敵 → 攻擊;在自己格 → 待命
	if tgt := g.st.UnitAt(g.curX, g.curY); tgt != nil && tgt != g.sel &&
		tgt.Camp != battle.Own && g.st.InAttackRange(g.sel, g.curX, g.curY) {
		// 攻擊者面向目標(FDICON 方向幀)
		g.sel.Dir = dirToward(g.sel.X, g.sel.Y, g.curX, g.curY)
		nm := tgt.Name
		if nm == "" {
			nm = tgt.ClsName
		}
		anm := g.sel.Name
		if anm == "" {
			anm = g.sel.ClsName
		}
		defHP0 := tgt.HP
		dmg := g.st.Attack(g.sel, tgt)
		g.msg = fmt.Sprintf("%s 攻擊 %s,造成 %d 傷害", anm, nm, dmg)
		g.atk = &atkAnim{atkFig: figaniIndex(g.sel.Fig), defFig: figaniIndex(tgt.Fig), atkName: anm, defName: nm,
			atkHP: g.sel.HP, atkMax: g.sel.MaxHP, atkLV: g.sel.Lv, atkMP: g.sel.MP,
			defLV: tgt.Lv, defMP: tgt.MP,
			defHP0: defHP0, defHP1: tgt.HP, defMax: tgt.MaxHP, timer: 48, total: 48,
			terrain: g.terrainAt(g.curX, g.curY)} // 戰鬥背景 = 守方格地形(FDFIELD)
		g.sel, g.reach, g.moved = nil, nil, false
		g.checkResult()
	} else if g.curX == g.sel.X && g.curY == g.sel.Y { // 原地待命
		g.sel.Acted = true
		g.sel, g.reach, g.moved = nil, nil, false
	}
}

// checkResult 檢查勝負(失敗條件:索爾死;勝利:敵全滅,doc28 第1章)。
func (g *Game) checkResult() {
	if g.result != "" || g.sc == nil {
		return
	}
	if r := g.st.Result("索爾"); r != "" {
		g.result = r
	}
}

// 陣營顏色(M1 暫用色塊,M2/sprite 後換真圖)。
func campColor(c battle.Camp) color.RGBA {
	switch c {
	case battle.Own:
		return color.RGBA{0x40, 0x80, 0xff, 0xff} // 藍
	case battle.Ally:
		return color.RGBA{0x40, 0xc0, 0x40, 0xff} // 綠
	default:
		return color.RGBA{0xe0, 0x40, 0x40, 0xff} // 紅
	}
}

func (g *Game) tileAt(idx int) *ebiten.Image {
	if idx < 0 || idx >= len(g.tiles) {
		return nil
	}
	return g.tiles[idx]
}

func (g *Game) Update() error {
	g.frame++
	// 攻擊演出推進(FIGANI 全身分鏡;演出期間鎖玩家輸入)
	if g.atk != nil {
		g.atk.timer--
		if g.atk.timer <= 0 {
			g.atk = nil
		}
	}
	// 行軍動畫(spawn_march):進場位移緩動歸零,到位轉正面待機
	if g.st != nil {
		for _, u := range g.st.Units {
			if u.OffX != 0 {
				u.OffX *= 0.85
				if u.OffX < 1 && u.OffX > -1 {
					u.OffX = 0
				}
			}
			if u.OffY != 0 {
				u.OffY *= 0.85
				if u.OffY < 1 && u.OffY > -1 {
					u.OffY = 0
					u.Dir = 0 // 到位面向鏡頭待機
				}
			}
		}
	}
	// 嘴型動畫(忠實原版 0x16d00,doc14):每 2 frame 一 tick;閉嘴隨機 2-31 tick、開嘴一瞬
	if len(g.dialog) > 0 && g.frame%2 == 0 {
		if g.mouthOpen {
			g.mouthOpen = false
			g.mouthTimer = rand.Intn(30) + 2
		} else if g.mouthTimer--; g.mouthTimer <= 0 {
			g.mouthOpen = true
		}
	}
	// 截圖模式:到指定幀後自動退出(畫面已於 Draw 存檔)
	if g.shotPath != "" {
		if g.frame == 1 {
			for i := 0; i < g.shotTurn; i++ { // 推進 N 個回合(觸發增援事件),驗證進場
				g.endTurn()
			}
			if g.shotCurX != 0 || g.shotCurY != 0 {
				g.curX, g.curY = g.shotCurX, g.shotCurY
			}
			if g.shotSel {
				g.confirm()
			}
			if v := os.Getenv("FD2_SHOT_ATTACK"); v != "" { // 全螢幕戰鬥演出(驗證用):索爾打盜賊
				fig, _ := strconv.Atoi(v)
				g.atk = &atkAnim{atkFig: figaniIndex(fig), defFig: figaniIndex(96), atkName: "亞雷斯", defName: "盜賊",
					atkHP: 48, atkMax: 48, atkLV: 1, atkMP: 0, defLV: 2, defMP: 0,
					defHP0: 28, defHP1: 8, defMax: 36, timer: 48, total: 48}
			}
		}
		if g.frame > g.shotFrame {
			return ebiten.Termination
		}
	}
	if g.m == nil {
		return nil
	}
	// 游標移動:方向鍵 / WASD / 觸控
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.curX--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.curX++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.curY--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.curY++
	}
	// 觸控:點哪格移到哪格
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		tx, ty := ebiten.TouchPosition(id)
		g.curX = (int(g.camX) + tx) / g.m.TileW
		g.curY = (int(g.camY) + ty) / g.m.TileH
	}
	// 邊界
	if g.curX < 0 {
		g.curX = 0
	}
	if g.curY < 0 {
		g.curY = 0
	}
	if g.curX >= g.m.W {
		g.curX = g.m.W - 1
	}
	if g.curY >= g.m.H {
		g.curY = g.m.H - 1
	}
	// 選取 / 移動 / 取消
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.confirm()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.sel, g.reach = nil, nil
	}
	// Tab:結束回合(觸發 on_turn_end 增援事件)。正式版改我方全動完自動結束。
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.endTurn()
	}
	// 相機跟隨游標(置中,夾在地圖內)
	g.camX = float64(g.curX*g.m.TileW - logicalW/2 + g.m.TileW/2)
	g.camY = float64(g.curY*g.m.TileH - logicalH/2 + g.m.TileH/2)
	clamp(&g.camX, 0, float64(g.m.W*g.m.TileW-logicalW))
	clamp(&g.camY, 0, float64(g.m.H*g.m.TileH-logicalH))
	return nil
}

func clamp(v *float64, lo, hi float64) {
	if hi < lo {
		hi = lo
	}
	if *v < lo {
		*v = lo
	}
	if *v > hi {
		*v = hi
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.m == nil {
		ebitenutil.DebugPrint(screen, "FD2 重製 MVP\n缺 assets/(tileset.png + map.json)\n用 tools/export_engine_assets.py 產生\n"+g.loadErr)
		return
	}
	// 攻擊演出:切全螢幕戰鬥畫面(蓋地圖,對照原版 orig_05 全螢幕戰鬥)
	if g.atk != nil {
		g.drawBattleScene(screen)
		if g.shotPath != "" && g.frame == g.shotFrame {
			saveShot(screen, g.shotPath)
		}
		return
	}
	tw, th := g.m.TileW, g.m.TileH
	// 只畫可見範圍
	x0 := int(g.camX) / tw
	y0 := int(g.camY) / th
	x1 := (int(g.camX)+logicalW)/tw + 1
	y1 := (int(g.camY)+logicalH)/th + 1
	for cy := y0; cy <= y1 && cy < g.m.H; cy++ {
		for cx := x0; cx <= x1 && cx < g.m.W; cx++ {
			if cy < 0 || cx < 0 {
				continue
			}
			t := g.tileAt(g.m.Tiles[cy*g.m.W+cx])
			if t == nil {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(cx*tw)-g.camX, float64(cy*th)-g.camY)
			screen.DrawImage(t, op)
		}
	}
	// 移動範圍高亮(已選單位:藍色半透明格)
	if g.sel != nil {
		hl := ebiten.NewImage(tw, th)
		hl.Fill(color.RGBA{0x40, 0x80, 0xff, 0x66})
		for c := range g.reach {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(c.X*tw)-g.camX, float64(c.Y*th)-g.camY)
			screen.DrawImage(hl, op)
		}
	}
	// 單位層(M1:FIGANI 待機動畫 sprite + 陣營腳標 + HP bar;無 sprite 退回色塊)
	if g.st != nil {
		for _, u := range g.st.Units {
			if !u.OnField || !u.Alive() { // 待命(未進場)單位不畫
				continue
			}
			ux := float64(u.X*tw) - g.camX
			uy := float64(u.Y*th) - g.camY
			if ux < -float64(tw) || ux > logicalW || uy < -float64(th) || uy > logicalH {
				continue
			}
			g.drawUnitSprite(screen, ux, uy, float64(tw), float64(th), u)
		}
	}
	// 游標(白框)
	curPx := float64(g.curX*tw) - g.camX
	curPy := float64(g.curY*th) - g.camY
	drawCursor(screen, curPx, curPy, float64(tw), float64(th))
	// 狀態列:選中單位資訊(中文職業名待 M2 TTF)
	info := "炎龍騎士團2 — M1 戰棋核心 / 方向鍵·WASD·觸控移動游標"
	if g.st != nil {
		info += fmt.Sprintf("  回合%d  我方%d 友%d 敵%d",
			g.st.Turn, g.st.AliveCount(battle.Own), g.st.AliveCount(battle.Ally), g.st.AliveCount(battle.Enemy))
		if u := g.st.UnitAt(g.curX, g.curY); u != nil {
			nm := u.Name
			if nm == "" {
				nm = u.ClsName
			}
			info += fmt.Sprintf("\n[%d,%d] %s %s Lv%d HP%d/%d AP%d DP%d MV%d",
				u.X, u.Y, u.Camp, nm, u.Lv, u.HP, u.MaxHP, u.AP, u.DP, u.MV)
		}
		if g.sc != nil {
			info += "  (Tab:結束回合)"
		}
	}
	ebitenutil.DebugPrint(screen, info)

	// 中文層(原版點陣字型,doc 08):選中單位名 + 對話框(DebugPrint 不支援中文)
	if g.font != nil {
		if g.st != nil { // 選中單位中文名(放游標格上方,避開頂部 DebugPrint)
			if u := g.st.UnitAt(g.curX, g.curY); u != nil {
				nm := u.Name
				if nm == "" {
					nm = u.ClsName
				}
				if nm != "" {
					nx := float64(g.curX*tw) - g.camX
					ny := float64(g.curY*th) - g.camY - 18
					g.font.Draw(screen, nm, nx, ny, 1.0, color.RGBA{0xff, 0xeb, 0x78, 0xff})
				}
			}
		}
		if len(g.dialog) > 0 { // 底部對話框:左頭像 + 右文字(原版佈局,real_pic)
			dl := g.dialog[len(g.dialog)-1]
			boxH := 120.0 // 原版約佔底部 1/4~1/3
			// 依說話者切上/下框 + 左/右頭像(對照原版 orig_02_dialog:我方下框左頭像、對方/NPC 上框右頭像)
			upper := dl.Speaker >= 32 // >=32 為對方/敵/NPC(我方角色 id 0-31)
			top := float64(logicalH) - boxH
			if upper {
				top = 0
			}
			box := ebiten.NewImage(logicalW, int(boxH))
			box.Fill(color.RGBA{0x10, 0x18, 0x48, 0xf2})
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(0, top)
			screen.DrawImage(box, op)
			edge := ebiten.NewImage(logicalW, 2)
			edge.Fill(color.RGBA{0xc8, 0xa0, 0x40, 0xff})
			oe := &ebiten.DrawImageOptions{}
			edgeY := top
			if upper {
				edgeY = top + boxH - 2
			}
			oe.GeoM.Translate(0, edgeY)
			screen.DrawImage(edge, oe)
			// 頭像:我方左、對方右(側臉半身,佔框高滿版;嘴型 m0閉/m3開)
			s := boxH / 80.0
			hx, tx := 6.0, 6.0+80*s+12
			if upper { // 右頭像,文字在左
				hx = float64(logicalW) - 6 - 80*s
				tx = 16
			}
			if fr := g.portraits[dl.Speaker]; len(fr) > 0 {
				mi := 0
				if g.mouthOpen && len(fr) > 3 {
					mi = 3
				}
				po := &ebiten.DrawImageOptions{}
				po.GeoM.Scale(s, s)
				po.GeoM.Translate(hx, top)
				screen.DrawImage(fr[mi], po)
			} else {
				tx = 16
			}
			g.font.Draw(screen, "『"+toFullWidth(dl.Text)+"』", tx, top+24, 1.7, color.RGBA{0xf0, 0xf4, 0xff, 0xff})
		}
		if g.msg != "" && len(g.dialog) == 0 { // 攻擊等短訊(無對話框時)
			g.font.Draw(screen, g.msg, 8, float64(logicalH)-30, 1.2, color.RGBA{0xff, 0xf0, 0xb4, 0xff})
		}
		if g.result != "" { // 勝負(中央大字)
			t := "勝　利"
			c := color.RGBA{0xff, 0xdc, 0x50, 0xff}
			if g.result == "lose" {
				t = "敗　北"
				c = color.RGBA{0xff, 0x70, 0x70, 0xff}
			}
			g.font.Draw(screen, t, float64(logicalW)/2-78, float64(logicalH)/2-30, 3.0, c)
		}
	}

	// 截圖鉤子:指定幀把畫面存 PNG(無人值守驗證用)
	if g.shotPath != "" && g.frame == g.shotFrame {
		saveShot(screen, g.shotPath)
	}
}

func saveShot(img *ebiten.Image, path string) {
	f, err := os.Create(path)
	if err != nil {
		log.Println("shot:", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil { // *ebiten.Image 實作 image.Image
		log.Println("shot encode:", err)
		return
	}
	log.Println("screenshot ->", path)
}

// drawBattleScene 全螢幕戰鬥演出(對照原版 orig_05:守方左面右/攻方右土台/斬擊弧/血條/命中閃紅抽血)。
func (g *Game) drawBattleScene(screen *ebiten.Image) {
	a := g.atk
	prog := a.total - a.timer
	// 背景:遠景天空 + 戰場地形 tile(草地)平鋪(跟戰場資料有關,對照 orig_05;確切地形物件/遠景樹待續RE)
	// 背景:BG.DAT 森林(戰鬥背景 = 戰場資料對應 BG;map0=BG_004)。320×100 拉伸填上 2/3,草地 tile 補地面
	if g.bg != nil {
		op := &ebiten.DrawImageOptions{}
		bb := g.bg.Bounds()
		op.GeoM.Scale(float64(logicalW)/float64(bb.Dx()), 280.0/float64(bb.Dy()))
		screen.DrawImage(g.bg, op)
	} else {
		screen.Fill(color.RGBA{0x5c, 0x6e, 0x66, 0xff})
	}
	terr := a.terrain // 地面草地補(接 BG 下緣到底)
	if terr <= 0 || terr >= len(g.tiles) {
		terr = 52
	}
	if terr < len(g.tiles) && g.tiles[terr] != nil {
		gt := g.tiles[terr]
		gw, gh := gt.Bounds().Dx(), gt.Bounds().Dy()
		for yy := 252; yy < logicalH; yy += gh {
			for xx := 0; xx < logicalW; xx += gw {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(xx), float64(yy))
				screen.DrawImage(gt, op)
			}
		}
	}
	// 對照 orig_05 網格量測(640×480→本畫布×0.833):守方盜賊左上偏小、攻方亞雷斯右下站土台偏大(景深)
	// 守方盜賊(左,翻轉面右,腳 y≈258、中心 x≈130、sc1.63);命中閃紅
	if fr := g.figani[a.defFig]; len(fr) > 0 {
		const scDef = 1.63
		gyDef := 258.0
		img := fr[0]
		b := img.Bounds()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(-scDef, scDef)
		op.GeoM.Translate(130+float64(b.Dx())*scDef/2, gyDef-float64(b.Dy())*scDef)
		if prog >= 22 && prog < 40 {
			op.ColorScale.Scale(2.2, 0.0, 0.0, 1)
		}
		screen.DrawImage(img, op)
	}
	// 橢圓土台(右下,中心 x≈480、y≈352)
	plat := ebiten.NewImage(150, 22)
	plat.Fill(color.RGBA{0x46, 0x58, 0x30, 0xff})
	opP := &ebiten.DrawImageOptions{}
	opP.GeoM.Translate(480-75, 350)
	screen.DrawImage(plat, opP)
	// 攻方亞雷斯(右下,腳 y≈367 站土台、中心 x≈480、sc2.0、揮擊序列)
	if fr := g.figani[a.atkFig]; len(fr) > 0 {
		const scAtk = 2.0
		gyAtk := 367.0
		img := fr[(prog/4)%len(fr)]
		b := img.Bounds()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scAtk, scAtk)
		op.GeoM.Translate(480-float64(b.Dx())*scAtk/2, gyAtk-float64(b.Dy())*scAtk)
		screen.DrawImage(img, op)
	}
	// swing/impact 大白斬擊弧:從右上揮掃到左(對照原版 orig_05,粗白弧)
	if prog >= 12 && prog < 34 {
		cxA, cyA := float64(logicalW)*0.46, float64(logicalH)*0.62
		r := 230.0
		seg := 14
		a0, a1 := -0.15, 1.25 // 弧角範圍(弧度,右上→左)
		alpha := uint8(0xb0)
		if prog >= 26 {
			alpha = 0x60 // 收尾淡出
		}
		for i := 0; i < seg; i++ {
			t0 := a0 + (a1-a0)*float64(i)/float64(seg)
			t1 := a0 + (a1-a0)*float64(i+1)/float64(seg)
			x0 := cxA + r*math.Cos(t0)
			y0 := cyA - r*math.Sin(t0)
			x1 := cxA + r*math.Cos(t1)
			y1 := cyA - r*math.Sin(t1)
			vector.StrokeLine(screen, float32(x0), float32(y0), float32(x1), float32(y1), 14, color.RGBA{0xff, 0xff, 0xf0, alpha}, true)
		}
	}
	if g.font != nil { // 守方資訊左下、攻方右上(對照原版血條位置)
		dhp := a.defHP0 // 命中時 HP 漸抽(defHP0→defHP1)
		if prog >= 24 {
			t := float64(prog-24) / 12
			if t > 1 {
				t = 1
			}
			dhp = a.defHP0 + int(float64(a.defHP1-a.defHP0)*t)
		}
		drawBattlePanel(screen, g.font, 322, 6, 312, 62, a.atkName, a.atkLV, a.atkHP, a.atkMax, a.atkMP)  // 攻方右上
		drawBattlePanel(screen, g.font, 6, 288, 300, 94, a.defName, a.defLV, dhp, a.defMax, a.defMP)     // 守方左下
	}
}

// drawBattlePanel 原版戰鬥狀態欄(對照 orig_05:深藍框+淺藍立體邊+名+LV‧NN右上+HP黃條+MP紅條+數字右)。
func drawBattlePanel(screen *ebiten.Image, f *Font, x, y, w, h float64, name string, lv, hp, mx, mp int) {
	panel := ebiten.NewImage(int(w), int(h))
	panel.Fill(color.RGBA{0x30, 0x50, 0x98, 0xf2})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(panel, op)
	edge := ebiten.NewImage(int(w), 2) // 上亮邊(立體框)
	edge.Fill(color.RGBA{0x98, 0xc0, 0xf0, 0xff})
	oe := &ebiten.DrawImageOptions{}
	oe.GeoM.Translate(x, y)
	screen.DrawImage(edge, oe)
	f.Draw(screen, name, x+10, y+5, 1.5, color.RGBA{0xc8, 0xe0, 0xff, 0xff}) // 名(左上,大)
	if lv > 0 {
		f.Draw(screen, fmt.Sprintf("LV‧%02d", lv), x+w-86, y+8, 1.1, color.RGBA{0xff, 0xff, 0xff, 0xff})
	}
	barX, barW := x+54, w-104
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	f.Draw(screen, "HP", x+10, y+h*0.42, 1.0, white)
	drawStatBar(screen, barX, y+h*0.47, barW, float64(hp)/float64(mx), color.RGBA{0xf0, 0xc8, 0x30, 0xff})
	f.Draw(screen, fmt.Sprintf("%03d", hp), x+w-42, y+h*0.42, 1.0, white)
	mpmx := mp
	if mpmx < 1 {
		mpmx = 1
	}
	f.Draw(screen, "MP", x+10, y+h*0.72, 1.0, white)
	drawStatBar(screen, barX, y+h*0.77, barW, float64(mp)/float64(mpmx), color.RGBA{0xc0, 0x28, 0x28, 0xff})
	f.Draw(screen, fmt.Sprintf("%03d", mp), x+w-42, y+h*0.72, 1.0, white)
}

// drawStatBar 狀態條(暗槽 + 填充)。
func drawStatBar(screen *ebiten.Image, x, y, w, frac float64, c color.RGBA) {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	slot := ebiten.NewImage(int(w), 9)
	slot.Fill(color.RGBA{0x18, 0x18, 0x28, 0xff})
	os := &ebiten.DrawImageOptions{}
	os.GeoM.Translate(x, y)
	screen.DrawImage(slot, os)
	if frac > 0 {
		bar := ebiten.NewImage(int(w*frac)+1, 9)
		bar.Fill(c)
		ob := &ebiten.DrawImageOptions{}
		ob.GeoM.Translate(x, y)
		screen.DrawImage(bar, ob)
	}
}

// drawUnitSprite 畫一個單位:純 FDICON Q 版 sprite(原版無 HP bar/腳標,還原乾淨)。
// 用方向走動分鏡(FDICON 12幀=4方向×3:站/抬左手/抬右手);行軍時套用 OffX/OffY 位移。
func (g *Game) drawUnitSprite(screen *ebiten.Image, x, y, w, h float64, u *battle.Unit) {
	x += u.OffX // 行軍/移動位移
	y += u.OffY
	frames := g.sprites[u.Fig]
	if len(frames) == 0 {
		drawUnit(screen, x, y, w, h, campColor(u.Camp), u) // fallback 色塊
		return
	}
	// 方向走動幀:dir(0下1左2上3右)×3 + 走動相位;不足 12 幀(只導下方向)則退回
	f := (g.frame / 8) % 3
	idx := u.Dir*3 + f
	if idx >= len(frames) {
		idx = f % len(frames)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y-4) // 略上移讓單位「站」在格上
	if u.Acted {
		op.ColorScale.Scale(0.55, 0.55, 0.6, 1) // 已行動變暗(對映原版灰階)
	}
	screen.DrawImage(frames[idx], op)
}

// drawUnit 畫一個單位(fallback:內縮色塊 + 頂部 HP bar)。
func drawUnit(dst *ebiten.Image, x, y, w, h float64, col color.RGBA, u *battle.Unit) {
	pad := 3.0
	body := ebiten.NewImage(int(w-2*pad), int(h-2*pad))
	body.Fill(col)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x+pad, y+pad)
	dst.DrawImage(body, op)
	// HP bar
	bw := w - 2*pad
	frac := float64(u.HP) / float64(u.MaxHP)
	if frac < 0 {
		frac = 0
	}
	bar := ebiten.NewImage(int(bw*frac)+1, 2)
	bar.Fill(color.RGBA{0x30, 0xff, 0x30, 0xff})
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(x+pad, y+pad-3)
	dst.DrawImage(bar, op2)
}

func drawCursor(dst *ebiten.Image, x, y, w, h float64) {
	col := image.White
	t := 2.0
	bars := []struct{ x, y, w, h float64 }{
		{x, y, w, t}, {x, y + h - t, w, t}, {x, y, t, h}, {x + w - t, y, t, h},
	}
	for _, b := range bars {
		sub := ebiten.NewImage(int(b.w), int(b.h))
		sub.Fill(col)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(b.x, b.y)
		dst.DrawImage(sub, op)
	}
}

func (g *Game) Layout(outsideW, outsideH int) (int, int) {
	return logicalW, logicalH
}

func loadGame() *Game {
	g := &Game{shotFrame: 20}
	g.shotPath = os.Getenv("FD2_SHOT")
	if v := os.Getenv("FD2_SHOT_FRAME"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			g.shotFrame = n
		}
	}
	if v := os.Getenv("FD2_SHOT_CUR"); v != "" {
		fmt.Sscanf(v, "%d,%d", &g.shotCurX, &g.shotCurY)
	}
	if v := os.Getenv("FD2_SHOT_TURN"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			g.shotTurn = n
		}
	}
	raw, err := os.ReadFile("assets/map.json")
	if err != nil {
		g.loadErr = err.Error()
		return g
	}
	var m MapData
	if err := json.Unmarshal(raw, &m); err != nil {
		g.loadErr = err.Error()
		return g
	}
	pngRaw, err := os.ReadFile("assets/tileset.png")
	if err != nil {
		g.loadErr = err.Error()
		return g
	}
	img, _, err := image.Decode(bytes.NewReader(pngRaw))
	if err != nil {
		g.loadErr = err.Error()
		return g
	}
	g.tileset = ebiten.NewImageFromImage(img)
	// 切圖塊
	tsW := g.tileset.Bounds().Dx()
	cols := m.Cols
	if cols == 0 {
		cols = tsW / m.TileW
	}
	n := (g.tileset.Bounds().Dy() / m.TileH) * cols
	for i := 0; i < n; i++ {
		sx := (i % cols) * m.TileW
		sy := (i / cols) * m.TileH
		r := image.Rect(sx, sy, sx+m.TileW, sy+m.TileH)
		g.tiles = append(g.tiles, g.tileset.SubImage(r).(*ebiten.Image))
	}
	g.m = &m
	// 載入單位(M1)
	if st, err := battle.Load("assets/map0_units.json"); err == nil {
		g.st = st
	} else if g.loadErr == "" {
		g.loadErr = "units: " + err.Error()
	}
	// 載入劇本 + 套用初始(待命 group + on_battle_start 主角隊進場,doc 25/29)
	if g.st != nil {
		if sc, err := battle.LoadScenario("assets/scenarios/ch01.json"); err == nil {
			g.sc = sc
			g.dialog = append(g.dialog, sc.Setup(g.st)...)
		} else if g.loadErr == "" {
			g.loadErr = "scenario: " + err.Error()
		}
	}
	g.sprites = loadSprites()
	g.portraits = loadPortraits()
	g.figani = loadFIGANI()
	if raw, e := os.ReadFile("assets/bg/bg.png"); e == nil { // 戰鬥背景(BG.DAT)
		if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
			g.bg = ebiten.NewImageFromImage(im)
		}
	}
	g.font = loadFont()
	return g
}

// endTurn 結束當前回合:觸發 on_turn_end 事件(增援等),回合 +1,清除已行動。
// 回合無上限(doc 27);只由劇本事件決定勝負。
func (g *Game) endTurn() {
	if g.st == nil || g.result != "" {
		return
	}
	if g.shotPath == "" { // 截圖驗證模式不跑 AI(純看進場/事件,免站著被打死)
		g.st.AITurn() // 敵方 + 友軍 NPC 行動(combat.go)
		g.checkResult()
	}
	if g.sc != nil { // on_turn_end 事件(增援/對話)
		g.dialog = append(g.dialog, g.sc.Fire(g.st, "on_turn_end", "")...)
	}
	g.st.Turn++
	for _, u := range g.st.Units {
		u.Acted = false
	}
	g.sel, g.reach, g.moved = nil, nil, false
	g.checkResult()
}

func main() {
	ebiten.SetWindowSize(logicalW*2, logicalH*2)
	ebiten.SetWindowTitle("炎龍騎士團2 重製 (fd2_re)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(loadGame()); err != nil {
		log.Fatal(err)
	}
}
