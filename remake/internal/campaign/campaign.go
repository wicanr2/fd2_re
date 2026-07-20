// Package campaign — 劇本節點圖系統(doc 19):把「固定線性流程」變成可分支的有向圖。
// 節點 = 一個遊戲段落(story/battle/town/preparation/choice/event/ending),轉場依結果(win/lose/next/optN)
// 與旗標決定下一節點;敗北可走敗北路線而非 game over。
package campaign

import (
	"encoding/json"
	"fmt"
	"os"
)

// Line 一句對話(story 節點內嵌)。Speaker 是靜態或稽核用 DATO 頭像 id；
// SpeakerSlot 保存原版 FFED/FFEC 的 runtime unit direct index，執行時必須
// 從該 unit 的 Portrait 解析，不能把 slot 數字誤當全域角色 id。
type Line struct {
	Speaker     int    `json:"speaker"`
	SpeakerSlot *int   `json:"speaker_slot,omitempty"`
	Upper       *bool  `json:"upper,omitempty"`
	Text        string `json:"text"`
}

// Option choice 節點的選項;If 非空時需旗標為真才顯示。
type Option struct {
	Label string `json:"label"`
	To    string `json:"to"`
	If    string `json:"if,omitempty"`
}

// Good 商店商品(名稱/價格;id 對映 EXE item.json)。
type Good struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

// Actor story+Map 場景背景上的靜態角色(cutscene NPC/主角,無 AI/戰鬥邏輯,純擺位;
// Fig=地圖 sprite 組 id(同 battle.Unit.Fig,恆等於角色 id,doc31);Dir:0下1左2上3右。
// 座標/肖像多可直接取自 FDFIELD 該地圖出場位置段(見 tools/parse_field.py positions),
// 見 doc23 §4 補述(map32 王座廳:國王portrait48@(7,5)/王后portrait66@(10,5))。
//
// FromX/FromY/WalkFrames(doc46 §5.3,走位動畫):若 WalkFrames>0 且 From 與 X/Y 不同,
// 進節點時該角色從 (FromX,FromY) 走位到 (X,Y),耗時 WalkFrames 幀(60fps);
// 例:map31 密林「發現悠妮與蓋亞」一幕,索爾/亞雷斯從比劍點走到發現點(FDFIELD 出場位置證實
// 兩組座標相距 14 格,非同格瞬移,見 doc46 §2)。
type Actor struct {
	Fig        int `json:"fig"`
	X          int `json:"x"`
	Y          int `json:"y"`
	Dir        int `json:"dir,omitempty"`
	FromX      int `json:"from_x,omitempty"`
	FromY      int `json:"from_y,omitempty"`
	WalkFrames int `json:"walk_frames,omitempty"`
}

// ActorWalk 節點「退場」走位動畫(doc46 §5.3):對白播完、換場前,已在場的 actor 先走一段路
// 再淡出(例:王座廳索爾對白說完沿紅毯走下場,~1.5s)。Fig 指定 Node.Actors 裡哪一個角色。
type ActorWalk struct {
	Fig    int  `json:"fig"`
	ToX    int  `json:"to_x"`
	ToY    int  `json:"to_y"`
	Frames int  `json:"frames"`
	Dir    *int `json:"dir,omitempty"` // 走完後面向(指標,nil=保留走位末向;指定則覆蓋,如索爾走到亞雷斯旁定住面右)
}

// ActingUnit is one original acting frame target.  Slot is the original
// FDFIELD/unit-array index and is the authoritative identity for decoded
// 0x1366a data: a roster can contain many guards with the same Fig.  Fig is a
// legacy fallback for hand-authored scenes that have no original roster; an
// editable transcription of a decoded handler must use Slot instead.  Pose
// follows the original direction encoding: 0 down, 1 left, 2 up, 3 right.
type ActingUnit struct {
	Slot *int `json:"slot,omitempty"`
	Fig  int  `json:"fig,omitempty"`
	Pose int  `json:"pose"`
}

