# Hook Contract

## Trigger

Apply this spec when adding or changing custom hooks, timers, subscriptions,
browser storage, or reusable effect lifecycle.

## Pattern

- Create a custom hook only for reusable stateful/effect behavior. Pure
  calculations belong in `lib/`; one component's simple local state stays in
  that component.
- Hook names MUST begin with `use`; exported state and actions MUST have an
  explicit return type when they form a reusable contract.
- Effects that allocate a timer, listener, or subscription MUST return cleanup.
- When a timer needs the latest callback but should not restart on every render,
  store the callback in a ref and update that ref in a separate effect.
- Persist only validated, finite choices. `useAutoRefresh` reads localStorage,
  distinguishes missing from numeric zero, validates against
  `REFRESH_OPTIONS`, and falls back to `DEFAULT_REFRESH`.
- Browser globals MUST be guarded when a hook can run during non-browser build
  or evaluation.

Evidence: `frontend/src/hooks/useAutoRefresh.ts`.

## Timer Contract

- `0` means disabled and MUST create no interval.
- Changing interval MUST clean up the old timer before installing the next.
- Component unmount MUST clear the timer.
- Timer ticks call the latest callback through a ref; callback identity alone
  MUST NOT reset cadence.
- Request overlap prevention belongs with the page request lifecycle, not inside
  a generic timer hook.

## Avoid

- An effect with `setInterval` and no cleanup.
- Adding `onTick` to the interval effect and restarting cadence on every page
  callback change.
- Treating `Number(null) === 0` as the user's persisted disabled choice.
- Writing arbitrary localStorage numbers into a literal-union state.
- A hook that mixes timer lifecycle, API fetching, toast policy, and component
  presentation.

## Verify

- Review effect dependencies and cleanup manually.
- Verify disabled, default, each allowed interval, invalid persisted value, and
  unmount behavior when the hook changes.
- Run `npm run build`.
