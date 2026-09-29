# Capture launch permissions independently of feature lifetime

Status: accepted; implemented and qualified in MA-033.

PicFetch captures validated launch permissions and storage selection before
covered startup effects, and requires root composition and updater admission to
consume that fixed decision. Deriving permission from live trial feature objects
couples update safety to feature construction and lifetime; centralizing only
viewer predicates would still leave direct updater calls dependent on caller
discipline. Missing policy refuses effects, while explicit construction from
ordinary launch options preserves normal behavior.

Effectful preparation owns validation, evidence reservation and resource cleanup;
the policy value owns none of those resources. This keeps application identity
and update admission usable before Fyne construction without turning launch
policy into a general network/filesystem permission service. Distribution and
trial purpose compose their restrictions. One preparation owner surrounds the
UI run, retaining cleanup responsibility while the UI borrows recorders and
joins trial-evidence producers before final evidence closure. This avoids a
resource handoff gap on startup errors and keeps the immutable decision resource-free.
The [accepted design](../launch-policy.md) records all ten decisions, including
the existing path-routing isolation guarantee and its external-replacement limit.