// ActingFrame 是原版 0x1366a 資源的一幀行為轉錄，不包含原始 bytes。
// Special=false = bit7=0 正常模式：每個 Beat 沿 Pose 移動一格；Special=true = bit7=1：
// 原地顯示／姿態，Beat 只表示停留節奏。規則見 doc50 §1.2。
type ActingFrame struct {
	Beats   int          `json:"beats"`
	Special bool         `json:"special,omitempty"`
	Units   []ActingUnit `json:"units"`
}

// LoadCHState is the editable remake state selected by an original LOADCH
// call.  The original routine loads all three together: FDFIELD map, FDFIELD
// roster, and FDTXT (doc23 §4).  Keeping those paths in one value makes an
// incomplete reconstruction impossible to mistake for a harmless map change.
// Paths are asset-root relative (for example assets/maps/map5/map5_units.json)
// rather than relative to the handler binding file.
type LoadCHState struct {
	Chapter       int    `json:"chapter"`
	Map           string `json:"map"`
	Roster        string `json:"roster"`
	SlotCount     int    `json:"slot_count"`
	Script        string `json:"script"`
	PartyScenario string `json:"party_scenario,omitempty"` // persistent party constructed before FDFIELD groups
	PartyOrder    []int  `json:"party_order,omitempty"`    // original JOIN chronology; direct-replay fallback and runtime assertion
	CamX          int    `json:"cam_x,omitempty"`
	CamY          int    `json:"cam_y,omitempty"`
	CamMaxY       int    `json:"cam_max_y,omitempty"`
}

// BeatCondition is the runtime form of a proven handler predicate.
type BeatCondition struct {
	Op        string `json:"op"`
	UnitSlots []int  `json:"unit_slots,omitempty"`
}

// HandlerUnitLayout is one absolute runtime-slot placement recovered from a
// native post-battle layout routine. Coordinates remain original map tiles;
// CamX/CamY below are the verified remake pixel origin.
type HandlerUnitLayout struct {
	Slot int `json:"slot"`
	X    int `json:"x"`
	Y    int `json:"y"`
	Pose int `json:"pose"`
}

type HandlerLayout struct {
	Units []HandlerUnitLayout `json:"units"`
	CamX  int                 `json:"cam_x"`
	CamY  int                 `json:"cam_y"`
}

