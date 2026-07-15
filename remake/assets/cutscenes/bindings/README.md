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
