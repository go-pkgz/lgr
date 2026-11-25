---
worth: yes
where: logger.go:logf
added: 2026-11-25
---
# secret ending in a newline is not masked after a level prefix

`logf` parses the level first, and the parser trims whitespace around the text after a level prefix. Secrets
are hidden after that, so a secret that ends in whitespace, such as a PEM key ending in `\n`, no longer
matches when it is the last thing in a prefixed message.

- With `lgr.New(lgr.SlogHandler(h), lgr.Secret(pem)).Logf("WARN loaded %s", pem)` the handler gets the key
  unmasked, since the message is the only text hidden on that path. `slog.New(lgr.ToSlogHandler(thatLogger))`
  leaks the same way. The PANIC banner uses the same message.
- On lgr's own output the final pass over the whole line matches the secret together with the line's
  trailing newline: the key is masked, but the newline goes with it and the next log line is joined on.

Same on master de23384. Found while reviewing PR #34, which fixed the case it introduced in `Fatalf` by
hiding secrets before adding the `FATAL` prefix.

Fix constraint: hide secrets in the untrimmed text, or before trimming, so a secret's trailing whitespace
still matches; level detection must still work, and the final-line pass must not consume the newline lgr
adds. With that in `logf`, the extra hiding in `Fatalf` is no longer needed.