// Beat 過場原語(doc 50 §1/§2):cutscene 節點的 beats 通常依序執行；if 會在 runtime
// 選一條 structured arm 插入目前拍之後，再回到共同 continuation。
// 一比一對映原版 EXE handler 的呼叫序列(LOADCH/PAN/TXT/ACT/SPAWN/JOIN/BGM/FADE/DELAY)。
// 每個 op 只用到自己相關的欄位,其餘留零值即可(同 Node 的稀疏欄位風格)。
type Beat struct {
	Op             string                 `json:"op"`               // loadch/pan/walk/dialog/act/spawn/spawn_intro/deactivate_unit/reset_pose/redraw/...
	Source         string                 `json:"source,omitempty"` // original handler call-site; empty for authored-only beats
	Condition      *BeatCondition         `json:"condition,omitempty"`
	Then           []Beat                 `json:"then,omitempty"`
	Else           []Beat                 `json:"else,omitempty"`
	RuntimeContext *HandlerRuntimeContext `json:"runtime_context,omitempty"`
	Layout         *HandlerLayout         `json:"layout,omitempty"`

	// loadch: atomically replace the active map, FDFIELD roster and FDTXT
	// story context.  It is deliberately a nested required state object so a
	// handler cannot compile a map-only imitation of original 0x205da.
	LoadCH *LoadCHState `json:"loadch,omitempty"`

	// pan/walk 共用:目標格(walk 用 X/Y 當終點);pan 的 X/Y 沿用 Node.CamX/CamY 語意——
	// 已由畫面回饋校準的「像素座標」,不是 doc47 §3 原始 grid(col,row)值(grid→px 未逐點驗證,
	// 不自行換算,見 rulebook 62)。
	X        int  `json:"x,omitempty"`
	Y        int  `json:"y,omitempty"`
	FromX    int  `json:"from_x,omitempty"` // walk 起點;省略=沿用該角色目前座標(接續上一拍)
	FromY    int  `json:"from_y,omitempty"`
	Fig      int  `json:"fig,omitempty"`       // walk/act:對應 Node.Actors 裡的角色(依 Fig 尋找,同 ActorWalk)
	Slot     *int `json:"slot,omitempty"`      // original runtime unit-array slot; identity-critical handler primitives only
	Frames   int  `json:"frames,omitempty"`    // pan/walk/fade 位移或漸變幀數;delay 用幀數(見 Ms)
	TileStep bool `json:"tile_step,omitempty"` // pan:0x135dd 每 tick 先 X 後 Y 移一個 tile
	Follow   bool `json:"follow,omitempty"`    // walk:走位期間鏡頭鎖定跟隨(doc47 §9,同 Node.FollowWalk 機制)
	Dir      *int `json:"dir,omitempty"`       // walk:走完後面向(指標,nil=保留走位末向;指定則面向它,如索爾走前面轉身面向亞雷斯)
	Steps    int  `json:"steps,omitempty"`     // scroll_step:原版 0x13185 的重複上移格數

	// act:Acting 非空時播放原版 acting frame 的行為轉錄：正常 frame 每 Beat 依 Pose 搬一格，
	// special frame 只原地換姿態(doc50 §1.2)。Poses/PoseFrames 是舊的原地姿態近似欄位，
	// 為舊場景相容而保留；不可與 Acting 混用。
	Acting     []ActingFrame `json:"acting_frames,omitempty"`
	Poses      []int         `json:"poses,omitempty"`
	PoseFrames int           `json:"pose_frames,omitempty"` // 每個 pose 停留幀數(預設見 main.go)

	// dialog:章文本第 Line 條起連續 Count 句(Count 省略=1)。Line 對應目前節點 Script+Scene
	// 載入的那份 lines(同 Node.Scene 語意),不是 FDTXT 原始 idx(譯文精校版常把一條原文拆成
	// 多句對白,見 doc47 §7 教訓:機制懂了但內容沒逐句對齊前不假裝一一對應)。
	Line       int    `json:"line,omitempty"`
	Count      int    `json:"count,omitempty"`
	Script     string `json:"script,omitempty"`      // handler compiler context; empty=Node.Script
	Scene      string `json:"scene,omitempty"`       // handler compiler context; empty can mean unlabeled scene
	SceneIndex *int   `json:"scene_index,omitempty"` // authoritative for unlabeled/reused scene labels

	// Upper:dialog 對話框上下位置覆蓋(指標,nil=沿用預設規則「說話者 id>=32 走上框」)。
	// 草地撞見幕實測(doc55 截圖 18-03-10):亞雷斯(id4,<32)進場那句仍走上框——原版並非單純按
	// id 分上下框,推測與進場/位置有關,尚未逆得通則;先開這個 per-beat 覆蓋做最小修正,別動全域規則。
	Upper *bool `json:"upper,omitempty"`

	Group int `json:"group,omitempty"` // spawn:群組編號(doc25 spawn(g));remake 無群組資料表,僅記錄,見 main.go stub 註解
	// CharID is JOIN's permanent-player identity.  It intentionally remains
	// separate from a scene actor's Fig/portrait: JOIN accepts only the
	// original 0..31 player roster, while a cutscene may contain arbitrary
	// NPC portraits (for example map31's shop-clerk portrait 75).
	CharID int    `json:"char_id,omitempty"`
	Track  string `json:"track,omitempty"` // bgm:曲目 id(對映 assets/bgm)

	Out bool `json:"out,omitempty"` // fade:true=淡出 false=淡入(重用 storyFade,doc46 §5.2)

	Ms int `json:"ms,omitempty"` // delay:毫秒(原版 0x375b2 語意);換算成 60fps 幀數,Frames 優先

	// set_chapter: original [0x53c03] campaign/resource chapter assignment.
	// Pointer form preserves chapter zero as an explicit editable value.
	Chapter *int `json:"chapter,omitempty"`

	// grant_item: original unsigned-byte item identity.
	ItemID *int `json:"item_id,omitempty"`
}

