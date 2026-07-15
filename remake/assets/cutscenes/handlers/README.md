# Editable original-handler scripts

`chNN_pre.json` and `chNN_post.json` are the editable source representation of
the 30 original FD2 chapter handler pairs.  Each entry is a named operation
with its original call-site address in `source`; the remake never executes
that address.  This makes a script reviewable against the EXE while keeping
the campaign choreography independent of the original binary.

The files intentionally contain only decoded behavioural facts (operation,
parameters and source locations), not original resource bytes or dialogue
text.  Dialogue is referenced by its original table/index and is resolved by
the remake's separately authored text layer.

`unknown` is a deliberate, valid operation.  It marks a native call whose
semantics are not yet proven.  Do not delete it or replace it with a guessed
effect; extend the schema and engine only after RE evidence exists.

Regenerate all scripts after improving the instruction decoder:

```sh
python3 tools/export_handler_scripts.py /path/to/FD2.EXE all remake/assets/cutscenes/handlers
```

The command deterministically regenerates the 60 JSON scripts plus
`_manifest.json`.  The README is maintained by hand.
