# 炎龍騎士團 2 反向工程與重製

本專案以合法原版《炎龍騎士團 2：黃金城傳說》作為行為基準，保存 DOS
程式的資料格式、介面與遊戲機制，並以 Go／Ebiten 建立可編輯、可擴充的
潔淨室重製引擎。《炎龍騎士團 2》是 1990 年代華文單機戰略角色扮演遊戲
（SRPG）的代表作之一；本專案也希望為 1995 年的台灣遊戲留下可重現的技術紀錄。

目前已有多個可操作、可比較的垂直切片，但**尚未達成 30 章原版等價通關**。
原版程式與美術、文字、音樂等受著作權保護的資產不包含在本倉庫中；使用者
必須自備合法原版。

## 目前進度

| 領域 | 已驗證成果 | 主要缺口 |
|---|---|---|
| 資產與格式 | `.DAT`、RLE 圖像、FDTXT／字型、AFM／FIGANI、XMIDI、地圖與部分 EXE 資料表可重現解析 | 部分執行期改寫、合成器與音訊播放尚未完整接入 |
| 反向工程 | 戰役狀態機、事件處理器、戰鬥規則、敵方 AI 原始模式與城鎮／商店／教會及存檔邊界已有證據化切片；mode 1 blocked-coordinate、mode 2 物理候選、mode 5 事件尾端，以及 mode 11 的 `0x15311` 命令／`0x1548E` 物理／`0x14121→0x13FD4` indexed 音訊 owner，均已接上失敗即關閉的執行期窄切片；`aiStep` 另已驗證兩名 raw mode-7 actor 的同回合順序，原始路由未處理且無錯誤時敵方／友軍 NPC 可依可編輯 `SpellBook`／`Spells` 完成正規化（normalized）法術決策與結算；未修改原版 CONTINUE→END→ENEMY PHASE 的章節0 E2 時間線也已可重現 | 尚缺重製端同一 raw 狀態的敵方回合 E2 配對、未知命令／法術／物品的完整演出、逐章戰後流程與 CONTINUE 四格的動作 owner／確認效果；原版 E2 錨點及現有 owner 都不等於逐像素或逐音訊一致 |
| Go／Ebiten 重製 | 地圖、對話、部分戰鬥、城鎮、商店、教會、整備、自有存檔及場景 BGM 消費可操作；戰鬥曲與城鎮曲已有原版表格回歸；近似模式可從最終節點播放已證實的 `0x2BCE5` 前綴、原資源角色最終狀態蒙太奇，以及來源 `FDOTHER#59` 的終局靜態圖 | 尚缺完整 30 章玩家路徑、`0x2C194` 的 20 段 indexed 尾段 renderer／精確時序與 BIOS 按鍵對映、一般玩家終局 E2 與跨平台驗收 |
| 原版視覺比對 | ch02 城鎮 variant0 六項、variant1 正常五項、variant2 正常五項（後兩者為修改 LOAD 路徑），以及部分商店、讀檔選單已有整幀 RGB 相同證據 | 完整操作介面估計約 40–45%；秘密選項、一般玩家城鎮路徑、戰場、整備、教會與其餘章節仍需同狀態比較 |

工作清單中的完成項代表已驗證的函式、格式或切片，**不是遊戲完成百分比**。
資產解碼完成也不等於玩法、介面或戰役流程已完成。

最新可驗證戰役切片：玩家第16戰戰後（raw `ch15_post`）已進入
`town_ch17`；玩家第17戰戰後（raw `ch16_post`）已進入 `town_ch18`；玩家第18戰
戰後（raw `ch17_post`）已進入 `town_ch19`；玩家第20戰戰後（raw `ch19_post`）已進入
`town_ch21`；玩家第22戰戰後（raw `ch21_post`）已由73／79-slot E1 邊界進入
`preparation_ch23`，並以正式戰鬥結果確認、持久隊伍及隔離存檔／讀檔驗證整備邊界；玩家第23戰戰前
（raw `ch22_pre`）已由 LOADCH 視圖來源 E1 切片進入 `battle_ch23`。玩家第23戰
戰後、24、25、29戰仍維持失敗即關閉；
詳細位址、分支與證據等級見[工作清單](docs/knowledge-base/91-worklist.md)、
[介面證據矩陣](docs/knowledge-base/57-ui-evidence-matrix.md)與
[最新交接勘誤](docs/knowledge-base/SESSION-HANDOFF-2026-07-06.md)。

為了讓重製端先維持可玩戰役流程，未綁定的戰後節點可在明確設定
`FD2_APPROXIMATE=1` 時顯示戰後整理提示，確認後沿可編輯腳本進入既有城鎮／整備；
這不猜原版 JOIN、獎勵或分支，未設定時仍採忠實模式的失敗即關閉。最近一輪
四個晚期未綁定戰後節點也已由正式勝利結果確認後才進入近似戰間段落；重製版／未修改 DOSBox 的實測命令、雜湊與限制見
[遊戲測試報告](docs/reports/game-test-2026-08-11.md)。