// Node 節點。Type: story / cutscene / battle / town / preparation / church / choice /
// inventory_gate / inventory_recipe / event / shop / ending。
// cutscene(doc 50):story 的 beats 驅動版——用 Beats 一比一承接原版章 handler 的原語序列,
// 對白與走位/演出天然交錯(平面序列,非「一幕一段」)。Map/Actors/BGM/ExitWalk(s) 等欄位與
// story 共用同一套場景設置(進節點時的初始擺位、退場走位、淡出轉場),Beats 只負責節點「進行中」
// 的编排;story 節點型別保留相容,兩者可並存於同一 campaign(逐步遷移,doc50 §2)。
type Node struct {
	Type     string `json:"type"`
	Lines    []Line `json:"lines,omitempty"`    // story:對白(內嵌;Script 有檔時被覆蓋)
	Script   string `json:"script,omitempty"`   // story:本機劇情文本檔(assets/story/chNN.json,不入庫)
	Scenario string `json:"scenario,omitempty"` // battle:戰場事件劇本檔
	Map      string `json:"map,omitempty"`      // battle:戰場資產目錄;story:場景背景圖(doc23 §4:
	// 原版序幕王城/草地背景是 FDFIELD map32 複合場景,與戰場同一渲染器非另開圖片系統;
	// story 填同一 assets/maps/mapN 目錄即可換場景背景;battle 空=沿用當前)
	Units  string  `json:"units,omitempty"` // battle:單位配置檔
	CamX   int     `json:"cam_x,omitempty"` // story+Map:固定鏡頭像素座標(場景不跟游標走,取代預設 focusOnParty)
	CamY   int     `json:"cam_y,omitempty"`
	Actors []Actor `json:"actors,omitempty"` // story+Map:場景背景上的靜態角色擺位
	Scene  string  `json:"scene,omitempty"`  // story+Script:只取 Script 檔裡 label 對映的那個 scene(doc46 §5.2;
	// 空=舊行為,整份 Script 攤平全部 scenes 成一條對白隊列——別讓一個節點播完整份劇本)
	ExitWalk  *ActorWalk  `json:"exit_walk,omitempty"`  // story:對白播完、換場前先走一段路再淡出(doc46 §5.3;單一角色)
	ExitWalks []ActorWalk `json:"exit_walks,omitempty"` // 多角色同時退場(全部走完才轉場)。
	// ⚠ 更正(doc55 逐幀量測 2026-07-05):早前「草地幕兩人一起走離」是錯的——實測=亞雷斯對話中先走近、
	// 索爾對話後才單獨走到亞雷斯旁、隨即淡出(無一起走離畫面)。草地幕已改「亞雷斯進場走位+索爾單人 ExitWalk」;
	// 本欄保留供其他真有「多人同時退場」的幕使用;並行多角色不在 Beat.walk 單角色設計內,收尾沿用本欄不重造輪子)
	Beats          []Beat `json:"beats,omitempty"`           // cutscene:過場原語序列(doc 50);Beats 跑完後走 ExitWalk(s)+淡出+Advance,同 story 節點收尾
	HandlerBinding string `json:"handler_binding,omitempty"` // cutscene:editable handler binding; runtime must reject unresolved compile issues
	AutoAdvance    int    `json:"auto_advance,omitempty"`    // story:無對白/Script 時,進節點後幾幀自動轉場(doc46 行軍蒙太奇)
	WalkFirst      bool   `json:"walk_first,omitempty"`      // story:進場走位全走完才顯示對白(2-1:王座廳索爾沿紅毯走到王座前對話框才出現)
	FollowWalk     bool   `json:"follow_walk,omitempty"`     // story:走位期間鏡頭鎖定跟隨走位者(原版 13×8 格視野長廊運鏡,doc25 0x11eee)
	CamMaxY        int    `json:"cam_max_y,omitempty"`       // story:鏡頭 Y 上限(px;0=不限)。王座廳=808 擋住 map32 底部草地段
	// (原版第一幕畫面無草地,索爾從畫面外沿紅毯走入,使用者回饋 2026-07-04 #1)
	BGM             string          `json:"bgm,omitempty"`
	Next            string          `json:"next,omitempty"`             // story/event
	OnWin           string          `json:"on_win,omitempty"`           // battle
	OnLose          string          `json:"on_lose,omitempty"`          // battle(敗北路線;空=game over)
	Protect         string          `json:"protect,omitempty"`          // battle:保護目標；空值沿用主角索爾
	ItemID          *int            `json:"item_id,omitempty"`          // inventory_gate:原版 unsigned-byte item identity
	IfPresent       string          `json:"if_present,omitempty"`       // inventory_gate:全隊任一角色持有 ItemID
	IfMissing       string          `json:"if_missing,omitempty"`       // inventory_gate:全隊皆未持有 ItemID
	ItemIDs         []int           `json:"item_ids,omitempty"`         // inventory_recipe:逐 item×runtime slot 計數／移除
	SlotCount       int             `json:"slot_count,omitempty"`       // inventory_recipe:只掃前 N 個 runtime records
	RequiredMatches int             `json:"required_matches,omitempty"` // inventory_recipe:原版要求的精確命中組合數
	RewardItemID    *int            `json:"reward_item_id,omitempty"`   // inventory_recipe:成功後 grant 的 item
	IfCrafted       string          `json:"if_crafted,omitempty"`       // inventory_recipe:成功 arm
	IfInsufficient  string          `json:"if_insufficient,omitempty"`  // inventory_recipe:命中數不符 arm
	Prompt          string          `json:"prompt,omitempty"`           // choice/preparation
	PartyLimit      int             `json:"party_limit,omitempty"`      // preparation: original 0x318ad selection cap (15, late route 19)
	Town            string          `json:"town,omitempty"`             // town:原版戰後城鎮/營地名稱(可編輯、可存檔的整備 hub)
	Options         []Option        `json:"options,omitempty"`          // choice
	SetFlags        map[string]bool `json:"set_flags,omitempty"`
	Text            string          `json:"text,omitempty"`      // ending:結語
	Goods           []Good          `json:"goods,omitempty"`     // shop:商品
	Secret          []Good          `json:"secret,omitempty"`    // shop:祕密商店商品
	SecretIf        string          `json:"secret_if,omitempty"` // shop:旗標為真才開祕密商品(原版祕密商店機制)
}

