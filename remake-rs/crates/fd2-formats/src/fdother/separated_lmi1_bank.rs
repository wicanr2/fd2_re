//! 移植自 Go `separated_lmi1_bank.go`：原生地圖與戰鬥呈現端消費的完整 LMI1 圖庫。

use super::go_compat::{nullable, nullable_struct, nullable_struct_vec};
use super::separated_item_panel::load_item_panel_indexed_png;
use super::separated_surface::read_document;
use super::{FdotherError, LMI1Entry, Result};
use serde::Deserialize;
use std::path::Path;

/// Go `spawn_intro.go` 的 `NativeSpawnIntroFrameResource`／`NativeSpawnIntroPassCount`。
/// spawn_intro.go 不在本批移植範圍（第三組），這裡先以私有常數照抄同值；移植那個檔案時
/// 改用它的公開常數。
const NATIVE_SPAWN_INTRO_FRAME_RESOURCE: i64 = 9;
const NATIVE_SPAWN_INTRO_PASS_COUNT: usize = 12;

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedLMI1BankSource {
    #[serde(deserialize_with = "nullable")]
    file: String,
    #[serde(deserialize_with = "nullable")]
    resource: i64,
    #[serde(deserialize_with = "nullable")]
    size: i64,
    #[serde(deserialize_with = "nullable")]
    md5: String,
    #[serde(deserialize_with = "nullable")]
    sha256: String,
    #[serde(deserialize_with = "nullable")]
    raw_size: i64,
    #[serde(deserialize_with = "nullable")]
    raw_md5: String,
    #[serde(deserialize_with = "nullable")]
    raw_sha256: String,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedLMI1BankEntry {
    #[serde(deserialize_with = "nullable")]
    index: i64,
    #[serde(deserialize_with = "nullable")]
    codec: String,
    #[serde(deserialize_with = "nullable")]
    width: i64,
    #[serde(deserialize_with = "nullable")]
    height: i64,
    #[serde(deserialize_with = "nullable")]
    frame: String,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedLMI1BankDocument {
    #[serde(deserialize_with = "nullable")]
    schema_version: i64,
    #[serde(deserialize_with = "nullable")]
    kind: String,
    #[serde(deserialize_with = "nullable")]
    asset_id: String,
    #[serde(deserialize_with = "nullable")]
    status: String,
    #[serde(deserialize_with = "nullable")]
    evidence: String,
    #[serde(deserialize_with = "nullable_struct")]
    source: SeparatedLMI1BankSource,
    #[serde(deserialize_with = "nullable_struct_vec")]
    entries: Vec<SeparatedLMI1BankEntry>,
}

struct LMI1BankContract {
    asset_id: &'static str,
    label: &'static str,
    resource: i64,
    raw_size: i64,
    entry_count: i64,
    raw_md5: &'static str,
    raw_sha256: &'static str,
}

fn lmi1_bank_contract(resource: i64) -> Option<LMI1BankContract> {
    Some(match resource {
        5 => LMI1BankContract {
            asset_id: "ui/FDOTHER_005/lmi1_opaque",
            label: "FDOTHER #5 LMI1",
            resource: 5,
            raw_size: 44181,
            entry_count: 138,
            raw_md5: "646bfc938f3f459ed54a8af34909ee47",
            raw_sha256: "561eb8ca579b13a6f2bc1f36436ad3af7d8d0bf77eab25a4ab6411142a7eb118",
        },
        6 => LMI1BankContract {
            asset_id: "effects/FDOTHER_006/lmi1_opaque",
            label: "FDOTHER #6 LMI1",
            resource: 6,
            raw_size: 33415,
            entry_count: 230,
            raw_md5: "19e0e00500eb0ce41739f17615238cc3",
            raw_sha256: "47cc4f7136b553879f960e9902d223c9f7e4462d726fb7ea3124bf51c912f71c",
        },
        9 => LMI1BankContract {
            asset_id: "animation/FDOTHER_009/spawn_intro",
            label: "FDOTHER #9 spawn intro",
            resource: 9,
            raw_size: 3999,
            entry_count: 12,
            raw_md5: "a20e79010a333abaab060f9199eb6d7c",
            raw_sha256: "0eeaeb88320287e0c60c530c9d3a2592fea54378ebe5d5ac6a321e4fb1de7f0b",
        },
        _ => return None,
    })
}

/// 載入原生地圖與戰鬥呈現端消費的其中一個完整 LMI1 圖庫。絕不開啟 FDOTHER.DAT。
pub fn load_separated_lmi1_bank(root: impl AsRef<Path>, resource: i64) -> Result<Vec<LMI1Entry>> {
    let Some(contract) = lmi1_bank_contract(resource) else {
        return Err(FdotherError::fixed("fdother: unsupported separated LMI1 bank"));
    };
    let root = root.as_ref();
    let label = contract.label;
    let document: SeparatedLMI1BankDocument = read_document(
        &root.join("bank.json"),
        &format!("fdother: separated {label} metadata: "),
        &format!("fdother: separated {label} JSON: "),
    )?;
    if document.schema_version != 1
        || document.kind != "fdother_lmi1_bank"
        || document.asset_id != contract.asset_id
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.source.file != "FDOTHER.DAT"
        || document.source.resource != contract.resource
        || document.source.size != 3382481
        || document.source.md5 != "22f56e5027edc7c766ad34ca4e5aca93"
        || document.source.sha256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
        || document.source.raw_size != contract.raw_size
        || document.source.raw_md5 != contract.raw_md5
        || document.source.raw_sha256 != contract.raw_sha256
        || document.entries.len() as i64 != contract.entry_count
    {
        return Err(FdotherError::message(format!("fdother: separated {label} contract mismatch")));
    }
    let count = contract.entry_count;
    let mut entries = vec![LMI1Entry::default(); count as usize];
    let mut seen = vec![false; count as usize];
    for entry in &document.entries {
        let index = entry.index;
        if index < 0
            || index >= count
            || seen[index as usize]
            || entry.codec != "opaque_high_run"
            || entry.width <= 0
            || entry.height <= 0
            || entry.frame != format!("entry_{index:03}/frame.png")
        {
            return Err(FdotherError::message(format!("fdother: invalid separated {label} entry {index}")));
        }
        let pixels = load_item_panel_indexed_png(&root.join(&entry.frame), entry.width, entry.height)
            .map_err(|e| e.wrap(format!("fdother: separated {label} entry {index}: ")))?;
        entries[index as usize] = LMI1Entry { width: entry.width, height: entry.height, pixels };
        seen[index as usize] = true;
    }
    if let Some(index) = seen.iter().position(|&present| !present) {
        return Err(FdotherError::message(format!("fdother: separated {label} entry {index} is missing")));
    }
    if resource == NATIVE_SPAWN_INTRO_FRAME_RESOURCE {
        validate_native_spawn_intro_frames(&entries)?;
    }
    Ok(entries)
}

fn validate_native_spawn_intro_frames(entries: &[LMI1Entry]) -> Result<()> {
    const WANT: [(i64, i64); NATIVE_SPAWN_INTRO_PASS_COUNT] = [
        (42, 24),
        (44, 24),
        (45, 24),
        (46, 24),
        (46, 24),
        (66, 24),
        (69, 24),
        (70, 24),
        (69, 24),
        (63, 24),
        (65, 24),
        (66, 24),
    ];
    if entries.len() != WANT.len() {
        return Err(FdotherError::message(format!(
            "fdother: spawn-intro resource has {} entries, want {}",
            entries.len(),
            WANT.len()
        )));
    }
    for (index, (entry, &(w, h))) in entries.iter().zip(&WANT).enumerate() {
        if entry.width != w || entry.height != h || entry.pixels.len() as i64 != entry.width * entry.height {
            return Err(FdotherError::message(format!("fdother: spawn-intro entry {index} geometry differs")));
        }
    }
    Ok(())
}
