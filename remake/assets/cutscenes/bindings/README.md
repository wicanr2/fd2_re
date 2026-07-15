# Handler bindings

Bindings are the author-controlled bridge from an extracted handler script to
the remake engine.  They are keyed by the original call-site address, not by
resource ID or text index: the same original value can legitimately mean a
different thing after another `loadch`.

Keep a binding partial until every operation it needs is proven.  The compiler
reports every absent binding as an issue and a caller must not start an
incomplete handler as though it were faithful.  This separation lets future
campaigns reuse the engine without copying FD2-specific addresses or guessing
coordinate/text/actor conversions.

When one original `dialog` call contains several FDTXT messages, use
`dialog.lines` to list its remake line beats in source order.  Include its
`script` and `scene`: line indices are local to that editable text scene and
must not be inferred from the original numeric FDTXT index alone.

For a count-exact resource, a binding can instead set `story_index_map` and
address-keyed `dialogue_contexts`.  Each context supplies both `source_dat`
and the story-relative `script`; the engine resolves the original string index
only through `dialogue-index/count-aligned.json`.  Hand-authored `dialog`
overrides still win, which is required for observed per-line presentation
metadata such as `upper`.  A raw string that spans scenes remains unresolved
until a scene-transition adapter is implemented.
