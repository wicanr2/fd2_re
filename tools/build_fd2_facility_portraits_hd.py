#!/usr/bin/env python3
"""Compile full hotel/church scenes and exact portrait crops from one master."""
from __future__ import annotations
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import PIL
from PIL import Image

def sha(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()

def png(image: Image.Image) -> bytes:
    stream=io.BytesIO()
    image.save(stream,format="PNG",compress_level=9)
    return stream.getvalue()

def compile_pairs(recipe: dict, source: Path, original: Path, out: Path) -> dict:
    if not Path("/.dockerenv").exists() or PIL.__version__!="11.3.0":
        raise ValueError("requires the maintained Pillow11.3.0 Docker image")
    if recipe["schema"]!="fd2-facility-scene-portrait-pair-hd6-v2" or [e["source_resource"] for e in recipe["entries"]]!=[13,14]:
        raise ValueError("unsupported paired facility source")
    prepared={};assets=[];proofs=[]
    for e in recipe["entries"]:
        resource=e["source_resource"];speaker,offset={13:(129,0x6ab),14:(131,0x576)}[resource]
        if e["speaker_id"]!=speaker or e["native_offset"]!=offset:
            raise ValueError("wrong original portrait binding")
        metadata=original/e["source_metadata"]["file"]
        if sha(metadata.read_bytes())!=e["source_metadata"]["sha256"]:
            raise ValueError("original metadata differs")
        doc=json.loads(metadata.read_text())
        if doc["source"]["file"]!="FDOTHER.DAT" or doc["source"]["resource"]!=resource or doc["source"]["sha256"]!="a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce":
            raise ValueError("wrong fixed source archive")
        bg=e["background"];path=original/bg["file"];image=Image.open(path)
        if image.mode!="P" or image.size!=(320,200) or sha(path.read_bytes())!=bg["file_sha256"] or sha(image.tobytes())!=bg["indexed_sha256"]:
            raise ValueError("original background differs")
        indexed=image.tobytes()
        if bg["mask_file"]:
            path=original/bg["mask_file"];mask=Image.open(path)
            if mask.mode!="L" or mask.size!=(320,200) or sha(path.read_bytes())!=bg["mask_sha256"] or not set(mask.tobytes())<={0,255}:
                raise ValueError("background mask differs")
            indexed=bytes(v if m else 0 for v,m in zip(indexed,mask.tobytes()))
        if sha(indexed)!=bg["typed_background_sha256"]:
            raise ValueError("typed background differs")
        frames=[]
        if [f["frame"] for f in e["original_frames"]]!=[0,1,2,3]:
            raise ValueError("original DATO frame set differs")
        for f in e["original_frames"]:
            if f["file"]!=f"DATO_{speaker:03d}_m{f['frame']}.png":
                raise ValueError("wrong original DATO filename")
            path=original/"portraits"/f["file"];image=Image.open(path)
            if image.mode!="P" or image.size!=(80,80) or sha(path.read_bytes())!=f["file_sha256"] or sha(image.tobytes())!=f["pixels_sha256"]:
                raise ValueError("original DATO source differs")
            frames.append(image.tobytes())
        x,y=offset%320,offset//320
        native_crop=b"".join(indexed[(y+row)*320+x:(y+row)*320+x+80] for row in range(80))
        if native_crop!=frames[0]:
            raise ValueError("native scene and original closed portrait do not join")
        master=e["master"];name=master["file"]
        if Path(name).name!=name or Path(name).suffix!=".png":
            raise ValueError("unsafe master filename")
        raw=(source/name).read_bytes();image=Image.open(io.BytesIO(raw)).convert("RGBA")
        if sha(raw)!=master["sha256"] or list(image.size)!=master["size"] or image.width<1280 or image.height<800 or abs(image.width/image.height-1.6)>=0.01 or image.getchannel("A").getextrema()!=(255,255):
            raise ValueError("selected full-scene master differs")
        prepared[name]=raw
        scene=image.resize((1920,1200),Image.Resampling.LANCZOS);encoded=png(scene)
        s=e["scene"]
        if (s["width"],s["height"])!=(1920,1200) or sha(encoded)!=s["sha256"]:
            raise ValueError("compiled scene differs")
        prepared[s["file"]]=encoded
        refs=[f"asset:FDOTHER_{resource:03d}",f"asset:DATO_{speaker:03d}_m0","docs/data/fd2-reference-files.json",
            "remake/assets/themes/modern/facility-portraits.recipe.json","tools/build_fd2_facility_portraits_hd.py",
            "docs/knowledge-base/61-modern-theme-runtime-spec.md"]
        assets.append(dict(asset_id=f"modern.fdother.{resource:03d}.scene.hd6.v1",role="facility_scene_hd",
            status="runtime_candidate",file=s["file"],width=1920,height=1200,sha256=sha(encoded),source_refs=refs,
            consumer_contract="native_facility_scene_hd6_v1",source_archive="FDOTHER.DAT",source_resource=resource,
            source_metadata_sha256=e["source_metadata"]["sha256"],master_file=name,source_files=[name],source_sha256=[master["sha256"]]))
        rect=[x*6,y*6,(x+80)*6,(y+80)*6]
        if e["portrait_rect_hd"]!=rect or [p["width"] for p in e["portraits"]]!=[80,480]:
            raise ValueError("portrait crop geometry differs")
        crop=scene.crop(rect)
        for p in e["portraits"]:
            size=p["width"];data=png(crop.resize((size,size),Image.Resampling.LANCZOS))
            if p["height"]!=size or sha(data)!=p["sha256"]:
                raise ValueError("compiled paired portrait differs")
            prepared[p["file"]]=data;hd=size==480
            assets.append(dict(asset_id=f"modern.dato.{speaker}.facility.portrait.{size}.v1",
                role="story_portrait_hd_frame" if hd else "story_portrait_frame",status="runtime_candidate",
                file=p["file"],master_file=name,width=size,height=size,sha256=sha(data),source_refs=refs,
                consumer_contract="native_story_dialogue_rgba_overlay_hd6_v1" if hd else "native_story_dialogue_rgba_overlay_v1",
                speaker_id=speaker,frame=0,mouth_state="closed"))
        proofs.append(dict(resource=resource,speaker_id=speaker,native_offset=offset,portrait_rect_hd=rect,
            scene_crop_rgba_sha256=sha(crop.tobytes()),portrait480_rgba_sha256=sha(crop.tobytes()),
            native_closed_portrait_equals_background=True,one_master=True))
    # Check all encoded bytes, filenames and ownership before publishing any file.
    out.mkdir(parents=True,exist_ok=True)
    if (out.stat().st_uid,out.stat().st_gid)!=(os.getuid(),os.getgid()):
        raise ValueError("output directory belongs to another user")
    for name,raw in prepared.items():
        if Path(name).name!=name or Path(name).suffix!=".png":
            raise ValueError("unsafe output filename")
        p=out/name
        if p.exists() and ((p.stat().st_uid,p.stat().st_gid)!=(os.getuid(),os.getgid()) or p.read_bytes()!=raw):
            raise ValueError("existing output differs: "+str(p))
    for name,raw in prepared.items():
        p=out/name
        if not p.exists():p.write_bytes(raw)
    return dict(status="PASS-SAME-MASTER-FACILITY-SCENE-PORTRAITS",assets=assets,proofs=proofs,
        files=[dict(file=n,sha256=sha(raw),size=len(raw)) for n,raw in sorted(prepared.items())],
        alpha="fully opaque RGBA",resampling="full 16:10 canvas to1920x1200; exact480 crop; 80 from480")

def main() -> None:
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ["recipe","source","original-pack","out","report"]:
        parser.add_argument("--"+name,type=Path,required=True)
    a=parser.parse_args()
    report=compile_pairs(json.loads(a.recipe.read_text()),a.source,a.original_pack,a.out)
    a.report.write_text(json.dumps(report,ensure_ascii=False,indent=2)+"\n")
    print(report["status"],len(report["assets"]),len(report["files"]))

if __name__=="__main__":
    main()
