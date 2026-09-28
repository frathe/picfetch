# Share Favorite membership and ownership while keeping cache policy local

Status: accepted design; implementation pending.

`favstore` will own common membership validation, captured directory/list
identity, currentness checks and resource release so Favorite consumers no
longer coordinate those storage facts independently. Analysis leases, GPS
namespaces, thumbnail encoding and cache budgets/publication protocols stay in
their existing owners: moving them into one cache manager would couple distinct
policies, while sharing only JSON decoding would leave ownership duplication
intact. A captured membership or successful identity check is not an atomic
authorization for a later write; the [accepted MA-031 design](../favorite-ownership.md)
records all twelve decisions and the limits of external-change detection.

Captured ownership retires when the directory moves or the directory/list is
replaced, including identical-content replacement. Consumers retain ownership
values and open/revalidate handles for bounded work, making retirement consistent
across features without keeping idle per-Favorite handles that can prevent
removal on Windows.
