//! 移植自 Go `native_unit_mode_branch.go`。

/// 只保存 `0x13a9f` 已閉合模式的原始呼叫／寫入順序。名稱刻意使用位址，不把函式猜成
/// 狂暴、護衛或其他玩法語意。數值與 Go 版 `iota` 順序相同（`as u8` 即 Go 的值）。
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
#[repr(u8)]
pub enum NativeUnitModeAction {
    Call14EF0,
    Call14237,
    Call1598A,
    Call15311,
    Call1548E,
    Call14121,
    Call13FD4,
    Call13E9C,
    Call12C60,
    Call12D7B,
    Call14B78,
    Call15DF3,
    Call1BB8C,
    Call25B45,
    Call12263,
    Call32975,
    Write51A83Zero,
    WriteUnit31And32,
    Write53AD5One,
    WriteUnit34Mode7,
}

use NativeUnitModeAction::*;

fn native_mode0_fallback_actions(
    blocked_search_succeeded: bool,
    nearest_coordinate_search_succeeded: bool,
) -> Vec<NativeUnitModeAction> {
    let mut actions = vec![Call14121];
    if blocked_search_succeeded {
        return actions;
    }
    actions.push(Call13E9C);
    if !nearest_coordinate_search_succeeded {
        actions.push(Call13FD4);
    }
    actions
}

/// 保存 mode 0 的原始備援順序。三個布林值都必須由呼叫端以相同 raw 記錄與 callee 回傳值
/// 提供；本函式不替它們命名成「攻擊」或「追擊」。`0x14121` 失敗後才呼叫 `0x13e9c`，後者
/// 回傳 0 才會經共用 `0x13c06` 呼叫 `0x13fd4`。
pub fn plan_native_unit_mode0(
    candidate_dispatch_succeeded: bool,
    blocked_search_succeeded: bool,
    nearest_coordinate_search_succeeded: bool,
) -> Vec<NativeUnitModeAction> {
    if candidate_dispatch_succeeded {
        return vec![Call14EF0];
    }
    let mut actions = vec![Call14EF0];
    actions.extend(native_mode0_fallback_actions(blocked_search_succeeded, nearest_coordinate_search_succeeded));
    actions
}

/// 保存 mode 1 的原始順序。它在 `0x14ef0` 失敗後只呼叫 `0x14121`，並透過同一個 `0x13c06`
/// 分支在回傳 0 時進入 `0x13fd4`；不會呼叫 mode 0 專用的 `0x13e9c`。
pub fn plan_native_unit_mode1(
    candidate_dispatch_succeeded: bool,
    blocked_search_succeeded: bool,
) -> Vec<NativeUnitModeAction> {
    if candidate_dispatch_succeeded {
        return vec![Call14EF0];
    }
    let mut actions = vec![Call14EF0, Call14121];
    if !blocked_search_succeeded {
        actions.push(Call13FD4);
    }
    actions
}

/// 保存 mode 3 的 raw lookup／移動鏈。`target_index` 使用 `0x12c60` 的原始回傳值，負值代表
/// 跳到 mode 0 的 `0x14121` 備援；找到索引後才會呼叫 `0x12d7b→0x14b78`，零回傳才列入
/// `0x13fd4`，最後寫入 `0x51a83=0`。這些輸入都不承載單位或劇情語意。
pub fn plan_native_unit_mode3(
    candidate_dispatch_succeeded: bool,
    target_index: i64,
    blocked_search_succeeded: bool,
    nearest_coordinate_search_succeeded: bool,
    movement_succeeded: bool,
) -> Vec<NativeUnitModeAction> {
    if candidate_dispatch_succeeded {
        return vec![Call14EF0];
    }
    let mut actions = vec![Call14EF0, Call12C60];
    if target_index < 0 {
        actions.extend(native_mode0_fallback_actions(blocked_search_succeeded, nearest_coordinate_search_succeeded));
        return actions;
    }
    actions.extend([Call12D7B, Call14B78]);
    if !movement_succeeded {
        actions.push(Call13FD4);
    }
    actions.push(Write51A83Zero);
    actions
}

/// 保存全域 `0x51a83` 清零後的 `0x12d7b→0x14b78` 鏈。`0x14b78` 回傳 0 才會進入共用 `0x13fd4`。
pub fn plan_native_unit_mode4(movement_succeeded: bool) -> Vec<NativeUnitModeAction> {
    let mut actions = vec![Write51A83Zero, Call12D7B, Call14B78];
    if !movement_succeeded {
        actions.push(Call13FD4);
    }
    actions
}

/// 保存 mode 5 的 raw caller inputs。`event_state_byte` 對應 `[0x53ad5+ebp]`；`event_record_type`
/// 對應 `0x53a55` entry 的 byte +0x53。其餘旗標只描述原始 callee 回傳或座標比較，不替欄位
/// 命名成玩法。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeUnitMode5Inputs {
    pub candidate_dispatch_succeeded: bool,
    pub event_state_byte: u8,
    pub event_lookup_succeeded: bool,
    pub movement_succeeded: bool,
    pub arrived_at_event_coordinate: bool,
    pub event_record_type: u8,
    pub blocked_search_succeeded: bool,
    pub nearest_coordinate_search_succeeded: bool,
}

