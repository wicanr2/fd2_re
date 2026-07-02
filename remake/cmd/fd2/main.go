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
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
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
	walk    *walkAnim           // 移動動畫(沿路徑逐格走,FDICON 方向幀)
	camp    *campaign.Runner    // 劇本節點圖(doc 19;FD2_CAMPAIGN 啟用)
	campSel int                 // choice 節點游標
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
	shotPath   string
	shotFrame  int
	shotSeries string // 逐幀截圖目錄(FD2_SHOT_SERIES):戰鬥演出每幀存 frame_NN.png,演出結束自動退出
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
	tai     *ebiten.Image           // 我方腳下台座(TAI.DAT;0x29164 載 0x28c46,doc35 §3.3)
	panel   *ebiten.Image           // 狀態欄框素材(FDOTHER#5 LMI1 #22,149×42;含bevel+HP/MP標籤+槽,doc35 §4)
	dlgBox  *ebiten.Image           // 對話框框素材(FDOTHER#5 LMI1 #21,310×99;orig 下框(5,112)@320)
	fontNm  *Font                   // 狀態欄名字(整數尺寸 face,scale1 銳利)
	digits  [10]*ebiten.Image       // 狀態欄數字 0-9(LMI1 #31-40 原版 digit cell,白/藍影)
	redSil  map[*ebiten.Image]*ebiten.Image // 命中閃紅的全紅剪影快取(orig=VGA 色盤閃紅)
	figMeta map[int][][2]int                // FIGANI 每幀內嵌絕對螢幕座標 (dx,dy)@320(doc06;動畫走位全靠它)
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
	fpt              int // 播放速度(tick/幀;FD2_BATTLE_FPT 可調)
	terrain          int // 攻擊格地形索引(戰鬥背景 = 戰場地形,跟 FDFIELD 戰場資料有關)
}


// terrainAt 回傳某格的地形索引(戰鬥背景用;越界回 -1)。
func (g *Game) terrainAt(x, y int) int {
	if g.m == nil || x < 0 || y < 0 || x >= g.m.W || y >= g.m.H {
		return -1
	}
	return g.m.Tiles[y*g.m.W+x]
}

// ── campaign(劇本節點圖,doc 19)引擎接線 ──────────────────────────

// enterNode 進入 camp 目前節點:story→掛對白、battle→重開戰場、event→套旗標直通、choice/ending→等輸入。
func (g *Game) enterNode() {
	if g.camp == nil {
		return
	}
	n := g.camp.Node()
	if n == nil {
		return // 流程結束(game over)
	}
	switch n.Type {
	case "story":
		g.dialog = nil
		for i := len(n.Lines) - 1; i >= 0; i-- { // 反序堆疊:顯示取末端,Enter 逐句 pop
			g.dialog = append(g.dialog, battle.DialogLine{Speaker: n.Lines[i].Speaker, Text: n.Lines[i].Text})
		}
	case "battle":
		g.resetBattle(n.Units, n.Scenario)
	case "event":
		g.camp.Advance("")
		g.enterNode()
	case "choice":
		g.campSel = 0
	}
}

// resetBattle 重開一場戰鬥(campaign battle 節點;敗北重試也走這裡)。
func (g *Game) resetBattle(unitsPath, scnPath string) {
	if unitsPath == "" {
		unitsPath = "assets/map0_units.json"
	}
	if scnPath == "" {
		scnPath = "assets/scenarios/ch01.json"
	}
	if st, err := battle.Load(unitsPath); err == nil {
		g.st = st
	}
	g.result, g.sel, g.reach, g.moved = "", nil, nil, false
	g.atk, g.walk, g.dialog, g.msg = nil, nil, nil, ""
	if g.st != nil {
		if sc, err := battle.LoadScenario(scnPath); err == nil {
			g.sc = sc
			g.dialog = append(g.dialog, sc.Setup(g.st)...)
		}
	}
}

