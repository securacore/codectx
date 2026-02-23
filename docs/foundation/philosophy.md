# Philosophy

This section describes the conceptual philosophy that guides decision-making in this codebase. These are not concrete rules; architectural patterns, file organization, and implementation specifics are documented separately. The principles in the Guiding Principles section inform _why_ those concrete rules exist.

## Guiding Principles

### 1. SOLID and DRY

Follow SOLID and DRY as the foundation for maintainable, extensible code. These are not negotiable starting points. All other principles build on them.

### 2. Consistency

The codebase is a consensus on approach. When an established pattern exists, follow it. Deviation requires a pragmatic reason: one that demonstrably improves clarity, performance, or maintainability. Such deviations reveal blind spots in current design and inform updates to the documented approach.

### 3. Clarity Over Dogma

Rules serve clarity and maintainability. When principles conflict, favor the approach that makes code easier to understand and modify. Pragmatism over purity.

### 4. Structural Balance

Group related code by cohesion; split when cohesion is lost. Neither a thousand tiny files nor monolithic files serve the developer. One logical unit per file (a component, a type, a utility function). Topic conventions may define what "one unit" means for their domain.

### 5. Abstractions Must Earn Their Place

Every abstraction (wrapper, utility, hook) must serve clarity, performance, testability, or maintainability.

Naming must reflect intent: functions are functions, hooks are hooks (uses React lifecycle). Avoid unnecessary indirection. Use separation when it aids understanding.

### 6. Leverage Before Building

Before writing a custom solution, search for existing packages that address the problem. The ecosystem evolves rapidly; well-maintained packages handle edge cases, performance concerns, and cross-browser issues that custom implementations often miss.

Not all packages are equal. Evaluate candidates critically: maintenance activity, community adoption, bundle size, API quality, security posture, and alignment with the project's conventions. Third-party dependencies are a potential attack vector; due diligence is not optional.

Building custom is justified when no existing solution meets the evaluation criteria, when a package would introduce disproportionate complexity, or when the problem is genuinely project-specific. The goal is informed evaluation, not blind adoption.

### 7. Configuration is Truth

Documentation references configuration; it does not duplicate it. Validate paths, aliases, and settings against actual config files. When documentation and configuration conflict, configuration wins.

### 8. Write for the Floor

Write documentation and instructions for the least capable model expected to consume them. Clarity at the baseline never degrades performance for advanced models, but complexity at the ceiling always excludes lesser ones. See [ai-authoring.md](ai-authoring.md) for the concrete conventions that implement this principle.
