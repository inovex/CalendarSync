# Contributing

Thanks for you interest in improving Calendarsync.

## Getting started

Clone repo, then:

```bash
go mod init
go build -o calendarsync cmd/calendarsync/main.go
```

## To run without a build

```bash
go run cmd/calendarsync/main.go --config <yourConfigFile>
```

## Git commit messages

For proper semantic versioning, we use
[go-semrel-gitlab](https://juhani.gitlab.io/go-semrel-gitlab). Please use the
[documented format](https://juhani.gitlab.io/go-semrel-gitlab/commit-message/) to
write the commmit messages.