// Campaign 整張節點圖。
type Campaign struct {
	Title string           `json:"title"`
	Start string           `json:"start"`
	Flags map[string]bool  `json:"flags"`
	Nodes map[string]*Node `json:"nodes"`
}

// Load 讀 campaign.json 並驗證轉場目標都存在。
func Load(path string) (*Campaign, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Campaign
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if _, ok := c.Nodes[c.Start]; !ok {
		return nil, fmt.Errorf("start 節點 %q 不存在", c.Start)
	}
	check := func(from, to string) error {
		if to == "" {
			return nil
		}
		if _, ok := c.Nodes[to]; !ok {
			return fmt.Errorf("節點 %q 的轉場目標 %q 不存在", from, to)
		}
		return nil
	}
	for id, n := range c.Nodes {
		for _, to := range []string{n.Next, n.OnWin, n.OnLose, n.IfPresent, n.IfMissing, n.IfCrafted, n.IfInsufficient} {
			if err := check(id, to); err != nil {
				return nil, err
			}
		}
		if n.Type == "inventory_gate" {
			if n.ItemID == nil || *n.ItemID < 0 || *n.ItemID > 255 {
				return nil, fmt.Errorf("inventory_gate 節點 %q 的 item_id 必須是 0..255", id)
			}
			if n.IfPresent == "" || n.IfMissing == "" {
				return nil, fmt.Errorf("inventory_gate 節點 %q 必須同時定義 if_present / if_missing", id)
			}
		}
		if n.Type == "inventory_recipe" {
			if len(n.ItemIDs) == 0 || n.SlotCount <= 0 || n.RequiredMatches <= 0 {
				return nil, fmt.Errorf("inventory_recipe 節點 %q 必須定義 item_ids / slot_count / required_matches", id)
			}
			for _, itemID := range n.ItemIDs {
				if itemID < 0 || itemID > 255 {
					return nil, fmt.Errorf("inventory_recipe 節點 %q 的 item_ids 必須是 0..255", id)
				}
			}
			if n.RewardItemID == nil || *n.RewardItemID < 0 || *n.RewardItemID > 255 {
				return nil, fmt.Errorf("inventory_recipe 節點 %q 的 reward_item_id 必須是 0..255", id)
			}
			if n.IfCrafted == "" || n.IfInsufficient == "" {
				return nil, fmt.Errorf("inventory_recipe 節點 %q 必須同時定義 if_crafted / if_insufficient", id)
			}
		}
		for _, o := range n.Options {
			if err := check(id, o.To); err != nil {
				return nil, err
			}
		}
	}
	return &c, nil
}

