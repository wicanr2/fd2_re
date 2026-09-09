# 102 — 戰鬥結果字串沒有原版出處（2026-09-09）

第 6 項的後半：「不應該在戰鬥後顯示傷害多少」。

## 兩項證據

**一、字串出自重製端自己的原始碼。** `remake/assets/locales/zh-Hant/pack.json`
裡五筆 `battle.attack.*` 的 `source_string_id` 全部指向重製端的 Go 檔：

| 條目 | source_string_id |
|---|---|
| `battle.attack.miss` | `legacy.go.remake.cmd.fd2.main.l5973-c22` |
| `battle.attack.hit` | `legacy.go.remake.cmd.fd2.main.l5975-c25` |
| `battle.attack.critical_suffix` | `legacy.go.remake.cmd.fd2.main.l5977-c14` |
| `battle.attack.exp_suffix` | `legacy.go.remake.cmd.fd2.main.l5980-c26` |
| `battle.attack.choose_target` | `legacy.go.remake.cmd.fd2.main.l5193-c13` |

**二、原版 FDTXT 沒有這種模板。** 掃過分離出的全部 FDTXT 資源，「傷害」
「造成」「點傷」只出現五次，全部在劇情對白句子裡（FDTXT_010／018／025／027／
030），沒有任何「造成 N 傷害」型的戰鬥結果模板。

**三、原版演出裡的傷害是 HP 數字與長條。** 推進兩個敵方回合直到雙方接戰，
拍到三段物理攻擊演出（收據
[fd2-physical-attack-presentation-20260909.json](../data/ui-traces/fd2-physical-attack-presentation-20260909.json)）。
版面是右上攻方、左下守方，各為「姓名　LV·NN」＋ HP 條與三位數字 ＋ MP 條與
三位數字；命中時 HP 數字與長條同步變化（索爾 042→031、盜賊 028→018）。
整段演出**沒有任何傷害量文字**。

## 處置

字串本身保留——它同時是語言包完整性的前置檢查，缺條目要在改動 HP 之前就
失敗即關閉。改的是顯示：`publishPhysicalAttackMessage` 只在 F3 診斷模式寫進
`g.msg`，一般玩家路徑不顯示。玩家、AI 一般路徑與 native AI mode 11 三個發布
點都改走它。

回歸 `TestPhysicalAttackMessageStaysOutOfThePlayerPath`。

## 前半的「戰鬥動畫退化」：目前重現不出來

第 6 項的前半另外查了三件事，三件都沒有找到退化：

**一、既有回歸 fixture 逐位元組重現。** `battle-impact-no-global-tint.json` 記
的擷取條件（`FD2_SHOT_ATTACK=4`、`FD2_SHOT_FRAME=3`、`FD2_SHOT_SERIES`、
`FD2_SHOT_DETERMINISTIC=1`、原版 `FDOTHER.DAT`）在目前 HEAD 重跑，序列中的
`frame_77` 與 2026-08-25 存檔的
[`battle-impact-no-global-tint.png`](../figures/battle-impact-no-global-tint.png)
**AE = 0**（640×400 共 256000 像素，逐 RGB 比對）。命中幀沒有退化。

**二、版面與原版一致。** 同一段序列的第一幀：右上 `亞雷斯　LV·01` ＋ HP 條與
`048` ＋ MP 條與 `000`，左下 `盜賊　LV·02` ＋ `028`／`000`，與原版擷取的版面
相同（原版是右上 `索爾　LV·01`／`042`、左下 `盜賊　LV·02`／`028`）。

**三、素材覆蓋沒有缺口。** 分離包有 409 筆 FIGANI 資源；scenario 內出現的 96 個
`battle_fig` 值，`fig*3`（待機）與 `fig*3+1`（攻擊）**全部都在**。完整包的
組裝腳本也明文要求並複製 `animations/`，所以發行包不會缺這批資源。

傷害字串那一行是畫在地圖上、演出結束之後（`drawBattleScene` 那條路徑直接
`return`，不會疊在演出上），本輪已移出一般玩家路徑。

## 尚未涵蓋

- 「戰鬥動畫退化」的具體症狀。以上三項都沒有重現出退化；要繼續查需要更具體的
  現象描述，或指定是哪一個發行包、哪一組角色。
- 原版演出的逐幀分鏡與時序，以及與重製端逐張對照。本輪只比對了版面與既有
  命中 fixture。
- 音訊。
