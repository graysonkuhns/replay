# Cursor Project Rules

This directory contains [Cursor Project Rules](https://docs.cursor.com/en/context/rules) that help the AI understand our codebase better.

## Current Rules

### project-context.mdc
- **Type**: Always Applied
- **Purpose**: Provides comprehensive project context, architecture overview, and conventions
- **Update when**: Making architectural changes, adding new features, or modifying patterns

### testing-conventions.mdc
- **Type**: Auto Attached (when working with test files)
- **Globs**: `**/e2e_tests/**/*.go`, `**/testhelpers/**/*.go`, `**/run_tests.sh`
- **Purpose**: E2E testing patterns and helper functions
- **Update when**: Adding new test helpers or changing testing patterns

### command-implementation.mdc
- **Type**: Auto Attached (when working with command files)
- **Globs**: `**/cmd/*.go`
- **Purpose**: Patterns for implementing CLI commands and message handlers
- **Update when**: Changing command structure or handler patterns

## Maintenance

These rules replace the deprecated `.cursorrules` file. Always keep them updated when:
- Making architectural changes
- Adding new patterns or conventions
- Changing testing approaches
- Modifying command structures

## Adding New Rules

1. Create a new `.mdc` file in this directory
2. Add metadata at the top:
   ```
   ---
   description: Brief description of the rule
   globs:        # Optional: file patterns for auto-attachment
     - "**/*.go"
   alwaysApply: false  # Or true for always-active rules
   ---
   ```
3. Write the rule content in Markdown format

## Rule Types

- **Always**: Always included in model context (`alwaysApply: true`)
- **Auto Attached**: Included when files matching glob patterns are referenced
- **Agent Requested**: AI decides whether to include (must have description)
- **Manual**: Only included when explicitly mentioned using @ruleName
