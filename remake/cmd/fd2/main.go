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
	"errors"
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
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
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
	// Native fields are optional: legacy PNG-only maps must not be treated as
	// verified indexed input. They mirror export_engine_assets.py's raw FDFIELD
	// event-high bytes and FDSHAP four-byte control table.
	NativeTileBlitModes  []byte `json:"native_tile_blit_modes,omitempty"`
	NativeTerrainControl []byte `json:"native_terrain_control,omitempty"`
}

type Game struct {
	m                          *MapData
	nativeMapAssets            *nativeMapAssets                   // all original map HUD resources, nil on any missing/malformed asset
	nativeMapWork              []byte                             // persistent 456-stride original tactical framebuffer
	nativeMapVGA               []byte                             // persistent 320x200 indexed VGA surface
	nativeMapDAC               []byte                             // current 256xRGB six-bit DAC state for handler palette ramps
	nativePaletteRamp          *nativePaletteRampJob              // exact 0x1f882/0x1f525 indexed DAC presentation
	nativePalettePulse         *nativePalettePulseJob             // exact 0x35E5A 0..63/hold/62..0 indexed DAC presentation
	nativeCh20SkyKey           *nativeCh20SkyKeyJob               // raw ch20 post 0x24336 fixed FDOTHER/ANI/palette sequence
	nativeCh23State            *nativeCh23AdapterState            // raw ch23 staging/latch/timer state shared across both handler loops
	nativeCh23Loop             *nativeCh23LoopJob                 // blocking raw ch23 indexed presentation loop
	native2189A                *native2189AJob                    // blocking raw ch22 post 0x2189A ten-pass presentation
	nativeUnitPresent          *nativeUnitPresentJob              // blocking shared 0x22253 11+6+bridge+10 indexed presentation
	nativeCh28PostPresent      *nativeCh28PostPresentJob          // blocking 0x1DB65 13+6+6 indexed presentation
	nativeCh22Reload           *nativeCh22ReloadState             // atomic FDFIELD69/FDSHAP46/47/FDOTHER42 tail transaction
	nativeFDOTHERPalettePhase  int                                // process-lifetime 0x4DFCC phase projection (0..15)
	nativeFullDACWhite         bool                               // exact 0x11DF2(0,255,255) overlay for legacy RGB scenes
	nativeFullDACBlack         bool                               // exact ch07 post 0x11D40(0,255,64)+mode-13h clear
	nativeMapHUDPersistent     battle.NativeMapHUDPersistentState // gate A save-persistent；anchor process-persistent
	tileset                    *ebiten.Image
	tiles                      []*ebiten.Image     // 切好的圖塊
	st                         *battle.State       // 戰鬥狀態(單位)
	nativeMapClock             nativeBIOSClock     // battle-local 18.2065Hz BIOS low-word adapter
	sc                         *battle.Scenario    // 劇本(事件系統,doc 29)
	dialog                     []battle.DialogLine // 待顯示對話(事件產生,含說話者)
	storyBG                    bool                // 場景背景模式(story 節點指定 Map):鏡頭固定不跟游標,不畫單位/游標/HUD(doc23 §4)
	storyActors                []battle.Unit       // 原版目前已 materialize 的 scene unit array；index 只在該 load/spawn 時序內有意義
	storyRoster                []battle.Unit       // LOADCH 保留的 FDFIELD records；SPAWN 按 group 順序 append 到 storyActors
	storyCompositionEventBytes []byte              // LOADCH 的 immutable FDFIELD composition +2；future-group placement 的原始輸入
	storySpawned               map[int]bool        // 原版 group 已 materialize；防止 handler 重複 SPAWN 時重複 append
	storyRosterPath            string              // 最近一次 handler LOADCH 的 exact roster source；battle handoff gate
	storyPartyScenario         string              // 最近一次 handler LOADCH 的 exact party scenario；battle handoff gate
	partyMembers               map[int]bool        // JOIN 建立的永久玩家名冊；key=原版 0..31 charID，不使用 NPC portrait
	partyJoinOrder             []int               // JOIN 首次出現順序；章0 cutscene 的 party runtime slot 以此為準
	partyRoster                map[int]battle.Unit // 0x11506 戰後同步的跨關角色能力／HP／MP／經驗快照
	partyDeploy                map[int]bool        // preparation 0x318ad 的本戰出擊勾選；不改永久 JOIN 名冊
	prepIDs                    []int               // preparation UI 角色順序（JOIN chronology）
	prepSel                    int                 // preparation UI 游標
	prepLimit                  int                 // preparation UI 原版出擊上限（15，末段 19）
	prepSelecting              bool                // 已通過前置確認，且流程要求進入原版選人階段
	prepConfirm                bool                // 選滿或小隊確認後的最終出戰確認階段
	prepConfirmSel             int                 // 0=肯定，1=取消
	prepClock                  nativeBIOSClock     // preparation 0x31e80→0x1297d 的 BIOS 低字來源
	prepIdleCycle              int                 // 原版 [0x53c0b] 0..3；繪圖時 3 正規化為1
	prepLastTick               int                 // 原版 [0x53c0f] 有號 BIOS 低字 latch
	prepPromptSource           []byte              // 0x1956b 前的 town 畫面或 0x2cc04 黑色來源
	churchSel                  int                 // church service menu cursor (0..3)
	churchMode                 string              // menu / status_* / transfer_* / revive* / class / class_confirm
	churchIDs                  []int               // current church candidate ids
	churchRosterStart          int                 // 0x2e6b8 [0x5412f], even six-entry viewport origin
	churchVerticalStart        int                 // 0x30c22/0x311dc three-row viewport origin
	churchStatusID             int                 // selected actor passed to 0x17aed
	churchStatusPanel          []byte              // 0x17eef/0x17fc0 + 0x184c0(actor,-1)
	churchCommandPanel         []byte              // 0x17eef/0x17fc0 + 0x1ceed(actor,-1)
	churchItemStart            int                 // 0x2df6b even six-entry item viewport origin
	churchTransferSource       int                 // raw transfer source roster id
	churchTransferItem         int                 // compact source inventory index
	churchTransferItems        []int               // compact source inventory indices
	churchTransferDest         int                 // raw destination id used by FDTXT506 FFFC
	churchReviveID             int                 // selected 0x30dc3 candidate
	churchReviveFee            int                 // level * raw class fee
	churchClassID              int                 // selected class-change candidate
	churchBranches             []campaign.ClassChangeBranch
	hotelSel                   int // raw 0x2fc85 selector (0..3)
	hotelRoute                 fdother.NativeHotelServiceRoute
	hotelHasRoute              bool
	titleSlotSel               int // title LOAD selector: native 0x30550 slots 0..3
	classChangeTable           campaign.ClassChangeTable
	classChangeGrowth          map[int]campaign.ClassChangeGrowth
	nativeJoinConstructor      campaign.NativeJoinConstructorTable
	hasNativeJoinConstructor   bool
	nativeJoinBases            campaign.NativeJoinBaseTable
	hasNativeJoinBases         bool
	nativeJoinItemEffectRows   []byte
	handlerChapter             int             // 原版 [0x53c03]；set_chapter 與無立即數 LOADCH 的 resource chapter
	storyWalks                 []*storyWalkJob // 場景走位動畫佇列(doc46 §5.3);逐幀推進、完成後移除
	storyAutoAdvance           int             // story 節點無對白時的自動轉場倒數幀(doc46 行軍蒙太奇,0=不自動)
	storyView                  *ebiten.Image   // story 場景離屏世界層(320×200,放大 storyZoom 倍貼上畫布;2-1 原版取景)
	walkFirst                  bool            // 本節點:進場走位走完才顯示對白(campaign.Node.WalkFirst)
	followWalk                 bool            // 本節點:走位期間鏡頭跟隨走位者(campaign.Node.FollowWalk;beat walk 依 Follow 逐拍設值)
	camMaxY                    float64         // 本節點:鏡頭 Y 上限(campaign.Node.CamMaxY;0=不限)
	camPan                     *camPanJob      // beat「pan」進行中(doc50 §1);storyBG 專用,與 followWalk 互斥
	focusJob                   *focusUnitJob   // beat「focus_unit」：依原版 0x12cea 先 X 後 Y 逐格移動游標／鏡頭
	actJob                     *actPoseJob     // beat「act」進行中(近似姿態循環,見 actPoseJob 註解)
	beats                      []campaign.Beat // 目前 cutscene 節點的過場原語序列(doc50 §2)
	beatIdx                    int             // 目前執行到第幾拍(-1=尚未開始)
	beatDelay                  int             // beat「delay」剩餘幀數(0=非等待中)
	battleEvent                *battleEventRun // 戰場事件的阻塞 action 序列；與 campaign BeatRunner 分離
	battleEventDelay           int             // battle event delay 剩餘幀數
	campLines                  []campaign.Line // cutscene 節點載入的章文本(dialog beat 依 Line/Count 取子段)
	dlgShown                   int             // 對話框目前顯示的說話者(dlgNone=無;換人時播縮/展動畫)
	dlgUpper                   *bool           // 與 dlgShown 同步的上/下框覆蓋(來自 DialogLine.Upper;nil=沿用預設規則)
	dlgPhase                   int             // 對話框動畫相位:0=常態 1=縮小(換人前收合) 2=展開
	dlgT                       int             // 對話框動畫相位內計時(幀)
	dlgPage                    int             // 目前對白的頁碼(0起);一句>3行時分頁,Enter 先翻頁翻完才換句(使用者回饋 2026-07-05)
	dlgScrollT                 int             // 分頁捲動剩餘幀數(0=靜止)
	dlgScrollFrom              int             // 分頁捲動開始頁碼
	fade                       *storyFade      // 場景淡出/淡入轉場(doc46 §5.2)
	transitionReveal           *transitionRevealJob
	indexedTransition          *nativeIndexedTransitionJob
	nativeHealPresentation     *nativeCommandHealPresentationJob
	spawnIntroTransition       *nativeSpawnIntroJob
	nativeTurnStaging          *nativeTurnStagingJob
	nativeFieldEvent61         *nativeFieldEvent61Job
	nativeAIIdleRecovery       *nativeAIIdleRecoveryJob // direct 0x13FD4 indexed/audio owner
	nativeEnding               *nativeEndingPreview     // FD2_ENDING_PREFIX 或來源約束 campaign ending；缺原始資料時走明示 fallback
	endingNotice               string                   // 原始素材不足或來源約束終局無法發布時的玩家提示
	walk                       *walkAnim                // 移動動畫(沿路徑逐格走,FDICON 方向幀)
	camp                       *campaign.Runner         // 劇本節點圖(doc 19;FD2_CAMPAIGN 啟用)
	campSel                    int                      // choice 節點游標
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
	// radial 指令環(原版 [0x3C57]:↑0=攻擊/←1=法術/→2=物品/↓3=待機)
	ring                     bool
	ringSel                  int
	actionOverlayPhase       string
	actionOverlayFrame       int
	actionOverlayAfter       func()
	actionOverlayDrawn       bool
	actionOverlayShotHold    bool
	ringIcons                [4]*ebiten.Image // fallback only: 0上=攻擊 1左=法術 2右=物品 3下=待機
	nativeActionCells        []*ebiten.Image  // FDOTHER#2 的完整 78 格；只取自使用者提供的原始資料
	nativeUIPalette          color.Palette
	nativeClassUI            *nativeClassUIAssets
	nativeLoadSlotsUI        *nativeLoadSlotsUIAssets
	nativeClassUIJob         *nativeClassUIJob
	nativeClassUIClock       nativeBIOSClock
	nativeClassUIPulse       int
	nativeClassUILastTick    int
	nativeClassUIHasTick     bool
	nativeTownUI             *nativeTownUIAssets
	nativeTownUIClock        nativeBIOSClock
	nativeTownUIPulse        int
	nativeTownUILastTick     int
	nativeTownUIHasTick      bool
	nativePreparationUI      *nativePreparationUIAssets
	nativeChurchUIJob        *nativeChurchUIJob
	nativeChurchUIClock      nativeBIOSClock
	nativeChurchUIPulse      int
	nativeChurchUILastTick   int
	nativeChurchUIHasTick    bool
	nativeChurchTextIndex    int
	nativeShopUI             *nativeShopUIAssets
	nativeShopUIJob          *nativeClassUIJob
	nativeShopUIClock        nativeBIOSClock
	nativeShopUIPulse        int
	nativeShopUILastTick     int
	nativeShopUIHasTick      bool
	nativeShopVariant        int
	nativeShopMode           string
	nativeShopServiceSel     int
	nativeShopItemStart      int
	nativeShopConfirmSel     int
	nativeShopRecipientStart int
	nativeShopRecipientCycle int
	nativeShopEquipSel       int
	nativeShopPendingUnit    battle.Unit
	nativeShopHasPendingUnit bool
	nativeShopPendingGold    int
	nativeShopSellRosterTop  int
	nativeShopSellItemTop    int
	nativeShopSellConfirmSel int
	nativeShopSellItemIDs    []int
	nativeShopEquipRosterTop int
	nativeShopEquipUnitSel   int
	nativeShopTransferSource int
	nativeShopTransferItem   int
	nativeShopTransferItems  []int
	nativeShopTransferDest   int
	nativeShopTransferIDs    []int
	nativeShopTransferSel    int
	nativeShopTransferTop    int

	nativeShopSellRosterCycle int
	nativeCommandLabels       map[int]string
	nativeCommandOpen         bool
	nativeCommandSel          int
	nativeCommand0Targeting   bool
	nativeCommandTargetID     int
	// nativeContinueOpeningConfirm 只由已驗證的原版 FD2.SAV current-runtime
	// 發布點設一次。它讓該 E2 錨點的第一個 Return 直接開 action overlay；
	// 一經消費，後續空游標確認由已證實的共用 0x117E7 owner 處理。
	nativeContinueOpeningConfirm bool
	// nativeSystemCursorOverlay 對應共用 0x117E7 在 0x12C0D 回傳 -1 時
	// 呼叫的 0x16F55 空游標面板。只有 direction3／END 已有 action owner；
	// 其餘三格維持失敗即關閉。
	nativeSystemCursorOverlay bool
	spellOpen                 bool
	spellSel                  int
	itemOpen                  bool // native 0x1b932 eight-slot selector; unsupported effect presentations remain fail-closed
	itemSel                   int
	itemAnimStep              int
	itemClosing               bool
	nativeItemTargeting       bool
	nativeItemTargetID        int
	nativeItemTargetRawSlot   int
	nativeItemRelocating      bool
	nativeItemRelocationUnit  int
	nativeMovementCostRows    [][]byte
	nativeRNGState            uint16 // original 0x627b8: initialized to zero, process-lifetime only
	nativeItemPanel           *ebiten.Image
	nativeItemPanelBase       []byte
	nativeItemPanelRecord     []byte
	nativeItemPanelAssets     *battle.NativeItemPanelDataAssets
	nativeItemEffectRows      []byte
	castSp                    *battle.Spell // 施法目標選擇中
	spells                    []battle.Spell
	nativeCommandBook         []battle.NativeCommandRecord
	nativeCommandResistances  map[int]int
	commandLearn              map[int][]battle.CommandLearnEntry // native portrait-indexed level-up command table
	bgm                       *audio.Player                      // BGM(doc12 play_bgm 語意:同曲不重播)
	bgmCur                    string
	bgmSource                 string                // 音源設定 "fm"/"mt32"(settings.go;F2 切換)
	debug                     bool                  // F3:開發除錯 HUD(座標/陣營原文等)
	approximateMode           bool                  // FD2_APPROXIMATE=1:可玩近似模式；不宣稱原版 handler 等價
	approximatePostbattle     bool                  // 未綁定戰後節點的近似整理提示，等待玩家確認後才進城鎮／整備
	unitLabels                bool                  // FD2_UNIT_LABELS=1:cutscene sprite 左上標 [idx]fig+名+座標(協助回報/對映原版 slot)
	cutsceneLog               bool                  // FD2_CUTSCENE_LOG=1:過場 node/beat/走位逐步 log 到 stderr(協助對原版資料比對)
	banner                    string                // 回合橫幅文字(PLAYER/ENEMY PHASE)
	bannerT                   int                   // 橫幅剩餘 tick
	sfx                       map[int][]byte        // SFX PCM(doc36 FDOTHER#31 14樣本)
	sfxSwing                  []byte                // 戰鬥揮擊音(doc36 戰鬥池 #48-64 sub0,七池共用)
	sfxImpact                 []byte                // 命中音(近似:最短最尖池;attack_id→sfx 對照表 doc36 未 RE)
	sfxDeath                  []byte                // 陣亡/重擊音(近似:最長池)
	sfxTransition             []byte                // FDOTHER #88 sub1: ch24 transition SFX
	sfxSpawnIntro             []byte                // FDOTHER #95 sub0: 0x32999 pass1 raw sample（11025Hz 為既有工具鏈推論）
	handlerResource           int                   // currently loaded handler resource-table id
	prevCurX, prevCurY        int                   // 游標移動音偵測
	aiBusy                    bool                  // AI 回合進行中(逐單位行走動畫)
	deathRewarded             map[*battle.Unit]bool // 每個死亡 transition 的 reward 只執行一次
	rng                       *rand.Rand            // 施法擲骰(FD2_SEED 可固定,headless 重現)
	gold                      int                   // 金幣(商店)
	items                     []string              // 隊伍道具(名稱;道具效果待實裝)
	shopSel                   int                   // 商店游標
	shopRecipientSel          int
	shopRecipients            []int
	shopPicking               bool
	shopPending               campaign.Good
	shopEquipPrompt           bool
	shopEquipUnit             int
	shopEquipSlot             int
	shopItemTypes             map[int]int
	shopEquipTypes            map[int][]int
	shopItemPrices            map[int]int
	shopItemStats             map[int]campaign.ItemStats
	reviveFeeRates            []int  // church 0x30dc3 class fee words
	shopMode                  string // buy or sell
	shopSellPicking           bool
	shopSellUnitSel           int
	shopSellSlotSel           int
	portraits                 map[int][]*ebiten.Image // DATO 頭像:肖像 id → 4 嘴型幀
	mouthOpen                 bool                    // 嘴型動畫狀態(原版 0x16d00:m0閉/m3開)
	mouthTimer                int                     // 閉嘴倒數(原版 rand%30+2 tick)
	mouthState                dato.MouthState         // native 0x16d00 cadence adapter
	curX                      int
	curY                      int
	camX                      float64
	camY                      float64
	loadErr                   string

	// nativeSystemEndTurnConfirm 承接共用 0x16F55 的 END 確認生命週期；
	// 其餘三個 cell 沒有已證實 owner，仍失敗即關閉。
	nativeSystemEndTurnConfirm bool
	nativeSystemEndTurnDelay   int
	nativeSystemEndTurnUI      *nativeSystemEndTurnUIState

	// 截圖鉤子(FD2_SHOT=path 啟用):第 shotFrame 幀存 PNG 後自動退出(有界,供無人值守驗證)
	frame      int
	shotPath   string
	shotFrame  int
	shotSeries string // 逐幀截圖目錄(FD2_SHOT_SERIES):戰鬥演出每幀存 frame_NN.png,演出結束自動退出
	shotTurn   int    // 截圖前自動推進到第 N 回合(FD2_SHOT_TURN,驗證增援進場)
	shotCurX   int    // 截圖時把游標放這(FD2_SHOT_CUR=x,y)
	shotCurY   int
	shotSel    bool // 截圖前自動選取游標單位(FD2_SHOT_SELECT=1)
	shotSetup  bool // screenshot setup also must tolerate skipped exact frames
	shotTaken  bool // frame scheduling may skip an exact number; capture once at-or-after it
	// 選取狀態
	sel                *battle.Unit
	reach              map[battle.Cell]bool
	selOrigX, selOrigY int    // 選取單位當下的原始格(ESC 取消移動時退回,playfix #4)
	moved              bool   // 已選單位是否移動完(進入攻擊階段)
	result             string // 勝負:""/win/lose
	msg                string // 短訊息(攻擊傷害等)
	// 地圖單位 sprite(FDICON 待機分鏡):fig index → 幀序列
	sprites            map[int][]*ebiten.Image
	figani             map[int][]*ebiten.Image         // 攻擊全身動畫(FIGANI):fig → 幀序列
	figaniDelays       map[int][]int                   // 原始 FIGANI descriptor +6 delay，與 PNG 幀數一一對齊
	atk                *atkAnim                        // 進行中的攻擊演出
	bg                 *ebiten.Image                   // 戰鬥背景(BG.DAT,by 戰場;map0=BG_004 森林)
	tai                *ebiten.Image                   // 我方腳下台座(TAI.DAT;0x29164 載 0x28c46,doc35 §3.3)
	panel              *ebiten.Image                   // 狀態欄框素材(FDOTHER#5 LMI1 #22,149×42;含bevel+HP/MP標籤+槽,doc35 §4)
	dlgBox             *ebiten.Image                   // 對話框框素材(FDOTHER#5 LMI1 #21,310×99;orig 下框(5,112)@320)
	dlgGrad            *ebiten.Image                   // 對話框內部漸層(比對頭像底色 40,69,138→56,85,154 消接縫色差;lazy 建)
	fontNm             *Font                           // 狀態欄名字(整數尺寸 face,scale1 銳利)
	nativeBattleFont   *fdtxt.Font                     // 全螢幕戰鬥狀態欄 FDOTHER#4 16×16 字模
	nativeBattleGlyphs map[string]int                  // Unicode→原版 glyph 索引（未知字元失敗即關閉）
	digits             [10]*ebiten.Image               // 狀態欄數字 0-9(LMI1 #31-40 原版 digit cell,白/藍影)
	redSil             map[*ebiten.Image]*ebiten.Image // E1 紅色剪影近似快取；不是 raw DAC 脈衝本身
	dim                *ebiten.Image                   // 全螢幕暗化/底板共用(回合橫幅、單位面板)
	figMeta            map[int][][2]int                // FIGANI 每幀內嵌絕對螢幕座標 (dx,dy)@320(doc06;動畫走位全靠它)
	font               *Font                           // 原版點陣中文字型(doc 08)

	nativeChapterRestore *campaign.NativeChapterSlotRestorePlan // 四槽 LOAD 的已驗證戰間狀態；未知 raw bytes 僅保存、不猜接

	storyNativeMapView    battle.NativeMapViewState // LOADCH 後原版六個視圖全域的場景專用載體；不冒充 battle.State
	hasStoryNativeMapView bool                      // 僅在已證實的 LOADCH 視圖重設與 pan 步進後有效
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
	fpt              int                      // 播放速度(tick/幀;FD2_BATTLE_FPT 可調)
	atkOwn           bool                     // 攻方是否我方(狀態欄按陣營:我方欄右上/敵方欄左下)
	terrain          int                      // 攻擊格地形索引(戰鬥背景 = 戰場地形,跟 FDFIELD 戰場資料有關)
	figaniTimeline   *figani.DisplayScheduler // 已證實 FIGANI 幀延遲；不承載命中／傷害語意
	nativeImpactRaw  *nativeImpactDACInput    // 尚未接線；缺 raw provenance 時保持 nil
	frameIndex       int                      // 目前已呈現的 FIGANI 幀
	bodyTicks        int                      // 幀本體的精確延遲總長，尾段停格另計
	after            func()                   // 原版 action handler 完成後才進 selector1；不得在演出前提交
}

// nativeImpactDACInput 只保存原版 0x2939d 命中分支仍可回查的 raw 條件。
// rawOutput20/rawOutput1C 對應 0x29f72 輸出暫存的 stack local 位移；它們不是
// 暴擊、狀態或其他高階語意名稱。正式 renderer 尚未取得這組來源，故指標為 nil。
type nativeImpactDACInput struct {
	frameFlag          byte
	damageStepComplete bool
	rawOutput20        bool
	rawOutput1C        bool
}

