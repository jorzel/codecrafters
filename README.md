# codecrafters

My solutions to [CodeCrafters](https://codecrafters.io) challenges, written in Go.
These are learning projects: the goal is understanding how these systems work
and practising design, not just passing the stages.

## Challenges

| Challenge | Progress | Folder |
|-----------|----------|--------|
| Build your own Redis | ![progress](https://backend.codecrafters.io/progress/redis/2d085b99-37e3-4738-8b64-f384b9d0a699) | [`codecrafters-redis-go`](codecrafters-redis-go) |
| Build your own Kafka | ![progress](https://backend.codecrafters.io/progress/kafka/03e9bb0f-de2f-4dc2-9b52-40f36279b03c) | [`codecrafters-kafka-go`](codecrafters-kafka-go) |
| Build your own DNS server | ![progress](https://backend.codecrafters.io/progress/dns-server/b6edd383-2865-4122-85ff-1cba252d2498) | [`codecrafters-dns-server-go`](codecrafters-dns-server-go) |
| Build your own Shell | ![progress](https://backend.codecrafters.io/progress/shell/1912fbb8-f146-414a-b293-20cfad6512bc) | [`codecrafters-shell-go`](codecrafters-shell-go) |

## Running a challenge

Each folder is a standalone Go module (Go 1.23+):

```sh
cd codecrafters-redis-go
./your_program.sh        # build and run locally
go test ./...            # unit tests
codecrafters test        # run the current stage's tests (requires the CodeCrafters CLI)
```

## Working with AI

AI is used as a mentor, not an author. Claude Code can discuss design, review
code, teach concepts and write tests on request, but it doesn't write the
solutions. The rules are in [`CLAUDE.md`](CLAUDE.md).

## Repo layout

Each challenge folder is also its own git repo that pushes to
`git.codecrafters.io`. See [`docs/git-workflow.md`](docs/git-workflow.md)
for how to commit changes and add new challenges.
