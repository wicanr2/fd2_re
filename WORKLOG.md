# 工作歷程

目前分層狀態以 [58](docs/knowledge-base/58-fd2-exe-re-coverage.md)為準，完整匹配範圍依 [118](docs/goal/118-goal-fd2-matching-decompilation-20261008.md)。既有時間序列保留於[工作清單歷史](docs/knowledge-base/91-worklist-history.md)與該頁引用的研究紀錄。

## 2026-10-09 原始 ESI 輸入的完整 C 匹配

- 新增 [C 來源](tools/fd2_matching_game_class_stack.c)與[回歸](tools/test_fd2_matching_input_esi.py)，登錄於既有編譯驅動器的 `game_class_stack`。`sub_12DAC` 完整 140 bytes 匹配。輸入 ESI 在原始指令先讀後寫，保留其未知初值，不補零。
- 14 個表示模型各編譯 3s／4s／5s 與預設／space／speed，共 126 份候選，獨立重建兩輪。所有 OMF、完整報告及產碼相同，126 份另經 objcopy 核對。五份匹配只增加一個原始函式；31385／2DF6B 的模型仍未匹配。
- 全檔組合保留舊 75 份來源報告的順序，新報告最後加入。舊 185 段 C 出處逐項相同；SDK 19 函式的出處保持，SDK 原生連結與全檔收據另重建兩輪。完整 357074-byte EXE 的 SHA-256 與固定輸入相同。其餘原版 code 尚未還原，#198 與 Goal 保持開啟。
- 回歸核對完整原始函式、ESI 先讀後寫、EBX／ESI 保存與 16-bit 有號讀取。缺來源、來源內容與來源雜湊錯誤均於輸出前拒收。既有符號／LE 反例回歸首次誤用未含 190AC 的新物件，改以原有 `treasure-fixed-owner-r1` 乾淨重跑通過，未改守衛。
- 原作參數宣告、輸入 ESI 的來源與實際 GS 修改仍未知。`parm`／`modify exact` 僅為相容產碼表示。已知位址由既有 IDA 9.4 匯出機械核對，不重開既有隊伍／面板語意。正式 Go／Ebiten 與章驗收等級保持。

