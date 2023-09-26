# Contributing

Thanks for your interest in improving Calendarsync.

We'd love some feedback or some input from the Open Source Community. Feel free
to open up a PR. This project is maintained by inovex employees mostly in their
freetime, so have mercy if a response to your PRs or issues may take a while -
we're doing our best :)

## Getting started

Clone repo, then:

```bash
make build
```

## To run without a build

```bash
CALENDARSYNC_ENCRYPTION_KEY=<yourPasswordForTheLocalAuthFile> go run cmd/calendarsync/main.go --config <yourConfigFile>
```

## Git commit messages

For proper semantic versioning, we use
[go-semrel-gitlab](https://juhani.gitlab.io/go-semrel-gitlab). Please use the
[documented format](https://juhani.gitlab.io/go-semrel-gitlab/commit-message/) to
write the commmit messages.

