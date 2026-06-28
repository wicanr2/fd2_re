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
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	sel   *battle.Unit
	reach map[battle.Cell]bool
	// 地圖單位 sprite(FIGANI 待機分鏡):fig index → 幀序列
	sprites map[int][]*ebiten.Image
	font    *Font // 原版點陣中文字型(doc 08)
}


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

// confirm 處理 Enter/Space:選取我方單位顯示移動範圍,或移動到可達格 / 原地待機。
func (g *Game) confirm() {
	if g.st == nil {
		return
	}
	cur := battle.Cell{X: g.curX, Y: g.curY}
	if g.sel == nil {
		u := g.st.UnitAt(g.curX, g.curY)
		if u != nil && u.Camp == battle.Own && !u.Acted {
			g.sel = u
			g.reach = g.st.Reachable(u)
		}
		return
	}
	switch {
	case g.curX == g.sel.X && g.curY == g.sel.Y: // 原地待機
		g.sel.Acted = true
		g.sel, g.reach = nil, nil
	case g.reach[cur] && g.st.UnitAt(g.curX, g.curY) == nil: // 移動
		g.sel.X, g.sel.Y = g.curX, g.curY
		g.sel.Acted = true
		g.sel, g.reach = nil, nil
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
			boxH := 108.0 // 原版約佔底部 1/4~1/3
			top := float64(logicalH) - boxH
			box := ebiten.NewImage(logicalW, int(boxH))
			box.Fill(color.RGBA{0x10, 0x18, 0x48, 0xf0})
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(0, top)
			screen.DrawImage(box, op)
			edge := ebiten.NewImage(logicalW, 2)
			edge.Fill(color.RGBA{0xc8, 0xa0, 0x40, 0xff})
			oe := &ebiten.DrawImageOptions{}
			oe.GeoM.Translate(0, top)
			screen.DrawImage(edge, oe)
			tx := 16.0
			// 左頭像(嘴型:目前 m0 靜態;開合動畫節奏待 RE 原版確認)
			if fr := g.portraits[dl.Speaker]; len(fr) > 0 {
				mi := 0 // 嘴型:m0 閉 / m3 開(原版 0x16559,doc14)
				if g.mouthOpen && len(fr) > 3 {
					mi = 3
				}
				s := (boxH - 16) / 80.0
				po := &ebiten.DrawImageOptions{}
				po.GeoM.Scale(s, s)
				po.GeoM.Translate(12, top+8)
				screen.DrawImage(fr[mi], po)
				tx = 12 + 80*s + 14
			}
			g.font.Draw(screen, "『"+toFullWidth(dl.Text)+"』", tx, top+18, 1.6, color.RGBA{0xff, 0xff, 0xf0, 0xff})
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

// drawUnitSprite 畫一個單位:腳下陣營標記 + FIGANI 待機動畫 sprite(縮放/底部對齊/循環)+ HP bar。
func (g *Game) drawUnitSprite(screen *ebiten.Image, x, y, w, h float64, u *battle.Unit) {
	// 腳下陣營色標(敵我同 sprite,靠此區分)
	mark := ebiten.NewImage(int(w)-6, 4)
	mc := campColor(u.Camp)
	mark.Fill(mc)
	om := &ebiten.DrawImageOptions{}
	om.GeoM.Translate(x+3, y+h-3)
	screen.DrawImage(mark, om)

	frames := g.sprites[u.Fig]
	if len(frames) > 0 {
		img := frames[(g.frame/12)%len(frames)] // 待機循環(手擺動;每 12 幀換一張)
		// FDICON 24×24 = 格大小,直接貼;略上移讓單位「站」在格上
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y-4)
		if u.Acted {
			op.ColorScale.Scale(0.55, 0.55, 0.6, 1) // 已行動變暗(對映原版灰階)
		}
		screen.DrawImage(img, op)
	} else {
		drawUnit(screen, x, y, w, h, mc, u) // fallback 色塊
		return
	}
	// HP bar(格頂)
	bw := w - 6
	frac := float64(u.HP) / float64(u.MaxHP)
	if frac < 0 {
		frac = 0
	}
	bar := ebiten.NewImage(int(bw*frac)+1, 2)
	bar.Fill(color.RGBA{0x30, 0xff, 0x30, 0xff})
	ob := &ebiten.DrawImageOptions{}
	ob.GeoM.Translate(x+3, y+1)
	screen.DrawImage(bar, ob)
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
	g.font = loadFont()
	return g
}

// endTurn 結束當前回合:觸發 on_turn_end 事件(增援等),回合 +1,清除已行動。
// 回合無上限(doc 27);只由劇本事件決定勝負。
func (g *Game) endTurn() {
	if g.st == nil {
		return
	}
	if g.sc != nil {
		g.dialog = append(g.dialog, g.sc.Fire(g.st, "on_turn_end", "")...)
	}
	g.st.Turn++
	for _, u := range g.st.Units {
		u.Acted = false
	}
	g.sel, g.reach = nil, nil
}

func main() {
	ebiten.SetWindowSize(logicalW*2, logicalH*2)
	ebiten.SetWindowTitle("炎龍騎士團2 重製 (fd2_re)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(loadGame()); err != nil {
		log.Fatal(err)
	}
}