// Runner 執行狀態:目前節點 + 旗標。
type Runner struct {
	C     *Campaign
	Cur   string
	Flags map[string]bool
}

// NewRunner 從起點開跑(複製初始旗標)。
func NewRunner(c *Campaign) *Runner {
	f := map[string]bool{}
	for k, v := range c.Flags {
		f[k] = v
	}
	return &Runner{C: c, Cur: c.Start, Flags: f}
}

// Node 目前節點。
func (r *Runner) Node() *Node { return r.C.Nodes[r.Cur] }

// NodeID exposes the editable node key for runtime data fallback decisions.
// Keeping this separate from Node avoids making callers infer identity from
// the node payload (which is intentionally allowed to be identical across
// multiple story segments).
func (r *Runner) NodeID() string { return r.Cur }

// Visible 回傳 choice 節點依旗標過濾後的選項。
func (r *Runner) Visible() []Option {
	n := r.Node()
	var out []Option
	for _, o := range n.Options {
		if o.If == "" || r.Flags[o.If] {
			out = append(out, o)
		}
	}
	return out
}

// ShopGoods shop 節點的商品(祕密商店:SecretIf 旗標為真時加開 Secret 商品)。
func (r *Runner) ShopGoods() []Good {
	n := r.Node()
	out := append([]Good{}, n.Goods...)
	if n.SecretIf != "" && r.Flags[n.SecretIf] {
		out = append(out, n.Secret...)
	}
	return out
}

// Advance 依結果離開目前節點:套用 set_flags,回傳下一節點 id(""=流程結束/game over)。
// outcome:story/event→忽略;battle→"win"/"lose";choice→"optN"(過濾後 index)。
func (r *Runner) Advance(outcome string) string {
	n := r.Node()
	if n == nil {
		return ""
	}
	for k, v := range n.SetFlags {
		r.Flags[k] = v
	}
	next := ""
	switch n.Type {
	case "battle":
		if outcome == "win" {
			next = n.OnWin
		} else {
			next = n.OnLose
		}
	case "choice", "town":
		var i int
		if _, err := fmt.Sscanf(outcome, "opt%d", &i); err == nil {
			if vis := r.Visible(); i >= 0 && i < len(vis) {
				next = vis[i].To
			}
		}
	case "inventory_gate":
		if outcome == "present" {
			next = n.IfPresent
		} else if outcome == "missing" {
			next = n.IfMissing
		}
	case "inventory_recipe":
		if outcome == "crafted" {
			next = n.IfCrafted
		} else if outcome == "insufficient" {
			next = n.IfInsufficient
		}
	case "ending":
		next = ""
	default: // story / event / shop
		next = n.Next
	}
	r.Cur = next
	return next
}