本輪另固定兩項可編輯邊界：各城鎮的祕密商店不是共用一組按鍵，而是由
`native_secret_gate` 保存每章 selection／掃描碼；戰鬥 BGM 依原版章節表、城鎮／
商店使用 `FDMUS_010`。預設忠實模式的 `ending` 節點沒有已證實終局曲目時仍會停止
前一曲。只有明確設定 `FD2_APPROXIMATE=1`，且戰役節點通過嚴格
`native_ending_prefix` 合約，才會播放已證實的 `0x2BCE5` 前綴；到 `0x2C548`
閘門時只消費已核對的 `FDMUS_004`，再以 persistent raw roster 播放原資源角色蒙太奇。
cycle 完成後，近似模式會驗證 `#57` 調色盤、`#58` 的 20 影格表與 `#60/#59` 單影格，
顯示來源 `#59` 的終局靜態圖並停留；外部片尾錄影把它對應為 `THE END` 僅屬
[強推論旁證](docs/data/ui-traces/ch30-ending-youtube-visual-side-evidence.json)，不是一般玩家 E2。按 Enter／空白鍵可選擇重播每位隊員的最終狀態，Enter／空白鍵／Esc
會回到終局定格；這是重製版延伸，不宣稱為原版按鍵行為。素材或 provenance 不足才會
確認回到可編輯結語。`FDMUS_018` 在此近似尾段只作曲目接線，時序不宣稱相同；
`0x2C194` 的 20 段 renderer、精確 BIOS 按鍵與一般玩家通關仍失敗即關閉。

![重製端近似模式的結局前段：到第一個原版文字閘門](docs/figures/ending-prefix-approximate-remake-e1.png)

*這是 Docker／Xvfb 的重製端 E1 直接節點擷取；不是第 30 戰一般玩家路徑、音訊播放
或原版逐像素比較。輸入、雜湊與限制見[擷取紀錄](docs/data/ui-traces/ending-prefix-approximate-remake-e1.json)。*

所有文件中的 `FD2.EXE` 位址目前只適用於大小 `357074` 位元組、MD5
`b97caf2239a27a896069d03549d96e1e` 的版本。SHA-256 與相關檔案雜湊見
[`fd2-reference-files.json`](docs/data/fd2-reference-files.json)；版本不同時
必須重新定位，不能直接套用既有位址。

## 專案核心

這不只是素材瀏覽器，也不是把原版流程重新寫死一次。專案同時維護兩條互相
驗證的主線：

- **技術保存**：以原始位元組、IDA Pro 指令、呼叫關係與 DOSBox 實驗記錄
  1995 年原版的資產格式、工具鏈、狀態機、介面、數值與遊戲規則。背景整理
  見 [`15-how-fd2-was-made-1995.md`](docs/knowledge-base/15-how-fd2-was-made-1995.md)。
- **潔淨室重製**：把寫死於 `FD2.EXE` 的對話、事件、戰後流程、城鎮、商店、
  整備與隊伍狀態轉成具型別、可編輯的資料與規則。
- **忠實模式與擴充模式共存**：原版戰役以證據還原；相同引擎日後可載入新
  劇本、分支與戰役，不必修改原版執行檔。
- **證據先於猜測**：未知處理器、存檔欄位或介面語意維持失敗即關閉，不用
  看似合理的替代行為冒充原版。

目前唯一持續整合的實作是 [`remake/`](remake/) 內的 Go／Ebiten 引擎。
架構、資料邊界與完成條件以
[`56-fd2-remake-sdd.md`](docs/knowledge-base/56-fd2-remake-sdd.md) 為準。

### 為台灣留一份技術紀念

> **AFM — Animation File Manager Version 1.00　Copyright (C) 1993 Lo Yuan Tsung**

這行留在原版動畫資料裡的署名，記錄了漢堂程式設計師自製工具的一角。本專案
不只要讓遊戲重新運作，也要把破解出的每一項技術整理成保存品質的文件，記錄
1995 年台灣團隊如何在 DOS 的限制下完成一款大型戰棋角色扮演遊戲：
[`04` 原版工具鏈](docs/knowledge-base/04-original-toolchain.md)、
[`05` 圖像壓縮](docs/knowledge-base/05-image-compression-format.md)、
[`06` 動畫格式](docs/knowledge-base/06-animation-format.md)、
[`07` XMIDI 音樂](docs/knowledge-base/07-music-xmidi-format.md)。

## 1995 年怎麼做出這款遊戲

