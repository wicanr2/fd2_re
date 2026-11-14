//! Go 來源：`remake/cmd/fd2/locale.go`（230 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`locale.go:12`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct LocaleEntryContract {
    pub(crate) variables: Vec<String>,
    pub(crate) source_id: String,
}

/// Go：`locale.go:17`
///
/// Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。
pub(crate) static OFFICIAL_LOCALE_CONTRACT: LazyLock<RwLock<BTreeMap<String, LocaleEntryContract>>> =
    LazyLock::new(|| {
        RwLock::new(BTreeMap::from([
            (
                String::from("battle.status.expiry.attack"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0481") },
            ),
            (
                String::from("battle.status.expiry.defence"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0482") },
            ),
            (
                String::from("battle.status.expiry.speed"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0483") },
            ),
            (
                String::from("battle.status.expiry.poison"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0484") },
            ),
            (
                String::from("battle.status.expiry.paralysis"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0485") },
            ),
            (
                String::from("battle.status.expiry.silence"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0486") },
            ),
            (
                String::from("battle.attack.miss"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%s")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5973-c22"),
                },
            ),
            (
                String::from("battle.attack.hit"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%s"), String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5975-c25"),
                },
            ),
            (
                String::from("battle.attack.critical_suffix"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5977-c14"),
                },
            ),
            (
                String::from("battle.attack.exp_suffix"),
                LocaleEntryContract {
                    variables: vec![String::from("%.0f")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5980-c26"),
                },
            ),
            (
                String::from("battle.mp.insufficient"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5091-c14"),
                },
            ),
            (
                String::from("battle.command.unavailable"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5187-c12"),
                },
            ),
            (
                String::from("battle.attack.choose_target"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5193-c13"),
                },
            ),
            (
                String::from("battle.command.native_unavailable"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5201-c14"),
                },
            ),
            (
                String::from("battle.spell.sealed"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5211-c14"),
                },
            ),
            (
                String::from("battle.spell.none"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5218-c13"),
                },
            ),
            (
                String::from("battle.item.choose_slot"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5225-c13"),
                },
            ),
            (
                String::from("battle.item.empty_slot"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5065-c14"),
                },
            ),
            (
                String::from("battle.item.data_incomplete"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5070-c14"),
                },
            ),
            (
                String::from("battle.item.unusable"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5078-c14"),
                },
            ),
            (
                String::from("battle.item.effect_complete"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5085-c14"),
                },
            ),
            (
                String::from("battle.item.choose_target"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5092-c14"),
                },
            ),
            (
                String::from("battle.item.effect_unverified"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5095-c14"),
                },
            ),
            (
                String::from("battle.item.choose_destination"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l6278-c14"),
                },
            ),
            (
                String::from("battle.item.damage_presenting"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l6283-c14"),
                },
            ),
            (
                String::from("battle.item.recovery_complete"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l6285-c14"),
                },
            ),
            (
                String::from("battle.item.relocation_presenting"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l6297-c14"),
                },
            ),
            (
                String::from("battle.item.relocation_complete"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l6299-c14"),
                },
            ),
            (
                String::from("battle.item.title"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8825-c22"),
                },
            ),
            (
                String::from("battle.item.empty_label"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8830-c13"),
                },
            ),
            (
                String::from("battle.spell.target_prompt"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5168-c14"),
                },
            ),
            (
                String::from("battle.spell.blocked"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l6170-c12"),
                },
            ),
            (
                String::from("battle.unit.paralyzed"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l6145-c12"),
                },
            ),
            (
                String::from("battle.spell.result_damage"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%s"), String::from("%d"), String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l6194-c10"),
                },
            ),
            (
                String::from("battle.spell.result_heal"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%s"), String::from("%d"), String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l6194-c10"),
                },
            ),
            (
                String::from("battle.spell.miss_suffix"),
                LocaleEntryContract {
                    variables: vec![String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l6196-c13"),
                },
            ),
            (
                String::from("battle.treasure.gold"),
                LocaleEntryContract {
                    variables: vec![String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5396-c13"),
                },
            ),
            (
                String::from("battle.treasure.item"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5398-c13"),
                },
            ),
            (
                String::from("battle.treasure.inventory_full"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5402-c12"),
                },
            ),
            (
                String::from("battle.reward.item"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5457-c12"),
                },
            ),
            (
                String::from("battle.reward.item_full"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5459-c12"),
                },
            ),
            (
                String::from("battle.reward.gold"),
                LocaleEntryContract {
                    variables: vec![String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l5463-c11"),
                },
            ),
            (
                String::from("battle.spell.menu_title"),
                LocaleEntryContract {
                    variables: vec![String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8687-c22"),
                },
            ),
            (
                String::from("church.transfer.empty"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.native_church_input.l209-c12"),
                },
            ),
            (
                String::from("church.transfer.success"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.native_church_input.l271-c12"),
                },
            ),
            (
                String::from("church.revive.success"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l4711-c10"),
                },
            ),
            (
                String::from("church.class_change.success"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%s")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l4674-c10"),
                },
            ),
            (
                String::from("church.class_change.confirm_title"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9090-c13"),
                },
            ),
            (
                String::from("church.class_change.empty"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9094-c24"),
                },
            ),
            (
                String::from("church.class_change.target"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9100-c35"),
                },
            ),
            (
                String::from("common.yes"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9101-c36"),
                },
            ),
            (
                String::from("common.no"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9101-c41"),
                },
            ),
            (
                String::from("system.locale.changed"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("runtime.settings.locale.changed"),
                },
            ),
            (
                String::from("system.audio.changed"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("runtime.settings.audio.changed"),
                },
            ),
            (
                String::from("save.unsupported"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("runtime.save.unsupported") },
            ),
            (
                String::from("save.postbattle_blocked"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("runtime.save.postbattle_blocked"),
                },
            ),
            (
                String::from("save.battle_busy"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("runtime.save.battle_busy") },
            ),
            (
                String::from("save.battle_restore_failed"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("runtime.save.battle_restore_failed"),
                },
            ),
            (
                String::from("save.saved"),
                LocaleEntryContract {
                    variables: vec![String::from("%d"), String::from("%s")],
                    source_id: String::from("runtime.save.saved"),
                },
            ),
            (
                String::from("save.none"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("runtime.save.none") },
            ),
            (
                String::from("save.node_missing"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("runtime.save.node_missing"),
                },
            ),
            (
                String::from("save.loaded"),
                LocaleEntryContract {
                    variables: vec![String::from("%d"), String::from("%s")],
                    source_id: String::from("runtime.save.loaded"),
                },
            ),
            (
                String::from("shop.greeting.weapon"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0440") },
            ),
            (
                String::from("shop.greeting.item"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0501") },
            ),
            (
                String::from("shop.purchase.question.weapon"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%d")],
                    source_id: String::from("FDTXT_000/string_0439"),
                },
            ),
            (
                String::from("shop.purchase.question.item"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%d")],
                    source_id: String::from("FDTXT_000/string_0502"),
                },
            ),
            (
                String::from("shop.purchase.insufficient.weapon"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0438") },
            ),
            (
                String::from("shop.purchase.insufficient.item"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0504") },
            ),
            (
                String::from("shop.purchase.no_recipient.weapon"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0437") },
            ),
            (
                String::from("shop.purchase.no_recipient.item"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0505") },
            ),
            (
                String::from("shop.purchase.equip_question"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0507") },
            ),
            (
                String::from("shop.recipient.full"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("FDTXT_000/string_0506"),
                },
            ),
            (
                String::from("shop.transfer.destination_prompt"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0510") },
            ),
            (
                String::from("shop.transfer.empty_source"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("FDTXT_000/string_0511"),
                },
            ),
            (
                String::from("shop.transfer.source_prompt"),
                LocaleEntryContract { variables: Vec::new(), source_id: String::from("FDTXT_000/string_0512") },
            ),
            (
                String::from("shop.purchase.success"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l4468-c17"),
                },
            ),
            (
                String::from("shop.sell.success"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X"), String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l4506-c15"),
                },
            ),
            (
                String::from("shop.purchase.equip_prompt"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l4555-c15"),
                },
            ),
            (
                String::from("shop.recipient.none"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l4576-c13"),
                },
            ),
            (
                String::from("shop.purchase.equip_prompt.title"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8864-c24"),
                },
            ),
            (
                String::from("shop.purchase.equip_prompt.controls"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8865-c24"),
                },
            ),
            (
                String::from("shop.sell.item_title"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8874-c24"),
                },
            ),
            (
                String::from("shop.item.equipped_label"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8883-c14"),
                },
            ),
            (
                String::from("shop.sell.roster_title"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8891-c23"),
                },
            ),
            (
                String::from("shop.sell.inventory_count"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8898-c24"),
                },
            ),
            (
                String::from("shop.purchase.recipient_title"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8905-c23"),
                },
            ),
            (
                String::from("shop.purchase.recipient_inventory"),
                LocaleEntryContract {
                    variables: vec![String::from("%s"), String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8912-c24"),
                },
            ),
            (
                String::from("shop.panel.title_controls"),
                LocaleEntryContract {
                    variables: vec![String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8919-c22"),
                },
            ),
            (
                String::from("hotel.title.fallback"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8957-c12"),
                },
            ),
            (
                String::from("hotel.controls"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8971-c23"),
                },
            ),
            (
                String::from("preparation.title"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8981-c23"),
                },
            ),
            (
                String::from("preparation.save_prompt"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8985-c14"),
                },
            ),
            (
                String::from("preparation.deploy.controls"),
                LocaleEntryContract {
                    variables: vec![String::from("%d"), String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9006-c23"),
                },
            ),
            (
                String::from("preparation.unknown_character"),
                LocaleEntryContract {
                    variables: vec![String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9020-c13"),
                },
            ),
            (
                String::from("preparation.enter_confirm"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9032-c23"),
                },
            ),
            (
                String::from("preparation.save_hint"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9053-c22"),
                },
            ),
            (
                String::from("battle.result.win"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8410-c9"),
                },
            ),
            (
                String::from("battle.result.lose"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8413-c9"),
                },
            ),
            (
                String::from("battle.result.continue"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8418-c24"),
                },
            ),
            (
                String::from("postbattle.preparation.title"),
                LocaleEntryContract {
                    variables: vec![String::from("%s")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l8857-c12"),
                },
            ),
            (
                String::from("title.load.controls"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.title.l826-c22"),
                },
            ),
            (
                String::from("title.load.slot"),
                LocaleEntryContract {
                    variables: vec![String::from("%d")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.title.l834-c13"),
                },
            ),
            (
                String::from("save.slot.empty"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.title.l836-c14"),
                },
            ),
            (
                String::from("save.slot.present"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.title.l838-c14"),
                },
            ),
            (
                String::from("church.service.status"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9129-c21-i0"),
                },
            ),
            (
                String::from("church.service.transfer"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9129-c21-i1"),
                },
            ),
            (
                String::from("church.service.revive"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9129-c21-i2"),
                },
            ),
            (
                String::from("church.service.class_change"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9129-c21-i3"),
                },
            ),
            (
                String::from("church.controls"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9137-c22"),
                },
            ),
            (
                String::from("church.transfer.source"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9140-c13"),
                },
            ),
            (
                String::from("church.transfer.unequipped_item"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9142-c13"),
                },
            ),
            (
                String::from("church.transfer.destination"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9144-c13"),
                },
            ),
            (
                String::from("church.empty_selection"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9152-c24"),
                },
            ),
            (
                String::from("church.transfer.item_label"),
                LocaleEntryContract {
                    variables: vec![String::from("%02X")],
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9161-c14"),
                },
            ),
            (
                String::from("church.selection_controls"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9174-c22"),
                },
            ),
            (
                String::from("church.execute_controls"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9237-c22"),
                },
            ),
            (
                String::from("ending.title"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.main.l9248-c22"),
                },
            ),
            (
                String::from("ending.assets_missing"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.ending_preview.l231-c20"),
                },
            ),
            (
                String::from("ending.assets_missing_montage"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.ending_preview.l233-c20"),
                },
            ),
            (
                String::from("ending.continue_editable"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.ending_preview.l815-c26"),
                },
            ),
            (
                String::from("ending.terminal_missing"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.ending_preview.l817-c26"),
                },
            ),
            (
                String::from("ending.montage_requirements"),
                LocaleEntryContract {
                    variables: Vec::new(),
                    source_id: String::from("legacy.go.remake.cmd.fd2.ending_preview.l819-c26"),
                },
            ),
        ]))
    });

/// Go：`locale.go:142`，`func loadOfficialLocale(localeID string) (*localization.Catalog, error)`
pub(crate) fn load_official_locale(locale_id: &str) -> Result<Option<localization::Catalog>> {
    todo!("locale.go:loadOfficialLocale")
}

/// Go：`locale.go:174`，`func loadOfficialLocaleContent(localeID string) (*localization.ContentCatalog, error)`
pub(crate) fn load_official_locale_content(locale_id: &str) -> Result<Option<localization::ContentCatalog>> {
    todo!("locale.go:loadOfficialLocaleContent")
}

/// Go：`locale.go:178`，`func loadOfficialLocaleEntities(localeID string) (*localization.EntityCatalog, error)`
pub(crate) fn load_official_locale_entities(locale_id: &str) -> Result<Option<localization::EntityCatalog>> {
    todo!("locale.go:loadOfficialLocaleEntities")
}

impl Game {
    /// Go：`locale.go:182`，`func (g *Game) localizedStoryText(line campaign.Line) (string, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn localized_story_text(&self, line: &campaign::Line) -> Result<String> {
        todo!("locale.go:localizedStoryText")
    }

    /// localeMessage formats a required official key. Callers must stop their
    /// current UI transaction when ok is false; silently falling back to embedded
    /// Traditional Chinese would create a mixed-language official pack.
    ///
    /// Go：`locale.go:198`，`func (g *Game) localeMessage(key string, args ...any) (message string, ok bool)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn locale_message(&mut self, key: &str, args: &[Option<OwnerToken>]) -> (String, bool) {
        todo!("locale.go:localeMessage")
    }
}
