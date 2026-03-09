# Memoization

Memoization conventions for React 19 with the React Compiler. For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

<rules>

- Trust the React Compiler. Do not manually memoize. The Compiler inserts the equivalent of `React.memo`, `useMemo`, and `useCallback` automatically based on dependency analysis.
- Manual memoization (`useMemo`, `useCallback`, `React.memo`) is acceptable only as a targeted fix for a specific performance bottleneck identified through profiling. The key word is _profiling_: measure it, identify the bottleneck, apply the fix.
- Speculative memoization is prohibited. Do not wrap things in `useMemo`, `useCallback`, or `React.memo` "just in case" or because they "might be expensive." This adds noise, obscures intent, and duplicates work the Compiler already does.

</rules>

```typescript
// Correct: let the Compiler handle it
export const ProductList: FC<Props> = ({ products, filter }) => {
  const filtered = products.filter((p) => p.category === filter);
  const sorted = filtered.sort((a, b) => a.name.localeCompare(b.name));
  return <ul>{sorted.map((p) => <ProductItem key={p.id} product={p} />)}</ul>;
};

// Incorrect: speculative memoization
export const ProductList: FC<Props> = ({ products, filter }) => {
  const filtered = useMemo(                              // WRONG: the Compiler does this
    () => products.filter((p) => p.category === filter),
    [products, filter],
  );
  const sorted = useMemo(                                // WRONG: speculative
    () => filtered.sort((a, b) => a.name.localeCompare(b.name)),
    [filtered],
  );
  return <ul>{sorted.map((p) => <ProductItem key={p.id} product={p} />)}</ul>;
};

// Incorrect: wrapping components in React.memo by default
export const ProductItem: FC<Props> = React.memo(({ product }) => {  // WRONG: speculative
  return <li>{product.name}</li>;
});
```

The combination of the state management hierarchy (see [state.md](state.md)) and the React Compiler covers nearly the entire optimization surface. `useState` scopes re-renders to the owning component. Jotai provides atom-level subscriptions. Context re-renders broadly only when that is the correct behavior. The Compiler memoizes within re-rendered components. Manual memoization addresses the narrow gap where profiling reveals a bottleneck these mechanisms did not cover.
