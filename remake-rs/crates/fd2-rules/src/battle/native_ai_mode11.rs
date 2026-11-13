//! Go 來源：`remake/internal/battle/native_ai_mode11.go`（92 行）。

use super::*;

/// NativeAIMode11Transaction is the raw two-stage branch after 0x13A9F has
/// selected mode 11.  The fields are address-level routes only; they are not
/// command, attack, spell or movement names.
/// NativeAIMode11Route keeps the mode-11 branch separate from the 0x14ef0
/// tail enum: 0x14121 is a pre-tail movement/recovery route, not a 0x14ef0
/// consumer.
///
/// Go：`native_ai_mode11.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeAIMode11Route(pub u8);

/// Go 原式：`iota`
pub const NATIVE_AI_MODE11_NO_FIRST: NativeAIMode11Route = NativeAIMode11Route(0);

pub const NATIVE_AI_MODE11_CALL15311: NativeAIMode11Route = NativeAIMode11Route(1);

pub const NATIVE_AI_MODE11_CALL14121: NativeAIMode11Route = NativeAIMode11Route(2);

pub const NATIVE_AI_MODE11_CALL1548_E: NativeAIMode11Route = NativeAIMode11Route(3);

/// Go：`native_ai_mode11.go:20`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAIMode11Transaction {
    /// 0x15311 when raw [0x53C23] >= 6, otherwise none
    pub first_route: NativeAIMode11Route,
    /// 0x1548E when raw [0x53C4F] >= 6, otherwise 0x14121
    pub second_route: NativeAIMode11Route,
}

/// NativeAIMode11Stage is one address-level stage in the mode-11 transaction.
/// Ordinal is the native call order, not a gameplay action number.  The
/// callback owner must supply the complete raw candidate/presentation contract
/// for the route before mutating battle state.
///
/// Go：`native_ai_mode11.go:29`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAIMode11Stage {
    pub ordinal: i64,
    pub route: NativeAIMode11Route,
}

impl NativeAIMode11Transaction {
    /// Stages returns the exact dispatcher order after 0x1598A and 0x14237 have
    /// produced their raw scores.  A missing first route is intentionally omitted;
    /// the second route is still unconditional in the original dispatcher.
    ///
    /// Go：`native_ai_mode11.go:37`，`func (t NativeAIMode11Transaction) Stages() ([]NativeAIMode11Stage, error)`
    pub fn stages(self) -> Result<Vec<NativeAIMode11Stage>> {
        if self.second_route != NATIVE_AI_MODE11_CALL14121 && self.second_route != NATIVE_AI_MODE11_CALL1548_E {
            return Err(battle_err!("native AI mode 11 second route {:#x} is invalid", self.second_route.0));
        }
        let mut stages = Vec::with_capacity(2);
        if self.first_route != NATIVE_AI_MODE11_NO_FIRST {
            if self.first_route != NATIVE_AI_MODE11_CALL15311 {
                return Err(battle_err!("native AI mode 11 first route {:#x} is invalid", self.first_route.0));
            }
            stages.push(NativeAIMode11Stage { ordinal: 1, route: self.first_route });
        }
        stages.push(NativeAIMode11Stage { ordinal: 2, route: self.second_route });
        Ok(stages)
    }
}

/// ExecuteNativeAIMode11Transaction owns only the verified stage ordering. It
/// deliberately receives a caller-owned stage callback: command effects,
/// movement, indexed presentation and the 0x13FD4 fallback are not silently
/// substituted here. A callback failure stops before a later native stage.
///
/// Go：`native_ai_mode11.go:56`，`func ExecuteNativeAIMode11Transaction( t NativeAIMode11Transaction, execute func(NativeAIMode11Stage) error, ) error`
pub fn execute_native_ai_mode11_transaction(
    t: NativeAIMode11Transaction,
    execute: Option<&mut dyn FnMut(NativeAIMode11Stage) -> Result<()>>,
) -> Result<()> {
    let Some(execute) = execute else {
        return Err(battle_err!("native AI mode 11 stage executor is unavailable"));
    };
    let stages = t.stages()?;
    for stage in stages {
        if let Err(err) = execute(stage) {
            return Err(err.wrap(format!("native AI mode 11 stage {} route {:#x}: ", stage.ordinal, stage.route.0)));
        }
    }
    Ok(())
}

/// SelectNativeAIMode11Transaction preserves the direct mode-11 control flow:
/// 0x1598A has already produced [0x53C23], 0x15311 is called at signed >=6,
/// 0x14237 then runs unconditionally, and [0x53C4F] selects 0x1548E or
/// 0x14121.  It is deliberately a pure E0 adapter; it does not execute either
/// stage or claim that the two-stage presentation/commit owner is recovered.
///
/// Go：`native_ai_mode11.go:80`，`func SelectNativeAIMode11Transaction(scoreC23, scoreC4F int32, hasRawScoreC23, hasRawScoreC4F bool) (NativeAIMode11Transaction, error)`
pub fn select_native_ai_mode11_transaction(
    score_c23: i32,
    score_c4_f: i32,
    has_raw_score_c23: bool,
    has_raw_score_c4_f: bool,
) -> Result<NativeAIMode11Transaction> {
    if !has_raw_score_c23 || !has_raw_score_c4_f {
        return Err(battle_err!("native AI mode 11 raw scores are incomplete"));
    }
    let mut transaction =
        NativeAIMode11Transaction { first_route: NATIVE_AI_MODE11_NO_FIRST, second_route: NATIVE_AI_MODE11_CALL14121 };
    if score_c23 >= 6 {
        transaction.first_route = NATIVE_AI_MODE11_CALL15311;
    }
    if score_c4_f >= 6 {
        transaction.second_route = NATIVE_AI_MODE11_CALL1548_E;
    }
    Ok(transaction)
}

#[cfg(test)]
mod native_ai_mode11_test;
