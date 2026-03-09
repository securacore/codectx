# Hooks

Hook conventions and extraction triggers. Hooks follow the same directory and barrel pattern as components. For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Hook Conventions

<rules>

- Every hook lives in its own named directory with its own `index.ts` barrel.
- Use `type Props` for hook input when the hook accepts configuration. This is the same naming convention as components.
- Use `type Return` for hook output when the return is a complex type that serves as a contract.
- `Return` is a standardized name for recognition, not a constraint on shape. The type can be an object, tuple, union, or any other valid TypeScript type.
- Omit `type Return` when the return type is obvious from the implementation and can be properly inferred. The goal is type safety, not ceremony.
- Zero `any` types. No exceptions without explicit approval.

</rules>

```typescript
// Hook with explicit Props and Return contracts
type Props = {
  cardId: string;
};

type Return = {
  data: Card | null;
  loading: boolean;
  error: Error | null;
  refresh: () => void;
};

export const useCardApi = ({ cardId }: Props): Return => {
  // ...
};

// Return can be any shape
type Return = [boolean, () => void];
type Return = Card | null;
type Return = string;

// Inference is acceptable when the return type is obvious
export const useToggle = (initial: boolean = false) => {
  const [value, setValue] = useState(initial);
  const toggle = useCallback(() => setValue((v) => !v), []);
  return [value, toggle] as const;
};

// Incorrect: naming the type after the hook
type UseCardApiReturn = { ... };
type UseCardApiProps = { ... };

// Incorrect: inline return type
export const useCardApi = ({ cardId }: Props): { data: Card | null; loading: boolean } => ...

// Incorrect: using any
type Return = {
  data: any;
};
```

## Extraction Triggers

When to add internal structure to a component. This is not an arbitrary complexity threshold. Extract when concerns have different reasons to change.

<rules>

- **External integrations.** A component calls an API, connects to a WebSocket, or interacts with a third-party service. Extract the integration lifecycle into a hook. The hook's return type is the stable interface. If the integration changes, only the hook changes. The component renders; the hook integrates.
- **State complexity.** Multiple pieces of state depend on each other or derive from each other. Extract into a hook that encapsulates the state machine.
- **Effect density.** Multiple effects managing related concerns (subscriptions, timers, event listeners). These are integration concerns, not rendering concerns. Group them in a hook.

</rules>

The common thread: the component renders, hooks integrate. If something is not about rendering, it belongs in a hook. The hook provides a stable interface to the component. If the underlying integration changes, only the hook implementation changes. This is the same abstraction principle as the [Just operational directive](../just/README.md): the consumer interacts with a stable interface, and the implementation can change behind it.
