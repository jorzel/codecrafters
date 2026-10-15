# Concepts

A living index of broad concepts across sessions, sorted alphabetically.
When a concept comes up again, add a session link and sharpen its summary.
Details stay in the session files.

| Concept | Summary | Challenge | Sessions |
|---------|---------|-----------|----------|
| Architecture of an interpreter | Keep logic pure (functional core) and push I/O to the edge (imperative shell). Layers map onto a shell, but parsing is domain, because the language is the product | shell | [001](001-2026-10-14-shell.md) |
| Command lookup in a shell | Builtins first, then PATH in order; a match is a regular file with an execute bit. PATH is user data, so failures mean "try the next entry". bash caches lookups lazily, zsh can fill its table up front | shell | [002](002-2026-10-15-shell.md) |
| Go's I/O model | `io.Reader`/`io.Writer` are one-method interfaces satisfied implicitly; wrappers like `bufio.Reader` add behaviour, and a buffered reader must live as long as its stream | shell | [001](001-2026-10-14-shell.md) |
| Interfaces and dependency injection in Go | The consumer owns the interface ("accept interfaces, return structs"); implicit satisfaction connects them; `main` is the composition root. Inject a capability, not a snapshot of data | shell | [002](002-2026-10-15-shell.md) |
| Modelling command results | A result carries output, exit code and "should stop". The exit status is a separate channel (`os.Exit`, `$?`). Handlers decide; the caller performs the side effect | shell | [001](001-2026-10-14-shell.md) |
| Package and API design | Name packages after what they provide, avoid stutter, export only what callers need. Deep modules over shallow layers; comma-ok over sentinels when absence is possible | shell | [002](002-2026-10-15-shell.md) |
| Testability as design feedback | Hard-to-test code is design feedback. Inject I/O and dependencies, test through the contract (external `_test` packages), and fake only at boundaries | shell | [001](001-2026-10-14-shell.md), [002](002-2026-10-15-shell.md) |
| When to refactor | Preparatory refactoring: refactor when a stage fights the design ("make the change easy, then make the easy change"). Extract a package when a stage forces that boundary, not up front | shell | [001](001-2026-10-14-shell.md), [002](002-2026-10-15-shell.md) |
