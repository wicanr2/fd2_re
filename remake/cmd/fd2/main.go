// 炎龍騎士團2 重製 — Go/Ebiten 垂直切片(MVP)。
//
// 目標:證明「Go/Ebiten 跑得起來,且讀得到我們逆向出的資料」。
// 本切片:載入一張戰場(tileset PNG + 地圖 JSON)→ 用 hi-res 畫布渲染 →
//
//	方向鍵 / WASD / 觸控移動游標,相機跟隨。桌面 / Web(WASM)/ 手機共用。
//
// 資產(玩家自備原版後由 tools/ 產生,不隨庫散布):
//
//	assets/tileset.png  一張 24×24 圖塊的網格圖(cols 欄)
//	assets/map.json     {"w","h","tileW","tileH","cols","tiles":[地形索引...]}
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
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

const (
	logicalW = 640 // hi-res 內部畫布(CJK/觀感原則:拉畫布、別縮字)
	logicalH = 400
	// storyZoom:story 場景(cutscene)世界層放大倍率。原版視窗固定 13×8 格
	// (312×192px @320×200,doc25 0x11eee),remake 戰場自訂 640 寬 FOV 一屏裝下整廳,
	// 走入/運鏡完全失去意義(使用者 2-1);story 場景世界層 2×(48px/格,視野 13.3×8.3 格)
	// 即還原原版取景與長廊運鏡。戰場是否同步 2× 另議(動 HUD/指令環佈局,worklist)。
	storyZoom = 2
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
	m                 *MapData
	tileset           *ebiten.Image
	tiles             []*ebiten.Image     // 切好的圖塊
	st                *battle.State       // 戰鬥狀態(單位)
	sc                *battle.Scenario    // 劇本(事件系統,doc 29)
	dialog            []battle.DialogLine // 待顯示對話(事件產生,含說話者)
	storyBG           bool                // 場景背景模式(story 節點指定 Map):鏡頭固定不跟游標,不畫單位/游標/HUD(doc23 §4)
	storyActors       []battle.Unit       // 原版目前已 materialize 的 scene unit array；index 只在該 load/spawn 時序內有意義
	storyRoster       []battle.Unit       // LOADCH 保留的 FDFIELD records；SPAWN 按 group 順序 append 到 storyActors
	storySpawned      map[int]bool        // 原版 group 已 materialize；防止 handler 重複 SPAWN 時重複 append
	partyMembers      map[int]bool        // JOIN 建立的永久玩家名冊；key=原版 0..31 charID，不使用 NPC portrait
	partyJoinOrder    []int               // JOIN 首次出現順序；章0 cutscene 的 party runtime slot 以此為準
	partyRoster       map[int]battle.Unit // 0x11506 戰後同步的跨關角色能力／HP／MP／經驗快照
	partyDeploy       map[int]bool        // preparation 0x318ad 的本戰出擊勾選；不改永久 JOIN 名冊
	prepIDs           []int               // preparation UI 角色順序（JOIN chronology）
	prepSel           int                 // preparation UI 游標
	prepLimit         int                 // preparation UI 原版出擊上限（15，末段 19）
	churchSel         int                 // church service menu cursor (0..3)
	churchMode        string              // menu / revive / class
	churchIDs         []int               // current church candidate ids
	churchClassID     int                 // selected class-change candidate
	churchBranches    []campaign.ClassChangeBranch
	classChangeTable  campaign.ClassChangeTable
	classChangeGrowth map[int]campaign.ClassChangeGrowth
	handlerChapter    int              // 原版 [0x53c03]；set_chapter 與無立即數 LOADCH 的 resource chapter
	storyWalks        []*storyWalkJob  // 場景走位動畫佇列(doc46 §5.3);逐幀推進、完成後移除
	storyAutoAdvance  int              // story 節點無對白時的自動轉場倒數幀(doc46 行軍蒙太奇,0=不自動)
	storyView         *ebiten.Image    // story 場景離屏世界層(320×200,放大 storyZoom 倍貼上畫布;2-1 原版取景)
	walkFirst         bool             // 本節點:進場走位走完才顯示對白(campaign.Node.WalkFirst)
	followWalk        bool             // 本節點:走位期間鏡頭跟隨走位者(campaign.Node.FollowWalk;beat walk 依 Follow 逐拍設值)
	camMaxY           float64          // 本節點:鏡頭 Y 上限(campaign.Node.CamMaxY;0=不限)
	camPan            *camPanJob       // beat「pan」進行中(doc50 §1);storyBG 專用,與 followWalk 互斥
	focusJob          *focusUnitJob    // beat「focus_unit」：依原版 0x12cea 先 X 後 Y 逐格移動游標／鏡頭
	actJob            *actPoseJob      // beat「act」進行中(近似姿態循環,見 actPoseJob 註解)
	beats             []campaign.Beat  // 目前 cutscene 節點的過場原語序列(doc50 §2)
	beatIdx           int              // 目前執行到第幾拍(-1=尚未開始)
	beatDelay         int              // beat「delay」剩餘幀數(0=非等待中)
	battleEvent       *battleEventRun  // 戰場事件的阻塞 action 序列；與 campaign BeatRunner 分離
	battleEventDelay  int              // battle event delay 剩餘幀數
	campLines         []campaign.Line  // cutscene 節點載入的章文本(dialog beat 依 Line/Count 取子段)
	dlgShown          int              // 對話框目前顯示的說話者(dlgNone=無;換人時播縮/展動畫)
	dlgUpper          *bool            // 與 dlgShown 同步的上/下框覆蓋(來自 DialogLine.Upper;nil=沿用預設規則)
	dlgPhase          int              // 對話框動畫相位:0=常態 1=縮小(換人前收合) 2=展開
	dlgT              int              // 對話框動畫相位內計時(幀)
	dlgPage           int              // 目前對白的頁碼(0起);一句>3行時分頁,Enter 先翻頁翻完才換句(使用者回饋 2026-07-05)
	fade              *storyFade       // 場景淡出/淡入轉場(doc46 §5.2)
	walk              *walkAnim        // 移動動畫(沿路徑逐格走,FDICON 方向幀)
	camp              *campaign.Runner // 劇本節點圖(doc 19;FD2_CAMPAIGN 啟用)
	campSel           int              // choice 節點游標
	// 開頭動畫/主選單(title.go,doc23)
	titleAssets *titleAssets
	titlePhase  string  // "scroll"→"menu"→""(進遊戲)
	scrollY     float64 // 捲動來源列(535→0)
	titleSel    int
	titleFlash  int
	titleTick   int
	// 開場 AFM 過場(title.go cutscene phase)
	cutIdx   int
	cutFrame int
	cutTick  int
	cutCur   []*ebiten.Image
	// radial 指令環(原版 4 圖示十字繞單位,doc13 [0x3C57]:↑0/←1/→2/↓3)+ 法術
	ring               bool
	ringSel            int
	ringIcons          [4]*ebiten.Image // 0上=道具 1左=攻擊 2右=魔法/狀態 3下=待機
	spellOpen          bool
	spellSel           int
	castSp             *battle.Spell // 施法目標選擇中
	spells             []battle.Spell
	bgm                *audio.Player // BGM(doc12 play_bgm 語意:同曲不重播)
	bgmCur             string
	bgmSource          string                // 音源設定 "fm"/"mt32"(settings.go;F2 切換)
	debug              bool                  // F3:開發除錯 HUD(座標/陣營原文等)
	unitLabels         bool                  // FD2_UNIT_LABELS=1:cutscene sprite 左上標 [idx]fig+名+座標(協助回報/對映原版 slot)
	cutsceneLog        bool                  // FD2_CUTSCENE_LOG=1:過場 node/beat/走位逐步 log 到 stderr(協助對原版資料比對)
	banner             string                // 回合橫幅文字(PLAYER/ENEMY PHASE)
	bannerT            int                   // 橫幅剩餘 tick
	sfx                map[int][]byte        // SFX PCM(doc36 FDOTHER#31 14樣本)
	sfxSwing           []byte                // 戰鬥揮擊音(doc36 戰鬥池 #48-64 sub0,七池共用)
	sfxImpact          []byte                // 命中音(近似:最短最尖池;attack_id→sfx 對照表 doc36 未 RE)
	sfxDeath           []byte                // 陣亡/重擊音(近似:最長池)
	prevCurX, prevCurY int                   // 游標移動音偵測
	aiBusy             bool                  // AI 回合進行中(逐單位行走動畫)
	deathRewarded      map[*battle.Unit]bool // 每個死亡 transition 的 reward 只執行一次
	rng                *rand.Rand            // 施法擲骰(FD2_SEED 可固定,headless 重現)
	gold               int                   // 金幣(商店)
	items              []string              // 隊伍道具(名稱;道具效果待實裝)
	shopSel            int                   // 商店游標
	shopRecipientSel   int
	shopRecipients     []int
	shopPicking        bool
	shopPending        campaign.Good
	shopEquipPrompt    bool
	shopEquipUnit      int
	shopEquipSlot      int
	shopItemTypes      map[int]int
	shopEquipTypes     map[int][]int
	shopItemPrices     map[int]int
	shopItemStats      map[int]campaign.ItemStats
	reviveFeeRates     []int  // church 0x30dc3 class fee words
	shopMode           string // buy or sell
	shopSellPicking    bool
	shopSellUnitSel    int
	shopSellSlotSel    int
	portraits          map[int][]*ebiten.Image // DATO 頭像:肖像 id → 4 嘴型幀
	mouthOpen          bool                    // 嘴型動畫狀態(原版 0x16d00:m0閉/m3開)
	mouthTimer         int                     // 閉嘴倒數(原版 rand%30+2 tick)
	curX               int
	curY               int
	camX               float64
	camY               float64
	loadErr            string
	// 截圖鉤子(FD2_SHOT=path 啟用):第 shotFrame 幀存 PNG 後自動退出(有界,供無人值守驗證)
	frame      int
	shotPath   string
	shotFrame  int
	shotSeries string // 逐幀截圖目錄(FD2_SHOT_SERIES):戰鬥演出每幀存 frame_NN.png,演出結束自動退出
	shotTurn   int    // 截圖前自動推進到第 N 回合(FD2_SHOT_TURN,驗證增援進場)
	shotCurX   int    // 截圖時把游標放這(FD2_SHOT_CUR=x,y)
	shotCurY   int
	shotSel    bool // 截圖前自動選取游標單位(FD2_SHOT_SELECT=1)
	// 選取狀態
	sel                *battle.Unit
	reach              map[battle.Cell]bool
	selOrigX, selOrigY int    // 選取單位當下的原始格(ESC 取消移動時退回,playfix #4)
	moved              bool   // 已選單位是否移動完(進入攻擊階段)
	result             string // 勝負:""/win/lose
	msg                string // 短訊息(攻擊傷害等)
	// 地圖單位 sprite(FDICON 待機分鏡):fig index → 幀序列
	sprites  map[int][]*ebiten.Image
	figani   map[int][]*ebiten.Image         // 攻擊全身動畫(FIGANI):fig → 幀序列
	atk      *atkAnim                        // 進行中的攻擊演出
	bg       *ebiten.Image                   // 戰鬥背景(BG.DAT,by 戰場;map0=BG_004 森林)
	tai      *ebiten.Image                   // 我方腳下台座(TAI.DAT;0x29164 載 0x28c46,doc35 §3.3)
	panel    *ebiten.Image                   // 狀態欄框素材(FDOTHER#5 LMI1 #22,149×42;含bevel+HP/MP標籤+槽,doc35 §4)
	dlgBox   *ebiten.Image                   // 對話框框素材(FDOTHER#5 LMI1 #21,310×99;orig 下框(5,112)@320)
	dlgGrad  *ebiten.Image                   // 對話框內部漸層(比對頭像底色 40,69,138→56,85,154 消接縫色差;lazy 建)
	fontNm   *Font                           // 狀態欄名字(整數尺寸 face,scale1 銳利)
	digits   [10]*ebiten.Image               // 狀態欄數字 0-9(LMI1 #31-40 原版 digit cell,白/藍影)
	redSil   map[*ebiten.Image]*ebiten.Image // 命中閃紅的全紅剪影快取(orig=VGA 色盤閃紅)
	redFlash *ebiten.Image                   // 命中全螢幕紅罩(orig=DAC 整組色盤設紅→整片泛紅)
	dim      *ebiten.Image                   // 全螢幕暗化/底板共用(回合橫幅、單位面板)
	figMeta  map[int][][2]int                // FIGANI 每幀內嵌絕對螢幕座標 (dx,dy)@320(doc06;動畫走位全靠它)
	font     *Font                           // 原版點陣中文字型(doc 08)
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
	fpt              int  // 播放速度(tick/幀;FD2_BATTLE_FPT 可調)
	atkOwn           bool // 攻方是否我方(狀態欄按陣營:我方欄右上/敵方欄左下)
	terrain          int  // 攻擊格地形索引(戰鬥背景 = 戰場地形,跟 FDFIELD 戰場資料有關)
}

// storyWalkJob 場景走位動畫(doc46 §5.3):cutscene 固定路徑位移,非玩家可控,重用
// battle.Unit.OffX/OffY/Dir 插值(同行軍/移動動畫的畫法),完成時呼叫 then(可為 nil)。
type storyWalkJob struct {
	actor        int // g.storyActors 索引
	fromX, fromY int
	toX, toY     int
	t, frames    int
	finalDir     int // 走完後面向(-1=保留走位末向;>=0=設定,如進場走位面向 actor 目標 dir)
	scrollFollow bool
	fromCamY     float64 // 0x13185 專用：超過原版 screen-row 安全帶後與 actor 同速捲圖
	scrollFree   int     // actor 可先上移而不捲圖的格數（screenY-1）
	then         func()
}

// storyFade 場景淡出/淡入轉場(doc46 §5.2):story 節點換場時不硬切。out=true 淡出(變黑)、
// false 淡入(轉亮);走完(不分 out/in)觸發 then(可為 nil;beat fade 靠它接回下一拍)。
type storyFade struct {
	out   bool
	t     int
	total int
	then  func()
}

// storyFadeFrames 淡出/淡入各自幀數(60fps;doc46 要求 0.5–1s,先做快版 0.6s,實測後可調)。
const storyFadeFrames = 36

// camPanJob beat「pan」原語。tileStep 精確重現 0x135dd：先 X 後 Y、每 tick
// 移一個 tile 並 redraw；舊 authored scenes 保留 frames 線性內插相容模式。
type camPanJob struct {
	fromX, fromY float64
	toX, toY     float64
	t, frames    int
	tileStep     bool
	then         func()
}

// battleEventRun preserves the authored order of on-turn battle actions.
// It deliberately does not reuse campaign beats: finishing a battle event
// must finish the turn, never advance the campaign node.
type battleEventRun struct {
	actions []battle.Action
	index   int
	then    func()
}

// focusUnitJob 保留 0x12cea 的阻塞移動目標。原版每輪只移動一格，X 到位後才移 Y；
// 游標接近 13×8 視窗邊緣時才推進 map origin，並非直接把目標置中。
type focusUnitJob struct {
	targetX, targetY int
}

// actPoseJob 承接 beat「act」。acting 非空時按原版 0x1366a 規則播放：正常 frame
// (bit7=0)的每一 Beat 都走一格、每格 7 個內插 tick；special frame(bit7=1)原地停留。
// poses/frames 則保留給尚未轉錄的舊場景作原地姿態相容。
type actPoseJob struct {
	actor  int
	poses  []int
	frames int
	t, idx int
	acting []campaign.ActingFrame
	frame  int
	beat   int
	tick   int
	then   func()
}

// startFadeTransition 淡出 storyFadeFrames 幀 → 執行 action(通常是 Advance+enterNode 或
// 換擺位)→ 淡入 storyFadeFrames 幀。
func (g *Game) startFadeTransition(action func()) {
	g.fade = &storyFade{out: true, total: storyFadeFrames, then: func() {
		action()
		g.fade = &storyFade{out: false, total: storyFadeFrames}
	}}
}

// advanceStoryNode 對白播完離開 story 節點:若節點設 ExitWalk(s),先讓對應 actor(可多人,
// 使用者回饋 #A:草地小徑幕結尾索爾+亞雷斯一起走離)全部走完一段路(doc46 §5.3)再淡出換場;
// 否則直接淡出換場。多個 exit walk 共用同一個「全部走完才轉場」計數器,不是走完第一個就轉場。
func (g *Game) advanceStoryNode(n *campaign.Node) {
	doAdvance := func() {
		g.startFadeTransition(func() {
			g.camp.Advance("")
			g.enterNode()
		})
	}
	var walks []campaign.ActorWalk
	if n.ExitWalk != nil {
		walks = append(walks, *n.ExitWalk)
	}
	walks = append(walks, n.ExitWalks...)
	if len(walks) == 0 {
		doAdvance()
		return
	}
	remaining := len(walks)
	onDone := func() {
		remaining--
		if remaining == 0 {
			doAdvance()
		}
	}
	for _, ew := range walks {
		for i := range g.storyActors {
			if g.storyActors[i].Fig == ew.Fig {
				u := &g.storyActors[i]
				finalDir := -1 // 預設保留走位末段方向;ew.Dir 指定則覆蓋(如索爾走到亞雷斯旁定住面右)
				if ew.Dir != nil {
					finalDir = *ew.Dir
				}
				g.storyWalks = append(g.storyWalks, &storyWalkJob{
					actor: i, fromX: u.X, fromY: u.Y,
					toX: ew.ToX, toY: ew.ToY,
					frames: ew.Frames, finalDir: finalDir, then: onDone,
				})
				break
			}
		}
	}
}

// stepStoryWalks 逐幀推進場景走位動畫佇列,完成的 job 更新最終座標、呼叫 then 後移除。
// 走位沿格線(先長軸後短軸,不斜切)——原版單位走位以戰場格為單位軸向移動
// (使用者回饋 2026-07-04 #4:「走路沒照著格子走」=舊版起訖點直線內插會斜切格線)。
func (g *Game) stepStoryWalks() {
	if len(g.storyWalks) == 0 {
		return
	}
	sgn := func(v int) float64 {
		if v > 0 {
			return 1
		} else if v < 0 {
			return -1
		}
		return 0
	}
	absi := func(v int) int {
		if v < 0 {
			return -v
		}
		return v
	}
	live := g.storyWalks[:0]
	for _, w := range g.storyWalks {
		w.t++
		frac := float64(w.t) / float64(w.frames)
		if frac > 1 {
			frac = 1
		}
		u := &g.storyActors[w.actor]
		dx, dy := w.toX-w.fromX, w.toY-w.fromY
		adx, ady := absi(dx), absi(dy)
		total := adx + ady // 曼哈頓距離(格)
		if total == 0 {
			total = 1
		}
		dist := frac * float64(total) // 已走格數(含小數)
		var cx, cy float64            // 相對起點的位移(格)
		if adx >= ady {               // 先走長軸,再走短軸(格線行走)
			if dist <= float64(adx) {
				cx, cy = dist*sgn(dx), 0
				u.Dir = dirToward(0, 0, dx, 0)
			} else {
				cx, cy = float64(dx), (dist-float64(adx))*sgn(dy)
				u.Dir = dirToward(0, 0, 0, dy)
			}
		} else {
			if dist <= float64(ady) {
				cx, cy = 0, dist*sgn(dy)
				u.Dir = dirToward(0, 0, 0, dy)
			} else {
				cx, cy = (dist-float64(ady))*sgn(dx), float64(dy)
				u.Dir = dirToward(0, 0, dx, 0)
			}
		}
		u.X, u.Y = w.toX, w.toY // 掛在終點格,Off 為「當前位置-終點」(同 walkAnim 慣例)
		u.OffX = (float64(w.fromX) + cx - float64(w.toX)) * float64(g.m.TileW)
		u.OffY = (float64(w.fromY) + cy - float64(w.toY)) * float64(g.m.TileH)
		if w.scrollFollow {
			scrollDist := dist - float64(w.scrollFree)
			if scrollDist < 0 {
				scrollDist = 0
			}
			g.camY = w.fromCamY - scrollDist*float64(g.m.TileH)
			clamp(&g.camY, 0, float64(g.m.H*g.m.TileH-logicalH/storyZoom))
		}
		if w.t >= w.frames {
			u.OffX, u.OffY = 0, 0
			if w.finalDir >= 0 { // 走完面向目標(如 Ares 走到索爾旁面向他),不停在走位末段的短軸方向
				u.Dir = w.finalDir
			}
			if g.cutsceneLog { // FD2_CUTSCENE_LOG:印走位完成(誰、從哪到哪、末向),對原版走位比對
				fmt.Fprintf(os.Stderr, "[cutscene] walk done: %s (%d,%d)->(%d,%d) dir=%d\n",
					figName(u.Fig), w.fromX, w.fromY, w.toX, w.toY, u.Dir)
			}
			if w.then != nil {
				w.then()
			}
			continue // 不放回 live,即移除
		}
		live = append(live, w)
	}
	g.storyWalks = live
}

