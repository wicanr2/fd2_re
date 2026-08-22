# 37 — 法術 id → FIGANI 特效動畫 id 對映(反組譯結論)

> 任務:確認施法演出時「法術 id → FIGANI 特效動畫 id」的對映公式或表。
> 結論(附位址,已視覺驗證):目前沒有證據顯示 spell id 會挑選另一段 FIGANI「特效」動畫；施法演出會播放施法者自身的職業/角色動畫(doc 06 公式 `unit[+7]×3`/`×3+1`)。
> 但 spell/command 仍會參與 target geometry、family selector、傷害/狀態與訊息等路徑（見 doc13 `0x1cff0→0x149f8`）；因此撤回「spell id 只影響傷害與文字」的過強斷言。本篇只證明沒有獨立 spell-id→FIGANI 索引，不宣稱完整 spell runtime 已閉合。

## 0. 追蹤路徑總覽

```
選單施法 cmd byte(0..0xF=武器攻擊, 0x10+=法術)  [byte[esi+0x10], esi=0x4e56c(單位)取得的 FDICON 描述子]
  └─ cmd>=0x10 → spell_id = cmd-0x10           [0x0150d3]
       ├─ call 0x149f8(...)   ← target-candidate geometry/selector builder；不含任何 0x111ba / FIGANI 呼叫
       └─ 續行 0x01511c → 0x015168 → **call 0x28784(ebp)**   ← 施法演出,唯一參數 = 施法者 unit idx(ebp)
```

- **`0x28784`(單圖施法演出,doc 35 §0)全函式反組譯確認**:唯一輸入是施法者 unit index;
  內部只做 `unit[+7]` → FIGANI `組×3`(待機)、`組×3+1`(出招)兩次 `0x111ba(0x52388, …)` 載入(0x028848–0x02886d),
  **未讀取、未接收、未使用 spell_id**。函式尾端另有一段 `0x111ba("FDSHAP.DAT"@0x51a65, …)`(0x28a09-0x28a45)——
  ⚠ 這不是特效資源,FDSHAP 是地形圖塊庫(doc01 §8),此處是演出結束後場景圖塊的收尾載入,與法術特效無關。
- **`0x28784` 唯一呼叫端 = `0x015195`**(`push ebp; call 0x28784`,doc 35 已載明),`ebp` 全程 = 施法者 idx,
  spell_id(暫存於 `[esp]`/`al`)在 `call 0x149f8` 後即成死值,從未被讀出傳入 0x28784 或任何 `0x111ba` 呼叫。
- **`0x149f8` 另兩個呼叫端**(`0x150f1`、`0x157b5`、`0x1d35c`,`callgraph_le.py callers 0x149f8` 確認)同構:
  都只是「算傷害用的 spell_id」入口(AI 決策評分 0x157b5 在 0x1548e 附近、快速目標重算 0x1d35c),
  **三處都不含 FIGANI 載入**,佐證 spell_id 在全程式碼中從未流向 FIGANI 索引運算。

## 1. 視覺驗證:施法動畫是「通用施法手勢」,不是法術專屬特效

角色 **悠妮**(idx9,法師,`sprite_group=9`,`docs/data/exe_tables/characters.json`)→ FIGANI index = 9×3=27(待機)/28(出招),
`extracted/animations/FIGANI_027/`、`FIGANI_028/`(已解出,`tools/decode_figani.py` 產出):

- `FIGANI_027_f00.png`:待機站姿(持杖)。
- `FIGANI_028_f05.png`/`FIGANI_028_f10.png`:出招動作 —— 舉杖 + 手部小火花軌跡,**火花是畫在悠妮自己的 sprite 幀裡**
  (角色美術自帶的施法特效,燒進素材,同 doc35 §2.1「大小/朝向燒進素材」同一手法),不是另一張獨立疊圖。

因為程式碼證明 spell_id 從不流入 FIGANI 選擇(§0),**這組 27/28 動畫是悠妮所有法術共用的施法手勢**——
無論她施展哪一個 id 的法術,施法者畫面都播放同一段「舉杖 + 火花」動作。這只排除獨立
spell-id→FIGANI 索引；spell/command 仍會分流 target geometry、family selector、數值／status 與訊息，
不能再縮寫成「差異只在傷害數字與文字」。
（若要驗證另一名施法者也一樣，可用同一公式 `sprite_group×3(+1)` 解任一法師/僧侶/召喚師角色，結論同構，因為
`0x28784` 的程式碼路徑對所有角色一致，不因職業分支。）