[`15-how-fd2-was-made-1995.md`](docs/knowledge-base/15-how-fd2-was-made-1995.md)
把原版工具鏈、資料架構、畫面、動畫、音樂、中文、戰鬥規則與敵方人工智慧
（AI）串成一份完整紀錄：一支台灣團隊如何在 DOS、記憶體與音效卡各有限制的
年代，做出《炎龍騎士團 2》。

### DOS 上的中文是怎麼做出來的

DOS 原生不能直接顯示中文。漢堂隨遊戲攜帶點陣字型，文字則以內部字模索引
儲存：`FDTXT` 不是 Big5 文字檔，而是 `uint16` 字模索引序列，共 1016 條字串、
約 5.8 萬字；`FDOTHER` 資源 #4 收錄 1824 個 16×16、1 位元平面（1bpp）字模，
索引 0–35 是數字與英文字母，其後才是漢字。格式、控制碼與解碼證據見
[`08` 文字與字型](docs/knowledge-base/08-text-and-font-format.md)及
[`14` 文字控制碼](docs/knowledge-base/14-text-control-codes.md)。

| 文字／字型解碼研究圖（非重製執行期截圖） | 原版自製字型的解碼字模表 |
|---|---|
| ![文字與字型解碼研究圖；非重製執行期截圖](docs/figures/dialogue.png) | ![原版自製字型的解碼字模表](docs/figures/font_atlas.png) |

### 開場動畫：一個 1993 年的繪圖位元組碼虛擬機

原版開場過場不是逐幀點陣圖。AFM 是一台只有 10 個運算碼的增量繪圖虛擬機：
每幀只保存一小段位元組碼，在前一幀殘留畫面上更新調色盤或 VGA 像素，不清空
整個畫面重畫。96 幀的金鎖動畫因此只需約 1 MB，而不是 96 個 64000 位元組的
完整畫格陣列。

```text
每幀：[compSize u16][cmdCount u16][保留欄位 × 2]，後接 cmdCount 條指令
運算碼 0–3：調色盤操作
運算碼 4–9：VGA 畫面寫入
```

原版派發器位於 `0x36c9e`、跳躍表位於 `0x5276a`；播放器介面為
`0x020421(index, delayMs, skippable)`。`0x3dc9f` 以忙等測速校準動畫延遲；目前
尚未找到它的直接呼叫端，因此不把執行時機提升為已證實。這仍反映當年必須
適應不同電腦速度的做法。重製端的純 Go 解碼器位於
[`remake/internal/afm/`](remake/internal/afm/)，但不夾帶任何受著作權保護的原版
畫格；格式證據見 [`39-ani-afm-format.md`](docs/knowledge-base/39-ani-afm-format.md)。

### 音樂：兩種音源、兩種玩家記憶

同一首 XMIDI 曲目在 Roland MT-32 與 Sound Blaster／AdLib 上會呈現不同音色。
場景曲號不是靠聽曲風猜測，而是從 `play_bgm`（`0x25977`）的 32 處呼叫逐一追查；
這項方法也推翻過兩個早期的曲目推定。Sound Blaster／AdLib 路線使用遊戲自帶的
`SAMPLE.AD` 音色庫，也是原版預設音效卡與許多玩家記憶中的聲音；MT-32 則走
另一套合成路線。兩者都由玩家自備合法原版檔案與所需 ROM 在本機渲染，本倉庫
不散布音色庫或 ROM。詳見
[`12` 音樂播放與場景](docs/knowledge-base/12-music-playback-and-scene.md)及
[`16` 音訊合成](docs/knowledge-base/16-audio-synthesis-soundfont-mt32.md)；
`SAMPLE.AD` 的檔案角色與限制另見
[`36` 音效資料](docs/knowledge-base/36-sfx-audio-data.md)。

## 已建立的主要貢獻