func nativeImpactDACAllowed(raw *nativeImpactDACInput) bool {
	return raw != nil && raw.frameFlag == 1 && raw.damageStepComplete &&
		(raw.rawOutput20 || raw.rawOutput1C)
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

// transitionRevealJob owns native 0x24B4D's two proven row-shifted indexed
// viewports. The address-specific constructor validates and prepares every
// buffer before this blocking job can become visible.
type transitionRevealJob struct {
	work      []byte
	frames    [2][]byte
	palette   color.Palette
	index     int
	remaining int
	ticks     int
	delay     int
	drawn     bool
	then      func()
	rollback  func()
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
				u.SetMapPose(dirToward(0, 0, dx, 0))
			} else {
				cx, cy = float64(dx), (dist-float64(adx))*sgn(dy)
				u.SetMapPose(dirToward(0, 0, 0, dy))
			}
		} else {
			if dist <= float64(ady) {
				cx, cy = 0, dist*sgn(dy)
				u.SetMapPose(dirToward(0, 0, 0, dy))
			} else {
				cx, cy = (dist-float64(ady))*sgn(dx), float64(dy)
				u.SetMapPose(dirToward(0, 0, dx, 0))
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
				u.SetMapPose(w.finalDir)
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

func (g *Game) stepTransitionReveal() {
	job := g.transitionReveal
	if job == nil {
		return
	}
	if !job.drawn {
		return
	}
	if job.ticks > 0 {
		job.ticks--
		return
	}
	job.remaining--
	if job.remaining <= 0 {
		job.rollback = nil
		g.transitionReveal = nil
		if job.then != nil {
			job.then()
		}
		return
	}
	job.index ^= 1
	g.nativeMapVGA = append(g.nativeMapVGA[:0], job.frames[job.index]...)
	job.drawn = false
	// The current tick already presents one frame; wait only the remaining
	// delay ticks so a 20ms native delay maps to one 60Hz update, not two.
	job.ticks = job.delay - 1
	if job.ticks < 0 {
		job.ticks = 0
	}
}

// syncStoryNativeMapPanView mirrors raw 0x135dd's proven camera/absolute
// cursor lockstep.  The helper deliberately preserves the visible cursor:
// IDA shows that 0x135dd writes [0x53aa9/0x53aad] and
// [0x53ab1/0x53ab5], but not [0x53ab9/0x53abd].
func (g *Game) syncStoryNativeMapPanView() bool {
	if g == nil {
		return true
	}
	ch28Continuity := g.ch28HandlerNativeMapViewContinuity()
	hasBattleView := ch28Continuity && g.st != nil && g.st.HasNativeMapViewState
	if !g.hasStoryNativeMapView && !hasBattleView {
		return true
	}
	if g.m == nil || g.m.TileW <= 0 || g.m.TileH <= 0 || g.m.W <= 0 || g.m.H <= 0 ||
		int(g.camX)%g.m.TileW != 0 || int(g.camY)%g.m.TileH != 0 ||
		g.camX != float64(int(g.camX)) || g.camY != float64(int(g.camY)) {
		g.loadErr = "native story map view: pan camera is not tile-aligned"
		return false
	}
	view := g.storyNativeMapView
	if !g.hasStoryNativeMapView {
		view = g.st.NativeMapViewState
	}
	view.CameraX, view.CameraY = int(g.camX)/g.m.TileW, int(g.camY)/g.m.TileH
	view.CursorX = view.CameraX + view.VisibleCursorX
	view.CursorY = view.CameraY + view.VisibleCursorY
	carrier := &battle.State{W: g.m.W, H: g.m.H}
	if err := carrier.MaterializeNativeMapViewState(view); err != nil {
		g.loadErr = "native story map view: " + err.Error()
		return false
	}
	if g.hasStoryNativeMapView {
		g.storyNativeMapView = carrier.NativeMapViewState
	} else {
		g.st.NativeMapViewState = carrier.NativeMapViewState
	}
	// The original uses the same absolute cursor globals throughout the
	// handler. Keep the generic renderer cursor aligned with that typed carrier
	// so a following 0x12D7B focus starts from the post-pan position.
	if ch28Continuity {
		g.curX, g.curY = view.CursorX, view.CursorY
	}
	return true
}

// ch28HandlerNativeMapViewContinuity is deliberately caller-specific. Raw
// ch28 pre/post share the six native view globals across the remake's
// artificial cutscene/battle node boundary. Battle turn-event staging owns a
// separate private snapshot and must not be mutated by this bridge.
func (g *Game) ch28HandlerNativeMapViewContinuity() bool {
	if g == nil {
		return false
	}
	if g.camp == nil {
		return g.handlerChapter == 28 // narrow unit fixture
	}
	id := g.camp.NodeID()
	return id == "story_ch29" || id == "postbattle_ch29_persist"
}

// syncStoryNativeMapFocusView mirrors the six globals after one 0x12CEA
// focus step. Unlike 0x135DD, focus updates absolute and visible cursor while
// moving the camera only when the native safe band is crossed.
func (g *Game) syncStoryNativeMapFocusView(visibleX, visibleY int) bool {
	if g == nil || !g.ch28HandlerNativeMapViewContinuity() {
		return true
	}
	hasBattleView := g.st != nil && g.st.HasNativeMapViewState
	if !g.hasStoryNativeMapView && !hasBattleView {
		return true
	}
	if g.m == nil || g.m.TileW <= 0 || g.m.TileH <= 0 ||
		int(g.camX)%g.m.TileW != 0 || int(g.camY)%g.m.TileH != 0 ||
		g.camX != float64(int(g.camX)) || g.camY != float64(int(g.camY)) {
		g.loadErr = "native story map view: focus camera is not tile-aligned"
		return false
	}
	originX, originY := int(g.camX)/g.m.TileW, int(g.camY)/g.m.TileH
	view := battle.NativeMapViewState{
		CameraX: originX, CameraY: originY,
		CursorX: g.curX, CursorY: g.curY,
		VisibleCursorX: visibleX, VisibleCursorY: visibleY,
	}
	carrier := &battle.State{W: g.m.W, H: g.m.H}
	if err := carrier.MaterializeNativeMapViewState(view); err != nil {
		g.loadErr = "native story map view: " + err.Error()
		return false
	}
	if g.hasStoryNativeMapView {
		g.storyNativeMapView = carrier.NativeMapViewState
	} else {
		g.st.NativeMapViewState = carrier.NativeMapViewState
	}
	return true
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
		if !g.syncStoryNativeMapPanView() {
			g.camPan = nil
			return
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
		// focus_unit 也會被獨立 renderer／證據測試使用；只有 campaign
		// 擁有者存在時才能接下一拍。這不使 direct-entry 偽造戰役進度。
		if g.camp != nil {
			g.beatAdvance()
		}
	}
	originX, originY := int(g.camX)/g.m.TileW, int(g.camY)/g.m.TileH
	screenX, screenY := g.curX-originX, g.curY-originY
	if g.ch28HandlerNativeMapViewContinuity() && g.hasStoryNativeMapView {
		screenX, screenY = g.storyNativeMapView.VisibleCursorX, g.storyNativeMapView.VisibleCursorY
	} else if g.ch28HandlerNativeMapViewContinuity() && g.st != nil && g.st.HasNativeMapViewState {
		screenX, screenY = g.st.NativeMapViewState.VisibleCursorX, g.st.NativeMapViewState.VisibleCursorY
	}
	if g.curX == j.targetX && g.curY == j.targetY {
		if !g.syncStoryNativeMapFocusView(screenX, screenY) {
			g.focusJob = nil
			return
		}
		finish()
		return
	}
	maxOriginX, maxOriginY := g.m.W-13, g.m.H-8
	if maxOriginX < 0 {
		maxOriginX = 0
	}
	if maxOriginY < 0 {
		maxOriginY = 0
	}
	switch {
	case g.curX > j.targetX:
		g.curX--
		if screenX < 2 && originX > 0 {
			originX--
		} else {
			screenX--
		}
	case g.curX < j.targetX:
		g.curX++
		if screenX > 10 && originX < maxOriginX {
			originX++
		} else {
			screenX++
		}
	case g.curY > j.targetY:
		g.curY--
		if screenY < 2 && originY > 0 {
			originY--
		} else {
			screenY--
		}
	case g.curY < j.targetY:
		g.curY++
		if screenY > 5 && originY < maxOriginY {
			originY++
		} else {
			screenY++
		}
	}
	g.camX, g.camY = float64(originX*g.m.TileW), float64(originY*g.m.TileH)
	if !g.syncStoryNativeMapFocusView(screenX, screenY) {
		g.focusJob = nil
		return
	}
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

// handlerRuntimeSlotCount preserves LOADCH's raw FDFIELD count before later
// SPAWN beats append records to the scene actor array.  A runtime_context beat
// therefore validates the original load context, not only the currently
// visible/materialized actors.
func (g *Game) handlerRuntimeSlotCount() int {
	if g.st != nil {
		return len(g.st.Units)
	}
	if g.storyRoster != nil {
		return len(g.storyRoster)
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
			u.SetMapPose(au.Pose)
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
		// The native loop writes pose (+3) before every redraw even for this
		// zero-duration special frame.  Re-apply it here so a concurrent
		// actor update cannot leave the frame with a stale direction.
		for _, au := range f.Units {
			if u := g.actingActor(au); u != nil {
				u.SetMapPose(au.Pose)
			}
		}
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
			if j.tick < 7 {
				u.SetNativeMapGridMotion(au.Pose, j.tick)
			}
		}
	}
	if j.tick < 7 {
		return
	}
	for _, au := range f.Units {
		if u := g.actingActor(au); u != nil {
			dx, dy := actingDelta(au.Pose)
			x, y := u.X+dx, u.Y+dy
			if !u.FinishNativeMapGridStep(au.Pose, x, y) {
				u.X, u.Y = x, y
				u.OffX, u.OffY = 0, 0
				u.SetMapPose(au.Pose)
			}
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
	u.SetMapPose(j.poses[j.idx])
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

// fastForwardShotCampaign 只供 FD2_SHOT_FAST_FORWARD=1 的隔離畫面證據。
// 它仍從目前 campaign 節點執行既有 BeatRunner，逐一完成阻塞拍的「狀態副作用」；
// 不會在一般玩家路徑啟用，也不把不同節點或未證實的 handler 直接接成戰鬥。
// 對白／走位／姿態／鏡頭／淡出只略過可見時間，LOADCH、SPAWN、JOIN、SYNC、
// SET_CHAPTER 與原生資源驗證仍由 beatStart 真實執行。遇到未完成的原生 renderer
// 或無法辨識的阻塞狀態時立即失敗（fail-closed），避免產生看似正式的截圖。
func (g *Game) fastForwardShotCampaign() error {
	if g == nil || g.camp == nil {
		return errors.New("shot fast-forward requires campaign")
	}
	const maxSteps = 200000
	for step := 0; step < maxSteps; step++ {
		n := g.camp.Node()
		if n == nil || n.Type == "battle" {
			return nil
		}
		if n.Type != "story" && n.Type != "cutscene" {
			return fmt.Errorf("shot fast-forward reached non-battle node=%q type=%q", g.camp.Cur, n.Type)
		}
		if g.loadErr != "" {
			return fmt.Errorf("shot fast-forward stopped: %s", g.loadErr)
		}
		switch {
		case len(g.dialog) > 0:
			// 逐頁、逐句重播 Enter 的狀態轉移；不可直接清空長句，否則
			// 截圖雖然前進，卻無法證明對白腳本與頁數已被消費。
			g.dlgScrollT = 0
			if g.dlgAdvance() && len(g.dialog) == 0 {
				g.dlgShown, g.dlgPhase, g.dlgT = dlgNone, 0, 0
				g.beatAdvance()
			}
		case g.storyWalks != nil && len(g.storyWalks) > 0:
			// 使用正式逐幀方法完成走位，保留格線終點、姿態與 callback。
			for ticks := 0; len(g.storyWalks) > 0 && ticks < maxSteps; ticks++ {
				g.stepStoryWalks()
			}
			if len(g.storyWalks) != 0 {
				return errors.New("shot fast-forward story walk exceeded step bound")
			}
		case g.actJob != nil:
			// acting frame 仍透過原本的 0x1366a 轉錄執行；只壓縮每格的
			// 7 tick 顯示等待，不改變最終座標／pose。
			for ticks := 0; g.actJob != nil && ticks < maxSteps; ticks++ {
				g.stepActJob()
			}
			if g.actJob != nil {
				return errors.New("shot fast-forward acting exceeded step bound")
			}
		case g.focusJob != nil:
			for ticks := 0; g.focusJob != nil && ticks < maxSteps; ticks++ {
				g.stepFocusUnit()
			}
			if g.focusJob != nil {
				return errors.New("shot fast-forward focus exceeded step bound")
			}
		case g.camPan != nil:
			// callback 會進入下一拍；先清 job 避免 callback 重入時被視為
			// 舊拍仍在執行。線性 pan 的終點就是 Beat payload 的終點。
			job := g.camPan
			g.camX, g.camY, g.camPan = job.toX, job.toY, nil
			if !g.syncStoryNativeMapPanView() {
				return fmt.Errorf("shot fast-forward camera view: %s", g.loadErr)
			}
			if job.then != nil {
				job.then()
			}
		case g.fade != nil:
			job := g.fade
			g.fade = nil
			if job.then != nil {
				job.then()
			}
		case g.transitionReveal != nil:
			for ticks := 0; g.transitionReveal != nil && ticks < maxSteps; ticks++ {
				g.transitionReveal.drawn = true
				g.transitionReveal.ticks = 0
				g.stepTransitionReveal()
			}
			if g.transitionReveal != nil {
				return errors.New("shot fast-forward native 0x24B4D exceeded step bound")
			}
		case g.nativePaletteRamp != nil:
			for ticks := 0; g.nativePaletteRamp != nil && ticks < maxSteps; ticks++ {
				// Draw() normally acknowledges the currently presented DAC step.
				g.nativePaletteRamp.drawn = true
				g.stepNativePaletteRamp()
			}
			if g.nativePaletteRamp != nil {
				return errors.New("shot fast-forward palette ramp exceeded step bound")
			}
		case g.nativePalettePulse != nil:
			for ticks := 0; g.nativePalettePulse != nil && ticks < maxSteps; ticks++ {
				g.nativePalettePulse.drawn = true
				g.nativePalettePulse.wait = 0
				g.stepNativePalettePulse()
			}
			if g.nativePalettePulse != nil {
				return errors.New("shot fast-forward native 0x35E5A exceeded step bound")
			}
		case g.spawnIntroTransition != nil:
			for ticks := 0; g.spawnIntroTransition != nil && ticks < maxSteps; ticks++ {
				// Each pass still goes through the real stepper and preserves its
				// sound callback; only the visual present acknowledgement is synthetic.
				g.spawnIntroTransition.drawn = true
				g.stepNativeSpawnIntro()
			}
			if g.spawnIntroTransition != nil {
				return errors.New("shot fast-forward spawn intro exceeded step bound")
			}
		case g.indexedTransition != nil:
			for ticks := 0; g.indexedTransition != nil && ticks < maxSteps; ticks++ {
				g.indexedTransition.drawn = true
				g.stepNativeIndexedTransition()
			}
			if g.indexedTransition != nil {
				return errors.New("shot fast-forward indexed transition exceeded step bound")
			}
		case g.nativeCh20SkyKey != nil:
			for ticks := 0; g.nativeCh20SkyKey != nil && ticks < maxSteps; ticks++ {
				g.nativeCh20SkyKey.drawn = true
				g.nativeCh20SkyKey.ticks = 0
				g.stepNativeCh20SkyKey()
			}
			if g.nativeCh20SkyKey != nil {
				return errors.New("shot fast-forward native 0x24336 exceeded step bound")
			}
		case g.nativeCh23Loop != nil:
			for ticks := 0; g.nativeCh23Loop != nil && ticks < maxSteps; ticks++ {
				g.nativeCh23Loop.drawn = true
				g.nativeCh23Loop.waitFrames = 0
				now := g.nativeMapClock.last.Add(nativeBIOSTickPeriod)
				g.stepNativeCh23LoopAt(now)
			}
			if g.nativeCh23Loop != nil {
				return errors.New("shot fast-forward native ch23 loop exceeded step bound")
			}
		case g.native2189A != nil:
			for ticks := 0; g.native2189A != nil && ticks < maxSteps; ticks++ {
				g.native2189A.drawn = true
				g.stepNative2189A()
			}
			if g.native2189A != nil {
				return errors.New("shot fast-forward native 0x2189A exceeded step bound")
			}
		case g.nativeUnitPresent != nil:
			for ticks := 0; g.nativeUnitPresent != nil && ticks < maxSteps; ticks++ {
				g.nativeUnitPresent.drawn = true
				g.nativeUnitPresent.wait = 0
				g.stepNativeUnitPresent()
			}
			if g.nativeUnitPresent != nil {
				return errors.New("shot fast-forward native 0x22253 exceeded step bound")
			}
		case g.nativeCh28PostPresent != nil:
			for ticks := 0; g.nativeCh28PostPresent != nil && ticks < maxSteps; ticks++ {
				g.nativeCh28PostPresent.drawn = true
				g.nativeCh28PostPresent.wait = 0
				g.stepNativeCh28PostPresent()
			}
			if g.nativeCh28PostPresent != nil {
				return errors.New("shot fast-forward native 0x1DB65 exceeded step bound")
			}
		case g.beatDelay > 0:
			g.beatDelay = 0
			g.beatAdvance()
		case g.storyAutoAdvance > 0:
			g.storyAutoAdvance = 0
			g.advanceStoryNode(n)
		default:
			return fmt.Errorf("shot fast-forward stuck at node=%q beat=%d", g.camp.Cur, g.beatIdx)
		}
	}
	return errors.New("shot fast-forward exceeded campaign step bound")
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
		if !b.RuntimeContext.AcceptsSlotCount(g.handlerRuntimeSlotCount()) {
			// Compiler metadata is emitted before LOADCH, while the previous
			// battle state may still be installed (or the scene array may be
			// empty). Validate the declared LOADCH record count here instead of
			// comparing unrelated prior-battle slots; applyLoadCH then validates
			// the actual roster bytes before replacing the scene state.
			nextIsLoadCH := g.beatIdx+1 < len(g.beats) && g.beats[g.beatIdx+1].Op == "loadch" && g.beats[g.beatIdx+1].LoadCH != nil
			if !nextIsLoadCH || !b.RuntimeContext.AcceptsSlotCount(g.beats[g.beatIdx+1].LoadCH.SlotCount) {
				g.loadErr = fmt.Sprintf("beat runtime_context: runtime slots=%d, want exact %d or one of %v", g.handlerRuntimeSlotCount(), b.RuntimeContext.SlotCount, b.RuntimeContext.SlotCounts)
				return
			}
		}
		if b.RuntimeContext.StoryViewport {
			g.storyBG = true
		}
		g.beatAdvance()
	case "unit_present":
		// Legacy metadata described only the six-frame 0x22547 tail and did
		// not identify a proven caller.  The separate native_unit_present op
		// owns the recovered 0x25535 battle-state path; keep this legacy shape
		// closed instead of silently treating it as that caller.
		if b.UnitPresent == nil {
			g.loadErr = "beat unit_present:缺少 placement payload"
			return
		}
		g.loadErr = "beat unit_present: legacy payload未綁定已證實的0x22253 caller"
		return
	case "native_unit_present":
		if b.NativeUnitPresent == nil || b.Source != "0x25535" || !b.NativeUnitPresent.LastRuntimeSlot {
			g.loadErr = "beat native_unit_present:缺少原版 0x25535 payload"
			return
		}
		if err := g.startNativeUnitPresent(*b.NativeUnitPresent, g.beatAdvance); err != nil {
			g.loadErr = "beat native_unit_present: " + err.Error()
		}
		return
	case "native_ch28_post_present":
		if b.NativeCh28PostPresent == nil || b.Source != "0x254c0" ||
			!b.NativeCh28PostPresent.IsRecoveredContract() {
			g.loadErr = "beat native_ch28_post_present:缺少原版 0x254c0 payload"
			return
		}
		if err := g.startNativeCh28PostPresent(g.beatAdvance); err != nil {
			g.loadErr = "beat native_ch28_post_present: " + err.Error()
		}
		return
	case "indexed_transition":
		if b.IndexedTransition == nil {
			g.loadErr = "beat indexed_transition:缺少 transition payload"
			return
		}
		if err := g.startNativeIndexedTransition(*b.IndexedTransition, b.Source, g.beatAdvance); err != nil {
			g.loadErr = "beat indexed_transition: " + err.Error()
		}
		return
	case "native_ch20_sky_key_sequence":
		if b.Source != "0x242c9" || !b.NativeCh20SkyKey.IsRecoveredContract() {
			g.loadErr = "beat native_ch20_sky_key_sequence:缺少原版 0x242c9 payload"
			return
		}
		if err := g.startNativeCh20SkyKeySequence(*b.NativeCh20SkyKey, g.beatAdvance); err != nil {
			g.loadErr = "beat native_ch20_sky_key_sequence: " + err.Error()
		}
		return
	case "native_palette_fade_out":
		if b.NativePaletteFade == nil || b.NativePaletteFade.Start != 0 || b.NativePaletteFade.End != 63 || b.NativePaletteFade.DelayMs != 2 {
			g.loadErr = "beat native_palette_fade_out:缺少原版 64-step DAC payload"
			return
		}
		if err := g.startNativePaletteRamp(0, 63, 2, g.beatAdvance); err != nil {
			g.loadErr = "beat native_palette_fade_out: " + err.Error()
		}
		return
	case "native_palette_fade_in":
		if b.NativePaletteFadeIn == nil || b.NativePaletteFadeIn.Start != 64 || b.NativePaletteFadeIn.End != 0 || b.NativePaletteFadeIn.DelayMs != 2 {
			g.loadErr = "beat native_palette_fade_in:缺少原版 65-step DAC payload"
			return
		}
		if err := g.startNativePaletteRamp(64, 0, 2, g.beatAdvance); err != nil {
			g.loadErr = "beat native_palette_fade_in: " + err.Error()
		}
		return
	case "native_palette_pulse":
		pulse := b.NativePalettePulse
		if pulse == nil || pulse.RiseStart != 0 || pulse.RiseEnd != 63 || pulse.RiseDelayMs != 8 || pulse.HoldMs != 400 || pulse.FallStart != 62 || pulse.FallEnd != 0 || pulse.FallDelayMs != 8 {
			g.loadErr = "beat native_palette_pulse:缺少原版 DAC pulse payload"
			return
		}
		if err := g.startNativePalettePulse(*pulse, g.beatAdvance); err != nil {
			g.loadErr = "beat native_palette_pulse: " + err.Error()
		}
		return
	case "native_palette_blackout":
		blackout := b.NativePaletteBlackout
		if blackout == nil || blackout.Start != 0 || blackout.End != 255 ||
			blackout.Delta != 64 || blackout.ClearBytes != 0xFA00 || b.Source != "0x23599" {
			g.loadErr = "beat native_palette_blackout:缺少原版全色盤與畫面清除 payload"
			return
		}
		// Every baseline DAC component is at most 63, so the exact native
		// subtraction clamps all 256 entries to zero.  The adjacent memset
		// clears the complete 320x200 framebuffer.  A full-surface black cover
		// is therefore exact for this one call site without inventing an
		// indexed-palette approximation for other 0x11d40 callers.
		if len(g.nativeMapVGA) != 0 {
			if len(g.nativeMapVGA) < blackout.ClearBytes {
				g.loadErr = fmt.Sprintf("beat native_palette_blackout: indexed framebuffer=%d, want at least %d", len(g.nativeMapVGA), blackout.ClearBytes)
				return
			}
			clear(g.nativeMapVGA[:blackout.ClearBytes])
		}
		g.nativeFullDACBlack = true
		g.beatAdvance()
	case "native_staging_present":
		present := b.NativeStagingPresent
		if present == nil || present.Slot < 0 || present.X < 0 || present.Y < 0 || present.FocusX != present.Slot || present.FocusY != present.X {
			g.loadErr = "beat native_staging_present:缺少原版 wrapper ABI payload"
			return
		}
		// 0x33f78 calls 0x22253 after focusing. The shared battle-state
		// presenter is available, but this story-array/focus owner remains
		// separate; it is not a spawn, position change, or ordinary camera pan.
		g.loadErr = "beat native_staging_present: 0x33f78 story/focus adapter未完成"
		return
	case "native_ch23_loop":
		if b.NativeCh23Loop == nil {
			g.loadErr = "beat native_ch23_loop:缺少原始兩段 loop payload"
			return
		}
		if err := g.startNativeCh23Loop(*b.NativeCh23Loop, g.beatAdvance); err != nil {
			g.loadErr = "beat native_ch23_loop: " + err.Error()
		}
		return
	case "native_2189a_loop":
		if b.Native2189ALoop == nil {
			g.loadErr = "beat native_2189a_loop:缺少原始十次 loop payload"
			return
		}
		if err := g.startNative2189A(*b.Native2189ALoop, g.beatAdvance); err != nil {
			g.loadErr = "beat native_2189a_loop: " + err.Error()
		}
		return
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
			unit.SetMapPlacement(placement.X, placement.Y, placement.Pose)
		}
		g.camX, g.camY = float64(b.Layout.CamX), float64(b.Layout.CamY)
		g.beatAdvance()
	case "direct_record_patch":
		if (b.Source != "0x2362d" && b.Source != "0x23ec4") || b.DirectRecordPatch == nil {
			g.loadErr = "beat direct_record_patch:缺少原版來源或 sparse payload"
			return
		}
		if err := g.applyHandlerDirectRecordPatch(b.DirectRecordPatch); err != nil {
			g.loadErr = "beat direct_record_patch: " + err.Error()
			return
		}
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
		materialized := 0
		if g.st != nil {
			if b.RawPlacementGate != nil {
				var err error
				materialized, err = g.st.AppendGroupWithNativePlacement(
					b.Group, byte(*b.RawPlacementGate),
				)
				if err != nil {
					g.loadErr = fmt.Sprintf("beat spawn %s: %v", b.Source, err)
					return
				}
			} else {
				materialized = g.st.AppendGroup(b.Group)
			}
		} else {
			materialized = g.materializeStoryGroup(b.Group)
		}
		if materialized == 0 && (b.Source != "" || b.Count > 0) {
			g.loadErr = fmt.Sprintf("beat spawn %s: group %d unavailable in runtime roster", b.Source, b.Group)
			return
		}
		if b.Count > 0 && materialized != b.Count {
			g.loadErr = fmt.Sprintf("beat spawn %s: group %d materialized=%d want=%d", b.Source, b.Group, materialized, b.Count)
			return
		}
		g.beatAdvance()
	case "spawn_intro":
		if b.Source != "" {
			if b.RawPlacementGate == nil {
				g.loadErr = fmt.Sprintf("beat spawn_intro %s: raw placement gate unavailable", b.Source)
				return
			}
			if err := g.startNativeSpawnIntro(b.Group, byte(*b.RawPlacementGate), g.beatAdvance); err != nil {
				g.loadErr = fmt.Sprintf("beat spawn_intro %s: %v", b.Source, err)
			}
			return
		}
		if g.st != nil && b.RawPlacementGate != nil {
			if _, err := g.st.AppendGroupWithNativePlacement(
				b.Group, byte(*b.RawPlacementGate),
			); err != nil {
				g.loadErr = fmt.Sprintf("beat spawn_intro %s: %v", b.Source, err)
				return
			}
		} else {
			g.materializeStoryGroup(b.Group)
		}
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
		u := g.handlerUnitAt(*b.Slot)
		u.OnField = false
		if u.HasNativeRecordByte5 {
			// 0x32975 overwrites the complete byte with 1; do not OR only bit0.
			u.NativeRecordByte5 = 1
		}
		g.beatAdvance()
	case "reactivate_nonzero_hp":
		if b.Source != "0x33cea" || b.Count <= 0 {
			g.loadErr = "beat reactivate_nonzero_hp:缺少已證實的 counted-loop 來源"
			return
		}
		// Preflight the whole raw loop before changing any record.  The native
		// +0x40 word is the already-closed current-HP field represented by Unit.HP;
		// byte +5 remains explicit provenance and is never inferred from HP.
		for slot := 0; slot < b.Count; slot++ {
			u := g.handlerUnitAt(slot)
			if u == nil || u.Camp != battle.Own || !u.HasNativeRecordByte5 {
				g.loadErr = fmt.Sprintf("beat reactivate_nonzero_hp: slot%d lacks player/raw byte+5 provenance", slot)
				return
			}
		}
		for slot := 0; slot < b.Count; slot++ {
			u := g.handlerUnitAt(slot)
			if u.HP != 0 {
				u.OnField = true
				u.NativeRecordByte5 = 0
			}
		}
		g.beatAdvance()
	case "reset_pose":
		if g.st != nil {
			for _, unit := range g.st.Units {
				if unit != nil {
					unit.SetMapPose(0)
				}
			}
		} else {
			for i := range g.storyActors {
				g.storyActors[i].SetMapPose(0)
			}
		}
		g.beatDelay = 1 // original 20ms at a 60Hz remake clock
	case "redraw":
		if b.Source == "0x236ee" || b.Source == "0x23f69" {
			if err := g.composeNativeMapFrame(); err != nil {
				g.loadErr = "beat redraw: native " + b.Source + ": " + err.Error()
				return
			}
		}
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
		if _, exists := g.partyRoster[b.CharID]; b.Source != "" && !exists {
			var base *battle.Unit
			if g.st != nil {
				matches := 0
				for _, unit := range g.st.Units {
					if unit != nil && unit.HasNativeRecordByte8 && int(unit.NativeRecordByte8) == b.CharID {
						base = unit
						matches++
					}
				}
				if matches > 1 {
					g.loadErr = fmt.Sprintf("beat join:char_id=%d 有 %d 筆 raw +8 runtime records", b.CharID, matches)
					return
				}
			}
			if base == nil {
				matches := 0
				for i := range g.storyActors {
					unit := &g.storyActors[i]
					if unit.HasNativeRecordByte8 && int(unit.NativeRecordByte8) == b.CharID {
						base = unit
						matches++
					}
				}
				if matches > 1 {
					g.loadErr = fmt.Sprintf("beat join:char_id=%d 有 %d 筆 raw +8 story records", b.CharID, matches)
					return
				}
			}
			if base == nil {
				// Native sub_112A5 appends a new persistent record from its
				// character-data helpers; it does not require the character to
				// already occupy the current field array.  Only an explicit,
				// evidence-labelled base roster may bridge that boundary.
				if !g.hasNativeJoinBases {
					joinPath := assetPath("assets/data/native_join_base_units.json")
					table, err := campaign.LoadNativeJoinBaseTable(joinPath)
					if err != nil {
						g.loadErr = fmt.Sprintf("beat join:char_id=%d 缺少 raw +8 runtime record 且 base table 無法載入: %v", b.CharID, err)
						return
					}
					g.nativeJoinBases = table
					g.hasNativeJoinBases = true
				}
				fallback, err := g.nativeJoinBases.LoadBaseUnit(b.CharID)
				if err != nil {
					g.loadErr = "beat join: " + err.Error()
					return
				}
				base = &fallback
			}
			materialized, err := g.materializeNativeJoinPersistentUnit(b.CharID, *base)
			if err != nil {
				g.loadErr = "beat join: " + err.Error()
				return
			}
			if g.partyRoster == nil {
				g.partyRoster = make(map[int]battle.Unit)
			}
			g.partyRoster[b.CharID] = cloneNativeShopUnit(materialized)
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
	case "clear_native_record_bit7":
		if g.st == nil || len(g.st.Units) == 0 {
			g.loadErr = "beat clear_native_record_bit7:缺少 runtime battle records"
			return
		}
		for i, unit := range g.st.Units {
			if unit == nil || !unit.HasNativeRecordByte5 {
				g.loadErr = fmt.Sprintf("beat clear_native_record_bit7: slot%d lacks raw byte+5 provenance", i)
				return
			}
		}
		for _, unit := range g.st.Units {
			unit.NativeRecordByte5 &= 0x7f
		}
		g.beatAdvance()
	case "reset_persistent_roster_state":
		g.resetPersistentRosterState()
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
	case "transition_reveal":
		if b.RevealFrames <= 0 {
			g.loadErr = "beat transition_reveal:缺少正確 frame count"
			return
		}
		if err := g.startNative24B4D(b.RevealFrames, b.RevealDelayMs, g.beatAdvance); err != nil {
			g.loadErr = "beat transition_reveal: " + err.Error()
		}
	case "load_res":
		if b.ResourceID == nil || *b.ResourceID < 0 {
			g.loadErr = "beat load_res:缺少 resource_id"
			return
		}
		if b.Source == "0x24a4b" || b.Source == "0x24a65" || b.Source == "0x24a7f" {
			if err := g.stageNativeCh22Resource(b); err != nil {
				g.nativeCh22Reload = nil
				g.loadErr = "beat load_res: " + err.Error()
				return
			}
		} else {
			g.handlerResource = *b.ResourceID
		}
		g.beatAdvance()
	case "native_ch22_reset_grid":
		if err := g.resetNativeCh22ReloadGrid(); err != nil {
			g.nativeCh22Reload = nil
			g.loadErr = "beat native_ch22_reset_grid: " + err.Error()
			return
		}
		g.beatAdvance()
	case "native_ch22_prepare_aux":
		if err := g.prepareNativeCh22Aux(); err != nil {
			g.nativeCh22Reload = nil
			g.loadErr = "beat native_ch22_prepare_aux: " + err.Error()
			return
		}
		g.beatAdvance()
	case "play_sfx":
		if b.ResourceID == nil || b.SFXIndex == nil || *b.ResourceID != 88 || (*b.SFXIndex != 1 && *b.SFXIndex != -1) {
			g.loadErr = fmt.Sprintf("beat play_sfx:未映射 resource=%v index=%v", b.ResourceID, b.SFXIndex)
			return
		}
		if g.handlerResource != 88 {
			g.loadErr = fmt.Sprintf("beat play_sfx:resource handle=%d want 88", g.handlerResource)
			return
		}
		if *b.SFXIndex == 1 {
			g.playRaw(g.sfxTransition)
		}
		g.beatAdvance()
	case "release_res":
		if b.ResourceID == nil || *b.ResourceID != g.handlerResource {
			g.loadErr = fmt.Sprintf("beat release_res:resource=%v handle=%d", b.ResourceID, g.handlerResource)
			return
		}
		g.handlerResource = 0
		g.beatAdvance()
	case "palette_update":
		// Full range delta=255 is independent of source RGB: every six-bit
		// component saturates to white. This exact operation can cover legacy
		// scene presentation without inventing a general RGB approximation.
		// Delta=0 restores baseline; every other non-zero shape stays closed.
		switch {
		case b.PaletteStart == 0 && b.PaletteEnd == 255 && b.PaletteDelta == 255:
			g.nativeFullDACWhite = true
		case b.PaletteDelta == 0:
			g.nativeFullDACWhite = false
		default:
			g.loadErr = fmt.Sprintf(
				"beat palette_update: range %d..%d delta %d requires indexed palette renderer",
				b.PaletteStart, b.PaletteEnd, b.PaletteDelta,
			)
			return
		}
		g.beatAdvance()
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
	if condition == nil {
		return false, fmt.Errorf("缺少有效 handler condition")
	}
	switch condition.Op {
	case "native_event_state_nonzero":
		if g.st == nil || condition.EventStateIndex == nil || *condition.EventStateIndex < 0 || *condition.EventStateIndex >= len(g.st.NativeEventState) {
			return false, fmt.Errorf("缺少有效 native_event_state condition")
		}
		return g.st.NativeEventState[*condition.EventStateIndex] != 0, nil
	case "native_event_state_eq":
		if g.st == nil || condition.EventStateIndex == nil || *condition.EventStateIndex < 0 || *condition.EventStateIndex >= len(g.st.NativeEventState) ||
			condition.EventStateValue == nil || *condition.EventStateValue < 0 || *condition.EventStateValue > 0xff ||
			condition.RequiredSlotCount == nil || *condition.RequiredSlotCount <= 0 {
			return false, fmt.Errorf("缺少有效 native_event_state_eq condition")
		}
		matched := g.st.NativeEventState[*condition.EventStateIndex] == byte(*condition.EventStateValue)
		if matched && len(g.st.Units) != *condition.RequiredSlotCount {
			return false, fmt.Errorf(
				"native_event_state_eq matched but runtime slots=%d, want exact %d",
				len(g.st.Units), *condition.RequiredSlotCount,
			)
		}
		return matched, nil
	case "roster_has":
		if condition.CharID == nil || !campaign.JoinableCharacterID(*condition.CharID) {
			return false, fmt.Errorf("缺少有效 roster_has char_id")
		}
		// 0x33499 reads the permanent player roster, not the temporary deployed
		// battle party.  Do not infer it from story actors: a direct/debug entry
		// without persistent membership must stop instead of choosing a branch.
		if g.partyMembers == nil {
			return false, fmt.Errorf("roster_has 缺少 permanent party roster")
		}
		return g.partyMembers[*condition.CharID], nil
	case "any_unit_inactive":
		if len(condition.UnitSlots) == 0 {
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
		rawComplete := true
		for _, unit := range g.st.Units {
			if unit == nil || !unit.HasNativeRecordByte5 {
				rawComplete = false
				break
			}
		}
		for _, slot := range condition.UnitSlots {
			unit := g.st.Units[slot]
			if rawComplete {
				// A fully materialized runtime must use the native raw predicate;
				// do not silently substitute HP/OnField semantics.
				if unit.NativeRecordByte5&1 != 0 {
					return true, nil
				}
				continue
			}
			// Compatibility projection for old authored JSON without complete raw
			// provenance. This remains E1, not native parity; SDD56 documents it.
			if unit.HasNativeRecordByte5 {
				if unit.NativeRecordByte5&1 != 0 {
					return true, nil
				}
				continue
			}
			if !unit.OnField || !unit.Alive() {
				return true, nil
			}
		}
		return false, nil
	case "native_inventory_item_present":
		if g.st == nil || condition.NativeInventoryItemID == nil || *condition.NativeInventoryItemID < 0 || *condition.NativeInventoryItemID > 0xff {
			return false, fmt.Errorf("native_inventory_item_present lacks raw item byte or battle state")
		}
		if len(g.st.Units) < 16 {
			return false, fmt.Errorf("native_inventory_item_present requires 16 runtime records, got %d", len(g.st.Units))
		}
		records, err := battle.NativeInventoryRecords(g.st.Units, 16)
		if err != nil {
			return false, fmt.Errorf("native_inventory_item_present raw records: %w", err)
		}
		unit, _, err := battle.FindNativeInventoryItem(records, byte(*condition.NativeInventoryItemID))
		if err != nil {
			return false, fmt.Errorf("native_inventory_item_present search: %w", err)
		}
		return unit >= 0, nil
	case "native_persistent_identity_present":
		if condition.NativePersistentIdentity == nil || *condition.NativePersistentIdentity < 0 || *condition.NativePersistentIdentity > 0xff {
			return false, fmt.Errorf("native_persistent_identity_present lacks raw record+0x08 byte")
		}
		if g.partyRoster == nil {
			return false, fmt.Errorf("native_persistent_identity_present lacks persistent roster")
		}
		for index, unit := range g.partyRoster {
			if !unit.HasNativeIdentity {
				return false, fmt.Errorf("native_persistent_identity_present roster record %d lacks raw +0x08", index)
			}
			if unit.NativeIdentity == *condition.NativePersistentIdentity {
				return true, nil
			}
		}
		return false, nil
	case "native_inactive_count_gt":
		if len(condition.UnitSlots) == 0 || condition.Threshold == nil || *condition.Threshold < 0 {
			return false, fmt.Errorf("缺少有效 native_inactive_count_gt condition")
		}
		if g.st == nil {
			return false, fmt.Errorf("native_inactive_count_gt 缺少 runtime battle state")
		}
		inactive := 0
		for _, slot := range condition.UnitSlots {
			if slot < 0 || slot >= len(g.st.Units) || g.st.Units[slot] == nil {
				return false, fmt.Errorf("native_inactive_count_gt slot %d unavailable (units=%d)", slot, len(g.st.Units))
			}
			unit := g.st.Units[slot]
			if !unit.HasNativeRecordByte5 {
				return false, fmt.Errorf("native_inactive_count_gt slot %d lacks raw byte5", slot)
			}
			if unit.NativeRecordByte5&1 != 0 {
				inactive++
			}
		}
		return inactive > *condition.Threshold, nil
	case "native_round_gt":
		if g.st == nil || condition.NativeRound == nil || *condition.NativeRound < 0 || g.st.NativeRoundCounter <= 0 {
			return false, fmt.Errorf("native_round_gt lacks raw [0x53bef] provenance")
		}
		return g.st.NativeRoundCounter > *condition.NativeRound, nil
	case "native_round_lt":
		if g.st == nil || condition.NativeRound == nil || *condition.NativeRound < 0 || g.st.NativeRoundCounter <= 0 {
			return false, fmt.Errorf("native_round_lt lacks raw [0x53bef] provenance")
		}
		return g.st.NativeRoundCounter < *condition.NativeRound, nil
	case "native_record_word_gte":
		if g.st == nil || condition.UnitSlot == nil || condition.NativeRecordWordOffset == nil || *condition.NativeRecordWordOffset != 0x42 || condition.NativeRecordWordValue == nil || *condition.NativeRecordWordValue < 0 || *condition.NativeRecordWordValue > 0xffff {
			return false, fmt.Errorf("native_record_word_gte lacks raw +0x42 contract")
		}
		slot := *condition.UnitSlot
		if slot < 0 || slot >= len(g.st.Units) || g.st.Units[slot] == nil {
			return false, fmt.Errorf("native_record_word_gte slot %d unavailable (units=%d)", slot, len(g.st.Units))
		}
		u := g.st.Units[slot]
		if !u.HasNativeRecordWord42 {
			return false, fmt.Errorf("native_record_word_gte slot %d lacks raw +0x42", slot)
		}
		return int(u.NativeRecordWord42) >= *condition.NativeRecordWordValue, nil
	case "native_any_of":
		if len(condition.Any) == 0 {
			return false, fmt.Errorf("native_any_of lacks proven child predicates")
		}
		var firstErr error
		for i := range condition.Any {
			matched, err := g.evalBeatCondition(&condition.Any[i])
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			if matched {
				return true, nil
			}
		}
		if firstErr != nil {
			return false, firstErr
		}
		return false, nil
	default:
		return false, fmt.Errorf("未知 handler condition %q", condition.Op)
	}
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

func (g *Game) materializeStoryGroup(group int) int {
	if g.storyRoster != nil {
		if g.storySpawned[group] {
			return 0
		}
		materialized := 0
		for _, actor := range g.storyRoster {
			if actor.Group == group {
				actor.OnField = true
				g.storyActors = append(g.storyActors, actor)
				materialized++
			}
		}
		if materialized > 0 {
			g.storySpawned[group] = true
		}
		return materialized
	}
	materialized := 0
	for i := range g.storyActors {
		if g.storyActors[i].Group == group {
			g.storyActors[i].OnField = true
			materialized++
		}
	}
	return materialized
}

// loadCHPartyOrder projects the permanent JOIN chronology through the current
// preparation selection.  The original LOADCH constructs the deployed party
// first, so a late-game cutscene must not silently resurrect every joined
// character merely because partyJoinOrder is longer than the selected roster.
// With no JOIN history this remains a direct/debug binding's authored order.
func (g *Game) loadCHPartyOrder(state *campaign.LoadCHState) []int {
	if g == nil || len(g.partyJoinOrder) == 0 {
		if state == nil {
			return nil
		}
		return append([]int(nil), state.PartyOrder...)
	}
	members := g.battlePartyMembers()
	order := make([]int, 0, len(g.partyJoinOrder))
	for _, id := range g.partyJoinOrder {
		if members[id] {
			order = append(order, id)
		}
	}
	return order
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
	// 0x205da/0x1088d resets camera, absolute cursor and visible cursor to
	// zero after loading the field. Keep this raw view in a scene-only carrier;
	// putting it in g.st would make later story SPAWN beats use the battle-state
	// append path and invent runtime records that LOADCH has not materialized.
	loadCHView := battle.NativeMapViewState{}
	viewCarrier := &battle.State{W: g.m.W, H: g.m.H}
	if err := viewCarrier.MaterializeNativeMapViewState(loadCHView); err != nil {
		return fmt.Errorf("map %q native view reset: %w", state.Map, err)
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
		filterScenarioParty(scenario, g.battlePartyMembers())
		partyOrder := g.loadCHPartyOrder(state)
		// A binding's PartyOrder describes the normal permanent chronology.  Once
		// preparation has an active selection, the projected deployed order is the
		// native runtime source and may intentionally be shorter than that field.
		if len(g.partyJoinOrder) != 0 && len(g.partyDeploy) == 0 && len(state.PartyOrder) != 0 && !equalIntOrder(partyOrder, state.PartyOrder) {
			return fmt.Errorf("party JOIN chronology %v differs from binding %v", partyOrder, state.PartyOrder)
		}
		if err := reorderScenarioParty(scenario, partyOrder); err != nil {
			return fmt.Errorf("party scenario %q: %w", state.PartyScenario, err)
		}
		party = scenario.PartyUnits(roster.OwnDeploy)
		// Native JOIN has already created persistent records before a normal
		// campaign reaches this LOADCH. The remake JOIN beat records membership
		// and chronology, so seed only records that are still absent from the
		// typed roster. Direct/debug LOADCH replay has no JOIN history and must
		// not silently manufacture persistent campaign state.
		if len(g.partyJoinOrder) != 0 {
			g.initializeEquipmentBases(&battle.State{Units: party})
			if err := g.seedPersistentPartyFromLoadCH(partyOrder, party); err != nil {
				return fmt.Errorf("party scenario %q persistence: %w", state.PartyScenario, err)
			}
		}
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
	g.storyCompositionEventBytes = append(g.storyCompositionEventBytes[:0], roster.NativeCompositionEventBytes...)
	g.storyRosterPath = state.Roster
	g.storyPartyScenario = state.PartyScenario
	g.storyWalks = nil
	g.storyBG = true
	g.walkFirst, g.followWalk = false, false
	g.camMaxY = float64(state.CamMaxY)
	g.camX, g.camY = float64(state.CamX), float64(state.CamY)
	g.storyNativeMapView = loadCHView
	g.hasStoryNativeMapView = true
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
	if g.dlgScrollT > 0 { // 捲動尚未完成時，Enter 不得跳過下一頁
		return false
	}
	if len(g.dialog) > 0 && g.dlgPage+1 < dlgPageCount(g.dialog[len(g.dialog)-1]) {
		g.dlgScrollFrom = g.dlgPage
		g.dlgPage++
		g.dlgScrollT = dlgScrollFrames
		return false
	}
	if len(g.dialog) > 0 {
		g.dialog = g.dialog[:len(g.dialog)-1]
	}
	g.dlgPage = 0
	g.dlgScrollT = 0
	g.dlgScrollFrom = 0
	return true
}

// ── 對話框切換動畫(使用者回饋 2026-07-04 #3:換人說話時框先收合再展開)──
const dlgNone = -999       // dlgShown 無框哨兵
const dlgAnimFrames = 7    // 縮/展各自幀數(~0.12s@60fps)
const dlgScrollFrames = 10 // 分頁文字往上捲動幀數(~0.17s@60fps)

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
	g.captureNativeMapHUDPersistence()
	g.resetActionOverlayLifecycle()
	g.nativeEnding = nil
	g.endingNotice = ""
	n := g.camp.Node()
	if n == nil {
		return // 流程結束(game over)
	}
	// campaign 結局尚無已證實的場景→曲目對映；空白 ending BGM 先停止前一場景，
	// 避免戰鬥音樂漏到終局頁。若資料明確填入 BGM，仍保留可編輯的曲目入口，
	// 待證據閉合後即可使用。
	if n.Type == "ending" && n.BGM == "" {
		g.stopBGM()
	} else {
		g.playBGM(n.BGM)
	}
	g.storyBG = false // 預設離開場景背景模式;story+Map 節點下面再開回
	g.storyWalks = nil
	g.storyAutoAdvance = 0
	g.walkFirst, g.followWalk, g.camMaxY = false, false, 0
	g.storyNativeMapView = battle.NativeMapViewState{}
	g.hasStoryNativeMapView = false
	g.camPan, g.focusJob, g.actJob, g.beats, g.beatIdx, g.beatDelay = nil, nil, nil, nil, -1, 0
	g.transitionReveal = nil
	g.indexedTransition = nil
	g.nativeCh20SkyKey = nil
	g.nativeCh23State = nil
	g.nativeCh23Loop = nil
	g.native2189A = nil
	g.nativeUnitPresent = nil
	g.nativeCh28PostPresent = nil
	g.nativeCh22Reload = nil
	g.spawnIntroTransition = nil
	g.nativeTurnStaging = nil
	g.nativeFullDACWhite = false
	g.nativeFullDACBlack = false
	g.handlerResource = 0
	g.battleEvent, g.battleEventDelay = nil, 0
	g.dlgShown, g.dlgPhase, g.dlgT = dlgNone, 0, 0
	g.dlgUpper = nil
	g.dlgScrollT, g.dlgScrollFrom = 0, 0
	switch n.Type {
	case "story", "cutscene": // cutscene(doc50):同一套場景設置,進行中改由 Beats 驅動(見下)
		g.dialog = nil
		g.dlgPage = 0 // 新對白從第一頁起
		lines := n.Lines
		script := n.Script
		if script == "" && n.Type == "story" {
			script = defaultChapterStoryScript(g.camp.NodeID())
		}
		if script != "" { // 本機劇情文本檔(assets/story/chNN.json,人工精校;無檔 fallback 內嵌 lines)
			if ls := loadStoryScript(script, n.Scene); len(ls) > 0 {
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
			g.storyRosterPath, g.storyPartyScenario = "", ""
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
			g.storyRosterPath, g.storyPartyScenario = "", ""
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
			// An unbound postbattle node must never be treated as an empty
			// interlude: doing so would silently skip persistent sync/rewards and
			// jump straight to town. Keep the authored graph cursor in place until
			// an active handler binding is supplied.
			if len(g.beats) == 0 && n.HandlerBinding == "" && strings.HasPrefix(g.camp.NodeID(), "postbattle_") {
				if g.approximateMode {
					// 此節點的原版 handler 仍未知。明確啟用的近似模式只保留玩家可見的
					// 戰役邊界，不捏造 JOIN／獎勵語意：只同步已物化的戰場隊伍，等待
					// Enter 後才沿 authored Next 邊進入城鎮／整備。
					if err := g.syncPartyFromBattle(); err != nil {
						g.loadErr = fmt.Sprintf("approximate postbattle %q: party sync unavailable: %v", g.camp.NodeID(), err)
						return
					}
					g.approximatePostbattle = true
					// 這是未綁定原版 handler 時才允許的後備流程，只維持 authored
					// 戰役邊界；不得用來宣稱該章戰後 handler 或一般玩家路徑已達 E2。
					g.msg = "戰後整理（近似模式；原版 handler 尚未證實）按 Enter 繼續"
					return
				}
				g.loadErr = fmt.Sprintf("postbattle node %q has no active handler binding", g.camp.NodeID())
				g.msg = "戰後 handler 尚未接線，流程已停止"
				return
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
		if !g.materializeNativeMapRuntime(n) {
			return
		}
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
		if n.Type == "town" {
			g.resetNativeTownUIPulse()
		}
	case "preparation", "church":
		g.dialog, g.st, g.sel = nil, nil, nil
		// 節點邊界 UI；preparation 可在此安全 F5 存檔，Enter 才進下一章 pre handler。
		if n.Type == "preparation" {
			g.setupPreparation(n)
		} else {
			g.setupChurch()
		}
	case "hotel":
		g.dialog, g.st, g.sel = nil, nil, nil
		g.hotelSel = 0
		g.hotelRoute = fdother.NativeHotelServiceRoute{}
		g.hotelHasRoute = false
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
		g.nativeShopUIJob = nil
		g.nativeShopVariant = 0
		g.nativeShopMode = ""
		g.nativeShopServiceSel = 0
		g.nativeShopItemStart = 0
		g.nativeShopEquipRosterTop = 0
		g.nativeShopEquipUnitSel = 0
		g.nativeShopTransferSource = -1
		g.nativeShopTransferItem = -1
		g.nativeShopTransferItems = nil
		g.nativeShopTransferDest = -1
		g.nativeShopTransferIDs = nil
		g.nativeShopTransferSel = 0
		g.nativeShopTransferTop = 0
		g.clearNativeItemPanel()
		g.setupNativeShop()
	case "ending":
		g.dialog, g.st, g.sel = nil, nil, nil
		if n.NativeEndingPrefix != nil {
			if err := g.startCampaignNativeEnding(n.NativeEndingPrefix); err != nil {
				// 原始結局資源是玩家自備且不隨專案散布；載入失敗時保留
				// 可編輯結語，不發布部分原生演出。
				g.endingNotice = "原始結局素材不足，顯示可編輯結語。"
			}
		}
	}
}

func (g *Game) materializeNativeMapRuntime(n *campaign.Node) bool {
	if n == nil || (n.NativeMapView == nil && n.NativeMapHUD == nil && n.NativeMapHUDInherited == nil) {
		return true
	}
	if g.st == nil {
		g.loadErr = "native map runtime: battle state is unavailable"
		return false
	}
	if n.NativeMapView == nil {
		g.loadErr = "native map runtime: HUD lacks a sourced view"
		return false
	}
	// Preflight into a narrow candidate so an invalid optional HUD cannot leave
	// a published view/range half-state. A sourced view may stand alone: ch27's
	// handler closes the six view globals, while gate A remains inherited from
	// a separate persistent option and must not be fabricated.
	candidate := &battle.State{W: g.st.W, H: g.st.H}
	view := n.NativeMapView
	if err := candidate.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CameraX: view.CameraX, CameraY: view.CameraY,
		CursorX: view.CursorX, CursorY: view.CursorY,
		VisibleCursorX: view.VisibleCursorX, VisibleCursorY: view.VisibleCursorY,
	}); err != nil {
		g.loadErr = "native map runtime view: " + err.Error()
		return false
	}
	if view.RangeMode == nil || (*view.RangeMode != 0 && *view.RangeMode != 1) ||
		!candidate.MaterializeNativeMapRangeMode(*view.RangeMode) {
		g.loadErr = "native map campaign selector is not a verified entry value"
		return false
	}
	if hud := n.NativeMapHUD; hud != nil {
		if hud.DisplayGateA < 0 || hud.DisplayGateA > 0xff ||
			hud.DisplayGateB < 0 || hud.DisplayGateB > 0xff ||
			!candidate.MaterializeNativeMapHUDState(byte(hud.DisplayGateA), byte(hud.DisplayGateB), hud.AnchorX) {
			g.loadErr = "native map runtime HUD is outside raw bounds"
			return false
		}
	} else if inherited := n.NativeMapHUDInherited; inherited != nil {
		persistentSource := g.nativeMapHUDPersistent
		// A wholly uninitialized Game still represents a fresh original process,
		// whose data image seeds gate A and anchor to 1. A partial state is not a
		// fresh process and remains rejected rather than filling one missing field.
		if !persistentSource.HasDisplayGateA && !persistentSource.HasAnchorX {
			persistentSource = battle.InitialNativeMapHUDPersistentState()
		}
		hud, ok := persistentSource.MaterializeRuntime(byte(inherited.DisplayGateB))
		if !ok || !candidate.MaterializeNativeMapHUDState(hud.DisplayGateA, hud.DisplayGateB, hud.AnchorX) {
			g.loadErr = "native map runtime HUD persistent state is incomplete"
			return false
		}
	}
	persistentCandidate := g.nativeMapHUDPersistent
	if candidate.HasNativeMapHUDState &&
		!persistentCandidate.CaptureNativeMapHUD(candidate.NativeMapHUDState) {
		g.loadErr = "native map runtime HUD persistent capture failed"
		return false
	}
	g.st.NativeMapViewState = candidate.NativeMapViewState
	g.st.HasNativeMapViewState = true
	g.st.NativeMapRangeMode = candidate.NativeMapRangeMode
	g.st.HasNativeMapRangeModeState = true
	g.st.NativeMapHUDState = candidate.NativeMapHUDState
	g.st.HasNativeMapHUDState = candidate.HasNativeMapHUDState
	g.nativeMapHUDPersistent = persistentCandidate
	g.syncNativeMapView()
	return true
}

// captureNativeMapHUDPersistence runs before a campaign node can clear or
// replace the previous battle state. The original carries gate A and anchor
// across nodes, but not the transient gate B.
func (g *Game) captureNativeMapHUDPersistence() {
	if g == nil || g.st == nil || !g.st.HasNativeMapHUDState {
		return
	}
	g.nativeMapHUDPersistent.CaptureNativeMapHUD(g.st.NativeMapHUDState)
}

// resetBattle 重開一場戰鬥(campaign battle 節點;敗北重試也走這裡)。
func (g *Game) resetBattle(unitsPath, scnPath string) {
	g.resetActionOverlayLifecycle()
	g.nativeContinueOpeningConfirm = false
	g.nativeSystemCursorOverlay = false
	g.nativeCh20SkyKey = nil
	g.nativeCh23State = nil
	g.nativeCh23Loop = nil
	g.native2189A = nil
	g.nativeUnitPresent = nil
	g.nativeCh28PostPresent = nil
	g.nativeCh22Reload = nil
	g.indexedTransition = nil
	g.spawnIntroTransition = nil
	g.nativeTurnStaging = nil
	g.nativeFullDACWhite = false
	g.nativeFullDACBlack = false
	g.nativeMapClock.Reset()
	g.nativeMapWork, g.nativeMapVGA = nil, nil
	if unitsPath == "" {
		unitsPath = "assets/map0_units.json"
	}
	// A verified pre-handler LOADCH owns the same runtime array used by the
	// following battle. Preserve it only when both editable source paths match
	// exactly; direct/debug starts and unrelated scene maps still rebuild.
	adoptHandlerState := len(g.storyActors) > 0 &&
		g.storyRosterPath == unitsPath &&
		g.storyPartyScenario == scnPath
	handlerActors := g.storyActors
	handlerRoster := g.storyRoster
	g.storyActors, g.storyRoster, g.storySpawned = nil, nil, nil
	g.storyCompositionEventBytes = nil
	g.storyRosterPath, g.storyPartyScenario = "", ""
	if st, err := battle.Load(assetPath(unitsPath)); err == nil {
		g.st = st
		if err := g.bindNativeFutureItemRows(st); err != nil {
			g.loadErr = "native future constructor item rows: " + err.Error()
			return
		}
		if err := g.bindNativeMovementCostRows(st); err != nil {
			g.loadErr = "native movement rows: " + err.Error()
			return
		}
		g.bindCommandLearn(st)
		g.bindNativeCommandBook(st)
		g.bindNativeCommandResistances(st)
		if adoptHandlerState {
			st.Units = nil
			st.Roster = make([]*battle.Unit, len(handlerRoster))
			for i := range handlerRoster {
				st.Roster[i] = &handlerRoster[i]
			}
			st.NativeMapSelectorCache = nil
			st.NativeMapSelectorError = nil
			actors := make([]*battle.Unit, len(handlerActors))
			for i := range handlerActors {
				actors[i] = &handlerActors[i]
			}
			if err := st.AppendNativeMapSelectorBatch(actors); err != nil {
				g.loadErr = "handler battle roster: " + err.Error()
				return
			}
		}
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
			if adoptHandlerState && !sc.RuntimeAppendGroups {
				// Matching asset paths alone do not prove that the following battle
				// consumes the handler's partial runtime array.  Without the
				// explicit runtime-append contract, rebuild the authored battle
				// state and keep the boundary fail-closed rather than calling
				// AdoptHandlerBattleState on an incompatible scenario.
				adoptHandlerState = false
			}
			g.sc = sc
			if adoptHandlerState {
				if err := sc.AdoptHandlerBattleState(g.st); err != nil {
					g.loadErr = "handler battle scenario: " + err.Error()
					return
				}
			} else {
				dialogue, err := sc.SetupChecked(g.st)
				if err != nil {
					g.loadErr = "scenario setup: " + err.Error()
					return
				}
				g.dialog = append(g.dialog, dialogue...)
			}
			g.initializeEquipmentBases(g.st)
			g.applyScenarioPartyJoins()
			g.applyPersistentParty(g.st)
			g.focusOnParty()
		}
	}
}

// bindNativeCommandBook gives each freshly loaded battle the immutable raw
// ABI table.  It never derives records from the legacy SpellBook.
func (g *Game) bindNativeCommandBook(st *battle.State) {
	if st == nil {
		return
	}
	st.NativeCommandBook = append([]battle.NativeCommandRecord(nil), g.nativeCommandBook...)
}

// bindNativeMovementCostRows 將有機會進入 AI runner 的每個 battle state 綁定
// 版本化 0x4e555 表。缺少或格式錯誤時，在 mode 2 窄切片嘗試以正規化移動成本
// 取代原始成本前先停止。
func (g *Game) bindNativeMovementCostRows(st *battle.State) error {
	if st == nil {
		return fmt.Errorf("native movement rows: nil battle state")
	}
	rows, err := battle.LoadNativeMovementCostRows(
		assetPath("assets/data/native_movement_cost_rows.json"),
	)
	if err != nil {
		return err
	}
	return st.BindNativeMovementCostRows(rows)
}

func (g *Game) bindNativeCommandResistances(st *battle.State) {
	if st == nil || len(g.nativeCommandResistances) == 0 {
		return
	}
	st.NativeCommandResistances = make(map[int]int, len(g.nativeCommandResistances))
	for classID, raw := range g.nativeCommandResistances {
		st.NativeCommandResistances[classID] = raw
	}
}

// bindCommandLearn makes every newly loaded battle state use the same
// explicit editable export. A missing table leaves the state fail-closed;
// it never falls back to the legacy normalized Spells list.
func (g *Game) bindCommandLearn(st *battle.State) {
	if st != nil && g.commandLearn != nil {
		st.CommandLearn = g.commandLearn
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

// seedPersistentPartyFromLoadCH materializes the remake counterpart of the
// persistent records already created by preceding native JOIN calls. Existing
// records always win, preserving battle/shop/class-change progress. This is a
// normal-campaign bridge, not FD2.SAV byte compatibility.
func (g *Game) seedPersistentPartyFromLoadCH(
	order []int,
	units []*battle.Unit,
) error {
	if len(order) == 0 || len(units) != len(order) {
		return fmt.Errorf(
			"LOADCH party/order length mismatch: units=%d order=%d",
			len(units), len(order),
		)
	}
	if g.partyRoster == nil {
		g.partyRoster = make(map[int]battle.Unit, len(order))
	}
	for i, id := range order {
		unit := units[i]
		if unit == nil || unit.Fig != id || !g.partyMembers[id] {
			return fmt.Errorf(
				"LOADCH party slot %d does not match joined identity %d",
				i, id,
			)
		}
		if _, exists := g.partyRoster[id]; exists {
			continue
		}
		materialized, err := g.materializeNativeJoinPersistentUnit(id, *unit)
		if err != nil {
			return err
		}
		g.partyRoster[id] = cloneNativeShopUnit(materialized)
	}
	return nil
}

func (g *Game) materializeNativeJoinPersistentUnit(id int, base battle.Unit) (battle.Unit, error) {
	if g == nil {
		return battle.Unit{}, fmt.Errorf("native JOIN constructor owner is unavailable")
	}
	if !g.hasNativeJoinConstructor {
		table, err := campaign.LoadNativeJoinConstructorTable(assetPath("assets/data/native_join_constructor.json"))
		if err != nil {
			return battle.Unit{}, fmt.Errorf("native JOIN constructor table: %w", err)
		}
		g.nativeJoinConstructor = table
		g.hasNativeJoinConstructor = true
	}
	if len(g.nativeJoinItemEffectRows) == 0 {
		rows, err := battle.LoadNativeItemEffectRowPrefix(
			assetPath("assets/data/native_item_effect_rows.json"),
		)
		if err != nil {
			return battle.Unit{}, fmt.Errorf("native JOIN item effect rows: %w", err)
		}
		g.nativeJoinItemEffectRows = rows
	}
	return g.nativeJoinConstructor.MaterializePersistentUnit(
		id, base, g.nativeJoinItemEffectRows,
	)
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
	dst.Portrait, dst.Fig, dst.BattleFig = src.Portrait, src.Fig, src.BattleFig
	dst.MapSelectorKey, dst.HasMapSelectorKey = src.MapSelectorKey, src.HasMapSelectorKey
	dst.Exp, dst.ExpPerLevel = src.Exp, src.ExpPerLevel
	dst.Spells = append(dst.Spells[:0], src.Spells...)
	dst.NativeCommandMask = src.NativeCommandMask
	dst.NativeIdentity, dst.HasNativeIdentity = src.NativeIdentity, src.HasNativeIdentity
	dst.NativeRecordRace, dst.HasNativeRecordRace = src.NativeRecordRace, src.HasNativeRecordRace
	dst.NativeRecordClass, dst.HasNativeRecordClass = src.NativeRecordClass, src.HasNativeRecordClass
	dst.NativeTransient = src.NativeTransient
	dst.NativeRecordByte5, dst.HasNativeRecordByte5 = src.NativeRecordByte5, src.HasNativeRecordByte5
	dst.NativeRecordByte6, dst.HasNativeRecordByte6 = src.NativeRecordByte6, src.HasNativeRecordByte6
	// +0x42 is a raw persistent word used by ch15_post; preserve it only when
	// the source carries explicit provenance, never derive it from normalized HP.
	dst.NativeRecordWord42, dst.HasNativeRecordWord42 = src.NativeRecordWord42, src.HasNativeRecordWord42
	dst.NativeRecordWord46, dst.HasNativeRecordWord46 = src.NativeRecordWord46, src.HasNativeRecordWord46
	dst.Inventory = append(dst.Inventory[:0], src.Inventory...)
	dst.Equipped = append(dst.Equipped[:0], src.Equipped...)
	dst.InventorySlots = append(dst.InventorySlots[:0], src.InventorySlots...)
	dst.NativeInventoryFlags = append(dst.NativeInventoryFlags[:0], src.NativeInventoryFlags...)
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

// partyHasItemID is the campaign projection used when only normalized Unit
// inventory values are available: it searches runtime slots 0..15 without a
// camp/activity filter, then falls back to the persistent roster at a
// node-boundary save/load. It is deliberately not called byte-identical to
// native 0x24b14: that routine first counts raw flag-bit7-clear cells and only
// scans the resulting prefix. battle.FindNativeInventoryItem is the exact raw
// adapter; this projection remains a compatibility path until runtime raw
// records are carried through the campaign boundary.
func (g *Game) partyHasItemID(itemID int) bool {
	if g.st != nil {
		if itemID >= 0 && itemID <= 0xff {
			if records, err := battle.NativeInventoryRecords(g.st.Units, 16); err == nil {
				unit, _, searchErr := battle.FindNativeInventoryItem(records, byte(itemID))
				return searchErr == nil && unit >= 0
			}
		}
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
// matches persistent records by raw +0x08 identity before copying its full
// 0x50-byte runtime record. When optional raw identities are present, this
// projection uses that exact key; records with an unknown raw key are skipped
// (fail closed). Records without raw identity retain the legacy Fig projection.
// It clears transient state/path bytes, restores
// active survivors to full HP and restores everyone's MP. Defeated/inactive
// members retain their zero HP. The projection snapshots JOIN member 0 as
// compatibility behavior; it is not byte-identical proof of native 0x11506.
func (g *Game) syncPartyFromBattle() error {
	_, err := g.syncPartyFromBattleRecords()
	return err
}

// syncPartyFromBattleRecords 回傳實際發布到持續隊伍的紀錄（record）數。一般戰後
// 呼叫維持既有相容行為；終局來源約束邊界另要求至少一筆，避免所有原始身分
// （raw identity）都不相符時帶著舊名冊靜默進入角色蒙太奇。
func (g *Game) syncPartyFromBattleRecords() (int, error) {
	if g.st == nil {
		return 0, fmt.Errorf("缺少已完成的戰場狀態")
	}
	if g.partyRoster == nil {
		g.partyRoster = make(map[int]battle.Unit)
	}
	synced := 0
	for _, current := range g.st.Units {
		if current == nil {
			continue
		}
		id := current.Fig
		rawIdentity, hasRawIdentity := current.NativeIdentity, current.HasNativeIdentity
		if !hasRawIdentity && current.HasNativeRecordByte8 {
			rawIdentity, hasRawIdentity = int(current.NativeRecordByte8), true
		}
		if hasRawIdentity {
			matched := false
			for rosterID, roster := range g.partyRoster {
				if roster.HasNativeIdentity && roster.NativeIdentity == rawIdentity {
					id, matched = rosterID, true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if len(g.partyMembers) != 0 {
			if !g.partyMembers[id] {
				continue
			}
		} else if current.Camp != battle.Own {
			continue
		}
		snapshot := *current
		if hasRawIdentity {
			snapshot.NativeIdentity, snapshot.HasNativeIdentity = rawIdentity, true
		}
		snapshot.Spells = append([]int(nil), current.Spells...)
		snapshot.Inventory = append([]int(nil), current.Inventory...)
		snapshot.Equipped = append([]bool(nil), current.Equipped...)
		snapshot.InventorySlots = append([]int(nil), current.InventorySlots...)
		snapshot.NativeInventoryFlags = append([]int(nil), current.NativeInventoryFlags...)
		if snapshot.MaxMP < snapshot.MP {
			snapshot.MaxMP = snapshot.MP
		}
		if snapshot.HasNativeRecordByte5 {
			if snapshot.NativeRecordByte5&1 == 0 {
				snapshot.HP = snapshot.MaxHP
			}
		} else if snapshot.OnField && snapshot.Alive() {
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
		synced++
	}
	return synced, nil
}

// resetPersistentRosterState mirrors native 0x25089, which is separate from
// the battle-to-roster projection.  The routine clears the persistent status
// byte and copies each member's max HP/MP into the current values before the
// next town/preparation or ending branch.
func (g *Game) resetPersistentRosterState() {
	for id, u := range g.partyRoster {
		u.Acted = false
		u.HP = u.MaxHP
		u.MP = u.MaxMP
		u.OffX, u.OffY = 0, 0
		u.BuffAPPct, u.BuffDPPct, u.BuffHit, u.BuffEV, u.BuffTurns = 0, 0, 0, 0, 0
		u.Sealed, u.SealTurns = false, 0
		u.Poisoned, u.PoisonTurns = false, 0
		u.Paralyzed, u.ParalyzeTurns = false, 0
		g.partyRoster[id] = u
	}
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
		if _, exists := g.partyRoster[id]; exists || g.st == nil {
			continue
		}
		var joined *battle.Unit
		for _, unit := range g.st.Units {
			// Native JOIN establishes permanent membership independently of the
			// actor's current camp colour. Many recruits are still Ally here.
			if unit != nil && unit.Fig == id {
				if joined != nil {
					g.loadErr = fmt.Sprintf("scenario join_party:角色%d有多筆場上記錄", id)
					joined = nil
					break
				}
				joined = unit
			}
		}
		if joined == nil {
			if g.loadErr == "" {
				g.loadErr = fmt.Sprintf("scenario join_party:找不到角色%d的我方記錄", id)
			}
			continue
		}
		g.initializeEquipmentBases(&battle.State{Units: []*battle.Unit{joined}})
		if g.partyRoster == nil {
			g.partyRoster = make(map[int]battle.Unit)
		}
		materialized, err := g.materializeNativeJoinPersistentUnit(id, *joined)
		if err != nil {
			g.loadErr = fmt.Sprintf("scenario join_party:角色%d persistent record: %v", id, err)
			continue
		}
		g.partyRoster[id] = cloneNativeShopUnit(materialized)
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
		// sub_320FC keeps persistent record zero fixed and reorders only
		// selection-table entry i -> persistent record i+1. partyDeploy stores
		// those selectable flags, so add the fixed record without mutating the
		// editable selection set itself.
		members := make(map[int]bool, len(g.partyDeploy)+1)
		for id, selected := range g.partyDeploy {
			if selected {
				members[id] = true
			}
		}
		if len(g.partyJoinOrder) != 0 {
			members[g.partyJoinOrder[0]] = true
		}
		return members
	}
	return g.partyMembers
}

func (g *Game) setupPreparation(n *campaign.Node) {
	// 0x318AD exposes [0x53BFB]-1 flags. sub_320FC maps flag i to
	// persistent record i+1 and always leaves record zero in slot zero.
	// The fixed leader therefore must not consume one of the 15/19 choices.
	g.prepIDs = g.prepIDs[:0]
	if len(g.partyJoinOrder) > 1 {
		g.prepIDs = append(g.prepIDs, g.partyJoinOrder[1:]...)
	}
	seen := make(map[int]bool, len(g.prepIDs))
	if len(g.partyJoinOrder) != 0 {
		seen[g.partyJoinOrder[0]] = true
	}
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
	g.prepSelecting = false
	g.prepConfirm = false
	g.prepConfirmSel = 0
	g.nativeClassUIJob = nil
	g.resetNativeClassUIPulse()
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
	// 0x318c7 calls memset(flags, 0, 30) on every entry. A previous battle's
	// deployment and the remake save projection therefore cannot preselect
	// this native selection pass.
	g.partyDeploy = make(map[int]bool)
	if n == nil || n.Cancel == "" {
		// 0x2cc04 clears the complete VGA target before the standalone
		// FDTXT 0x19a record prompt.
		g.prepPromptSource = make([]byte, 320*200)
	} else if len(g.prepPromptSource) != 320*200 {
		// A town-backed prompt must retain its actual town frame; never
		// substitute a black or modern background for missing provenance.
		g.prepPromptSource = nil
	}
	g.beginNativePreparationPromptOpening()
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

// acceptTownDeparturePrompt reproduces the 0x2d13d..0x2d161 caller gate.
// prepIDs models the selectable records (native [0x53bfb]-1); persistent
// record zero is fixed outside this list. At most cap selectable records skip
// 0x318ad and depart immediately. Larger rosters enter the zero-initialized
// selection pass.
func (g *Game) acceptTownDeparturePrompt() bool {
	if len(g.prepIDs) <= g.prepLimit {
		return true
	}
	g.restartPreparationSelection()
	return false
}

func (g *Game) restartPreparationSelection() {
	g.prepSelecting = true
	g.prepConfirm = false
	g.prepConfirmSel = 0
	g.nativeClassUIJob = nil
	g.resetNativeClassUIPulse()
	g.prepPromptSource = nil
	g.partyDeploy = make(map[int]bool)
}

func (g *Game) setupChurch() {
	g.nativeClassUIJob = nil
	g.nativeChurchUIJob = nil
	g.churchSel = 0
	g.churchMode = "menu"
	g.churchIDs = nil
	g.churchRosterStart = 0
	g.churchVerticalStart = 0
	g.churchTransferSource = -1
	g.churchTransferItem = -1
	g.churchTransferItems = nil
	g.churchTransferDest = -1
	g.churchItemStart = 0
	g.churchReviveID = -1
	g.churchReviveFee = 0
	g.churchClassID = -1
	g.churchStatusID = -1
	g.churchStatusPanel = nil
	g.churchCommandPanel = nil
	g.churchBranches = nil
	g.nativeChurchTextIndex = 585
	g.beginNativeChurchMenuOpening()
}

func (g *Game) churchTransferSourceIDs() []int {
	// The native branch keeps every occupied cell whose signed flag is
	// non-negative (including 0x40 equipped cells); only 0x80 reserved cells
	// are excluded. Raw constructor flags take precedence over projections.
	ids := make([]int, 0)
	for _, id := range g.partyJoinOrder {
		u, ok := g.partyRoster[id]
		if !ok {
			continue
		}
		if len(u.Equipped) != len(u.Inventory) {
			continue
		}
		for i := range u.Inventory {
			eligible := true
			if i < len(u.Equipped) {
				// Legacy JSON has no raw flags; preserve its conservative projection
				// until source provenance is available.
				eligible = !u.Equipped[i]
			}
			if len(u.NativeInventoryFlags) == 8 && len(u.InventorySlots) == 8 {
				var err error
				eligible, err = battle.NativeInventoryCompactEligible(u.NativeInventoryFlags, u.InventorySlots, i)
				if err != nil {
					eligible = false
				}
			}
			if eligible {
				ids = append(ids, id)
				break
			}
		}
	}
	return ids
}

func (g *Game) churchRosterIDs() []int {
	ids := make([]int, 0, len(g.partyJoinOrder))
	for _, id := range g.partyJoinOrder {
		if _, ok := g.partyRoster[id]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func (g *Game) churchTransferItemSlots(id int) []int {
	u, ok := g.partyRoster[id]
	if !ok {
		return nil
	}
	if len(u.Equipped) != len(u.Inventory) {
		return nil
	}
	slots := make([]int, 0)
	for i := range u.Inventory {
		eligible := true
		if i < len(u.Equipped) {
			eligible = !u.Equipped[i]
		}
		if len(u.NativeInventoryFlags) == 8 && len(u.InventorySlots) == 8 {
			var err error
			eligible, err = battle.NativeInventoryCompactEligible(u.NativeInventoryFlags, u.InventorySlots, i)
			if err != nil {
				return nil
			}
		}
		if eligible {
			slots = append(slots, i)
		}
	}
	return slots
}

func (g *Game) churchTransferDestinationIDs(_ int) []int {
	ids := make([]int, 0, len(g.partyJoinOrder))
	for _, id := range g.partyJoinOrder {
		if _, ok := g.partyRoster[id]; ok {
			ids = append(ids, id)
		}
	}
	return ids
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
// 「鏡頭對準部隊」的合理預設,不是重現原版鏡頭運鏡（原版 0x3231b 以 0x13185/0x135dd
// 平移攝影機，再由 0x32999 對新增群組做索引轉場；兩者都不是單位行走，且未對主角隊做；
// DOSBox 複驗全序章無任何單位行走動畫）。
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

// continueApproximatePostbattle 在明確近似提示後只沿 authored graph 邊前進；
// 不合成原版 JOIN、獎勵、章節值或 handler 分支。
func (g *Game) continueApproximatePostbattle() bool {
	if g == nil || !g.approximatePostbattle || g.camp == nil {
		return false
	}
	g.approximatePostbattle = false
	g.msg = ""
	g.camp.Advance("")
	g.enterNode()
	return true
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
		if g.approximatePostbattle {
			if enter {
				g.continueApproximatePostbattle()
			}
			return true
		}
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
		nativeTown := n.Type == "town" && n.NativeTownVariant != nil
		if nativeTown {
			if scanCode, ok := pressedNativeTownSecretScan(); ok &&
				g.revealNativeTownSecret(scanCode) {
				return true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				g.moveNativeTownSelection(-1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				g.moveNativeTownSelection(1)
			}
			if enter {
				if g.camp.ConfirmNativeTownSecret(g.campSel) {
					g.enterNode()
					return true
				}
				if g.campSel >= 0 && g.campSel < 5 {
					if source, ok := g.composeNativeTownFrame(); ok {
						g.prepPromptSource = append([]byte(nil), source...)
					} else {
						g.prepPromptSource = nil
					}
					g.camp.Advance(fmt.Sprintf("opt%d", g.campSel))
					g.enterNode()
				}
			}
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.stepCampaignMenu(campaign.MenuUp)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.stepCampaignMenu(campaign.MenuDown)
		}
		selected, confirm := g.stepCampaignMenu(campaign.MenuTick)
		if enter {
			selected, confirm = g.stepCampaignMenu(campaign.MenuConfirm)
		}
		if confirm {
			g.camp.Advance(fmt.Sprintf("opt%d", selected))
			g.enterNode()
		}
		return true
	case "preparation", "church":
		if n.Type == "preparation" {
			if g.nativeClassUIBlocksInput() {
				return true
			}
			townBacked := n.Cancel != ""
			leavePreparation := func(outcome string) {
				if g.camp.Advance(outcome) != "" {
					g.enterNode()
				}
			}
			if g.prepConfirm {
				closeThen := func(after func()) {
					if !g.beginNativePreparationConfirmationClosing(after) {
						after()
					}
				}
				if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
					g.prepConfirmSel ^= 1
					g.resetNativeClassUIPulse()
				}
				if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
					if townBacked {
						closeThen(func() { leavePreparation("cancel") })
					} else {
						closeThen(g.restartPreparationSelection)
					}
				}
				if enter {
					if g.prepConfirmSel == 0 {
						closeThen(func() { leavePreparation("confirm") })
					} else if townBacked {
						closeThen(func() { leavePreparation("cancel") })
					} else {
						closeThen(g.restartPreparationSelection)
					}
				}
				return true
			}
			if !g.prepSelecting {
				closeThen := func(after func()) {
					if !g.beginNativePreparationPromptClosing(after) {
						after()
					}
				}
				if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
					g.prepConfirmSel ^= 1
					g.resetNativeClassUIPulse()
				}
				if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
					if townBacked {
						closeThen(func() {
							g.prepPromptSource = nil
							leavePreparation("cancel")
						})
					} else {
						closeThen(g.restartPreparationSelection)
					}
					return true
				}
				if enter {
					if !townBacked {
						closeThen(func() {
							if g.prepConfirmSel == 0 {
								g.saveGame()
							}
							g.restartPreparationSelection()
						})
					} else if g.prepConfirmSel != 0 {
						closeThen(func() {
							g.prepPromptSource = nil
							leavePreparation("cancel")
						})
					} else {
						closeThen(func() {
							if g.acceptTownDeparturePrompt() {
								g.prepPromptSource = nil
								leavePreparation("confirm")
							}
						})
					}
				}
				return true
			}
			movePreparation := func(scanCode byte) {
				if next, err := fdother.MoveNativePreparationRosterCursor(
					g.prepSel, len(g.prepIDs), scanCode,
				); err == nil {
					g.prepSel = next
				}
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				movePreparation(0x4b)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				movePreparation(0x4d)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
				movePreparation(0x48)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
				movePreparation(0x50)
			}
			if enter && len(g.prepIDs) > 0 {
				id := g.prepIDs[g.prepSel]
				g.partyDeploy[id] = !g.partyDeploy[id]
				// 0x31a68 exits the selection loop once its 0x0f/0x13 quota is
				// met, but 0x31d3c..0x31db4 still presents final confirmation.
				if g.preparationSelected() == g.prepLimit {
					g.prepSelecting = false
					g.prepConfirm = true
					g.prepConfirmSel = 0
					g.beginNativePreparationConfirmationOpening()
				}
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				if townBacked {
					leavePreparation("cancel")
				} else {
					g.restartPreparationSelection()
				}
			}
			return true
		}
		if g.nativeClassUIBlocksInput() || g.nativeChurchUIBlocksInput() {
			return true
		}
		if g.churchMode == "menu" {
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				g.churchSel = campaign.AdvanceNativeChurchServiceSelection(g.churchSel, -1)
				g.resetNativeChurchUIPulse()
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				g.churchSel = campaign.AdvanceNativeChurchServiceSelection(g.churchSel, 1)
				g.resetNativeChurchUIPulse()
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				if !g.beginNativeChurchMenuClosing(g.leaveChurch) {
					g.leaveChurch()
				}
				return true
			}
			if enter {
				selected := g.churchSel
				openService := func() {
					switch selected {
					case 0: // 0x2ffa5 caller-owned roster → 0x17aed(actor)
						g.churchMode = "status_roster"
						g.churchIDs = g.churchRosterIDs()
						g.churchSel = 0
						g.churchRosterStart = 0
						g.beginNativeChurchRosterOpening()
					case 1: // native 0x2f8ea raw source→destination inventory transfer
						g.churchMode = "transfer_source"
						g.churchIDs = g.churchTransferSourceIDs()
						g.churchSel = 0
						g.churchRosterStart = 0
						g.nativeChurchTextIndex = 512
						g.beginNativeChurchRosterOpening()
					case 2, 3: // native 0x30dc3 revive / 0x31385 class-change services
						g.churchMode = map[int]string{2: "revive", 3: "class"}[selected]
						g.churchIDs = g.churchCandidates(g.churchMode)
						g.churchSel = 0
						g.churchVerticalStart = 0
						if g.churchMode == "class" {
							g.beginNativeClassListOpening()
						} else if len(g.churchIDs) == 0 {
							g.openNativeChurchReviveEmpty()
						} else {
							g.nativeChurchTextIndex = 589
							g.beginNativeChurchReviveListOpening()
						}
					default:
						g.msg = "此教會服務尚待原版 callee 完整接線"
						g.returnToNativeChurchMenu()
					}
				}
				if !g.beginNativeChurchMenuClosing(openService) {
					openService()
				}
			}
			return true
		}
		if g.churchMode == "status_roster" {
			listLen := len(g.churchIDs)
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				g.churchSel = campaign.AdvanceNativeTwoColumnSelection(g.churchSel, listLen, -1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				g.churchSel = campaign.AdvanceNativeTwoColumnSelection(g.churchSel, listLen, 1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
				g.churchSel = campaign.AdvanceNativeTwoColumnSelection(g.churchSel, listLen, -2)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
				g.churchSel = campaign.AdvanceNativeTwoColumnSelection(g.churchSel, listLen, 2)
			}
			g.churchRosterStart, _ = campaign.NativeTwoColumnWindow(
				listLen, g.churchSel, g.churchRosterStart,
			)
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				if !g.beginNativeChurchRosterClosing(g.returnToNativeChurchMenu) {
					g.returnToNativeChurchMenu()
				}
				return true
			}
			if enter && listLen > 0 && g.churchSel < listLen {
				id := g.churchIDs[g.churchSel]
				openStatus := func() {
					if !g.beginNativeChurchStatus(id) {
						g.msg = "角色缺少原版 status/command panel provenance"
						g.returnToNativeStatusRoster()
					}
				}
				if !g.beginNativeChurchRosterClosing(openStatus) {
					openStatus()
				}
			}
			return true
		}
		if g.churchMode == "status_view" || g.churchMode == "status_commands" {
			ack := enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
				inpututil.IsKeyJustPressed(ebiten.KeyBackspace)
			if !ack {
				return true
			}
			if g.churchMode == "status_view" && len(g.churchCommandPanel) != 0 {
				if !g.beginNativeChurchStatusCommandTransition() {
					g.closeNativeChurchStatus(g.churchStatusPanel)
				}
				return true
			}
			panel := g.churchStatusPanel
			if g.churchMode == "status_commands" {
				panel = g.churchCommandPanel
			}
			g.closeNativeChurchStatus(panel)
			return true
		}
		if g.churchMode == "transfer_full" {
			if enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				if !g.beginNativeChurchTransferFullClosing(g.returnToNativeTransferSource) {
					g.returnToNativeTransferSource()
				}
			}
			return true
		}
		if g.churchMode == "transfer_source" || g.churchMode == "transfer_item" || g.churchMode == "transfer_dest" {
			listLen := len(g.churchIDs)
			if g.churchMode == "transfer_item" {
				listLen = len(g.churchTransferItems)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				g.churchSel = campaign.AdvanceNativeTwoColumnSelection(g.churchSel, listLen, -1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				g.churchSel = campaign.AdvanceNativeTwoColumnSelection(g.churchSel, listLen, 1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
				g.churchSel = campaign.AdvanceNativeTwoColumnSelection(g.churchSel, listLen, -2)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
				g.churchSel = campaign.AdvanceNativeTwoColumnSelection(g.churchSel, listLen, 2)
			}
			if g.churchMode == "transfer_item" {
				g.churchItemStart, _ = campaign.NativeTwoColumnWindow(
					listLen, g.churchSel, g.churchItemStart,
				)
			} else {
				g.churchRosterStart, _ = campaign.NativeTwoColumnWindow(
					listLen, g.churchSel, g.churchRosterStart,
				)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				switch g.churchMode {
				case "transfer_source":
					if !g.beginNativeChurchRosterClosing(g.returnToNativeChurchMenu) {
						g.returnToNativeChurchMenu()
					}
					return true
				case "transfer_item":
					if !g.beginNativeChurchTransferItemClosing(g.returnToNativeTransferSource) {
						g.returnToNativeTransferSource()
					}
					return true
				case "transfer_dest":
					if !g.beginNativeChurchRosterClosing(g.returnToNativeTransferSource) {
						g.returnToNativeTransferSource()
					}
					return true
				}
			}
			if enter && listLen > 0 && g.churchSel < listLen {
				switch g.churchMode {
				case "transfer_source":
					sourceID := g.churchIDs[g.churchSel]
					items := g.churchTransferItemSlots(sourceID)
					if len(items) == 0 {
						g.msg = "沒東西了！"
						return true
					}
					openItems := func() {
						g.churchTransferSource = sourceID
						g.churchTransferItems = items
						g.churchMode = "transfer_item"
						g.churchSel = 0
						g.churchItemStart = 0
						g.beginNativeChurchTransferItemOpening()
					}
					if !g.beginNativeChurchRosterClosing(openItems) {
						openItems()
					}
				case "transfer_item":
					itemSlot := g.churchTransferItems[g.churchSel]
					openDestinations := func() {
						g.churchTransferItem = itemSlot
						g.churchIDs = g.churchTransferDestinationIDs(g.churchTransferSource)
						g.churchMode = "transfer_dest"
						g.churchSel = 0
						g.churchRosterStart = 0
						g.nativeChurchTextIndex = 510
						g.beginNativeChurchRosterOpening()
					}
					if !g.beginNativeChurchTransferItemClosing(openDestinations) {
						openDestinations()
					}
				case "transfer_dest":
					destinationID := g.churchIDs[g.churchSel]
					apply := func() {
						source := g.partyRoster[g.churchTransferSource]
						itemID := source.Inventory[g.churchTransferItem]
						destination := g.partyRoster[destinationID]
						count, err := battle.NativeInventoryAvailableCount(destination.NativeInventoryFlags)
						if err != nil {
							g.msg = fmt.Sprintf("目的角色缺少原版 8-byte 物品欄旗標：%v", err)
							g.returnToNativeTransferSource()
							return
						}
						if count == 8 {
							g.churchTransferDest = destinationID
							g.churchMode = "transfer_full"
							if !g.beginNativeChurchTransferFullOpening() {
								g.msg = "無法還原原版物品欄已滿提示"
								g.returnToNativeTransferSource()
							}
							return
						}
						if destinationID == g.churchTransferSource {
							if err := battle.TransferNativeInventoryItem(
								&source, g.churchTransferItem, &source,
							); err != nil {
								g.msg = err.Error()
							} else {
								campaign.RecomputeEquipment(
									&source, g.shopItemStats,
								)
								g.partyRoster[g.churchTransferSource] = source
								g.msg = fmt.Sprintf(
									"物品 %02Xh 已轉移", itemID,
								)
							}
						} else if err := battle.TransferNativeInventoryItem(&source, g.churchTransferItem, &destination); err != nil {
							g.msg = err.Error()
						} else {
							campaign.RecomputeEquipment(
								&source, g.shopItemStats,
							)
							g.partyRoster[g.churchTransferSource] = source
							g.partyRoster[destinationID] = destination
							g.msg = fmt.Sprintf("物品 %02Xh 已轉移", itemID)
						}
						g.returnToNativeTransferSource()
					}
					if !g.beginNativeChurchRosterClosing(apply) {
						apply()
					}
				}
			}
			return true
		}
		if g.churchMode == "revive_confirm" {
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				if !g.beginNativeChurchReviveConfirmationClosing(g.returnToNativeReviveList) {
					g.returnToNativeReviveList()
				}
				return true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				g.churchSel = campaign.AdvanceNativeClassConfirmation(g.churchSel, -1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				g.churchSel = campaign.AdvanceNativeClassConfirmation(g.churchSel, 1)
			}
			if enter {
				if g.churchSel == 0 {
					apply := func() {
						if g.gold < g.churchReviveFee {
							g.churchMode = "revive_insufficient"
							return
						}
						g.reviveChurchUnit(g.churchReviveID)
						success := campaign.PlanNativeChurchReviveSuccess()
						g.playBGMCount(
							fmt.Sprintf("FDMUS_%03d", success.StartMusicTrack),
							success.MusicLoopCount,
						)
						after := func() {
							g.playBGMCount(
								fmt.Sprintf("FDMUS_%03d", success.ReturnMusicTrack),
								success.MusicLoopCount,
							)
							g.returnToNativeReviveList()
						}
						if !g.beginNativeChurchReviveSuccess(after) {
							g.playBGMCount(
								fmt.Sprintf("FDMUS_%03d", success.ReturnMusicTrack),
								success.MusicLoopCount,
							)
							g.returnToNativeReviveList()
						}
					}
					if !g.beginNativeChurchReviveChoiceClosing(apply) {
						apply()
					}
				} else if !g.beginNativeChurchReviveConfirmationClosing(g.returnToNativeReviveList) {
					g.returnToNativeReviveList()
				}
			}
			return true
		}
		if g.churchMode == "revive_empty" || g.churchMode == "revive_insufficient" {
			if enter || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				after := g.returnToNativeReviveList
				if g.churchMode == "revive_empty" {
					after = g.returnToNativeChurchMenu
				}
				if !g.beginNativeChurchReviveMessageClosing(after) {
					after()
				}
			}
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if g.churchMode == "class_confirm" {
				if !g.beginNativeClassConfirmationClosing(g.returnToNativeClassList) {
					g.returnToNativeClassList()
				}
				return true
			}
			if g.churchMode == "class" {
				if !g.beginNativeClassListClosing(g.returnToNativeChurchMenu) {
					g.returnToNativeChurchMenu()
				}
				return true
			}
			if g.churchMode == "revive" {
				if !g.beginNativeChurchReviveListClosing(g.returnToNativeChurchMenu) {
					g.returnToNativeChurchMenu()
				}
				return true
			}
			g.returnToNativeChurchMenu()
			return true
		}
		if g.churchMode == "class_confirm" {
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
				g.churchSel = campaign.AdvanceNativeClassConfirmation(g.churchSel, -1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
				g.churchSel = campaign.AdvanceNativeClassConfirmation(g.churchSel, 1)
			}
			if enter {
				if g.churchSel == 0 {
					apply := func() {
						if g.applyChurchClassChange(0) {
							g.beginNativeClassListOpening()
							return
						}
						g.returnToNativeClassList()
					}
					if !g.beginNativeClassConfirmationClosing(apply) {
						apply()
					}
				} else {
					if !g.beginNativeClassConfirmationClosing(g.returnToNativeClassList) {
						g.returnToNativeClassList()
					}
				}
			}
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.churchSel > 0 {
			g.churchSel--
		}
		listLen := len(g.churchIDs)
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.churchSel+1 < listLen {
			g.churchSel++
		}
		if g.churchMode == "class" || g.churchMode == "revive" {
			g.churchVerticalStart, _ = campaign.NativeThreeRowWindow(
				listLen, g.churchSel, g.churchVerticalStart,
			)
		}
		if enter && len(g.churchIDs) > 0 {
			id := g.churchIDs[g.churchSel]
			u := g.partyRoster[id]
			if g.churchMode == "revive" {
				fee, ok := g.nativeReviveFeeForUnit(u)
				if !ok {
					g.msg = "缺少原版復活費率資料"
					return true
				}
				openConfirmation := func() {
					g.churchReviveID = id
					g.churchReviveFee = fee
					g.churchMode = "revive_confirm"
					g.churchSel = 0
					g.beginNativeChurchReviveConfirmationOpening()
				}
				if !g.beginNativeChurchReviveListClosing(openConfirmation) {
					openConfirmation()
				}
			} else {
				target, ok := campaign.NativeClassChangeTarget(&u, g.classChangeTable)
				if !ok {
					g.msg = "缺少原版轉職目標資料"
				} else {
					openConfirmation := func() {
						g.churchClassID = id
						g.churchBranches = []campaign.ClassChangeBranch{target}
						g.churchMode = "class_confirm"
						g.churchSel = 0
						g.beginNativeClassConfirmationOpening()
					}
					if !g.beginNativeClassListClosing(openConfirmation) {
						openConfirmation()
					}
				}
			}
		}
		return true
	case "shop":
		if g.handleNativeShopInput(enter) {
			return true
		}
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
			g.leaveShop()
		}
		return true
	case "hotel":
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.hotelSel > 0 {
			g.hotelSel--
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.hotelSel < 3 {
			g.hotelSel++
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.leaveHotel()
			return true
		}
		if enter {
			g.applyHotelServiceSelection(byte(g.hotelSel))
		}
		return true
	case "ending":
		return true
	case "battle":
		if g.result != "" && enter { // 勝敗後 Enter → 依結果轉場(敗北可走敗北路線)
			return g.confirmBattleResult()
		}
		return false // 戰鬥照常
	}
	return false
}

// confirmBattleResult consumes the already displayed battle result at the
// explicit Enter boundary.  Keeping this seam separate prevents tests and
// future input adapters from bypassing the same postbattle/campaign handoff
// used by the player-facing control path.
func (g *Game) confirmBattleResult() bool {
	if g == nil || g.camp == nil || g.result == "" {
		return false
	}
	outcome := g.result
	current := g.camp.Node()
	// 第30戰的可編輯 ending edge明確要求保存最後隊伍，供來源約束 E1 的角色
	// 終局回顧使用；這是 remake 資料邊界，不冒稱原版 FD2.SAV ABI。
	if outcome == "win" && current != nil && current.EndingPartySnapshotOnWin {
		synced, err := g.syncPartyFromBattleRecords()
		if err == nil && synced == 0 {
			err = fmt.Errorf("完成的戰場沒有任何持續隊伍身分符合")
		}
		if err != nil {
			g.loadErr = "終局隊伍同步失敗: " + err.Error()
			g.msg = "最終戰結果尚未保存，流程已停止"
			return false
		}
	}
	// An empty next id is a valid terminal/game-over result; consume the same
	// Enter boundary and let enterNode observe the ended campaign.
	g.camp.Advance(outcome)
	// The result belongs to the battle node only.  Postbattle/cutscene and town
	// nodes must not inherit a stale win/lose overlay while they persist the
	// party and expose the next hub.
	g.result = ""
	g.enterNode()
	return true
}

// applyChurchClassChange is the runtime seam for the native class-change
// mutation. 0x31793 preselects exactly one target per candidate; the branch
// slice therefore contains one confirmation target, never a player menu.
// Candidate/target data stays editable and unknown growth rows fail
// closed before touching the persistent roster.
func (g *Game) applyChurchClassChange(branchIndex int) bool {
	if g.churchClassID < 0 || branchIndex < 0 || branchIndex >= len(g.churchBranches) {
		g.msg = "缺少有效轉職目標"
		return false
	}
	id := g.churchClassID
	u, ok := g.partyRoster[id]
	if !ok {
		g.msg = fmt.Sprintf("轉職角色不存在 id=%d", id)
		return false
	}
	branch := g.churchBranches[branchIndex]
	row, ok := g.classChangeGrowth[branch.Portrait]
	if !ok {
		g.msg = fmt.Sprintf("缺少轉職成長列 portrait=%02Xh", branch.Portrait)
		return false
	}
	if err := campaign.ApplyClassChange(&u, branch.Portrait, branch.ClassID, branch.MobilityIncrement, row, g.rng, branch.InventoryIndex); err != nil {
		g.msg = err.Error()
		return false
	}
	u.ClsName = campaign.ClassName(branch.ClassID)
	campaign.RecomputeAfterClassChange(&u, g.shopItemStats)
	g.partyRoster[id] = u
	g.msg = fmt.Sprintf("%s 已轉職為%s", u.Name, u.ClsName)
	g.churchMode, g.churchBranches, g.churchIDs = "class", nil, g.churchCandidates("class")
	g.churchSel = 0
	g.churchVerticalStart = 0
	return true
}

// leaveChurch is the editable campaign boundary for the church menu's Escape
// action. It never assumes a specific town or service return target.
func (g *Game) leaveChurch() {
	if g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != "church" {
		return
	}
	g.camp.Advance("")
	g.enterNode()
}

// reviveChurchUnit is the runtime seam for the native revive branch. Fee data
// and candidate filtering remain explicit; unknown class rows fail closed.
func (g *Game) reviveChurchUnit(id int) bool {
	u, ok := g.partyRoster[id]
	if !ok {
		g.msg = fmt.Sprintf("復活角色不存在 id=%d", id)
		return false
	}
	if !u.HasNativeRecordClass || int(u.NativeRecordClass) >= len(g.reviveFeeRates) {
		g.msg = fmt.Sprintf("復活費率缺少 raw class=%d", u.NativeRecordClass)
		return false
	}
	gold, cost, err := campaign.ReviveUnit(
		g.gold, &u, g.reviveFeeRates[int(u.NativeRecordClass)],
	)
	if err != nil {
		g.msg = fmt.Sprintf("復活費用 %d G：%v", cost, err)
		return false
	}
	g.gold, g.partyRoster[id] = gold, u
	g.msg = fmt.Sprintf("%s 已復活（-%d G）", u.Name, cost)
	g.churchIDs = g.churchCandidates("revive")
	if g.churchSel >= len(g.churchIDs) {
		g.churchSel = 0
	}
	return true
}

// leaveShop is the campaign boundary for the shop's Escape/leave action. It
// deliberately delegates the next node to editable campaign data instead of
// assuming every shop returns to the same town.
func (g *Game) leaveShop() {
	if g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != "shop" {
		return
	}
	returnSelection := g.nativeShopVariant
	g.nativeShopUIJob = nil
	g.nativeShopMode = ""
	g.nativeShopVariant = 0
	g.camp.Advance("")
	g.enterNode()
	if n := g.camp.Node(); n != nil && n.Type == "town" &&
		(returnSelection == 1 || returnSelection == 3 ||
			returnSelection == 5) {
		// 0x2e341 returns to the same town selector that dispatched the
		// weapon/item/secret branch. In particular, DOSBox E2 confirms
		// secret variant 5 restores the revealed hidden selection 5.
		g.campSel = returnSelection
	}
}

// applyHotelServiceSelection is deliberately raw: official 0x2fc85 proves
// selector/resource/callee order, but not the high-level service names. It
// records the route for an eventual indexed/UI consumer and never mutates
// party, gold, or campaign state on its own.
func (g *Game) applyHotelServiceSelection(selector byte) bool {
	route, ok := fdother.ResolveNativeHotelServiceRoute(selector)
	if !ok {
		g.msg = fmt.Sprintf("旅館 raw selector %d 無效", selector)
		return false
	}
	g.hotelRoute, g.hotelHasRoute = route, true
	if route.Secondary != 0 {
		g.msg = fmt.Sprintf("旅館 raw selector %d：%05X→%05X（待 UI callee）", selector, route.Primary, route.Secondary)
	} else {
		g.msg = fmt.Sprintf("旅館 raw selector %d：%05X（待 UI callee）", selector, route.Primary)
	}
	return true
}

// leaveHotel returns through the authored campaign edge. It never assumes a
// specific town or converts an unresolved raw hotel callee into gameplay.
func (g *Game) leaveHotel() {
	if g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != "hotel" {
		return
	}
	g.camp.Advance("")
	g.enterNode()
}

// stepCampaignMenu is the runtime seam for deterministic choice/town input
// traces. It owns only cursor state; the runner still owns editable option
// visibility and transition targets.
func (g *Game) stepCampaignMenu(event campaign.MenuEvent) (selected int, confirm bool) {
	if g.camp == nil {
		return 0, false
	}
	menu := campaign.MenuState{Selection: g.campSel, Count: len(g.camp.Visible())}
	selected, confirm = menu.Step(event)
	g.campSel = menu.Selection
	return selected, confirm
}

// ringInput 暫定四向命令選單 + 法術選單輸入。回傳 true = 已攔截。
// 方向配對已由 Docker Capstone 0x18d8c switch 釘死：↑0攻擊/←1法術/→2物品/↓3待機。
func (g *Game) ringInput() bool {
	enter := inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)
	esc := inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace)
	if g.nativeSystemEndTurnUI != nil && g.nativeClassUIJob != nil {
		return true
	}
	if g.nativeSystemEndTurnDelay > 0 {
		return true
	}
	if g.nativeSystemEndTurnConfirm {
		if g.nativeClassUIJob != nil {
			return true
		}
		if esc {
			g.cancelNativeSystemEndTurn()
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) && g.nativeSystemEndTurnUI != nil {
			g.nativeSystemEndTurnUI.choice = 0
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) && g.nativeSystemEndTurnUI != nil {
			g.nativeSystemEndTurnUI.choice = 1
		}
		if enter {
			if g.nativeSystemEndTurnUI != nil && g.nativeSystemEndTurnUI.choice == 1 {
				g.cancelNativeSystemEndTurn()
			} else {
				g.confirmNativeSystemEndTurn()
			}
		}
		return true
	}
	if g.nativeSystemCursorOverlay {
		if !g.ring {
			g.nativeSystemCursorOverlay = false
			return false
		}
		if g.actionOverlayBlocksInput() {
			return true
		}
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
		if esc {
			g.beginActionOverlayClose(func() {
				g.nativeSystemCursorOverlay = false
				g.msg = ""
			})
			return true
		}
		if enter {
			if g.ringSel == 3 {
				if !g.beginNativeSystemEndTurn() {
					g.msg = "原版 END 確認資產不完整，未結束回合"
				}
				return true
			}
			// 只有 Down→END 已有同一存檔、同一輸入的未修改原版 E2；
			// 其餘三格不由圖示外觀猜測動作 owner。
			g.msg = "原版續戰指令的此動作擁有者尚未驗證"
		}
		return true
	}
	if g.itemOpen {
		if g.sel == nil {
			g.itemOpen = false
			g.clearNativeItemPanel()
			return false
		}
		if g.stepNativeItemPanelAnimation() {
			return true
		}
		if esc {
			g.beginNativeItemPanelClose()
			return true
		}
		if g.nativeItemPanel != nil {
			rawSlots := nativeItemRawSlots(g.sel)
			if len(rawSlots) == 0 {
				g.itemOpen = false
				g.beginActionOverlayOpen(g.ringSel)
				g.clearNativeItemPanel()
				return true
			}
			key := 0
			switch {
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
				key = 72
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
				key = 80
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft):
				key = 75
			case inpututil.IsKeyJustPressed(ebiten.KeyArrowRight):
				key = 77
			}
			if key != 0 {
				selected, _, err := battle.AdvanceNativeItemSelector(g.itemSel, len(rawSlots), key, true, 1)
				if err == nil && selected != g.itemSel {
					g.itemSel = selected
					g.refreshNativeItemPanel(g.sel)
				}
			}
			if enter {
				rawSlot := rawSlots[g.itemSel]
				occupied, itemID := g.nativeItemMenuSlot(g.sel, rawSlot)
				if !occupied {
					g.msg = "空物品欄"
					return true
				}
				row := itemID * battle.NativeItemEffectRowSize
				if row < 0 || row+battle.NativeItemEffectRowSize > len(g.nativeItemEffectRows) {
					g.msg = "物品資料不完整"
					return true
				}
				_, result, err := battle.AdvanceNativeItemSelector(
					g.itemSel, len(rawSlots), 28, true,
					g.nativeItemEffectRows[row+0x0d],
				)
				if err != nil || result != battle.NativeItemSelectorConfirm {
					g.msg = "此物品不能在戰場使用"
					return true
				}
				if applied, applyErr := g.applyNativeImmediateItem(rawSlot, itemID); applyErr != nil {
					g.msg = fmt.Sprintf("物品 %02Xh：%v", itemID, applyErr)
					return true
				} else if applied {
					g.msg = fmt.Sprintf("物品 %02Xh：原始效果完成", itemID)
					return true
				}
				if targeting, targetErr := g.beginNativeTargetItem(rawSlot, itemID); targetErr != nil {
					g.msg = fmt.Sprintf("物品 %02Xh：%v", itemID, targetErr)
					return true
				} else if targeting {
					g.msg = fmt.Sprintf("物品 %02Xh：選擇目標", itemID)
					return true
				}
				g.msg = fmt.Sprintf("物品 %02Xh：使用效果尚未驗證", itemID)
			}
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.itemSel > 0 {
			g.itemSel--
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.itemSel < 7 {
			g.itemSel++
		}
		if enter {
			occupied, itemID := g.nativeItemMenuSlot(g.sel, g.itemSel)
			if !occupied {
				g.msg = "空物品欄"
				return true
			}
			// 0x1bbdc case 0 delegates to 0x20c6f.  Its effect/target table
			// is not yet closed, so selection is visible but never mutates state.
			g.msg = fmt.Sprintf("物品 %02Xh：使用效果尚未驗證", itemID)
			return true
		}
		return true
	}
	if g.nativeCommandOpen {
		if g.sel == nil {
			g.nativeCommandOpen = false
			return false
		}
		ids := g.sel.NativeCommandIDs()
		if esc {
			g.nativeCommandOpen = false
			g.beginActionOverlayOpen(g.ringSel)
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.nativeCommandSel = battle.NativeCommandGridMove(g.nativeCommandSel, len(ids), 0)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.nativeCommandSel = battle.NativeCommandGridMove(g.nativeCommandSel, len(ids), 1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			g.nativeCommandSel = battle.NativeCommandGridMove(g.nativeCommandSel, len(ids), 2)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			g.nativeCommandSel = battle.NativeCommandGridMove(g.nativeCommandSel, len(ids), 3)
		}
		if enter && g.nativeCommandSel >= 0 && g.nativeCommandSel < len(ids) {
			id := ids[g.nativeCommandSel]
			if g.nativeCommandTargetSupported(id) && g.st != nil && len(g.st.NativeCommandBook) == 36 {
				record := g.st.NativeCommandBook[id]
				if id == 0 && len(g.st.NativeCommandResistances) == 0 {
					g.msg = "原始指令 0：抗性資料未驗證"
					return true
				}
				if g.sel.MP < record.MPCost {
					g.msg = "MP 不足!"
					return true
				}
				if _, err := g.nativeCommandTargetUnitsFor(id); err == nil {
					if err := g.materializeNativeCommandTargetField(record); err != nil {
						g.msg = fmt.Sprintf("原始指令 %d：target grid 無法 materialize", id)
						return true
					}
					g.nativeCommandOpen, g.nativeCommand0Targeting = false, true
					g.nativeCommandTargetID = id
					g.msg = fmt.Sprintf("原始指令 %d：選擇目標", id)
					return true
				}
			}
			// Do not translate a raw command ID into legacy CastArea.  Native
			// 0x1cff0 is two-stage: record+3 picks a cursor candidate, then
			// record+4 builds the final effect list from the confirmed cursor.
			// Matching bytes in spells.json only prove the table identity for
			// IDs 0..35, not legacy target/effect equivalence.
			g.nativeCommandOpen = false
			g.beginActionOverlayOpen(g.ringSel)
			g.msg = fmt.Sprintf("原始指令 %d：目標／效果尚未驗證", id)
		}
		return true
	}
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
			g.spellOpen = false
			g.beginActionOverlayOpen(g.ringSel)
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
	if g.actionOverlayBlocksInput() {
		return true
	}
	// 環導航(doc13 [0x3C57]:↑0攻擊/←1法術/→2物品/↓3待機)
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
	if esc { // ESC = 取消(doc13):先播放0x176b4四幀，再退回移動前位置
		g.beginActionOverlayClose(func() {
			g.msg = ""
			if g.sel.X == g.selOrigX && g.sel.Y == g.selOrigY {
				g.sel, g.reach, g.moved = nil, nil, false
			} else {
				g.sel.SetMapPlacement(g.selOrigX, g.selOrigY, g.sel.Dir)
				g.moved = false
				g.reach = g.st.Reachable(g.sel)
				g.curX, g.curY = g.sel.X, g.sel.Y
			}
		})
		return true
	}
	if enter {
		// 0x177fc accepts a direction only when its corresponding 0x18d8c
		// availability word is zero.  Rendering a disabled FDOTHER cell without
		// enforcing the same input gate would let the remake execute an action
		// the native chooser rejects.
		if !nativeActionSelectable(g.actionOverlayAvailability(), g.ringSel) {
			g.msg = "此指令目前不可用"
			return true
		}
		switch g.ringSel {
		case 0: // 攻擊 → 關環,進選目標(游標移到攻擊範圍內的敵人;範圍依武器射程,doc32)
			g.beginActionOverlayClose(func() {
				g.msg = "攻擊:選擇目標"
			})
		case 1: // 法術(原版 0x1cff0；有法術者才可用)
			if ids := g.sel.NativeCommandIDs(); len(ids) > 0 && len(g.nativeCommandLabels) > 0 && len(g.nativeUIPalette) >= 0xce {
				// Native 0x18d8c disables its command action when raw unit+0x27
				// is nonzero.  NativeTransient[5] preserves exactly that byte;
				// legacy Sealed is a separate normalized compatibility status.
				if nativeCommandActionBlocked(g.sel) {
					g.msg = "原始指令目前不可用"
					return true
				}
				g.beginActionOverlayClose(func() {
					g.nativeCommandOpen, g.nativeCommandSel = true, 0
				})
				return true
			}
			if len(g.sel.Spells) > 0 {
				if g.sel.Sealed {
					g.msg = "被封咒,無法施法!"
				} else {
					g.beginActionOverlayClose(func() {
						g.spellOpen, g.spellSel = true, 0
					})
				}
			} else {
				g.msg = "沒有可用法術"
			}
		case 2: // 物品(原版 0x1bbdc；完整 item action 仍 fail-closed)
			g.beginActionOverlayClose(func() {
				g.itemOpen, g.itemSel = true, 0
				g.itemAnimStep, g.itemClosing = 0, false
				g.prepareNativeItemPanel(g.sel)
				g.msg = "物品：選擇欄位"
			})
		case 3: // 休息回復／格子互動(原版 0x13fd4→0x190ac)
			g.beginActionOverlayClose(g.finishSelectedWait)
		}
	}
	return true
}

// nativeCommandTargetSupported is deliberately a small whitelist.  These
// IDs have both the recovered two-stage target contract and a state-only
// executor; other command labels remain visible but fail closed at confirm.
func (g *Game) nativeCommandTargetSupported(id int) bool {
	switch id {
	case 0, 13, 14, 15, 16, 20, 21, 22, 24, 25, 26, 27, 28, 29, 31:
		return true
	default:
		return false
	}
}

// nativeCommandTargetUnits is the shared target-candidate projection for the
// command cursor and its renderer.  The selected raw command ID, rather than
// command 0 or a normalized spell, supplies the record+3 selection fields.
// Missing book/flags/actor state remains an error so UI cannot highlight or
// confirm a target from an unrelated record.
func (g *Game) nativeCommandTargetUnits() ([]*battle.Unit, error) {
	if g == nil {
		return nil, fmt.Errorf("native target command context unavailable")
	}
	return g.nativeCommandTargetUnitsFor(g.nativeCommandTargetID)
}

func (g *Game) nativeCommandTargetUnitsFor(id int) ([]*battle.Unit, error) {
	if g == nil || g.st == nil || g.sel == nil || !g.nativeCommandTargetSupported(id) {
		return nil, fmt.Errorf("native target command context unavailable")
	}
	if len(g.st.NativeCommandBook) != 36 {
		return nil, fmt.Errorf("native command book incomplete")
	}
	record := g.st.NativeCommandBook[id]
	flags, err := g.st.NativeCommandBaseFlags()
	if err != nil {
		return nil, err
	}
	return battle.NativeCommandTargets(
		g.st.W, g.st.H,
		battle.Cell{X: g.sel.X, Y: g.sel.Y},
		record.SelectionMode, record.TargetCode,
		flags, g.st.Units,
	)
}

func (g *Game) materializeNativeCommandTargetField(record battle.NativeCommandRecord) error {
	if g == nil || g.st == nil || g.sel == nil {
		return fmt.Errorf("native command target field context unavailable")
	}
	flags, err := g.st.NativeCommandBaseFlags()
	if err != nil {
		return err
	}
	fieldBytes, err := battle.NativeCommandTargetFieldBytes(
		g.st.W, g.st.H,
		battle.Cell{X: g.sel.X, Y: g.sel.Y},
		record.SelectionMode, 0, flags,
	)
	if err != nil || len(g.st.NativeTileBlitModes) != len(fieldBytes) {
		return fmt.Errorf("native command target field incomplete")
	}
	if record.EffectMode < 0 || record.EffectMode > 0xff ||
		!g.st.MaterializeNativeMapRangeMode(
			battle.NativeMapOverlaySelectorFromRecordByte(byte(record.EffectMode)),
		) {
		return fmt.Errorf("native command overlay selector invalid")
	}
	copy(g.st.NativeTileBlitModes, fieldBytes)
	return nil
}

func (g *Game) resetNativeTargetField() bool {
	if g == nil || g.st == nil || g.st.W <= 0 || g.st.H <= 0 ||
		len(g.st.NativeTileBlitModes) != g.st.W*g.st.H {
		return false
	}
	// 0x4dbfc runs immediately after 0x115b6 returns.
	for i := range g.st.NativeTileBlitModes {
		g.st.NativeTileBlitModes[i] = 0xff
	}
	return true
}

func (g *Game) cancelNativeItemTargetModal() bool {
	if g == nil || g.st == nil || (!g.nativeItemTargeting && !g.nativeItemRelocating) {
		return false
	}
	g.resetNativeTargetField()
	g.st.MaterializeNativeMapRangeMode(1)
	g.nativeItemTargeting = false
	g.nativeItemRelocating = false
	g.nativeMovementCostRows = nil
	g.itemOpen = true
	// The native item panel remains the caller-owned parent while either
	// selector is active.  Redraw the retained panel state; re-preparing it
	// would clear the already loaded raw effect table before a second confirm.
	g.refreshNativeItemPanel(g.sel)
	return true
}

// finishSelectedWait 對應原版行動選單第四項「下／休息」。未移動時走
// 0x13fd4：current HP != max HP 且 raw +0x25/+0x26 都為零才增加
// floor(max HP/5)，不保證最少回復 1。其後 0x190ac 檢查當格寶物；
// 不是踩上格子立即開箱，也不是道具指令。
func (g *Game) finishSelectedWait() {
	u := g.sel
	if u == nil {
		return
	}
	g.ring = false
	if u.X == g.selOrigX && u.Y == g.selOrigY &&
		u.HP != u.MaxHP &&
		u.NativeTransient[3] == 0 &&
		u.NativeTransient[4] == 0 {
		heal := u.MaxHP / 5
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
		} else if (before.Kind == "item" || before.Kind == "event") &&
			len(u.Inventory) >= 8 {
			g.msg = "物品欄已滿，寶物仍留在原處"
		}
	}
	g.finishSuccessfulUnitAction(u, func() {
		g.sel, g.reach, g.moved = nil, nil, false
	})
}

// finishSuccessfulUnitAction 對應 0x18890 外層在 action handler 成功返回後
// 才呼叫 selector1 的共同提交點。呼叫者必須先完成實際 mutation；取消、目標
// 不合法及 executor 錯誤不得抵達此處。after 保留各動作自己的介面清理。
func (g *Game) finishSuccessfulUnitAction(actor *battle.Unit, after func()) {
	if actor == nil {
		return
	}
	finish := func() {
		actor.Acted = true
		if after != nil {
			after()
		}
	}
	if g.beginNativeFieldEvent61(actor, finish) {
		return
	}
	if g.beginNativeFieldEvent75(actor, finish) {
		return
	}
	finish()
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
	tick int           // 原版 unit+4:每格 1..6，第7 tick提交目的格
	then func()        // 走完回呼(nil=玩家預設:開指令環)
}

func (g *Game) stepBattleWalk() {
	w := g.walk
	if w == nil || g.m == nil {
		return
	}
	finish := func(pose int) {
		last := w.path[len(w.path)-1]
		w.u.SetMapPlacement(last.X, last.Y, pose)
		g.walk = nil
		if w.then != nil {
			w.then()
		} else {
			g.moved = true
			g.beginActionOverlayOpen(1)
		}
	}
	if len(w.path) < 2 || w.seg >= len(w.path)-1 {
		pose := w.u.Dir
		if pose < 0 || pose > 3 {
			pose = 0
		}
		finish(pose)
		return
	}
	a, b := w.path[w.seg], w.path[w.seg+1]
	pose := dirToward(a.X, a.Y, b.X, b.Y)
	w.tick++
	if w.tick < 7 {
		w.u.X, w.u.Y = a.X, a.Y
		w.u.SetMapPose(pose)
		w.u.OffX = float64(b.X-a.X) * float64(g.m.TileW) * float64(w.tick) / 7
		w.u.OffY = float64(b.Y-a.Y) * float64(g.m.TileH) * float64(w.tick) / 7
		w.u.SetNativeMapGridMotion(pose, w.tick)
		return
	}
	if !w.u.FinishNativeMapGridStep(pose, b.X, b.Y) {
		w.u.SetMapPlacement(b.X, b.Y, pose)
	}
	// 原版 0x13488 只有 path byte 1 進 0x1300D；該函式在第七拍
	// 提交 x-1 後，才以新座標呼叫 0x13A44(..., selector0)。
	// 其餘方向及整條路徑完成都不得泛化成 selector0。
	if b.X == a.X-1 && b.Y == a.Y {
		if eventID, ok := battle.NativeFieldEventIDAt(g.st, b.X, b.Y, 0); ok && eventID == 62 {
			if _, err := battle.ApplyNativeFieldTurnActivationEvent(g.st, b.X, b.Y, 0); err != nil {
				g.loadErr = "battle field event62: " + err.Error()
				g.walk = nil
				return
			}
		} else {
			battle.ApplyNativeFieldModeEvent(g.st, w.u, b.X, b.Y, 0)
		}
	}
	w.seg++
	w.tick = 0
	if w.seg >= len(w.path)-1 {
		finish(pose)
	}
}

// SFX 事件 index（doc36 第 9 輪對照）：index 0=游標移動已確認（5 處方向鍵
// 分支證據）；0xc=「已選定」旗標伴隨音（疑確認，handle B 疊播）。戰鬥命中音
// 屬另一獨立池（[0x5411f] 動態子容器，尚未導出）；目前 sfx_03 只是重製端
// E1 近似音，不是已證實的原版 owner 或音訊 parity，待戰鬥池證據閉合後替換。
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
	// Keep the existing PNG map playable, but expose native resources only as
	// an all-or-nothing bundle. The indexed presentation bridge consumes this
	// field later; no partial resource is allowed to affect gameplay.
	g.nativeMapAssets = nil
	g.nativeMapDAC = nil
	g.nativePaletteRamp = nil
	g.nativePalettePulse = nil
	g.nativeCh28PostPresent = nil
	if native, nativeErr := loadNativeMapAssets(dir); nativeErr == nil && nativeMapAssetsAvailable(native) {
		g.nativeMapAssets = native
		g.nativeMapDAC = append(g.nativeMapDAC[:0], native.PaletteDAC...)
	}
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
// 幀長由 FIGANI descriptor +6 與明確的 fpt 顯示倍率決定；位置/走位由幀內嵌 (dx,dy)
// 資料驅動(doc06)。缺少一一配對的延遲表時返回 nil，呼叫端必須明確記錄呈現缺口，
// 不得宣稱畫面已播放。
func (g *Game) newAtkAnim(atkGroup, defGroup int, atkName, defName string,
	atkHP, atkMax, atkLV, atkMP, defLV, defMP, defHP0, defHP1, defMax, terrain int, atkOwn bool) *atkAnim {
	fpt := battleFPT()
	af := figaniIndex(atkGroup) + 1
	frames := g.figani[af]
	delays, ok := g.figaniDelays[af]
	if len(frames) == 0 || !ok || len(delays) != len(frames) {
		// Never invent a 15-frame animation or silently fall back to a fixed
		// delay: an unpaired PNG/export schedule is not an original attack.
		return nil
	}
	df := figaniIndex(defGroup)
	defFrames := g.figani[df]
	defDelays, defOK := g.figaniDelays[df]
	if len(defFrames) == 0 || !defOK || len(defDelays) != len(defFrames) {
		// The defender idle figure is a native presentation input too; do not
		// fall back to a guessed fixed-period breathing loop.
		return nil
	}
	timeline, err := figani.NewDisplayScheduler(delays, fpt)
	if err != nil {
		return nil
	}
	bodyTicks := timeline.BodyTicks()
	total := bodyTicks + 4*fpt // 尾段停格 4 幀時間
	return &atkAnim{atkFig: af, defFig: figaniIndex(defGroup), atkName: atkName, defName: defName,
		atkHP: atkHP, atkMax: atkMax, atkLV: atkLV, atkMP: atkMP, defLV: defLV, defMP: defMP,
		defHP0: defHP0, defHP1: defHP1, defMax: defMax, timer: total, total: total,
		fpt: fpt, terrain: terrain, atkOwn: atkOwn, figaniTimeline: timeline,
		bodyTicks: bodyTicks}
}

func (g *Game) finishAttackPresentation() {
	if g.atk == nil {
		return
	}
	after := g.atk.after
	g.atk = nil
	if after != nil {
		after()
	}
}

// figaniIndex maps the battle FIGANI visual selector to its resource trio.
// Native ABI keeps this separate from the map FDICON selector: 0x127e0 uses
// unit+2 ×12 for map sprites, while 0x287b5..0x2884c uses unit+7 ×3 for
// FIGANI. Current exported records only pass the shared visual id where that
// equality is evidenced; this helper must not be read as an ABI field alias.
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

// loadFIGANIDelays loads descriptor +6 values exported from the fixed
// player-provided FIGANI.DAT. It is deliberately separate from the PNG
// presentation assets: a missing, malformed, or mismatched schedule must not
// be replaced with a guessed frame count or delay.
func loadFIGANIDelays() (map[int][]int, error) {
	raw, err := os.ReadFile(assetPath("assets/figani/delays.json"))
	if err != nil {
		return nil, err
	}
	var encoded map[string][]int
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, err
	}
	out := make(map[int][]int, len(encoded))
	for key, delays := range encoded {
		id, err := strconv.Atoi(key)
		if err != nil || id < 0 {
			return nil, fmt.Errorf("invalid FIGANI delay resource %q", key)
		}
		if len(delays) == 0 {
			return nil, fmt.Errorf("FIGANI %d has no frame delays", id)
		}
		for frame, delay := range delays {
			if delay <= 0 {
				return nil, fmt.Errorf("FIGANI %d frame %d has invalid delay %d", id, frame, delay)
			}
		}
		out[id] = append([]int(nil), delays...)
	}
	if len(out) == 0 {
		return nil, errors.New("FIGANI delay table is empty")
	}
	return out, nil
}

// loadStoryScript 讀本機劇情文本檔(story-pipe 管線輸出:{"scenes":[{"label","lines":[{speaker,text}]}]})。
// scene 為空:舊行為,攤平全部 scenes 的 lines(整份劇本塞一個節點,ch02-33 尚未逐段接線時的 fallback)。
// scene 非空(doc46 §5.2):只取 label 對映的那一段——**每個 story 節點播一段,別把整份劇本攤平**,
// 才能讓場景/鏡頭/擺位跟著劇情分段切換,不會「一次播完才進戰場」。找不到該 label 時回 nil(呼叫端
// fallback 用節點內嵌 Lines,不會靜默播錯段)。檔案缺失(玩家未自備素材)同樣回 nil。
func loadStoryScript(path, scene string) []campaign.Line {
	return loadStoryScriptAt(path, scene, nil)
}

// defaultChapterStoryScript supplies the editable full-chapter transcript for
// generic story nodes whose campaign entry only carries a short fallback line.
// It is deliberately limited to the exact story_chNN key shape: handler
// cutscenes (story_chNN_pre/post or named scenes) must keep their explicit
// beats/bindings instead of silently replaying an entire chapter.
func defaultChapterStoryScript(nodeID string) string {
	if !strings.HasPrefix(nodeID, "story_ch") || len(nodeID) != len("story_ch00") {
		return ""
	}
	chapter, err := strconv.Atoi(nodeID[len("story_ch"):])
	if err != nil || chapter < 1 || chapter > 33 {
		return ""
	}
	return fmt.Sprintf("assets/story/ch%02d.json", chapter)
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

// resolvePhysicalAttack is the production bridge from a battle action path to
// the typed physical-settlement result.  The old UI path used
// State.Attack, which consumes a process-global RNG and discards Miss/Crit/
// experience metadata even though AttackWithRNG already exposes them.  A
// missing game RNG is an input failure: do not mutate the battle state through
// an implicit fallback source.
func (g *Game) resolvePhysicalAttack(actor, target *battle.Unit) (battle.AttackResult, error) {
	if g == nil || g.st == nil || actor == nil || target == nil {
		return battle.AttackResult{}, errors.New("physical attack context unavailable")
	}
	if g.rng == nil {
		return battle.AttackResult{}, errors.New("physical attack RNG unavailable")
	}
	return g.st.AttackWithRNG(actor, target, g.rng), nil
}

func (g *Game) resolvePlayerPhysicalAttack(actor, target *battle.Unit) (battle.AttackResult, error) {
	return g.resolvePhysicalAttack(actor, target)
}

func playerPhysicalAttackMessage(actor, target *battle.Unit, result battle.AttackResult) string {
	actorName, targetName := "攻方", "目標"
	if actor != nil {
		if actor.Name != "" {
			actorName = actor.Name
		} else if actor.ClsName != "" {
			actorName = actor.ClsName
		}
	}
	if target != nil {
		if target.Name != "" {
			targetName = target.Name
		} else if target.ClsName != "" {
			targetName = target.ClsName
		}
	}
	if result.Missed {
		return fmt.Sprintf("%s 攻擊 %s，未命中", actorName, targetName)
	}
	message := fmt.Sprintf("%s 攻擊 %s，造成 %d 傷害", actorName, targetName, result.Amount)
	if result.Crit {
		message += "，暴擊"
	}
	if result.ExpGained > 0 {
		message += fmt.Sprintf("，經驗 +%.0f", result.ExpGained)
	}
	return message
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
		g.consumeNativeContinueOpeningConfirm()
		if u == nil && g.nativeSystemOverlayReady() {
			// 0x117E7 在 0x12C0D 回傳 -1 時直接呼叫 0x16F55；這是共用
			// 玩家控制器，不是 chapter0 CONTINUE 特例。ActionOverlayOrigin
			// 以 visible map cursor 為基準，不借用選取單位座標。
			g.nativeSystemCursorOverlay = true
			// 0x16f55 將 action selector 初始化為 0；此值只保留輸入游標
			// 的原始順序，並不替四格指派動作語意。
			g.beginActionOverlayOpen(0)
			return
		}
		if u != nil && u.Camp == battle.Own && u.Paralyzed {
			g.msg = "麻痺中,無法行動!"
			return
		}
		if u != nil && u.Camp == battle.Own && !u.Acted {
			g.sel = u
			g.selOrigX, g.selOrigY = u.X, u.Y // 記移動前位置(ESC 取消退回,playfix #4)
			g.moved = false
			g.reach = g.st.Reachable(u)
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
			g.atk = g.newAtkAnim(g.sel.BattleFig, tgt.BattleFig, g.sel.Name, nm,
				g.sel.HP, g.sel.MaxHP, g.sel.Lv, g.sel.MP, tgt.Lv, tgt.MP,
				tgt.HP+first.Amount, tgt.HP, tgt.MaxHP, g.terrainAt(tgt.X, tgt.Y), true)
		}
		actor := g.sel
		actor.SetMapPose(dirToward(actor.X, actor.Y, g.curX, g.curY))
		g.castSp, g.sel, g.reach, g.moved = nil, nil, nil, false
		if g.atk != nil {
			g.atk.after = func() { g.finishSuccessfulUnitAction(actor, nil) }
		} else {
			g.finishSuccessfulUnitAction(actor, nil)
		}
		g.checkResult()
		return
	}
	if g.nativeItemTargeting {
		tgt := g.st.UnitAt(g.curX, g.curY)
		applied, err := g.applyNativeTargetItem(tgt)
		if err != nil {
			g.msg = fmt.Sprintf("物品 %02Xh：%v", g.nativeItemTargetID, err)
			return
		}
		if !applied {
			if g.nativeItemRelocating {
				g.msg = fmt.Sprintf("物品 %02Xh：選擇目的地", g.nativeItemTargetID)
			}
			return
		}
		g.msg = fmt.Sprintf("物品 %02Xh：原始回復效果完成", g.nativeItemTargetID)
		return
	}
	if g.nativeItemRelocating {
		applied, err := g.applyNativeRelocationDestination(g.curX, g.curY)
		if err != nil {
			g.msg = fmt.Sprintf("物品 %02Xh：%v", g.nativeItemTargetID, err)
			return
		}
		if applied {
			g.msg = fmt.Sprintf("物品 %02Xh：原始移位效果完成", g.nativeItemTargetID)
		}
		return
	}
	if g.nativeCommand0Targeting {
		id := g.nativeCommandTargetID
		tgt := g.st.UnitAt(g.curX, g.curY)
		if len(g.st.NativeCommandBook) != 36 || id < 0 || id >= len(g.st.NativeCommandBook) ||
			len(g.st.NativeTileBlitModes) != g.st.W*g.st.H ||
			g.curX < 0 || g.curX >= g.st.W || g.curY < 0 || g.curY >= g.st.H {
			g.msg = fmt.Sprintf("原始指令 %d：cursor-confirm raw state 不完整", id)
			return
		}
		record := g.st.NativeCommandBook[id]
		allowed, gateErr := battle.NativeCursorConfirmationAllowed(
			battle.Cell{X: g.curX, Y: g.curY},
			g.st.NativeTileBlitModes[g.curY*g.st.W+g.curX],
			g.st.NativeMapRangeMode, record.TargetCode, g.st.Units,
		)
		if gateErr != nil || !allowed {
			g.msg = fmt.Sprintf("原始指令 %d：游標確認不合法", id)
			return
		}
		if id >= 13 && id <= 16 {
			actor := g.sel
			targets, err := g.st.NativeCommandHealTargets(actor, tgt, id)
			if err != nil {
				g.msg = fmt.Sprintf("原始指令 %d：請選擇有效目標 (%v)", id, err)
				return
			}
			err = g.startNativeCommandHealPresentation(id, targets, func() ([]battle.NativeCommandHealResult, error) {
				return g.st.ExecuteNativeCommandHeal(actor, tgt, id, g.rng)
			}, func(results []battle.NativeCommandHealResult) {
				total := 0
				for _, result := range results {
					total += result.Restore.Actual
				}
				actor.SetMapPose(dirToward(actor.X, actor.Y, g.curX, g.curY))
				g.msg = fmt.Sprintf("原始指令 %d：回復 %d", id, total)
				g.finishSuccessfulUnitAction(actor, func() {
					g.resetNativeTargetField()
					g.st.MaterializeNativeMapRangeMode(1)
					g.nativeCommand0Targeting, g.nativeCommandTargetID, g.sel, g.reach, g.moved = false, 0, nil, nil, false
				})
				g.checkResult()
			})
			if err != nil {
				g.msg = fmt.Sprintf("原始指令 %d：indexed 演出不可用 (%v)", id, err)
			}
			return
		}
		message := ""
		var err error
		var damageTargets []*battle.Unit
		switch {
		case id == 0:
			results, state, e := g.st.ExecuteBoundNativeCommand0(g.sel, tgt, g.nativeRNGState)
			err = e
			if e == nil {
				g.nativeRNGState = state
			}
			hit, total := 0, 0
			for _, result := range results {
				if result.Hit {
					hit++
					total += result.Damage
				}
				damageTargets = append(damageTargets, result.Target)
			}
			message = fmt.Sprintf("原始指令 0：命中 %d，傷害 %d", hit, total)
		case id == 20 || id == 21:
			results, e := g.st.ExecuteNativeCommandClearRestore(g.sel, tgt, id, g.rng)
			err = e
			message = fmt.Sprintf("原始指令 %d：完成 raw interval 處理 (%d targets)", id, len(results))
		case id == 22 || id == 26 || id == 27:
			results, e := g.st.ExecuteNativeCommandApplication(g.sel, tgt, id, g.rng)
			err = e
			for _, result := range results {
				if result.Damage > 0 {
					damageTargets = append(damageTargets, result.Target)
				}
			}
			message = fmt.Sprintf("原始指令 %d：完成 raw application (%d targets)", id, len(results))
		case id == 24 || id == 28 || id == 29 || id == 31:
			results, e := g.st.ExecuteNativeCommandDerivedStrike(g.sel, tgt, id, g.rng)
			err = e
			total := 0
			for _, result := range results {
				total += result.Damage
				damageTargets = append(damageTargets, result.Target)
			}
			message = fmt.Sprintf("原始指令 %d：傷害 %d", id, total)
		case id == 25:
			results, e := g.st.ExecuteNativeCommand25(g.sel, tgt)
			err = e
			message = fmt.Sprintf("原始指令 25：完成 raw clear (%d targets)", len(results))
		default:
			err = fmt.Errorf("native command target executor unavailable id=%d", id)
		}
		if err != nil {
			g.msg = fmt.Sprintf("原始指令 %d：請選擇有效目標 (%v)", id, err)
			return
		}
		for _, target := range damageTargets {
			g.awardDeathReward(target, g.sel)
		}
		actor := g.sel
		actor.SetMapPose(dirToward(actor.X, actor.Y, g.curX, g.curY))
		g.msg = message
		g.finishSuccessfulUnitAction(actor, func() {
			g.resetNativeTargetField()
			g.st.MaterializeNativeMapRangeMode(1)
			g.nativeCommand0Targeting, g.nativeCommandTargetID, g.sel, g.reach, g.moved = false, 0, nil, nil, false
		})
		g.checkResult()
		return
	}
	if !g.moved { // 移動階段
		switch {
		case g.curX == g.sel.X && g.curY == g.sel.Y: // 原地 → 不移動,開指令環
			g.moved = true
			g.reach = nil
			g.beginActionOverlayOpen(1)
		case g.reach[cur] && g.st.UnitAt(g.curX, g.curY) == nil: // 移動到可達空格:沿路徑逐格走
			if p := g.st.Path(g.sel, g.curX, g.curY); len(p) >= 2 {
				g.walk = &walkAnim{u: g.sel, path: p}
			} else { // 理論上不會(reach 內必可達),保底瞬移
				g.sel.SetMapPlacement(g.curX, g.curY, g.sel.Dir)
				g.moved = true
				g.beginActionOverlayOpen(1)
			}
			g.reach = nil
		}
		return
	}
	// 攻擊階段:游標在攻擊範圍內的敵 → 攻擊;在自己格 → 待命
	if tgt := g.st.UnitAt(g.curX, g.curY); tgt != nil && tgt != g.sel &&
		tgt.Camp != battle.Own && g.st.InAttackRange(g.sel, g.curX, g.curY) {
		// 攻擊者面向目標(FDICON 方向幀)
		g.sel.SetMapPose(dirToward(g.sel.X, g.sel.Y, g.curX, g.curY))
		nm := tgt.Name
		if nm == "" {
			nm = tgt.ClsName
		}
		anm := g.sel.Name
		if anm == "" {
			anm = g.sel.ClsName
		}
		defHP0 := tgt.HP
		attackResult, err := g.resolvePhysicalAttack(g.sel, tgt)
		if err != nil {
			g.msg = "攻擊：" + err.Error()
			return
		}
		g.awardDeathReward(tgt, g.sel)
		g.msg = playerPhysicalAttackMessage(g.sel, tgt, attackResult)
		actor := g.sel
		g.atk = g.newAtkAnim(actor.BattleFig, tgt.BattleFig, anm, nm,
			actor.HP, actor.MaxHP, actor.Lv, actor.MP, tgt.Lv, tgt.MP,
			defHP0, tgt.HP, tgt.MaxHP, g.terrainAt(g.curX, g.curY), true) // 戰鬥背景 = 守方格地形
		if g.atk != nil {
			g.atk.after = func() {
				g.finishSuccessfulUnitAction(actor, nil)
			}
		} else {
			// Damage was already resolved by the typed battle rule; record the
			// missing presentation without fabricating a frame sequence.
			g.loadErr = fmt.Sprintf("FIGANI attack presentation unavailable: %d -> %d", actor.BattleFig, tgt.BattleFig)
			g.finishSuccessfulUnitAction(actor, nil)
		}
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
	protect := "索爾"
	if g.camp != nil {
		if n := g.camp.Node(); n != nil && n.Protect != "" {
			protect = n.Protect
		}
	}
	if r := g.st.Result(protect); r != "" {
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
	g.stepActionOverlayLifecycle()
	g.stepNativeSystemEndTurn()
	g.stepNativeClassUILifecycle(time.Now())
	g.stepNativeChurchUILifecycle(time.Now())
	g.stepNativeShopUILifecycle(time.Now())
	g.stepNativeTownUILifecycle(g.nativeTownUIClock.Sample(time.Now()))
	g.stepNativePreparationUILifecycle(time.Now())
	if g.nativeEnding == nil && !nativeModifierHeld() && inpututil.IsKeyJustPressed(ebiten.KeyF2) { // 全域:切換音源(MT-32 / Sound Blaster)
		g.cycleBGMSource()
	}
	if g.nativeEnding == nil && !nativeModifierHeld() && inpututil.IsKeyJustPressed(ebiten.KeyF3) { // 全域:開發除錯 HUD 開關
		g.debug = !g.debug
	}
	if g.bannerT > 0 {
		g.bannerT--
	}
	if g.titlePhase != "" {
		if g.titleUpdate() {
			if g.shotPath != "" && g.shotTaken { // 截圖模式在 title 也要能退出
				return ebiten.Termination
			}
			return nil
		}
	}
	if g.nativeEnding != nil {
		endingConfirm := inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)
		reviewExit := inpututil.IsKeyJustPressed(ebiten.KeyEscape)
		if g.nativeEnding.reviewingCampaignPartyOutcomes() && (endingConfirm || reviewExit) {
			if err := g.returnCampaignTerminalFromReview(); err != nil {
				g.loadErr = "native ending review: " + err.Error()
				return err
			}
		} else if g.nativeEnding.presentingCampaignTerminal() && endingConfirm {
			if err := g.startCampaignPartyOutcomeReview(); err != nil {
				g.loadErr = "native ending review: " + err.Error()
				return err
			}
		}
		if !g.nativeEnding.reviewingCampaignPartyOutcomes() && g.nativeEnding.montage != nil && !g.nativeEnding.montage.Ready() && len(inpututil.AppendJustPressedKeys(nil)) != 0 {
			// 0x2c950 does not decode Enter/Space.  Preserve a raw changed-input
			// condition until the recovered portrait loop polls it.
			g.nativeEnding.montageInputPending = true
		}
		if err := g.nativeEnding.advance(time.Now(), &g.nativeRNGState); err != nil {
			g.loadErr = "native ending: " + err.Error()
			return err
		}
		if g.nativeEnding.atNativeMontageGate() && g.nativeEnding.montage == nil && !g.nativeEnding.montageStartAttempted {
			// Failure here keeps the explicitly approximate fallback available;
			// it never substitutes a guessed native renderer in faithful mode.
			_ = g.startCampaignNativeMontage()
		}
		if g.nativeEnding.atNativeMontageGate() && g.nativeEnding.montage != nil &&
			g.nativeEnding.montage.Ready() && !g.nativeEnding.tailStartAttempted {
			// Explicit approximate mode consumes the provenance-checked 20-entry
			// source schedule, then holds the final native image. Faithful mode
			// never reaches this adapter.
			_ = g.startCampaignNativeTail()
		}
		if err := g.queueNativeEndingDialogue(); err != nil {
			g.loadErr = "native ending dialogue: " + err.Error()
			return err
		}
		g.consumeNativeEndingAudioAtGate()
		g.stepDlgAnim()
		if g.dlgScrollT > 0 {
			g.dlgScrollT--
		}
		if len(g.dialog) > 0 && endingConfirm {
			if g.dlgAdvance() && len(g.dialog) == 0 {
				g.resumeNativeEndingDialogue()
			}
		}
		if endingConfirm && g.finishCampaignNativeEndingFallback() {
			return nil
		}
		if g.shotPath != "" && g.shotTaken {
			return ebiten.Termination
		}
		return nil
	}
	if g.nativeFieldEvent61 != nil {
		g.stepNativeFieldEvent61Tick(
			g.nativeFieldEvent61.clock.Sample(time.Now()),
		)
		if g.shotPath != "" && g.shotTaken {
			return ebiten.Termination
		}
		return nil
	}
	if g.nativeAIIdleRecovery != nil {
		g.stepNativeAIIdleRecovery()
		if g.shotPath != "" && g.shotTaken {
			return ebiten.Termination
		}
		return nil
	}
	if g.nativeCh20SkyKey != nil {
		g.stepNativeCh20SkyKey()
		if g.shotPath != "" && g.shotTaken {
			return ebiten.Termination
		}
		return nil
	}
	if g.nativeCh23Loop != nil {
		g.stepNativeCh23Loop()
		if g.shotPath != "" && g.shotTaken {
			return ebiten.Termination
		}
		return nil
	}
	if g.native2189A != nil {
		g.stepNative2189A()
		if g.shotPath != "" && g.shotTaken {
			return ebiten.Termination
		}
		return nil
	}
	if g.nativeUnitPresent != nil {
		g.stepNativeUnitPresent()
		if g.shotPath != "" && g.shotTaken {
			return ebiten.Termination
		}
		return nil
	}
	if g.nativeCh28PostPresent != nil {
		g.stepNativeCh28PostPresent()
		if g.shotPath != "" && g.shotTaken {
			return ebiten.Termination
		}
		return nil
	}
	// 攻擊演出推進(FIGANI 全身分鏡;演出期間鎖玩家輸入)
	if g.atk != nil {
		g.atk.timer--
		a := g.atk
		prog := a.total - a.timer
		if a.figaniTimeline != nil && prog <= a.bodyTicks {
			frame, presented, _, err := a.figaniTimeline.Step()
			if err != nil {
				// The timeline was validated at load time; a runtime failure must
				// not run the action continuation with an unpresented frame.
				a.after = nil
				g.atk = nil
				g.loadErr = "FIGANI attack timeline: " + err.Error()
				return nil
			}
			if presented {
				a.frameIndex = frame
			}
		}
		if a.fpt > 0 { // 三段音效(== 比對每 tick 遞增的 prog,各觸發一次)
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
			g.finishAttackPresentation()
			if g.nativeFieldEvent61 != nil || g.battleEvent != nil {
				return nil
			}
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
					u.SetMapPose(0) // 到位面向鏡頭待機
				}
			}
		}
	}
	// 移動動畫:原版每格 unit+4=1..6，第7 tick提交目的格。
	g.stepBattleWalk()
	g.aiStep() // AI 回合驅動(aiBusy 時逐單位行走→攻擊演出)
	// 嘴型動畫(忠實原版 0x16d00,doc14):每 2 frame 一 tick;閉嘴隨機 2-31 tick、開嘴一瞬
	if len(g.dialog) > 0 && g.frame%2 == 0 {
		randomMod30 := 0
		if g.mouthState.Open {
			randomMod30 = rand.Intn(30)
		}
		if next, err := g.mouthState.Tick(randomMod30); err == nil {
			g.mouthState = next
			g.mouthOpen = next.FrameIndex() == 3
			g.mouthTimer = next.Countdown
		}
	}
	// 截圖模式:到指定幀後自動退出(畫面已於 Draw 存檔)。逐幀攻擊序列
	// 只有 `FD2_SHOT_SERIES` 時也必須進入同一個 setup，否則攻擊演出永遠
	// 不會建立，逐幀工具只能得到空目錄而無法驗證 GUI 時序。
	if g.shotPath != "" || g.shotSeries != "" {
		// Apply screenshot-only setup immediately before capture, after scenario
		// setup has had time to spawn its party. Frame 1 is too early for
		// FD2_SHOT_RING on battle-start event scenarios.
		if !g.shotSetup && g.frame >= g.shotFrame-1 {
			g.shotSetup = true
			if spec := os.Getenv("FD2_SHOT_TOWN_STATE"); spec != "" {
				selection, pulse, ok := parseNativeTownShotState(spec)
				if !ok || !g.setNativeTownShotState(selection, pulse) {
					return fmt.Errorf(
						"FD2_SHOT_TOWN_STATE expects selection 0..5,pulse 0..3 on a native town node: %q",
						spec,
					)
				}
			}
			if spec := os.Getenv("FD2_SHOT_SHOP_STATE"); spec != "" {
				service, pulse, gold, ok := parseNativeShopShotState(spec)
				if !ok || !g.setNativeShopShotState(service, pulse, gold) {
					return fmt.Errorf(
						"FD2_SHOT_SHOP_STATE expects service 0..3,pulse 0..3,gold 0..99999999 on a stable native shop menu: %q",
						spec,
					)
				}
			}
			if spec := os.Getenv("FD2_SHOT_SHOP_PURCHASE_STATE"); spec != "" {
				selection, start, gold, ok :=
					parseNativeShopPurchaseShotState(spec)
				if !ok ||
					!g.setNativeShopPurchaseShotState(selection, start, gold) {
					return fmt.Errorf(
						"FD2_SHOT_SHOP_PURCHASE_STATE expects selection,start-even,gold on a claimed native shop with a stable valid item window: %q",
						spec,
					)
				}
			}
			if spec := os.Getenv("FD2_SHOT_SHOP_CONFIRM_STATE"); spec != "" {
				good, choice, pulse, gold, ok :=
					parseNativeShopConfirmShotState(spec)
				if !ok ||
					!g.setNativeShopConfirmShotState(
						good, choice, pulse, gold,
					) {
					return fmt.Errorf(
						"FD2_SHOT_SHOP_CONFIRM_STATE expects good,choice,pulse,gold on a claimed native shop with a valid editable good: %q",
						spec,
					)
				}
			}
			if spec := os.Getenv("FD2_SHOT_SHOP_INSUFFICIENT_STATE"); spec != "" {
				good, gold, ok :=
					parseNativeShopInsufficientShotState(spec)
				if !ok ||
					!g.setNativeShopInsufficientShotState(good, gold) {
					return fmt.Errorf(
						"FD2_SHOT_SHOP_INSUFFICIENT_STATE expects good,gold with gold below the editable price on a claimed native shop: %q",
						spec,
					)
				}
			}
			if spec := os.Getenv("FD2_SHOT_SHOP_EQUIPMENT_RECIPIENT_STATE"); spec != "" {
				good, selection, start, cycle, gold, ok :=
					parseNativeShopEquipmentRecipientShotState(spec)
				if !ok ||
					!g.setNativeShopEquipmentRecipientShotState(
						good, selection, start, cycle, gold,
					) {
					return fmt.Errorf(
						"FD2_SHOT_SHOP_EQUIPMENT_RECIPIENT_STATE expects good,selection,start,cycle,gold on an admitted native equipment recipient party: %q",
						spec,
					)
				}
			}
			if spec := os.Getenv("FD2_SHOT_SHOP_SELL_STATE"); spec != "" {
				mode, unit, selection, start, cycle, gold, ok :=
					parseNativeShopSellShotState(spec)
				if !ok ||
					!g.setNativeShopSellShotState(
						mode, unit, selection, start, cycle, gold,
					) {
					return fmt.Errorf(
						"FD2_SHOT_SHOP_SELL_STATE expects roster|items,unit,selection,start,cycle,gold on an admitted native sell party: %q",
						spec,
					)
				}
			}
			if spec := os.Getenv("FD2_SHOT_SHOP_SELL_CONFIRM_STATE"); spec != "" {
				unit, item, choice, pulse, gold, ok :=
					parseNativeShopSellConfirmShotState(spec)
				if !ok ||
					!g.setNativeShopSellConfirmShotState(
						unit, item, choice, pulse, gold,
					) {
					return fmt.Errorf(
						"FD2_SHOT_SHOP_SELL_CONFIRM_STATE expects unit,item,choice,pulse,gold on an admitted native sell party: %q",
						spec,
					)
				}
			}
			if spec := os.Getenv("FD2_SHOT_SHOP_SELL_SUCCESS_STATE"); spec != "" {
				phase, unit, item, state, gold, ok :=
					parseNativeShopSellSuccessShotState(spec)
				if !ok ||
					!g.setNativeShopSellSuccessShotState(
						phase, unit, item, state, gold,
					) {
					return fmt.Errorf(
						"FD2_SHOT_SHOP_SELL_SUCCESS_STATE expects timeline|credit|return,unit,item,step|cycle,gold on an admitted native sell party: %q",
						spec,
					)
				}
			}
			if os.Getenv("FD2_SHOT_DISMISS_DIALOG") != "" {
				for len(g.dialog) > 0 {
					g.dialog = g.dialog[:len(g.dialog)-1]
				}
				g.dlgShown, g.dlgPhase, g.dlgPage, g.dlgScrollT = dlgNone, 0, 0, 0
			}
			for i := 0; i < g.shotTurn; i++ { // 推進 N 個回合(觸發增援事件),驗證進場
				g.endTurn()
			}
			if g.shotCurX != 0 || g.shotCurY != 0 {
				g.positionScreenshotCursor(g.shotCurX, g.shotCurY)
			}
			if g.shotSel {
				g.confirm()
			}
			if os.Getenv("FD2_SHOT_RING") != "" { // 截圖驗證:建立可重現的戰場 action-overlay state
				g.dialog = nil // 清開場對白(避免蓋住環)
				for _, unit := range g.st.Units {
					// The shot harness proves renderer state, not player eligibility;
					// event-driven scenarios can still have all party records hidden
					// at this exact frame. Pick the first materialized unit deterministically.
					if unit != nil {
						g.sel, g.curX, g.curY = unit, unit.X, unit.Y
						break
					}
				}
				if g.sel != nil {
					g.moved, g.reach = true, nil
					g.beginActionOverlayOpen(1)
					// Default capture is the settled oracle. A documented
					// open:N/close:N override exposes each recovered present
					// without introducing synthetic input timing.
					g.actionOverlayPhase = actionOverlayOpen
					g.actionOverlayFrame = 3
					if spec := os.Getenv("FD2_SHOT_RING_FRAME"); spec != "" {
						phase, rawFrame, found := strings.Cut(spec, ":")
						frame, err := strconv.Atoi(rawFrame)
						if found && err == nil && frame >= 0 && frame < 4 {
							switch phase {
							case "open":
								g.actionOverlayPhase, g.actionOverlayFrame = actionOverlayOpening, frame
								g.actionOverlayShotHold = true
							case "close":
								g.actionOverlayPhase, g.actionOverlayFrame = actionOverlayClosing, frame
								g.actionOverlayShotHold = true
							}
						}
					}
				}
			}
			if os.Getenv("FD2_SHOT_COMMAND") != "" && g.st != nil {
				// The raw command-grid oracle must choose a unit that actually owns
				// a materialized native command mask. The generic ring fixture may
				// have selected an enemy or a story-only record first.
				g.dialog = nil
				for _, unit := range g.st.Units {
					if unit != nil && len(unit.NativeCommandIDs()) > 0 {
						g.sel, g.curX, g.curY = unit, unit.X, unit.Y
						break
					}
				}
				if g.sel != nil {
					g.resetActionOverlayLifecycle()
					g.nativeCommandOpen, g.nativeCommandSel = true, 0
				}
			}
			if os.Getenv("FD2_SHOT_SPELL") != "" { // 截圖驗證:開法術選單
				g.dialog = nil
				g.confirm()
				g.confirm()
				if g.sel != nil && len(g.sel.Spells) > 0 {
					g.resetActionOverlayLifecycle()
					g.spellOpen, g.spellSel = true, 0
				}
			}
			if v := os.Getenv("FD2_SHOT_ATTACK"); v != "" { // 全螢幕戰鬥演出(驗證用):亞雷斯打盜賊
				g.dialog = nil // 清開場對白(避免蓋住演出)
				fig, _ := strconv.Atoi(v)
				g.atk = g.newAtkAnim(fig, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 2, 0, 28, 8, 28, 0, true)
				if g.atk == nil {
					g.loadErr = fmt.Sprintf("FD2_SHOT_ATTACK FIGANI presentation unavailable: %d", fig)
				}
			}
			if os.Getenv("FD2_SHOT_ATKSEL") != "" { // 截圖驗證:選單位→原地開環→模擬環選「攻擊」(ringSel==1)
				// 關環,進攻擊目標選擇階段(驗證武器攻擊距離高亮,doc32;搭配 FD2_SHOT_CUR 指定選哪個單位)。
				// 環的 case1 本身由 ringInput() 真實按鍵觸發(inpututil 偵測,截圖模式無法送假按鍵),
				// 這裡直接複製 case1 的狀態轉移(g.ring=false),不能改呼叫 g.confirm() 三次
				// (ring 開啟時 confirm() 不會攔下,會落到「在自己格→待命」把 g.sel 清掉)。
				g.dialog = nil
				g.confirm() // 選取游標上的單位
				g.confirm() // 原地(游標未動)→ 開指令環(moved=true, ring=true, ringSel=1)
				g.resetActionOverlayLifecycle()
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
		} else if g.shotTaken {
			return ebiten.Termination
		}
	}
	if g.m == nil {
		return nil
	}
	g.stepStoryWalks() // 場景走位動畫(doc46 §5.3);storyWalks 為空時內部直接返回
	g.stepActJob()     // beat「act」姿態循環(doc50);actJob 為空時內部直接返回
	g.stepFocusUnit()  // beat「focus_unit」依原版安全帶逐格移動游標／鏡頭
	g.stepDlgAnim()    // 對話框換人縮/展動畫(使用者回饋 #3)
	if g.dlgScrollT > 0 {
		g.dlgScrollT--
	}
	g.stepFade()                                 // 場景淡出/淡入轉場(doc46 §5.2;beat「fade」兩個方向都靠 then 接回下一拍)
	g.stepTransitionReveal()                     // native 0x24b4d alternating present loop
	g.stepNativeIndexedTransition()              // native 0x24618 indexed map/palette transition
	g.stepNativeCommandHealPresentation()        // native 0x21EB1 command 13..16 indexed presentation
	g.stepNativePaletteRamp()                    // native 0x1f882/0x1f525 whole-DAC ramps
	g.stepNativePalettePulse()                   // native 0x35E5A whole-DAC pulse
	g.stepNativeSpawnIntro()                     // native 0x32999 twelve-pass indexed spawn transition
	g.stepNativeTurnStaging()                    // typed raw-camp0 pre-AI staging helper
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
	} else if g.battleEvent == nil && g.nativeTurnStaging == nil {
		if !g.syncNativeMapView() {
			g.camX = float64(g.curX*g.m.TileW - logicalW/2 + g.m.TileW/2)
			g.camY = float64(g.curY*g.m.TileH - logicalH/2 + g.m.TileH/2)
			clamp(&g.camX, 0, float64(g.m.W*g.m.TileW-logicalW))
			clamp(&g.camY, 0, float64(g.m.H*g.m.TileH-logicalH))
		}
	}
	if g.nativeHealPresentation != nil {
		return nil
	}
	if !nativeModifierHeld() && inpututil.IsKeyJustPressed(ebiten.KeyF5) { // 快速存檔(節點邊界語意:存 campaign 進度)
		g.saveGame()
	}
	if !nativeModifierHeld() && inpututil.IsKeyJustPressed(ebiten.KeyF9) { // 快速讀檔
		g.loadGame()
	}
	if g.battleEvent != nil || g.nativeTurnStaging != nil {
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
	if g.nativeItemTargeting { // 原始物品第一階段 target selector：ESC 回物品列
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			g.cancelNativeItemTargetModal()
			return nil
		}
	}
	if g.nativeItemRelocating {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			g.cancelNativeItemTargetModal()
			return nil
		}
	}
	if g.castSp != nil || g.nativeCommand0Targeting { // native target selection: ESC 回 command grid
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if g.nativeCommand0Targeting && g.st != nil {
				g.resetNativeTargetField()
				g.st.MaterializeNativeMapRangeMode(1)
			}
			g.castSp, g.nativeCommand0Targeting, g.nativeCommandOpen = nil, false, true
			return nil
		}
	}
	// 游標移動:方向鍵 / WASD(按住持續移動,keyRepeat)/ 觸控
	if keyRepeat(ebiten.KeyArrowLeft) || keyRepeat(ebiten.KeyA) {
		g.moveMapCursor(-1, 0)
	}
	if keyRepeat(ebiten.KeyArrowRight) || keyRepeat(ebiten.KeyD) {
		g.moveMapCursor(1, 0)
	}
	if keyRepeat(ebiten.KeyArrowUp) || keyRepeat(ebiten.KeyW) {
		g.moveMapCursor(0, -1)
	}
	if keyRepeat(ebiten.KeyArrowDown) || keyRepeat(ebiten.KeyS) {
		g.moveMapCursor(0, 1)
	}
	// 觸控:點哪格移到哪格
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		tx, ty := ebiten.TouchPosition(id)
		targetX := (int(g.camX) + tx) / g.m.TileW
		targetY := (int(g.camY) + ty) / g.m.TileH
		if targetX < 0 {
			targetX = 0
		} else if targetX >= g.m.W {
			targetX = g.m.W - 1
		}
		if targetY < 0 {
			targetY = 0
		} else if targetY >= g.m.H {
			targetY = g.m.H - 1
		}
		for g.curX < targetX {
			g.moveMapCursor(1, 0)
		}
		for g.curX > targetX {
			g.moveMapCursor(-1, 0)
		}
		for g.curY < targetY {
			g.moveMapCursor(0, 1)
		}
		for g.curY > targetY {
			g.moveMapCursor(0, -1)
		}
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
			g.beginActionOverlayOpen(g.ringSel)
			g.msg = ""
		} else {
			g.sel, g.reach = nil, nil
			g.msg = ""
		}
	}
	// Tab 是重製端快速鍵；原版式正常入口是空游標四格系統面板的 Down→END。
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.endTurn()
	}
	// 相機跟隨游標(置中,夾在地圖內)
	if !g.syncNativeMapView() {
		g.camX = float64(g.curX*g.m.TileW - logicalW/2 + g.m.TileW/2)
		g.camY = float64(g.curY*g.m.TileH - logicalH/2 + g.m.TileH/2)
		clamp(&g.camX, 0, float64(g.m.W*g.m.TileW-logicalW))
		clamp(&g.camY, 0, float64(g.m.H*g.m.TileH-logicalH))
	}
	return nil
}

func (g *Game) syncNativeMapView() bool {
	if g == nil || g.m == nil || g.st == nil || !g.st.HasNativeMapViewState {
		return false
	}
	view := g.st.NativeMapViewState
	g.curX, g.curY = view.CursorX, view.CursorY
	g.camX = float64(view.CameraX * g.m.TileW)
	g.camY = float64(view.CameraY * g.m.TileH)
	return true
}

func (g *Game) moveMapCursor(dx, dy int) {
	if g.st != nil && g.st.HasNativeMapViewState {
		if _, ok := g.st.MoveNativeMapCursor(dx, dy); ok {
			view := g.st.NativeMapViewState
			if g.st.HasNativeMapHUDState {
				g.st.AdvanceNativeMapHUDAnchor(view.VisibleCursorX, view.VisibleCursorY)
			}
			g.syncNativeMapView()
		}
		return
	}
	g.curX += dx
	g.curY += dy
}

// positionScreenshotCursor drives the same recovered cursor/camera state
// machine as interactive input. Directly assigning curX/curY leaves the
// production indexed frame on stale NativeMapViewState and produces a false
// visual oracle.
func (g *Game) positionScreenshotCursor(x, y int) bool {
	if g == nil || g.st == nil || x < 0 || y < 0 || x >= g.st.W || y >= g.st.H {
		return false
	}
	if !g.st.HasNativeMapViewState {
		g.curX, g.curY = x, y
		return true
	}
	for g.st.NativeMapViewState.CursorX != x {
		dx := 1
		if g.st.NativeMapViewState.CursorX > x {
			dx = -1
		}
		g.moveMapCursor(dx, 0)
	}
	for g.st.NativeMapViewState.CursorY != y {
		dy := 1
		if g.st.NativeMapViewState.CursorY > y {
			dy = -1
		}
		g.moveMapCursor(0, dy)
	}
	return g.curX == x && g.curY == y
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

func (g *Game) nativeMapFrameAdmission(legacyViewport, campaignBattleView bool) bool {
	if g == nil || legacyViewport || !campaignBattleView {
		return false
	}
	if g.spellOpen || g.itemOpen || g.castSp != nil {
		return false
	}
	return g.sel == nil || g.nativeCommand0Targeting ||
		g.nativeItemTargeting || g.nativeItemRelocating ||
		g.ring || g.nativeCommandOpen
}

func (g *Game) Draw(screen *ebiten.Image) {
	// The exact full-DAC saturation covers the complete mode-13h surface,
	// including HUD and dialogue. Defer preserves that hardware ordering across
	// every early-return presentation branch.
	defer func() {
		if g.nativeFullDACWhite {
			screen.Fill(color.White)
		}
		if g.nativeFullDACBlack {
			screen.Fill(color.Black)
		}
		if g.nativePaletteRamp == nil && nativeDACIsBlack(g.nativeMapDAC) {
			screen.Fill(color.Black)
		}
		if g.nativeTurnStaging != nil && !g.nativeTurnStaging.indexed &&
			g.nativeTurnStaging.phase == nativeTurnStagingFlash {
			g.nativeTurnStaging.drawn = true
		}
	}()
	if g.titlePhase != "" {
		g.drawTitle(screen)
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.nativeEnding != nil {
		g.drawNativeEndingPreview(screen)
		return
	}
	if g.nativeCh20SkyKey != nil {
		screen.Fill(color.Black)
		if !g.drawNativeCh20SkyKey(screen) {
			g.failNativeCh20SkyKey(errors.New("presentation unavailable"))
			ebitenutil.DebugPrint(screen, "native 0x24336 presentation unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.nativeCh23Loop != nil {
		screen.Fill(color.Black)
		if !g.drawNativeCh23Loop(screen) {
			g.failNativeCh23Loop(errors.New("presentation unavailable"))
			ebitenutil.DebugPrint(screen, "native ch23 presentation unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.native2189A != nil {
		screen.Fill(color.Black)
		if !g.drawNative2189A(screen) {
			g.failNative2189A(errors.New("presentation unavailable"))
			ebitenutil.DebugPrint(screen, "native 0x2189A presentation unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.nativeUnitPresent != nil {
		screen.Fill(color.Black)
		if !g.drawNativeUnitPresent(screen) {
			g.failNativeUnitPresent(errors.New("presentation unavailable"))
			ebitenutil.DebugPrint(screen, "native 0x22253 presentation unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.nativeCh28PostPresent != nil {
		screen.Fill(color.Black)
		if !g.drawNativeCh28PostPresent(screen) {
			g.failNativeCh28PostPresent(errors.New("presentation unavailable"))
			ebitenutil.DebugPrint(screen, "native 0x1DB65 presentation unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.transitionReveal != nil {
		screen.Fill(color.Black)
		if !g.drawNative24B4D(screen) {
			g.failNative24B4D(errors.New("presentation unavailable"))
			ebitenutil.DebugPrint(screen, "native 0x24B4D presentation unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.m == nil {
		ebitenutil.DebugPrint(screen, "FD2 重製 MVP\n缺 assets/(tileset.png + map.json)\n用 tools/export_engine_assets.py 產生\n"+g.loadErr)
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.indexedTransition != nil {
		screen.Fill(color.Black)
		if !g.drawNativeIndexedTransition(screen) {
			ebitenutil.DebugPrint(screen, "native 0x24618 transition unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.nativeHealPresentation != nil {
		screen.Fill(color.Black)
		if !g.drawNativeCommandHealPresentation(screen) {
			ebitenutil.DebugPrint(screen, "native 0x21EB1 presentation unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.nativePaletteRamp != nil {
		screen.Fill(color.Black)
		if !g.drawNativePaletteRamp(screen) {
			ebitenutil.DebugPrint(screen, "native palette ramp unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.nativePalettePulse != nil {
		screen.Fill(color.Black)
		if !g.drawNativePalettePulse(screen) {
			ebitenutil.DebugPrint(screen, "native 0x35E5A palette pulse unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.spawnIntroTransition != nil {
		screen.Fill(color.Black)
		if !g.drawNativeSpawnIntro(screen) {
			ebitenutil.DebugPrint(screen, "native 0x32999 spawn-intro unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.nativeTurnStaging != nil && g.nativeTurnStaging.indexed {
		screen.Fill(color.Black)
		if !g.drawNativeTurnStaging(screen) {
			ebitenutil.DebugPrint(screen, "native turn staging unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.nativeFieldEvent61 != nil {
		screen.Fill(color.Black)
		if !g.drawNativeFieldEvent61(screen) {
			ebitenutil.DebugPrint(screen, "native event61 presentation unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	if g.nativeAIIdleRecovery != nil {
		screen.Fill(color.Black)
		if !g.drawNativeAIIdleRecovery(screen) {
			job := g.nativeAIIdleRecovery
			g.restoreNativeAIIdleRecoveryRange(job)
			g.nativeAIIdleRecovery = nil
			g.loadErr = "native AI 0x13fd4 presentation unavailable"
			g.aiBusy = false
			ebitenutil.DebugPrint(screen, "native AI 0x13fd4 presentation unavailable")
		}
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	// 攻擊演出:切全螢幕戰鬥畫面(蓋地圖,對照原版 orig_05 全螢幕戰鬥)
	if g.atk != nil {
		g.drawBattleScene(screen)
		if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
			g.captureShot(screen)
		}
		return
	}
	// 清背景:地圖比畫面窄時右/上下留黑邊(非殘影黃白)。
	// RE 結論(0x11eee 地形迴圈,doc 見 knowledge-base/25):原版戰場視窗固定 13×8 格(312×192px)、
	// 逐格 blit 全無 memset/fillrect,且全 34 張地圖最小 18×20 格恆大於視窗——這個「地圖比視窗窄」情境
	// 原版從未觸發,無「原版清色」可對齊;黑色是 remake 自訂 FOV(640 寬、tile 維持原生 24px)才會露出的
	// 邊,選黑純為視覺乾淨、非還原原版行為。
	screen.Fill(color.RGBA{0, 0, 0, 0xff})
	nativeMapPresented := false
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
		if g.nativeItemTargeting {
			ih := ebiten.NewImage(tw, th)
			ih.Fill(color.RGBA{0x50, 0xd0, 0x80, 0x68})
			for _, unit := range g.nativeItemSelectionTargets() {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(unit.X*tw)-g.camX, float64(unit.Y*th)-g.camY)
				target.DrawImage(ih, op)
			}
		}
		if g.nativeItemRelocating {
			rh := ebiten.NewImage(tw, th)
			rh.Fill(color.RGBA{0x40, 0xd8, 0xd8, 0x68})
			for cell := range g.nativeRelocationDestinations() {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(cell.X*tw)-g.camX, float64(cell.Y*th)-g.camY)
				target.DrawImage(rh, op)
			}
		}
		if g.nativeCommand0Targeting {
			if targets, err := g.nativeCommandTargetUnits(); err == nil {
				ch := ebiten.NewImage(tw, th)
				ch.Fill(color.RGBA{0xff, 0x80, 0x20, 0x68})
				for _, unit := range targets {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(float64(unit.X*tw)-g.camX, float64(unit.Y*th)-g.camY)
					target.DrawImage(ch, op)
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
			g.drawUnitSprite(target, ux, uy, float64(tw), float64(th), u, g.mapSpriteGroup(u))
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
		g.drawUnitSprite(target, ux, uy, float64(tw), float64(th), u, u.Fig)
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
			if !g.drawNativeMapHUD(screen) {
				g.drawUnitHUD(screen, u)
			}
		}
		if g.debug { // F3:詳細除錯(回合/戰況/座標)
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("T%d own%d ally%d enemy%d cur(%d,%d)",
				g.st.Turn, g.st.AliveCount(battle.Own), g.st.AliveCount(battle.Ally), g.st.AliveCount(battle.Enemy), g.curX, g.curY), 6, 4)
		}
	}
	if !legacyViewport && campaignBattleView {
		g.drawPhaseBanner(screen) // 回合橫幅(PLAYER/ENEMY PHASE,transient)
	}
	// A complete original indexed frame supersedes the normalized map/unit/HUD
	// layers for the verified drawable selectors 1..5. Target selection keeps
	// g.sel as its actor, so explicitly admit those modal states. The recovered
	// action overlay and raw command grid are composited after this frame; if
	// their parent remains on the normalized map, short maps leave an unproven
	// black band below the 320x200 native viewport. Record-byte+2 writers can
	// exceed six and target validation still consumes those values even when
	// 0x122dc draws no overlay; those states retain the playable renderer until
	// their complete presentation lifecycle is materialized.
	if g.nativeMapFrameAdmission(legacyViewport, campaignBattleView) {
		nativeMapPresented = g.drawNativeMapFrame(screen)
	}

	// 中文層(原版點陣字型,doc 08):選中單位名 + 對話框(DebugPrint 不支援中文)
	if g.font != nil {
		if g.st != nil && !legacyViewport && !nativeMapPresented { // 選中單位中文名(放游標格上方,避開頂部 DebugPrint)
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
				// 自動換行(dlgWrap 與 Enter 分頁共用)。Enter 翻頁時保留舊頁與新頁，
				// 以平滑往上捲動取代瞬間切換(原版文字捲動效果;速度為 remake 可編輯參數)。
				lines := dlgWrap(dl)
				drawPage := func(page int, offset float64) {
					start := page * 3
					for i := 0; i < 3 && start+i < len(lines); i++ {
						y := ty + float64(i)*38 + offset
						if y < by+8 || y > by+184 { // clip 於框內，避免捲出框外
							continue
						}
						g.font.Draw(screen, lines[start+i], tx, y, 1.7, color.RGBA{0xf0, 0xf4, 0xff, 0xff})
					}
				}
				if g.dlgScrollT > 0 {
					progress := 1 - float64(g.dlgScrollT)/float64(dlgScrollFrames)
					drawPage(g.dlgScrollFrom, -progress*114)
					drawPage(g.dlgPage, (1-progress)*114)
				} else {
					drawPage(g.dlgPage, 0)
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

	// Command UI is an indexed-sprite layer; it must not depend on an optional
	// Chinese font being available in the runtime environment.
	if !g.drawNativeSystemEndTurn(screen) {
		g.drawRing(screen)
		g.drawNativeCommandGrid(screen)
		g.drawSpellMenu(screen)
		g.drawItemMenu(screen)
	}

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
	if g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame {
		g.captureShot(screen)
	}
}

func (g *Game) captureShot(screen *ebiten.Image) {
	g.shotTaken = true
	if g.loadErr != "" {
		// 截圖是證據產物，不得把失敗即關閉（fail-closed）的錯誤狀態
		// 偽裝成可比較的執行期畫面。shotTaken 仍讓無人值守程序結束，
		// 呼叫端可用輸出檔不存在判定驗證失敗。
		log.Printf("shot rejected: %s", g.loadErr)
		return
	}
	if tracePath := os.Getenv("FD2_SHOT_STATE"); tracePath != "" {
		if err := g.writeShotStateTrace(tracePath); err != nil {
			g.loadErr = "截圖狀態追蹤：" + err.Error()
			log.Printf("shot trace rejected: %s", g.loadErr)
			return
		}
	}
	saveShot(screen, g.shotPath)
}

// drawRing shows the native FDOTHER #2 action overlay when the player has
// supplied FDOTHER.DAT. The historical PNG ring remains a fail-closed fallback.
func (g *Game) drawRing(screen *ebiten.Image) {
	if !g.ring || g.m == nil || (g.sel == nil && !g.nativeSystemCursorOverlay) {
		return
	}
	tw, th := g.m.TileW, g.m.TileH
	ux, uy, _ := g.actionOverlayAnchor(g.sel)
	if g.drawNativeActionOverlay(screen, ux, uy) {
		g.markActionOverlayDrawn()
		return
	}
	if g.sel == nil {
		// 游標命令格只能使用使用者提供的原始 indexed cell；缺資產時
		// 不以現代圖示或虛構選取單位取代。
		return
	}
	// 行動中單位標記 + 補畫其 sprite 在最上層:部署較密的隊形下,鄰格友軍的 sprite
	// 可能探進環的中央空隙,讓人誤以為環中央是別的角色(playfix #5)。用橘色底標記
	// 「這是誰在動」+ 把 g.sel 自己的 sprite 疊到最上層,消除歧義。
	mark := ebiten.NewImage(tw, th)
	mark.Fill(color.RGBA{0xff, 0xa8, 0x20, 0x50})
	mop := &ebiten.DrawImageOptions{}
	mop.GeoM.Translate(ux, uy)
	screen.DrawImage(mark, mop)
	g.drawUnitSprite(screen, ux, uy, float64(tw), float64(th), g.sel, g.mapSpriteGroup(g.sel))
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
	g.markActionOverlayDrawn()
}

// actionOverlayAnchor converts the native visible map cursor (or the selected
// unit in the legacy normalized renderer) into the screen coordinate system
// used by the overlay. The raw 0x1741c address contract, exported as
// fdother.ActionOverlayOrigin, is cursor-based; native frames must therefore
// not anchor the overlay to a unit that happens to be nearby.
func (g *Game) actionOverlayAnchor(u *battle.Unit) (x, y, scale float64) {
	if g == nil || g.m == nil {
		return 0, 0, 1
	}
	scale = 1
	if g.nativeMapAssets != nil && g.st != nil && g.st.HasNativeMapViewState &&
		g.nativeMapFrameAdmission(false, g.camp == nil || (g.camp.Node() != nil && g.camp.Node().Type == "battle")) {
		scale = 2
		cursor := g.st.NativeMapViewState
		return (float64(cursor.CursorX*g.m.TileW) - g.camX) * scale,
			(float64(cursor.CursorY*g.m.TileH) - g.camY) * scale, scale
	}
	if u == nil {
		return 0, 0, scale
	}
	return (float64(u.X*g.m.TileW) - g.camX) * scale,
		(float64(u.Y*g.m.TileH) - g.camY) * scale, scale
}

func (g *Game) actionOverlayAvailability() [4]int {
	availability := [4]int{}
	if g.sel == nil || g.st == nil {
		return [4]int{1, 1, 1, 1}
	}
	attack := false
	// 0x1b83d first requires an equipped inventory entry whose ID is < 0x80.
	// The following target geometry is still the current remake range model;
	// the original item-record +0xb/+0xc calculation is not yet an adapter.
	if hasNativeEquippedWeapon(g.sel) {
		for _, unit := range g.st.Units {
			if unit.OnField && unit.HP > 0 && unit.Camp != battle.Own && g.st.InAttackRange(g.sel, unit.X, unit.Y) {
				attack = true
				break
			}
		}
	}
	if !attack {
		availability[0] = 1
	}
	if hasNativeCommand(g.sel) {
		// A raw command inventory follows the original unit+0x27 gate.  The
		// normalized Spells list is retained only for old editable scenarios;
		// it has its own legacy Sealed gate and is not evidence about FD2 ABI.
		if nativeCommandActionBlocked(g.sel) {
			availability[1] = 1
		}
	} else if g.sel.Sealed || len(g.sel.Spells) == 0 {
		availability[1] = 1
	}
	if len(g.sel.NativeInventoryFlags) == 8 {
		if count, err := battle.NativeInventoryAvailableCount(g.sel.NativeInventoryFlags); err != nil || count == 0 {
			availability[2] = 1
		}
	} else if len(g.sel.Inventory) == 0 {
		// Legacy editable JSON has no raw eight-cell flags; this is explicitly
		// a compatibility approximation, not native item availability.
		availability[2] = 1
	}
	return availability
}

// nativeItemMenuSlot preserves the eight-cell selector boundary used by
// 0x1b932.  InventorySlots/NativeInventoryFlags are required for the raw
// layout; legacy units fall back to their compact editable inventory solely
// to keep the selector visible.  It never assigns an effect meaning to the
// selected row.
func (g *Game) nativeItemMenuSlot(u *battle.Unit, slot int) (occupied bool, itemID int) {
	if u == nil || slot < 0 || slot >= 8 {
		return false, 0
	}
	if len(u.InventorySlots) == 8 && len(u.NativeInventoryFlags) == 8 {
		if u.NativeInventoryFlags[slot]&0x80 != 0 {
			return false, u.InventorySlots[slot]
		}
		return true, u.InventorySlots[slot]
	}
	if slot >= len(u.Inventory) {
		return false, 0
	}
	return true, u.Inventory[slot]
}

// hasNativeEquippedWeapon is the recovered 0x1b83d inventory precondition.
// Inventory/Equipped are the remake's compact projection of the native eight
// slots; a missing equipped entry is deliberately not treated as a weapon.
func hasNativeEquippedWeapon(unit *battle.Unit) bool {
	if unit == nil {
		return false
	}
	if len(unit.NativeInventoryFlags) == 8 && len(unit.InventorySlots) == 8 {
		slot, err := battle.NativeEquippedInventorySlot(unit.NativeInventoryFlags, unit.InventorySlots, 0)
		return err == nil && slot >= 0
	}
	for i, itemID := range unit.Inventory {
		if itemID >= 0 && itemID < 0x80 && i < len(unit.Equipped) && unit.Equipped[i] {
			return true
		}
	}
	return false
}

// hasNativeCommand uses only the exact 0x1c269 bit inventory.  It must not
// infer native command availability from the normalized editable Spells list.
func hasNativeCommand(unit *battle.Unit) bool {
	if unit == nil {
		return false
	}
	for _, bits := range unit.NativeCommandMask {
		if bits != 0 {
			return true
		}
	}
	return false
}

// nativeCommandActionBlocked is the raw action-menu gate recovered from
// 0x18d8c: after 0x1c269 finds a command, unit+0x27 nonzero disables the
// command direction. Command 22 is a verified writer of this duration byte,
// but its gameplay name and exhaustive producer set remain intentionally
// unknown; NativeTransient[5] is the fail-closed storage of that byte.
func nativeCommandActionBlocked(unit *battle.Unit) bool {
	return unit == nil || unit.NativeTransient[5] != 0
}

// nativeActionSelectable mirrors the final 0x177fc availability check after
// a direction is chosen: only a zero disabled-word may dispatch an action.
func nativeActionSelectable(availability [4]int, direction int) bool {
	return direction >= 0 && direction < len(availability) && availability[direction] == 0
}

func nativeActionOffsetXY(offset int) (int, int) {
	const stride = 0x1c8
	y, x := offset/stride, offset%stride
	if x > stride/2 {
		x -= stride
		y++
	}
	return x, y
}

func (g *Game) drawNativeActionOverlay(screen *ebiten.Image, cursorX, cursorY float64) bool {
	state := g.nativeActionOverlayState()
	frame, closing := g.actionOverlayRenderState()
	offsets, err := fdother.ActionOverlayFrameOffsets(frame, closing)
	if err != nil {
		return false
	}
	for direction, offset := range offsets {
		index, err := state.CellIndex(direction)
		if err != nil || index >= len(g.nativeActionCells) || g.nativeActionCells[index] == nil {
			return false
		}
		dx, dy := nativeActionOffsetXY(offset)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(cursorX+float64(dx*2), cursorY+float64(dy*2))
		screen.DrawImage(g.nativeActionCells[index], op)
	}
	return true
}

// nativeActionOverlayState keeps each original 0x1741c caller's two raw
// tables separate. The shared empty-cursor system owner is not a battle-unit
// wrapper: it uses 0x16f55's [7,5,6,4]/[0,0,0,0] tables and therefore must not
// borrow the normalized selected-unit availability approximation.
func (g *Game) nativeActionOverlayState() fdother.ActionOverlayState {
	if g != nil && g.nativeSystemCursorOverlay {
		return fdother.NativeContinueActionOverlayState()
	}
	return fdother.BattleActionOverlayState(g.actionOverlayAvailability())
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

// drawItemMenu prefers the proven indexed native panel. The text shell below
// is only a legacy/missing-original-assets fallback; using an item remains
// blocked when its 0x20c6f effect/target transaction is unavailable.
func (g *Game) drawItemMenu(screen *ebiten.Image) {
	if !g.itemOpen || g.sel == nil || g.font == nil {
		return
	}
	if g.drawNativeItemPanel(screen) {
		return
	}
	box := ebiten.NewImage(250, 270)
	box.Fill(color.RGBA{0x10, 0x1c, 0x40, 0xee})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(18, 42)
	screen.DrawImage(box, op)
	g.font.Draw(screen, fmt.Sprintf("物品　%s", g.sel.Name), 32, 52, 1.0, color.RGBA{0xff, 0xe0, 0x90, 0xff})
	for slot := 0; slot < 8; slot++ {
		occupied, itemID := g.nativeItemMenuSlot(g.sel, slot)
		c := color.RGBA{0x80, 0x88, 0x98, 0xff}
		label := "空"
		if occupied {
			c = color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			label = fmt.Sprintf("%02Xh", itemID)
		}
		pre := "　"
		if slot == g.itemSel {
			pre = "▶"
			c = color.RGBA{0xff, 0xff, 0xff, 0xff}
		}
		g.font.Draw(screen, fmt.Sprintf("%s[%d] %s", pre, slot+1, label), 32, 82+float64(slot)*25, 1.0, c)
	}
}

// drawNativeCommandGrid renders the recovered 0x1ceed four-row layout only
// when its player-provided labels and VGA palette are both available.
func (g *Game) drawNativeCommandGrid(screen *ebiten.Image) {
	if !g.nativeCommandOpen || g.sel == nil || g.font == nil || len(g.nativeUIPalette) < 0xce {
		return
	}
	for _, cell := range battle.NativeCommandGrid(g.sel.NativeCommandIDs(), g.nativeCommandSel) {
		label := g.nativeCommandLabels[cell.CommandID]
		if label == "" {
			continue
		}
		index := 0xcd
		if cell.Selected {
			index = 0xc9
		}
		r, green, b, a := g.nativeUIPalette[index].RGBA()
		c := color.RGBA{R: uint8(r >> 8), G: uint8(green >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
		g.font.Draw(screen, label, float64(cell.X*2), float64(cell.Y*2), 1.0, c)
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
		if n.Type == "town" && g.drawNativeTown(screen) {
			return
		}
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
		if g.drawNativeShop(screen) {
			return
		}
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
	case n.Type == "hotel":
		fillBox(140, 105, 360, 210)
		title := n.Text
		if title == "" {
			title = "旅館／整備（raw route）"
		}
		g.font.Draw(screen, title, 166, 122, 1.1, color.RGBA{0xff, 0xe0, 0x90, 0xff})
		for i := 0; i < 4; i++ {
			pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			if i == g.hotelSel {
				pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			label := fmt.Sprintf("raw selector %d", i)
			if g.hotelHasRoute && int(g.hotelRoute.Selector) == i {
				label = fmt.Sprintf("raw selector %d → %05X", i, g.hotelRoute.Primary)
			}
			g.font.Draw(screen, pre+label, 176, 154+float64(i)*28, 1.0, c)
		}
		g.font.Draw(screen, "↑↓ 選擇／Enter 記錄 raw route／ESC 返回", 176, 286, 0.85, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
	case n.Type == "preparation":
		if g.drawNativePreparation(screen) {
			return
		}
		h := 118 + float64((len(g.prepIDs)+1)/2)*24
		if h < 170 {
			h = 170
		}
		fillBox(64, 42, 512, h)
		g.font.Draw(screen, "出戰整備", 84, 56, 1.2, color.RGBA{0xff, 0xe0, 0x90, 0xff})
		if !g.prepSelecting && !g.prepConfirm {
			prompt := n.Prompt
			if prompt == "" {
				prompt = "要記錄戰況嗎？"
			}
			g.font.Draw(screen, prompt, 184, 120, 1.0, color.RGBA{0xff, 0xe0, 0x90, 0xff})
			yesColor := color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			noColor := color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			if g.prepConfirmSel == 0 {
				yesColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
			} else {
				noColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			g.font.Draw(screen, "是", 240, 158, 0.95, yesColor)
			g.font.Draw(screen, "否", 370, 158, 0.95, noColor)
		} else {
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
		}
		if g.prepConfirm {
			fillBox(154, 174, 332, 92)
			g.font.Draw(screen, "確定要進入戰場嗎？", 184, 190, 1.0, color.RGBA{0xff, 0xe0, 0x90, 0xff})
			yesColor := color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			noColor := color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
			if g.prepConfirmSel == 0 {
				yesColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
			} else {
				noColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			g.font.Draw(screen, "是", 240, 222, 0.95, yesColor)
			g.font.Draw(screen, "否", 370, 222, 0.95, noColor)
		}
		g.font.Draw(screen, "F5 保存戰況", 84, 88+h-24, 0.9, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
	case n.Type == "church":
		if g.drawNativeChurchUIJob(screen) {
			return
		}
		if g.drawNativeClassUIJob(screen) {
			return
		}
		if g.drawNativeChurchStatus(screen) {
			return
		}
		if g.drawNativeChurchTransferItem(screen) {
			return
		}
		if g.drawNativeChurchTransferFull(screen) {
			return
		}
		if g.drawNativeChurchReviveConfirmation(screen) {
			return
		}
		if g.drawNativeChurchReviveMessage(screen) {
			return
		}
		if g.drawNativeChurchReviveList(screen) {
			return
		}
		if g.drawNativeChurchRoster(screen) {
			return
		}
		if g.drawNativeClassConfirmation(screen) {
			return
		}
		if g.drawNativeClassList(screen) {
			return
		}
		if g.drawNativeChurchMenu(screen) {
			return
		}
		if g.churchMode == "menu" {
			fillBox(150, 110, 340, 180)
			g.font.Draw(screen, n.Text, 182, 126, 1.2, color.RGBA{0xff, 0xe0, 0x90, 0xff})
			labels := []string{"角色資訊", "物品轉交", "復活", "轉職"}
			for i, label := range labels {
				pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
				if i == g.churchSel {
					pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
				}
				g.font.Draw(screen, pre+label, 188, 158+float64(i)*24, 1.0, c)
			}
			g.font.Draw(screen, "←/→ 切換／Enter 選擇／ESC 返回城鎮", 188, 266, 0.9, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
		} else if g.churchMode == "transfer_source" || g.churchMode == "transfer_item" || g.churchMode == "transfer_dest" {
			fillBox(120, 90, 400, 260)
			title := "選擇來源角色"
			if g.churchMode == "transfer_item" {
				title = "選擇未裝備物品"
			} else if g.churchMode == "transfer_dest" {
				title = "選擇目的角色"
			}
			g.font.Draw(screen, title, 150, 108, 1.2, color.RGBA{0xff, 0xe0, 0x90, 0xff})
			listLen := len(g.churchIDs)
			if g.churchMode == "transfer_item" {
				listLen = len(g.churchTransferItems)
			}
			if listLen == 0 {
				g.font.Draw(screen, "目前沒有可選項目", 150, 150, 1.0, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
			}
			for i := 0; i < listLen; i++ {
				pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
				if i == g.churchSel {
					pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
				}
				label := ""
				if g.churchMode == "transfer_item" {
					label = fmt.Sprintf("物品 %02Xh", g.partyRoster[g.churchTransferSource].Inventory[g.churchTransferItems[i]])
				} else {
					id := g.churchIDs[i]
					name := fmt.Sprintf("角色%d", id)
					if u, ok := g.partyRoster[id]; ok && u.Name != "" {
						name = u.Name
					}
					label = name
				}
				x := 150.0 + float64(i%2)*180
				y := 150.0 + float64(i/2)*26
				g.font.Draw(screen, fmt.Sprintf("%s%s", pre, label), x, y, 1.0, c)
			}
			g.font.Draw(screen, "←→±1／↑↓±2／Enter 確認／ESC 返回", 150, 330, 0.9, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
		} else {
			listLen := len(g.churchIDs)
			if g.churchMode == "class_confirm" {
				listLen = 2
			}
			visibleLen := listLen
			if g.churchMode == "class" && visibleLen > 3 {
				visibleLen = 3
			}
			h := 120 + float64(visibleLen)*26
			fillBox(120, 90, 400, h)
			title := "復活"
			if g.churchMode == "class" {
				title = "轉職"
			} else if g.churchMode == "class_confirm" {
				title = "確定要轉職嗎？"
			}
			g.font.Draw(screen, title, 150, 108, 1.2, color.RGBA{0xff, 0xe0, 0x90, 0xff})
			if listLen == 0 {
				g.font.Draw(screen, "目前沒有符合條件的角色", 150, 150, 1.0, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
			} else if g.churchMode == "class_confirm" {
				target := campaign.ClassChangeBranch{}
				if len(g.churchBranches) == 1 {
					target = g.churchBranches[0]
				}
				g.font.Draw(screen, fmt.Sprintf("目標：%s", campaign.ClassName(target.ClassID)), 150, 150, 1.0, color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
				for i, label := range []string{"是", "否"} {
					pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
					if i == g.churchSel {
						pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
					}
					g.font.Draw(screen, pre+label, 150+float64(i)*100, 180, 1.0, c)
				}
			} else {
				start, visible := 0, len(g.churchIDs)
				if g.churchMode == "class" {
					start, visible = campaign.NativeClassCandidateWindow(len(g.churchIDs), g.churchSel)
				}
				for row := 0; row < visible; row++ {
					i := start + row
					id := g.churchIDs[i]
					pre, c := "　", color.RGBA{0xd0, 0xd8, 0xe8, 0xff}
					if i == g.churchSel {
						pre, c = "▶", color.RGBA{0xff, 0xff, 0xff, 0xff}
					}
					u := g.partyRoster[id]
					g.font.Draw(screen, fmt.Sprintf("%s%s Lv%d", pre, u.Name, u.Lv), 150, 150+float64(row)*26, 1.0, c)
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
		if g.endingNotice != "" {
			g.font.Draw(screen, g.endingNotice, 28, logicalH-30, 0.9,
				color.RGBA{0xd0, 0xd8, 0xe8, 0xff})
		}
	}
}

// nativeImpactSilhouetteColor 是目前 E1 影像證據中的原版紅色（RGB
// 190,0,0）。它只約束剪影近似的顏色，不代表已接上 0x2939d 的 DAC 脈衝。
var nativeImpactSilhouetteColor = color.RGBA{0xbe, 0x00, 0x00, 0xff}

// battleImpactHP 對應 orig_05 可見的命中邊界：守方在命中演出開始前保留原 HP，
// 命中開始後立即顯示扣血後 HP。原始傷害／演出寫入者仍未知，因此不從這個邊界
// 推導其他 native 時序。
func battleImpactHP(prog, impactStart, before, after int) int {
	if prog < impactStart {
		return before
	}
	return after
}

// figaniFrameAtDisplayTick 依 FIGANI descriptor +6 與顯示倍率選出指定 tick
// 的幀。它是 0x2b9a1／0x2935b 的純排程橋，不替幀賦予命中或傷害語意。
func figaniFrameAtDisplayTick(delays []int, ticksPerNative, tick int) (int, bool) {
	if len(delays) == 0 || ticksPerNative <= 0 || tick < 0 {
		return 0, false
	}
	total := 0
	for _, delay := range delays {
		if delay <= 0 {
			return 0, false
		}
		total += delay * ticksPerNative
	}
	if total <= 0 {
		return 0, false
	}
	remaining := tick % total
	for i, delay := range delays {
		span := delay * ticksPerNative
		if remaining < span {
			return i, true
		}
		remaining -= span
	}
	return len(delays) - 1, true
}

// redSilhouette 全紅剪影(快取):目前只作 E1 視覺近似。原版 0x2939d 的命中
// DAC 分支受 raw frame flag、傷害步進與 0x29f72 輸出欄位控制；在這些欄位
// 尚未接入前，不把剪影快取宣稱為原版色盤寫入的等價實作。
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
				out.Set(x, y, nativeImpactSilhouetteColor)
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
	// 走位/伸擊/突刺全在資料裡。幀停留消費原始 descriptor +6 delay，並以
	// FD2_BATTLE_FPT 作為明確的顯示速度倍率；未配對的資源不會建立攻擊演出。
	atkFrames := g.figani[a.atkFig]
	atkFi := a.frameIndex
	if atkFi < 0 {
		atkFi = 0
	}
	if len(atkFrames) > 0 && atkFi >= len(atkFrames) {
		atkFi = len(atkFrames) - 1
	}
	impactFi := len(atkFrames) - 4
	if impactFi < 1 {
		impactFi = 1
	}
	impactS := impactFi * a.fpt
	if a.figaniTimeline != nil {
		if start, ok := a.figaniTimeline.FrameStart(impactFi); ok {
			impactS = start
		}
	}
	impactE := impactS + 8
	// (1) 狀態欄先畫(會被 figure 蓋住一部分,如原版)
	if g.font != nil {
		dhp := battleImpactHP(prog, impactS, a.defHP0, a.defHP1)
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

	// (2) 敵方盜賊 figure(正面;蓋住狀態欄):待機幀依 descriptor +6 排程循環，
	// 貼各幀內嵌 (dx,dy)。缺少排程時 newAtkAnim 已失敗即關閉。
	if fr := g.figani[a.defFig]; len(fr) > 0 {
		fi, ok := figaniFrameAtDisplayTick(g.figaniDelays[a.defFig], a.fpt, prog)
		if !ok || fi >= len(fr) {
			return
		}
		img := fr[fi]
		// E1 紅色剪影近似；原版 DAC 條件尚未由 raw presentation adapter 提供。
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
		// 原版 impact 參考圖（orig_05_attack_03_impact）只把守方受擊者
		// 顯示為紅色剪影；攻方仍使用原本的 FIGANI 幀。未有 raw 旗標前，
		// 不把攻方也染紅成未證實的對稱效果。
		dx, dy := 141.0, 3.0
		if m := g.figMeta[a.atkFig]; atkFi < len(m) {
			dx, dy = float64(m[atkFi][0]), float64(m[atkFi][1])
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(dx*sc, dy*sc)
		screen.DrawImage(img, op)
	}
	// (4) 原版的 VGA DAC 脈衝不能以 RGBA 全畫面紅罩替代。IDA 在
	// 0x2939d 只證實它受 FIGANI frame flag、傷害步進及 sub_29f72 的
	// 原始輸出欄位控制；目前 AttackResult 沒有這些欄位的可追溯來源，
	// 因此此處保持失敗即關閉，不繪製未證實的全畫面效果。
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
	// 名字：原版 0x15f84→0x4ea2a 使用 FDOTHER#4 16×16 glyph；只有
	// 字模／Unicode 索引缺失時才退回現代字型，避免未證實字元讓戰鬥畫面失敗。
	if g.drawNativeBattleName(screen, x, y, name) {
		// native bitmap path consumed the name; do not paint a second TTF layer.
	} else if g.fontNm != nil {
		nx, ny := rnd(x+8*sc), rnd(y+2*sc)-2
		dk := color.RGBA{0x20, 0x30, 0x60, 0xff}
		for _, o := range [][2]float64{{-2, 0}, {2, 0}, {0, -2}, {0, 2}, {2, 2}} {
			g.fontNm.Draw(screen, name, nx+o[0], ny+o[1], 1.0, dk)
		}
		g.fontNm.Draw(screen, name, nx, ny, 1.0, color.RGBA{0xe0, 0xee, 0xff, 0xff})
	}
	// 數字使用原版 6×8 digit cell 與 native 7px advance；這只固定素材與
	// 局部幾何，不代表完整戰鬥狀態欄已達逐像素 E2。
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

// mapSpriteGroup chooses the exact native raw FDICON key only for a battle
// State whose whole construction sequence materialized successfully. Story
// actors remain on their editable legacy Fig path by construction.
func (g *Game) mapSpriteGroup(u *battle.Unit) int {
	if g.st != nil {
		if key, ok := g.st.NativeMapSpriteKey(u); ok {
			return key
		}
	}
	return u.Fig
}

// drawUnitSprite 畫一個單位:純 FDICON Q 版 sprite(原版無 HP bar/腳標,還原乾淨)。
// 用方向走動分鏡(FDICON 12幀=4方向×3:站/抬左手/抬右手);行軍時套用 OffX/OffY 位移。
// spriteGroup is either a proven native raw key or an explicit legacy Fig;
// this helper never infers one from the other.
func (g *Game) drawUnitSprite(screen *ebiten.Image, x, y, w, h float64, u *battle.Unit, spriteGroup int) {
	x += u.OffX // 行軍/移動位移
	y += u.OffY
	frames := g.sprites[spriteGroup]
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

// nativeFDOTHERPath opts into original UI data without distributing it. A
// player may point FD2_ORIGINAL_FDOTHER at their own FLAME2/FDOTHER.DAT, or
// install it as the user-overridable assets/original/FDOTHER.DAT.
func nativeFDOTHERPath() string {
	if path := os.Getenv("FD2_ORIGINAL_FDOTHER"); path != "" {
		return path
	}
	path := assetPath("assets/original/FDOTHER.DAT")
	if fileExists(path) {
		return path
	}
	return ""
}

func loadNativeUIPalette() color.Palette {
	datPath := nativeFDOTHERPath()
	if datPath == "" {
		return nil
	}
	raw, err := fdother.ReadResource(datPath, 0)
	if err != nil {
		return nil
	}
	palette, err := fdother.ParseVGAPalette(raw)
	if err != nil {
		return nil
	}
	return palette
}

const nativeActionOverlayCellCount = 78

func loadNativeActionCells(palette color.Palette) []*ebiten.Image {
	datPath := nativeFDOTHERPath()
	if datPath == "" || len(palette) < 256 {
		return nil
	}
	cells, err := fdother.DecodeRawCellResource(datPath, 2)
	if err != nil || len(cells) != nativeActionOverlayCellCount {
		return nil
	}
	out := make([]*ebiten.Image, len(cells))
	for i := range cells {
		im, err := cells[i].Paletted(palette)
		if err != nil {
			return nil
		}
		out[i] = ebiten.NewImageFromImage(im)
	}
	return out
}

// loadNativeCommandLabels reads the editable export of FDTXT_000 command
// labels. It is optional because the export is player-provided original text;
// absence deliberately leaves the normalized presentation names intact.
func loadNativeCommandLabels() map[int]string {
	path := assetPath("assets/data/command_labels.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var document struct {
		Entries []struct {
			CommandID int    `json:"command_id"`
			Label     string `json:"label"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil
	}
	labels := make(map[int]string, len(document.Entries))
	for _, entry := range document.Entries {
		if entry.CommandID >= 0 && entry.CommandID < 40 && entry.Label != "" {
			labels[entry.CommandID] = entry.Label
		}
	}
	return labels
}

func loadGame() *Game {
	g := &Game{
		shotFrame:              20,
		nativeMapHUDPersistent: battle.InitialNativeMapHUDPersistentState(),
	}
	g.bgmSource = loadSettings().BGMSource // 音源設定(預設 fm=Sound Blaster)
	if v := os.Getenv("FD2_BGM_SOURCE"); v != "" && bgmSourceName[v] != "" {
		g.bgmSource = v // 覆寫(截圖/測試用)
	}
	g.unitLabels = os.Getenv("FD2_UNIT_LABELS") != ""
	g.cutsceneLog = os.Getenv("FD2_CUTSCENE_LOG") != ""
	g.approximateMode = os.Getenv("FD2_APPROXIMATE") == "1"
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
	if os.Getenv("FD2_ENDING_PREFIX") != "" {
		preview, err := newNativeEndingPreview()
		if err != nil {
			g.loadErr = "native ending: " + err.Error()
			return g
		}
		g.nativeEnding = preview
		// The preview intentionally bypasses map/battle loading, but native
		// 0x2c39b dialogue still uses the ordinary player-provided DATO faces
		// and FD font once it reaches its recovered text branch.
		g.portraits = loadPortraits()
		g.font = loadFont()
		return g
	}
	if err := g.loadMap("assets"); err != nil {
		g.loadErr = err.Error()
		return g
	}
	// 載入單位(M1)
	if st, err := battle.Load(assetPath("assets/map0_units.json")); err == nil {
		g.st = st
		if err := g.bindNativeFutureItemRows(st); err != nil {
			g.loadErr = "native future constructor item rows: " + err.Error()
		} else if err := g.bindNativeMovementCostRows(st); err != nil {
			g.loadErr = "native movement rows: " + err.Error()
		}
	} else if g.loadErr == "" {
		g.loadErr = "units: " + err.Error()
	}
	// 載入劇本 + 套用初始(待命 group + on_battle_start 主角隊進場,doc 25/29)
	if g.st != nil {
		if sc, err := battle.LoadScenario(assetPath("assets/scenarios/ch01.json")); err == nil {
			g.sc = sc
			dialogue, setupErr := sc.SetupChecked(g.st)
			if setupErr != nil {
				g.loadErr = "scenario setup: " + setupErr.Error()
			} else {
				g.dialog = append(g.dialog, dialogue...)
			}
			g.focusOnParty()
		} else if g.loadErr == "" {
			g.loadErr = "scenario: " + err.Error()
		}
	}
	g.sprites = loadSprites()
	g.portraits = loadPortraits()
	g.figani = loadFIGANI()
	if delays, e := loadFIGANIDelays(); e == nil {
		g.figaniDelays = delays
	} else if g.loadErr == "" {
		g.loadErr = "FIGANI delays: " + e.Error()
	}
	g.figMeta = loadFigMeta()
	g.nativeUIPalette = loadNativeUIPalette()
	if nativeFont, nativeGlyphs, err := loadNativeBattleNameAssets(); err == nil {
		g.nativeBattleFont = nativeFont
		g.nativeBattleGlyphs = nativeGlyphs
	}
	g.nativeActionCells = loadNativeActionCells(g.nativeUIPalette)
	if loadSlotsUI, err := loadNativeLoadSlotsUIAssets(); err == nil {
		g.nativeLoadSlotsUI = loadSlotsUI
	}
	if classUI, err := loadNativeClassUIAssets(); err == nil {
		g.nativeClassUI = classUI
		if preparationUI, preparationErr := loadNativePreparationUIAssets(); preparationErr == nil {
			g.nativePreparationUI = preparationUI
		}
		if townUI, townErr := loadNativeTownUIAssets(); townErr == nil {
			g.nativeTownUI = townUI
		}
		if shopUI, shopErr := loadNativeShopUIAssets(classUI); shopErr == nil {
			g.nativeShopUI = shopUI
		}
	}
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
	// Native 0x18d8c result order is attack, spell, item, wait.  There is no
	// separately proven spell icon in the extracted set; keep the historical
	// status asset as the spell slot until the resource table is decoded.
	for i, nm := range []string{"attack", "status", "item", "wait"} {
		if raw, e := os.ReadFile(assetPath("assets/ui/ring_" + nm + ".png")); e == nil {
			if im, _, e2 := image.Decode(bytes.NewReader(raw)); e2 == nil {
				g.ringIcons[i] = ebiten.NewImageFromImage(im)
			}
		}
	}
	if sp, e := battle.LoadSpells(assetPath("assets/spells.json")); e == nil { // 法術表(EXE dump)
		labels := loadNativeCommandLabels()
		g.nativeCommandLabels = labels
		for i := range sp {
			if label := labels[sp[i].ID]; label != "" {
				sp[i].Name = label
			}
		}
		g.spells = sp
		if g.st != nil {
			g.st.SpellBook = append([]battle.Spell(nil), sp...)
		}
	}
	if records, e := battle.LoadNativeCommandRecords(assetPath("assets/spells.json")); e == nil {
		g.nativeCommandBook = records
		g.bindNativeCommandBook(g.st)
	} else if g.loadErr == "" {
		g.loadErr = "native command records: " + e.Error()
	}
	if resistances, e := battle.LoadNativeCommandResistances(assetPath("assets/data/native_command_resistances.json")); e == nil {
		g.nativeCommandResistances = resistances
		g.bindNativeCommandResistances(g.st)
	} else if g.loadErr == "" {
		g.loadErr = "native command resistances: " + e.Error()
	}
	learnPath := assetPath("assets/data/command_learn.json")
	if !fileExists(learnPath) {
		learnPath = "../docs/data/exe_tables/command_learn.json"
	}
	if table, e := battle.LoadCommandLearn(learnPath); e == nil {
		g.commandLearn = table
		g.bindCommandLearn(g.st)
	} else if g.loadErr == "" {
		g.loadErr = "command learn: " + e.Error()
	}
	if commands, e := campaign.LoadAICommandSpellMap(assetPath("assets/data/item.json")); e == nil && g.st != nil {
		g.st.AICommandSpell = commands
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
	joinPath := assetPath("assets/data/native_join_constructor.json")
	if table, e := campaign.LoadNativeJoinConstructorTable(joinPath); e == nil {
		g.nativeJoinConstructor = table
		g.hasNativeJoinConstructor = true
	}
	if table, e := campaign.LoadNativeJoinBaseTable(assetPath("assets/data/native_join_base_units.json")); e == nil {
		g.nativeJoinBases = table
		g.hasNativeJoinBases = true
	}
	if rows, e := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	); e == nil {
		g.nativeJoinItemEffectRows = rows
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
	g.sfxSwing = loadWav("assets/sfx/battle_48_00.wav")      // 揮擊(池 sub0,七池共用)
	g.sfxImpact = loadWav("assets/sfx/battle_64_00.wav")     // 命中(最短最尖池)
	g.sfxDeath = loadWav("assets/sfx/battle_88_00.wav")      // 陣亡/重擊(最長池)
	g.sfxTransition = loadWav("assets/sfx/battle_88_01.wav") // ch24 FDOTHER #88 sub1
	g.sfxSpawnIntro = loadWav("assets/sfx/battle_95_00.wav") // 0x32999 pass1 FDOTHER #95 sub0
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
	if shotState := os.Getenv("FD2_SHOT_LOAD_STATE"); shotState != "" {
		selection, ok := parseNativeLoadSlotShotState(shotState)
		if !ok || g.shotPath == "" || g.nativeLoadSlotsUI == nil {
			g.loadErr = "FD2_SHOT_LOAD_STATE expects selection 0..3 with native load-slot assets and FD2_SHOT"
			return g
		}
		g.titlePhase = "loadslots"
		g.titleSlotSel = selection
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
				g.partyMembers = map[int]bool{9: true}
				g.partyJoinOrder = []int{9}
				g.partyRoster = map[int]battle.Unit{9: {
					Camp: battle.Own, Name: "悠妮", ClsName: "法師", ClassID: 5,
					Lv: 20, HP: 80, MaxHP: 80, MP: 20, MaxMP: 20,
					AP: 30, DP: 20, DX: 10, MV: 5, HIT: 10, EV: 10,
					Portrait: 9, Fig: 9, BattleFig: 9,
					NativeIdentity: 9, HasNativeIdentity: true,
					MapSelectorKey: 9, HasMapSelectorKey: true, OnField: true,
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
			if binding := os.Getenv("FD2_SHOT_PARTY_BINDING"); binding != "" {
				if err := g.materializeShotPartyFromBinding(binding); err != nil {
					g.loadErr = "shot party binding: " + err.Error()
				}
			}
			if n := os.Getenv("FD2_CAMP_NODE"); n != "" { // 驗證鉤子:直接跳指定節點
				if _, ok := c.Nodes[n]; ok {
					g.camp.Cur = n
					g.camp.Flags["found_secret"] = os.Getenv("FD2_CAMP_SECRET") != ""
				}
			}
			g.enterNode()
			// 只在明確的截圖證據模式壓縮既有 cutscene 的等待時間。一般
			// 玩家輸入、正常戰役與存檔永遠不會走這條分支；若任何 beat
			// 或原生 renderer 尚未閉合，保留 loadErr 而停止，不產生假正式畫面。
			if g.shotPath != "" && os.Getenv("FD2_SHOT_FAST_FORWARD") == "1" &&
				g.camp.Cur == "story_ch00_handler" && g.loadErr == "" {
				if err := g.fastForwardShotCampaign(); err != nil {
					g.loadErr = "shot campaign fast-forward: " + err.Error()
				}
			}
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

// drawUnitHUD 是 native full-frame admission 失敗時的 playable fallback，
// 復用 full-screen battle panel 的 FDOTHER#5 LMI1 #22 frame。ch01 production
// path 已由 ComposeNativeFrame 接入完整 0x1acf3 HUD；本函式不可再用來描述
// native HUD 的完成度。AP/DP/MV 素材也未證實屬於 #22，故 fallback 仍以文字補列。
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

// drawNativeMapHUD presents the proven indexed 0x1acf3 panel/terrain/AP/DP
// subpasses at the native 320x200 surface. It returns false on any missing or
// unverified raw input, preserving the legacy approximation without partial
// drawing. Raw gates/anchor and optional unit/HP must all come from the
// materialized battle runtime; this function never hardcodes native globals.
func (g *Game) nativeMapHUDInput() (indexedmap.NativeMapHUDInput, bool) {
	a := g.nativeMapAssets
	if !nativeMapAssetsAvailable(a) || g.m == nil || g.st == nil ||
		!g.st.HasNativeMapHUDState || !g.st.HasNativeMapCycleState || g.st.NativeMapSelectorCache == nil ||
		g.curX < 0 || g.curY < 0 || g.curX >= g.m.W || g.curY >= g.m.H {
		return indexedmap.NativeMapHUDInput{}, false
	}
	tile := g.m.Tiles[g.curY*g.m.W+g.curX]
	if tile < 0 || tile >= len(a.Controls)/4 || tile > 0x3ff {
		return indexedmap.NativeMapHUDInput{}, false
	}
	control := a.Controls[tile*4+1]
	rawHUD := g.st.NativeMapHUDState
	in := indexedmap.NativeMapHUDInput{
		DisplayGateA: rawHUD.DisplayGateA != 0,
		DisplayGateB: rawHUD.DisplayGateB != 0,
		AnchorX:      rawHUD.AnchorX, TerrainDescriptor: tile, TerrainControl: control,
	}
	if u := g.st.UnitAt(g.curX, g.curY); u != nil {
		if !u.HasMapSelectorSlot || !u.HasBattleFig || u.BattleFig < 0 || u.BattleFig > 0xff ||
			!u.HasNativeRecordRace || !u.HasNativeRecordByte6 ||
			!u.HasNativeRecordWord42 || u.HP < 0 || u.HP > 0xffff {
			return indexedmap.NativeMapHUDInput{}, false
		}
		if indexedmap.NativeMapHUDOptionalUnitEligible(
			byte(u.BattleFig), u.NativeRecordRace, u.NativeRecordByte6,
		) {
			in.OptionalUnit = &indexedmap.NativeMapHUDOptionalUnit{
				SelectorSlot: u.MapSelectorSlot,
				RawState:     g.st.NativeMapCycleState.Idle,
				Current:      uint16(u.HP),
				Maximum:      u.NativeRecordWord42,
			}
		}
	}
	return in, true
}

func (g *Game) drawNativeMapHUD(screen *ebiten.Image) bool {
	a := g.nativeMapAssets
	in, ok := g.nativeMapHUDInput()
	if !ok {
		return false
	}
	frame := make([]byte, fdicon.NativeMapStride*200)
	if err := indexedmap.BlitNativeMapHUD(a.Frames, a.Terrain, a.Units, g.st.NativeMapSelectorCache, frame, in); err != nil {
		return false
	}
	overlayPalette := append(color.Palette(nil), a.Palette...)
	r, green, b, _ := overlayPalette[0].RGBA()
	overlayPalette[0] = color.NRGBA{
		R: uint8(r >> 8), G: uint8(green >> 8), B: uint8(b >> 8), A: 0,
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), overlayPalette)
	for y := 0; y < 200; y++ {
		copy(img.Pix[y*img.Stride:y*img.Stride+320], frame[y*fdicon.NativeMapStride:y*fdicon.NativeMapStride+320])
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	return true
}

// drawNativeMapFrame is the production 0x11cac bridge for the currently
// materialized neutral tactical state. It composes terrain, range, units,
// foreground and HUD atomically, then presents the corrected 312x192 viewport
// at VGA (4,4) on a 320x200 paletted surface.
func (g *Game) drawNativeMapFrame(screen *ebiten.Image) bool {
	if err := g.composeNativeMapFrame(); err != nil {
		return false
	}
	a := g.nativeMapAssets
	palette := a.Palette
	if len(g.nativeMapDAC) == 256*3 {
		if current, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC); err == nil {
			palette = current
		}
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(img.Pix, g.nativeMapVGA)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	return true
}

func (g *Game) composeNativeMapFrame() error {
	now := time.Now()
	if g != nil && (g.shotPath != "" || g.shotSeries != "") && os.Getenv("FD2_SHOT_DETERMINISTIC") == "1" {
		// 截圖證據只取固定 60 Hz 虛擬時鐘；一般玩家仍使用實際 BIOS
		// 時鐘。這避免 Xvfb 排程差異把同一狀態存成不同動畫幀。
		now = time.Unix(0, int64(g.frame)*int64(time.Second/60))
	}
	return g.composeNativeMapFrameAt(now)
}

// composeNativeMapFrameAt owns one complete 0x11CAC-style transaction:
// sample the BIOS low word, advance all frame-local timing state, build the
// indexed frame, and publish both timing and pixels only after every preflight
// succeeds. The explicit time argument keeps CONTINUE redraw tests
// deterministic.
func (g *Game) composeNativeMapFrameAt(now time.Time) error {
	a := g.nativeMapAssets
	hud, ok := g.nativeMapHUDInput()
	if !ok || g.st == nil || !g.st.HasNativeMapRangeModeState ||
		g.st.NativeMapRangeMode < 0 || g.st.NativeMapRangeMode > 5 {
		return errors.New("native map frame: HUD or drawable selector state unavailable")
	}
	candidateState := *g.st
	candidateClock := g.nativeMapClock
	candidateGame := Game{st: &candidateState, nativeMapClock: candidateClock}
	if !candidateGame.advanceNativeMapClock(now) {
		return errors.New("native map frame: timing state unavailable")
	}
	in, err := buildNativeMapFrameInput(
		a, g.m, &candidateState, nativeMapFrameRuntime{HUD: hud},
	)
	if err != nil {
		return err
	}
	if len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize {
		g.nativeMapWork = make([]byte, indexedmap.NativeUnitPresentWorkSize)
	}
	if len(g.nativeMapVGA) != indexedmap.NativeMapVGASize {
		g.nativeMapVGA = make([]byte, indexedmap.NativeMapVGASize)
	}
	if err := indexedmap.ComposeNativeFrame(g.nativeMapWork, g.nativeMapVGA, in); err != nil {
		return err
	}
	*g.st = candidateState
	g.nativeMapClock = candidateGame.nativeMapClock
	return nil
}

func (g *Game) endTurn() {
	if g.st == nil || g.result != "" || g.aiBusy || g.nativeTurnStaging != nil {
		return
	}
	started, err := g.startNativeRawCamp0TurnEvents()
	if err != nil {
		g.loadErr = "native raw camp0 phase: " + err.Error()
		return
	}
	if started {
		return
	}
	g.beginEnemyPhase()
}

func (g *Game) beginEnemyPhase() {
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
	g.battleEvent, g.battleEventDelay, g.camPan = nil, 0, nil
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
		case "native_acting":
			if g.st == nil {
				g.finishBattleEventWithError("native acting 缺少 runtime battle state")
				return
			}
			frames, err := g.loadNativeBattleFollowingActing(
				action.NativeActing, len(g.st.Units),
			)
			if err != nil {
				g.finishBattleEventWithError(err.Error())
				return
			}
			g.startNativeBattleFollowingActing(frames, g.advanceBattleEvent)
			return
		default:
			call, isNativeIntro, introErr := nativeBattleIntroCall(action)
			if introErr != nil {
				g.finishBattleEventWithError(introErr.Error())
				return
			}
			if isNativeIntro {
				if err := g.startNativeBattleSpawnIntro(action, call, g.advanceBattleEvent); err != nil {
					g.finishBattleEventWithError(err.Error())
				}
				return
			}
			dialogue, isDialogue, err := g.sc.ExecuteActionChecked(g.st, action)
			if err != nil {
				g.finishBattleEventWithError(err.Error())
				return
			}
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
	// Keep the executable [0x53bef] counter only for states that entered with
	// explicit native provenance; hand-authored tests/legacy states remain
	// unproven (zero) and therefore fail closed for native round predicates.
	if g.st.NativeRoundCounter > 0 {
		g.st.NativeRoundCounter++
	}
	for _, u := range g.st.Units {
		u.Acted = false
		u.TickStatus()             // buff/封咒/中毒/麻痺回合遞減+中毒扣血(doc02 §6.4)
		g.awardDeathReward(u, nil) // poison/status death shares the same once-only reward path
	}
	started, err := g.startNativeRawCamp2TurnEvents(g.completeTurnPlayerPhase)
	if err != nil {
		g.loadErr = "native raw camp2 phase: " + err.Error()
		return
	}
	if started {
		return
	}
	g.completeTurnPlayerPhase()
}

func (g *Game) completeTurnPlayerPhase() {
	if g.result == "" {
		g.showBanner("PLAYER PHASE")
	}
	g.sel, g.reach, g.moved = nil, nil, false
	g.checkResult()
}

// aiStep AI 回合驅動:一次取一個單位的行動計畫,播行走動畫→到位攻擊(全螢幕演出)。
// 全單位動完 → finishTurn。
func (g *Game) aiStep() {
	if !g.aiBusy || g.walk != nil || g.atk != nil || g.nativeHealPresentation != nil || g.result != "" {
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
	if plan.NativeError != nil {
		// Native mode 2 有明確的原始來源閘門；閘門失敗時不可消耗行動，也不可
		// 靜默替換成正規化 AI。
		g.loadErr = "native AI: " + plan.NativeError.Error()
		g.aiBusy = false
		return
	}
	if len(plan.NativeMode11Stages) > 0 {
		g.startNativeAIMode11(plan)
		return
	}
	u := plan.U
	act := func() {
		if plan.NativeModeWriteRangeZero {
			// The dispatcher writes the raw map-range global [0x51a83] on
			// these branches. Keep the state field explicitly raw; it is not
			// a normalized command-selection value.
			g.st.NativeMapRangeMode = 0
			g.st.HasNativeMapRangeModeState = true
		}
		if plan.SpellID >= 0 {
			if err := g.executeAISpell(plan); err != nil {
				g.loadErr = "敵方 AI 法術：" + err.Error()
				g.aiBusy = false
			}
			return
		}
		if plan.NativeActionKind == battle.NativeAIActionCommand ||
			plan.NativeActionKind == battle.NativeAIActionItem {
			if err := g.executeNativeAIAction(plan); err != nil {
				g.loadErr = "native AI action: " + err.Error()
				g.aiBusy = false
			}
			return
		}
		finish := func() {
			if plan.NativeModeEventActive {
				audioCue := battle.NativeAIMode5AudioCueForRawTail()
				if os.Getenv("FD2_MUTE") == "" && g.shotPath == "" &&
					(g.sfx == nil || len(g.sfx[audioCue.Index]) == 0) {
					g.loadErr = fmt.Sprintf("native AI mode 5 raw sample unavailable: resource=%d index=%d", audioCue.ResourceID, audioCue.Index)
					g.aiBusy = false
					return
				}
				emitMode5Audio := func(cue battle.NativeAIMode5AudioCue) {
					// Native 0x13D0D calls 0x25B45([0x53EE8],12,1).
					// The extracted FDOTHER #31 sample index is the only
					// accepted mapping; do not substitute a normalized SFX.
					if cue.HandleLinearAddress == battle.NativeAIMode5AudioHandle &&
						cue.ResourceID == battle.NativeAIMode5AudioResource &&
						cue.Index == battle.NativeAIMode5AudioIndex &&
						cue.LoopCount == battle.NativeAIMode5AudioLoopCount {
						g.playSFX(cue.Index)
					}
				}
				if err := g.st.ApplyNativeAIMode5EventWithAudioCue(
					u, plan.NativeModeEventID, plan.NativeModeEventDestination,
					emitMode5Audio,
				); err != nil {
					g.loadErr = "native AI mode event: " + err.Error()
					g.aiBusy = false
					return
				}
			}
			if plan.NativeModeWriteByte5 {
				// 0x32975 writes the complete runtime +0x05 byte only after
				// mode 7's raw destination comparison succeeds.  Keep the
				// mutation on the native field; Acted remains the engine
				// projection updated by the common completion owner below.
				u.NativeRecordByte5 = 1
				u.HasNativeRecordByte5 = true
			}
			g.finishSuccessfulUnitAction(u, nil)
		}
		if plan.Target != nil && plan.Target.Alive() {
			tgt := plan.Target
			u.SetMapPose(dirToward(u.X, u.Y, tgt.X, tgt.Y))
			nm, anm := tgt.Name, u.Name
			if nm == "" {
				nm = tgt.ClsName
			}
			if anm == "" {
				anm = u.ClsName
			}
			hp0 := tgt.HP
			attackResult, err := g.resolvePhysicalAttack(u, tgt)
			if err != nil {
				// The normalized planner must not consume an action when the
				// production RNG boundary is unavailable.
				g.loadErr = "AI physical attack: " + err.Error()
				g.aiBusy = false
				return
			}
			g.awardDeathReward(tgt, u)
			g.msg = playerPhysicalAttackMessage(u, tgt, attackResult)
			g.atk = g.newAtkAnim(u.BattleFig, tgt.BattleFig, anm, nm,
				u.HP, u.MaxHP, u.Lv, u.MP, tgt.Lv, tgt.MP,
				hp0, tgt.HP, tgt.MaxHP, g.terrainAt(tgt.X, tgt.Y), u.Camp == battle.Own)
			if g.atk != nil {
				g.atk.after = finish
			} else {
				g.loadErr = fmt.Sprintf("AI FIGANI attack presentation unavailable: %d -> %d", u.BattleFig, tgt.BattleFig)
				finish()
			}
			g.checkResult()
			return
		}
		finish()
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
