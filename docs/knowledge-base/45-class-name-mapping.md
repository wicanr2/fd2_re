# 45 — 敵/友單位「職業名」顯示錯位根因(worklist 第 8 輪修復)

> 起因:實測回報「原版海盜 → remake 顯示劍士、原版士兵 → remake 顯示聖騎士」。與另一線(ch1-verify,
> 用青衫攻略截圖 + 影片鐵證交叉核對)獨立得出同一根因,本檔記錄反組譯/資料面的完整證據鏈與修法,
> 供之後補其他章節怪物 portrait 對照時參照。方法遵守 `rulebook/62`(靜態溯源、描述前先驗證)。

## 一句話總覽

FDFIELD 出場人物 26B 結構裡的 `(race,cls)` 是**戰鬥數值範本索引**(決定 HP/AP/DP/DX/MV 成長曲線、
戰鬥動畫類型),**不是身分**。map0 裡海盜、哈瓦特、哈諾、T6 海防隊士兵四個完全不同的敘事角色,
`(race,cls)` 全部是 `(1,1)`——同一組數值範本被至少四種不同身分共用,證明它天生就不可能拿來反推顯示名。
真正的敘事身分綁定在 **portrait**(= DATO 角色 id)。`export_units.py` 舊版拿 `(race,cls)` 去查
`docs/data/exe_tables/unit.json` 的 `cls_name` 欄位(該欄本身是拿**玩家轉職職業表**
`CLASS_NAMES=["龍","劍士","戰士",...,"聖騎士",...]` 填的),等於整批敵方/友方雜兵的顯示名被誤用
「玩家轉職職業」代替,`cls=1→劍士`、`cls=11→聖騎士`,才會出現海盜/士兵都顯示「劍士」這種錯位。

## 證據鏈

### 1. `(race,cls)` 多對一,且對不到敘事身分

`references/text/modify2.md` §8「敵/友單位等級資訊」(0x558F9,10B/列,68 列)親自列出:
同一 `(race=01,cls=02)` 對應**六個不同 NPC**——士兵、傭兵、精英戰士、鎧甲武士、黑暗戰士、狂戰士,
純靠數值(HP/AP/DP/DX/MV/EX 每級成長)區分,`cls` 本身只決定「用哪套戰鬥機制模板」。

### 2. map0 實測:四個不同角色共用 `(race,cls)=(1,1)`

直接讀 `extracted/raw/FDFIELD` map0 控制段(資源 1)的出場人物原始 bytes(`camp,portrait,race,cls,lv`):

| camp | portrait | race,cls | lv | group | 實際身分(青衫攻略/影片) |
|---|---|---|---|---|---|
| enemy | 96 | 1,1 | 2 | 1/2/4/5 | 盜賊(俗稱「海盜」) |
| enemy | 97 | 1,1 | 3 | 5 | 盜賊頭目 |
| own | 3 | 1,1 | 3 | 7 | 哈瓦特(T3 加入) |
| own | 1 | 1,1 | 1 | 3 | 哈諾(T3 加入) |
| ally | 68 | 1,1 | 2 | 6 | 士兵(T6 海防隊) |

五種身分,`(race,cls)` 全部是 `(1,1)`。而 `docs/data/exe_tables/unit.json` 裡唯一配到
`(race=1,cls=1)` 的一列是 idx31「黑暗殺手」(offset `0x55a2f`,對照 modify2.md `7AC43 黑暗殺手
01 01 36 0 20 12 4 6 110` 數值逐位吻合)——跟海盜/士兵/哈瓦特/哈諾都毫無關係。舊版
`export_units.py` 的 `base_stats()` 用 `(race,cls)` 第一筆匹配到這列,`cls_name` 欄位又是拿
`CLASS_NAMES[cls=1]="劍士"` 填的(這張表驗證用途本來是給**玩家轉職**,26 筆對 modify2.md §7
職業魔抗/暴擊率表逐一吻合,表本身沒錯,是被拿來做了不對的事),兩個誤用疊加,才會顯示「劍士」。

### 3. 真身分綁在 portrait

`docs/knowledge-base/31-map-unit-sprites-fdicon.md` §8「已定論」:**id = 肖像(FA)= sprite組(Z1)=
角色,基本態恆等**;敵方/通用 `id>31` 已知 3 筆:`士兵68、盜賊96、頭目97`。

交叉驗證:`extracted/story/full_story_auto.md`(FDTXT_000 解碼字串,glyph_map 直接還原遊戲內嵌文字,
非攻略轉述)行 41–101 有一份 54 筆的 NPC/怪物身分名清單,順序與 modify2.md §8「敵人」表幾乎逐字對上
(士兵、精英戰士、鎧甲武士、傭兵、黑暗戰士、狂戰士、騎兵、突擊騎兵、地獄騎士、黑暗騎士、龍騎士、…、
盜賊、盜賊頭目、影之忍者、黑暗殺手、…),證實這批名字是遊戲真的內嵌的敘事身分字串,不是攻略作者自訂。
目前僅 68/96/97 三筆有 portrait↔名稱的確認對照(來自 doc31 的既有 RE 成果);把這 54 筆逐一配對到
portrait id 需要額外反組譯(找 spawn/繪製 code 讀 portrait 查這張表的呼叫點),留待後續章節碰到新怪物
再補,本輪不擴大範圍。

## 修法(已實作,`tools/export_units.py`)

```python
PORTRAIT_CLS_NAME = {
    68: "士兵",     # 友軍海防隊員(第一章 T6 增援)
    96: "盜賊",     # 第一章開場敵方(俗稱「海盜」)
    97: "盜賊頭目",  # 第一章 T5 增援首領
}
```

`cls_name` 改成先查 `PORTRAIT_CLS_NAME[portrait]`,查不到才退回原本 `CLASS_NAMES[cls]` 行為
(fallback 保留,避免其他尚未 RE portrait 對照的怪物顯示空字串)。**`(race,cls)` 驅動的
`base_stats()` 沒有動**——HP/AP/DP/DX/MV 走的是數值範本這條路徑,青衫攻略與 growth.go 的轉職成長
系統都靠這條,是對的機制,只是不該借去當顯示名。

## 已知限制(範圍外,留待後續)

- 只有 3 個 portrait 有覆寫;其他章節怪物(如 map0 group10/11 的 portrait 76/103)目前仍會退回
  `CLASS_NAMES[cls]` 顯示機械職業名(「戰士」「聖騎士」),但 ch01.json 的 `initial_groups`/`events`
  已不引用這兩組(見 worklist 第 8 輪另一項修正),第一章實際不會出現。
- 這批雜兵單位的 HP/AP/DP 數值本身可能也不準確(`base_stats()` 仍用 `(race,cls)` 配到不相干的
  「黑暗殺手」列,理論上盜賊該對到 modify2.md §8 的「盜賊」列 HP14/AP7 而非現在的 HP36/AP20)——
  `export_units.py` docstring 已記錄「同 `(race,cls)` 只取第一筆」是既有限制,本輪只修顯示名,
  數值準確度留給之後有需要再排。

## 驗證

`extracted/remake_shots/clsname_enemy_pirate_v3.png`(裁切版 `_crop.png`):campaign battle_ch01,
游標點開場海盜(1,3),狀態列顯示「盜賊」。`extracted/remake_shots/clsname_ally_soldier.png`
(裁切版 `_crop.png`):T6 海防隊(23,14)顯示「士兵」。`go build`/`go test ./...`(docker
`golang:1.22-bookworm`)全綠。