// campInput 處理 campaign 節點的輸入。回傳 true = 已攔截(擋掉戰場一般輸入)。
func (g *Game) campInput() bool {
	if g.camp == nil {
		return false
	}
	enter := inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)
	n := g.camp.Node()
	if n == nil {
		return true // game over:鎖定
	}
	switch n.Type {
	case "story":
		if enter {
			if len(g.dialog) > 0 {
				g.dialog = g.dialog[:len(g.dialog)-1]
			}
			if len(g.dialog) == 0 {
				g.camp.Advance("")
				g.enterNode()
			}
		}
		return true
	case "choice":
		vis := g.camp.Visible()
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.campSel > 0 {
			g.campSel--
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.campSel < len(vis)-1 {
			g.campSel++
		}
		if enter && len(vis) > 0 {
			g.camp.Advance(fmt.Sprintf("opt%d", g.campSel))
			g.enterNode()
		}
		return true
	case "ending":
		return true
	case "battle":
		if g.result != "" && enter { // 勝敗後 Enter → 依結果轉場(敗北可走敗北路線)
			g.camp.Advance(g.result)
			g.enterNode()
			return true
		}
		return false // 戰鬥照常
	}
	return false
}

// walkAnim 沿路徑逐格行走(玩家/AI 移動;FDICON 方向走動幀 + OffX/OffY 內插)。
type walkAnim struct {
	u    *battle.Unit
	path []battle.Cell // 含起點
	seg  int           // 目前段:path[seg] → path[seg+1]
	t    float64       // 段內進度 0→1
}

// battleFPT 戰鬥演出播放速度(tick/幀):環境變數 FD2_BATTLE_FPT 可調(調慢=數字大),預設 3。
func battleFPT() int {
	if v, e := strconv.Atoi(os.Getenv("FD2_BATTLE_FPT")); e == nil && v > 0 {
		return v
	}
	return 3
}