| 貢獻 | 可重現成果 | 深入閱讀 |
|---|---|---|
| 原版資產保存工具鏈 | 統一解包 `.DAT`，解析 RLE 圖像、FDTXT、16×16 中文字型、地圖、XMIDI、AFM 與 FIGANI；產物可由玩家自備原版重生 | [`01` 容器與資產](docs/knowledge-base/01-container-and-asset-formats.md)、[`07` XMIDI](docs/knowledge-base/07-music-xmidi-format.md) |
| 繁體中文文字系統還原 | 將原版字模索引、控制碼、對話框與動態文字整理為資料層中可讀、可編輯的內容，不把它誤認為 Big5 文字檔 | [`08` 文字與字型](docs/knowledge-base/08-text-and-font-format.md)、[`14` 控制碼](docs/knowledge-base/14-text-control-codes.md) |
| 動畫與戰鬥演出解碼 | 建立 AFM 增量繪圖虛擬機及 FIGANI 幀解碼器，保存原始座標、調色盤與時序資料 | [`39` AFM](docs/knowledge-base/39-ani-afm-format.md)、[`35` 戰鬥演出](docs/knowledge-base/35-battle-animation-rendering.md) |
| 遊戲機制反向工程 | 以版本雜湊綁定已閉合的戰鬥規則切片、物品、事件處理器、敵方 AI、章節狀態與戰後路徑；撤回缺少寫入端／消費端的舊斷言 | [`11` 敵方 AI](docs/knowledge-base/11-enemy-ai.md)、[`27` 戰鬥規則](docs/knowledge-base/27-combat-rules-and-validation-checklist.md) |
| 原版介面逐狀態重建 | 城鎮、商店、讀檔及部分戰鬥介面以原版 indexed 資源重建；多個 ch02 狀態已有整幀 RGB 相同對照 | [`57` 介面證據矩陣](docs/knowledge-base/57-ui-evidence-matrix.md) |
| 可編輯戰役與持續隊伍 | 對話、事件、章節節點、商店／教會／整備與持續隊伍逐步脫離硬編碼；序章兩次增援及第一章 turn4／5 增援已接原版12次索引呈現與獨立 ACTING，第8戰已依29..41 slots frontier完成洛娜加入與進城，第10戰亦接上60／61 slots、原版 DAC 淡出／淡入、直接 record patch、JOIN11／6與 `town_ch11`。第16戰 raw `ch15_post` 現以 persistent-first 76 slots、四條 raw 分支、JOIN18 與 `town_ch17` save/load 完成重製端 E1；第20戰現依固定名冊第0筆＋15人整備建立83 slots，正確區分round15的增援／JOIN28與round16略過路徑，兩路都保留JOIN25並進`town_ch21`；第27章 event62→event63 已接敵軍 AI 前兩批增援與全白／恢復演出（均為重製端 E1，尚非 DOSBox E2）。第一章測試現已由正式 `State.Result`→`checkResult`→`confirmBattleResult` 進入 `town_ch02`／整備；未綁定 postbattle 節點另有明確近似模式，流程仍保持戰鬥→戰後→城鎮→整備；原生 `FD2.SAV` 四槽 LOAD 已能還原到具型別隊伍及城鎮／整備邊界；chapter0 current-runtime CONTINUE 已接明確存檔／計時種子的重製端 E1 發布，尚非一般玩家 E2 | [`29` 事件系統](docs/knowledge-base/29-remake-extensible-event-system.md)、[`23` 啟動與存檔流程](docs/knowledge-base/23-boot-title-and-scenario-flow.md) |

這些貢獻各自代表已驗證的子系統，不代表完整 30 章已經可以從頭到尾等價
遊玩。尚未閉合的玩家路徑、原生存檔還原與介面差距仍列在
[`42-re-vs-remake-gap-audit.md`](docs/knowledge-base/42-re-vs-remake-gap-audit.md)
與 [`91-worklist.md`](docs/knowledge-base/91-worklist.md)。

## 可驗證畫面

以下圖片只代表其標示的狀態，不可外推為整套遊戲已完成。

### 城鎮與商店

ch02 城鎮六個選項：上排為原版 DOSBox，下排為重製；每格皆有整幀相同的
對照結果。

![ch02 城鎮六個選項原版與重製對照](docs/figures/town-hub-six-selections-original-vs-remake.png)

ch02 城鎮 variant2 的正常五項：上排原版、下排重製；原版來自明確標示的
修改 LOAD 研究副本，選項0–4各自與指定脈動整幀相同，並非一般玩家戰後路徑。

![ch02 城鎮 variant2 五項原版與重製對照](docs/figures/town-variant2-five-selections-original-vs-remake.png)

ch02 城鎮 variant1 的正常五項：上排原版、下排重製；同樣來自明確標示的
修改 LOAD 研究副本，選項0–4各自與實測脈動相位整幀相同，並非一般玩家戰後路徑。
完整雜湊與限制見 [`native_town_variant1_e2.json`](docs/data/native_town_variant1_e2.json)。

![ch02 城鎮 variant1 五項原版與重製對照](docs/figures/town-variant1-five-selections-original-vs-remake.png)

ch02 武器店、道具店與秘密商店：上排為原版，下排為重製；這只證明圖中三個
商店主選單狀態。

![ch02 三種商店原版與重製對照](docs/figures/shop-variants-1-3-5-original-vs-remake.png)

### 讀檔選單

原版與重製的四個空存檔槽畫面已達整幀 RGB 相同；有效槽排版也已有相同結果。
四槽 LOAD 的 checksum、名冊、金幣與城鎮／整備落點已接入正式路徑，但目前
只有合成有效槽的決定性測試；尚缺未修改一般玩家有效槽的成功載入實驗。

