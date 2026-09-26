# Share command admission while retaining feature-owned execution

Status: accepted design; implementation pending

Use one pure decision module in root UI over captured feature observations to
address repeated menu, shortcut and direct-entry admission inconsistencies;
handlers retain payload capture, effects and asynchronous validity.
Represent intent, route, input owner, visible surface and retained visit
explicitly so deliberate differences such as toggle versus Show, comparison's
Open refusal and focused-text editing survive centralization.
This concentrates cross-feature rules while preserving feature state ownership
and explicit composition, at the cost of maintaining observation/execution
adapters instead of introducing a central mutable controller or feature registry.

The [accepted design](../command-admission.md) records scope, behavior corrections
and the verification required before implementation can be considered complete.