// stepFade 逐幀推進場景淡出/淡入(storyFade,doc46 §5.2):走完(不分 out/in)呼叫 then;
// g.fade 為空時內部直接返回(同 stepStoryWalks/stepActJob/stepCamPan 慣例)。
func (g *Game) stepFade() {
	if g.fade == nil {
		return
	}
	g.fade.t++
	if g.fade.t >= g.fade.total {
		cb := g.fade.then
		g.fade = nil
		if cb != nil {
			cb()
		}
	}
}

// stepCamPan 逐幀推進 beat「pan」鏡頭位移；原版模式逐 tile、X-first，
// 相容模式線性內插。走完清除 job 並接下一拍。
func (g *Game) stepCamPan() {
	j := g.camPan
	if j == nil {
		return
	}
	if j.tileStep {
		step := func(current, target float64, tile int) float64 {
			if current < target {
				current += float64(tile)
				if current > target {
					current = target
				}
			} else if current > target {
				current -= float64(tile)
				if current < target {
					current = target
				}
			}
			return current
		}
		if g.camX != j.toX {
			g.camX = step(g.camX, j.toX, g.m.TileW)
		} else if g.camY != j.toY {
			g.camY = step(g.camY, j.toY, g.m.TileH)
		}
		if g.camX == j.toX && g.camY == j.toY {
			g.camPan = nil
			if j.then != nil {
				j.then()
			}
		}
		return
	}
	j.t++
	frac := float64(j.t) / float64(j.frames)
	if frac > 1 {
		frac = 1
	}
	g.camX = j.fromX + (j.toX-j.fromX)*frac
	g.camY = j.fromY + (j.toY-j.fromY)*frac
	if j.t >= j.frames {
		g.camPan = nil
		if j.then != nil {
			j.then()
		}
	}
}

// stepFocusUnit 逐格重現 0x12cea 與 0x11b48/0x11b9b/0x11bfa/0x11c59。
// 原版視窗游標的安全帶為 X=2..10、Y=2..5；超出安全帶後，能捲圖時保持
// screen cursor 不動並移動 map origin，碰地圖邊界時才讓 screen cursor 靠邊。
func (g *Game) stepFocusUnit() {
	j := g.focusJob
	if j == nil || g.m == nil || g.m.TileW <= 0 || g.m.TileH <= 0 {
		return
	}
	finish := func() {
		g.focusJob = nil
		g.beatAdvance()
	}
	if g.curX == j.targetX && g.curY == j.targetY {
		finish()
		return
	}
	originX, originY := int(g.camX)/g.m.TileW, int(g.camY)/g.m.TileH
	maxOriginX, maxOriginY := g.m.W-13, g.m.H-8
	if maxOriginX < 0 {
		maxOriginX = 0
	}
	if maxOriginY < 0 {
		maxOriginY = 0
	}
	screenX, screenY := g.curX-originX, g.curY-originY
	switch {
	case g.curX > j.targetX:
		g.curX--
		if screenX < 2 && originX > 0 {
			originX--
		}
	case g.curX < j.targetX:
		g.curX++
		if screenX > 10 && originX < maxOriginX {
			originX++
		}
	case g.curY > j.targetY:
		g.curY--
		if screenY < 2 && originY > 0 {
			originY--
		}
	case g.curY < j.targetY:
		g.curY++
		if screenY > 5 && originY < maxOriginY {
			originY++
		}
	}
	g.camX, g.camY = float64(originX*g.m.TileW), float64(originY*g.m.TileH)
	if g.curX == j.targetX && g.curY == j.targetY {
		finish()
	}
}

// finishActJob 清除 acting job 後接下一個 beat。
func (g *Game) finishActJob(j *actPoseJob) {
	g.actJob = nil
	if j.then != nil {
		j.then()
	}
}

// actingActor resolves decoded acting against the original FDFIELD roster
// slot first.  Fig is deliberately only a legacy authored-scene fallback:
// using it for decoded frames would move the first matching guard when the
// original bytecode targeted a different same-Fig guard.
func (g *Game) actingActor(target campaign.ActingUnit) *battle.Unit {
	if target.Slot != nil {
		slot := *target.Slot
		if g.st != nil {
			if slot >= 0 && slot < len(g.st.Units) {
				return g.st.Units[slot]
			}
			return nil
		}
		if slot >= 0 && slot < len(g.storyActors) {
			return &g.storyActors[slot]
		}
		return nil
	}
	fig := target.Fig
	for i := range g.storyActors {
		if g.storyActors[i].Fig == fig {
			return &g.storyActors[i]
		}
	}
	return nil
}

func (g *Game) handlerUnitCount() int {
	if g.st != nil {
		return len(g.st.Units)
	}
	return len(g.storyActors)
}

func (g *Game) handlerUnitAt(slot int) *battle.Unit {
	if slot < 0 {
		return nil
	}
	if g.st != nil {
		if slot < len(g.st.Units) {
			return g.st.Units[slot]
		}
		return nil
	}
	if slot < len(g.storyActors) {
		return &g.storyActors[slot]
	}
	return nil
}

// beginActingFrame 設定目前 frame 的姿態。呼叫端保證 j.frame 有效。
func (g *Game) beginActingFrame(j *actPoseJob) {
	j.beat, j.tick = 0, 0
	for _, au := range j.acting[j.frame].Units {
		if u := g.actingActor(au); u != nil {
			u.Dir = au.Pose
		}
	}
}

func actingDelta(pose int) (int, int) {
	switch pose {
	case 0:
		return 0, 1
	case 1:
		return -1, 0
	case 2:
		return 0, -1
	case 3:
		return 1, 0
	default:
		return 0, 0
	}
}

func (g *Game) nextActingFrame(j *actPoseJob) {
	j.frame++
	if j.frame >= len(j.acting) {
		g.finishActJob(j)
		return
	}
	g.beginActingFrame(j)
}

// stepOriginalActing 精確承接已破解的 acting frame 行為(doc50 §1.2)。
func (g *Game) stepOriginalActing(j *actPoseJob) {
	f := j.acting[j.frame]
	if f.Special && f.Beats == 0 {
		// Original bit7=1/low7=0 is not an empty terminator. 0x136d9..
		// 0x137c5 performs a terrain/unit composite bracketed by delay(1) and
		// delay(2), then redraws. Ebiten redraws continuously, so retain its
		// three-tick hold before advancing to preserve handler timing.
		j.tick++
		if j.tick >= 3 {
			g.nextActingFrame(j)
		}
		return
	}
	if f.Beats <= 0 {
		g.nextActingFrame(j)
		return
	}
	if f.Special { // bit7=1：原地姿態，beat 為顯示節奏
		j.tick++
		if j.tick >= f.Beats {
			g.nextActingFrame(j)
		}
		return
	}

	// bit7=0：原版對每一格跑 tick=1..6，再在第 7 tick 寫入 X/Y。
	j.tick++
	frac := float64(j.tick) / 7
	for _, au := range f.Units {
		if u := g.actingActor(au); u != nil {
			dx, dy := actingDelta(au.Pose)
			u.OffX = float64(dx) * float64(g.m.TileW) * frac
			u.OffY = float64(dy) * float64(g.m.TileH) * frac
		}
	}
	if j.tick < 7 {
		return
	}
	for _, au := range f.Units {
		if u := g.actingActor(au); u != nil {
			dx, dy := actingDelta(au.Pose)
			u.X += dx
			u.Y += dy
			u.OffX, u.OffY = 0, 0
		}
	}
	j.tick = 0
	j.beat++
	if j.beat >= f.Beats {
		g.nextActingFrame(j)
	}
}

// stepActJob 逐幀推進原版 acting frame 或舊版姿態近似。
func (g *Game) stepActJob() {
	j := g.actJob
	if j == nil {
		return
	}
	if len(j.acting) > 0 {
		g.stepOriginalActing(j)
		return
	}
	u := &g.storyActors[j.actor]
	u.Dir = j.poses[j.idx]
	j.t++
	if j.t >= j.frames {
		j.t = 0
		j.idx++
		if j.idx >= len(j.poses) {
			g.finishActJob(j)
		}
	}
}

// ── BeatRunner(doc50):cutscene 節點的過場原語序列引擎 ──────────────
// beats 是平面序列,一次只有一拍在跑(pan/walk/dialog/act/fade/delay 皆為阻塞拍,
// 完成後呼叫 beatAdvance 進下一拍;spawn/join/bgm 為非阻塞拍,beatStart 內直接連呼
// beatAdvance)。全部跑完後比照 story 節點收尾:先走 ExitWalk(s)、再淡出、Advance、enterNode
// (advanceStoryNode 已實作,直接重用,不重造輪子)。

// beatAdvance 進下一拍;序列跑完則走節點收尾流程。
func (g *Game) beatAdvance() {
	g.beatIdx++
	if g.beatIdx >= len(g.beats) {
		g.followWalk = false // 走位跟焦只在 walk 拍內有效,收尾(ExitWalk/淡出)一律鏡頭固定
		if n := g.camp.Node(); n != nil {
			g.advanceStoryNode(n)
		}
		return
	}
	g.beatStart(g.beats[g.beatIdx])
}

// beatStart 依原語種類啟動目前這一拍(狀態掛到 g.camPan/g.storyWalks/g.dialog/g.actJob/
// g.fade/g.beatDelay,交給 Update 既有機制逐幀推進)。找不到對應角色 / 資料缺漏時直接跳拍
// 並記到 loadErr,不讓整個過場卡死(誠實 stub,勝過假裝完成)。
// figName cutscene sprite 標號用:fig id → 角色名(協助使用者回報 + 對映原版 slot)。
func figName(fig int) string {
	switch fig {
	case 0:
		return "索爾"
	case 4:
		return "亞雷斯"
	case 9:
		return "悠妮"
	case 30:
		return "蓋亞"
	case 48:
		return "國王"
	case 66:
		return "王后"
	case 68, 69:
		return "守衛"
	}
	return fmt.Sprintf("fig%d", fig)
}

func (g *Game) beatStart(b campaign.Beat) {
	if g.cutsceneLog { // FD2_CUTSCENE_LOG:印每一拍(op+參數),對原版 handler beat 序列比對
		fmt.Fprintf(os.Stderr, "[cutscene] beat op=%s source=%s fig=%d x=%d y=%d frames=%d line=%d count=%d script=%s scene=%s scene_index=%v loadch=%+v\n",
			b.Op, b.Source, b.Fig, b.X, b.Y, b.Frames, b.Line, b.Count, b.Script, b.Scene, b.SceneIndex, b.LoadCH)
	}
	if b.Op != "walk" && b.Op != "scroll_step" { // 兩種格線走位拍才啟用跟焦
		g.followWalk = false
	}
	findActor := func(fig int) int {
		for i := range g.storyActors {
			if g.storyActors[i].Fig == fig {
				return i
			}
		}
		return -1
	}
	switch b.Op {
	case "runtime_context":
		if b.RuntimeContext == nil || b.RuntimeContext.MinimumSlotCount() <= 0 {
			g.loadErr = "beat runtime_context:缺少有效 slot_count/slot_counts"
			return
		}
		if g.st == nil || !b.RuntimeContext.AcceptsSlotCount(len(g.st.Units)) {
			g.loadErr = fmt.Sprintf("beat runtime_context: runtime slots=%d, want exact %d or one of %v", g.handlerUnitCount(), b.RuntimeContext.SlotCount, b.RuntimeContext.SlotCounts)
			return
		}
		if b.RuntimeContext.StoryViewport {
			g.storyBG = true
		}
		g.beatAdvance()
	case "layout_units":
		if b.Layout == nil || len(b.Layout.Units) == 0 {
			g.loadErr = "beat layout_units:缺少可編輯的 runtime layout"
			return
		}
		for _, placement := range b.Layout.Units {
			unit := g.handlerUnitAt(placement.Slot)
			if unit == nil || placement.Pose < 0 || placement.Pose > 3 {
				g.loadErr = fmt.Sprintf("beat layout_units: slot%d/pose%d unavailable", placement.Slot, placement.Pose)
				return
			}
		}
		for _, placement := range b.Layout.Units {
			unit := g.handlerUnitAt(placement.Slot)
			unit.X, unit.Y, unit.Dir = placement.X, placement.Y, placement.Pose
			unit.OffX, unit.OffY = 0, 0
		}
		g.camX, g.camY = float64(b.Layout.CamX), float64(b.Layout.CamY)
		g.beatAdvance()
	case "if":
		matched, err := g.evalBeatCondition(b.Condition)
		if err != nil {
			g.loadErr = "beat if:" + err.Error()
			return
		}
		arm := b.Else
		if matched {
			arm = b.Then
		}
		g.spliceBeatsAfterCurrent(arm)
		g.beatAdvance()
	case "loadch":
		if b.LoadCH == nil {
			g.loadErr = "beat loadch:缺少完整狀態映射"
			return // compiler should prevent this; never turn it into a no-op.
		}
		if err := g.applyLoadCH(b.LoadCH); err != nil {
			g.loadErr = "beat loadch: " + err.Error()
			return // fail closed rather than continuing on the old map/roster.
		}
		g.beatAdvance()
	case "pan":
		frames := b.Frames
		if frames == 0 {
			frames = 30
		}
		g.camPan = &camPanJob{fromX: g.camX, fromY: g.camY, toX: float64(b.X), toY: float64(b.Y), frames: frames, tileStep: b.TileStep, then: g.beatAdvance}
	case "walk":
		idx := findActor(b.Fig)
		if idx < 0 {
			g.loadErr = fmt.Sprintf("beat walk:找不到 fig=%d", b.Fig)
			g.beatAdvance()
			return
		}
		u := &g.storyActors[idx]
		fromX, fromY := b.FromX, b.FromY
		if fromX == 0 && fromY == 0 { // 未指定起點:沿用角色目前座標(接續上一拍)
			fromX, fromY = u.X, u.Y
		}
		frames := b.Frames
		if frames == 0 {
			frames = 60
		}
		g.followWalk = b.Follow
		bdir := -1 // beat walk 面向:預設保留走位末向(如索爾往上走完仍面上);b.Dir 指定則走完面向它
		if b.Dir != nil {
			bdir = *b.Dir
		}
		g.storyWalks = append(g.storyWalks, &storyWalkJob{
			actor: idx, fromX: fromX, fromY: fromY,
			toX: b.X, toY: b.Y, frames: frames, finalDir: bdir, then: g.beatAdvance,
		})
	case "scroll_step":
		if b.Slot == nil || *b.Slot < 0 || *b.Slot >= len(g.storyActors) || b.Steps <= 0 {
			g.loadErr = fmt.Sprintf("beat scroll_step: runtime slot %v/steps=%d unavailable (materialized=%d)", b.Slot, b.Steps, len(g.storyActors))
			return
		}
		idx := *b.Slot
		u := &g.storyActors[idx]
		frames := b.Frames
		if frames <= 0 {
			frames = b.Steps * 7
		}
		// 0x13185 does not center on the actor. It lets the actor reach screen
		// row 1, then scrolls one tile (smoothly over the same seven ticks) per
		// further upward step. Preserve the current map-origin boundary.
		g.followWalk = false
		originY := int(g.camY) / g.m.TileH
		free := u.Y - originY - 1
		if free < 0 {
			free = 0
		}
		if originY == 0 {
			free = b.Steps
		}
		g.storyWalks = append(g.storyWalks, &storyWalkJob{
			actor: idx, fromX: u.X, fromY: u.Y,
			toX: u.X, toY: u.Y - b.Steps, frames: frames, finalDir: 2,
			scrollFollow: true, fromCamY: g.camY, scrollFree: free, then: g.beatAdvance,
		})
	case "dialog":
		lines := g.campLines
		if b.Script != "" {
			// A compiled handler carries an explicit editable-story context.  Do
			// not fall back to the enclosing Node's lines here: that could play a
			// valid index from the wrong FDTXT/loadch segment.
			lines = loadStoryScriptAt(handlerStoryPath(b.Script), b.Scene, b.SceneIndex)
			if lines == nil {
				g.loadErr = fmt.Sprintf("beat dialog:無法載入 script=%q scene=%q scene_index=%v", b.Script, b.Scene, b.SceneIndex)
				g.beatAdvance()
				return
			}
		}
		n := b.Count
		if n <= 0 {
			n = 1
		}
		end := b.Line + n
		if end > len(lines) {
			end = len(lines)
		}
		g.dialog = nil
		g.dlgPage = 0                                  // 新對白從第一頁起
		for i := end - 1; i >= b.Line && i >= 0; i-- { // 反序堆疊(同 enterNode story 分支慣例)
			ln := lines[i]
			dialogLine, err := g.resolveCampaignDialogLine(ln, b.Upper)
			if err != nil {
				g.dialog = nil
				g.loadErr = "beat dialog:" + err.Error()
				return
			}
			g.dialog = append(g.dialog, dialogLine)
		}
		if len(g.dialog) == 0 { // line/count 對不到資料:跳拍避免卡死
			g.loadErr = fmt.Sprintf("beat dialog:line=%d count=%d 對不到 script lines(len=%d)", b.Line, n, len(lines))
			g.beatAdvance()
		}
	case "act":
		if len(b.Acting) > 0 {
			// Decoded acting refers to the current materialized unit array. Never
			// turn an unavailable original slot into a silent no-op: the source may
			// be a different load-context resource or require an unmodelled spawn.
			for _, frame := range b.Acting {
				for _, target := range frame.Units {
					if target.Slot != nil && (*target.Slot < 0 || *target.Slot >= g.handlerUnitCount()) {
						g.loadErr = fmt.Sprintf("beat act %s: original runtime slot %d unavailable (materialized=%d)", b.Source, *target.Slot, g.handlerUnitCount())
						return
					}
				}
			}
			g.actJob = &actPoseJob{acting: b.Acting, then: g.beatAdvance}
			g.beginActingFrame(g.actJob)
			return
		}
		idx := findActor(b.Fig)
		if idx < 0 || len(b.Poses) == 0 {
			g.beatAdvance()
			return
		}
		frames := b.PoseFrames
		if frames == 0 {
			frames = 30
		}
		g.actJob = &actPoseJob{actor: idx, poses: b.Poses, frames: frames, then: g.beatAdvance}
	case "spawn":
		// Original 0x10b4e scans the FDFIELD records for this group and calls
		// the unit constructor once per match.  That constructor writes at the
		// current unit_count and increments it: this is append, not toggling an
		// already slot-stable full roster.  Keep the legacy activation fallback
		// for authored scene beats that have no LOADCH roster.
		if g.st != nil {
			if g.st.AppendGroup(b.Group) == 0 {
				g.loadErr = fmt.Sprintf("beat spawn %s: group %d unavailable in runtime roster", b.Source, b.Group)
				return
			}
		} else {
			g.materializeStoryGroup(b.Group)
		}
		g.beatAdvance()
	case "spawn_intro":
		g.materializeStoryGroup(b.Group)
		frames := b.Frames
		if frames <= 0 {
			frames = 12
		}
		g.beatDelay = frames
	case "deactivate_unit":
		if b.Slot == nil || g.handlerUnitAt(*b.Slot) == nil {
			g.loadErr = fmt.Sprintf("beat deactivate_unit: runtime slot %v unavailable (materialized=%d)", b.Slot, g.handlerUnitCount())
			return
		}
		g.handlerUnitAt(*b.Slot).OnField = false
		g.beatAdvance()
	case "reset_pose":
		if g.st != nil {
			for _, unit := range g.st.Units {
				if unit != nil {
					unit.Dir = 0
				}
			}
		} else {
			for i := range g.storyActors {
				g.storyActors[i].Dir = 0
			}
		}
		g.beatDelay = 1 // original 20ms at a 60Hz remake clock
	case "redraw":
		// Ebiten presents the current state once after this Update.  Blocking
		// one frame preserves the standalone original 0x11cac(0) boundary.
		g.beatDelay = 1
	case "focus_unit":
		if b.Slot == nil || *b.Slot < 0 || *b.Slot >= len(g.storyActors) {
			g.loadErr = fmt.Sprintf("beat focus_unit: runtime slot %v unavailable (materialized=%d)", b.Slot, len(g.storyActors))
			return
		}
		u := &g.storyActors[*b.Slot]
		g.focusJob = &focusUnitJob{targetX: u.X, targetY: u.Y}
	case "join":
		if !campaign.JoinableCharacterID(b.CharID) {
			g.loadErr = fmt.Sprintf("beat join:非法 player char_id=%d", b.CharID)
			return
		}
		if g.partyMembers == nil {
			g.partyMembers = make(map[int]bool)
		}
		if !g.partyMembers[b.CharID] {
			g.partyMembers[b.CharID] = true
			g.partyJoinOrder = append(g.partyJoinOrder, b.CharID)
		}
		g.beatAdvance()
	case "sync_party":
		if err := g.syncPartyFromBattle(); err != nil {
			g.loadErr = "beat sync_party: " + err.Error()
			return
		}
		g.beatAdvance()
	case "set_chapter":
		if b.Chapter == nil || *b.Chapter < 0 {
			g.loadErr = "beat set_chapter:缺少有效章節"
			return
		}
		g.handlerChapter = *b.Chapter
		g.beatAdvance()
	case "grant_item":
		if b.ItemID == nil || *b.ItemID < 0 || *b.ItemID > 0xff {
			g.loadErr = "beat grant_item:缺少有效 item_id"
			return
		}
		if g.st == nil {
			g.loadErr = "beat grant_item:缺少 runtime battle state"
			return
		}
		g.grantItemToParty(*b.ItemID)
		g.beatAdvance()
	case "bgm":
		g.playBGM(b.Track)
		g.beatAdvance()
	case "bgm_stop":
		g.stopBGM()
		g.beatAdvance()
	case "fade":
		frames := b.Frames
		if frames == 0 {
			frames = storyFadeFrames
		}
		g.fade = &storyFade{out: b.Out, total: frames, then: g.beatAdvance}
	case "delay":
		frames := b.Frames
		if frames == 0 && b.Ms > 0 {
			frames = b.Ms * 60 / 1000
		}
		if frames <= 0 {
			frames = 1
		}
		g.beatDelay = frames
	default:
		g.loadErr = "beat:未知原語 " + b.Op
		g.beatAdvance()
	}
}