| 原版 | 重製 |
|---|---|
| ![原版空槽讀檔畫面](docs/figures/load-empty-original-dosbox.png) | ![重製空槽讀檔畫面](docs/figures/load-empty-remake.png) |

### 標題選單

原版與重製端的標題選單同樣以 320×200 內容呈現；重製圖是 Docker／Xvfb
實際執行期擷取，保留目前非原版的音源提示列，僅作介面對照，不代表
一般玩家 E2 或完整開場動畫已完成。chapter0 的 CONTINUE E1 另見下方戰場段落。

| 原版 DOSBox | 重製端執行期 |
|---|---|
| ![原版標題選單](docs/figures/title-original-dosbox.png) | ![重製端標題選單執行期畫面](docs/figures/title-remake-runtime.png) |

### 對話框

重製端已能從可編輯的序章腳本實際跑出對話框執行期（runtime）畫面；下圖是
Docker／Xvfb 產生的 640×400 E1 證據。它證明腳本、肖像、藍框與中文字型已通過
重製端消費端，但不是原版同狀態的一般玩家 E2 等價證明。將重製圖縮放至原版
320×200 後，與原版 DOSBox 對話畫面的平均絕對誤差（AE）為 60414，仍可看出
版面、背景與文字避讓尚未一致。

| 原版 DOSBox | 重製端執行期（E1） |
|---|---|
| ![ch01 原版 DOSBox 對話框](docs/figures/ch01-dialogue-original-dosbox.png) | ![重製端對話框執行期畫面](docs/figures/dialogue-remake-runtime.png) |

### 戰場與戰鬥演出

未修改原版已由 Docker 沙箱走到第一戰的一般玩家回合，固定取得第一個我方單位的
戰術指令格、肖像、HP／MP 與能力欄；這是原版玩家回合 E2 錨點，不是重製端對照圖，
也不代表敵方人工智慧回合已驗收。來源雜湊、按鍵時間線與限制見
[`native-player-turn-original.json`](docs/data/ui-traces/native-player-turn-original.json)。

![未修改原版第一戰玩家回合指令格](docs/figures/battle-player-turn-original-dosbox.png)

ch01 戰場目前已能由原始地圖、單位、前景與 HUD 資產合成。首頁主要展示同一份
`FD2.SAV`、同一鏡頭／游標／回合的 320×200 比較：左為原版 DOSBox，中為重製端
最近鄰縮小結果，右為差異遮罩。terrain icon 已依 IDA 證實的
`base + stride*5 + 6` 修正；內容區只剩左下畫布邊界的 22 個差異像素，不能外推為
其他章節或完整操作界面已完成。

![ch01 同狀態原版／重製戰場與差異遮罩](docs/figures/battle-field-ch01-scoped-compare-20260810.png)

### 可玩近似戰後提示

明確設定 `FD2_APPROXIMATE=1` 時，尚未有正式原版 handler 的戰後節點會先同步已物化
隊伍，顯示可確認的戰後整理提示，再沿腳本進入城鎮／整備；下圖是 Docker／Xvfb
實際執行期擷取。它是可玩近似模式畫面，不代表原版演出或逐像素等價。

![可玩近似模式戰後整理提示](docs/figures/postbattle-approximate-remake.png)

完整尺寸、狀態指紋、影像雜湊與限制見
[`battle-visual-gap-ch01.json`](docs/data/ui-traces/battle-visual-gap-ch01.json)。

另有一張由既有 `story_ch00_handler` 截圖快進器產生的重製端第1戰畫面，僅作
E1 執行期產物展示：[`native-battle-ch01-remake-e1.png`](docs/figures/native-battle-ch01-remake-e1.png)。
它不是一般玩家輸入或原版同狀態 E2 對照；完整限制與雜湊見
[`native-battle-ch01-remake-e1.json`](docs/data/ui-traces/native-battle-ch01-remake-e1.json)。
較早的 [`native-map-ch01-original-video.png`](docs/figures/native-map-ch01-original-video.png)
與正式 handler 截圖
[`native-map-ch01-remake-handler.png`](docs/figures/native-map-ch01-remake-handler.png)
不是同一狀態，仍保留作為歷史流程參考；不能拿它們判定單位、游標或 HUD 的像素差異。

戰鬥動畫解碼與局部位置比較：首頁不再把 2026-07 的舊分鏡
[`battle_restore.gif`](docs/figures/battle_restore.gif) 當成目前執行期畫面；該圖保留在
儲存庫作歷史證據。以下是 2026-08-10 以目前 source 重新擷取的命中幀：左為原版
影像正規化結果，中為重製端，右為逐 RGB 差異遮罩。它可直接看出剩餘差異，並不
宣稱整個戰鬥介面已完成像素級一致。

![戰鬥命中幀正規化對照與差異遮罩](docs/figures/battle-impact-compare-20260810.png)

