"""在一次性IDA資料庫複核22253背景來源，保留原始位址、bytes與分級索引。"""
import hashlib
import json
import os
import sys
from pathlib import Path

import ida_auto
import ida_bytes
import ida_funcs
import ida_lines
import ida_nalt
import ida_xref
import idaapi
import idautils
import idc

sys.path.insert(0, str(Path(__file__).parent))
from fd2_semantic_index import load_semantic_index, validate_input_identity


def main():
    raw = Path('/input/FD2.EXE').read_bytes()
    identity = dict(file='FD2.EXE',size=len(raw),md5=hashlib.md5(raw).hexdigest(),sha256=hashlib.sha256(raw).hexdigest())
    doc, semantics = load_semantic_index('/repo/docs/data/ida/fd2_semantic_index.json','/repo',include_instructions=True)
    validate_input_identity(doc['input'],identity)
    ida_auto.auto_wait()
    digest = ida_nalt.retrieve_input_file_sha256()
    assert digest and bytes(digest).hex()==identity['sha256']
    result = dict(schema_version=1,input=identity,database_input_sha256=bytes(digest).hex(),
        tool=dict(name='IDA Pro',version=idaapi.get_kernel_version(),address_space='IDA LE loader linear address'),
        exporter_sha256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        database_function_count=sum(1 for _ in idautils.Functions()),imagebase=idaapi.get_imagebase())
    if os.environ.get('FD2_IDA_SMOKE')!='1':
        fn=ida_funcs.get_func(0x22253)
        assert fn and fn.start_ea==0x22253
        callers=[]
        ea=ida_xref.get_first_cref_to(fn.start_ea)
        while ea!=idaapi.BADADDR:
            caller=ida_funcs.get_func(ea)
            callers.append(dict(ida_linear_address=hex(ea),function_start=hex(caller.start_ea) if caller else None,
                original_instruction=ida_lines.tag_remove(idc.generate_disasm_line(ea,0) or '')))
            ea=ida_xref.get_next_cref_to(fn.start_ea,ea)
        instructions=[]
        for ea in idautils.FuncItems(fn.start_ea):
            if not ida_bytes.is_code(ida_bytes.get_full_flags(ea)):continue
            size=idc.get_item_size(ea)
            annotations=semantics.get(ea,[])
            instructions.append(dict(ida_linear_address=hex(ea),original_name=idc.get_name(ea),
                raw_bytes=ida_bytes.get_bytes(ea,size).hex(),
                original_instruction=ida_lines.tag_remove(idc.generate_disasm_line(ea,0) or ''),
                semantic_annotations=annotations,semantic_confidence=annotations[0]['confidence'] if annotations else '未知',
                unconfirmed_semantics=not annotations or any(a['confidence']!='已證實' for a in annotations)))
        result['function']=dict(start=hex(fn.start_ea),end=hex(fn.end_ea),original_name=idc.get_func_name(fn.start_ea),
            semantic_annotations=semantics.get(fn.start_ea,[]),callers=callers,instructions=instructions)
    output=Path(os.environ['FD2_IDA_OUTPUT'])
    assert not output.exists()
    output.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    idc.qexit(0)


if __name__=='__main__':main()
