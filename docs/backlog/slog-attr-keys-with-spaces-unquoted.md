---
worth: no
where: slog.go:formatAttr
added: 2026-11-25
---
# ToSlogHandler writes attribute keys with spaces unquoted

`ToSlogHandler` writes keys as they are, so `slog.Int("plain key", 1)` comes out as `plain key=1` and a
grouped `slog.Group("request", slog.Int("retry count", 2))` as `request.retry count=2`, where the slog text
handler writes `"plain key"=1` and `"request.retry count"=2`. A reader splitting on spaces loses the field.
Found while reviewing PR #34 (where the function becomes `writeAttr`); same on master de23384.

Not worth it: keys with spaces go against slog's naming conventions and are rare, and lgr's line is text
for people to read. Quoting keys the way slog's text handler does would change output for every caller that
relies on today's form, for a case nobody has reported.