func (g *Game) resolveCampaignDialogLine(line campaign.Line, upperOverride *bool) (battle.DialogLine, error) {
	speaker := line.Speaker
	if line.SpeakerSlot != nil {
		slot := *line.SpeakerSlot
		var unit *battle.Unit
		if g.st != nil {
			if slot < 0 || slot >= len(g.st.Units) || g.st.Units[slot] == nil {
				return battle.DialogLine{}, fmt.Errorf("speaker_slot %d unavailable (battle units=%d)", slot, len(g.st.Units))
			}
			unit = g.st.Units[slot]
		} else {
			if slot < 0 || slot >= len(g.storyActors) {
				return battle.DialogLine{}, fmt.Errorf("speaker_slot %d unavailable (cutscene units=%d)", slot, len(g.storyActors))
			}
			unit = &g.storyActors[slot]
		}
		speaker = unit.Portrait
	}
	upper := line.Upper
	if upperOverride != nil {
		upper = upperOverride
	}
	return battle.DialogLine{Speaker: speaker, Text: line.Text, Upper: upper}, nil
}

func (g *Game) evalBeatCondition(condition *campaign.BeatCondition) (bool, error) {
	if condition == nil || condition.Op != "any_unit_inactive" || len(condition.UnitSlots) == 0 {
		return false, fmt.Errorf("缺少有效 any_unit_inactive condition")
	}
	if g.st == nil {
		return false, fmt.Errorf("any_unit_inactive 缺少 runtime battle state")
	}
	for _, slot := range condition.UnitSlots {
		if slot < 0 || slot >= len(g.st.Units) || g.st.Units[slot] == nil {
			return false, fmt.Errorf("any_unit_inactive slot %d unavailable (units=%d)", slot, len(g.st.Units))
		}
	}
	for _, slot := range condition.UnitSlots {
		unit := g.st.Units[slot]
		if !unit.OnField || !unit.Alive() {
			return true, nil
		}
	}
	return false, nil
}

// spliceBeatsAfterCurrent chooses one structured branch without mutating the
// campaign node's backing array. The common continuation remains exactly once
// after the selected arm.
func (g *Game) spliceBeatsAfterCurrent(arm []campaign.Beat) {
	prefix := g.beats[:g.beatIdx+1]
	tail := g.beats[g.beatIdx+1:]
	selected := make([]campaign.Beat, 0, len(prefix)+len(arm)+len(tail))
	selected = append(selected, prefix...)
	selected = append(selected, arm...)
	selected = append(selected, tail...)
	g.beats = selected
}

func (g *Game) materializeStoryGroup(group int) {
	if g.storyRoster != nil {
		if g.storySpawned[group] {
			return
		}
		for _, actor := range g.storyRoster {
			if actor.Group == group {
				actor.OnField = true
				g.storyActors = append(g.storyActors, actor)
			}
		}
		g.storySpawned[group] = true
		return
	}
	for i := range g.storyActors {
		if g.storyActors[i].Group == group {
			g.storyActors[i].OnField = true
		}
	}
}

// applyLoadCH is the remake adapter for original 0x205da/0x1088d.  The
// original operation selects FDTXT chapter+1 and the three FDFIELD resources
// for the same chapter in one call; it is not merely a camera/map command.
// The binding therefore provides all three editable counterparts.  Validate
// roster and text before replacing the rendered map so a malformed asset does
// not leave a half-applied chapter transition behind.
func (g *Game) applyLoadCH(state *campaign.LoadCHState) error {
	if state == nil || state.Chapter < 0 || state.Map == "" || state.Roster == "" || state.SlotCount <= 0 || state.Script == "" {
		return fmt.Errorf("incomplete map/roster/story state")
	}
	roster, err := battle.Load(assetPath(state.Roster))
	if err != nil {
		return fmt.Errorf("roster %q: %w", state.Roster, err)
	}
	if len(roster.Units) != state.SlotCount {
		return fmt.Errorf("roster %q has %d slots, binding declares %d", state.Roster, len(roster.Units), state.SlotCount)
	}
	lines := loadStoryScriptAt(state.Script, "", nil)
	if lines == nil {
		return fmt.Errorf("story script %q", state.Script)
	}
	if err := g.loadMap(state.Map); err != nil {
		return fmt.Errorf("map %q: %w", state.Map, err)
	}
	var party []*battle.Unit
	if state.PartyScenario != "" {
		scenario, err := battle.LoadScenario(assetPath(state.PartyScenario))
		if err != nil {
			return fmt.Errorf("party scenario %q: %w", state.PartyScenario, err)
		}
		// A normal campaign reaches this LOADCH after JOIN established permanent
		// membership. Direct scene/debug starts have no membership history and
		// use the evidence-backed PartyOrder stored in the editable binding.
		filterScenarioParty(scenario, g.partyMembers)
		partyOrder := g.partyJoinOrder
		if len(partyOrder) == 0 {
			partyOrder = state.PartyOrder
		} else if len(state.PartyOrder) != 0 && !equalIntOrder(partyOrder, state.PartyOrder) {
			return fmt.Errorf("party JOIN chronology %v differs from binding %v", partyOrder, state.PartyOrder)
		}
		if err := reorderScenarioParty(scenario, partyOrder); err != nil {
			return fmt.Errorf("party scenario %q: %w", state.PartyScenario, err)
		}
		party = scenario.PartyUnits(roster.OwnDeploy)
		if len(g.partyRoster) != 0 {
			g.applyPersistentParty(&battle.State{Units: party})
		}
	}

	// A handler cutscene uses the loaded FDFIELD records as a source, not as a
	// pre-built unit array. Original 0x10b4e appends matching group records to
	// the current array. Persistent party members are constructed first when a
	// binding supplies PartyScenario; LOADCH then materializes group 0 and later
	// SPAWN calls append groups in FDFIELD order. This preserves the actual
	// runtime slot identity for every evidence-backed load/spawn sequence.
	g.st, g.sel, g.sc = nil, nil, nil
	g.storyActors = make([]battle.Unit, 0, len(party)+len(roster.Units))
	g.storyRoster = make([]battle.Unit, 0, len(roster.Units))
	g.storySpawned = make(map[int]bool)
	for _, unit := range party {
		g.storyActors = append(g.storyActors, *unit)
	}
	for _, u := range roster.Units {
		if u == nil {
			// State.Load does not create holes. Keep a harmless roster placeholder
			// if a future loader does, without inventing a live unit.
			g.storyRoster = append(g.storyRoster, battle.Unit{Group: 255})
			continue
		}
		actor := *u
		actor.OffX, actor.OffY = 0, 0
		actor.OnField = false
		g.storyRoster = append(g.storyRoster, actor)
		if actor.Group == 0 {
			actor.OnField = true
			g.storyActors = append(g.storyActors, actor)
		}
	}
	g.storySpawned[0] = true
	g.storyWalks = nil
	g.storyBG = true
	g.walkFirst, g.followWalk = false, false
	g.camMaxY = float64(state.CamMaxY)
	g.camX, g.camY = float64(state.CamX), float64(state.CamY)
	g.campLines = lines
	return nil
}

