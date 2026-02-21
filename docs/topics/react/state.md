# State Management

State management hierarchy and conventions. State is managed through three mechanisms, chosen by scope and re-render impact. For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## State Hierarchy

<rules>

- **`useState` is the first choice.** Use it for state that lives and dies with a single component. Most component state is `useState`.
- **Jotai is the first choice when state needs to exist beyond a single component.** Atoms provide granular subscriptions: only the specific consumers of a changed atom re-render. Use Jotai when multiple components need the same state but re-renders should be targeted.
- **React Context is for cross-component state where broad re-rendering is the correct behavior.** When a state change fundamentally affects an entire subtree (not just individual consumers), Context's wide-net re-render is intentional, not a limitation. Use Context for compound component internal state, theme, locale, and similar subtree-scoped concerns.

</rules>

```typescript
// Correct: useState for single-component state
export const SearchInput: FC<Props> = () => {
  const [query, setQuery] = useState("");   // local to this component
  return <input value={query} onChange={(e) => setQuery(e.target.value)} />;
};

// Incorrect: Jotai for state that only one component uses
const queryAtom = atom("");                 // WRONG: overkill for single-component state
export const SearchInput: FC<Props> = () => {
  const [query, setQuery] = useAtom(queryAtom);
  return <input value={query} onChange={(e) => setQuery(e.target.value)} />;
};

// Incorrect: Context for frequently-changing state across unrelated components
const SearchContext = createContext("");     // WRONG: every consumer re-renders on every keystroke
```

### Escalation Triggers

- `useState` to Jotai: another component needs this state.
- Jotai to Context: this state change should re-render the whole subtree, not just individual consumers.

### Jotai Conventions

<rules>

- Atoms are the primary unit of shared state. Each atom represents one piece of state with one reason to exist.
- Derived atoms (`atom((get) => get(baseAtom) + 1)`) make state relationships explicit and automatic.
- Atoms can be colocated with the component that owns them or placed in a shared location when multiple unrelated components consume them.
- Do not create god-object stores. Each atom is independent by default and composes only when needed.

</rules>

```typescript
// Single atom: one piece of state
export const countAtom = atom(0);

// Derived atom: explicit relationship
export const doubledAtom = atom((get) => get(countAtom) * 2);

// Component consuming an atom
export const Counter: FC<Props> = () => {
  const [count, setCount] = useAtom(countAtom);
  return <button onClick={() => setCount((c) => c + 1)}>{count}</button>;
};
```

### React Context Conventions

<rules>

- Use Context for state scoped to a component subtree where all children should re-render on change.
- Context is the correct mechanism for compound component internal state (e.g., a `Disclosure` parent sharing open/closed state with its children).
- Context is acceptable for infrequently-changing subtree configuration: theme, locale, feature flags.
- Do not use Context for frequently-changing state shared across unrelated components. Use Jotai instead.
- Do not layer Jotai on top of Context for the same concern. If a piece of state starts in Context and hits a Context limitation, migrate it to Jotai.

</rules>

```typescript
// Compound component using Context for internal shared state
type DisclosureContext = {
  open: boolean;
  toggle: () => void;
};

const Context = createContext<DisclosureContext | null>(null);

export const Disclosure: Component = ({ children }) => {
  const [open, setOpen] = useState(false);
  const toggle = () => setOpen((prev) => !prev);
  return <Context value={{ open, toggle }}>{children}</Context>;
};

// Sub-component consuming parent's context
const Trigger: FC<Props> = ({ children }) => {
  const { toggle } = use(Context)!;
  return <button onClick={toggle}>{children}</button>;
};

// Incorrect: Jotai for compound component internal state
const disclosureOpenAtom = atom(false); // WRONG: this is component-scoped, not global
```

```typescript
// Incorrect: god-object atom that aggregates unrelated state
const appStateAtom = atom({              // WRONG: couples unrelated concerns
  user: null,
  theme: "light",
  notifications: [],
  cartItems: [],
});

// Correct: independent atoms for independent concerns
export const userAtom = atom<User | null>(null);
export const themeAtom = atom("light");
export const notificationsAtom = atom<Notification[]>([]);
export const cartItemsAtom = atom<CartItem[]>([]);
```

## Props Drilling

<rules>

- Props drilling is prohibited. Do not pass props through multiple component layers where intermediate components only forward them.
- Passing props 1-2 levels is acceptable within composable components.

</rules>

```typescript
// Incorrect: props drilling through intermediate components
const Page: FC<Props> = ({ user }) => {
  return <Sidebar user={user} />;         // Page doesn't use user, just forwards it
};

const Sidebar: FC<Props> = ({ user }) => {
  return <UserMenu user={user} />;        // Sidebar doesn't use user, just forwards it
};

const UserMenu: FC<Props> = ({ user }) => {
  return <span>{user.name}</span>;        // only UserMenu actually uses user
};

// Correct: Jotai atom, each component reads directly
const Page: FC<Props> = () => {
  return <Sidebar />;
};

const UserMenu: FC<Props> = () => {
  const [user] = useAtom(userAtom);       // reads directly, no drilling
  return <span>{user.name}</span>;
};
```

### Alternatives

1. **`useState`.** Keep state local to the component that owns it. Pass data to children through composition, not props chains.
2. **Jotai.** Use atoms for shared state that multiple components need with granular re-render control.
3. **React Context.** Use for subtree-scoped state where broad re-rendering on change is the correct behavior.
4. **Composition.** Pass components as children instead of passing data through layers.
5. **Server Components.** Fetch data where it is needed rather than passing it down from a parent.