提交前核對 102 份來源雜湊、1205 條本地連結、191 條教訓與 76 個守衛。真正主機 Issue 清單經官方 pull／render／verify，16 條仍開啟，8 條需人工判定，可能已完成 0 條。原始 IDA 資料庫雜湊保持，變更檔與新產物 UID／GID 為 1000；歷史 root-owned 共 2811 項保持，沒有 `.md` 目錄。本批一次性 FD2 容器均已退出。提交與真正遠端 HEAD 結果回填 [#198](https://github.com/wicanr2/fd2_re/issues/198)。

原始位址、雜湊、工具版本、caller、精確旗標、完整候選與驗證結果保存在[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_class_stack_matches`、`class_stack_original_evidence`、`validation.class_stack` 與 `bootstrap`。公開提交保存自行重寫的 C、工具及整理證據；完整 EXE、SDK 物件與工具二進位留在本機 `work/matching-full-20261008/`。

編譯與連結沿用[驅動器](tools/fd2_matching_game_restore.py)：在既有 Docker 工具鏈內執行 `compile --source-key game_class_stack --output /work/class-stack-final-r1`，再執行 `link --objects /work/class-stack-final-r1 --evidence /work/ida-full-image.json --output /work/class-stack-final-linked-r1`。容器與唯讀編譯器掛載方式沿用主收據的工具鏈紀錄。第二輪使用新的 `r2` 輸出，不覆寫第一輪。

## 2026-10-10 五個 DOS／檔案流程的完整 C 匹配

- 新增 [C 來源](tools/fd2_matching_game_dos_file.c)與[完整函式回歸](tools/test_fd2_matching_dos_file.py)，登錄於既有驅動器的 `game_dos_file`。361CC／36284／36344／36900／36955 完整 137／109／311／85／173 bytes 匹配，共 815 bytes。
- 36284 保留兩個分開的無號條件。36344 保留 204-byte 局部布局、早期錯誤返回及下一次讀取位置。其餘三項以相容 callee 宣告保留原始 EBX 跨呼叫存活，沒有 inline code 或位元組補丁。原始 unknown 分類、作者宣告與完整間接副作用集合仍分開記錄。
- 14 個模型共 126 份候選，獨立編譯、連結兩輪，OMF、完整報告與產碼相同；126 份全部由 objcopy 獨立核對。17 份正例只計五個原始函式；其餘模型保持負例。
- 舊 76 份來源報告順序保持，新報告最後加入。舊 186 段 C 出處逐項相同；SDK 19 函式／1200 bytes 出處保持，在新 C 基準實際重跑原生連結兩輪，全檔與收據相同。完整 EXE 的大小及 SHA-256 與固定輸入相同，其餘原版 code 尚未還原，#198 與 Goal 保持開啟。
- 回歸核對完整 815 bytes、原始無號分支、堆疊布局與指標跨 close 存活。十一項原始名稱／CDECL 修飾別名錯址及一項原始 LE 矛盾均拒收，CLI 矛盾在輸出前停止。既有符號、明示位移與 LE 守衛通過。
- 第一輪連結曾缺少 CDECL 前置底線別名。直接檢查 COFF relocation 後補上受原始 caller／LE 核對的別名，未放寬守衛。將錯誤返回改回原始布局後，完整函式才收件。已知位址沿既有 IDA 9.4 匯出機械核對，沒有重開玩家語意或提升正式 Go／Ebiten 與 PLAYER-E2。

精確正例旗標、原始函式與 caller、來源雜湊及驗證結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_dos_file_matches`、`dos_file_original_evidence`、`validation.dos_file` 與 `bootstrap`。編譯與連結使用[既有驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_dos_file --output /work/dos-file-final-r1` 與 `link --objects /work/dos-file-final-r1 --evidence /work/ida-full-image.json --output /work/dos-file-final-linked-r1`，第二輪使用新的 `r2` 輸出。公開保存重寫 C、工具與整理證據；完整原版／SDK 二進位留本機。

提交前核對 104 份來源雜湊、1212 條本地連結、192 條教訓與 77 個守衛。真正主機 Issue 清單經官方 pull／render／verify，16 條仍開啟，8 條需人工判定，可能已完成 0 條。正式 IDA 資料庫雜湊保持，變更檔與新產物 UID／GID 為 1000；歷史 root-owned 共 2811 項保持，沒有 `.md` 目錄。本批 FD2 容器已全部退出。提交、推送及真正遠端 HEAD 核對結果回填既有 #198。

## 2026-10-10 狀態群組與原始載入位元組守衛

- 新增 [C 候選](tools/fd2_matching_game_state_group.c)及[完整群組／載入位元組回歸](tools/test_fd2_matching_state_group.py)。Watcom 10.0a 與 9.5 各 261 份候選均未完整匹配，獨立編譯與連結兩輪，OMF、完整報告及產碼相同；522 份全部由 objcopy 核對。原有 C 覆蓋、191 段出處及 19 段 SDK 出處保持。
- 22AF6 的原始欄位比較使用 EAX，基準候選使用 ESI，差異限於三個編碼位元組。固定 C 的 3s 預設四函式群組中，9.5 保留原始條件跳躍及 RET；10.0a 額外共用 22AA7 的 RET。最後 27 bytes 局部相同仍不收件，不推定原作精確版本、作者宣告或新玩法。
- 隔離反例只改匯出的三條 `loaded_bytes`，舊比較器會誤報 22AF6 匹配。原版與正式匯出保持不變，偽造輸出不計覆蓋。新增守衛從原始 `file_bytes` 及真正 LE fixup 重算每條載入指令；正式 75,427 條指令與 7,003 個 CODE 重定位位置全部相同，三項改寫與 CLI 偽造在輸出前拒收。
- 舊有 815-byte 五函式正例重新連結及回歸通過，報告除新驅動器雜湊外完全相同。既有符號、LE 與明示位移守衛通過。[56](docs/knowledge-base/56-fd2-remake-sdd.md)、[57](docs/knowledge-base/57-ui-evidence-matrix.md)與[58](docs/knowledge-base/58-fd2-exe-re-coverage.md)同步說明修正範圍，正式 Go／Ebiten 與 PLAYER-E2 保持。
- 261 份 9.5 候選兩輪曾在固定 90 秒內只完成 217／215 份物件，屬批次時限不足。驅動器依 `max(90, min(600, 30 + trial_count))` 設定有界時限，寫入 `runner_timeout_seconds`，同 C 與同候選的兩輪 291 秒批次均完成。失敗目錄保留，不算 C 不匹配。

來源、精確旗標、原始函式、修正前後反例與驗證結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_state_group_attempts`、`state_group_original_evidence`、`validation.state_group`、`validation.loaded_image_guard` 及 `validation.compiler_batch_bounds`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_state_group --compiler-version 10.0a` 或 `9.5`，各用新的輸出目錄，再以固定原檔及 IDA 匯出執行 `link`。全檔 C 反編譯尚未完成，#198 與 Goal 保持開啟。

提交前核對 106 份來源雜湊、1229 條本地連結、194 條教訓與 79 個守衛。真正主機清單經官方 pull／render／verify，16 條仍開啟，8 條需人工判定，可能已完成 0 條。正式 IDA 匯出與資料庫雜湊保持，變更檔與新產物 UID／GID 為 1000；歷史 root-owned 共 2811 項保持，沒有 `.md` 目錄。本批 FD2 容器全部退出。提交、推送及真正遠端 HEAD 核對結果回填既有 #198。

## 2026-10-10 開場與戰後較大函式完整 C 匹配

- 新增 [C 來源](tools/fd2_matching_game_long_sequences.c)與[完整序列回歸](tools/test_fd2_matching_long_sequences.py)。3231B 完整 1626 bytes 及 2548C 完整 715 bytes 匹配，共新增兩個原始函式2341 bytes。既有開場／戰後語意沿用原證據，原始分類、作者宣告、正式 Go／Ebiten 與 PLAYER-E2 保持。
- 六個模型各比較三種 CPU 與三種成本策略，共54份候選，獨立編譯、連結兩輪，OMF、完整報告及產碼相同，全部由 objcopy 核對。25份正例只計兩個原始函式；32-bit開場計數控制未匹配。
- 開場保留 BL 的8-bit計數、15／13次迴圈、所有呼叫與32／31／0全域寫入順序。戰後保留35BBA後讀取指標、原始欄位寫入與兩個淡出／淡入迴圈。初輪指標模型曾提前讀取，已依原始順序修正並新增回歸；缺少32975的外部綁定已按直接call補齊。宣告及volatile只作相容產碼表示，延遲呼叫參數不證明硬體wall-clock。
- 舊77份來源報告順序保持，新報告最後加入。舊191段C出處及SDK19函式／1200 bytes出處逐項相同；新C基準實際原生連結兩輪，全檔與收據相同，完整EXE大小及SHA-256與固定輸入相同。其餘原版code未還原，#198與Goal保持開啟。
- 176列原始具名參照、兩個完整IDA函式及caller保存；五項新符號錯址、既有LE／位移與75427筆載入記錄／偽造拒收回歸通過。較早六份9.01單輪候選均未匹配，另經objcopy核對，不納雙輪小計。
- 發布前核對發現IDA的caller清單來自間接派送關係，沒有E8直接呼叫。改以原始LE表項及四個scale4間接call位置保存consumer；初版文件產生器的直接caller假設在寫入主收據前已拒收，沒有提升錯誤證據。

精確旗標、原始函式、來源與驗證結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_long_sequences_matches`、`long_sequences_original_evidence`、`validation.long_sequences` 與 `bootstrap`。單輪9.01比較在 `early_901_comparisons`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_long_sequences`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具及整理證據，完整原版／SDK二進位留本機。

提交前核對108份來源雜湊、1229條本地連結、195條教訓與80個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出及資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 演出／介面C與堆疊實參核對

- 新增[C候選](tools/fd2_matching_game_animation_panel.c)、[實參與保存回歸](tools/test_fd2_matching_animation_panel.py)及[IDA導覽匯出器](tools/ida_export_fd2_local_navigation.py)。28784完整744 bytes與2EBE0完整943 bytes的13個模型，由10.0a／9.5各編譯117份，兩輪OMF、完整報告與產碼相同，234份全由objcopy核對，均未完整匹配。C覆蓋、193段C出處及19段SDK出處保持。
- 原始288E5的`[esp+24h]`與288E9的`[esp+28h]`在不同PUSH之後，都讀caller局部基準+18h。初版C誤傳saved指標的位置已修正為兩個figure來源，原始運算元與有效偏移分列，不用運算元表面差異推定不同來源。
- 演出原始EDI在迴圈初次判斷前沒有局部初始化；來源保留未知，不補成零值。29164兩個已核對返回尾端均還原EDI。三個graphics callee也實際保存EBX；候選pragma允許修改EBX只作保守compiler契約，不能說成實際clobber或原作者宣告。

零值初始化只作`ANIM_LOCAL_2`的未匹配對照，不當作原始初值。
- 原始36／56-byte局部配置、16-bit容量讀取、三個真正E8 caller窗口、107列原始參照與十項錯址拒收回歸保存。既有LE／位移及75427筆載入記錄／偽造拒收回歸通過。兩份WPP前端單輪對照也未匹配，明示前端及固定包裝，另列且不納C前端雙輪小計。
- IDA9.4沿既有py312映像，以正式i64唯讀掛載後的一次性複本輸出導覽。受版控工具重生原檔／DB SHA一致、1305函式、schema及UID核對的JSON；Hex-Rays型別、變數名與推測stack位置只作導覽，不當原作者或位元組證據。技能入口的符號連結指向工具專案，已沿真正權威路徑讀取，沒有另建映像。
- 空的`parm []`後不能再列EDI的編譯語法失敗已修正為EDI輸入在先、堆疊參數在後，機器輸入位置保持。未完成編譯與其後缺收據的診斷不計負例。較早2A2E8的旗標依賴尚未建立C候選，沒有猜補或改列原作組語。

完整來源與結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_animation_panel_attempts`、`animation_panel_original_evidence`、`validation.animation_panel` 與 `animation_panel_cpp_supplement`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_animation_panel --compiler-version 10.0a` 或 `9.5`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，完整原版、i64／導覽及SDK物件留本機；#198與Goal保持開啟。

提交前核對111份來源雜湊、1238條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 戰後對話與資源切換完整 C 匹配

- 新增[C來源](tools/fd2_matching_game_post_dialogue.c)及[備份／副作用與派送回歸](tools/test_fd2_matching_post_dialogue.py)，24754完整960 bytes匹配。三個C表示都重現三個17-byte局部備份、分支與章節INC在context推入之後／CALL之前的原始順序，不能由相同產碼推定原作者副作用寫法。
- 27份候選獨立編譯、連結兩輪，OMF、完整報告與產碼相同，全部由objcopy核對。3s預設／speed的六份正例只計一個原始函式；其餘CPU／成本策略按完整區間拒收。
- 舊78份來源報告順序保持，新報告最後加入。舊193段C出處及SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同，357074-byte EXE的SHA-256與固定輸入相同。原版其餘code尚未還原，#198與Goal保持開啟。
- 72列原始具名參照、完整IDA函式與LE表項22／25E23的scale4間接consumer保存。原始60-byte局部配置、4×MOVSD＋MOVSB三次複製、fade步進2／上限64、六項錯址、既有LE／位移與75427筆載入記錄／偽造拒收回歸通過。既有戰後語意、原始分類、正式Go／Ebiten、PLAYER-E2及硬體時序保持。
- 另查237D5為含多個入口與區間外共用尾端的owner，沿既有覆蓋與完整區間政策保留限制，沒有將入口片段裁切成完整C匹配。

精確旗標、來源與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_post_dialogue_matches`、`post_dialogue_original_evidence`、`validation.post_dialogue` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_post_dialogue`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對113份來源雜湊、1245條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 六模式分派完整 C 匹配

- 新增[C來源](tools/fd2_matching_game_overlay_dispatch.c)及[條件邊／記錄寫入回歸](tools/test_fd2_matching_overlay_dispatch.py)，122DC完整1051 bytes匹配。if鏈及逐分支返回兩種表示都在3s預設成本策略下匹配；原始分類、型別與作者宣告未知，既有語意保持。
- 18份候選獨立編譯、連結兩輪，OMF、完整報告及產碼相同，全部由objcopy核對。初次switch探針在CODE前置24-byte跳表，函式入口不在起始，工具拒收；該未完成探針不列雙輪總數，也不增加覆蓋。
- 核對六個原始模式條件邊、37個明示helper CALL、兩個返回位置及第五模式共用CALL目標。模式6保留原始四位元組記錄的+7寫入，不把C型別或offset推成新語意。125列原始參照、完整IDA函式及11CF0／18BF7兩個E8 caller窗口保存；六項錯址、既有LE／位移與75427筆載入記錄／偽造拒收回歸通過。
- 舊79份來源報告順序保持，新報告最後加入。舊194段C出處與SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256與固定輸入相同，其餘原版code尚未還原，#198與Goal保持開啟。
- 篩選時另外核對32D18、244B6、234BB等完整owner；多入口、額外chunk與區間外共用尾端不裁切計數。這些已知布局限制未改動既有分類，也未重開戰役語意或正式Go／Ebiten。

精確旗標、來源與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_overlay_dispatch_matches`、`overlay_dispatch_original_evidence`、`validation.overlay_dispatch` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_overlay_dispatch`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對115份來源雜湊、1252條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 音效過場、事件尾呼叫與對話備份候選

- 新增[過場C](tools/fd2_matching_game_sound_transition.c)、[事件C](tools/fd2_matching_game_tail_calls.c)與[完整回歸](tools/test_fd2_matching_tail_and_backups.py)。3396A／34C76／35487完整324／61／86 bytes匹配，三個E9均到完整callee入口，由C自然產生。15份正例只計三個原始函式，共471 bytes。
- 另保存[對話備份C](tools/fd2_matching_game_dialogue_backups.c)的九種表示。165AC完整688 bytes的81份候選均未匹配；參數槽、求值順序與暫存器配置仍有差異，聚合及volatile表示不證明作者宣告。後續先核對callee保存契約，再決定相容宣告，不裁切相同片段或提高原始分類。
- 三組108份候選完整編譯、連結兩輪，OMF、完整報告及產碼相同，全由objcopy核對。早期探索來源只留本機，正式矩陣由目前受版控來源乾淨重跑，未完成探針不列入小計。
- 核對三筆LE表項：3396A為51D71表項24，由25E3A消費；34C76／35487為51B91表項35／56，由19511消費。表項檔案偏移與IDA線性位址分列。過場只載入#88 selector1，保留一般玩家指令#80的既有證據；三次delay實參600與20／20／20／60等待值不推成硬體wall-clock。
- 原始42列對話參照、兩個E8 caller窗口、兩次有號除法與sum零值旁路、五次26668-byte配置、五次備份／繪製及返回53A18保持。七項備份錯址、三個錯誤尾目標、既有LE／位移及75427筆載入記錄／偽造拒收回歸通過。
- 舊80份來源報告順序保持，兩份正例報告最後加入；195段舊C與SDK19函式／1200 bytes出處保持。新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256仍等於固定輸入；其餘原版code尚未還原，#198與Goal保持開啟。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_sound_transition_matches`、`game_tail_event_matches`、`game_dialogue_backup_attempts`、`tail_and_backups_original_evidence`、`validation.tail_and_backups` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_sound_transition`、`game_tail_events` 或 `game_dialogue_backups`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對119份來源雜湊、1265條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 START／LOAD完整主流程與對話保存契約

- 新增[主流程C](tools/fd2_matching_game_start_load.c)及[欄位／重試回歸](tools/test_fd2_matching_start_load.py)，25EBB完整663 bytes匹配。槽位址的LEA排列與零實參表示共同重現原始產碼；不能由相同產碼推定作者C表示或型別。
- 22987-byte緩衝、12587-byte header、2600-byte槽、2560-byte記錄複製與七個原始欄位讀取保持。+2欄位為未對齊4-byte載入，其餘指定欄位為byte；255轉selected零值後回26016，原始51D71間接表與25DBD直接caller保留。
- [對話C](tools/fd2_matching_game_dialogue_backups.c)新增五個宣告候選，共45份完整負例，原81份歷史負例保持。原始4E031只改AX並還原ESI；4E96F以PUSHA／POPA保存通用暫存器；15E71保存EBX，15E9E經22BBE共享收尾還原EBX。保守pragma允許修改與實際callee保存分列，不宣稱原作者ABI。
- 兩組90份候選完整編譯、連結兩輪，OMF、完整報告及產碼相同，全由objcopy核對。七項錯址、既有LE／常數位移及75427筆載入記錄／偽造拒收回歸通過。
- 首輪pack(push,1)／pack(pop)被10.0a以E1054／E1009拒收，未產生完整編譯收據；不計成產碼負例。改用pack(1)／pack()後乾淨重跑，sizeof斷言確認10／2600 bytes。教訓與守衛寫入既有[台帳](docs/data/fd2-lessons.json)。
- 舊82份來源報告順序保持，新正例最後加入。198段舊C與SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256保持；其餘原版code未還原，#198與Goal保持開啟。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_start_load_matches`、`game_dialogue_backup_contract_attempts`、`start_load_original_evidence`、`validation.start_load` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_start_load`；對話補充用 `game_dialogue_backups --cases DIALOGUE_BACKUP_9 DIALOGUE_BACKUP_10 DIALOGUE_BACKUP_11 DIALOGUE_BACKUP_12 DIALOGUE_BACKUP_13`。各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對121份來源雜湊、1274條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 終局對話完整C與1718-byte owner候選

- 新增[對話C](tools/fd2_matching_game_terminal_dialogue.c)、[owner C](tools/fd2_matching_game_terminal_body.c)與[完整回歸](tools/test_fd2_matching_terminal.py)。2C39B完整106 bytes匹配，保留A9514h目的位址、原始呼叫順序、EBX保存及2BCE5內12個直接caller。兩種保守宣告的四份正例只計一個函式。
- 2BCE5完整1718 bytes仍未匹配。八種表示保留三份20-byte表、84-byte局部布局、40／200／20迴圈、有號除法、原始欄位寫入及兩個外層caller。局部相同不增加覆蓋；原有終局語意、分類與一般玩家驗收保持。
- 兩組99份候選獨立編譯、連結兩輪，OMF、完整報告及產碼相同，全由objcopy核對。72份owner候選均負例，預設對話宣告103 bytes也按完整區間拒收。
- 4E031實際只改AX，1956B及4E63D保存EBX的原始指令另存；保守pragma允許修改不當作實際clobber或作者ABI。九項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。
- 初次連結因sub_20421未登錄被ld拒收，已依原始E8目標補齊，不計成產碼負例。回歸初稿假設預設對照104 bytes，實際物件為103 bytes；已改讀實際收據，完整位元組負例檢查保持。兩者屬工具／驗證問題，不列為產品缺陷。
- 舊83份來源報告順序保持，新正例最後加入。199段舊C與SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256保持，其餘原版code未還原，#198與Goal保持開啟。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_terminal_dialogue_matches`、`game_terminal_body_attempts`、`terminal_original_evidence`、`validation.terminal` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_terminal_dialogue` 或 `game_terminal_body`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存完整C候選、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對124份來源雜湊、1283條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 原廠SDK完整群組與額外來源

- 先核對剩餘分布及固定SDK證據，未將未知函式按名稱或相似片段升格。新增[來源工具](tools/fd2_matching_library_extension.py)與[拒收回歸](tools/test_fd2_matching_library_extension.py)，只補既有保留函式庫的來源。
- 原生重建ioalloc／setvbuf／unlink完整119／120／36 bytes，import由既有runtime定位及原始參照逐筆核對。另重建amodf整個302-byte CODE，兩個公開入口0／144及三個完整IDA函式144／75／83 bytes共同驗證。沒有裁切、遮罩、補opcode或改原始名稱。
- 新增六個來源、577 bytes；SDK來源由19函式／1200 bytes增至25函式／1777 bytes。C199函式／29424 bytes、另146-byte未歸屬區塊、175個保留函式庫及931個待還原／分類函式保持。這些來源原本已列為函式庫，不增加C覆蓋。
- 兩輪實際WLINK／WDIS重建的全部CODE、完整報告、收據及357074-byte EXE相同。舊19個SDK來源、200段C出處與完整函式台帳逐筆保持；原始固定EXE SHA-256相同，Goal與#198保持開啟。
- 十項未知／C目標、C出處重疊、裁切邊界、移動或歧義公開入口、偽造載入位元組、跨chunk與物件來源變動均在輸出前拒收。原作精確版本與高階函式語意仍未知。
- 首輪錯把RAW的OFFSET對齊基準當作容器起點，長度守衛拒收。原生map證實唯一CODE從4D7BC起，RAW本身就是完整302 bytes；已依既有工具鏈契約修正並從新目錄重跑，整段比較未放寬。

契約先登記於[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `sdk_extension_spec`，驗證後為CONFORMED；來源與結果在 `sdk_library_extension_origin`、`validation.sdk_library_extension` 與 `bootstrap`。後續C基準先沿用既有library_restore／library_native_restore形成19函式SDK基準，再執行來源工具。CLI的 `--baseline` 與 `--receipt` 指向該中間基準，`--objects`、`--library`、`--evidence`、`--original` 指向固定本機輸入，`--output` 每輪用新目錄。原廠物件、完整CODE／map與EXE留本機；公開工具、雜湊、原始定位與有限caller證據。

提交前核對126份來源雜湊、1290條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 AI物品完整清單與執行流程C匹配

- 新增[C來源](tools/fd2_matching_game_ai_item_execute.c)及[清單／byte／夾限回歸](tools/test_fd2_matching_ai_item_execute.py)，15055完整700 bytes匹配。兩種局部整數重用表示同碼，不推定作者的volatile、pragma或局部變數宣告。
- 保留36-byte局部、byte32讀寫、14818／149F8的完整target list與count，以及交給20C6F的原始順序。小command分支在152E5共同接續；兩個14EF0 caller、八位元減16、signed X／Y夾限、64起始漸暗與1..8閃爍保持原始指令證據，沒有新增效果或E2結論。
- 4E56C實際只改EAX／EDX，4DBFC實際保存EBX／ESI／EDI；保守允許修改的C宣告分開標示。物品欄位與record活躍區間、X暫存值及Y局部重用共同重現原始配置。相同700-byte長度的負例仍按完整區間拒收。
- 90份候選完整編譯、連結兩輪，OMF、完整報告與產碼相同，全由objcopy核對；兩份正例只計一個原始函式。八項錯址、既有LE／常數位移及75427筆載入記錄／偽造拒收回歸通過。
- 舊84份來源報告順序保持，新正例最後加入。200段舊C出處逐筆保持。SDK完整11／19／25來源鏈在新C基準實際重跑兩輪，25函式／1777 bytes出處與原始來源報告保持；十項SDK未知／C／裁切等拒收在新基準通過。擴充輸出的六個來源是相對19函式中間層，與上一輪25來源比較沒有新增SDK覆蓋。
- 357074-byte全檔與收據相同，SHA-256等於固定輸入。累計200個原始函式以C匹配，其餘930個函式待還原或分類；Goal與#198保持開啟，正式Go／Ebiten與既有AI效果驗收不提升。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_ai_item_execute_matches`、`ai_item_execute_original_evidence`、`validation.ai_item_execute` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_ai_item_execute`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。SDK沿既有三層鏈接續重建；公開C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對128份來源雜湊、1296條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 AI物品評分候選與REGS完整C匹配

- 新增[評分C](tools/fd2_matching_game_ai_item_select.c)、[REGS C](tools/fd2_matching_game_int31_regs.c)與[完整回歸](tools/test_fd2_matching_ai_item_select_regs.py)。36255完整47 bytes匹配，保留56-byte REGS、第三參數32-bit讀取後16-bit遮罩、兩段相距28 bytes與int386原始呼叫。三種C表示的18份正例只計一個函式；四個owner內八個直接caller另存，原分類unknown保持，不推定高階INT31h服務或原作者宣告。
- 1567E先核對既有game_ai_sequences來源與歷史負例，再依已知callee保存契約建立18種表示。162份完整候選均未匹配。部分候選與原版同為514 bytes，只差actor乘80的七個bytes，仍按完整區間拒收；C++四份單輪完整負例另列，不併入C雙輪統計。
- 72-byte局部、400-byte配置先於count、零count略過free、位置表步幅2、14818／149F8完整清單、15880的完整count與strict-greater寫入53C33／37／3B／3F均核對原始指令。4E56C只改EAX／EDX，4DBFC保存EBX／ESI／EDI；保守pragma與volatile只作相容產碼表示。
- 兩組189份候選獨立編譯、連結兩輪，OMF、完整報告與產碼相同，全由objcopy核對。六項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。舊85份來源報告順序保持，新正例最後加入，201段舊C出處逐筆保持。
- SDK完整11／19／25來源鏈在新C基準重跑兩輪，25函式／1777 bytes出處與原始來源報告保持；十項未知／C／裁切等拒收通過。首次掛載少一層clib3s.dos而找不到SDK，修正唯讀路徑後用相同命令乾淨重跑。C++初輪malloc隱式轉型E166無完整收據，明確cast後重跑；這些失敗不計產碼負例。
- 357074-byte全檔與兩輪收據相同，SHA-256等於固定輸入。累計201個原始函式以C匹配，其餘929個函式待還原或分類；其餘原版code未還原，Goal與#198保持開啟。正式Go／Ebiten與既有AI效果驗收保持。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_int31_regs_matches`、`game_ai_item_select_attempts`、`ai_item_select_regs_original_evidence`、`ai_item_select_cpp_supplement`、`validation.ai_item_select_regs` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_int31_regs` 或 `game_ai_item_select`，各輪使用新輸出目錄，再以固定原檔及IDA匯出執行 `link`。SDK沿既有三層鏈重建；公開完整C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對131份來源雜湊、1305條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 物品效果與命中完整C候選

- 新增[效果C](tools/fd2_matching_game_item_effect.c)、[命中C](tools/fd2_matching_game_command_hit.c)與[完整回歸](tools/test_fd2_matching_item_effect_hit.py)。20C6F完整1043-byte來源接續已匹配的15055，也保留1BBDC內的玩家caller。1C75E完整193-byte來源保留七個E8 caller；IDA匯出的code xref總數8另列，不把兩種口徑混用。既有物品與職業規則不重解。
- 效果來源保留108-byte局部，將偏移14的16位元欄位擴成32位元存於局部偏移104，將record+60的位元組暫存於局部偏移100再還原；完整目標清單與count、各分支消耗邊界及共用收尾保持。命中來源保留116-byte局部、112-byte表複製、record+32原生職業減一索引、有號16位元讀取、有號除10與餘100、原始命中閘門及1C81F呼叫。
- 12種效果表示與10種命中表示在Watcom 10.0a／9.5、三種CPU與三種成本策略，共396份候選完整編譯、連結兩輪。全部OMF、報告與完整產碼相同，全由objcopy核對，全部仍為負例。型別複製已重現原始REP MOVSD，但同長度193-byte候選仍有六個bytes的暫存器差異，沒有補opcode、遮罩、裁切或計入覆蓋。
- 13項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。1C9DD的兩條退出邊仍經1C916內的1C9C7／1C9D5接22BBE共享收尾；不把172-byte區間單獨當作可返回的完整C函式。保守pragma、volatile、聚合與指標表示只作產碼候選，不推定作者宣告或實際clobber。
- 既有201個原始函式／30171 bytes、另146-byte未歸屬區塊、202段C出處與SDK25函式／1777 bytes逐項保持。沒有新增匹配，沿用既有全檔基準及SDK來源鏈，不把負例放入EXE，不重跑未受影響的SDK重建。固定全檔SHA-256保持，929個函式仍待還原或分類；Goal與#198保持開啟。
- 後續先核對「分開局部變數＋保守的迴圈被呼叫端宣告」及「型別複製＋4E516保守宣告」的交互作用；本批分別驗過各因素，尚未驗證這兩個組合。不重跑本批相同來源矩陣來尋找偶然成功。RND只核對原始產碼，沒有執行或宣稱亂數對拍。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_item_effect_attempts`、`game_command_hit_attempts`、`item_effect_hit_original_evidence`、`validation.item_effect_hit` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_item_effect` 或 `game_command_hit`，版本用 `--compiler-version 10.0a` 或 `9.5`。各輪使用新輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開完整C候選、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對134份來源雜湊、1314條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 ACTING與標題共享返回段C群組

- 前輪預定的兩個宣告組合已核對。20C6F新增分開局部與保守呼叫宣告、16位元欄位／32位元容器、直接真值等六種表示；1C75E新增型別複製與4E516保守宣告的組合。兩個歷史編譯器共126份新候選均負例，舊396份不重跑。1C75E的原始矩陣仍由78a05a7b提交中的來源與驅動器重生，不改寫舊收據。
- 新增[完整群組C](tools/fd2_matching_game_title_acting_group.c)與[回歸](tools/test_fd2_matching_title_shared_group.py)。1366A完整818 bytes與1F894完整1765 bytes共有92-byte局部框架；1FF74以E9跳13994的八位元組收尾。先核對固定原檔、IDA函式邊界、raw bytes與呼叫端，將契約登記為READY，再實作同一C編譯單元，不把1765-byte函式裁成獨立返回函式。
- 來源保留ACTING兩份32-byte局部陣列、原始欄位及更新邊界；標題保留60-byte事件表、五張147列圖、535至0的捲動、原始插播點、存檔校驗、三種選單數量、鍵盤回捲與四次閃爍。聚合、bitfield、volatile與指標重用只作候選，不推定作者宣告。
- 群組六種表示在兩個歷史編譯器共108份候選仍未匹配。編譯器自然產生跨函式共享收尾，既有COFF工具將完整函式分開，只提升跨區間REL32供ld計算，保留指令寬度。共享結構成立不等於完整位元組或入口一致，也不增加C覆蓋。
- 本批六組共234份候選完整編譯、連結兩輪，OMF、完整報告與產碼相同，全由objcopy核對；群組分別擷取兩個完整PE區段再串接。12項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。
- 既有201個原始函式／30171 bytes、另146-byte未歸屬區塊、202段C出處與SDK25函式／1777 bytes逐項保持。沒有新增匹配，沿用既有全檔基準與SDK來源鏈，負例不放回EXE。929個函式仍待還原或分類，Goal與#198保持開啟；既有語意與Go／Ebiten驗收不提升。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `title_acting_group_spec`、`game_title_acting_group_attempts`、`title_acting_group_original_evidence`、`game_item_effect_mix_attempts`、`game_command_hit_mix_attempts` 與 `validation.title_shared_group`。群組編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_title_acting_group`；新組合分別使用 `game_item_effect --cases ITEM_EFFECT_12 ITEM_EFFECT_13 ITEM_EFFECT_14 ITEM_EFFECT_15 ITEM_EFFECT_16 ITEM_EFFECT_17` 與 `game_command_hit --cases COMMAND_HIT_10`。版本用 `--compiler-version 10.0a` 或 `9.5`，各輪使用新目錄後執行 `link`。原版、i64、SDK物件與原生產物留本機。

下一步只對1C75E測一次32位元actor的volatile讀取。原版以EAX讀actor、EBX計算80-byte記錄偏移，目前同長度候選的六個差異都在這段完整索引；若新表示沒有改善，就轉向尚未試過且有完整返回段的函式本體。這是尚未驗證的產碼假說，不是作者宣告。

提交前核對136份來源雜湊、1322條本地連結、197條教訓與82個守衛。舊396份矩陣的C及driver雜湊另以78a05a7b的實際Git內容核對。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 暫停匹配試驗，先提交現有C候選

使用者要求暫停後先commit及push。本批只核對並保存暫停前的來源與收據，沒有再編譯或新增試驗。

- 新增[完整C候選](tools/fd2_matching_game_record_resolution.c)，目標為1DB65的857 bytes。保留124-byte局部、record+5整位元組寫入、word+64、視口公式、13幀姿態、6＋6幀效果及153216-byte背景。完整原始指令與八個直接E8 caller由既有IDA9.4匯出保存，既有狀態／死亡演出語意不重開。
- 驅動器登記11種表示。目前最終來源只有RECORD_RESOLVE_8／9／10的Watcom 10.0a、-mf -3s單次收據，產碼863／845／852 bytes，首差偏移17／17／32，皆非完整匹配。較早局部試驗的來源已變動，不作當前來源的正式矩陣。其餘版本、CPU、成本策略與獨立第二輪仍未完成。
- COMMAND_HIT_11的32位元actor volatile讀取已在10.0a／9.5各測一次，產碼皆193 bytes，首差偏移31，沒有改善。收據保留當時driver雜湊，並驗證由本提交driver移除三條record-resolution註冊行可重建原雜湊；不把歷史driver寫成當前版本。
- 五份既有OMF、完整產碼與來源雜湊核對通過，objcopy獨立擷取五份完整區間。當前driver的錯址、常數位移、75427筆載入記錄及偽造拒收回歸通過。零筆路徑的EAX未在函式本體定義，35BEB caller仍傳遞EAX；不推定所有caller忽略回傳，C型別與pragma維持產碼假說。
- 本批沒有新增C匹配。201個原始函式／30171 bytes、另146-byte區塊、202段C出處、SDK25函式／1777 bytes及既有全檔基準保持。五份單次收據另列，不併入116組6405份候選／6375份獨立核對／30份布局拒收的雙輪統計，不把負例放回EXE。929個函式仍待還原或分類，Goal維持暫停，#198保持開啟，Go／Ebiten及PLAYER-E2不提升。

來源、旗標、物件與產碼雜湊見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的record_resolution_candidate_spec、record_resolution_original_evidence、record_resolution_single_pass、command_hit_actor_single_pass及validation.record_resolution_commit。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的compile --source-key game_record_resolution；單次試驗須明示--cases RECORD_RESOLVE_8 RECORD_RESOLVE_9 RECORD_RESOLVE_10，再用固定原檔及IDA匯出執行link。此入口只供日後重生現有負例，本輪不接續試驗。原版、i64、SDK物件與編譯產物留本機。

提交前核對137份來源雜湊、1328條本地連結、197條教訓與82個守衛。真正主機Issue清單經官方pull／render／verify，16條開啟、8條需人工判定、可能已完成0條。固定原檔、正式IDA匯出與資料庫雜湊保持；變更檔UID／GID1000，歷史root-owned 2811項保持，沒有`.md`目錄。提交前確認本批FD2容器全部退出；推送後核對真正遠端HEAD，結果回填#198。

## 2026-10-15 重製完成與HD排程，匹配研究再次暫停

使用者先要求接續，再明確要求暫停matching decompilation，改查重製缺口、加速重製與HD，並重新調整worklist及GitHub Issues。依復古重製、玩法稽核及工作清單路由載入任務入口；HD架構取捨依grilling確認，沒有新增production規則或素材格式。

- FD2起點2975ecb6，追蹤檔原先乾淨，既有未追蹤work保留。dosgolem已合併；fetch後main與origin/main一致且工作樹乾淨，從81c85be建立fd2-followup-20261015。尚未用整合後程式重生章oracle，分支建立不算行為驗證。
- 暫停前，既有record-resolution來源在Watcom10.0a編譯99份候選；連結容器缺wdis而退出，屬環境失敗。compile-report位於work/matching-full-20261008/record-resolution-resume-20261015/10.0a-r1，SHA-256為5c72a5002fc467bc87d4d196fc77530d705a5915eaf39af4a9f2da74b896a0d0。沒有完整link報告，不列產碼負例、不加雙輪統計；C來源、driver與主收據未改。使用者暫停後未接續編譯，容器已退出。
- 真正主機gh auth成功。讀取遠端Issue後更新16個既有題目的排程、現況與依賴，新增#199～#204六個執行子項。總項#14／#4不與子項重複計算。#198、#52、#197保持開啟，標P3與狀態:暫停；後續平台及編輯器的前置排除暫停研究，保留原驗收內容。
- 遠端22項逐條核對：P0六項、P1八項、P2五項、P3三項，每項恰有一個P級標籤。官方build_snapshot與pull快照完全相同，父子連結及#142→#203依賴通過。工作清單verify沒有可能已完成項；沒有關閉任何未達驗收題目。
- 唯一分層現況58新增本日狀態；56／57同步研究與重製的邊界，61同步HD執行子項，91由官方快照重生。舊匹配研究與章收據保留。沒有重跑已passed章、提高章分級、調整像素預算或恢復第12章目標追蹤。
- 本機缺少fd2-go-test-local:latest及fd2-assets-local:20260829-sfx，受版控Dockerfile仍在；#203記為環境前置。沿用既有eob-remake-release:1.26.7-ebiten2.9.9-20261008-r2容器的Python／Pillow完成清冊機械核對，不用它替代FD2正式Go／GPU驗收，也未建立重複映像。
- 全域sync_native_turn_events.py --check通過，仍有27筆未完整轉寫。現代catalog的161筆結構與私人檔尺寸／SHA／透明度驗證通過；file/files指向的1153檔全部存在。數量與正式章台帳只記在58；原型素材存在不代表全玩家路徑或高解析呈現已完成。
- 工作清單既有19個測試與現代catalog既有1個私人素材測試通過。高解析HD第一階段範圍已提問，#204維持DRAFT；現有核准風格、靜態嘴型、圖塊與存檔語意未改。

本輪可重跑入口，以下Python及測試命令均在上述既有容器內、repo唯讀掛載時執行；render需另將文件輸出所在工作樹可寫掛載並指定UID/GID1000：

```sh
# 只有官方Issue控制面在真正主機執行。
tools/fd2_worklist_issues.py pull
# 以下在Docker內執行。
python3 tools/fd2_worklist.py render
python3 tools/fd2_worklist.py verify
python3 -m unittest discover -s tools -p 'test_fd2_worklist*.py'
python3 -m unittest discover -s tools -p test_modern_theme_catalog.py
python3 tools/sync_native_turn_events.py --check
python3 tools/validate_modern_theme_catalog.py --verify-private
python3 -c 'import json; d=json.load(open("docs/data/parity-campaign-progress.json")); print([(c["chapter"],c["status"]) for c in d["chapters"]])'
python3 -c 'import json,collections; d=json.load(open("remake/assets/themes/modern/catalog.json")); print(collections.Counter((a["role"],a["status"]) for a in d["assets"]))'
```

本輪只重排工作與修正現況文件，未提交或推送程式變更；GitHub Issues已直接更新。歷史root-owned基線2811項未增加，沒有`.md`目錄，變更檔及新研究輸出UID/GID1000。本批容器均退出，其他專案容器未動。

## 2026-10-15 恢復正式工具鏈，補齊IRQ7並回到第24章

使用者選擇先補完整IRQ7鏈，再繼續章對拍。matching decompilation維持暫停。
依復古重製與平台規格優先路由，使用公開DPMI／DOS4GW契約及dosgolem既有305，
先建立429 READY規格，再接真實IVT、DSP確認、PIC EOI、IRET及caller返回。
沒有逐週期ISR考古、輪詢替代或遊戲規則變更。

- 由既有Dockerfile重建fd2-go-test-local及fd2-assets-local，最小context只含五個受版控輸入。
  Go1.22.12與Pillow11.3.0沿用原工具鏈；原版、私人素材、IDA與work未進build context。
  私人現代素材18項通過、零略過；完整Go19套件2383項通過、52項既有選用略過。
- 合併main81c85be的首輪正式原版，在5780804步／LE0x373FE因IRQ7接線缺口拒收。
  actual IVT0F=67FF:06F3，其他兩種0F向量皆零。原始失敗保持。
- dosgolem共用main保持乾淨，專用worktree位於work/dosgolem-fd2-followup-20261015。
  實作e26fb41及驗證規格2c5c104已推送fd2-followup-20261015，遠端HEAD已核對。
  cpu386／machine／FD2 oracle三套件2021項通過；10項略過皆缺MOO2可選原版。
- 真實FD2 ISR每次76條指令。600萬步3次進場／返回；52516893步到標題等待時504次皆返回。
  固定第三方CONTINUE槽及既有鍵盤計畫兩輪皆594524785步，233份完整狀態、233張PNG與3筆trace相同。
  舊／新嚴格比較仍failed，共446筆差異；完整單位、鍵盤讀取與DOS呼叫相同，view只在aux_phase不同。
  211張PNG不同，舊seq0缺圖；不將全部影像差異歸為動畫，不補圖或增加忽略欄位。
- native准入保留舊／合併未修／IRQ7修正版的完整來源組合，修正版另綁派送與實模式橋接兩檔。
  真實Git bytes測試接受三組與當前來源，拒收六檔單獨修改及兩種新舊混搭。oracle相關108項Python回歸通過。
- 現代靜態宣告盤點保留來源SHA與JSON pointer，不複製台詞。九個非負speaker候選缺頭像，負speaker另列；
  尚未證明正式玩家可達，不直接開產品缺陷或生成素材。高解析範圍仍DRAFT，catalog保持runtime_candidate。
- 真正主機Issue更新#142及#201，依限定工具收據關閉#205與#203。官方pull拉回21項；暫停項保持開啟。
  #142接續第23章實際保存槽，SHA-256為22bd63070189e0e6703e14d4bd4db8dc1f66452a21128dd5154579171ecbea63，
  seed4及祖先AP200／DP0／DX60政策保持。原版第24章首輪已到T3、單次清敵、戰後整備及成功DOS保存。
  重播因漏帶既有舞台trace而拒收，屬本輪驗證參數問題；補回既有0x24D48／0x11EED等入口後乾淨重跑。

唯一主證據為[工具恢復收據](docs/data/fd2-remake-toolchain-recovery-20261015.json)，保存輸入、來源、映像及輸出雜湊。
原始原版收據留在work/remake-recovery-20261015；不提升整章PLAYER-E2或公開私人素材。

可重跑命令均在Docker內，repo與原版唯讀、明確輸出可寫、UID/GID1000；Go及素材測試使用恢復的專用映像，
dosgolem使用dosgolem-go-ebiten:1.24-r1。oracle入口與原版／state掛載依[正式驅動器](tools/dosgolem_oracle.sh)。

```sh
FD2_ASSET_PACK=/out/pack FD2_TEST_MODERN_PACK=/repo/remake/generated-assets/modern-theme-prototypes \
  with-xvfb go test ./cmd/fd2 -run 'Test(LoadModern|ModernTheme)' -count=1 -v
FD2_ASSET_PACK=/out/pack with-xvfb go test ./... -count=1 -json
go test ./internal/cpu386 ./internal/machine ./apps/fd2/cmd/oracle -count=1 -timeout 480s -json
python3 tools/test_oracle_native_source_profiles.py --dosgolem <專用worktree>
python3 -m unittest discover -s tools -p 'test_*oracle*.py'
python3 tools/audit_modern_theme_coverage.py --verify-private --output <現代宣告報告>
python3 tools/verify_oracle_prefix.py \
  --prefix work/remake-recovery-20261015/normal-original-irq7-r1 \
  --full work/remake-recovery-20261015/normal-original-irq7-r2 \
  --compare-frames --compare-trace --out <新版兩輪比較報告>
```

舊／新版比較另用原memory197-command-native-original-r1及完整fa25f6e／e26fb41提交對，
`--compare-frames --compare-trace`不變。第24章使用受版控ch24-sample-r3計畫、native近堆及完整舞台／AI trace，
實際初始SAV另拷貝至state覆蓋層；每次運行有界，原始遊戲檔不寫入。

第24章後續結果：補回舞台trace後，r2重播通過，但seq162 PNG被下一個單步控制覆寫。
行為、節點與完整SAV相同，畫面閘門依缺圖拒收；容器exit143與缺少計畫尾端mark也如實保留。
通用單槽pendingFrame問題登記#206、P1，沒有修正或關閉。r4計畫在T3 mark後正常空等待200萬步，
原版r3完整走到退出存檔列表後的整備mark、exit0，211份狀態與211張PNG齊全。
重製側11.10秒通過，16筆AI入口全數消費、零順序分岔；四gate通過、13張完整RGB皆零差異，
兩側完整SAV SHA-256皆a1e6cc85722fc820cf38f820f4454785cb2138450d9c6000226466a340288406。

本次只驗T1～T3、移動、敵方回合與戰後保存，未抽T4／7／10事件、普通攻擊及其他適用戰鬥路徑。
不因四gate或13張零差異直接提升整章111，#142與正式章台帳保持。生成比較收據內嵌於工具主收據的
`chapter24_followup.bounded_receipt`；r4計畫已掛00索引。#206與缺舞台trace的前輪失敗分開記錄。
現代主題未生成新素材、未改格式或正式繪製路徑，高解析範圍仍DRAFT。

本批最後核對：遠端22個未完成Issue，官方pull／render／verify通過，可能已完成0項。
工作清單19項與oracle相關108項Python測試通過；教訓198條、83個守衛通過。
變更文件2238個本地連結通過，20份來源／輸出SHA全部相符。全知識庫掃描另發現9個既有問題，
分別為歷史本機交付物缺失及舊文件中的括號被簡易解析器當連結；均不在本批變更文件，不宣稱全庫連結通過。
原版及私人素材未改，變更檔UID/GID1000，歷史root-owned仍2811項、`.md`目錄0。
dosgolem專用分支及共用main乾淨；FD2提交只加入整理證據、工具與文件，work及大型擷取不入Git。

## 2026-10-15 第24章流程驗收與事件54補齊

使用者允許章前有界 DP 校準，隨後指定作弊過關、參考攻略與流程一致，不要求每個 frame 相同。已同步111／114、56／57／58及 GitHub #14／#142／#33。匹配式反編譯保持暫停。

- 第24章 event54 原先只有登場增援。本輪重用已核對的 IDA 9.4 匯出，與固定 EXE 及 Capstone 5.0.3 交叉確認125 bytes，保留 sub_3535D、sub_35822共用尾端及全域表consumer。規格 READY 後才補四角鏡頭與等待，九項具型別操作同步四筆正式／canonical回合事件。入口為 [event54主契約](docs/data/ida/fd2_turn_event54_20261015.json)。
- 本地青衫攻略第24章記載四角增援，回合為3／4／7／10；固定控制列與原版實測為2／4／7／10。保留差異，攻略只作玩家旁證。
- 新增同章 boost-only 建槽模式，拒收混用推章、升級、事件與金幣。第一輪沿既有 fresh 槽只加DP1000，seed4，25名冊僅100個DP資料bytes及2個checksum bytes不同；HP／MP／物品／金幣與其他槽相同。原最早ch02基底不存在，直接沿已驗fresh產物，沒有猜補來源。
- 流程驗收仍嚴格檢查行為、節點、交易與完整保存。畫面保留原始結果，advisory模式不參與完成判定。測試可省略圖片輸出，正式Game.Update／Draw完成閘門仍執行。31項Python工具測試與建槽、字串Go回歸通過。
- 原有第24章短鏈重新消費新劇本，16筆AI入口全消費且零順序分歧，13張完整RGB零差異，完整SAV相同；這份回歸只驗T1～T3。
- 官方全域事件同步檢查通過，剩23筆未轉寫，沒有將spawn-only提升為完整事件。
- 字串review與新清冊雜湊原已一致，本輪測試Go變更不改production inventory；summary與四語內容落後既有來源，沿官方遷移器補同步，保留原譯文與審查狀態。10筆新來源用明示候選，外語維持machine_draft。四語結構驗證及14項測試通過，其中1項依原環境條件略過。

環境修正：Python建槽測試在唯讀repo先建立cache失敗，改為先拒收錯誤基底檔名再碰cache，乾淨重跑通過。第一次DP位元組核對漏列既有末尾checksum，按fd2save契約補核對後通過，未放寬資料欄位範圍。全docs連結掃描列9個既有歷史／交付路徑缺件，本輪新增連結另行核對，不把它們寫成產品缺陷。

本輪第一輪已正常至T11，T2／4／7／10增援全觸發。重製端重播通過，280筆AI入口全消費且零順序分岔。清敵一次60筆後，原版正常END於9470673688steps異常跳至0x1666E8，next bytes疑似資料，真因未知。沒有戰後SAV，流程收據保留failed，章台帳不提升。已登記[P0 #207](https://github.com/wicanr2/fd2_re/issues/207)，不直接補opcode或再調DP掩蓋。第23章完整流程三閘門回歸通過。

本輪完整Go回歸19個含測試套件全部通過。原版追蹤實際確認event54入口與四個pan、返回均各四次，控制序號134／192／330／538；失敗流程收據已公開索引，章台帳仍20章passed。

提交前已核對本輪31個變更檔UID／GID1000、2259條本地連結與git diff --check。工作清單由真正主機GitHub拉取後render／verify，23項未完成、15項人工判定、可能已完成0項。199條教訓與83個守衛通過。本輪FD2容器均已退出，既有root-owned基線2811項保持，沒有`.md`目錄。FD2專用映像只保留既有可重現版本；未清理其他專案映像或原有IDA工具鏈。

## 2026-10-15 第24章清敵視野與戰後交接接續

接續b8e7ec37與#207。工作樹只有既有work產物，matching維持暫停。原版對拍／規格閘門／文件職責路由及復古重製、IDA技能已核對；dosgolem遠端main仍81c85be，前輪IRQ7分支尚未合併，沿乾淨2c5c104。主機gh auth成功，開工官方pull為23項。

先轉送既有oracle的trace-tail，原預算末端2000條唯讀EIP，並明示寫入runner。診斷重播到第5回合時，已由既有IDA9.4完整匯出找到更窄證據，因此主動停止並保留部分輸出，沒有當作新的完整原版收據。

原版sub_1DB65完整857 bytes與固定EXE／Capstone5.0.3逐指令及邊界一致。局部124 bytes中，30個畫面指標佔0x00..0x77，malloc指標位於0x78；0x1DC29無數量限制地寫入，0x1DEB6..1DEBD依原順序恢復暫存器並RET。失敗前checkpoint631有40個符合畫面與死亡條件的角色，第31..35項同時解釋四個損壞暫存器及0x1666E8返回位址。五項還原同一畫面基底0x14F018，原生heap事件亦有此153216-byte配置。原始writer／consumer已證實，崩潰關聯列強推論，未宣稱取得實際RET trace。

READY修正只改章測試輸入：第11回合後正常方向鍵移游標至(0,0)，再一次HP-only清敵及正常END。沿相同DP1000槽、seed4及T2／4／7／10流程，不改CPU、原版死亡規則或正式重製。原版r2正在執行，通過前章台帳維持todo。重製端新增對應mark，含最後一個方向鍵重播與原版游標／鏡頭四座標核對，不靠座標注入。證據、規格與命令沿[event54主契約](docs/data/ida/fd2_turn_event54_20261015.json)，[視野稽核工具](tools/fd2_clear_viewport_audit.py)可重算。破壞EBP的反例拒收。

來源准入正反例通過，兩個shell入口語法通過；TRACE_TAIL=2在oracle啟動前拒收，實際診斷runner記錄enabled=true。來源准入首輪未掛git worktree共同.git目錄而失敗，補唯讀掛載後用相同容器重跑通過，未改准入清冊。Capstone首次多傳python3造成entrypoint用法錯誤，改用原Dockerfile既有入口後乾淨重跑；均屬環境／命令問題。

原版r2於第6回合seq303被SIGTERM終止，exit143、oracle.log為空，沒有原版錯誤收據。前301份完整JSON bytes與c1相同，沒有忽略任何欄位。改由Docker detached容器自身的2040秒timeout管理生命週期，以相同來源／槽／計畫重跑r3；保留r2，分類環境中止，不再校準DP。第23章以最終重播程式回歸，行為、節點、交易與完整SAV通過。

等待原版期間先核對下一章：攻略第25章第6回合援軍對應全域event56，固定表入口sub_35487，IDA LE linear35487..354DD共86 bytes。原始呼叫依序為pan(6,40)、spawn group1、ACTING74、文字索引5的對白、跳至134E4重設pose。1366A的既有語意索引已證實為ACTING播放器，不能誤當單位聚焦或把74當角色槽。event55是另一條炎龍劍事件，不混入本回合援軍。此處只準備入口，依同步工具契約等第25章原版收據後才接完整正式劇本，剩餘未轉寫數量保持23。

原版r3已完成710份狀態、兩次成功DOS寫入、戰後四槽保存與退出列表，SAV SHA256 e67cf23b0c5f72e40742b69544cee572cfb155e01a75d7eff65bf3fbdc20ee95。T11改鏡頭前622份完整JSON與c1逐byte相同，原版來源與輸入雜湊在執行期間保持。清敵時實際可見候選只有3個，仍只寫60個敵軍HP，沒有改byte5。

第一輪重製重播在戰後被舊[28,86]前沿白名單拒收，已另列#208。677與685兩個原版快照分別含24C51／24CB2 caller及有效76筆陣列，依既有#145契約限制先寫READY修訂，再僅增加實測76，其他未知count仍拒收。重播再次通過，280筆AI全消費且零順序分岔。行為34點、節點36點、交易與完整SAV均通過；畫面未擷取，frames=false且required=false，不冒稱逐像素一致。

[正式收據](docs/data/ui-traces/parity-ch24.json)與台帳已由官方工具更新，第24章passed，合計21／30，另9章todo。19套件Go回歸通過；其後76槽binding資料修訂再通過正式章與相關Context／Frontier／Binding回歸。123項Python工具測試通過，27項既有Go／字串回歸及第23章完整流程／SAV回歸通過。匹配式反編譯保持暫停。

本批已關閉#142／#207／#208，#199成為下一個P0；正式保存名冊25筆、HP>0為11筆，下一章先驗正常選人並保留原來源。工作清單已由真正主機拉取／render／verify，21項未完成；最後核對18檔UID／GID1000、2262條本地連結零缺件、200條教訓與84個守衛通過。既有root-owned基線2811項保持，`.md`目錄0，本輪FD2容器全部退出。提交及真正遠端HEAD將回填已關閉Issue。

## 2026-10-15 第25章主線流程接續

目前狀態：第24章已passed，台帳21／30；第25章尚未驗收。沿f0c04db3及乾淨dosgolem2c5c104，主機Issue #199為P0。依原版對拍、規格閘門及文件職責路由載入技能與任務入口。使用者詢問必要時啟動matching，現階段先以原版收據定位差異；未啟動編譯試驗。

新輸入是第24章實際保存e67cf23b0c5f72e40742b69544cee572cfb155e01a75d7eff65bf3fbdc20ee95，複製到獨立state。名冊25筆，HP>0為11筆，零額外治療、升級或強化。攻略第25章第6回合己方結束援軍作旁證，依正常鍵盤推至T7後一次明示清敵；炎龍劍支線未列入此主線計畫。畫面advisory，流程三閘門與保存仍必要。

入口：[接續槽清冊](docs/data/parity-slots/ch25-manifest.json)、[主線計畫r1](docs/data/parity-plans/ch25-flow-cheat-r1.jsonl)。原版使用唯一tools/dosgolem_oracle_container.sh，來源與原版唯讀，輸出位於work/ch25-flow-20261015；不覆寫前章。

首輪原版至T5必存角色聖寇拉斯index16／identity26，HP104→0後返回標題，未觸發event56、未保存。保留555份狀態與失敗流程收據，不把正常敗北當執行器缺口。r2沿同SAV／seed，以每回合正常move_unit向(0,43)退避；T1退避前104份完整JSON bytes與r1相同。沒有追加DP或注入HP。

另以原版battle_start seq103建立唯讀前綴供重播診斷。重製原在story_ch25的339F5呼叫24B4D時要求g.st，無法消費LOADCH的故事載體，已開#209。沿既有3396A完整IDA匯出與24B4D／story adapter證據先寫READY，再沿materializeNativeStoryMapState／composeNativeStoryMapFrame組裝場景，輸入組裝完成即恢復g.st=nil。正常四次揭幕參數為20／20／20／60，未略過演出。相同前綴重播通過；18項相關Go回歸及場景成功／缺資產保持非戰鬥狀態通過。字串清冊SHA bba089ef90272e287c20260b037d3a26805aae7effc1a71fe4ee2ef1d6dd4ac3與review相同，不遷移四語。

第25章event56的完整86-byte靜態轉寫已READY並加入producer，尚未降階到正式劇本。既有35487在9份歷史編譯候選中5份完整匹配，C來源SHA相符；無新matching編譯。依規格等本章原版收據後才接正式consumer。唯一主契約為docs/data/ida/fd2_turn_event56_20261015.json，已掛00索引與58。

第二輪仍於T5因聖寇拉斯死亡返標題，沒有觸發event56。退避不是足以通關的結論，不再單靠調走位重跑。原始LOADCH紀錄證實其自帶兩瓶神聖之水raw槽2／3，沿既有type5、amount999、玩家目標與回復交易，先寫READY後補控制器及重播。第5／6回合正常自用，沒有改初始HP、增道具或鎖血。計畫完成閘門要求兩次item_use；125項Python回歸通過。

第三輪在T5實際完成正常物品使用：seq377→378，HP57→600、byte5=128、第二瓶補到槽2。原版無錯誤，控制器誤以為被用掉的原槽一定空掉而拒收。回查既有已證實sub_1B8E7與原版八格bytes，修正為完整左移補位及尾flag0x80、尾item byte保留；第二次仍選當時槽2。保留r3及其控制器SHA，r4沿相同容器／槽／seed乾淨重跑，不計成DP校準或遊戲缺陷。修正後126項Python測試通過，含取自r3控制鏈／八格內容的正常消耗正例。

進度訊息勘誤：先前所稱退避後第4回合仍600 HP，應為第2回合當下值；第4回合起手原版為199 HP，第5回合起手57 HP。以checkpoint及主契約為準，不用舊訊息推定存活。
原版r4以正常鍵盤完成T6 event56、T7、戰後及酒店保存／離開，共601份checkpoint。T5／6正常使用原有神聖之水，HP57→600及6→600；T7正常移鏡頭後一次HP-only清敵35筆。沒有追加道具、強化或鎖HP。event56固定入口與完整五操作實測後，正式producer、scenario及canonical同步。

前綴揭幕修正後，另查到battle_ch25缺原生view及HUD繼承，開#210並依READY補資料與generator。正式AI首個敵軍施法者record21延後，造成12筆順序漂移；診斷找到ScoreNativeAI1598A的三個caller未傳既有1F183閘門，開#211後先寫READY再接線。相同原版r4完整Game r2消費240筆AI、零順序分歧，正常物品、T6演出及戰後隊伍／酒店都到達。這些缺口皆由既有證據定位，本輪沒有新matching編譯。使用者另允許必要時局部匹配，118與#198已明示界線，全面仍暫停。

完整Go初次只敗在舊測試仍期待event56單一spawn；改成核對完整七個lower動作並保持戰後名冊隔離範圍後，19套件乾淨重跑通過。132項Python回歸通過；第24章以最終Game重新重播，280筆AI零順序分岔，行為／節點／交易及整檔SAV相同。全域sync check通過，剩餘未轉寫22筆；字串inventory SHA仍為bba089ef90272e287c20260b037d3a26805aae7effc1a71fe4ee2ef1d6dd4ac3，未遷移四語。

第25章目前未通過。原版SAV SHA0d420b379a38906b208a5e2a88dd74617b1c9a763010205b5c864f0ff102a9cb，重製52984034e916259e2c4a45eafd0bb36f1835edabfd8867e007dee9ad38c1996b。全部明文只有identity26的+0x28..+0x30區間七byte與衍生檢查碼兩byte不同。既有IDA建構器未寫此區，11506整筆複製，本次原版runtime一路保持同值；用途及最初producer仍未知，不能宣稱指標或無用欄位。新#212追蹤，正式失敗收據docs/data/ui-traces/parity-ch25.json、主契約opaque_save_gap保存完整證據。工具先達READY再加--oracle-save，核對實檔SHA與checksum後揭露所有差異，未排除任何byte或放寬交易閘門。台帳第25章blocked，passed仍21／30。

已提出具體九位元組區間驗收例外供使用者決定；尚未收到答覆，完整保存門檻保持，不進下一章、不把oracle常數填進正式runtime。前三項consumer缺陷的驗收範圍與未知存檔題拆開，章驗收由#199／#212繼續追蹤。

提交前官方主機pull／render／verify：22條未完成，可能已完成0條；三項接線缺陷#209／#210／#211已結案，#212與父#199保持。台帳21 passed／1 blocked／8 todo一致，202條教訓及86個守衛通過。40個變更／新檔UID/GID1000；本輪文件2289條本地連結零缺件。全域另有9項缺件／解析誤判，均位於未改動歷史文件，不算本輪新增或產品缺陷。原始輸入及r4控制器／計畫雜湊保持；既有root-owned基線2811項、md目錄0、掃描錯誤0。本輪FD2容器全部退出。提交及真正遠端HEAD核對後回填已結案Issue。

## 2026-10-16 第25章有界保存驗收定案

使用者確認接受逐筆揭露第25章保存差異並繼續。命中原版決定性對拍、規格閘門及文件職責路由，載入既有入口、重製技能與必要reference；開工工作樹只有既有work未追蹤資料，沒有FD2容器。真正主機gh auth成功，官方pull為22項未完成。

先逐byte保留原parity-ch25.json為parity-ch25-flow-r2-save-failed.json，再更新111及event56主契約save_acceptance_policy為READY。正式比較器新增明示--save-policy ch25-opaque-20261016，只接受第25章同流程profile、輸入SAV、計畫、原版EXE／dosgolem、兩側實檔SHA及完整差異清單；七byte與衍生檢查碼兩byte全部保留，bytes_equal=false。未知用途不升格、不回填oracle常數，不改正式遊戲與存檔格式，其他章及預設strict保持。

相同原版r4唯讀，正式Game乾淨重播remake-approved-r3至酒店，240筆AI全消費、零順序分歧，SAV SHA仍為52984034e916259e2c4a45eafd0bb36f1835edabfd8867e007dee9ad38c1996b。使用預設strict同pair仍failed；明示有界政策後行為／節點／交易通過。137項Python工具測試通過，涵蓋不同章／profile／來源／實檔SHA、新增／刪除／改值差異、缺保存writer等拒收；第24章流程與整檔SAV回歸仍passed。此輪沒有Go正式程式修改，未重跑原版或啟動matching編譯。

政策達CONFORMED，第25章按111例外登記PLAYER-E2，官方台帳set／verify更新為22／30。第25章slot_policy明示直接接續實際SAV、零追加升級／強化／event-state改寫，祖先政策獨立保留。58、56、57與索引同步，歷史failed及其SHA保持；#212以使用者接受有界證據限制結案，不宣稱未知語意已解或保存bytes已修成相同。下一個P0為#199第26章。

最終相關回歸141項通過，包含原137項工具與4項台帳測試。真實第25章缺少實檔報告反例及第24章錯套政策反例都交易failed，明示政策沒有一般化為遮罩。原failed收據與前一提交逐byte相同，正式收據來源SHA與原版EXE保持。官方主機pull／render／verify為21條未完成、可能已完成0條；台帳22 passed／8 todo／0 blocked，203條教訓及87個守衛通過。本輪16個變更／新檔UID/GID1000，2294條本地連結零缺件；全域既有9項缺件／解析誤判均在未改動歷史文件。root-owned既有基線2811項、md目錄0、掃描錯誤0，FD2容器全部退出。提交及真正遠端HEAD核對後回填#212及父#199。

## 2026-10-16 第26章原版主線接續

開工1f8fa562，工作樹只有既有work未追蹤資料，dosgolem專用工作樹2c5c104乾淨，沒有舊FD2容器。命中原版決定性對拍、規格閘門與文件職責，載入既有入口及重製技能。主機gh auth成功，官方pull為21項未完成，#199接續第26章。

沿第25章原版實際SAV 0d420b379a38906b208a5e2a88dd74617b1c9a763010205b5c864f0ff102a9cb複製獨立覆蓋層，名冊27筆，不追加治療／升級／強化／道具。悠妮identity9現HP0，先由正常LOAD／選人核對本章必存角色條件；不能章內鎖HP。第25章保存例外不套用本章。

攻略未知的迴廊與原有控制列表對照：event57在敵方T2／4／6／8／10／12／15／16／17，group由當前53BEF round取得。沿既有IDA9.4完整匯出保存sub_354DD 33bytes及353C4..353DA共用尾段；完整操作為spawn當前round group、pan(9,0)、delay400。現行九列只有spawn，先建立READY契約，等原版收據才接正式完整lower，不重開matching編譯。

95個正常鍵盤控制命令，預估19～29分鐘、2核；原版r1由唯一tools/dosgolem_oracle_container.sh執行，原始素材及dosgolem唯讀，獨立state-r1／original-r1，native近堆、40bn步預算、外層2040秒timeout、記錄AI及event57入口。預定T18後正常移鏡頭再一次HP-only清敵、戰後及酒店保存；台帳仍todo。入口docs/data/ida/fd2_turn_event57_20261016.json、docs/data/parity-slots/ch26-manifest.json及docs/data/parity-plans/ch26-flow-cheat-r1.jsonl，均已掛00索引。

第26章r1於cp78取消：原版LOAD回城鎮，獨立整備前綴誤入教會狀態窗。此為控制計畫問題，未進戰鬥，不列產品缺陷。接續原版槽悠妮HP0，復活費4000高於金幣2000。r2沿已授權建構政策，使用第24章slot-c1完整隊伍基底，以`fd2-chapter-slot -target 25 -levels-per-chapter 6 -seed 4 -boost-base-dp 1000`建槽；累計政策及所有27筆狀態見清冊。正常城鎮選人先選14筆，再以down／right選末位亞奇梅吉。輸入與規格來源已登記索引，保存仍嚴格比較。

第26章r2在戰前探訪取消：`town_probe`固定settle後於轉場狀態反覆送esc，建築選擇偏離，未進戰鬥。r3保留同一建構槽與正常選人輸入，移除戰前探訪，先驗九波與酒店保存；五處探訪與買賣未抽樣，不冒稱完成。預估104個邏輯序列，2核上限，單執行緒原版實測約每秒160萬指令，外層90分鐘有界。

第26章原版r3正常至T6。診斷前綴到T5的Game可執行且176筆AI零順序分歧，但比較器揭露站位／HP不同，不能稱通過。#213定位為缺少raw25的required[9,29]及依序front[9,29]資料；沿既有318AD原始caller與31DBE／321C8閉合helper，READY契約先記於第26章主證據preparation_contract，不重新深挖helper。

原版r3在T13正常敵方範圍法術後悠妮HP272→0，回標題；六波已觀測，零清敵，後三波及戰後／保存未驗，不將敗北當完成。依使用者已明示作弊通關方向，本章採章前一次HP／MaxHP+30000，沿既有signed word欄位，不章內鎖HP、不改正式規則。預檢原子性、保持傷勢與HP0、未指定時完全不改的READY契約先記主證據hp_slot_policy，準備受版控建槽旗標。

章前HP政策工具Go回歸通過，slot-r4由同章slot-r2以`-boost-only -target 25 -boost-max-hp 30000 -seed 4`建構，27筆，零handler／升級，SHA f04c6761f4dd4aa0a1047dc5d18fbfdbcfdfac94cccf0fd62169ca9895a645ff。索爾30420、悠妮30272、亞奇梅吉30416 HP；所有數值低於signed word上限。r4控制命令與r3相同，只有來源槽與政策標示不同，原版source仍乾淨2c5c104，90分鐘有界。

r3敗北診斷Game消費601筆AI零分歧，一般比較behavior=true，但實際end為cursor／T14，不能證明敗北終態。最初誤設只適用第15章的DEFEAT_TAIL測試旗標，已按同容器命令移除並乾淨重跑，失敗log保留，不列產品缺陷。新#214實際缺陷為第26章缺20B3C結果列及原生敗北返回設定。固定EXE54bytes、章25結果表51B7D及既有helper契約已核對，READY先存defeat_contract；只補資料，不重解一般敗北呈現或新增matching。

#214補兩列原生結果與敗北返回資料，移除未證retreat_ch26路由及孤兒節點，保留既有flag。同r3正常Game消費601筆AI，零順序分歧；record2悠妮由272承302傷害至0、raw+5 bit0成立，native_round13觸發record2_inactive，完整原生敗北／publisher／開場至title menu selection0，保存未變。有限工具只新增章26的已觀測終態准入，其他章仍拒收。原版9／36tick影像未逐像素比，不提升完整章；r4HP流程繼續。

第26章正式r4三項流程閘門通過，913筆AI全消費、零順序分歧；SAV 22987 bytes整檔相同，SHA 89a8d7a93f8baae11e2315f9d06c9a979ef5d6d608ee1558ec42ffb198cdbbd8。首次全套Go有三項失敗：覆蓋率舊計數、文字審查舊SHA，以及我加入全域FD2_MUTE造成command17缺sample反例失效。來源稽核先重生為117故事／過場、57handler、51fallback及26retreat，再同步斷言。文字清冊重綁98筆處置不變，四語5094筆全以既有id配對、new=0；canonical-root最初誤指父目錄，按工具規格改story後乾淨重跑。三個針對性Go測試通過，完整同映像回歸移除全域靜音重跑。

#216渲染收尾勘誤：原README仍有舊動態標記，首次渲染改寫了區塊，保留SHA斷言拒收該結果。現以穩定現況連結取代動態表，README區塊外所有文字對HEAD逐字相同，文化保存、圖片與技術導覽未變。進度頁的動態表改由工具產生，前言不再無條件稱四gate通過；CLI重跑與--check保持README SHA。

最終19套件Go回歸通過；第25章用目前Game乾淨重播，240筆AI零分歧，既有有界保存政策仍passed。另審查canonical時發現移除一節點使預設初次匯入規則重編121個ID；#217新增選用既有canonical來源，依legacy_key保留其餘318個ID與引用，不改schema或預設初次匯入。刪除／插入、錯文件／重複ID／錯kind回歸及官方CLI精確重生通過。9項相關Python工具及三相關Go套件通過；最終第26章再重播913筆AI零分歧，整份收據bytes與正式收據相同。README另修正驗收政策段落，不再無條件稱四項全過，其餘文化內容保持。下一章27先核對LOAD owner與已閉合318AD的raw26必出／前置資料，不沿用第26章選人鍵或重解helper。

最後渲染審查補正：非必驗畫面明示非必驗，未比像素不顯示0px，第25章保存明示差異已揭露並接受；10項渲染工具測試通過，相關Python合計85項。官方主機pull／render／verify為21條未完成、可能已完成0條，#213～#217已結案，#199接續第27章。台帳23 passed／7 todo／0 blocked，205條教訓及88個守衛通過。本輪49個變更／新檔UID/GID1000，2750條本地連結零新增缺件；全域既有9項缺件／解析誤判均在未改動歷史文件。root-owned既有基線2811項、md目錄0、掃描錯誤0。README僅更新進度入口與驗收政策，其餘文化／圖片／導覽逐byte保持；原EXE及最終正式收據雜湊保持。最終台帳verify初次誤掛唯讀而拒絕寫回，改為明示台帳可寫後同命令乾淨重跑一致，非產品缺陷。本輪FD2容器全部退出，其他專案容器不動。提交並核對真正遠端HEAD後回填各結案Issue。

## 2026-10-16 第27章主線接續

開工bf4bdab8，追蹤檔乾淨，只有既有work資料，FD2專用dosgolem2c5c104乾淨。路由命中原版對拍／規格閘門／文件職責，主機gh auth成功、官方pull21項未完成。攻略第27章及新浪旁證確認階梯增援、隊長死亡增援與天空之鑰後續，正式 ABI 沿既有IDA證據。直接複製第26章原版實際SAV，零追加升級／強化／事件改寫；祖先HP+30000及AP／DP／DX仍明示。正常LOAD與選人、索爾向y39移動觸發62→63，T9後一次清敵再驗64及戰後記錄，原始素材唯讀。必出缺陷只補raw26既有318AD caller，required9／front9的READY先於資料實作，不重解helper或啟動matching編譯。

#218必出／前置資料已依READY補入生成器及正式／canonical，318個ID保持。缺9拒收、單9通過與完整名冊前置測試通過；66筆IDA指令對固定原始file bytes相同，Capstone5.0.3對齊。原版r1漏設AI／成長trace，於開戰前取消，不作正式收據；r2同輸入同槽，補0x13A9F／1E54A後乾淨重跑。

r1取消邊界補正：最後cp74已載入戰場並處揭幕對白，尚無敵方AI；先前『開戰前』應讀為首次敵方行動前，診斷原始資料不改。r2補齊唯讀決策追蹤並保持同輸入，50分鐘、30bn指令上限。

第27章原版r2完整至T9及非城鎮保存返回選人，424筆AI入口、event62入口13次與event63一次；一次HP-only清敵59筆。
勘誤：最初計畫預期清敵觸發64，實際沒有358EA；64與隊長死亡增援仍未驗。原版SAV22987 bytes，SHA003d0394e1c5d9bb609949830450b4658284501f2d1e94456cbf96ed69b11663。

揭幕舊binding混入死亡對白且漏pan／ACT81／三轉場，依原caller先補READY再接原生19句；來源實際80筆、初始runtime63，撤回loader前錯置context。
故事inventory predicate接回已物化raw roster首16筆，不借持續隊伍。#220接250CC共同前綴與持鑰匙成功臂23句、ACT82..84、完整typed色盤／轉場及清理。
兩層direct patch25116白名單、透明inventory gate視圖交接及indexed palette consumer依READY修正。
先前『鏡頭歸零』只是症狀：真正根因為節點交接捕獲過時battle carrier，未保持當前story view；無原版歸零writer。

Game r7走完整戰後且SAV相同，但正式比較揭露90筆AI順序分歧，沒有因保存相同就標完成。
#222既有IDA直接反證1567E高command第一階段應用mode1／inner1，149F8 selector0應收raw+6非零；舊consumer誤收同陣營造成互傷。
READY先於實作，11／56及舊map0 score8靜態斷言同步勘誤；六項scorer回歸通過。
同原版r2重製r8全424筆零分歧，正式行為／節點／交易通過，完整SAV嚴格相同；frames0張且advisory，白／黑VGA memset尚未轉寫如實列限制。
415筆IDA9.4來源指令與固定EXE原bytes、Capstone5.0.3邊界一致；沒有新matching編譯或硬體RE。

故事稽核重生118節點／9直接腳本／59handler／50fallback，26舊撤退；既有318個canonical ID保持，只新增共同前綴節點。
字串重綁98筆處置保持，inventory SHA39cd7d84193c274946b5f30399f8a66a34e0b6151dd9d044a9b5faaf2f2938cd；四語各5094筆，4860依ID、234依signature，new0。
完整Go與前章正常Game回歸收尾中，台帳及GitHub最後核對後提交。下一步第28章，先核對非城鎮LOAD owner與已閉合必出／前置資料。

最終19套件Go全部通過，cmd/fd2 283.154秒；81項Python回歸零失敗／略過。第26章913筆AI零分歧、三流程閘門與整檔SAV保持。最後indexed色盤發布清除legacy白屏旗標的三項測試通過，第27章r9重播424筆零分歧，正式收據與r8逐byte相同。最後Go修改的文字清冊SHA保持，四語文字／role／variables全與HEAD多重集合相同。官方canonical importer逐byte重生，318個既有ID保留，只新增1個節點。

真正主機gh已更新#199至第28章，#218～#222結案，#33保持13筆未轉寫與未抽樣64限制；官方pull／render／verify為21條未完成、可能已完成0。206條教訓及89守衛通過。README SHA保持6fd6f74bc84a17d4f05bd31cd624b7d91306a3a756bd62b2405516dcbc38dd70。首次衛生檢查發現2812項root-owned，定位為本輪oracle掛載時建立的空dos-cache/gomodcache；只修該空目錄至1000:1000，未遞迴變更。後續所有mount來源先確認存在。歷史map0 item79分數與phase96／8已在11／56及時間序列文件追加勘誤。

最終衛生固定工作樹後乾淨通過：51個變更／新檔UID/GID1000，本輪3282條本地連結零新增缺件；全域9個既有缺件／解析誤判保持，其中交接檔兩項舊封包路徑逐字存在HEAD未變前綴。root-owned回到既有2811項，md目錄0、掃描錯誤0。前一掃描與git暫存並行且錯誤明細未記錄，固定後相同掃描零錯誤，不列產品缺陷。FD2本輪容器全部退出，其他專案容器不動。暫存51檔無原始EXE／SAV／資料庫或私人素材，正式收據strict保持。提交推送並核對真正遠端HEAD後回填#218～#222及父#199。

## 2026-10-16 第27章隊長增援再次抽樣

使用者補充鎖定索爾與敵方隊長血量直到增援抵達，並要求重製也有增援。先回第27章，暫不開第28章。初始工作樹只含work，主機gh登入成功、官方pull21項。路由載入原版對拍、規格閘門及文件職責；既有358EA直接指令顯示第二名死亡生三組、第三名死亡清場。採有界E1修改路徑，既有正式收據不替換。新dosgolem分支從目前main81c85be建立，補回先前未合併的IRQ7兩提交，核心來源tuple與已驗基線一致。430及主契約resample_hp_lock_policy先達READY，再實作選擇性控制及雙側重播。

第27章增援r1已驗前波12筆與保護清敵56筆，但朝15,22的曼哈頓候選停於15,32。成本格確認y31中央三格不可通行，r2明示經x13繞行六回合，候選先以同成本格核對。r1控制465另因牆鐘期限未收到checkpoint而結束，最後trace仍完成三隊長AI入口，未判為引擎缺陷；驅動逾時改用單調時鐘並加回歸。

增援診斷追加：重製r1-fixed已實際追加前波12筆，262筆AI零順序分歧。首輪測試鎖血漏接持續角色NativeIdentity的既有+8相容契約，已按NativeItemPanelRecordForUnit修正，沒有改正式Game。原版r2於雙隊長攻擊前主動停止，因已證實29F72負差額普通傷害為0；依使用者既有作弊授權，用既有fd2-chapter-slot boost-only生成同章全名冊AP+1000槽，SHA256 0eb3de0a84fd2907f28d5112380511a2e05d13423b0b04a66db5b96c9b9836ea。解碼差異僅27筆baseAP／effectiveAP兩word與兩byte校驗碼，共110byte；r3兩側使用同槽，原正式SAV及章台帳維持。

第27章增援r3／Game r3完成：正常兩次物理死亡派送358EA，三路21筆確實追加，新增位置／HP／raw+8一致；270筆AI零分歧，索爾30420及第三隊長3200存活。獨立E1收據passed；正式24／30及parity-ch27維持。第一次驗證腳本誤對不同AP槽要求BOOT暫存器相同，已分離同輸入95邊界嚴格回歸與AP槽EBX完整差異；未知EBX語意未猜補。終點view差異另列#224。最終Go19套件282.332s、102控制器加4章槽Python、來源准入及90教訓guard通過。

## 2026-10-16 驗證錄影清理

依使用者要求節省空間，核對既有收據與實檔雜湊後刪除下列五段本機暫存錄影，共10548497 bytes。保留收據、日誌、必要截圖與受版控重跑腳本。歷史收據中的影片路徑代表當時產物，不代表目前仍保存。原版第一章來源錄影與成品推廣片保留；後續章對拍沿流程模式，不產生非必要錄影。

- `dist/private-promo/fd2-v011-palace-remake-complete-raw.mp4`：3090291 bytes，SHA-256 `eccadf9fe99b76c253b907de6b449b141729c79058a7aa7472ddbcf3bf1960d2`。
- `dist/promo/fd2-ch01-march-remake.mp4`：3581208 bytes，SHA-256 `efe8a83313771e5be2e5e0fade67c4b7b534364f3395a548cd1bd2a8b5941bcf`。
- `extracted/title_re/dragonfx_compare/remake_intro_8s.mp4`：1642688 bytes，SHA-256 `b12937910d13a6ca0a8e1baa50bdeeaba358fba3ed33c6fa0e6580fb32d0eba3`。
- `dist/promo/fd2-v0.1.0-live-gameplay.mp4`：1566124 bytes，SHA-256 `c3fb3c0c07ebb8f05613bae712eefe9a37227452b4da2f0a62d674430b832d10`。
- `dist/promo/fd2-v0.1.0-ch01-dosbox-vs-remake.mp4`：668186 bytes，SHA-256 `a5e9b98b167d135a7167def90a07aa1450a700db18e158f7180f65f432f96e3c`。

## 2026-10-16 第28章主線接續

開工a3c8669d，保留上一輪錄影清理兩檔；真正主機gh登入與官方pull22項成功。路由命中原版對拍、規格閘門與文件職責。直接複製第27章正式實際SAV，SHA003d0394不變，原來源唯讀；非城鎮LOAD與19人選擇單獨短探針。舊第28章E1聲稱64-slot包含未啟動增援，沿既有IDA及休眠控制列核對，不重解共用helper。

第28章原版正常LOAD短探針完成，34-slot與19人選擇、非城鎮入口已確認。既有222筆IDA指令對固定file bytes及Capstone邊界相同，READY先於補接；新增66／68／70完整轉寫與65／69原子區域啟動，必出前置沿既有318AD。Game同源短前綴的行為／節點／取樣交易通過，0 AI且未驗戰鬥／保存，不登記整章。首次exporter把RET納入對白區間，被完整指令核對拒收；修正半開區間後全58事件及全域回合同步通過，13筆固定回合待轉寫保持。完整原版r1已啟動，T9後才單次清敵；未錄影。

完整r1於T5被實際敵軍與地圖邊界堵住，exit8／cp312，未清敵。校準槽依既有boost-only一次AP+1000，54個word政策值與原檔110byte差異核對，其他byte保持；首次檢查誤將全檔checksum算進第4槽meta，依fd2save.CHECKSUM_OFFSET更正後通過，非產品缺陷。r2先要求完整64筆再清敵。Game r1另在第一移動揭露缺HUD入口，原版cp71與25E5E共同caller先達READY再補typed資料。聚焦晚期測試的原版locator／語系／首張穩定地圖缺件分開修正fixture，沒有放寬正式Game閘門。

#226定位增援candidate回滾PAN六全域，限定66／68／70保留當前view與135F2的selector0；已閉合135DD／35822另68筆原指令獨立核對。修正後Game完整重播r1至原版堵路邊界，74筆AI全消費且零順序分歧。晚期來源測試fixture改從既有JOIN constructor物化持續名冊，語系與穩定地圖用正式helper補齊；環境／fixture失敗不列產品缺陷。r2原版已經正常觸發67死亡及68第二波，等待第三波與戰後保存。

原版r2及Game完整流程完成：初始34筆，66／68／70三波到40／46／64，第三波T9；T13後seq1035一次清敵34個存活敵人，seq1064成功兩次DOS寫入，seq1073返回選人。305筆AI全消費且零順序分歧；正式比較行為／節點／交易通過，整份22987 bytes SAV含checksum相同，SHA df50919111cccb37a00215dea248a069617aa222545d0f4493ab4b00dc5b819f。未錄影，frames advisory不宣稱逐像素。台帳登記25／30，下一章29，不在本章工作單位開新章。字串重新綁定98筆處置不變；完整Go回歸執行中。

2026-10-17收尾：原版與Game完整收據保持passed，13個正常回合／齊備mark視圖四座標相同。首次完整Go額外提供FDOTHER／DATO，揭露六項先前受來源環境閘門略過的舊fixture缺語系；補正式locale，另補第22章持續名冊後的正式部署、第29章冷讀選人素材與非同步END回復畫面。修正僅測試，不改正式規則，也不提升第29章章驗收。137項Python在專用Capstone映像及明示FD2_EXE後全通過；先前兩次容器分別缺Capstone及原版路徑，失敗日誌保留且分類為環境。全域死亡58事件、回合／死亡程式／33地圖同步check通過，固定回合待轉寫13保持。新增PAN candidate教訓後208條、91 guard通過。

來源fixture r7六項全通過，另外保留原生EXP整數byte限制：舊77.5改為77，不放寬正式sync。完整Go以相同FDOTHER／DATO環境乾淨r2重跑，原始org_game另加唯讀掛載。GitHub #225／#226依完整章收據結案，#199排至第29章；官方pull22項，最終render／verify收尾中。最終清冊與runtime修訂後清冊逐byte相同，review SHA3c58908df73ae0a80f3443892e57d6b5e80f2a0f92177df356c83edc461ebc80，98筆處置保持。固定工作樹47檔擁有權1000:1000，新增失效連結0，歷史封包兩項缺件保持；全樹root-owned2811、md目錄0、掃描錯誤0，README與原EXE雜湊不變。

最終完整Go r2於相同來源環境乾淨通過19套件，cmd/fd2 277.634秒；137項Python無失敗／略過，六項舊source fixture乾淨通過。官方worklist verify為22條、14項人工判斷、可能已完成0。所有FD2本輪容器退出，README與原EXE保持；暫存47檔無SAV／EXE／私人素材。提交推送後以真正遠端HEAD核對，並回填#225／#226、#199及#33。

## 2026-10-17 第29章護送與防衛系統流程

開工633d2e43，tracked工作樹乾淨，主機gh登入與官方pull22項成功，FD2容器無殘留。路由命中復古重製、原版對拍與文件職責，沿既有work研究區。原始來源為第28章正式實際SAV df509191，直接複製至新input及state覆蓋層，不追加強化。攻略第29章說明護送悠妮到石碑、四回合增援及三頭龍；ABI與規則以既有map28 IDA主證據為準。先做正常非城鎮LOAD／19人選擇短探針，再安排實際地形路線；章台帳仍25／30，沒有新matching或錄影。

第29章完整r1在round2／checkpoint192主動停止，未清敵。route_cost把敵軍當全程障礙，遠方封鎖列讓Yunie全無候選；這是驗證驅動錯誤，不是原版規則或Game缺陷。修正READY：未來成本按地形，當回合可達集合仍保留占用及ZOC，再由原版Enter裁決。r1計畫與收據保留。

勘誤：r1問題在checkpoint192辨認；實際停止命令完成於checkpoint238，仍round2、未清敵。r2保留同一AP5000槽，地形潛勢改正、正常接戰助手0與18、最多30回合；r1未計整章。
戰後局部回歸先缺原素材環境，再揭露呈現器舊prefix限制。補齊FD2_FDOTHER／原資產包，READY六筆event72插入契約先行後修改正式validator；post-frontier-r3乾淨通過，21種完整組間插入與錯序原子拒收、93筆正式戰後、舊76筆保存冷讀保持。正常動態順序仍以完整章收據為準。

原版r2在2026-10-16 17:24:26 UTC、checkpoint597／round3以143退出，oracle.log無錯誤，工具工作階段收到SIGTERM。分類為環境中斷，未清敵、不計整章。r3沿相同r2凍結計畫與AP5000槽重跑，改由獨立Docker容器內timeout6000s及退出紀錄管理生命週期。
第72號死亡事件兩次35822與第74號同一135DD／10B4E，READY先加入72後補PAN回交；三項局部回歸全部通過，35C18最後range_one保持。#224不因此結案。

戰後首個fixture失敗的原因勘誤：startNativeUnitDeathPresent只在非靜音且g.sfx缺sample3時拒收。direct-entry fixture未初始化音效，也未設FD2_MUTE；明示靜音後可繼續。先前籠統記為素材環境缺件，不代表FD2_FDOTHER或素材包確實缺檔；音訊聆聽驗收未提升。

原版r3超過12分鐘後穩定運作，前598份checkpoint JSON與r2逐bytes相同，已排除生命周期調整改變狀態。round7／8 Yunie卡16,46，raw26=0；前方17,46／16,45守軍尚在，助手仍選後方近敵。主動停止，未清敵、不計整章。READY修正正常接戰策略：每助手依原生MV，先前方敵軍、再距離與索引，無前敵才退回最近其他敵軍；不改原版規則。

r4原版在第30回合觸發75及首波74，3781／round31／84筆以控制器exit19結束，沒有戰敗、清敵或保存。接管shell時容器已退出，未保留原程序；FD2.TMP為207360 bytes工作檔，非完整SAV。r5延至40輪，新增選用正常貼身攻擊解阻，固定同一AP5000來源與seed4，避免再次等後方守軍。
Game r4在正常石碑wait seq3569因raw8來源缺件拒收，已列#229；MaterializeNativePersistentPartyRecord有來源，applyPersistentStats漏帶raw8，READY後補單行，拒收不放寬。

2026-10-17第29章追加勘誤：r5正常護送於T21到碑，T25物化93筆，T26抽79後一次HP清敵28筆，T27仍留戰場。暫停driver並保留原程序查證，確認結果表基址51B19的raw28表項51B89指向20B72；先前誤把單一表項51B51當基址，已由117E7實際call運算元修正。既有IDA完整131-byte handler與Capstone34指令相同，沒有205BE全滅掃描；state18／19／20須由三龍death73／77／78正常派送，單純HP0不設旗標。r5保留failed並以143停止，不計整章、不寫SAV。#230與chapter_result_contract先達READY，才補typed初始code0、三旗標code2及record0／1敗北優先，string9沿正式文字索引lower。r6沿同一AP5000來源及seed4重跑，全部增援與79後正常攻擊三頭龍，不再清敵；原版戰後35BBA自行清場。

工具檢查補記：186項Python回歸在Capstone容器通過但Pillow測試略過1項，已在素材容器補跑35項收據測試全部通過。新正常三龍控制122項driver測試通過。新增純文字canonical guard首次因掃描器未含.txt而失敗；只補.txt支援後同一檢查211條／94guard乾淨通過，不改證據marker。canonical重匯入先被過嚴的零warning斷言及絕對來源路徑拒收，屬操作問題；保留既有extensions.legacy並沿正式相對來源重跑，319節點ID保持。

r6於工具審查初期停止，finish_ui誤加move_unit已改回engage，新增直接接戰收尾測試後123項全過；r7同一plan r4及AP5000來源重新凍結，driver SHA9298cf88不再修改。Game r5負例前兩輪signal killed保留；較大12GB診斷容器取得實際失敗：seq2753的END在76 pulse期間重入，nativeEndTurnRecovery與pulse互等，峰值8792117248 bytes、cgroup OOM kill0。依READY #231補正式Update／END與重播控制權閘，原有frame budget不變。修後同源能完成，但79在END早跑、隨後13536清位，AI多出90／91；#232先依既有1A30B及35EE6達READY，再把79移到清位後。受控事件seed逐筆取原版入口40262／3223，分別得到27345／58716，metadata揭露，不要求整段內部骰序。負例r6乾淨通過963筆AI、零順序差異；原版仍缺三龍死亡旗標，不計整章。局部新fixture首輪跑入缺完整AI前置的mode2拒收，改成只驗本fixture的清位／pair交接；正式AI順序由同源長程重播驗證。

最後正式Go修正後再次重綁字串盤點，98筆候選與各處置內容／數量保持。原版r7實測單核及約400MiB，先前12GiB診斷容器已退出；改用兩CPU、8GiB、Go軟上限4GiB有界並行完整回歸，總容器上限維持12GiB，避免等原版重跑結束才發現舊fixture問題。

第29章原版r7完成：seq3342／T30，349動作／3343 checkpoint；三龍91／90／92依序於seq2986／3068／3258正常死亡，最後同seq寫20B93並進2548C。章內零state_injections、零HP鎖與零force-enemy-clear。post追加group9至94，record_question3323、正常保存3333有兩次成功DOS寫入，最後返回選人。首次Game完整1065 AI零分歧與整檔SAV相同，但3271多出勝利確認而節點失敗；#233先依main25E02／25E23達READY，再補可編輯auto-advance。auto-r2 production已進正確post，重播工具漏既有post分支而未消費劇情；修正由battle_start的正式on_win承接，沒有放寬gate或縮短演出。auto-r3乾淨通過349行為／351節點與strict保存，22,987 bytes含checksum零差異，SHA33c3d74deec1be3ca87b3754a26f2d5ee03799918e237958017be7518b164a6f。章台帳登記26／30，下一章30，本批不開新章。

完整Go r2十九套件已通過；最後auto-advance修正後完整r3另跑。191項Python零失敗／略過，回合與死亡同步check通過且固定回合待轉寫仍13，214條教訓／97guard通過。新增結果／旗標／phase指令另141筆對原始bytes及Capstone邊界相同，不與先前重疊區間相加作覆蓋率。隔離生成器scenario及battle節點逐值相同；最後盤點與上次清冊逐byte相同，98筆處置保持。沒有新matching編譯或錄影。

最後auto-advance與重播交接修正後，完整Go r3在相同FDOTHER／DATO環境乾淨通過19套件，cmd/fd2 205.181秒、exit0、峰值4468736000 bytes、OOM kill0。隔離生成器再次核對最終第29章scenario與battle節點逐值相同，preparation必出／前置欄位相同；正式執行來源雜湊全部相符。正式章收據與完整SAV一致結果保持。

生成器收尾檢查勘誤：新增整個preparation相等的斷言過廣，正式節點另含既有party_limit19，骨架未產生該欄位。未改正式來源；核對限於本輪必出／前置欄位，scenario與battle仍逐值完整相同。

GitHub #227～#233已依完整章與最後回歸結案，#199接第30章。官方pull／render／verify為22項、14項人工判斷、可能已完成0，標籤相符。53個異動檔擁有權1000:1000，66份canonical輸出雜湊相符，9個新證據／計畫入口與37條新增本地連結均有效。README、原EXE、父槽、章前來源、最終SAV與凍結driver雜湊保持；全樹歷史root-owned2811、md目錄0、掃描錯誤0。本輪無錄影、FD2容器無殘留。工作產物留在work且不加入版控。