func equalIntOrder(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// dlgWrap 把一句對白依框寬換行成顯示列(繪製與 Enter 分頁共用同一套,確保頁數一致)。
// 換行寬度與繪製碼一致:下框到框右緣;上框(說話者 id>=32,頭像在右)止於頭像左緣前。
func dlgWrap(dl battle.DialogLine) []string {
	const ps = 2.1
	upper := dl.Speaker >= 32
	bx, tx := 10.0, 216.0
	rightEdge := bx + 620 - 16
	if upper {
		tx = 32
		rightEdge = (float64(logicalW) - 16 - 80*ps) - 8 // 頭像左緣前 8px
	}
	perLine := int((rightEdge - tx) / (fontSize * 1.7))
	if perLine < 1 {
		perLine = 1
	}
	txt := []rune("『" + toFullWidth(dl.Text) + "』")
	var lines []string
	for len(txt) > 0 {
		nn := perLine
		if nn > len(txt) {
			nn = len(txt)
		}
		lines = append(lines, string(txt[:nn]))
		txt = txt[nn:]
	}
	return lines
}

// dlgPageCount 該句對白的總頁數(每頁最多 3 行)。
func dlgPageCount(dl battle.DialogLine) int {
	n := (len(dlgWrap(dl)) + 2) / 3
	if n < 1 {
		n = 1
	}
	return n
}

// dlgAdvance 處理 Enter:目前句還有下一頁就翻頁(回 false,不換句);翻完(或本就單頁)就 pop
// 到下一句、頁碼歸零(回 true=已換句)。修長對白被截斷丟棄的 bug(使用者回饋 2026-07-05)。
func (g *Game) dlgAdvance() bool {
	if len(g.dialog) > 0 && g.dlgPage+1 < dlgPageCount(g.dialog[len(g.dialog)-1]) {
		g.dlgPage++
		return false
	}
	if len(g.dialog) > 0 {
		g.dialog = g.dialog[:len(g.dialog)-1]
	}
	g.dlgPage = 0
	return true
}

// ── 對話框切換動畫(使用者回饋 2026-07-04 #3:換人說話時框先收合再展開)──
const dlgNone = -999    // dlgShown 無框哨兵
const dlgAnimFrames = 7 // 縮/展各自幀數(~0.12s@60fps)

// stepDlgAnim 逐幀推進對話框縮/展動畫:首句直接展開;說話者變更先縮(phase1)再換人展開(phase2)。
func (g *Game) stepDlgAnim() {
	show := len(g.dialog) > 0 && !(g.storyBG && g.walkFirst && len(g.storyWalks) > 0)
	if !show {
		g.dlgShown, g.dlgPhase, g.dlgT = dlgNone, 0, 0
		return
	}
	cur := g.dialog[len(g.dialog)-1].Speaker
	switch g.dlgPhase {
	case 0:
		if g.dlgShown == dlgNone { // 首句:直接展開
			g.dlgShown, g.dlgPhase, g.dlgT = cur, 2, 0
			g.dlgUpper = g.dialog[len(g.dialog)-1].Upper
		} else if cur != g.dlgShown { // 換人:先收合
			g.dlgPhase, g.dlgT = 1, 0
		}
	case 1:
		g.dlgT++
		if g.dlgT >= dlgAnimFrames {
			g.dlgShown, g.dlgPhase, g.dlgT = cur, 2, 0
			g.dlgUpper = g.dialog[len(g.dialog)-1].Upper
		}
	case 2:
		g.dlgT++
		if g.dlgT >= dlgAnimFrames {
			g.dlgPhase, g.dlgT = 0, 0
		}
	}
}

// terrainAt 回傳某格的地形索引(戰鬥背景用;越界回 -1)。
func (g *Game) terrainAt(x, y int) int {
	if g.m == nil || x < 0 || y < 0 || x >= g.m.W || y >= g.m.H {
		return -1
	}
	return g.m.Tiles[y*g.m.W+x]
}

// ── campaign(劇本節點圖,doc 19)引擎接線 ──────────────────────────

// enterNode 進入 camp 目前節點:story→掛對白、battle→重開戰場、inventory_gate→依角色物品欄自動分支、
// event→套旗標直通、town/preparation/choice/ending→等輸入。
func (g *Game) enterNode() {
	if g.camp == nil {
		return
	}
	n := g.camp.Node()
	if n == nil {
		return // 流程結束(game over)
	}
	g.playBGM(n.BGM)
	g.storyBG = false // 預設離開場景背景模式;story+Map 節點下面再開回
	g.storyWalks = nil
	g.storyAutoAdvance = 0
	g.walkFirst, g.followWalk, g.camMaxY = false, false, 0
	g.camPan, g.focusJob, g.actJob, g.beats, g.beatIdx, g.beatDelay = nil, nil, nil, nil, -1, 0
	g.battleEvent, g.battleEventDelay = nil, 0
	g.dlgShown, g.dlgPhase, g.dlgT = dlgNone, 0, 0
	g.dlgUpper = nil
	switch n.Type {
	case "story", "cutscene": // cutscene(doc50):同一套場景設置,進行中改由 Beats 驅動(見下)
		g.dialog = nil
		g.dlgPage = 0 // 新對白從第一頁起
		lines := n.Lines
		if n.Script != "" { // 本機劇情文本檔(assets/story/chNN.json,人工精校;無檔 fallback 內嵌 lines)
			if ls := loadStoryScript(n.Script, n.Scene); len(ls) > 0 {
				lines = ls
			}
		}
		g.campLines = lines // cutscene dialog beat 依 Line/Count 取子段;story 節點也存一份備用
		if n.Type == "story" {
			for i := len(lines) - 1; i >= 0; i-- { // 反序堆疊:顯示取末端,Enter 逐句 pop
				g.dialog = append(g.dialog, battle.DialogLine{Speaker: lines[i].Speaker, Text: lines[i].Text})
			}
		}
		if n.Map != "" { // 場景背景圖(doc23 §4:序幕王城/草地= FDFIELD map32 複合場景,非戰場地圖疊對白)
			if err := g.loadMap(n.Map); err != nil {
				g.loadErr = "map: " + err.Error()
			}
			g.st, g.sel = nil, nil // 清殘留單位/選取(避免上一戰場畫面疊在新背景上)
			g.storyBG = true
			g.walkFirst, g.followWalk = n.WalkFirst, n.FollowWalk
			g.camMaxY = float64(n.CamMaxY)
			g.camX, g.camY = float64(n.CamX), float64(n.CamY)
			g.storyActors = nil
			g.storyRoster, g.storySpawned = nil, nil
			for _, a := range n.Actors { // cutscene 靜態擺位(國王/王后/主角等),無 AI/戰鬥邏輯
				u := battle.Unit{Fig: a.Fig, X: a.X, Y: a.Y, Dir: a.Dir, OnField: true}
				g.storyActors = append(g.storyActors, u)
				if a.WalkFrames > 0 && (a.FromX != a.X || a.FromY != a.Y) { // 進場走位(doc46 §5.3)
					idx := len(g.storyActors) - 1
					g.storyWalks = append(g.storyWalks, &storyWalkJob{
						actor: idx, fromX: a.FromX, fromY: a.FromY,
						toX: a.X, toY: a.Y, frames: a.WalkFrames,
						finalDir: a.Dir, // 進場走完面向 actor 宣告的 dir(如 Ares 面向索爾)
					})
				}
			}
		} else {
			g.storyActors = nil
			g.storyRoster, g.storySpawned = nil, nil
		}
		if g.cutsceneLog { // FD2_CUTSCENE_LOG:進場印節點 + 每個 actor(idx/名/座標/dir)
			fmt.Fprintf(os.Stderr, "[cutscene] === node %q map=%s cam=(%d,%d) ===\n", g.camp.Cur, n.Map, n.CamX, n.CamY)
			for i, a := range g.storyActors {
				fmt.Fprintf(os.Stderr, "[cutscene]   actor[%d] %s (%d,%d) dir=%d\n", i, figName(a.Fig), a.X, a.Y, a.Dir)
			}
		}
		if n.Type == "cutscene" {
			g.beats = n.Beats
			if n.HandlerBinding != "" {
				beats, issues, err := campaign.CompileHandlerBinding(assetPath(n.HandlerBinding))
				if err != nil || len(issues) > 0 {
					g.loadErr = fmt.Sprintf("handler binding %q unresolved: %v issues=%d", n.HandlerBinding, err, len(issues))
					return // fail closed: never replace authored beats with a partial handler
				}
				g.beats = beats
			}
			g.beatAdvance() // beatIdx -1 → 0,啟動第一拍(doc50 BeatRunner)
		} else if len(lines) == 0 && n.AutoAdvance > 0 { // 無對白節點(行軍蒙太奇):進場後自動倒數轉場
			g.storyAutoAdvance = n.AutoAdvance
		}
	case "battle":
		if n.Map != "" { // 指定戰場(assets/maps/mapN;全 33 圖已匯出)
			if err := g.loadMap(n.Map); err != nil {
				g.loadErr = "map: " + err.Error()
			}
		}
		g.resetBattle(n.Units, n.Scenario)
	case "inventory_gate":
		if n.ItemID == nil { // Load 已拒絕；保留 runtime fail-closed 防線給手工測試 Campaign。
			g.loadErr = "inventory_gate: missing item_id"
			return
		}
		outcome := "missing"
		if g.partyHasItemID(*n.ItemID) {
			outcome = "present"
		}
		g.camp.Advance(outcome)
		g.enterNode()
	case "inventory_recipe":
		crafted, err := g.applyInventoryRecipe(n)
		if err != nil {
			g.loadErr = "inventory_recipe: " + err.Error()
			return
		}
		outcome := "insufficient"
		if crafted {
			outcome = "crafted"
		}
		g.camp.Advance(outcome)
		g.enterNode()
	case "event":
		g.camp.Advance("")
		g.enterNode()
	case "choice", "town":
		g.dialog, g.st, g.sel = nil, nil, nil // 戰間 hub 不可殘留上一戰的單位或勝利對白
		g.campSel = 0
	case "preparation", "church":
		g.dialog, g.st, g.sel = nil, nil, nil
		// 節點邊界 UI；preparation 可在此安全 F5 存檔，Enter 才進下一章 pre handler。
		if n.Type == "preparation" {
			g.setupPreparation(n)
		} else {
			g.setupChurch()
		}
	case "shop":
		g.dialog, g.st, g.sel = nil, nil, nil
		g.shopSel = 0
		g.shopPicking = false
		g.shopEquipPrompt = false
		g.shopMode = "buy"
		g.shopSellPicking = false
		g.shopSellUnitSel = 0
		g.shopSellSlotSel = 0
		g.shopRecipientSel = 0
		g.shopRecipients = nil
	case "ending":
		g.dialog, g.st, g.sel = nil, nil, nil
	}
}

// resetBattle 重開一場戰鬥(campaign battle 節點;敗北重試也走這裡)。
func (g *Game) resetBattle(unitsPath, scnPath string) {
	g.storyActors, g.storyRoster, g.storySpawned = nil, nil, nil // 不讓上一個 pre cutscene 的 scene units 疊進戰場
	if unitsPath == "" {
		unitsPath = "assets/map0_units.json"
	}
	if st, err := battle.Load(assetPath(unitsPath)); err == nil {
		g.st = st
	}
	g.result, g.sel, g.reach, g.moved = "", nil, nil, false
	g.atk, g.walk, g.dialog, g.msg = nil, nil, nil, ""
	g.sc = nil // scenario 空 = 無劇本(FDFIELD roster 全員照 units.json 登場;不 fallback ch01——
	// ch01 的 initial_groups/party/deploy 是 map0 專屬,錯配到他章會讓單位消失。每章 stub 見 worklist)
	if g.st != nil && scnPath != "" {
		if sc, err := battle.LoadScenario(assetPath(scnPath)); err == nil {
			// Scenario owns chapter-specific combat statistics, but its party list
			// is filtered by the permanent membership established by JOIN.  A
			// direct chapter/debug start has no JOIN history and therefore keeps
			// the authored scenario party intact.
			filterScenarioParty(sc, g.battlePartyMembers())
			if err := reorderScenarioParty(sc, g.partyJoinOrder); err != nil {
				g.loadErr = "scenario party order: " + err.Error()
				return
			}
			g.sc = sc
			g.dialog = append(g.dialog, sc.Setup(g.st)...)
			g.initializeEquipmentBases(g.st)
			g.applyScenarioPartyJoins()
			g.applyPersistentParty(g.st)
			g.focusOnParty()
		}
	}
}

// applyPersistentParty overlays the post-battle roster snapshot on freshly
// materialized player units while preserving this scenario's deployment,
// camp/group and on-field state. Original 0x11506 matches on charID; remake
// player Fig is the same stable 0..31 identity used by JOIN.
func (g *Game) applyPersistentParty(st *battle.State) {
	if st == nil || len(g.partyRoster) == 0 {
		return
	}
	for _, dst := range st.Units {
		if dst == nil || dst.Camp != battle.Own {
			continue
		}
		if src, ok := g.partyRoster[dst.Fig]; ok {
			applyPersistentStats(dst, &src)
		}
	}
}

func (g *Game) initializeEquipmentBases(st *battle.State) {
	if st == nil || g.shopItemStats == nil {
		return
	}
	for _, u := range st.Units {
		if u != nil && u.Camp == battle.Own {
			campaign.InitializeEquipmentBase(u, g.shopItemStats)
		}
	}
}

func applyPersistentStats(dst, src *battle.Unit) {
	if dst == nil || src == nil {
		return
	}
	dst.Name, dst.ClsName, dst.ClassID, dst.Lv = src.Name, src.ClsName, src.ClassID, src.Lv
	dst.HP, dst.MaxHP, dst.MP, dst.MaxMP = src.HP, src.MaxHP, src.MP, src.MaxMP
	dst.AP, dst.DP, dst.DX = src.AP, src.DP, src.DX
	dst.HIT, dst.EV, dst.CritPct, dst.MV = src.HIT, src.EV, src.CritPct, src.MV
	dst.AtkMin, dst.AtkMax = src.AtkMin, src.AtkMax
	dst.BaseAP, dst.BaseDP, dst.BaseHIT, dst.BaseEV, dst.BaseMV = src.BaseAP, src.BaseDP, src.BaseHIT, src.BaseEV, src.BaseMV
	dst.BaseAtkMin, dst.BaseAtkMax, dst.EquipmentBaseSet = src.BaseAtkMin, src.BaseAtkMax, src.EquipmentBaseSet
	dst.Portrait, dst.Fig = src.Portrait, src.Fig
	dst.Exp, dst.ExpPerLevel = src.Exp, src.ExpPerLevel
	dst.Spells = append(dst.Spells[:0], src.Spells...)
	dst.Inventory = append(dst.Inventory[:0], src.Inventory...)
	dst.Equipped = append(dst.Equipped[:0], src.Equipped...)
	dst.InventorySlots = append(dst.InventorySlots[:0], src.InventorySlots...)
}

// grantItemToParty projects original 0x1c220 + 0x1bb8c: scan runtime units in
// slot order, skip non-player camps and append an unequipped item to the first
// player inventory with room. If every player has eight items, the original
// silently drops the reward.
func (g *Game) grantItemToParty(itemID int) bool {
	if g.st == nil {
		return false
	}
	for _, unit := range g.st.Units {
		if unit == nil || unit.Camp != battle.Own || len(unit.Inventory) >= 8 {
			continue
		}
		if unit.AddInventoryItem(itemID, false) {
			return true
		}
	}
	return false
}

// partyHasItemID implements the campaign-level equivalent of original 0x24b14:
// search runtime slots 0..15 without filtering camp/activity for an exact
// unsigned-byte item identity. The persistent roster fallback also makes a
// node-boundary save/load lossless when there is no active runtime array.
func (g *Game) partyHasItemID(itemID int) bool {
	if g.st != nil {
		limit := len(g.st.Units)
		if limit > 16 {
			limit = 16
		}
		for _, unit := range g.st.Units[:limit] {
			if unit == nil {
				continue
			}
			for _, held := range unit.Inventory {
				if held == itemID {
					return true
				}
			}
		}
	}
	for _, unit := range g.partyRoster {
		for _, held := range unit.Inventory {
			if held == itemID {
				return true
			}
		}
	}
	return false
}

// applyInventoryRecipe projects the original ch20_post nested loops rather
// than normalising them into a friendlier "one of each" recipe. Each
// (item_id,runtime_slot) pair contributes at most one match; crafting happens
// only when the total equals RequiredMatches exactly. On success the first
// matching copy for every pair is removed in original item-then-slot order,
// then RewardItemID is granted by the normal 0x1c220 projection.
func (g *Game) applyInventoryRecipe(n *campaign.Node) (bool, error) {
	if n == nil || n.RewardItemID == nil || len(n.ItemIDs) == 0 || n.SlotCount <= 0 || n.RequiredMatches <= 0 {
		return false, fmt.Errorf("invalid recipe data")
	}
	if g.st == nil || len(g.st.Units) < n.SlotCount {
		return false, fmt.Errorf("runtime slots=%d, want at least %d", g.handlerUnitCount(), n.SlotCount)
	}
	find := func(unit *battle.Unit, itemID int) int {
		if unit == nil {
			return -1
		}
		for i, held := range unit.Inventory {
			if held == itemID {
				return i
			}
		}
		return -1
	}
	matches := 0
	for _, itemID := range n.ItemIDs {
		for slot := 0; slot < n.SlotCount; slot++ {
			if find(g.st.Units[slot], itemID) >= 0 {
				matches++
			}
		}
	}
	if matches != n.RequiredMatches {
		return false, nil
	}
	for _, itemID := range n.ItemIDs {
		for slot := 0; slot < n.SlotCount; slot++ {
			unit := g.st.Units[slot]
			if idx := find(unit, itemID); idx >= 0 {
				unit.RemoveInventoryIndex(idx)
			}
		}
	}
	g.grantItemToParty(*n.RewardItemID) // original silently drops the reward only if every player inventory is full
	return true, nil
}

// syncPartyFromBattle is the remake projection of original 0x11506. The EXE
// copies a matching 0x50-byte battle unit back to the persistent roster,
// clears transient state/path bytes, restores active survivors to full HP and
// restores everyone's MP. Defeated/inactive members retain their zero HP. The
// EXE skips an inactive charID 0 record; the remake snapshots JOIN member 0 as
// well because a successful post-battle handler cannot normally receive a
// defeated protagonist and dropping his progression would be destructive.
func (g *Game) syncPartyFromBattle() error {
	if g.st == nil {
		return fmt.Errorf("no completed battle state")
	}
	if g.partyRoster == nil {
		g.partyRoster = make(map[int]battle.Unit)
	}
	for _, current := range g.st.Units {
		if current == nil {
			continue
		}
		id := current.Fig
		if len(g.partyMembers) != 0 {
			if !g.partyMembers[id] {
				continue
			}
		} else if current.Camp != battle.Own {
			continue
		}
		snapshot := *current
		snapshot.Spells = append([]int(nil), current.Spells...)
		snapshot.Inventory = append([]int(nil), current.Inventory...)
		snapshot.Equipped = append([]bool(nil), current.Equipped...)
		snapshot.InventorySlots = append([]int(nil), current.InventorySlots...)
		if snapshot.MaxMP < snapshot.MP {
			snapshot.MaxMP = snapshot.MP
		}
		if snapshot.OnField && snapshot.Alive() {
			snapshot.HP = snapshot.MaxHP
		}
		snapshot.MP = snapshot.MaxMP
		snapshot.Acted = false
		snapshot.OffX, snapshot.OffY = 0, 0
		snapshot.BuffAPPct, snapshot.BuffDPPct = 0, 0
		snapshot.BuffHit, snapshot.BuffEV, snapshot.BuffTurns = 0, 0, 0
		snapshot.Sealed, snapshot.SealTurns = false, 0
		snapshot.Poisoned, snapshot.PoisonTurns = false, 0
		snapshot.Paralyzed, snapshot.ParalyzeTurns = false, 0
		g.partyRoster[id] = snapshot
	}
	return nil
}

func (g *Game) applyScenarioPartyJoins() {
	if g.sc == nil {
		return
	}
	for _, id := range g.sc.TakePartyJoins() {
		if !campaign.JoinableCharacterID(id) {
			g.loadErr = fmt.Sprintf("scenario join_party:非法 player char_id=%d", id)
			continue
		}
		if g.partyMembers == nil {
			g.partyMembers = make(map[int]bool)
		}
		if !g.partyMembers[id] {
			g.partyMembers[id] = true
			g.partyJoinOrder = append(g.partyJoinOrder, id)
		}
	}
}

// filterScenarioParty applies the campaign's JOIN membership to a freshly
// loaded battle scenario.  A nil/empty membership intentionally means a
// direct chapter/debug start, so the authored scenario party remains usable.
func filterScenarioParty(sc *battle.Scenario, members map[int]bool) {
	if sc == nil || len(members) == 0 {
		return
	}
	party := sc.Party[:0]
	var deploy [][2]int
	if len(sc.DeployCells) != 0 {
		deploy = sc.DeployCells[:0]
	}
	for i, member := range sc.Party {
		if members[member.Fig] {
			party = append(party, member)
			if i < len(sc.DeployCells) {
				deploy = append(deploy, sc.DeployCells[i])
			}
		}
	}
	sc.Party = party
	if len(sc.DeployCells) != 0 {
		sc.DeployCells = deploy
	}
}

// battlePartyMembers returns the temporary roster selected by the original
// preparation screen, falling back to the permanent JOIN roster for direct
// starts and campaigns that have not reached a preparation node yet.
func (g *Game) battlePartyMembers() map[int]bool {
	if len(g.partyDeploy) != 0 {
		return g.partyDeploy
	}
	return g.partyMembers
}

func (g *Game) setupPreparation(n *campaign.Node) {
	g.prepIDs = append(g.prepIDs[:0], g.partyJoinOrder...)
	seen := make(map[int]bool, len(g.prepIDs))
	for _, id := range g.prepIDs {
		seen[id] = true
	}
	for id := range g.partyRoster {
		if !seen[id] {
			g.prepIDs = append(g.prepIDs, id)
			seen[id] = true
		}
	}
	g.prepSel = 0
	g.prepLimit = 15
	if n != nil && n.PartyLimit > 0 {
		g.prepLimit = n.PartyLimit
	}
	// Direct EXE evidence: 0x318ad uses 0x13 instead of 0x0f after the
	// late-game chapter threshold. Keep this fallback editable via PartyLimit.
	if g.prepLimit == 15 && g.camp != nil {
		if strings.HasSuffix(g.camp.Cur, "28") || strings.HasSuffix(g.camp.Cur, "29") || strings.HasSuffix(g.camp.Cur, "30") {
			g.prepLimit = 19
		}
	}
	if len(g.partyDeploy) == 0 {
		g.partyDeploy = make(map[int]bool)
		for i, id := range g.prepIDs {
			if i >= g.prepLimit {
				break
			}
			g.partyDeploy[id] = true
		}
	}
}

func (g *Game) preparationSelected() int {
	n := 0
	for _, selected := range g.partyDeploy {
		if selected {
			n++
		}
	}
	return n
}

func (g *Game) setupChurch() {
	g.churchSel = 0
	g.churchMode = "menu"
	g.churchIDs = nil
	g.churchClassID = -1
	g.churchBranches = nil
}

func (g *Game) churchCandidates(mode string) []int {
	if mode == "revive" {
		ids := make([]int, 0)
		for _, id := range g.partyJoinOrder {
			if u, ok := g.partyRoster[id]; ok && campaign.CanRevive(&u) {
				ids = append(ids, id)
			}
		}
		return ids
	}
	return campaign.ClassChangeCandidates(g.partyRoster, g.partyJoinOrder)
}

// reorderScenarioParty applies the original JOIN chronology before either a
// battle or handler cutscene constructs its runtime unit array. Deployment
// cells stay attached to their characters; only slot construction changes.
// Chapter 0 proves the order 0,9,4,30 rather than the authored battle-UI order
// 0,4,9,30, and later acting/post handlers address those construction slots.
func reorderScenarioParty(sc *battle.Scenario, joinOrder []int) error {
	if sc == nil || len(joinOrder) == 0 {
		return nil
	}
	if len(sc.DeployCells) != 0 && len(sc.DeployCells) < len(sc.Party) {
		return fmt.Errorf("JOIN reordering requires complete deploy cells, got %d for %d party members", len(sc.DeployCells), len(sc.Party))
	}
	type partyEntry struct {
		member battle.PartyMember
		cell   [2]int
	}
	byID := make(map[int]partyEntry, len(sc.Party))
	for i, member := range sc.Party {
		entry := partyEntry{member: member}
		if i < len(sc.DeployCells) {
			entry.cell = sc.DeployCells[i]
		}
		byID[member.Fig] = entry
	}
	ordered := make([]battle.PartyMember, 0, len(sc.Party))
	orderedCells := make([][2]int, 0, len(sc.DeployCells))
	for _, id := range joinOrder {
		if entry, ok := byID[id]; ok {
			ordered = append(ordered, entry.member)
			if len(sc.DeployCells) != 0 {
				orderedCells = append(orderedCells, entry.cell)
			}
			delete(byID, id)
		}
	}
	if len(ordered) != len(sc.Party) {
		return fmt.Errorf("JOIN order covers %d of %d scenario party members", len(ordered), len(sc.Party))
	}
	sc.Party = ordered
	sc.DeployCells = orderedCells
	return nil
}

// focusOnParty 開局/戰鬥重開後把游標(=鏡頭中心)移到我方主角隊部署格的重心。
// 原鏡頭預設停在 (0,0),不對準的話玩家開局完全看不到主角隊(playfix #3)。
// 主角隊為直接定位(doc 25 §7.5.1,無進場動畫,見 event.go spawn_party 註解),此函式純粹是
// 「鏡頭對準部隊」的合理預設,不是重現原版鏡頭運鏡(原版 0x3231b 用 0x13185/0x32999 對特定群組做
// 攝影機平移 reveal,是鏡頭動不是單位動,且未對主角隊做;dosbox 複驗全序章無任何單位行走動畫)。
func (g *Game) focusOnParty() {
	if g.st == nil {
		return
	}
	n, sx, sy := 0, 0, 0
	for _, u := range g.st.Units {
		if u.Camp == battle.Own && u.OnField {
			sx += u.X
			sy += u.Y
			n++
		}
	}
	if n > 0 {
		g.curX, g.curY = sx/n, sy/n
	}
}

func (g *Game) shopReceiverIDs(good campaign.Good) []int {
	ids := append([]int(nil), g.partyJoinOrder...)
	seen := map[int]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	for id := range g.partyRoster {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	itemType, known := g.shopItemTypes[good.ID]
	out := ids[:0]
	for _, id := range ids {
		u, ok := g.partyRoster[id]
		if !ok {
			continue
		}
		if known && itemType < 0x20 && !campaign.CanEquip(u.ClassID, itemType, g.shopEquipTypes) {
			continue
		}
		out = append(out, id)
	}
	return out
}

func (g *Game) shopSellIDs() []int {
	ids := append([]int(nil), g.partyJoinOrder...)
	seen := map[int]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	for id := range g.partyRoster {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	out := ids[:0]
	for _, id := range ids {
		if u, ok := g.partyRoster[id]; ok && len(u.Inventory) > 0 {
			out = append(out, id)
		}
	}
	return out
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
		// 淡出/淡入或走位動畫進行中(doc46 §5.2/§5.3)不接受輸入,避免重複觸發轉場。
		if enter && g.fade == nil && len(g.storyWalks) == 0 {
			if g.dlgAdvance() && len(g.dialog) == 0 { // 翻頁優先;翻完換句、句盡才進下一節點
				g.advanceStoryNode(n)
			}
		}
		return true
	case "cutscene":
		// BeatRunner 驅動:目前這一拍是不是「等對白播完」全看 g.dialog 是否非空
		// (只有 dialog beat 會填它),其餘拍(pan/walk/act/fade/delay)Enter 無作用,
		// 交給 Update 各自的計時/佇列機制推進,不在這裡搶著 advance。
		if enter && len(g.dialog) > 0 {
			if g.dlgAdvance() && len(g.dialog) == 0 { // 翻頁優先;翻完換句、句盡才進下一拍
				g.beatAdvance()
			}
		}
		return true
	case "choice", "town":
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
	case "preparation", "church":
		if n.Type == "preparation" {
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.prepSel > 0 {
				g.prepSel--
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.prepSel+1 < len(g.prepIDs) {
				g.prepSel++
			}
			if enter && len(g.prepIDs) > 0 {
				id := g.prepIDs[g.prepSel]
				g.partyDeploy[id] = !g.partyDeploy[id]
				// 0x318ad automatically leaves once its 0x0f/0x13 quota is met.
				target := g.prepLimit
				if len(g.prepIDs) < target {
					target = len(g.prepIDs)
				}
				if g.preparationSelected() >= target {
					g.camp.Advance("")
					g.enterNode()
				}
			}
			// Early campaigns can have fewer joined characters than the native
			// quota; ESC is the lossless fallback to confirm that smaller roster.
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && g.preparationSelected() > 0 {
				g.camp.Advance("")
				g.enterNode()
			}
			return true
		}
		if g.churchMode == "menu" {
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.churchSel > 0 {
				g.churchSel--
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.churchSel < 3 {
				g.churchSel++
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				g.camp.Advance("")
				g.enterNode()
				return true
			}
			if enter {
				switch g.churchSel {
				case 2, 3: // native 0x30dc3 revive / 0x31385 class-change services
					g.churchMode = map[int]string{2: "revive", 3: "class"}[g.churchSel]
					g.churchIDs = g.churchCandidates(g.churchMode)
					g.churchSel = 0
				default:
					g.msg = "此教會服務尚待原版 callee 完整接線"
				}
			}
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.churchMode = "menu"
			g.churchIDs = nil
			g.churchBranches = nil
			g.churchClassID = -1
			g.churchSel = 0
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.churchSel > 0 {
			g.churchSel--
		}
		listLen := len(g.churchIDs)
		if g.churchMode == "class_target" {
			listLen = len(g.churchBranches)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.churchSel+1 < listLen {
			g.churchSel++
		}
		if enter && len(g.churchIDs) > 0 {
			if g.churchMode == "class_target" {
				if g.churchSel >= len(g.churchBranches) {
					return true
				}
				id := g.churchClassID
				u := g.partyRoster[id]
				branch := g.churchBranches[g.churchSel]
				row, ok := g.classChangeGrowth[branch.Portrait]
				if !ok {
					g.msg = fmt.Sprintf("缺少轉職成長列 portrait=%02Xh", branch.Portrait)
					return true
				}
				if err := campaign.ApplyClassChange(&u, branch.Portrait, branch.ClassID, branch.MobilityIncrement, row, g.rng, branch.InventoryIndex); err != nil {
					g.msg = err.Error()
					return true
				}
				u.ClsName = campaign.ClassName(branch.ClassID)
				campaign.RecomputeAfterClassChange(&u, g.shopItemStats)
				g.partyRoster[id] = u
				g.msg = fmt.Sprintf("%s 已轉職為%s", u.Name, u.ClsName)
				g.churchMode, g.churchBranches, g.churchIDs = "class", nil, g.churchCandidates("class")
				g.churchSel = 0
				return true
			}
			id := g.churchIDs[g.churchSel]
			u := g.partyRoster[id]
			if g.churchMode == "revive" {
				if u.ClassID < 0 || u.ClassID >= len(g.reviveFeeRates) {
					g.msg = fmt.Sprintf("復活費率缺少 class=%d", u.ClassID)
				} else if gold, cost, err := campaign.ReviveUnit(g.gold, &u, g.reviveFeeRates[u.ClassID]); err != nil {
					g.msg = fmt.Sprintf("復活費用 %d G：%v", cost, err)
				} else {
					g.gold, g.partyRoster[id] = gold, u
					g.msg = fmt.Sprintf("%s 已復活（-%d G）", u.Name, cost)
					g.churchIDs = g.churchCandidates("revive")
					if g.churchSel >= len(g.churchIDs) {
						g.churchSel = 0
					}
				}
			} else {
				g.churchClassID = id
				g.churchBranches = campaign.ClassChangeTargets(&u, g.classChangeTable)
				if len(g.churchBranches) == 0 {
					g.msg = "缺少可用的轉職分支資料"
				} else {
					g.churchMode = "class_target"
					g.churchSel = 0
				}
			}
		}
		return true
	case "shop":
		goods := g.camp.ShopGoods()
		if g.shopEquipPrompt {
			// Original ESC at the equip prompt means "leave it unequipped";
			// the purchase still completes and money is deducted last.
			if enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				u := g.partyRoster[g.shopEquipUnit]
				if enter {
					if err := campaign.EquipItem(&u, g.shopEquipSlot, g.shopItemStats); err != nil {
						g.msg = err.Error()
						return true
					}
					for len(u.Equipped) < len(u.Inventory) {
						u.Equipped = append(u.Equipped, false)
					}
					u.Equipped[g.shopEquipSlot] = true
				}
				g.gold = campaign.FinalizeGood(g.gold, g.shopPending)
				g.partyRoster[g.shopEquipUnit] = u
				g.shopEquipPrompt = false
				g.msg = fmt.Sprintf("買下 %s(-%d G)", g.shopPending.Name, g.shopPending.Price)
			}
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			if g.shopMode == "sell" {
				g.shopMode = "buy"
			} else {
				g.shopMode = "sell"
			}
			g.shopSellPicking = false
			g.shopSellUnitSel, g.shopSellSlotSel = 0, 0
			return true
		}
		if g.shopMode == "sell" {
			ids := g.shopSellIDs()
			if g.shopSellPicking {
				u := g.partyRoster[ids[g.shopSellUnitSel]]
				if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.shopSellSlotSel > 0 {
					g.shopSellSlotSel--
				}
				if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.shopSellSlotSel < len(u.Inventory)-1 {
					g.shopSellSlotSel++
				}
				if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
					g.shopSellPicking = false
					return true
				}
				if enter && g.shopSellSlotSel < len(u.Inventory) {
					itemID := u.Inventory[g.shopSellSlotSel]
					price, ok := g.shopItemPrices[itemID]
					if !ok {
						g.msg = fmt.Sprintf("物品 %02Xh 沒有價格資料", itemID)
					} else if gold, err := campaign.SellSlot(g.gold, &u, g.shopSellSlotSel, price); err != nil {
						g.msg = err.Error()
					} else {
						campaign.RecomputeEquipment(&u, g.shopItemStats)
						g.gold, g.partyRoster[ids[g.shopSellUnitSel]] = gold, u
						g.msg = fmt.Sprintf("賣出物品 %02Xh(+%d G)", itemID, price*3/4)
						if len(u.Inventory) == 0 {
							g.shopSellPicking = false
						}
						if g.shopSellSlotSel >= len(u.Inventory) && g.shopSellSlotSel > 0 {
							g.shopSellSlotSel--
						}
					}
				}
				return true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.shopSellUnitSel > 0 {
				g.shopSellUnitSel--
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.shopSellUnitSel < len(ids)-1 {
				g.shopSellUnitSel++
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				g.shopMode = "buy"
				return true
			}
			if enter && len(ids) > 0 {
				g.shopSellPicking = true
				g.shopSellSlotSel = 0
			}
			return true
		}
		if g.shopPicking {
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.shopRecipientSel > 0 {
				g.shopRecipientSel--
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.shopRecipientSel < len(g.shopRecipients)-1 {
				g.shopRecipientSel++
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				g.shopPicking = false
				return true
			}
			if enter && len(g.shopRecipients) > 0 {
				id := g.shopRecipients[g.shopRecipientSel]
				u := g.partyRoster[id]
				slot, err := campaign.ReserveGood(g.gold, &u, g.shopPending)
				if err != nil {
					g.msg = err.Error()
				} else {
					g.partyRoster[id] = u
					g.shopPicking = false
					if g.shopItemTypes[g.shopPending.ID] < 0x20 {
						g.shopEquipPrompt, g.shopEquipUnit, g.shopEquipSlot = true, id, slot
						g.msg = "要裝備上去嗎？ Enter=是，ESC=否"
					} else {
						g.gold = campaign.FinalizeGood(g.gold, g.shopPending)
						g.msg = fmt.Sprintf("買下 %s(-%d G)", g.shopPending.Name, g.shopPending.Price)
					}
				}
			}
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.shopSel > 0 {
			g.shopSel--
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.shopSel < len(goods)-1 {
			g.shopSel++
		}
		if enter && g.shopSel < len(goods) { // 購買
			gd := goods[g.shopSel]
			g.shopPending = gd
			g.shopRecipients = g.shopReceiverIDs(gd)
			g.shopRecipientSel = 0
			if len(g.shopRecipients) == 0 {
				g.msg = "沒有人可以收下這件物品!"
			} else {
				g.shopPicking = true
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) { // 離店
			g.camp.Advance("")
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

// ringInput radial 指令環 + 法術選單輸入。回傳 true = 已攔截。
// 方向配對(↑0道具/←1攻擊/→2魔法或狀態/↓3待機)為可玩性配置;原版方向↔指令待 dosbox 驗證(worklist)。
func (g *Game) ringInput() bool {
	enter := inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)
	esc := inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace)
	if g.spellOpen { // 法術選單
		if g.sel == nil {
			g.spellOpen = false
			return false
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.spellSel > 0 {
			g.spellSel--
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.spellSel < len(g.sel.Spells)-1 {
			g.spellSel++
		}
		if esc {
			g.spellOpen, g.ring = false, true
		}
		if enter && g.spellSel < len(g.sel.Spells) {
			id := g.sel.Spells[g.spellSel]
			for i := range g.spells {
				if g.spells[i].ID == id {
					if g.sel.MP < g.spells[i].MP {
						g.msg = "MP 不足!"
						return true
					}
					g.castSp = &g.spells[i]
					g.spellOpen = false
					g.msg = fmt.Sprintf("%s:選擇目標(射程 %d)", g.spells[i].Name, g.spells[i].Dist)
					break
				}
			}
		}
		return true
	}
	if !g.ring || g.sel == nil {
		return false
	}
	// 環導航(doc13 [0x3C57]:↑0/←1/→2/↓3)
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.ringSel = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.ringSel = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.ringSel = 2
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.ringSel = 3
	}
	if esc { // ESC = 取消(doc13):退回移動前位置,回到「選此單位待移動」;原地開環(未移動)則直接取消選取
		g.ring = false
		g.msg = ""
		if g.sel.X == g.selOrigX && g.sel.Y == g.selOrigY {
			g.sel, g.reach, g.moved = nil, nil, false
		} else {
			g.sel.X, g.sel.Y = g.selOrigX, g.selOrigY
			g.sel.OffX, g.sel.OffY = 0, 0
			g.moved = false
			g.reach = g.st.Reachable(g.sel)
			g.curX, g.curY = g.sel.X, g.sel.Y
		}
		return true
	}
	if enter {
		switch g.ringSel {
		case 0: // 道具(未實裝)
			g.msg = "道具:尚未實裝"
		case 1: // 攻擊 → 關環,進選目標(游標移到攻擊範圍內的敵人;範圍依武器射程,doc32)
			g.ring = false
			g.msg = "攻擊:選擇目標"
		case 2: // 魔法(有法術者)/狀態
			if len(g.sel.Spells) > 0 {
				if g.sel.Sealed {
					g.msg = "被封咒,無法施法!"
				} else {
					g.ring, g.spellOpen, g.spellSel = false, true, 0
				}
			} else {
				g.msg = fmt.Sprintf("%s Lv%d HP%d/%d MP%d AP%d DP%d",
					g.sel.Name, g.sel.Lv, g.sel.HP, g.sel.MaxHP, g.sel.MP, g.sel.AP, g.sel.DP)
			}
		case 3: // 待機
			g.finishSelectedWait()
		}
	}
	return true
}

// finishSelectedWait 對應原版行動選單第四項「下／休息」。0x19077 在未移動時
// 先回復 MaxHP 20%，接著 0x1908b 無論是否移動都呼叫 0x190ac 檢查當格寶物；
// 不是踩上格子立即開箱，也不是道具指令。
func (g *Game) finishSelectedWait() {
	u := g.sel
	if u == nil {
		return
	}
	g.ring = false
	if u.X == g.selOrigX && u.Y == g.selOrigY && u.HP > 0 && u.HP < u.MaxHP {
		heal := u.MaxHP / 5
		if heal < 1 {
			heal = 1
		}
		u.HP += heal
		if u.HP > u.MaxHP {
			u.HP = u.MaxHP
		}
	}
	if before, exists := g.st.TreasureAt(u.X, u.Y); exists {
		if got, ok := g.st.ClaimTreasure(u, u.X, u.Y); ok {
			if got.Kind == "gold" {
				g.gold += got.Value
				g.msg = fmt.Sprintf("取得 %d 金幣", got.Value)
			} else {
				g.msg = fmt.Sprintf("取得物品 %02Xh", got.Value)
			}
		} else if before.Kind == "item" && len(u.Inventory) >= 8 {
			g.msg = "物品欄已滿，寶物仍留在原處"
		}
	}
	u.Acted = true
	g.sel, g.reach, g.moved = nil, nil, false
}

// awardDeathReward 執行 exporter 已 lower 的可編輯 death_reward。原版特殊 handler
// id39/id41 分別把 00 D3 00／00 D5 00 交給同一 reward dispatcher；不是把敵人整個
// inventory 搬給攻擊者。item 優先放入擊殺者，滿欄時暫用隊伍空格承接，直到物品
// 使用／給予 UI 完成後再還原原版的互動轉移提示。
func (g *Game) awardDeathReward(dead, killer *battle.Unit) {
	if dead == nil || dead.Alive() || dead.DeathReward == nil {
		return
	}
	if g.deathRewarded == nil {
		g.deathRewarded = make(map[*battle.Unit]bool)
	}
	if g.deathRewarded[dead] {
		return
	}
	g.deathRewarded[dead] = true
	r := dead.DeathReward
	switch r.Type {
	case 0:
		awarded := false
		if killer != nil && killer.Camp == battle.Own && len(killer.Inventory) < 8 {
			awarded = killer.AddInventoryItem(r.Value, false)
		} else {
			awarded = g.grantItemToParty(r.Value)
		}
		if awarded {
			g.msg = fmt.Sprintf("擊破敵人，取得物品 %02Xh", r.Value)
		} else {
			g.msg = fmt.Sprintf("物品欄已滿，未能取得 %02Xh", r.Value)
		}
	case 1:
		g.gold += r.Value
		g.msg = fmt.Sprintf("擊破敵人，取得 %d 金幣", r.Value)
	}
}

// walkAnim 沿路徑逐格行走(玩家/AI 移動;FDICON 方向走動幀 + OffX/OffY 內插)。
type walkAnim struct {
	u    *battle.Unit
	path []battle.Cell // 含起點
	seg  int           // 目前段:path[seg] → path[seg+1]
	t    float64       // 段內進度 0→1
	then func()        // 走完回呼(nil=玩家預設:開指令環)
}

// SFX 事件 index(doc36 第 9 輪對照:index 0=游標移動已確認(5 處方向鍵分支證據);
// 0xc=「已選定」旗標伴隨音(疑確認,handle B 疊播)。戰鬥命中音屬另一獨立池
// ([0x5411f] 動態子容器,尚未導出)——暫用 UI 池 sfx_03(長音)代打,待戰鬥池導出換正。
const (
	sfxCursor  = 0
	sfxConfirm = 12
	sfxHit     = 3 // ⚠ 暫代(真戰鬥音效池待導出)
)

// loadMap 載入一張戰場(dir 下的 map.json + tileset.png,並切圖塊)。
// dir 例:"assets"(map0 舊結構)或 "assets/maps/map3"(全 33 圖匯出結構)。
func (g *Game) loadMap(dir string) error {
	dir = assetPath(dir)
	raw, err := os.ReadFile(dir + "/map.json")
	if err != nil {
		return err
	}
	var m MapData
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	pngRaw, err := os.ReadFile(dir + "/tileset.png")
	if err != nil {
		return err
	}
	img, _, err := image.Decode(bytes.NewReader(pngRaw))
	if err != nil {
		return err
	}
	g.tileset = ebiten.NewImageFromImage(img)
	g.tiles = nil
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
	return nil
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
	atkHP, atkMax, atkLV, atkMP, defLV, defMP, defHP0, defHP1, defMax, terrain int, atkOwn bool) *atkAnim {
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
		fpt: fpt, terrain: terrain, atkOwn: atkOwn}
}

// figaniIndex:戰鬥全身動畫 FIGANI index = FDICON組 × 3(恆等,反組譯 0x2884c:unit[+7]×3;doc06)。
// unit[+7]=FDICON組(0x11019 ×12→地圖sprite、0x2884c ×3→FIGANI,同一欄)。我方敵方統一:
// 索爾組0→FIGANI0、亞雷斯組4→12、盜賊組96→288。地圖組=FIGANI/3,不需對應表。
func figaniIndex(fig int) int { return fig * 3 }

// loadSprites 載入 assets/sprites/fig_NNN_fMM.png,按 fig index 分組成幀序列。
func loadSprites() map[int][]*ebiten.Image {
	out := map[int][]*ebiten.Image{}
	files := assetGlob("assets/sprites/fig_*_f*.png")
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
	files := assetGlob("assets/portraits/DATO_*_m*.png")
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
	files := assetGlob("assets/figani/FIGANI_*_f*.png")
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

// loadStoryScript 讀本機劇情文本檔(story-pipe 管線輸出:{"scenes":[{"label","lines":[{speaker,text}]}]})。
// scene 為空:舊行為,攤平全部 scenes 的 lines(整份劇本塞一個節點,ch02-33 尚未逐段接線時的 fallback)。
// scene 非空(doc46 §5.2):只取 label 對映的那一段——**每個 story 節點播一段,別把整份劇本攤平**,
// 才能讓場景/鏡頭/擺位跟著劇情分段切換,不會「一次播完才進戰場」。找不到該 label 時回 nil(呼叫端
// fallback 用節點內嵌 Lines,不會靜默播錯段)。檔案缺失(玩家未自備素材)同樣回 nil。
func loadStoryScript(path, scene string) []campaign.Line {
	return loadStoryScriptAt(path, scene, nil)
}

// loadStoryScriptAt extends the label-oriented legacy loader with an exact
// scene index.  Handler mappings use scene_index because editable scripts may
// intentionally contain an unlabeled scene or repeat a label; in that mode
// the index is authoritative and an invalid index fails closed.
func loadStoryScriptAt(path, scene string, sceneIndex *int) []campaign.Line {
	raw, err := os.ReadFile(assetPath(path))
	if err != nil {
		return nil
	}
	var f struct {
		Scenes []struct {
			Label string          `json:"label"`
			Lines []campaign.Line `json:"lines"`
		} `json:"scenes"`
	}
	if json.Unmarshal(raw, &f) != nil {
		return nil
	}
	if sceneIndex != nil {
		if *sceneIndex < 0 || *sceneIndex >= len(f.Scenes) {
			return nil
		}
		return f.Scenes[*sceneIndex].Lines
	}
	if scene == "" {
		var out []campaign.Line
		for _, sc := range f.Scenes {
			out = append(out, sc.Lines...)
		}
		return out
	}
	for _, sc := range f.Scenes {
		if sc.Label == scene {
			return sc.Lines
		}
	}
	return nil
}

// handlerStoryPath converts a StoryIndexMap script path (relative to
// assets/story) into the normal asset lookup path.  Existing authored beats
// may already carry assets/... or an absolute path, which stays untouched.
func handlerStoryPath(script string) string {
	if filepath.IsAbs(script) || strings.HasPrefix(filepath.ToSlash(script), "assets/") {
		return script
	}
	return filepath.Join("assets", "story", script)
}

// loadFigMeta 載入 FIGANI 每幀內嵌絕對螢幕座標 (dx,dy)@320(assets/figani/meta.json;doc06:
// 幀標頭 +0/+2,動畫的走位/伸擊/突刺全靠逐幀 (dx,dy) 變化,引擎不需錨點/位移計算)。
func loadFigMeta() map[int][][2]int {
	out := map[int][][2]int{}
	raw, err := os.ReadFile(assetPath("assets/figani/meta.json"))
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
	g.playSFX(sfxConfirm)
	cur := battle.Cell{X: g.curX, Y: g.curY}
	if g.sel == nil { // 選我方單位
		u := g.st.UnitAt(g.curX, g.curY)
		if u != nil && u.Camp == battle.Own && u.Paralyzed {
			g.msg = "麻痺中,無法行動!"
			return
		}
		if u != nil && u.Camp == battle.Own && !u.Acted {
			g.sel = u
			g.moved = false
			g.reach = g.st.Reachable(u)
			g.selOrigX, g.selOrigY = u.X, u.Y // 記移動前位置(ESC 取消退回,playfix #4)
		}
		return
	}
	if g.castSp != nil { // 施法目標選擇:游標=施放中心(AoE 可指空地;單體需單位)→ CastArea 結算
		sp := *g.castSp
		if !g.st.InCastRange(g.sel, sp, g.curX, g.curY) {
			return
		}
		if sp.Range == 0 { // 單體:中心格需有合法目標
			tgt := g.st.UnitAt(g.curX, g.curY)
			okCamp := tgt != nil && ((sp.Target == 0 && tgt.Camp != battle.Own) || (sp.Target == 1 && tgt.Camp == battle.Own))
			if !okCamp {
				return
			}
		}
		results := g.st.CastArea(g.sel, g.curX, g.curY, sp, g.rng)
		if results == nil {
			g.msg = "MP 不足或被封咒!"
			return
		}
		// 訊息彙總 + 單體攻擊接全螢幕演出
		hitN, missN, total := 0, 0, 0
		var first *battle.CastResult
		for i := range results {
			if results[i].Missed {
				missN++
				continue
			}
			hitN++
			total += results[i].Amount
			if first == nil {
				first = &results[i]
			}
		}
		for i := range results {
			g.awardDeathReward(results[i].Target, g.sel)
		}
		verb := "造成"
		if sp.Target == 1 {
			verb = "回復"
		}
		g.msg = fmt.Sprintf("%s 施放 %s:命中 %d(%s %d)", g.sel.Name, sp.Name, hitN, verb, total)
		if missN > 0 {
			g.msg += fmt.Sprintf("、Miss %d", missN)
		}
		if sp.Target == 0 && first != nil && first.Amount > 0 { // 攻擊法術演出(首目標)
			tgt := first.Target
			nm := tgt.Name
			if nm == "" {
				nm = tgt.ClsName
			}
			g.atk = g.newAtkAnim(g.sel.Fig, tgt.Fig, g.sel.Name, nm,
				g.sel.HP, g.sel.MaxHP, g.sel.Lv, g.sel.MP, tgt.Lv, tgt.MP,
				tgt.HP+first.Amount, tgt.HP, tgt.MaxHP, g.terrainAt(tgt.X, tgt.Y), true)
		}
		g.sel.Acted = true
		g.sel.Dir = dirToward(g.sel.X, g.sel.Y, g.curX, g.curY)
		g.castSp, g.sel, g.reach, g.moved = nil, nil, nil, false
		g.checkResult()
		return
	}
	if !g.moved { // 移動階段
		switch {
		case g.curX == g.sel.X && g.curY == g.sel.Y: // 原地 → 不移動,開指令環
			g.moved = true
			g.reach = nil
			g.ring, g.ringSel = true, 1
		case g.reach[cur] && g.st.UnitAt(g.curX, g.curY) == nil: // 移動到可達空格:沿路徑逐格走
			if p := g.st.Path(g.sel, g.curX, g.curY); len(p) >= 2 {
				g.walk = &walkAnim{u: g.sel, path: p}
			} else { // 理論上不會(reach 內必可達),保底瞬移
				g.sel.X, g.sel.Y = g.curX, g.curY
				g.moved = true
				g.ring, g.ringSel = true, 1
			}
			g.reach = nil
		}
		return
	}
	// 攻擊階段:游標在攻擊範圍內的敵 → 攻擊;在自己格 → 待命
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
		g.awardDeathReward(tgt, g.sel)
		g.msg = fmt.Sprintf("%s 攻擊 %s,造成 %d 傷害", anm, nm, dmg)
		g.atk = g.newAtkAnim(g.sel.Fig, tgt.Fig, anm, nm,
			g.sel.HP, g.sel.MaxHP, g.sel.Lv, g.sel.MP, tgt.Lv, tgt.MP,
			defHP0, tgt.HP, tgt.MaxHP, g.terrainAt(g.curX, g.curY), true) // 戰鬥背景 = 守方格地形
		g.sel, g.reach, g.moved = nil, nil, false
		g.checkResult()
	} else if g.curX == g.sel.X && g.curY == g.sel.Y { // 原地待命
		g.finishSelectedWait()
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
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) { // 全域:切換音源(MT-32 / Sound Blaster)
		g.cycleBGMSource()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) { // 全域:開發除錯 HUD 開關
		g.debug = !g.debug
	}
	if g.bannerT > 0 {
		g.bannerT--
	}
	if g.titlePhase != "" {
		if g.titleUpdate() {
			if g.shotPath != "" && g.frame > g.shotFrame { // 截圖模式在 title 也要能退出
				return ebiten.Termination
			}
			return nil
		}
	}
	// 攻擊演出推進(FIGANI 全身分鏡;演出期間鎖玩家輸入)
	if g.atk != nil {
		g.atk.timer--
		a := g.atk
		if a.fpt > 0 { // 三段音效(== 比對每 tick 遞增的 prog,各觸發一次)
			prog := a.total - a.timer
			swingAt := (len(g.figani[a.atkFig]) - 4) * a.fpt
			switch prog {
			case swingAt: // 揮擊(蓄力揮出)
				g.playRaw(g.sfxSwing)
			case swingAt + 3*a.fpt: // 命中(劈中、守方 HP 抽乾)
				g.playRaw(g.sfxImpact)
			}
		}
		if a.timer == a.fpt && a.defHP1 <= 0 { // 收勢那幀:守方陣亡音
			g.playRaw(g.sfxDeath)
		}
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
			// Dir 不重設:保留最後一段的移動朝向(playfix #8 回報「走完又轉回正面朝玩家」是 bug,
			// 原版待機朝向未 RE 出確切規則,查無反例前用「維持最後移動方向」為合理預設)
			g.walk = nil
			if w.then != nil { // AI:走完執行攻擊/收尾
				w.then()
			} else { // 玩家:開指令環
				g.moved = true
				g.ring, g.ringSel = true, 1
			}
		} else {
			a, b := w.path[w.seg], w.path[w.seg+1]
			w.u.Dir = dirToward(a.X, a.Y, b.X, b.Y)
			w.u.X, w.u.Y = b.X, b.Y // 單位掛在目標格,Off 從來源格內插到 0
			w.u.OffX = float64((a.X-b.X)*g.m.TileW) * (1 - w.t)
			w.u.OffY = float64((a.Y-b.Y)*g.m.TileH) * (1 - w.t)
		}
	}
	g.aiStep() // AI 回合驅動(aiBusy 時逐單位行走→攻擊演出)
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
			if os.Getenv("FD2_SHOT_RING") != "" { // 截圖驗證:選單位後原地開指令環
				g.dialog = nil // 清開場對白(避免蓋住環)
				g.confirm()
				g.confirm()
			}
			if os.Getenv("FD2_SHOT_SPELL") != "" { // 截圖驗證:開法術選單
				g.dialog = nil
				g.confirm()
				g.confirm()
				if g.sel != nil && len(g.sel.Spells) > 0 {
					g.ring, g.spellOpen, g.spellSel = false, true, 0
				}
			}
			if v := os.Getenv("FD2_SHOT_ATTACK"); v != "" { // 全螢幕戰鬥演出(驗證用):亞雷斯打盜賊
				g.dialog = nil // 清開場對白(避免蓋住演出)
				fig, _ := strconv.Atoi(v)
				g.atk = g.newAtkAnim(fig, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 2, 0, 28, 8, 28, 0, true)
			}
			if os.Getenv("FD2_SHOT_ATKSEL") != "" { // 截圖驗證:選單位→原地開環→模擬環選「攻擊」(ringSel==1)
				// 關環,進攻擊目標選擇階段(驗證武器攻擊距離高亮,doc32;搭配 FD2_SHOT_CUR 指定選哪個單位)。
				// 環的 case1 本身由 ringInput() 真實按鍵觸發(inpututil 偵測,截圖模式無法送假按鍵),
				// 這裡直接複製 case1 的狀態轉移(g.ring=false),不能改呼叫 g.confirm() 三次
				// (ring 開啟時 confirm() 不會攔下,會落到「在自己格→待命」把 g.sel 清掉)。
				g.dialog = nil
				g.confirm() // 選取游標上的單位
				g.confirm() // 原地(游標未動)→ 開指令環(moved=true, ring=true, ringSel=1)
				g.ring = false
				g.msg = "攻擊:選擇目標"
				if v := os.Getenv("FD2_SHOT_CUR2"); v != "" { // 進攻擊階段後把游標挪開(驗證高亮不被HUD面板擋住)
					fmt.Sscanf(v, "%d,%d", &g.curX, &g.curY)
				}
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
	g.stepStoryWalks()                           // 場景走位動畫(doc46 §5.3);storyWalks 為空時內部直接返回
	g.stepActJob()                               // beat「act」姿態循環(doc50);actJob 為空時內部直接返回
	g.stepFocusUnit()                            // beat「focus_unit」依原版安全帶逐格移動游標／鏡頭
	g.stepDlgAnim()                              // 對話框換人縮/展動畫(使用者回饋 #3)
	g.stepFade()                                 // 場景淡出/淡入轉場(doc46 §5.2;beat「fade」兩個方向都靠 then 接回下一拍)
	if g.camp != nil && g.storyAutoAdvance > 0 { // 無對白節點自動轉場倒數(行軍蒙太奇)
		g.storyAutoAdvance--
		if g.storyAutoAdvance == 0 {
			if n := g.camp.Node(); n != nil {
				g.advanceStoryNode(n)
			}
		}
	}
	if g.beatDelay > 0 { // beat「delay」倒數(doc50 0x375b2)
		g.beatDelay--
		if g.beatDelay == 0 {
			g.beatAdvance()
		}
	}
	g.stepBattleEventDelay()
	if g.curX != g.prevCurX || g.curY != g.prevCurY { // 游標移動音
		g.playSFX(sfxCursor)
		g.prevCurX, g.prevCurY = g.curX, g.curY
	}
	// 相機跟隨游標(置中,夾在地圖內;先於各攔截,避免環/選單開啟時相機停擺)
	// storyBG 場景背景模式鏡頭固定(enterNode 已設 CamX/CamY),不跟游標走。
	// FollowWalk 節點例外:走位期間鏡頭鎖定走位者(原版長廊運鏡,2-1;視野=320×200 世界px)。
	if g.camPan != nil {
		g.stepCamPan()
	} else if g.storyBG {
		switch {
		case g.followWalk && len(g.storyWalks) > 0:
			w := g.storyWalks[0]
			u := &g.storyActors[w.actor]
			vw, vh := logicalW/storyZoom, logicalH/storyZoom
			g.camX = float64(u.X*g.m.TileW) + u.OffX + float64(g.m.TileW)/2 - float64(vw)/2
			g.camY = float64(u.Y*g.m.TileH) + u.OffY + float64(g.m.TileH)/2 - float64(vh)/2
			clamp(&g.camX, 0, float64(g.m.W*g.m.TileW-vw))
			clamp(&g.camY, 0, float64(g.m.H*g.m.TileH-vh))
			if g.camMaxY > 0 && g.camY > g.camMaxY { // 場景鏡頭上限(王座廳擋草地;走位者可從畫面外走入)
				g.camY = g.camMaxY
			}
		}
	} else if g.battleEvent == nil {
		g.camX = float64(g.curX*g.m.TileW - logicalW/2 + g.m.TileW/2)
		g.camY = float64(g.curY*g.m.TileH - logicalH/2 + g.m.TileH/2)
		clamp(&g.camX, 0, float64(g.m.W*g.m.TileW-logicalW))
		clamp(&g.camY, 0, float64(g.m.H*g.m.TileH-logicalH))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) { // 快速存檔(節點邊界語意:存 campaign 進度)
		g.saveGame()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) { // 快速讀檔
		g.loadGame()
	}
	if g.battleEvent != nil {
		if len(g.dialog) > 0 && (inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)) {
			if g.dlgAdvance() && len(g.dialog) == 0 {
				g.advanceBattleEvent()
			}
		}
		return nil // PAN/delay/dialogue sequence blocks battle input and repeated end-turn
	}
	if g.campInput() { // campaign 節點(story/choice/ending/勝敗轉場)攔截輸入
		return nil
	}
	if g.ringInput() { // radial 指令環 / 法術選單
		return nil
	}
	if len(g.dialog) > 0 { // 戰鬥起手對白(g.camp==nil 直接開局,或 campaign battle 節點無 story 攔截):Enter/Space 逐句清除
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.dlgAdvance() // 翻頁優先,翻完換句(長對白分頁,不截斷)
		}
		return nil
	}
	if g.castSp != nil { // 施法目標選擇:ESC 取消回環
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.castSp = nil
			g.ring = true
			return nil
		}
	}
	// 游標移動:方向鍵 / WASD(按住持續移動,keyRepeat)/ 觸控
	if keyRepeat(ebiten.KeyArrowLeft) || keyRepeat(ebiten.KeyA) {
		g.curX--
	}
	if keyRepeat(ebiten.KeyArrowRight) || keyRepeat(ebiten.KeyD) {
		g.curX++
	}
	if keyRepeat(ebiten.KeyArrowUp) || keyRepeat(ebiten.KeyW) {
		g.curY--
	}
	if keyRepeat(ebiten.KeyArrowDown) || keyRepeat(ebiten.KeyS) {
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
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if g.sel != nil && g.moved { // 已移動、正在選攻擊目標:退回指令環(取消一層,doc13;ring 的 ESC 才真正退回原位)
			g.ring = true
			g.msg = ""
		} else {
			g.sel, g.reach = nil, nil
			g.msg = ""
		}
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

// keyRepeat 方向鍵按住持續觸發(playfix #1):首次按下立即動一格,按住 12 tick 後
// 每 5 tick 再動一格(可玩性手感參數,原版掃描碼節奏未逐格量測,見 doc13 §游標)。
func keyRepeat(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	if d == 0 {
		return false
	}
	if d == 1 {
		return true
	}
	if d < 12 {
		return false
	}
	return (d-12)%5 == 0
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
	if g.titlePhase != "" {
		g.drawTitle(screen)
		if g.shotPath != "" && g.frame == g.shotFrame {
			saveShot(screen, g.shotPath)
		}
		return
	}
	if g.m == nil {
		ebitenutil.DebugPrint(screen, "FD2 重製 MVP\n缺 assets/(tileset.png + map.json)\n用 tools/export_engine_assets.py 產生\n"+g.loadErr)
		if g.shotPath != "" && g.frame == g.shotFrame { // 打包驗證:資產缺失時也要能截圖存證(舊版此分支漏存,見打包 worklist)
			saveShot(screen, g.shotPath)
		}
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
	// 清背景:地圖比畫面窄時右/上下留黑邊(非殘影黃白)。
	// RE 結論(0x11eee 地形迴圈,doc 見 knowledge-base/25):原版戰場視窗固定 13×8 格(312×192px)、
	// 逐格 blit 全無 memset/fillrect,且全 34 張地圖最小 18×20 格恆大於視窗——這個「地圖比視窗窄」情境
	// 原版從未觸發,無「原版清色」可對齊;黑色是 remake 自訂 FOV(640 寬、tile 維持原生 24px)才會露出的
	// 邊,選黑純為視覺乾淨、非還原原版行為。
	screen.Fill(color.RGBA{0, 0, 0, 0xff})
	// story 場景與原版阻塞 battle event 走 320×200 離屏再放大 storyZoom 倍
	// (還原 13×8 格取景)；一般可操作戰場維持 640×400 直繪。
	// 對話框/HUD/淡幕仍畫在 screen 原生解析度。
	target, viewW, viewH := screen, logicalW, logicalH
	legacyViewport := g.storyBG || g.battleEvent != nil
	if legacyViewport {
		if g.storyView == nil {
			g.storyView = ebiten.NewImage(logicalW/storyZoom, logicalH/storyZoom)
		}
		g.storyView.Fill(color.RGBA{0, 0, 0, 0xff})
		target, viewW, viewH = g.storyView, logicalW/storyZoom, logicalH/storyZoom
	}
	tw, th := g.m.TileW, g.m.TileH
	// 只畫可見範圍
	x0 := int(g.camX) / tw
	y0 := int(g.camY) / th
	x1 := (int(g.camX)+viewW)/tw + 1
	y1 := (int(g.camY)+viewH)/th + 1
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
			target.DrawImage(t, op)
		}
	}
	// 移動範圍高亮(已選單位:藍色半透明格)
	if g.sel != nil {
		hl := ebiten.NewImage(tw, th)
		hl.Fill(color.RGBA{0x40, 0x80, 0xff, 0x66})
		for c := range g.reach {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(c.X*tw)-g.camX, float64(c.Y*th)-g.camY)
			target.DrawImage(hl, op)
		}
		if g.castSp != nil { // 施法射程高亮(紫)
			ch := ebiten.NewImage(tw, th)
			ch.Fill(color.RGBA{0xa0, 0x50, 0xe0, 0x5c})
			for y := 0; y < g.m.H; y++ {
				for x := 0; x < g.m.W; x++ {
					if g.st.InCastRange(g.sel, *g.castSp, x, y) {
						op := &ebiten.DrawImageOptions{}
						op.GeoM.Translate(float64(x*tw)-g.camX, float64(y*th)-g.camY)
						target.DrawImage(ch, op)
					}
				}
			}
		}
		// 攻擊射程高亮(紅;已移動、選攻擊、尚未選中目標的階段,doc32 武器攻擊距離接線 —
		// 沒有這格高亮,槍兵2格射程會「打得到但畫面看不出範圍」)
		if g.castSp == nil && g.moved && !g.ring && !g.spellOpen {
			ah := ebiten.NewImage(tw, th)
			ah.Fill(color.RGBA{0xe0, 0x30, 0x30, 0x5c})
			for y := 0; y < g.m.H; y++ {
				for x := 0; x < g.m.W; x++ {
					if g.st.InAttackRange(g.sel, x, y) {
						op := &ebiten.DrawImageOptions{}
						op.GeoM.Translate(float64(x*tw)-g.camX, float64(y*th)-g.camY)
						target.DrawImage(ah, op)
					}
				}
			}
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
			if ux < -float64(tw) || ux > float64(viewW) || uy < -float64(th) || uy > float64(viewH) {
				continue
			}
			g.drawUnitSprite(target, ux, uy, float64(tw), float64(th), u)
		}
	}
	// storyBG 場景靜態角色(doc23 §4:王座廳國王/王后/主角等 cutscene 擺位,同一 sprite 繪法無戰鬥邏輯)
	for i := range g.storyActors {
		u := &g.storyActors[i]
		if !u.OnField || !u.Alive() {
			continue
		}
		ux := float64(u.X*tw) - g.camX
		uy := float64(u.Y*th) - g.camY
		g.drawUnitSprite(target, ux, uy, float64(tw), float64(th), u)
	}
	if legacyViewport { // 離屏世界層放大貼回畫布(48px/格,原版取景)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(storyZoom, storyZoom)
		screen.DrawImage(g.storyView, op)
		if g.unitLabels { // FD2_UNIT_LABELS:cutscene sprite 左上標 [idx]名(x,y)dDir,協助回報/對映原版 slot
			for i := range g.storyActors {
				u := &g.storyActors[i]
				sx := (float64(u.X*tw) - g.camX + u.OffX) * storyZoom
				sy := (float64(u.Y*th) - g.camY + u.OffY) * storyZoom
				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("[%d]f%d(%d,%d)d%d", i, u.Fig, u.X, u.Y, u.Dir), int(sx), int(sy)-14)
			}
		}
	}
	// 游標(白框):指令環/法術選單開啟時不顯示(原版該狀態下選取指示只在環上的選中圖示,
	// 常駐白框會疊在中央、與環的選中框混淆,見 playfix #5)
	curPx := float64(g.curX*tw) - g.camX
	curPy := float64(g.curY*th) - g.camY
	campaignBattleView := g.camp == nil || (g.camp.Node() != nil && g.camp.Node().Type == "battle")
	if !g.ring && !g.spellOpen && !legacyViewport && campaignBattleView {
		drawCursor(screen, curPx, curPy, float64(tw), float64(th))
	}
	// HUD(對照原版 orig_04/08):游標單位資訊=左下面板(非常駐頂列);回合切換=中央大字橫幅。
	if g.st != nil && g.font != nil && !legacyViewport {
		if u := g.st.UnitAt(g.curX, g.curY); u != nil { // 左下單位面板(orig 樣式)
			g.drawUnitHUD(screen, u)
		}
		if g.debug { // F3:詳細除錯(回合/戰況/座標)
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("T%d own%d ally%d enemy%d cur(%d,%d)",
				g.st.Turn, g.st.AliveCount(battle.Own), g.st.AliveCount(battle.Ally), g.st.AliveCount(battle.Enemy), g.curX, g.curY), 6, 4)
		}
	}
	if !legacyViewport && campaignBattleView {
		g.drawPhaseBanner(screen) // 回合橫幅(PLAYER/ENEMY PHASE,transient)
	}

	// 中文層(原版點陣字型,doc 08):選中單位名 + 對話框(DebugPrint 不支援中文)
	if g.font != nil {
		if g.st != nil && !legacyViewport { // 選中單位中文名(放游標格上方,避開頂部 DebugPrint)
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
		g.drawRing(screen)
		g.drawSpellMenu(screen)
		if len(g.dialog) > 0 && !(g.storyBG && g.walkFirst && len(g.storyWalks) > 0) && g.dlgShown != dlgNone {
			// 對話框:原版素材(FDOTHER#5 LMI1 #21,310×99 素藍細邊框)+ orig 量測佈局。
			// walk_first 節點在進場走位期間不顯示(2-1:原版索爾走到王座前對話框才出現)。
			// 換人說話:框先垂直收合再展開(stepDlgAnim 相位;使用者回饋 #3),相位中不畫文字/頭像。
			dl := g.dialog[len(g.dialog)-1]
			sc := 1.0 // 垂直縮放(1=常態)
			switch g.dlgPhase {
			case 1:
				sc = 1 - float64(g.dlgT)/float64(dlgAnimFrames)
			case 2:
				sc = float64(g.dlgT) / float64(dlgAnimFrames)
			}
			// 依說話者切上/下框 + 左/右頭像(對照原版 orig_02_dialog:我方下框左頭像、對方/NPC 上框右頭像)
			// 相位中(收合舊框)以 dlgShown 為準,避免框在收合前就跳到新說話者的位置。
			upper := g.dlgShown >= 32 // >=32 為對方/敵/NPC(我方角色 id 0-31)
			if g.dlgUpper != nil {    // per-line 覆蓋(campaign.Beat.Upper,doc55 草地幕亞雷斯進場句例外)
				upper = *g.dlgUpper
			}
			// 框位置:模板匹配 orig 下框 (5,112)@320(底部裁 11px 超出畫面,原版如此);上框鏡射 y=-11
			bx, by := 10.0, 198.0 // 下框上移使底邊396在畫面內(原224底邊422出畫面,使用者回饋2026-07-05)
			if upper {
				by = 4 // 上框下移使頂邊4在畫面內(原-22頂邊出畫面)
			}
			top := by
			if g.dlgBox != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(2, 2*sc)
				op.GeoM.Translate(bx, by+(1-sc)*99) // 以框垂直中心收合
				screen.DrawImage(g.dlgBox, op)
			} else { // 無素材 fallback:純色框
				box := ebiten.NewImage(620, 198)
				box.Fill(color.RGBA{0x2c, 0x44, 0x84, 0xf2})
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(1, sc)
				op.GeoM.Translate(bx, by+(1-sc)*99)
				screen.DrawImage(box, op)
			}
			if g.dlgPhase == 0 { // 縮/展相位中不畫頭像與文字
				// 框內部漸層:與頭像底色同一漸層(頂 40,69,138 → 底 56,85,154),消除頭像↔框接縫色差
				// (使用者回饋 2026-07-05)。疊在框素材上、頭像/文字之下。
				if g.dlgGrad == nil {
					const gh = 198
					gi := ebiten.NewImage(1, gh)
					for y := 0; y < gh; y++ {
						f := float64(y) / float64(gh-1)
						gi.Set(0, y, color.RGBA{
							uint8(40 + (56-40)*f), uint8(69 + (85-69)*f), uint8(138 + (154-138)*f), 255})
					}
					g.dlgGrad = gi
				}
				gop := &ebiten.DrawImageOptions{}
				gop.GeoM.Scale(620-16, 182.0/198.0) // 1×198 → 內部 604×182(框邊界內縮 8px)
				gop.GeoM.Translate(bx+8, by+8)
				screen.DrawImage(g.dlgGrad, gop)
				// 頭像:側臉,收進框內(不凸出框頂),臉朝文字(對照 orig_02:我方左朝右、對方上框右朝左)。
				const ps = 2.1 // 80×80 DATO → 168px,收進框高(~176px)內
				hx, tx, ty := 16.0, 216.0, by+24
				hy := by + (198-80*ps)/2 // 頭像垂直置中於框內(框高198,頭像168),不凸出框上下邊
				if upper {
					hx = float64(logicalW) - 16 - 80*ps
					tx = 32
					ty = by + 46
				}
				if fr := g.portraits[dl.Speaker]; len(fr) > 0 {
					mi := 0
					if g.mouthOpen && len(fr) > 3 {
						mi = 3
					}
					// 原生 DATO 面朝右;要臉朝文字:下框(頭像在左)朝右=鏡像、上框(頭像在右)朝左=不鏡像。
					po := &ebiten.DrawImageOptions{}
					if upper { // 上框:頭像在右,臉朝左文字 → 原生朝右不鏡像
						po.GeoM.Scale(ps, ps)
						po.GeoM.Translate(hx, hy)
					} else { // 下框:頭像在左,臉朝右文字 → 鏡像
						po.GeoM.Scale(-ps, ps)
						po.GeoM.Translate(hx+80*ps, hy)
					}
					screen.DrawImage(fr[mi], po)
				} else {
					tx = 32
				}
				// 自動換行(dlgWrap 與 Enter 分頁共用)。每頁最多 3 行,超過分頁,渲染目前頁 g.dlgPage。
				lines := dlgWrap(dl)
				start := g.dlgPage * 3
				for i := 0; i < 3 && start+i < len(lines); i++ {
					g.font.Draw(screen, lines[start+i], tx, ty+float64(i)*38, 1.7, color.RGBA{0xf0, 0xf4, 0xff, 0xff})
				}
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

	// 場景淡出/淡入轉場(doc46 §5.2):全螢幕黑色疊層,alpha 隨 fade.t 漸變。
	if g.fade != nil {
		frac := float64(g.fade.t) / float64(g.fade.total)
		alpha := frac
		if !g.fade.out {
			alpha = 1 - frac
		}
		ov := ebiten.NewImage(logicalW, logicalH)
		ov.Fill(color.RGBA{0, 0, 0, uint8(alpha * 0xff)})
		screen.DrawImage(ov, &ebiten.DrawImageOptions{})
	}

	// 截圖鉤子:指定幀把畫面存 PNG(無人值守驗證用)
	if g.shotPath != "" && g.frame == g.shotFrame {
		saveShot(screen, g.shotPath)
	}
}

// drawRing radial 指令環(原版 4 圖示十字繞單位,orig_04;圖示=原版截圖裁出素材)。
func (g *Game) drawRing(screen *ebiten.Image) {
	if !g.ring || g.sel == nil || g.m == nil {
		return
	}
	tw, th := g.m.TileW, g.m.TileH
	ux := float64(g.sel.X*tw) - g.camX
	uy := float64(g.sel.Y*th) - g.camY
	// 行動中單位標記 + 補畫其 sprite 在最上層:部署較密的隊形下,鄰格友軍的 sprite
	// 可能探進環的中央空隙,讓人誤以為環中央是別的角色(playfix #5)。用橘色底標記
	// 「這是誰在動」+ 把 g.sel 自己的 sprite 疊到最上層,消除歧義。
	mark := ebiten.NewImage(tw, th)
	mark.Fill(color.RGBA{0xff, 0xa8, 0x20, 0x50})
	mop := &ebiten.DrawImageOptions{}
	mop.GeoM.Translate(ux, uy)
	screen.DrawImage(mark, mop)
	g.drawUnitSprite(screen, ux, uy, float64(tw), float64(th), g.sel)
	const iw, ih = 56.0, 52.0 // 28×26 ×2
	pos := [4][2]float64{     // 0上 1左 2右 3下
		{ux + float64(tw)/2 - iw/2, uy - ih - 6},
		{ux - iw - 6, uy + float64(th)/2 - ih/2},
		{ux + float64(tw) + 6, uy + float64(th)/2 - ih/2},
		{ux + float64(tw)/2 - iw/2, uy + float64(th) + 6},
	}
	border := func(x, y float64, c color.RGBA) {
		for _, r := range [][4]float64{{x - 3, y - 3, iw + 6, 3}, {x - 3, y + ih, iw + 6, 3},
			{x - 3, y, 3, ih}, {x + iw, y, 3, ih}} {
			b := ebiten.NewImage(int(r[2]), int(r[3]))
			b.Fill(c)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(r[0], r[1])
			screen.DrawImage(b, op)
		}
	}
	for i, ic := range g.ringIcons {
		if ic == nil {
			continue
		}
		x, y := pos[i][0], pos[i][1]
		if i == g.ringSel { // 選中:橘黃框(orig 選中樣式)
			border(x, y, color.RGBA{0xff, 0xa8, 0x20, 0xff})
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(x, y)
		screen.DrawImage(ic, op)
	}
}

// drawSpellMenu 法術選單(名稱 + MP;↑↓選、Enter 施放、ESC 回環)。
func (g *Game) drawSpellMenu(screen *ebiten.Image) {
	if !g.spellOpen || g.sel == nil || g.font == nil {
		return
	}
	h := 44 + float64(len(g.sel.Spells))*30
	box := ebiten.NewImage(230, int(h))
	box.Fill(color.RGBA{0x10, 0x1c, 0x40, 0xee})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(20, 60)
	screen.DrawImage(box, op)
	g.font.Draw(screen, fmt.Sprintf("法術  MP %d", g.sel.MP), 34, 68, 1.0, color.RGBA{0xff, 0xe0, 0x90, 0xff})
	for i, id := range g.sel.Spells {
		var sp *battle.Spell
		for k := range g.spells {
			if g.spells[k].ID == id {
				sp = &g.spells[k]
				break
			}
		}
		if sp == nil {
			continue
		}
		c := color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
		pre := "　"
		if i == g.spellSel {
			c = color.RGBA{0xff, 0xff, 0xff, 0xff}
			pre = "▶"
		}
		if g.sel.MP < sp.MP {
			c = color.RGBA{0x80, 0x80, 0x90, 0xff} // MP 不足變暗
		}
		g.font.Draw(screen, fmt.Sprintf("%s%s  MP%d", pre, sp.Name, sp.MP), 32, 96+float64(i)*30, 1.0, c)
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
	case n.Type == "choice" || n.Type == "town":
		vis := g.camp.Visible()
		h := 60 + float64(len(vis))*28
		fillBox(160, 120, 320, h)
		title := n.Prompt
		if n.Type == "town" && n.Town != "" {
			title = n.Town + "　戰後整備"
		}
		g.font.Draw(screen, title, 176, 130, 1.1, color.RGBA{0xff, 0xe0, 0x90, 0xff})
		for i, o := range vis {
			c := color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			pre := "　"
			if i == g.campSel {
				c = color.RGBA{0xff, 0xff, 0xff, 0xff}
				pre = "▶"
			}
			g.font.Draw(screen, pre+o.Label, 190, 162+float64(i)*28, 1.0, c)
		}
	case n.Type == "shop":
		if g.shopEquipPrompt {
			fillBox(150, 150, 340, 120)
			u := g.partyRoster[g.shopEquipUnit]
			g.font.Draw(screen, fmt.Sprintf("%s：要裝備上去嗎？", u.Name), 170, 170, 1.1, color.RGBA{0xff, 0xe0, 0x90, 0xff})
			g.font.Draw(screen, "Enter 是　ESC 否（保留在物品欄）", 170, 210, 1.0, color.RGBA{0xff, 0xff, 0xff, 0xff})
			break
		}
		if g.shopMode == "sell" {
			ids := g.shopSellIDs()
			if g.shopSellPicking && g.shopSellUnitSel < len(ids) {
				u := g.partyRoster[ids[g.shopSellUnitSel]]
				h := 76 + float64(len(u.Inventory))*28
				fillBox(140, 60, 360, h)
				g.font.Draw(screen, fmt.Sprintf("賣出：%s（Tab 返回購買）", u.Name), 156, 70, 1.0, color.RGBA{0xff, 0xe0, 0x90, 0xff})
				for i, id := range u.Inventory {
					pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
					if i == g.shopSellSlotSel {
						pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
					}
					price := g.shopItemPrices[id]
					flag := ""
					if i < len(u.Equipped) && u.Equipped[i] {
						flag = " [裝備]"
					}
					g.font.Draw(screen, fmt.Sprintf("%s%02Xh%s  +%d G", pre, id, flag, price*3/4), 156, 100+float64(i)*28, 1.0, c)
				}
				break
			}
			h := 76 + float64(len(ids))*30
			fillBox(140, 60, 360, h)
			g.font.Draw(screen, "賣出：選擇角色（Tab 返回購買）", 156, 70, 1.0, color.RGBA{0xff, 0xe0, 0x90, 0xff})
			for i, id := range ids {
				u := g.partyRoster[id]
				pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
				if i == g.shopSellUnitSel {
					pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
				}
				g.font.Draw(screen, fmt.Sprintf("%s%s（%d 件）", pre, u.Name, len(u.Inventory)), 156, 100+float64(i)*30, 1.0, c)
			}
			break
		}
		if g.shopPicking {
			h := 76 + float64(len(g.shopRecipients))*30
			fillBox(140, 60, 360, h)
			g.font.Draw(screen, fmt.Sprintf("選擇收件者：%s", g.shopPending.Name), 156, 70, 1.0, color.RGBA{0xff, 0xe0, 0x90, 0xff})
			for i, id := range g.shopRecipients {
				u := g.partyRoster[id]
				pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
				if i == g.shopRecipientSel {
					pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
				}
				g.font.Draw(screen, fmt.Sprintf("%s%s  (欄位%d/8)", pre, u.Name, len(u.Inventory)), 156, 100+float64(i)*30, 1.0, c)
			}
			break
		}
		goods := g.camp.ShopGoods()
		h := 76 + float64(len(goods))*30
		fillBox(140, 60, 360, h)
		g.font.Draw(screen, fmt.Sprintf("商店　持有 %d G(Enter 購買/Tab 賣出/ESC 離開)", g.gold), 156, 70, 1.0,
			color.RGBA{0xff, 0xe0, 0x90, 0xff})
		for i, gd := range goods {
			c := color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			pre := "　"
			if i == g.shopSel {
				c = color.RGBA{0xff, 0xff, 0xff, 0xff}
				pre = "▶"
			}
			if g.gold < gd.Price {
				c = color.RGBA{0x80, 0x80, 0x90, 0xff}
			}
			g.font.Draw(screen, fmt.Sprintf("%s%s  %d G", pre, gd.Name, gd.Price), 156, 100+float64(i)*30, 1.0, c)
		}
	case n.Type == "preparation":
		h := 118 + float64((len(g.prepIDs)+1)/2)*24
		if h < 170 {
			h = 170
		}
		fillBox(64, 42, 512, h)
		g.font.Draw(screen, "出戰整備", 84, 56, 1.2, color.RGBA{0xff, 0xe0, 0x90, 0xff})
		g.font.Draw(screen, fmt.Sprintf("出擊 %d/%d（↑↓移動，Enter 選擇）", g.preparationSelected(), g.prepLimit), 84, 82, 1.0, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
		for i, id := range g.prepIDs {
			x := 88.0
			if i%2 == 1 {
				x = 320
			}
			y := 108 + float64(i/2)*24
			prefix := "　"
			c := color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			if i == g.prepSel {
				prefix, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			mark := "□"
			if g.partyDeploy[id] {
				mark = "■"
			}
			name := fmt.Sprintf("角色%d", id)
			if u, ok := g.partyRoster[id]; ok && u.Name != "" {
				name = u.Name
			}
			g.font.Draw(screen, fmt.Sprintf("%s%s %s", prefix, mark, name), x, y, 0.95, c)
		}
		g.font.Draw(screen, "F5 保存戰況", 84, 88+h-24, 0.9, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
	case n.Type == "church":
		if g.churchMode == "menu" {
			fillBox(150, 110, 340, 180)
			g.font.Draw(screen, n.Text, 182, 126, 1.2, color.RGBA{0xff, 0xe0, 0x90, 0xff})
			labels := []string{"服務一（待 callee）", "服務二（待 callee）", "復活", "轉職"}
			for i, label := range labels {
				pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
				if i == g.churchSel {
					pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
				}
				g.font.Draw(screen, pre+label, 188, 158+float64(i)*24, 1.0, c)
			}
			g.font.Draw(screen, "Enter 選擇／ESC 返回城鎮", 188, 266, 0.9, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
		} else {
			listLen := len(g.churchIDs)
			if g.churchMode == "class_target" {
				listLen = len(g.churchBranches)
			}
			h := 120 + float64(listLen)*26
			fillBox(120, 90, 400, h)
			title := "復活"
			if g.churchMode == "class" {
				title = "轉職"
			} else if g.churchMode == "class_target" {
				title = "選擇轉職分支"
			}
			g.font.Draw(screen, title, 150, 108, 1.2, color.RGBA{0xff, 0xe0, 0x90, 0xff})
			if listLen == 0 {
				g.font.Draw(screen, "目前沒有符合條件的角色", 150, 150, 1.0, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
			} else if g.churchMode == "class_target" {
				for i, branch := range g.churchBranches {
					pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
					if i == g.churchSel {
						pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
					}
					label := "基本轉職"
					if branch.Branch == "optional" {
						label = fmt.Sprintf("道具 %02Xh", branch.RequiredItemID)
					} else if branch.Branch == "special" {
						label = fmt.Sprintf("特殊道具 %02Xh", branch.RequiredItemID)
					}
					g.font.Draw(screen, fmt.Sprintf("%s%s → portrait %02Xh / class %d", pre, label, branch.Portrait, branch.ClassID), 150, 150+float64(i)*26, 0.95, c)
				}
			} else {
				for i, id := range g.churchIDs {
					pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
					if i == g.churchSel {
						pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
					}
					u := g.partyRoster[id]
					g.font.Draw(screen, fmt.Sprintf("%s%s Lv%d", pre, u.Name, u.Lv), 150, 150+float64(i)*26, 1.0, c)
				}
			}
			g.font.Draw(screen, "Enter 執行／ESC 返回服務選單", 150, 108+h-24, 0.9, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
		}
	case n.Type == "ending":
		// Ending 是獨立頁，不可讓上一張 battle map／HUD 從半透明框後露出。
		screen.Fill(color.RGBA{0, 0, 0, 0xff})
		const scale, maxWidth = 1.2, 560.0
		lines := g.font.Wrap(n.Text, scale, maxWidth)
		lineH := g.font.LineHeight(scale)
		panelH := 70 + lineH*float64(len(lines))
		panelY := (float64(logicalH) - panelH) / 2
		fillBox(24, panelY, float64(logicalW)-48, panelH)
		g.font.Draw(screen, "結局", float64(logicalW)/2-g.font.Width("結局", 1.35)/2, panelY+18, 1.35,
			color.RGBA{0xff, 0xff, 0xff, 0xff})
		for i, line := range lines {
			x := float64(logicalW)/2 - g.font.Width(line, scale)/2
			g.font.Draw(screen, line, x, panelY+55+float64(i)*lineH, scale,
				color.RGBA{0xff, 0xe0, 0x90, 0xff})
		}
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
		// 欄位按「陣營」分:我方欄右上、敵方欄左下(atkOwn=false 表敵攻我,資料對調)
		if a.atkOwn {
			g.drawBattlePanel(screen, 342, 8, a.atkName, a.atkLV, a.atkHP, a.atkMax, a.atkMP) // 我方(攻)右上
			g.drawBattlePanel(screen, 0, 308, a.defName, a.defLV, dhp, a.defMax, a.defMP)     // 敵方(守)左下
		} else {
			g.drawBattlePanel(screen, 342, 8, a.defName, a.defLV, dhp, a.defMax, a.defMP)     // 我方(守)右上
			g.drawBattlePanel(screen, 0, 308, a.atkName, a.atkLV, a.atkHP, a.atkMax, a.atkMP) // 敵方(攻)左下
		}
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
		if prog >= impactS && prog < impactE && (prog/2)%2 == 0 {
			img = g.redSilhouette(img) // 命中閃紅:攻方也閃(orig_05_03 攻方紅剪影)
		}
		dx, dy := 141.0, 3.0
		if m := g.figMeta[a.atkFig]; atkFi < len(m) {
			dx, dy = float64(m[atkFi][0]), float64(m[atkFi][1])
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(dx*sc, dy*sc)
		screen.DrawImage(img, op)
	}
	// (4) 命中全螢幕紅閃:orig VGA DAC 整組色盤設紅→整片泛紅(非只 sprite);快速閃
	if prog >= impactS && prog < impactE && (prog/2)%2 == 0 {
		if g.redFlash == nil {
			g.redFlash = ebiten.NewImage(logicalW, logicalH)
			g.redFlash.Fill(color.RGBA{0xff, 0x28, 0x28, 0xff})
		}
		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(0.3) // 半透明紅罩(整片泛紅、不全遮)
		screen.DrawImage(g.redFlash, op)
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
	g.bgmSource = loadSettings().BGMSource // 音源設定(預設 fm=Sound Blaster)
	if v := os.Getenv("FD2_BGM_SOURCE"); v != "" && bgmSourceName[v] != "" {
		g.bgmSource = v // 覆寫(截圖/測試用)
	}
	g.unitLabels = os.Getenv("FD2_UNIT_LABELS") != ""
	g.cutsceneLog = os.Getenv("FD2_CUTSCENE_LOG") != ""
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
	if err := g.loadMap("assets"); err != nil {
		g.loadErr = err.Error()
		return g
	}
	// 載入單位(M1)
	if st, err := battle.Load(assetPath("assets/map0_units.json")); err == nil {
		g.st = st
	} else if g.loadErr == "" {
		g.loadErr = "units: " + err.Error()
	}
	// 載入劇本 + 套用初始(待命 group + on_battle_start 主角隊進場,doc 25/29)
	if g.st != nil {
		if sc, err := battle.LoadScenario(assetPath("assets/scenarios/ch01.json")); err == nil {
			g.sc = sc
			g.dialog = append(g.dialog, sc.Setup(g.st)...)
			g.focusOnParty()
		} else if g.loadErr == "" {
			g.loadErr = "scenario: " + err.Error()
		}
	}
	g.sprites = loadSprites()
	g.portraits = loadPortraits()
	g.figani = loadFIGANI()
	g.figMeta = loadFigMeta()
	if raw, e := os.ReadFile(assetPath("assets/bg/bg.png")); e == nil { // 戰鬥背景(BG.DAT)
		if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
			g.bg = ebiten.NewImageFromImage(im)
		}
	}
	if raw, e := os.ReadFile(assetPath("assets/tai/tai_004.png")); e == nil { // 我方台座(TAI_004 綠草橢圓)
		if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
			g.tai = ebiten.NewImageFromImage(im)
		}
	}
	if raw, e := os.ReadFile(assetPath("assets/ui/panel.png")); e == nil { // 狀態欄框(LMI1 #22)
		if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
			g.panel = ebiten.NewImageFromImage(im)
		}
	}
	if raw, e := os.ReadFile(assetPath("assets/ui/dialog.png")); e == nil { // 對話框框(LMI1 #21)
		if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
			g.dlgBox = ebiten.NewImageFromImage(im)
		}
	}
	for i, nm := range []string{"item", "attack", "status", "wait"} { // 指令環圖示(orig_04 截圖 oracle 裁出)
		if raw, e := os.ReadFile(assetPath("assets/ui/ring_" + nm + ".png")); e == nil {
			if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
				g.ringIcons[i] = ebiten.NewImageFromImage(im)
			}
		}
	}
	if sp, e := battle.LoadSpells(assetPath("assets/spells.json")); e == nil { // 法術表(EXE dump)
		g.spells = sp
	}
	if types, equip, e := campaign.LoadShopEligibility(assetPath("assets/data/item.json"), assetPath("assets/data/class_equip_types.json")); e == nil {
		g.shopItemTypes, g.shopEquipTypes = types, equip
	}
	if prices, e := campaign.LoadItemPrices(assetPath("assets/data/item.json")); e == nil {
		g.shopItemPrices = prices
	}
	if stats, e := campaign.LoadItemStats(assetPath("assets/data/item.json")); e == nil {
		g.shopItemStats = stats
	}
	if rates, e := campaign.LoadReviveFeeRates(assetPath("assets/data/revive_fee_rates.json")); e == nil {
		g.reviveFeeRates = rates
	}
	classTablePath := assetPath("assets/data/class_change_targets.json")
	if _, e := os.Stat(classTablePath); e != nil {
		classTablePath = "docs/data/exe_tables/class_change_targets.json"
	}
	if table, e := campaign.LoadClassChangeTable(classTablePath); e == nil {
		g.classChangeTable = table
	}
	growthPath := assetPath("assets/data/class_change_growth.json")
	if _, e := os.Stat(growthPath); e != nil {
		growthPath = "docs/data/exe_tables/growth.json"
	}
	if growth, e := campaign.LoadClassChangeGrowth(growthPath); e == nil {
		g.classChangeGrowth = growth
	}
	g.initializeEquipmentBases(g.st)
	g.font = loadFont()
	// 狀態欄名字專用整數尺寸 face(scale 1.0 繪製,避免非整數縮放模糊);orig 名墨高 13px→face 28
	g.fontNm = loadFontSized(28)
	for k := 0; k < 10; k++ { // 原版數字 cell(LMI1 #31-40)
		if raw, e := os.ReadFile(assetPath(fmt.Sprintf("assets/ui/digit_%d.png", k))); e == nil {
			if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
				g.digits[k] = ebiten.NewImageFromImage(im)
			}
		}
	}
	g.gold = 1000 // 初始金幣(商店用;原版開局金額待對照)
	seed := time.Now().UnixNano()
	if v, e := strconv.ParseInt(os.Getenv("FD2_SEED"), 10, 64); e == nil {
		seed = v
	}
	g.rng = rand.New(rand.NewSource(seed))
	g.sfx = loadSFX()
	// 戰鬥音效:揮擊/命中/陣亡三段(真素材;attack_id→池 精確對照 doc36 未 RE,故命中/陣亡池為近似選擇)
	g.sfxSwing = loadWav("assets/sfx/battle_48_00.wav")  // 揮擊(池 sub0,七池共用)
	g.sfxImpact = loadWav("assets/sfx/battle_64_00.wav") // 命中(最短最尖池)
	g.sfxDeath = loadWav("assets/sfx/battle_88_00.wav")  // 陣亡/重擊(最長池)
	// 戰場 BGM:doc12 推定 track18=戰鬥被使用者實聽推翻(18=商店音樂);戰鬥曲號待聽辨,先不播錯曲
	if os.Getenv("FD2_TITLE") == "1" || (g.shotPath == "" && os.Getenv("FD2_TITLE") != "0") { // 開頭動畫+主選單(headless 截圖預設跳過)
		if ta := loadTitleAssets(); ta != nil {
			g.titleAssets = ta
			if ta.aniPath != "" && os.Getenv("FD2_NOCUT") == "" {
				g.titlePhase = "cutscene" // 有 ANI.DAT:播完整 AFM 開場過場
			} else {
				g.titlePhase = "scroll" // 無 ANI.DAT:退回 FDOTHER 立繪捲動+logozoom
				g.scrollY = 535
			}
		}
	}
	if cp := os.Getenv("FD2_CAMPAIGN"); cp != "" { // 劇本節點圖模式(doc 19;放最後,story 對白不被開場 Setup 蓋掉)
		if cp == "1" {
			cp = "assets/scenarios/campaign.json"
		}
		if c, err := campaign.Load(assetPath(cp)); err == nil {
			g.camp = campaign.NewRunner(c)
			if os.Getenv("FD2_CAMP_CLASS_FIXTURE") != "" {
				// Bounded headless oracle only: construct one native-eligible
				// Lv20+ roster record so xvfb can exercise the church target UI.
				g.partyMembers = map[int]bool{0: true}
				g.partyJoinOrder = []int{0}
				g.partyRoster = map[int]battle.Unit{0: {
					Camp: battle.Own, Name: "索爾", ClsName: "法師", ClassID: 5,
					Lv: 20, HP: 80, MaxHP: 80, MP: 20, MaxMP: 20,
					AP: 30, DP: 20, DX: 10, MV: 5, HIT: 10, EV: 10,
					Portrait: 9, Fig: 0, OnField: true,
					Inventory: []int{0x58, 0x5a}, Equipped: []bool{true, false},
					InventorySlots: []int{0x58, 0x5a, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
				}}
			}
			// Headless post-handler oracle: materialize a named battle node before
			// jumping to its following cutscene. Normal campaign play never sets this
			// variable; it exists so screenshots can exercise the real canonical
			// battle slots instead of bypassing the required runtime context.
			if prep := os.Getenv("FD2_CAMP_PREP_BATTLE"); prep != "" {
				if battleNode, ok := c.Nodes[prep]; ok && battleNode.Type == "battle" {
					if battleNode.Map != "" {
						if err := g.loadMap(battleNode.Map); err != nil {
							g.loadErr = "prep map: " + err.Error()
						}
					}
					g.resetBattle(battleNode.Units, battleNode.Scenario)
					if turn, err := strconv.Atoi(os.Getenv("FD2_CAMP_PREP_TURN")); err == nil && turn > 0 && g.st != nil && g.sc != nil {
						g.st.Turn = turn
						g.sc.Fire(g.st, "on_turn_end", "")
						g.applyScenarioPartyJoins()
					}
					if group, err := strconv.Atoi(os.Getenv("FD2_CAMP_PREP_DEAD_GROUP")); err == nil && g.st != nil {
						for _, unit := range g.st.Units {
							if unit.Group == group {
								unit.HP = 0
							}
						}
					}
				}
			}
			if n := os.Getenv("FD2_CAMP_NODE"); n != "" { // 驗證鉤子:直接跳指定節點
				if _, ok := c.Nodes[n]; ok {
					g.camp.Cur = n
					g.camp.Flags["found_secret"] = os.Getenv("FD2_CAMP_SECRET") != ""
				}
			}
			g.enterNode()
		} else {
			g.loadErr = "campaign: " + err.Error()
		}
	}
	return g
}

// endTurn 結束當前回合:觸發 on_turn_end 事件(增援等),回合 +1,清除已行動。
// 回合無上限(doc 27);只由劇本事件決定勝負。
// showBanner 觸發回合橫幅(~90 tick=1.5s;截圖模式不顯以免擋驗證畫面)。
func (g *Game) showBanner(s string) {
	if g.shotPath != "" {
		return
	}
	g.banner, g.bannerT = s, 90
}

// drawPhaseBanner 回合橫幅:暗化地圖 + 中央金字(對照原版 orig_08 PLAYER PHASE)。
func (g *Game) drawPhaseBanner(screen *ebiten.Image) {
	if g.bannerT <= 0 || g.banner == "" || g.font == nil {
		return
	}
	a := 1.0
	if g.bannerT < 20 { // 末段淡出
		a = float64(g.bannerT) / 20
	}
	if g.dim == nil {
		g.dim = ebiten.NewImage(logicalW, logicalH)
		g.dim.Fill(color.RGBA{0, 0, 0, 0xff})
	}
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(0.45 * a)) // 暗化地圖
	screen.DrawImage(g.dim, op)
	w := g.font.Width(g.banner, 2.2)
	c := color.RGBA{uint8(0xff * a), uint8(0xc8 * a), uint8(0x50 * a), uint8(0xff * a)} // 金字(ColorScale 已預乘)
	g.font.Draw(screen, g.banner, (float64(logicalW)-w)/2, float64(logicalH)/2-24, 2.2, c)
}

// drawUnitHUD 左下單位資訊面板。改沿用 drawBattlePanel 同一份原版框素材(FDOTHER#5
// LMI1 #22,doc35 §4)而非自訂半透明黑框,地圖上與全螢幕戰鬥的單位欄才是同一套視覺
// (playfix #6:對照 orig_04/orig_07,原版狀態欄一律走這個框,remake 先前地圖版沒沿用)。
// AP/DP/MV 素材未涵蓋(框只留 HP/MP 槽),另加一行文字補在框下方。
func (g *Game) drawUnitHUD(screen *ebiten.Image, u *battle.Unit) {
	nm := u.Name
	if nm == "" {
		nm = u.ClsName
	}
	const bh = 84.0 // 149×42 原生 ×2
	bx, by := 6.0, float64(logicalH)-bh-6-20
	g.drawBattlePanel(screen, bx, by, nm, u.Lv, u.HP, u.MaxHP, u.MP)
	g.font.Draw(screen, fmt.Sprintf("AP %d  DP %d  MV %d", u.AP, u.DP, u.MV), bx+8, by+bh+2, 0.9, color.RGBA{0xc8, 0xe0, 0xff, 0xff})
}

func (g *Game) endTurn() {
	if g.st == nil || g.result != "" || g.aiBusy {
		return
	}
	if g.shotPath == "" || os.Getenv("FD2_SHOT_AI") != "" { // 截圖模式預設跳 AI;FD2_SHOT_AI=1 強制驗證 AI 行走
		g.aiBusy = true // AI 階段:逐單位行走動畫(Update 內 aiStep 驅動),播完 finishTurn
		g.showBanner("ENEMY PHASE")
		return
	}
	g.finishTurn()
}

// startBattleEvent runs ordered scenario actions without borrowing campaign's
// BeatRunner. Immediate state actions execute in-order; pan/delay/dialogue block
// until their visual/input boundary finishes, then resume at the next action.
func (g *Game) startBattleEvent(actions []battle.Action, then func()) {
	if len(actions) == 0 {
		then()
		return
	}
	g.battleEvent = &battleEventRun{actions: actions, index: -1, then: then}
	g.advanceBattleEvent()
}

func (g *Game) finishBattleEventWithError(message string) {
	g.loadErr = "battle event: " + message
	run := g.battleEvent
	g.battleEvent, g.battleEventDelay, g.camPan = nil, 0, nil
	if run != nil && run.then != nil {
		run.then()
	}
}

func (g *Game) advanceBattleEvent() {
	run := g.battleEvent
	if run == nil {
		return
	}
	for {
		run.index++
		if run.index >= len(run.actions) {
			g.battleEvent = nil
			if run.then != nil {
				run.then()
			}
			return
		}
		action := run.actions[run.index]
		switch action.Type {
		case "pan":
			if action.Grid == nil || g.m == nil || g.m.TileW <= 0 || g.m.TileH <= 0 {
				g.finishBattleEventWithError("pan 缺少有效 grid/map")
				return
			}
			g.camPan = &camPanJob{
				fromX: g.camX, fromY: g.camY,
				toX:      float64((*action.Grid)[0] * g.m.TileW),
				toY:      float64((*action.Grid)[1] * g.m.TileH),
				tileStep: true, then: g.advanceBattleEvent,
			}
			return
		case "delay":
			frames := action.Ms * 60 / 1000
			if frames <= 0 {
				frames = 1
			}
			g.battleEventDelay = frames
			return
		default:
			dialogue, isDialogue := g.sc.ExecuteAction(g.st, action)
			g.applyScenarioPartyJoins()
			if isDialogue {
				g.dialog = []battle.DialogLine{dialogue}
				g.dlgPage = 0
				return
			}
		}
	}
}

func (g *Game) stepBattleEventDelay() {
	if g.battleEventDelay <= 0 {
		return
	}
	g.battleEventDelay--
	if g.battleEventDelay == 0 {
		g.advanceBattleEvent()
	}
}

// finishTurn starts the ordered on_turn_end event. Turn/status bookkeeping is
// deferred until the complete visual sequence finishes.
func (g *Game) finishTurn() {
	if g.battleEvent != nil {
		return
	}
	if g.sc != nil {
		actions := g.sc.TriggerActions(g.st, "on_turn_end", "")
		if len(actions) > 0 {
			g.startBattleEvent(actions, g.completeTurn)
			return
		}
	}
	g.completeTurn()
}

func (g *Game) completeTurn() {
	g.st.Turn++
	for _, u := range g.st.Units {
		u.Acted = false
		u.TickStatus()             // buff/封咒/中毒/麻痺回合遞減+中毒扣血(doc02 §6.4)
		g.awardDeathReward(u, nil) // poison/status death shares the same once-only reward path
	}
	if g.result == "" {
		g.showBanner("PLAYER PHASE")
	}
	g.sel, g.reach, g.moved = nil, nil, false
	g.checkResult()
}

// aiStep AI 回合驅動:一次取一個單位的行動計畫,播行走動畫→到位攻擊(全螢幕演出)。
// 全單位動完 → finishTurn。
func (g *Game) aiStep() {
	if !g.aiBusy || g.walk != nil || g.atk != nil || g.result != "" {
		if g.result != "" {
			g.aiBusy = false
		}
		return
	}
	plan := g.st.NextAIPlan()
	if plan == nil {
		g.aiBusy = false
		g.finishTurn()
		return
	}
	u := plan.U
	act := func() {
		if plan.Target != nil && plan.Target.Alive() {
			tgt := plan.Target
			u.Dir = dirToward(u.X, u.Y, tgt.X, tgt.Y)
			nm, anm := tgt.Name, u.Name
			if nm == "" {
				nm = tgt.ClsName
			}
			if anm == "" {
				anm = u.ClsName
			}
			hp0 := tgt.HP
			dmg := g.st.Attack(u, tgt)
			g.awardDeathReward(tgt, u)
			g.msg = fmt.Sprintf("%s 攻擊 %s,造成 %d 傷害", anm, nm, dmg)
			g.atk = g.newAtkAnim(u.Fig, tgt.Fig, anm, nm,
				u.HP, u.MaxHP, u.Lv, u.MP, tgt.Lv, tgt.MP,
				hp0, tgt.HP, tgt.MaxHP, g.terrainAt(tgt.X, tgt.Y), u.Camp == battle.Own)
			g.checkResult()
		}
		u.Acted = true
	}
	if len(plan.Path) >= 2 {
		if os.Getenv("FD2_SHOT_AI") != "" {
			log.Printf("AI walk: %s(%d,%d)→(%d,%d) 段數%d 目標=%v", u.ClsName, plan.Path[0].X, plan.Path[0].Y,
				plan.Path[len(plan.Path)-1].X, plan.Path[len(plan.Path)-1].Y, len(plan.Path)-1, plan.Target != nil)
		}
		g.walk = &walkAnim{u: u, path: plan.Path, then: act}
	} else {
		act()
	}
}

func main() {
	ebiten.SetWindowSize(logicalW*2, logicalH*2)
	ebiten.SetWindowTitle("炎龍騎士團2 重製 (fd2_re)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(loadGame()); err != nil {
		log.Fatal(err)
	}
}
