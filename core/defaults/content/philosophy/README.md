# Philosophy

Guiding principles for decision-making in this project. These are not concrete rules; architectural patterns, file organization, and implementation specifics are documented separately. The principles here inform _why_ those concrete rules exist.

## Guiding Principles

### 1. Consistency

The codebase is a consensus on approach. When an established pattern exists, follow it. Deviation requires a pragmatic reason: one that demonstrably improves clarity, performance, or maintainability. Such deviations reveal blind spots in current design and inform updates to the documented approach.

### 2. Clarity Over Dogma

Rules serve clarity and maintainability. When principles conflict, favor the approach that makes code easier to understand and modify. Pragmatism over purity.

### 3. Structural Balance

Group related code by cohesion; split when cohesion is lost. Neither a thousand tiny files nor monolithic files serve the developer. One logical unit per file (a component, a type, a utility function). Topic conventions may define what "one unit" means for their domain.

### 4. Abstractions Must Earn Their Place

Every abstraction (wrapper, utility, helper) must serve clarity, performance, testability, or maintainability. Naming must reflect intent. Avoid unnecessary indirection. Use separation when it aids understanding.

### 5. Leverage Before Building

Before writing a custom solution, search for existing packages that address the problem. Well-maintained packages handle edge cases, performance concerns, and compatibility issues that custom implementations often miss.

Not all packages are equal. Evaluate candidates critically: maintenance activity, community adoption, bundle size, API quality, security posture, and alignment with the project's conventions. Building custom is justified when no existing solution meets the evaluation criteria, when a package would introduce disproportionate complexity, or when the problem is genuinely project-specific.

### 6. Configuration is Truth

Documentation references configuration; it does not duplicate it. When documentation and configuration conflict, configuration wins.

### 7. Write for the Floor

Write documentation and instructions for the least capable model expected to consume them. Clarity at the baseline never degrades performance for advanced models, but complexity at the ceiling always excludes lesser ones. See [ai-authoring](../ai-authoring/README.md) for the concrete conventions that implement this principle.
