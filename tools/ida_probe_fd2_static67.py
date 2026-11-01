"""第67組數值引用盤點；只在授權IDA容器的資料庫複本執行，身份保持未知。"""
import hashlib,json,os,sys
from pathlib import Path
import ida_auto,ida_bytes,ida_funcs,ida_idaapi,ida_kernwin,ida_nalt,ida_segment,ida_ua,idautils,idc
sys.path.insert(0,'/repo/tools')
from fd2_semantic_index import load_semantic_index
ida_auto.auto_wait()
raw=Path('/input/FD2.EXE').read_bytes()
identity=dict(file='FD2.EXE',size=len(raw),md5=hashlib.md5(raw).hexdigest(),sha256=hashlib.sha256(raw).hexdigest())
assert identity==dict(file='FD2.EXE',size=357074,md5='b97caf2239a27a896069d03549d96e1e',sha256='222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f')
db_sha=bytes(ida_nalt.retrieve_input_file_sha256()).hex();assert db_sha==identity['sha256'],db_sha
index,annotations=load_semantic_index('/repo/docs/data/ida/fd2_semantic_index.json',repo_root='/repo',include_instructions=True)
def function(ea):
    f=ida_funcs.get_func(ea)
    if not f:return None
    return dict(address=hex(f.start_ea),end=hex(f.end_ea),original_name=idc.get_func_name(f.start_ea),semantic_annotations=annotations.get(f.start_ea,[]),semantic_level='已證實' if any(a.get('confidence')=='已證實' for a in annotations.get(f.start_ea,[])) else '未知',source='FD2.EXE IDA LE linear '+hex(f.start_ea),unconfirmed_warning=not any(a.get('confidence')=='已證實' for a in annotations.get(f.start_ea,[])))
def instruction(ea):
    size=idc.get_item_size(ea)
    return dict(address=hex(ea),bytes=(ida_bytes.get_bytes(ea,size) or b'').hex(),original_disassembly=idc.generate_disasm_line(ea,0),semantic_annotations=annotations.get(ea,[]),semantic_level='已證實' if any(a.get('confidence')=='已證實' for a in annotations.get(ea,[])) else '未知',source='FD2.EXE IDA LE linear '+hex(ea),unconfirmed_warning=not any(a.get('confidence')=='已證實' for a in annotations.get(ea,[])))
def context(ea,before=12,after=6):
    seg=ida_segment.getseg(ea);left=[];right=[];cursor=ea
    for _ in range(before):
        cursor=idc.prev_head(cursor,seg.start_ea)
        if cursor==ida_idaapi.BADADDR:break
        left.append(cursor)
    cursor=ea
    for _ in range(after):
        cursor=idc.next_head(cursor,seg.end_ea)
        if cursor==ida_idaapi.BADADDR:break
        right.append(cursor)
    return [instruction(x) for x in list(reversed(left))+[ea]+right]
matches=[]
for ea in idautils.Heads(0x10000,0x4f000):
    if not ida_bytes.is_code(ida_bytes.get_full_flags(ea)):continue
    ins=ida_ua.insn_t();ida_ua.decode_insn(ins,ea)
    values=[int(op.value) for op in ins.ops if op.type==ida_ua.o_imm and int(op.value) in (67,)]
    if values:matches.append(dict(immediates=values,function=function(ea),instruction=instruction(ea),context=context(ea),level='未知：立即數本身不證明圖組或角色身份'))
callers=[]
for x in idautils.CodeRefsTo(0x11019,False):callers.append(dict(address=hex(x),function=function(x),context=context(x,18,8)))
d=dict(schema='fd2-static67-ida-probe-v1',input=identity,database_input_sha256=db_sha,ida_version=ida_kernwin.get_kernel_version(),address_space='IDA LE linear; code segment starts at0x10000, addresses are not file offsets',function_count=len(list(idautils.Functions())),script_sha256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),semantic_index_sha256=hashlib.sha256(Path('/repo/docs/data/ida/fd2_semantic_index.json').read_bytes()).hexdigest(),immediate_candidates=matches,selector_constructor_callers=callers,scope='原始名稱、bytes與來源索引保持；立即數只作查詢線索，不命名角色，不改正式資料庫。')
out=Path(os.environ.get('FD2_IDA_OUTPUT','/work/ida-static67-source.json'))
assert out.parent.stat().st_uid==os.getuid() and not out.exists()
tmp=out.with_suffix('.tmp');tmp.write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');tmp.replace(out)
idc.qexit(0)
