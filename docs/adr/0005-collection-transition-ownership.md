# Give collection facts one owner and keep reconciliation explicit

Status: accepted design; implementation pending.

The collection model owns membership, source/display order, unavailable entries,
the chosen occurrence, Favorite association and survivor mappings; root UI owns
the ordered feature updates that consume committed changes. This concentrates
collection invariants without moving browsing visits, Grid interaction,
filesystem work or display retries from their existing owners, at the cost of
maintaining explicit reconciliation adapters. Independent field setters would
leave callers coordinating the same invariants, while a global store or event
dispatcher would obscure the required feature ordering.

The [accepted MA-030 design](../collection-transitions.md) records all eleven
decisions, the intended behavior corrections and required verification. Both
the collection model and root reconciliation are required for completion.
