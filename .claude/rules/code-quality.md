# Code Quality (zen-kit)

Go specifics for this repo. The principles live in the global rules, which load
automatically; this file only adds what the stack demands. Don't restate the
global rules here.

## Naming

- **Packages**: short, lowercase, no underscores (`theme`, `syntax`, `paint`). One package per directory.
- **Files**: snake_case (`diff_line.go`, `hunk_header.go`). Tests are `foo_test.go` beside `foo.go`.
- **Identifiers**: exported PascalCase, unexported camelCase. No stutter (`syntax.New`, not `syntax.NewSyntax`).
- **Constants**: Go style (`MixedCaps`), not SCREAMING_SNAKE.

## Go specifics

- No `any` or `interface{}` where a concrete type or a small interface works. Accept interfaces, return structs.
- Wrap errors with `%w` and context: `fmt.Errorf("loading style %q: %w", name, err)`. Never discard an error silently.
- No naked returns in anything longer than a few lines.
- Table-driven tests for anything with more than two cases.

## Layers

One rule, and it is the reason this module exists separately from its consumers.

- **`paint` imports `theme` and `syntax`. Neither of them imports anything of ours.** `go list -deps ./theme ./syntax` naming a third `zen-kit` package means the boundary has already broken.

## What does not live here

zen-kit is the visual layer. No model, no state, no layout, no keys. Every
exported function is pure: same arguments, same string.

Folding, scroll, side-by-side layout, hunk grouping, the two-sided tokenise
split and review state belong to the caller. A feature that needs to remember
something between calls belongs in the tool, not here.

## Styling

Everything styles from the `theme.Theme` it was handed, never a hardcoded
Lipgloss color and never a Lipgloss default. A color that isn't in the theme
struct means the theme struct needs a field.

A nil color field means "leave the terminal's own showing". Guard for it rather
than passing it to Lipgloss, or a transparent background stops being
transparent.

## Tests

- Tests ship in the same PR as the logic, never a follow-up.
- Test the returned string, not an internal helper's arithmetic. Width, alignment and where the background stops are the whole product.

### Golden files

ANSI asserted inline is unreadable, so painted output compares against
`testdata/*.golden`, one file per case.

- `-update` regenerates them. Regenerating is how you write one; reading the diff is how you review it.
- A golden file is evidence only if it is stable. Anything varying by machine, terminal or clock has no place in one.
- A case earns a file by being a way the painter goes wrong, not by being another example of it going right.

## File size

Keep files focused. A file too big to review in one sitting is doing too much;
split it before it gets there rather than after.
