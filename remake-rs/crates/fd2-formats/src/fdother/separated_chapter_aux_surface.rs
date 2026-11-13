//! 移植自 Go `separated_chapter_aux_surface.go`：FDOTHER #15／#55 原始不透明 320x200 底面的分離素材。

use super::go_compat::{nullable, nullable_struct};
use super::separated_surface::{read_document, read_paletted_surface};
use super::{
    FdotherError, NativeChapterAuxSurface, NativeChapterAuxSurfaceContract, Result, native_chapter_aux_surface_for,
};
use serde::Deserialize;
use std::path::Path;

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedChapterAuxSource {
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
struct SeparatedChapterAuxDocument {
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
    #[serde(deserialize_with = "nullable")]
    codec: String,
    #[serde(deserialize_with = "nullable")]
    width: i64,
    #[serde(deserialize_with = "nullable")]
    height: i64,
    #[serde(deserialize_with = "nullable")]
    frame: String,
    #[serde(deserialize_with = "nullable_struct")]
    source: SeparatedChapterAuxSource,
}

/// 載入 FDOTHER #55 的原始不透明 320x200 索引底面。不接受四模式 RLE 底面，也不退回封存。
pub fn load_separated_chapter_aux_surface(root: impl AsRef<Path>) -> Result<NativeChapterAuxSurface> {
    let contract = native_chapter_aux_surface_for(28).unwrap_or_default();
    load_separated_chapter_aux_surface_resource(root, &contract)
}

/// 依固定原始身分載入一張 `sub_10652` 輔助底面。
pub fn load_separated_chapter_aux_surface_resource(
    root: impl AsRef<Path>,
    contract: &NativeChapterAuxSurfaceContract,
) -> Result<NativeChapterAuxSurface> {
    let resource = contract.resource;
    let directory = root.as_ref().join(format!("FDOTHER_{resource:03}"));
    let document: SeparatedChapterAuxDocument = read_document(
        &directory.join("resource.json"),
        "fdother: separated chapter auxiliary metadata: ",
        "fdother: separated chapter auxiliary JSON: ",
    )?;
    if document.schema_version != 1
        || document.kind != "indexed_surface"
        || document.asset_id != format!("surface/FDOTHER_{resource:03}")
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.codec != "raw_indexed_opaque"
        || document.width != 320
        || document.height != 200
        || document.frame != "frame.png"
        || document.source.file != "FDOTHER.DAT"
        || document.source.resource != resource
        || document.source.size != 3382481
        || document.source.md5 != "22f56e5027edc7c766ad34ca4e5aca93"
        || document.source.sha256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
        || document.source.raw_size != 64004
        || document.source.raw_md5 != contract.raw_md5
        || document.source.raw_sha256 != contract.raw_sha256
    {
        return Err(FdotherError::fixed("fdother: separated chapter auxiliary contract mismatch"));
    }
    let pixels = read_paletted_surface(&directory.join(&document.frame), 320, 200)?;
    Ok(NativeChapterAuxSurface { pixels })
}