戰鬥狀態欄姓名目前已改接原版 `FDOTHER#4` 16×16 點陣字模；下圖是 Docker／Xvfb
以目前 source 重新建置的 E1 演出截圖。它只證明姓名字模的消費端，不代表攻擊幀、
傷害／閃紅或整個戰鬥介面已與原版逐像素一致。

![重製端原版字模戰鬥狀態欄](docs/figures/battle-native-name-remake.png)

針對命中幀的另一項已確認偏差：重製端已移除沒有原始欄位來源支撐的全畫面紅罩，
避免背景、狀態欄與台座一起泛紅；目前只保留原版 impact 參考圖支持的守方紅剪影，
攻方維持原本 FIGANI 幀，仍標示為 E1 近似。下圖是修正後的
Docker／Xvfb 截圖，不是原版同狀態逐像素 E2；原始 DAC 條件與截圖雜湊見
[`battle-impact-no-global-tint.json`](docs/data/ui-traces/battle-impact-no-global-tint.json)。

![重製端命中幀（移除未證實全畫面紅罩；E1）](docs/figures/battle-impact-no-global-tint.png)

目前 source 的完整原生戰場 frame 也能承接指令環與 command grid；下圖是 Docker／Xvfb
的 E1 執行期畫面，資源缺失時仍會失敗即關閉，不代表原版逐像素等價。

![完整原生戰場與指令環疊加（重製端 E1）](docs/figures/action-overlay-native-remake-fullframe.png)

另有一份未修改原版 `FD2.SAV` 的 chapter0 current-runtime E2 錨點：從標題
`CONTINUE` 進入戰場，再按 Enter 開啟 command grid。這是原版操作證據，不是
重製端 parity，也不能外推到第23戰；完整輸入時間線與雜湊見
[`native-continue-current-runtime-e2.json`](docs/data/ui-traces/native-continue-current-runtime-e2.json)。

![原版 CONTINUE current-runtime 戰場](docs/figures/native-continue-current-runtime-original-dosbox.png)

![原版 current-runtime command grid](docs/figures/native-continue-current-command-original-dosbox.png)

重製端現在也能由同一份 `FD2.SAV` 走標題 Escape／Down／Down／Enter，
在明確提供 `FD2_NATIVE_TITLE_TICK` 後發布 `battle_ch01` current-runtime；
下圖是 Docker／Xvfb 的 E1 執行期畫面。它證明保存的鏡頭、游標、回合與地圖
已交給戰場控制器。這張較早的基準圖仍不證明 Enter 後的四格動作 owner 或原版
status/equipment panel，因此不宣稱逐像素 E2。條件與雜湊見
[`native-continue-current-runtime-remake-e1.json`](docs/data/ui-traces/native-continue-current-runtime-remake-e1.json)。

![重製端 CONTINUE current-runtime 戰場（E1）](docs/figures/native-continue-current-runtime-remake-e1.png)

同一份存檔另保存了原版 E2 與重製端普通 X11 輸入的 current-runtime 配對基準：
原版 320×200 放大至 640×400 比較的 AE 為 `164`、RMSE 為 `50.2631`。重製端畫面
仍使用明確的 `FD2_NATIVE_TITLE_TICK=0` 計時夾具，所以這是原版 E2＋重製端 E1
的配對證據，不是逐像素一致；完整限制與雜湊見
[`native-continue-current-runtime-remake-e2.json`](docs/data/ui-traces/native-continue-current-runtime-remake-e2.json)。

![重製端 CONTINUE current-runtime 配對畫面（E1）](docs/figures/native-continue-current-runtime-remake-e2.png)

2026-08-11 已修正 action overlay 圖塊索引乘數與「只載入前十格」的錯誤；同一普通
X11 `CONTINUE→Return` 路徑現在會在空游標顯示**強推論**對應原版 `0x16f55` 的四個圖塊。下圖
依序是原版、重製與差異遮罩；它清楚顯示畫面仍有差距（AE `8932`），所以只作重製端
E1 的可見／輸入切片，Enter 的動作效果仍失敗即關閉。原始位址、輸入、旁車與雜湊見
[`native-continue-current-command-remake-e1.json`](docs/data/ui-traces/native-continue-current-command-remake-e1.json)。

![CONTINUE 空游標命令面板：原版／重製／差異（重製端 E1）](docs/figures/native-continue-current-command-compare-e1.png)

同一份未修改原版 `FD2.SAV` 也已走到「結束目前單位行動嗎？」→「是」，實際看見
`ENEMY PHASE`，並在約 10 秒後看到敵方回合的另一個戰場畫面，約 20 秒後回到玩家
操作狀態。這是一般玩家敵方回合的原版 E2 輸入／畫面錨點，不是重製端對照，也不足以
單獨證明目標選擇、移動評分或命令／法術／物品語意；完整時間線、來源雜湊與限制見
[`native-enemy-turn-original-e2.json`](docs/data/ui-traces/native-enemy-turn-original-e2.json)。

