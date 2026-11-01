#!/usr/bin/env python3
"""Compile the pinned, anonymous Frame137 count symbol without actor binding."""
from __future__ import annotations
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import PIL
from PIL import Image

def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()

def compile_icon(recipe: dict, master: Path, original: Path, out: Path) -> dict:
    if PIL.__version__ != "11.3.0":
        raise ValueError("requires the maintained Pillow11.3.0 Docker image")
    if recipe["schema"] != "fd2-native-ui-embedded-icon-hd6-v1":
        raise ValueError("unsupported embedded UI recipe")
    source=recipe["source"]
    if (source["archive"]!="FDOTHER.DAT" or source["resource"]!=5 or source["codec"]!="four_mode_frame"
            or source["frame_size"]!=[86,86] or source["entry"] != 137 or source["rect"] != [20,22,44,40] or source["repeated_rect"] != [20,60,44,78]):
        raise ValueError("unsupported original embedded symbol")
    root=original/"ui/fdother_005_item_panel"
    if digest(root/"resource.json") != source["metadata_sha256"]:
        raise ValueError("original metadata differs")
    frame=Image.open(root/"entry_137/frame.png")
    mask=Image.open(root/"entry_137/mask.png")
    if frame.mode!="P" or frame.size!=(86,86) or mask.mode!="L" or mask.size!=(86,86):
        raise ValueError("original Frame137 representation differs")
    if hashlib.sha256(frame.tobytes()).hexdigest()!=source["indexed_sha256"] or hashlib.sha256(mask.tobytes()).hexdigest()!=source["mask_sha256"]:
        raise ValueError("original Frame137 pixels or mask differ")
    for im in [frame,mask]:
        if im.crop(source["rect"]).tobytes()!=im.crop(source["repeated_rect"]).tobytes():
            raise ValueError("original count symbols differ")
    if digest(master)!=recipe["master"]["sha256"]:
        raise ValueError("selected master SHA differs")
    image=Image.open(master)
    if image.mode!="RGBA" or image.size!=tuple(recipe["master"]["size"]) or image.getchannel("A").getextrema()!=(0,255):
        raise ValueError("master shape or real alpha differs")
    core=image.getchannel("A").point(lambda value:255 if value>=16 else 0).getbbox()
    if not core or core[0]<=0 or core[1]<=0 or core[2]>=image.width or core[3]>=image.height:
        raise ValueError("visible master subject is cropped")
    if (recipe["output"]["width"],recipe["output"]["height"])!=(144,108):
        raise ValueError("output must follow the original 24x18 source at HD6")
    encoded=io.BytesIO()
    image.resize((144,108),Image.Resampling.LANCZOS).save(encoded,format="PNG")
    png_bytes=encoded.getvalue()
    if hashlib.sha256(png_bytes).hexdigest()!=recipe["output"]["sha256"]:
        raise ValueError("compiled PNG differs from pinned output")
    out.mkdir(parents=True,exist_ok=True)
    if out.stat().st_uid!=os.getuid():
        raise ValueError("output directory belongs to another user")
    for name in [recipe["master"]["file"],recipe["output"]["file"]]:
        if Path(name).name!=name or Path(name).suffix!=".png":
            raise ValueError("unsafe output filename")
    target_master=out/recipe["master"]["file"]
    target_png=out/recipe["output"]["file"]
    for target,expected in [(target_master,recipe["master"]["sha256"]),(target_png,recipe["output"]["sha256"])]:
        if target.exists() and (target.stat().st_uid!=os.getuid() or digest(target)!=expected):
            raise ValueError("existing output differs: "+str(target))
    # All inputs and encoded output are checked before either destination writes.
    # Preserve the full RGBA canvas; the core threshold never edits its alpha.
    if not target_master.exists():
        target_master.write_bytes(master.read_bytes())
    if not target_png.exists():
        target_png.write_bytes(png_bytes)
    return {"status":"PASS-SOURCE-BOUND-EMBEDDED-ICON","master":str(target_master),"master_sha256":digest(target_master),
            "png":str(target_png),"png_sha256":digest(target_png),"source_entry":137,"native_source_rect":source["rect"],
            "master_size":list(image.size),"output_size":[144,108],"alpha16_bbox":list(core),
            "alpha_policy":"complete RGBA resampled; no thresholding","identity":"anonymous UI count symbol, no actor binding"}

def main() -> None:
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--recipe",type=Path,required=True)
    parser.add_argument("--master",type=Path,required=True)
    parser.add_argument("--original-pack",type=Path,required=True)
    parser.add_argument("--out",type=Path,required=True)
    parser.add_argument("--report",type=Path,required=True)
    args=parser.parse_args()
    report=compile_icon(json.loads(args.recipe.read_text()),args.master,args.original_pack,args.out)
    args.report.write_text(json.dumps(report,ensure_ascii=False,indent=2)+"\n")
    print(json.dumps(report,ensure_ascii=False))

if __name__=="__main__":
    main()
