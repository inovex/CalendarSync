# Changelog

## 0.5.0

### Bug Fixes

- Refactor: Outlook Adapter Interface rename
  [!60](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/60)
  @jbraun
- Fix: Use Pointers for the Metadata to make sure it is actually empty [!55](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/55)
- Improvement: We now hash emails for better diffs, Emails won't get replaced by
  random strings anymore [!50](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/50)
- Christoph Petrausch contributed some internal io.Reader and writer chaining
  modifications (:wave: :sad:) [!47](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/47)
- We made the whole syncing / diffing process way more robust and are catching more
  edge cases! :tada: [!50](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/50)


### Features
- Transformers now have a predefined order, Users don't need to check for the order
  [!57](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/57)
- CalendarSync now has a `dry-run` option [!54](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/54)
- CalendarSync now has a `clean` option [!51](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/51)
- Authentication Credentials which we save on the disk are now encrypted using
  [age](https://github.com/FiloSottile/age)
  [!46](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/46)
  @cschaub
- We're now using Go Version 1.20 [!44](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/44)
- The browser now opens automatically for Authentication, if a valid env is detected
  [!49](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/49)


### Documentation

- Documentation is now improved and ready for OSS Release [!59](https://gitlab.inovex.de/inovex-calendarsync/calendarsync/-/merge_requests/)