重製端另有受原始來源證據（raw provenance）保護的 mode 0／mode 1／mode 2／mode 3／mode 4／
mode 5／mode 7／mode 8／mode 9／mode 10／mode 11 遊戲層 E1 回歸：mode 0 可依 raw
nearest fallback、mode 1 可依 raw blocked-coordinate 完成 movement-only 路徑，mode 2 可完成物理移動／FIGANI 攻擊，mode 3／9
依 raw `+0x08` 查找完成 movement-only 路徑，mode 4／10 依 raw 目的地完成 movement-only
路徑，mode 5 可提交事件格的 raw state 尾端，mode 7 可完成 raw 目的地移動並提交 `+0x05`，
mode 8 走共同的行動完成分支，`0x14EF0` 已有 raw command route 進入已驗證的
數值執行器，`0x15055` 另可在已核對的 type-5 raw item row 上完成 HP 交易，mode 11 可依序消費兩段 command／
physical stage；缺少來源則停止而不標記單位已行動。這些是可驗證的 AI 消費端切片，不是原版 E2 等價性（parity）；完整目標選擇、
未知命令／法術／其他物品演出與未修改原版同狀態逐幀配對仍列於 [`11` 敵方 AI](docs/knowledge-base/11-enemy-ai.md)
與 [`91` 工作清單](docs/knowledge-base/91-worklist.md)。

原始路由未建立計畫且未回傳錯誤時，敵方／友軍 NPC 會以可編輯 `SpellBook`／`Spells` 選擇已知的治療、攻擊、輔助或狀態法術，
經 `aiStep → CastArea` 完成移動、數值結算與回合提交；這是重製端的正規化（normalized）近似，不替原版 `0x1598A` 評分、指令格、
特效或音效背書。原始路由一旦回報缺少 provenance 或其他錯誤，仍會在任何行動前失敗即關閉。

| 原版 ENEMY PHASE | 原版敵方回合畫面 | 原版回到玩家回合 |
|---|---|---|
| ![原版敵方回合開始](docs/figures/native-enemy-phase-original-dosbox.png) | ![原版敵方回合中段](docs/figures/native-enemy-ai-moved-original-dosbox.png) | ![原版敵方回合結束](docs/figures/native-enemy-phase-return-original-dosbox.png) |

完整介面覆蓋、證據等級與其餘比較圖請看
[`57-ui-evidence-matrix.md`](docs/knowledge-base/57-ui-evidence-matrix.md)。

## 快速開始

### 需求

- Docker
- 合法原版遊戲檔案
- 本倉庫不要求在主機安裝 Go、Capstone 或其他分析套件

本專案的分析、轉檔、建置、測試與抓圖一律在 Docker 內執行。可維護的容器
入口與資產匯出步驟見 [`remake/README.md`](remake/README.md) 及
[`tools/docker/`](tools/docker/)。

### 建置與測試

Go／Ebiten 是目前唯一持續整合的重製引擎：

```bash
docker run --rm --network none --memory 3g --cpus 2 --pids-limit 384 \
  --user "$(id -u):$(id -g)" -e HOME=/tmp -e GOCACHE=/tmp/go-cache \
  -e GOMAXPROCS=1 -e GOFLAGS=-p=1 \
  -v "$PWD:/src:ro" -w /src/remake fd2-go-test-local \
  sh -c 'set -eu; trap "pkill -TERM Xvfb 2>/dev/null || true" EXIT INT TERM; \
    timeout 900s xvfb-run -a -s "-screen 0 1280x1024x24" \
    go test ./... -count=1'
```

本機映像檔的建立方式與 Web、Linux、Windows 打包入口請看
[`remake/README.md`](remake/README.md)。資產工具位於 [`tools/`](tools/)；
輸出放在被忽略的 `extracted/`，不應把原版衍生資產提交至 Git。

## 文件導航

### 現況與工程入口

建議先依下列順序閱讀：

1. [`56-fd2-remake-sdd.md`](docs/knowledge-base/56-fd2-remake-sdd.md)：
   系統設計、證據分級、原版與重製的責任邊界。
2. [`57-ui-evidence-matrix.md`](docs/knowledge-base/57-ui-evidence-matrix.md)：
   操作介面的原版證據、重製狀態與未閉合項目。
3. [`42-re-vs-remake-gap-audit.md`](docs/knowledge-base/42-re-vs-remake-gap-audit.md)：
   從玩家功能檢視原版與重製差距。
4. [`91-worklist.md`](docs/knowledge-base/91-worklist.md)：
   目前工程佇列與驗證狀態。
