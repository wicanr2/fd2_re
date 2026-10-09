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