## 2. spell_id 實際流向(釐清「那 id 是拿去做什麼」)

- **target/geometry**:`0x149f8` 沿格步進、篩選 unit index；其 selector 對 spell family 的完整對照仍待 RE。傷害/命中資料流不可由此函式單獨宣稱。
- **overlay／target selector（2026-07-28 撤回舊斷言）**：
  `[0x51a83] = command_record.range + 2`（`0x015140`／`0x0153b1`／
  `0x01bd14`／`0x01d188`）。它不是訊息索引；`0x122dc` 依值1..5
  直接畫固定 overlay descriptors，值6改 FDFIELD byte，而
  `0x115b6` 對大於1的值以 `selector-1` 送入 target legality。
  `0x12d7b→0x12cea` 是把游標移到 unit/cell 的 camera-pan helper，
  並沒有印「使用 XX 之術」。舊「戰鬥訊息／純文字系統」說法已刪除。
- 兩者都不產生任何額外的 FIGANI 載入。

## 3. `0x2a6bd` 的已知邊界（避免後續誤用）

反組譯初期曾把 `0x2a6bd` 說成只屬於「武器特殊攻擊結果 UI」、與 native command 無關；此斷言已撤回。
官方 IDA 9.4 已證實玩家 command confirm 的 `0x1cff0` 對 IDs `0..8`、`0x18`、`>=0x1c` 直接呼叫它。
因此它是 native command 的大型 presentation／state dispatcher，不能再以物品 `atk_attr` 枚舉取代 command ID。

目前可保留的狹窄結論如下：

- **player dispatch**：`0x1cff0` 的 `0x18`（ID24）會在 `0x2a6bd` 選特別分支 `0x276ec`，而不是跳表
  `funcs_1541f[24]`；該 route 將 derived stat 運算送進 `0x1c81f` HP writer。這已排除「jump table alias 即玩家效果」的捷徑。
- **presentation boundary**：`0x2a6bd` 的 `funcs_2ac25` entry 確實負責 indexed compositor／結果數字等畫面工作；
  某些 entry 載入 DATO.DAT 並不代表整個 caller 是武器專用，更不構成「沒有 command-specific visual」的證明。
- **command 24 已閉合的窄鏈（2026-08-22）**：`0x276EC` 以 actor raw `+7` 載入
  `3*selector+2`；正常 default 轉職後 selector32 對應 resource98。其 header
  byte4=6 經 `0x2BC9A` 選 FDOTHER #53，actor phase sample3／target phase
  sample2。command24 的 damage denominator 是1，不是等份多段扣血；第一個
  target `raw+4==1` 即發布完整傷害。完整位址與 raw frame 表見
  [`fd2_command24_presentation_ida.txt`](../data/ida/fd2_command24_presentation_ida.txt)。
- **尚未證實**：其他 command ID 的完整 FIGANI／SFX mapping，以及 command24
  的完整逐像素背景轉場、精確 PCM 取樣率與一般玩家 E2。

## 結論

**尚未發現「法術 id → FIGANI 特效動畫」的對映公式或表**：`0x28784` 的已證實單圖施法手勢只取
施法者 `sprite_group×3`/`×3+1`，不讀 spell id。這只排除「另一段 FIGANI 由 spell id 選擇」；角色幀內
的火花確實是該手勢的一部分。它**不**證明整個 spell presentation 沒有 command-specific 視覺：
`0x2a6bd` 的多數 dispatcher／DATO、SFX與命中畫面分支尚未閉合；command24 的
`0x276EC` 是目前已閉合的窄例外。remake 可以把角色手勢當作已證實的
局部 adapter，但不能稱為完整原版施法演出或宣告不需再補特效。

## 待確認

- `0x2a6bd` 其餘 command-specific presentation branch（尤其 `id>=0x20` →
  `0x27fc9`）的完整 renderer／SFX contract；command24 不得再列成「多段命中
  未知」，但其 `0x29C90` 完整畫面與精確音訊／E2仍待。
- 施法 figure displacement 是否對不同法術 `target`(0=單體/1=範圍,spell.json)有差異走位——
  `+0x48/+0x4a` 與 `0x29f72` 不是此問題的座標來源（已重判為 derived AP/DP 與 combat result resolver）；
  仍屬「目標選取與範圍」子題，非本題「特效動畫 id」範圍，未查。
