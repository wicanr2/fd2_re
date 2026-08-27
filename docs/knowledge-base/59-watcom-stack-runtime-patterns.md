# 59 — Watcom DOS stack check／probe 辨識知識庫

## 目的

Watcom 32-bit DOS 程式常在大量函式序言插入 stack runtime helper。若把這種共用
函式當成遊戲事件、renderer 或人工智慧 helper，caller 數會嚴重污染未知函式清冊，
也會讓每個函式的區域變數位移判讀錯誤。本文件保存可跨函式重用的辨識方法；
FD2 的版本綁定證據另見
[`fd2_watcom_stack_check_36cd7_ida.txt`](../data/ida/fd2_watcom_stack_check_36cd7_ida.txt)。

## 不可混用的三種行為

| 類型 | 直接行為 | 常見證據 | remake 處理 |
|---|---|---|---|
| stack limit／overflow check | 計算預期 SP／ESP，與 process stack lower limit 比較；失敗時報錯退出 | `ESP-size`、stack-limit global、`Stack Overflow!` | 分類為 runtime，不移植 |
| page-touch stack probe | 依 page size 逐頁讀／寫即將使用的 stack，預先觸發 guard page | `sub reg,0x1000` 迴圈、每頁 memory touch | 分類為 runtime，不移植 |
| stack grow／allocator | 真正改變 stack segment、commit 範圍或配置替代 stack | SS／selector 切換、DOS extender API、持久改變 SP／ESP | 依平台 runtime 分類，必要時另查規格 |

只有第一列已在固定版 FD2 `0x36CD7→0x36CEA→0x36D07` 證實。舊筆記把它叫
「guard-page probe」過寬；其本體沒有逐頁 touch 迴圈，應稱 stack-demand／overflow
check。遇到其他 Watcom binary，不可只因 prologue 相似就假設三者之一，仍須看 callee。

## FD2 已證實 pattern

caller：

```asm
push required_stack_bytes
call stack_check_wrapper
push saved_registers...
sub  esp, local_bytes
```

wrapper：

```asm
xchg eax,[esp+arg_0]  ; EAX 取得需求量，參數槽暫存 caller EAX
call stack_limit_check
mov  eax,[esp+arg_0]  ; 復原 caller EAX
retn 4                ; 移除需求量參數
```

check：

```asm
prospective_esp = esp - required_stack_bytes
if prospective_esp is safely above process_stack_limit:
    return
if current_ss != expected_stack_selector:
    return
exit_with_message("Stack Overflow!\r\n", 1)
```

`required_stack_bytes` 是編譯器估算的最大 stack demand，可能涵蓋保存暫存器、local
frame 與最深 outgoing-call 參數；它不一定等於後方 `sub esp,N`，也不能直接用來
推導結構大小。helper 對 caller EAX 與參數 stack 的淨效果為零，真正 frame 配置仍
由 caller 後續序言完成。

## 快速分類流程

1. 先看 caller 形狀：是否大量函式都在入口附近 `push immediate; call same_target`。
2. 看 target 是否極短、caller 跨越互不相關子系統、沒有資料 xref，且參數呈現多種
   frame-size 常數。這些只構成 runtime 候選，不是結論。
3. 直接讀 target bytes：確認是否保留暫存器、計算 prospective SP／ESP、比較 stack
   bound，或逐頁 touch。不要以反編譯器自訂名稱代替指令。
4. 沿失敗臂找 consumer：overflow 字串、exit code、DOS extender exception 或 stack
   grow API。沒有失敗 consumer 時維持強推論。
5. 抽查至少三個大小不同、子系統不同的 caller，確認 call 位於序言且後續才配置
   frame；若 call 出現在函式中段，必須另行解釋。
6. 以非破壞性索引記錄原始位址／名稱、runtime 分類、推論等級與證據。分類後可從
   「待判讀遊戲函式」佇列排除，但原始 xref 仍保留。

## 可安全省略與不可省略

可在 remake 語意分析中省略：

- 每個 caller 重複出現的 stack-check call。
- stack-size immediate 本身，除非正在重建原版 ABI 或診斷 stack corruption。
- overflow message 與 DOS extender stack limit 實作；現代 Go／Rust／C++ runtime
  自行管理 stack。

不可因此省略：

- helper call 之後真正的 `push`／`sub esp`，因為參數與 local offsets 仍依它們變化。
- 函式中段的同 target call；必須確認不是 alloca、遞迴前檢查或另一種 wrapper。
- 失敗臂之外的額外 side effect。若 callee 同時寫遊戲 global，就不能只分類 runtime。
- 不同 binary／編譯器版本的位址與 pattern；必須重新綁定雜湊與直接 bytes。

## FD2 分類結果

- `0x36CD7..0x36CE7`：runtime wrapper，已證實。
- `0x36CEA..0x36D07`：stack limit／selector check，已證實。
- `0x36D07..0x36D16`：`Stack Overflow!` exit consumer，已證實。
- `0x36CD7` 的541個直接 call sites 不代表541項遊戲功能；後續 handler／AI／UI
  分析應先排除此 prologue call，再統計真正的產品 callee。
