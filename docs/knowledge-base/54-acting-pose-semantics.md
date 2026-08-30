# 54 — runtime map pose / motion lifecycle

> 本檔只保留可由原版 writer／consumer 直接重現的結論。2026-07-15 曾用錯位
> 48-entry acting dump 對影片推測角色、slot 與 pose；那批推論已刪除，不得作 remake
> binding。正確 acting bank 與 editable transcript 見 doc47、doc50。

## 1. 0x50-byte runtime record

| byte | 已證實的狹窄意義 | direct evidence |
|---|---|---|
| `+0` / `+1` | map grid X / Y | movement writers `0x12fd5`、`0x13137`、`0x132f5`、`0x13441`; acting `0x13937..0x13949` |
| `+2` | `0x11019` 回傳的 FDICON cache slot | player materializer `0x10a92..0x10aa2`; renderer `0x12831` |
| `+3` | map pose/direction, `0=下、1=左、2=上、3=右` | four movement entries `0x12edf`、`0x13040`、`0x131ba`、`0x13349`; renderer `0x12835` |
| `+4` | current grid-step placement offset | movement writers and renderer `0x12839..0x128e5`; **不是 animation frame** |

玩家 runtime materialization `0x10a77..0x10aad` 在 copy persistent record、覆寫
X/Y、取得 cache slot 後，明確寫 `+3=0`、`+4=0`。FDFIELD spawn constructor
`0x10ec5..0x10eed` 也寫同一組初值。因此 persistent JOIN/scenario `Dir` 不可直接冒充
這兩個 battle-local bytes；原版會在 materialization 時建立它們。

## 2. 一格移動的 writer lifecycle

`0x13488` 依方向 byte dispatch 四個函式：

| pose | function | 每格結束的座標 mutation |
|---|---|---|
| 0 下 | `0x12eaa` | `Y++` |
| 1 左 | `0x1300d` | `X--` |
| 2 上 | `0x13185` | `Y--` |
| 3 右 | `0x13315` | `X++` |

四個函式的共同 ABI：

1. entry 先把 runtime `+3` 寫成該方向。
2. 一格內的 draw loop 寫 `+4=1..6`，每拍呼 `0x1297d` 並重組 map frame。
3. 第七個邊界更新 `+0/+1` grid coordinate。
4. exit 固定寫 `+4=0`；`+3` 不復原，因此保留最後朝向。

scripted acting `0x1366a` normal branch相同：`0x1389a/+3`、`0x138a2/+4=1..6`，
到 `0x13937..0x13949` 搬一格後在 `0x1394c` 寫 `+4=0`。special branch只寫
`+3`、不搬格。acting unit byte是當下 runtime array 的原始 slot，不可用 Fig-first
解析同素材角色。

## 3. renderer selector lifecycle

`0x127e0` 的 frame index 是：

```text
unit[+2] * 12 + unit[+3] * 3 + cycle
```

- `unit[+4] == 0` 取 idle selector `[0x53c0b]`。
- `unit[+4] != 0` 取 moving selector `[0x53c07]`。
- selector 3 在 consumer 端 alias 成 1。
- `unit[+0x26] != 0` 強制 cycle 0，並另使用 native 0/1 pixel shift。

`sub_1297d` 是兩個 selector 的共同 writer：

- moving `[0x53c07]` **每次呼叫必定** `0→1→2→3→0`；
- idle `[0x53c0b]` 只有 signed `timerTick-last < 0` 或 `> 4` 才循環並更新
  `[0x53c0f]`。

remake pure ABI 是 `indexedmap.AdvanceNativeMapSpriteCycles`。實際 Ebiten adapter
仍須 materialize 原版 BIOS timer tick/call timing；不得拿一般 frame counter 猜代。

## 4. remake binding boundary

`battle.Unit.Dir` 現在與已解 pose enum 數值相同，但它仍是 normalized engine field。
原版 presentation adapter要成立，必須同時保存／供應：

1. materialization 後的 raw pose `+3`；
2. 移動中逐拍 raw motion `+4`，完成後歸零；
3. `0x1297d` 的兩個 selector globals與BIOS timer timing；
4. raw `+2` cache slot、`+5` flags、`+0x26` force/pixel-shift gate。

在這些 state 尚未由所有 movement/acting writer 一致維護前，GUI 必須 fail-closed，
不可只把 `Dir`、`OffX/OffY` 或 `g.frame` 塞入 native indexed compositor。

## 5. 剩餘驗證

- [x] `[0x46c]` 是 low 16-bit BIOS timer tick；`0x17aa9` 以 0x10000
      wrap correction 作 tick busy-wait，`0x16d00` 也以兩 ticks gate文字更新。
- [x] 以約 18.2065 Hz 的硬體規格近似驅動 acting 的七拍格線動作；60 Hz
      Ebiten 更新不可直接視為原版來源拍。此項只約束玩家可見移動時長，與
      native compositor 每次呼叫推進 moving selector 的 call-count 語意分離。
- [ ] 建立單一 battle-local raw presentation state，讓玩家移動、AI、acting與事件
      placement全部走同一 writer API，避免 normalized `Dir` 和 raw `+3` 漂移。
- [ ] 接上 steady／acting／`0x22253` 三條 indexed scheduler 後做原版逐幀截圖比對。