// newAtkAnim 建立全螢幕戰鬥演出(所有角色通用):攻方=攻擊動作(組×3+1)、守方=待機(組×3),
// 演出長度=幀數×fpt+尾段停格;位置/走位由 FIGANI 幀內嵌 (dx,dy) 資料驅動(doc06)。
func (g *Game) newAtkAnim(atkGroup, defGroup int, atkName, defName string,
	atkHP, atkMax, atkLV, atkMP, defLV, defMP, defHP0, defHP1, defMax, terrain int) *atkAnim {
	fpt := battleFPT()
	af := figaniIndex(atkGroup) + 1
	n := len(g.figani[af])
	if n == 0 {
		n = 15
	}
	total := (n + 4) * fpt // 尾段停格 4 幀時間
	return &atkAnim{atkFig: af, defFig: figaniIndex(defGroup), atkName: atkName, defName: defName,
		atkHP: atkHP, atkMax: atkMax, atkLV: atkLV, atkMP: atkMP, defLV: defLV, defMP: defMP,
		defHP0: defHP0, defHP1: defHP1, defMax: defMax, timer: total, total: total,
		fpt: fpt, terrain: terrain}
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

// loadFigMeta 載入 FIGANI 每幀內嵌絕對螢幕座標 (dx,dy)@320(assets/figani/meta.json;doc06:
// 幀標頭 +0/+2,動畫的走位/伸擊/突刺全靠逐幀 (dx,dy) 變化,引擎不需錨點/位移計算)。
func loadFigMeta() map[int][][2]int {
	out := map[int][][2]int{}
	raw, err := os.ReadFile("assets/figani/meta.json")
	if err != nil {
		return out
	}
	var m map[string][][2]int
	if json.Unmarshal(raw, &m) != nil {
		return out
	}
	for k, v := range m {
		if id, e := strconv.Atoi(k); e == nil {
			out[id] = v
		}
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
	if g.st == nil || g.walk != nil {
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
		case g.reach[cur] && g.st.UnitAt(g.curX, g.curY) == nil: // 移動到可達空格:沿路徑逐格走
			if p := g.st.Path(g.sel, g.curX, g.curY); len(p) >= 2 {
				g.walk = &walkAnim{u: g.sel, path: p}
			} else { // 理論上不會(reach 內必可達),保底瞬移
				g.sel.X, g.sel.Y = g.curX, g.curY
				g.moved = true
			}
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
		g.atk = g.newAtkAnim(g.sel.Fig, tgt.Fig, anm, nm,
			g.sel.HP, g.sel.MaxHP, g.sel.Lv, g.sel.MP, tgt.Lv, tgt.MP,
			defHP0, tgt.HP, tgt.MaxHP, g.terrainAt(g.curX, g.curY)) // 戰鬥背景 = 守方格地形
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
	// 移動動畫:沿路徑逐格走(方向幀 + OffX/OffY 內插;走完進入攻擊/待命階段)
	if w := g.walk; w != nil && g.m != nil {
		w.t += 0.22 // ~4-5 tick/格
		for w.t >= 1 && w.seg < len(w.path)-1 {
			w.t--
			w.seg++
		}
		if w.seg >= len(w.path)-1 { // 到位
			last := w.path[len(w.path)-1]
			w.u.X, w.u.Y = last.X, last.Y
			w.u.OffX, w.u.OffY = 0, 0
			g.walk = nil
			g.moved = true
		} else {
			a, b := w.path[w.seg], w.path[w.seg+1]
			w.u.Dir = dirToward(a.X, a.Y, b.X, b.Y)
			w.u.X, w.u.Y = b.X, b.Y // 單位掛在目標格,Off 從來源格內插到 0
			w.u.OffX = float64((a.X-b.X)*g.m.TileW) * (1 - w.t)
			w.u.OffY = float64((a.Y-b.Y)*g.m.TileH) * (1 - w.t)
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
			if v := os.Getenv("FD2_SHOT_ATTACK"); v != "" { // 全螢幕戰鬥演出(驗證用):亞雷斯打盜賊
				fig, _ := strconv.Atoi(v)
				g.atk = g.newAtkAnim(fig, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 2, 0, 28, 8, 28, 0)
			}
		}
		if g.shotSeries != "" { // 逐幀模式:演出播完才退出
			if g.frame > 2 && g.atk == nil {
				return ebiten.Termination
			}
		} else if g.frame > g.shotFrame {
			return ebiten.Termination
		}
	}
	if g.m == nil {
		return nil
	}
	if g.campInput() { // campaign 節點(story/choice/ending/勝敗轉場)攔截輸入
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
		if len(g.dialog) > 0 { // 對話框:原版素材(FDOTHER#5 LMI1 #21,310×99 素藍細邊框)+ orig 量測佈局
			dl := g.dialog[len(g.dialog)-1]
			// 依說話者切上/下框 + 左/右頭像(對照原版 orig_02_dialog:我方下框左頭像、對方/NPC 上框右頭像)
			upper := dl.Speaker >= 32 // >=32 為對方/敵/NPC(我方角色 id 0-31)
			// 框位置:模板匹配 orig 下框 (5,112)@320(底部裁 11px 超出畫面,原版如此);上框鏡射 y=-11
			bx, by := 10.0, 224.0
			if upper {
				by = -22
			}
			top := by
			if g.dlgBox != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(2, 2)
				op.GeoM.Translate(bx, by)
				screen.DrawImage(g.dlgBox, op)
			} else { // 無素材 fallback:純色框
				box := ebiten.NewImage(620, 198)
				box.Fill(color.RGBA{0x2c, 0x44, 0x84, 0xf2})
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(bx, by)
				screen.DrawImage(box, op)
			}
			// 頭像:大側臉,我方左(頭頂凸出框頂)、對方右(orig 佈局);嘴型 m0閉/m3開
			const ps = 2.6 // 80×80 DATO → ~208px(對照 orig 側臉高)
			hx, tx, ty := 16.0, 216.0, by+24
			hy := float64(logicalH) - 80*ps // 下框:底對齊畫布底
			if upper {
				hx = float64(logicalW) - 16 - 80*ps
				hy = -14 // 上框:頭像貼頂,臉朝下框
				tx = 32
				ty = by + 46
			}
			if fr := g.portraits[dl.Speaker]; len(fr) > 0 {
				mi := 0
				if g.mouthOpen && len(fr) > 3 {
					mi = 3
				}
				po := &ebiten.DrawImageOptions{}
				if upper { // 上框頭像水平鏡像(面左朝文字;原版下框走 0x4E8E1 鏡像 blit、上框 0x4E8AF,doc14/35)
					po.GeoM.Scale(-ps, ps)
					po.GeoM.Translate(hx+80*ps, hy)
				} else {
					po.GeoM.Scale(ps, ps)
					po.GeoM.Translate(hx, hy)
				}
				screen.DrawImage(fr[mi], po)
			} else {
				tx = 32
			}
			// 自動換行(框右緣內;原版每框最多 3 行,doc14)
			txt := []rune("『" + toFullWidth(dl.Text) + "』")
			perLine := int((bx + 620 - 16 - tx) / (fontSize * 1.7)) // 全形字寬 ≈ fontSize×scale
			if perLine < 1 {
				perLine = 1
			}
			for i := 0; len(txt) > 0 && i < 3; i++ {
				n := perLine
				if n > len(txt) {
					n = len(txt)
				}
				g.font.Draw(screen, string(txt[:n]), tx, ty+float64(i)*38, 1.7, color.RGBA{0xf0, 0xf4, 0xff, 0xff})
				txt = txt[n:]
			}
			_ = top
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
			if g.camp != nil {
				g.font.Draw(screen, "按 Enter 繼續", float64(logicalW)/2-70, float64(logicalH)/2+36, 1.0,
					color.RGBA{0xe0, 0xe0, 0xe0, 0xff})
			}
		}
	}
	g.drawCampaignUI(screen)

	// 截圖鉤子:指定幀把畫面存 PNG(無人值守驗證用)
	if g.shotPath != "" && g.frame == g.shotFrame {
		saveShot(screen, g.shotPath)
	}
}

// drawCampaignUI campaign 節點 UI:choice 選單 / ending 結語 / game over。
func (g *Game) drawCampaignUI(screen *ebiten.Image) {
	if g.camp == nil || g.font == nil {
		return
	}
	n := g.camp.Node()
	fillBox := func(x, y, w, h float64) {
		box := ebiten.NewImage(int(w), int(h))
		box.Fill(color.RGBA{0x10, 0x1c, 0x40, 0xe8})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y)
		screen.DrawImage(box, op)
	}
	switch {
	case n == nil: // 流程結束(無敗北路線的 game over)
		fillBox(0, float64(logicalH)/2-40, float64(logicalW), 80)
		g.font.Draw(screen, "GAME OVER", float64(logicalW)/2-90, float64(logicalH)/2-20, 2.0,
			color.RGBA{0xff, 0x70, 0x70, 0xff})
	case n.Type == "choice":
		vis := g.camp.Visible()
		h := 60 + float64(len(vis))*28
		fillBox(160, 120, 320, h)
		g.font.Draw(screen, n.Prompt, 176, 130, 1.1, color.RGBA{0xff, 0xe0, 0x90, 0xff})
		for i, o := range vis {
			c := color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			pre := "　"
			if i == g.campSel {
				c = color.RGBA{0xff, 0xff, 0xff, 0xff}
				pre = "▶"
			}
			g.font.Draw(screen, pre+o.Label, 190, 162+float64(i)*28, 1.0, c)
		}
	case n.Type == "ending":
		fillBox(0, float64(logicalH)/2-60, float64(logicalW), 120)
		g.font.Draw(screen, n.Text, float64(logicalW)/2-float64(len([]rune(n.Text)))*9, float64(logicalH)/2-30, 1.4,
			color.RGBA{0xff, 0xe0, 0x90, 0xff})
	}
}

// redSilhouette 全紅剪影(快取):orig 命中閃紅=VGA DAC 把 sprite 色盤整組設紅(0x11d40),
// 是整片飽和紅,非乘法調色 → 逐像素把非透明處塗紅,一次生成後快取。
func (g *Game) redSilhouette(src *ebiten.Image) *ebiten.Image {
	if g.redSil == nil {
		g.redSil = map[*ebiten.Image]*ebiten.Image{}
	}
	if r, ok := g.redSil[src]; ok {
		return r
	}
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			_, _, _, al := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			if al > 0x4000 {
				out.Set(x, y, color.RGBA{0xd0, 0x10, 0x10, 0xff})
			}
		}
	}
	r := ebiten.NewImageFromImage(out)
	g.redSil[src] = r
	return r
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
	// 原版 320×200 精確 layout(網格量測)→ 本畫布 ×2。黑底(畫面外圍黑邊)
	screen.Fill(color.RGBA{0, 0, 0, 0xff})
	if g.bg != nil { // BG(doc35:320×100 原生貼 (0,50) → ×2 貼 (0,100))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(0, 100)
		screen.DrawImage(g.bg, op)
	}
	// 繪製順序(doc35 §4 RE:演出 0x28a6c 內「狀態欄 0x2a289(0x28ce7/0x28d62)先畫、
	// figure(0x28e76 起 0x29164/0x2939d)後畫」→ figure z-order 高於狀態欄,動畫蓋住欄、動畫完整)。
	const sc = 2.0 // doc35:無 runtime 縮放,FIGANI 原生尺寸 ×2(原版 320→畫布 640)

	// 資料驅動動畫(doc06,所有角色通用):每幀貼幀標頭內嵌的絕對螢幕座標 (dx,dy)@320 ×2,
	// 走位/伸擊/突刺全在資料裡。播放速度 = a.fpt(tick/幀,FD2_BATTLE_FPT 可調);
	// 命中幀 = 攻擊動畫的倒數第 4 幀(FIGANI_013:f11 黃劈擊弧,其後 3 幀為突刺收勢——通用推定)。
	atkFrames := g.figani[a.atkFig]
	fpt := a.fpt
	if fpt < 1 {
		fpt = 3
	}
	atkFi := prog / fpt
	if len(atkFrames) > 0 && atkFi >= len(atkFrames) {
		atkFi = len(atkFrames) - 1
	}
	impactFi := len(atkFrames) - 4
	if impactFi < 1 {
		impactFi = 1
	}
	impactS := impactFi * fpt
	impactE := impactS + 8
	// (1) 狀態欄先畫(會被 figure 蓋住一部分,如原版)
	if g.font != nil {
		dhp := a.defHP0 // 命中當下快抽(orig impact 幀 HP 已抽完)
		if prog >= impactS {
			t := float64(prog-impactS) / float64(impactE-impactS)
			if t > 1 {
				t = 1
			}
			dhp = a.defHP0 + int(float64(a.defHP1-a.defHP0)*t)
		}
		// 位置=模板匹配 orig:我方 (171,4)@320、敵方 (0,154)@320(下欄匹配 err=0 像素全等)
		g.drawBattlePanel(screen, 342, 8, a.atkName, a.atkLV, a.atkHP, a.atkMax, a.atkMP) // 我方亞雷斯右上
		g.drawBattlePanel(screen, 0, 308, a.defName, a.defLV, dhp, a.defMax, a.defMP)     // 敵方盜賊左下
	}

	// (2) 敵方盜賊 figure(正面;蓋住狀態欄):4 幀待機呼吸循環,貼各幀內嵌 (dx,dy)
	if fr := g.figani[a.defFig]; len(fr) > 0 {
		fi := (prog / 6) % len(fr)
		img := fr[fi]
		// 命中閃紅:整片飽和紅剪影(orig=VGA 色盤整組設紅,非乘法調色),快速交替閃
		if prog >= impactS && prog < impactE && (prog/2)%2 == 0 {
			img = g.redSilhouette(img)
		}
		dx, dy := 16.0, 41.0
		if m := g.figMeta[a.defFig]; fi < len(m) {
			dx, dy = float64(m[fi][0]), float64(m[fi][1])
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(dx*sc, dy*sc)
		screen.DrawImage(img, op)
	}
	// (2.5) 我方台座(TAI_004;模板匹配 orig 台座左上=(165,157)@320 → ×2=(330,314))
	if g.tai != nil {
		tb := g.tai.Bounds()
		tw, th := float64(tb.Dx())*sc, float64(tb.Dy())*sc
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(482-tw/2, 356-th/2)
		screen.DrawImage(g.tai, op)
	}
	// (3) 我方亞雷斯 figure(背影,踩台座;蓋住狀態欄):攻擊幀序播放(停末幀=突刺收勢),
	// 位置=各幀內嵌 (dx,dy)(f11 劈擊伸左、f12-14 突刺,走位在資料裡,不需 lunge 計算)
	if len(atkFrames) > 0 {
		img := atkFrames[atkFi]
		dx, dy := 141.0, 3.0
		if m := g.figMeta[a.atkFig]; atkFi < len(m) {
			dx, dy = float64(m[atkFi][0]), float64(m[atkFi][1])
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(dx*sc, dy*sc)
		screen.DrawImage(img, op)
	}
	if g.shotSeries != "" { // 逐幀截圖(GIF/分鏡素材)
		saveShot(screen, fmt.Sprintf("%s/frame_%02d.png", g.shotSeries, prog))
	}
}

// drawBattlePanel 原版戰鬥狀態欄:用 FDOTHER#5 LMI1 #22 框素材(149×42,含 bevel + HP/MP標籤 +
// LV‧ + 血條槽,codec 反組譯 0x4e916 破解,doc35 §4),只疊上名字 / LV數字 / 血條填充 / HP-MP數值。
// 框內槽 native:HP y22-26、MP y31-35、x26-145(量測)。
func (g *Game) drawBattlePanel(screen *ebiten.Image, x, y float64, name string, lv, hp, mx, mp int) {
	panel := g.panel
	// orig 是 149×42 原生尺寸 blit(非拉伸滿半屏;網格比對 v37 抓到的差異)→ 固定 ×2
	const sc = 2.0
	fillRect := func(bx, by, bw, bh float64, c color.RGBA) {
		if bw < 1 {
			return
		}
		im := ebiten.NewImage(int(bw), int(bh))
		im.Fill(c)
		o := &ebiten.DrawImageOptions{}
		o.GeoM.Translate(bx, by)
		screen.DrawImage(im, o)
	}
	if panel != nil { // 框素材(bevel + HP/MP標籤 + LV‧ + 槽 全來自原版;palette 已 6→8bit 校正)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(x, y)
		screen.DrawImage(panel, op)
	}
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	// 血條填充(槽 native 量測:x21–123 寬102、HP y22–26、MP y31–35 各5列;靠左對齊槽左端)
	// 高度用四捨五入避免 int 截斷造成槽上下露紅邊
	rnd := func(v float64) float64 { return float64(int(v + 0.5)) }
	slotX, slotW := x+21*sc, 102*sc
	slotH, lightH := rnd(5*sc), rnd(1*sc)
	drawFill := func(slotY, frac float64, light, body color.RGBA) {
		if frac > 1 {
			frac = 1
		} else if frac < 0 {
			frac = 0
		}
		w := rnd(slotW * frac)
		fillRect(slotX, slotY, w, lightH, light)             // 頂邊亮(orig 漸層)
		fillRect(slotX, slotY+lightH, w, slotH-lightH, body) // 本體
	}
	drawFill(rnd(y+22*sc), float64(hp)/float64(mx),
		color.RGBA{0xf8, 0xe8, 0x80, 0xff}, color.RGBA{0xf0, 0xc8, 0x30, 0xff}) // HP 黃
	mpmx := mp
	if mpmx < 1 {
		mpmx = 1
	}
	drawFill(rnd(y+31*sc), float64(mp)/float64(mpmx),
		color.RGBA{0xf0, 0x70, 0x60, 0xff}, color.RGBA{0xc8, 0x28, 0x20, 0xff}) // MP 紅
	// 排版(對照 orig 放大量測,native):名(8,2) 16px;LV數字接框內「LV‧」後(133,3) 9px;
	// HP/MP 數值與槽同列(125,20)/(125,29) 8px
	_ = white
	// 名字:TTF + 深藍描邊(仿原版點陣的暗邊;字形風格為 TTF 既定決策,非像素級)
	if g.fontNm != nil {
		nx, ny := rnd(x+8*sc), rnd(y+2*sc)-2
		dk := color.RGBA{0x20, 0x30, 0x60, 0xff}
		for _, o := range [][2]float64{{-2, 0}, {2, 0}, {0, -2}, {0, 2}, {2, 2}} {
			g.fontNm.Draw(screen, name, nx+o[0], ny+o[1], 1.0, dk)
		}
		g.fontNm.Draw(screen, name, nx, ny, 1.0, color.RGBA{0xe0, 0xee, 0xff, 0xff})
	}
	// 數字:原版 digit cell 素材(6×8,advance 7px native),100% 還原
	drawNum := func(s string, nxN, nyN float64) {
		for k, ch := range s {
			if ch < '0' || ch > '9' || g.digits[ch-'0'] == nil {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(sc, sc)
			op.GeoM.Translate(x+(nxN+float64(k)*7)*sc, y+nyN*sc)
			screen.DrawImage(g.digits[ch-'0'], op)
		}
	}
	// 位置=模板匹配 orig 定位(LV/HP/MP 首位數字 local (132,4)/(126,21)/(126,30),advance 7)
	if lv > 0 {
		drawNum(fmt.Sprintf("%02d", lv), 132, 4)
	}
	drawNum(fmt.Sprintf("%03d", hp), 126, 21)
	drawNum(fmt.Sprintf("%03d", mp), 126, 30)
}

// drawStatBar 狀態條(暗槽 + 填充);暗槽 = 填充色暗版(對照 orig:空槽呈暗黃/暗紅,非統一黑)。
func drawStatBar(screen *ebiten.Image, x, y, w, frac float64, c color.RGBA) {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	slot := ebiten.NewImage(int(w), 9)
	slot.Fill(color.RGBA{c.R / 4, c.G / 4, c.B / 4, 0xff}) // 暗槽=填充色暗版
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
	g.shotSeries = os.Getenv("FD2_SHOT_SERIES")
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
	g.figMeta = loadFigMeta()
	if raw, e := os.ReadFile("assets/bg/bg.png"); e == nil { // 戰鬥背景(BG.DAT)
		if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
			g.bg = ebiten.NewImageFromImage(im)
		}
	}
	if raw, e := os.ReadFile("assets/tai/tai_004.png"); e == nil { // 我方台座(TAI_004 綠草橢圓)
		if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
			g.tai = ebiten.NewImageFromImage(im)
		}
	}
	if raw, e := os.ReadFile("assets/ui/panel.png"); e == nil { // 狀態欄框(LMI1 #22)
		if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
			g.panel = ebiten.NewImageFromImage(im)
		}
	}
	if raw, e := os.ReadFile("assets/ui/dialog.png"); e == nil { // 對話框框(LMI1 #21)
		if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
			g.dlgBox = ebiten.NewImageFromImage(im)
		}
	}
	g.font = loadFont()
	// 狀態欄名字專用整數尺寸 face(scale 1.0 繪製,避免非整數縮放模糊);orig 名墨高 13px→face 28
	g.fontNm = loadFontSized(28)
	for k := 0; k < 10; k++ { // 原版數字 cell(LMI1 #31-40)
		if raw, e := os.ReadFile(fmt.Sprintf("assets/ui/digit_%d.png", k)); e == nil {
			if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
				g.digits[k] = ebiten.NewImageFromImage(im)
			}
		}
	}
	if cp := os.Getenv("FD2_CAMPAIGN"); cp != "" { // 劇本節點圖模式(doc 19;放最後,story 對白不被開場 Setup 蓋掉)
		if cp == "1" {
			cp = "assets/scenarios/campaign.json"
		}
		if c, err := campaign.Load(cp); err == nil {
			g.camp = campaign.NewRunner(c)
			g.enterNode()
		} else {
			g.loadErr = "campaign: " + err.Error()
		}
	}
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