/// 保存 mode 5 的事件資料分支與抵達後 raw writes。缺事件狀態或 lookup 失敗時沿用 mode 0 的
/// `0x14121` 備援；抵達座標後，只有 `event_record_type<2` 才寫入 +0x31/+0x32，且 type==0 才
/// 呼叫 `0x1bb8c`。
pub fn plan_native_unit_mode5(input: NativeUnitMode5Inputs) -> Vec<NativeUnitModeAction> {
    if input.candidate_dispatch_succeeded {
        return vec![Call14EF0];
    }
    let mut actions = vec![Call14EF0, Write51A83Zero, Call12D7B];
    if input.event_state_byte != 0 || !input.event_lookup_succeeded {
        actions.extend(native_mode0_fallback_actions(
            input.blocked_search_succeeded,
            input.nearest_coordinate_search_succeeded,
        ));
        return actions;
    }
    actions.extend([Call15DF3, Call14B78]);
    if !input.movement_succeeded {
        actions.push(Call13FD4);
    }
    if !input.arrived_at_event_coordinate {
        return actions;
    }
    if input.event_record_type < 2 {
        actions.push(WriteUnit31And32);
        if input.event_record_type == 0 {
            actions.push(Call1BB8C);
        }
    }
    actions.extend([Write53AD5One, Call25B45, Call12263, WriteUnit34Mode7]);
    actions
}

/// 保存 `0x51a83` 清零、移動與抵達後 `0x32975` 的順序。座標比對由 caller 以原始 record 提供；
/// 即使移動回傳 0 仍保留後續比對閘門，不自行宣稱抵達。
pub fn plan_native_unit_mode7(movement_succeeded: bool, arrived_at_destination: bool) -> Vec<NativeUnitModeAction> {
    let mut actions = vec![Write51A83Zero, Call12D7B, Call14B78];
    if !movement_succeeded {
        actions.push(Call13FD4);
    }
    if arrived_at_destination {
        actions.push(Call32975);
    }
    actions
}

/// 保存 `0x12c60` lookup 後的兩條 raw 路徑。lookup 失敗時回到一般 `0x14ef0` 起點；找到索引後
/// 直接沿用 `0x12d7b→0x14b78`，零回傳才列入 `0x13fd4`。
pub fn plan_native_unit_mode9(
    target_index: i64,
    general_candidate_dispatch_succeeded: bool,
    blocked_search_succeeded: bool,
    nearest_coordinate_search_succeeded: bool,
    movement_succeeded: bool,
) -> Vec<NativeUnitModeAction> {
    let mut actions = vec![Call12C60];
    if target_index < 0 {
        actions.extend(plan_native_unit_mode0(
            general_candidate_dispatch_succeeded,
            blocked_search_succeeded,
            nearest_coordinate_search_succeeded,
        ));
        return actions;
    }
    actions.extend([Call12D7B, Call14B78]);
    if !movement_succeeded {
        actions.push(Call13FD4);
    }
    actions
}

/// 只在 `0x14ef0` 失敗時重用 mode 4 的 raw 路徑。
pub fn plan_native_unit_mode10(
    candidate_dispatch_succeeded: bool,
    movement_succeeded: bool,
) -> Vec<NativeUnitModeAction> {
    if candidate_dispatch_succeeded {
        return vec![Call14EF0];
    }
    let mut actions = vec![Call14EF0];
    actions.extend(plan_native_unit_mode4(movement_succeeded));
    actions
}

/// 保存 mode 2 的控制流。`0x14ef0` 失敗後直接呼叫 `0x14237`，再經 `0x13b1e→0x13c06` 共用
/// 分支；`0x14237` 的函式尾端以 `xor eax,eax` 回傳 0，因此這條失敗路徑會進入 `0x13fd4`。
/// 它不會走 `0x13e9c`。這裡的布林值只表示呼叫端已提供的 `0x14ef0` 回傳狀態，不把任何
/// 位址命名成玩法語意。
pub fn plan_native_unit_mode2(candidate_dispatch_succeeded: bool) -> Vec<NativeUnitModeAction> {
    if candidate_dispatch_succeeded {
        return vec![Call14EF0];
    }
    vec![Call14EF0, Call14237, Call13FD4]
}

/// 保存 mode 11 的兩個獨立 signed score gates。`0x53c23 >= 6` 只控制 `0x15311`；其後仍會
/// 呼叫 `0x14237`。`0x53c4f >= 6` 控制 `0x1548e`，否則才嘗試 `0x14121／0x13fd4`。
pub fn plan_native_unit_mode11(
    command_score: i32,
    physical_priority: i32,
    blocked_search_succeeded: bool,
) -> Vec<NativeUnitModeAction> {
    let mut actions = vec![Call1598A];
    if command_score >= 6 {
        actions.push(Call15311);
    }
    actions.push(Call14237);
    if physical_priority >= 6 {
        actions.push(Call1548E);
        return actions;
    }
    actions.push(Call14121);
    if !blocked_search_succeeded {
        actions.push(Call13FD4);
    }
    actions
}

#[cfg(test)]
mod tests;
