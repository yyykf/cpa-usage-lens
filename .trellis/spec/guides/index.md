# Project Thinking Guides

Guides are short pre-change questions. Normative implementation rules live in
the backend/frontend specs linked below.

## Router

| Trigger | Read |
| --- | --- |
| change spans source, storage, report, API, or UI | [Cross-Layer Guide](./cross-layer-thinking-guide.md) |
| add a helper, formatter, token rule, report aggregate, table, or primitive | [Code-Reuse Guide](./code-reuse-thinking-guide.md) |
| CPA queue, accounting, price, migration, DTO, deployment contract | both guides plus the routed backend/frontend specs |

## Mandatory Questions

- [ ] Which layer owns the authoritative value or decision?
- [ ] What source, test, schema, or config proves the current rule?
- [ ] Which downstream consumers and defaults carry the same field/meaning?
- [ ] Does an existing helper already encode this business rule?
- [ ] What invalid, null, zero, empty, partial, retry, and recovery cases exist?
- [ ] Which automated gate and manual evidence can actually verify the change?

Use backend [router](../backend/index.md) and frontend
[router](../frontend/index.md) to load the normative contracts before editing.
