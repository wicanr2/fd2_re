import hashlib, json, pathlib, os
import sys
import ida_auto, ida_bytes, ida_funcs, ida_xref, idaapi, idautils, idc
repo = pathlib.Path(__file__).resolve().parent.parent
sys.path.insert(0, str(repo / 'tools'))
from fd2_semantic_index import load_semantic_index, validate_input_identity
ida_auto.auto_wait()
p=pathlib.Path('/input/FD2.EXE'); b=p.read_bytes()
assert hashlib.sha256(b).hexdigest()=='222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f'
out={'input':p.name,'size':len(b),'md5':hashlib.md5(b).hexdigest(),'sha256':hashlib.sha256(b).hexdigest(),'tool':'IDA Pro '+idaapi.get_kernel_version(),'address_space':'IDA LE linear','functions':[]}
index, annotations = load_semantic_index(repo / 'docs/data/ida/fd2_semantic_index.json', repo,
                                         include_instructions=True)
validate_input_identity(index['input'], {'file':p.name, 'size':len(b), 'md5':out['md5'], 'sha256':out['sha256']})
def annotate(row, address):
 row['semantic_annotations'] = annotations.get(address, [])
 row.setdefault('semantic', '未附加推測語意')
 row.setdefault('level', 'unknown')
 row.setdefault('warning', '語意未經本次審查')
 row.setdefault('source', f'FD2.EXE IDA LE {address:#x}')
 if row['semantic_annotations']:
  entry = row['semantic_annotations'][0]
  row.update(semantic=entry['semantic'], level=entry['confidence'], source=entry['evidence'])
  row['warning'] = '' if entry['confidence']=='已證實' else '警示：語意尚未證實'
for ea in [int(v,0) for v in os.getenv('PROBE_ADDRESSES','0x1741c 0x179d5 0x117e7 0x1aeb1').split()]:
 f=ida_funcs.get_func(ea)
 if f is None: continue
 row={'requested_address':hex(ea),'address':hex(f.start_ea),'end':hex(f.end_ea),'original_name':idc.get_func_name(f.start_ea),'chunks':[{'start':hex(start),'end':hex(end)} for start,end in idautils.Chunks(f.start_ea)],'instructions':[],'xrefs':[]}
 x=ida_xref.get_first_cref_to(f.start_ea)
 while x!=idaapi.BADADDR:
  row['xrefs'].append(hex(x)); x=ida_xref.get_next_cref_to(f.start_ea,x)
 # 函式本體的 end_ea 不涵蓋非連續共用尾段；以資料庫的函式 items 保留所有 chunks。
 for i in sorted(set(idautils.FuncItems(f.start_ea))):
  if ida_bytes.is_code(ida_bytes.get_full_flags(i)):
   row['instructions'].append({'address':hex(i),'bytes':ida_bytes.get_bytes(i,idc.get_item_size(i)).hex(),'original':idc.generate_disasm_line(i,0),'semantic':'未附加推測語意','level':'unknown','warning':'語意未經本次審查','source':f'FD2.EXE IDA LE {i:#x}'})
 if not any(int(instruction['address'],0)==ea for instruction in row['instructions']):
  raise ValueError(f'探針要求的指令 {ea:#x} 不在函式匯出內')
 out['functions'].append(row)
for target in [int(v,0) for v in os.getenv('PROBE_DATA_ADDRESSES','').split()]:
 out.setdefault('data_initial',{})[hex(target)]={'bytes':ida_bytes.get_bytes(target,4).hex(),'address_space':'IDA LE linear','level':'confirmed','source':'IDA loader initial image; not a runtime snapshot'}
 rows=[]
 x=ida_xref.get_first_dref_to(target)
 while x!=idaapi.BADADDR:
  f=ida_funcs.get_func(x)
  rows.append({'address':hex(x),'bytes':ida_bytes.get_bytes(x,idc.get_item_size(x)).hex(),'original':idc.generate_disasm_line(x,0),'function':hex(f.start_ea) if f else None,'semantic':'未附加推測語意','level':'unknown','warning':'語意未經本次審查','source':f'FD2.EXE IDA LE {x:#x}'})
  x=ida_xref.get_next_dref_to(target,x)
 out.setdefault('data_xrefs',{})[hex(target)]=rows
for function in out['functions']:
 annotate(function, int(function['address'], 0))
 for instruction in function['instructions']:
  annotate(instruction, int(instruction['address'], 0))
for rows in out.get('data_xrefs', {}).values():
 for row in rows:
  annotate(row, int(row['address'], 0))
pathlib.Path(os.getenv('PROBE_OUTPUT','/shots/ida-input-probe.json')).write_text(json.dumps(out,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
idc.qexit(0)
