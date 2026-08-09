# Frontend State Contract

## Trigger

Apply this spec to server fetching, refresh behavior, auth, localStorage,
period/range selection, URL-derived values, or proposals for global state.

## State Ownership

| State | Current owner | Contract |
| --- | --- | --- |
| auth token | `lib/api.ts` localStorage helpers | API client attaches bearer token; 401 clears and reloads |
| page authentication view | `App.tsx` | initial authenticated/login route choice |
| period and custom range | `Dashboard.tsx` | applied query state; popover draft stays local |
| dashboard server data | `Dashboard.tsx` | overview/accounts/keys/trend/models/collector refreshed as one snapshot |
| refresh preference | `useAutoRefresh.ts` | validated localStorage-backed finite interval |
| table/chart controls | owning component | sort, metric, view, hidden series |

- Keep state at the narrowest owner that coordinates all consumers.
- Do not introduce Redux, Zustand, React Query, or another state library for a
  single-page flow. A new dependency needs repeated cache/synchronization needs
  that current page state cannot express simply.

## Server Refresh Contract

- One dashboard refresh MUST request overview, accounts, keys, trend, models,
  and collector concurrently with `Promise.all`.
- A monotonically increasing request sequence MUST prevent an older response or
  error from overwriting a newer request's state.
- Manual refresh MAY supersede automatic refresh. Silent automatic polls MUST
  use an in-flight lock so slow requests do not pile up.
- Silent refresh MUST not re-enable the page skeleton. It replaces data only
  after the full latest snapshot succeeds.
- Repeated silent failures SHOULD emit one toast until a successful refresh
  resets the error latch. Manual failures remain immediately visible.
- Period change MUST generate a new query; incomplete custom range falls back to
  the documented safe period in `periodQuery`.

Evidence: `frontend/src/pages/Dashboard.tsx` and `frontend/src/lib/api.ts`.

## Derived State

- Values that can be derived from current typed state SHOULD be calculated at
  render or with `useMemo`, not stored separately.
- Sorting MUST derive a copied array. Model token/cost view switches re-sort the
  already returned ranking; they do not issue a second request.
- Popover/editor drafts MAY be local state when apply/cancel semantics matter.

## Avoid

- Independent component fetches that produce mismatched dashboard snapshots.
- Clearing already visible data for every silent poll.
- Letting a stale request show an error after a newer request started.
- Overlapping automatic polls.
- Duplicating the auth token in React state and localStorage without one owner.
- Promoting local sort/view/toggle state to a global store.

## Verify

- Test reasoning for this sequence: slow old request, period change, fast new
  request; only new data may win.
- Verify long-running silent poll skips the next tick and failure toast dedupes.
- Verify 401 clears token and returns to login.
- Run `npm run build`.
