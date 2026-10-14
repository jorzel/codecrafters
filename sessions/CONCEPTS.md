# Concepts

A living index of broad concepts across sessions, sorted alphabetically.
When a concept comes up again, add a session link and sharpen its summary.
Details stay in the session files.

| Concept | Summary | Challenge | Sessions |
|---------|---------|-----------|----------|
| Architecture of an interpreter | Keep logic pure (functional core) and push I/O to the edge (imperative shell). Layers map onto a shell, but parsing is domain, because the language is the product | shell | [001](001-2026-10-14-shell.md) |
| Go's I/O model | `io.Reader`/`io.Writer` are one-method interfaces satisfied implicitly; wrappers like `bufio.Reader` add behaviour, and a buffered reader must live as long as its stream | shell | [001](001-2026-10-14-shell.md) |
| Modelling command results | A result carries output, exit code and "should stop". The exit status is a separate channel (`os.Exit`, `$?`). Handlers decide; the caller performs the side effect | shell | [001](001-2026-10-14-shell.md) |
| Testability as design feedback | Hard-to-test code is design feedback. Inject I/O, and test through the contract (behaviour), not internals | shell | [001](001-2026-10-14-shell.md) |
| When to refactor | Preparatory refactoring: refactor when a stage fights the design ("make the change easy, then make the easy change") | shell | [001](001-2026-10-14-shell.md) |