5. [`00-index.md`](docs/knowledge-base/00-index.md)：
   資產格式、戰鬥、劇情、介面等專題文件索引。
6. [`SESSION-HANDOFF-2026-07-06.md`](docs/knowledge-base/SESSION-HANDOFF-2026-07-06.md)：
   時間序列證據與後續勘誤，不應單獨視為現況真值。

早期設計、歷史反思或專題筆記若與原始位元組、執行期實驗或上述現況文件
衝突，以較新的直接證據與勘誤為準。未知語意不會為了讓流程前進而猜測接入。

### 技術保存專題

知識庫不只是工程工作清單；其中 `04`–`11` 等文件也在保存「1995 年台灣怎麼
做遊戲」的技術紀錄。完整目錄見 [`00-index.md`](docs/knowledge-base/00-index.md)，
以下是依問題分類的入口：

- **原版工具與資產格式**：[`01` 容器與資產](docs/knowledge-base/01-container-and-asset-formats.md)、
  [`04` 原版工具鏈](docs/knowledge-base/04-original-toolchain.md)、
  [`05` 圖像壓縮](docs/knowledge-base/05-image-compression-format.md)、
  [`06` 動畫](docs/knowledge-base/06-animation-format.md)、
  [`07` XMIDI](docs/knowledge-base/07-music-xmidi-format.md)、
  [`08` 文字與字型](docs/knowledge-base/08-text-and-font-format.md)、
  [`39` AFM](docs/knowledge-base/39-ani-afm-format.md)。
- **遊戲邏輯與機制**：[`03` 執行檔與資料結構](docs/knowledge-base/03-exe-and-data-structures.md)、
  [`09` 劇情與對話](docs/knowledge-base/09-story-and-dialogue.md)、
  [`10` 精靈繪製與狀態](docs/knowledge-base/10-sprite-rendering-camp-and-state.md)、
  [`11` 敵方 AI](docs/knowledge-base/11-enemy-ai.md)、
  [`13` 戰鬥選單](docs/knowledge-base/13-battle-menu-system.md)、
  [`14` 文字控制碼](docs/knowledge-base/14-text-control-codes.md)、
  [`27` 戰鬥規則](docs/knowledge-base/27-combat-rules-and-validation-checklist.md)。
- **引擎控制流與介面**：[`23` 啟動、標題與存檔](docs/knowledge-base/23-boot-title-and-scenario-flow.md)、
  [`24` 呼叫圖分析](docs/knowledge-base/24-callgraph-analysis-log.md)、
  [`35` 戰鬥演出](docs/knowledge-base/35-battle-animation-rendering.md)、
  [`50` 過場腳本設計](docs/knowledge-base/50-cutscene-script-system-design.md)。
- **重製設計與驗證**：[`29` 可擴充事件系統](docs/knowledge-base/29-remake-extensible-event-system.md)、
  [`42` 差距稽核](docs/knowledge-base/42-re-vs-remake-gap-audit.md)、
  [`56` 系統設計](docs/knowledge-base/56-fd2-remake-sdd.md)、
  [`57` 介面證據矩陣](docs/knowledge-base/57-ui-evidence-matrix.md)。

## 倉庫結構

```text
docs/knowledge-base/  系統設計、證據、專題研究與工作清單
docs/data/            結構化資料、原版檔案雜湊與可重播追蹤
docs/figures/         經整理的原版／重製比較圖
remake/               Go／Ebiten 重製引擎
tools/                資產解碼、反組譯與驗證工具
org_game/             使用者自備原版；不納入版本控制
extracted/            本機產物；不納入版本控制
```

## 研究與實作原則

- 原版執行檔是行為判定基準；反編譯輸出與攻略只能協助導航。
- 已證實結論必須綁定位址、原始位元組、呼叫者／消費端或同狀態實機比較。
- 寫死在原版中的對話、事件、戰後、城鎮、商店與整備流程，會轉成可編輯資料
  與具型別規則，不保留依賴原版位址的正式執行捷徑。
- 多數戰鬥結束後會進入城鎮、商店或整備，不會直接跳到下一場戰鬥。
- 除錯捷徑、修改存檔或只通過重製端測試，不等於一般玩家路徑已驗證。

協作規則與 Docker、IDA Pro、提交及證據政策見 [`AGENTS.md`](AGENTS.md)。

## 版權與致謝

- 《炎龍騎士團 2》及其原版資產著作權屬漢堂國際。本專案只提供研究紀錄、
  工具與潔淨室重製程式，不散布原版內容。
- 攻略資料參考青衫整理的
  [《炎龍騎士團 2》圖文攻略](https://chiuinan.github.io/game/game/intro/ch/c31/fd2/)；
  攻略只作玩家可見行為與數值旁證，不作二進位介面證據。
