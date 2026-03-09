# React

React component conventions for this repository. This directory covers component patterns, file organization, composability, state management, error handling, and performance. Framework-specific concerns (RSC boundaries, routing, server actions) are in [docs/topics/nextjs/](../nextjs/README.md). Type system foundations are in [docs/topics/typescript/README.md](../typescript/README.md).

For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

| Document                                             | Purpose                                              |
| ---------------------------------------------------- | ---------------------------------------------------- |
| [components.md](components.md)                       | Component definition, props, and global types        |
| [composable-components.md](composable-components.md) | Dot notation namespace pattern and sub-components    |
| [file-organization.md](file-organization.md)         | Directory structure, barrels, and recursive pattern   |
| [hooks.md](hooks.md)                                 | Hook conventions and extraction triggers             |
| [state.md](state.md)                                 | State hierarchy: useState, Jotai, Context            |
| [error-handling.md](error-handling.md)               | Error Boundaries, Suspense, and isolation boundaries |
| [memoization.md](memoization.md)                     | React Compiler and manual memoization policy         |
