// Package campaign — 劇本節點圖系統(doc 19):把「固定線性流程」變成可分支的有向圖。
// 節點 = 一個遊戲段落(story/battle/choice/event/ending),轉場依結果(win/lose/next/optN)
// 與旗標決定下一節點;敗北可走敗北路線而非 game over。
package campaign

import (
	"encoding/json"
	"fmt"
	"os"
)

// Line 一句對話(story 節點內嵌;speaker = 頭像 id)。
type Line struct {
	Speaker int    `json:"speaker"`
	Text    string `json:"text"`
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
	Fig    int `json:"fig"`
	ToX    int `json:"to_x"`
	ToY    int `json:"to_y"`
	Frames int `json:"frames"`
}

// Beat 過場原語(doc 50 §1/§2):cutscene 節點的 beats 是一條平面序列,依序執行,
// 一比一對映原版 EXE handler 的呼叫序列(LOADCH/PAN/TXT/ACT/SPAWN/JOIN/BGM/FADE/DELAY)。
// 每個 op 只用到自己相關的欄位,其餘留零值即可(同 Node 的稀疏欄位風格)。
type Beat struct {
	Op string `json:"op"` // pan/walk/dialog/act/spawn/join/bgm/fade/delay

	// pan/walk 共用:目標格(walk 用 X/Y 當終點);pan 的 X/Y 沿用 Node.CamX/CamY 語意——
	// 已由畫面回饋校準的「像素座標」,不是 doc47 §3 原始 grid(col,row)值(grid→px 未逐點驗證,
	// 不自行換算,見 rulebook 62)。
	X      int  `json:"x,omitempty"`
	Y      int  `json:"y,omitempty"`
	FromX  int  `json:"from_x,omitempty"` // walk 起點;省略=沿用該角色目前座標(接續上一拍)
	FromY  int  `json:"from_y,omitempty"`
	Fig    int  `json:"fig,omitempty"`    // walk/act:對應 Node.Actors 裡的角色(依 Fig 尋找,同 ActorWalk)
	Frames int  `json:"frames,omitempty"` // pan/walk/fade 位移或漸變幀數;delay 用幀數(見 Ms)
	Follow bool `json:"follow,omitempty"` // walk:走位期間鏡頭鎖定跟隨(doc47 §9,同 Node.FollowWalk 機制)

	// act:單位原地播 pose 序列(姿態循環,無位移)。remake 尚未把 74 筆 acting 資源解碼幀接上
	// 引擎播放(doc47 §5 未解),此處以「方向切換」近似演出節奏(見 main.go stepActJob 註解),
	// 非真實原版動畫——給悠妮昏迷轉向/蓋亞阻擋等簡單姿態用,不得當作已還原的 acting 播放器。
	Poses      []int `json:"poses,omitempty"`
	PoseFrames int   `json:"pose_frames,omitempty"` // 每個 pose 停留幀數(預設見 main.go)

	// dialog:章文本第 Line 條起連續 Count 句(Count 省略=1)。Line 對應目前節點 Script+Scene
	// 載入的那份 lines(同 Node.Scene 語意),不是 FDTXT 原始 idx(譯文精校版常把一條原文拆成
	// 多句對白,見 doc47 §7 教訓:機制懂了但內容沒逐句對齊前不假裝一一對應)。
	Line  int `json:"line,omitempty"`
	Count int `json:"count,omitempty"`

	Group int    `json:"group,omitempty"` // spawn:群組編號(doc25 spawn(g));remake 無群組資料表,僅記錄,見 main.go stub 註解
	Track string `json:"track,omitempty"` // bgm:曲目 id(對映 assets/bgm)

	Out bool `json:"out,omitempty"` // fade:true=淡出 false=淡入(重用 storyFade,doc46 §5.2)

	Ms int `json:"ms,omitempty"` // delay:毫秒(原版 0x375b2 語意);換算成 60fps 幀數,Frames 優先
}

// Node 節點。Type: story / cutscene / battle / choice / event / shop / ending。
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
	ExitWalks []ActorWalk `json:"exit_walks,omitempty"` // 同上,多角色一起退場(使用者回饋 2026-07-04 #A:
	// 影片證實草地小徑幕結尾索爾+亞雷斯兩人一起走離,非單人;ExitWalk/ExitWalks 可並用,全部走完才轉場;
	// 同時多角色並行走位不在 Beat.walk 的單角色設計內,cutscene 節點結束時仍沿用本欄位,不重造輪子)
	Beats       []Beat `json:"beats,omitempty"`        // cutscene:過場原語序列(doc 50);Beats 跑完後走 ExitWalk(s)+淡出+Advance,同 story 節點收尾
	AutoAdvance int    `json:"auto_advance,omitempty"` // story:無對白/Script 時,進節點後幾幀自動轉場(doc46 行軍蒙太奇)
	WalkFirst   bool   `json:"walk_first,omitempty"`   // story:進場走位全走完才顯示對白(2-1:王座廳索爾沿紅毯走到王座前對話框才出現)
	FollowWalk  bool   `json:"follow_walk,omitempty"`  // story:走位期間鏡頭鎖定跟隨走位者(原版 13×8 格視野長廊運鏡,doc25 0x11eee)
	CamMaxY     int    `json:"cam_max_y,omitempty"`    // story:鏡頭 Y 上限(px;0=不限)。王座廳=808 擋住 map32 底部草地段
	// (原版第一幕畫面無草地,索爾從畫面外沿紅毯走入,使用者回饋 2026-07-04 #1)
	BGM      string          `json:"bgm,omitempty"`
	Next     string          `json:"next,omitempty"`    // story/event
	OnWin    string          `json:"on_win,omitempty"`  // battle
	OnLose   string          `json:"on_lose,omitempty"` // battle(敗北路線;空=game over)
	Prompt   string          `json:"prompt,omitempty"`  // choice
	Options  []Option        `json:"options,omitempty"` // choice
	SetFlags map[string]bool `json:"set_flags,omitempty"`
	Text     string          `json:"text,omitempty"`      // ending:結語
	Goods    []Good          `json:"goods,omitempty"`     // shop:商品
	Secret   []Good          `json:"secret,omitempty"`    // shop:祕密商店商品
	SecretIf string          `json:"secret_if,omitempty"` // shop:旗標為真才開祕密商品(原版祕密商店機制)
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
		for _, to := range []string{n.Next, n.OnWin, n.OnLose} {
			if err := check(id, to); err != nil {
				return nil, err
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
	case "choice":
		var i int
		if _, err := fmt.Sscanf(outcome, "opt%d", &i); err == nil {
			if vis := r.Visible(); i >= 0 && i < len(vis) {
				next = vis[i].To
			}
		}
	case "ending":
		next = ""
	default: // story / event / shop
		next = n.Next
	}
	r.Cur = next
	return next
}
