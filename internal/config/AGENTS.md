# internal/config

Loads the API key and base URL.

## Gotchas

- Precedence is environment, then `config.toml`, then the default base URL
  (`DefaultBaseURL`). The API key has no default: its absence is an error that prints an example
  file, so keep that message complete and free of a trailing newline (the linter checks).
- `Load` and `DefaultPath` take `getenv` and the home-dir lookup as arguments. Keep it that way:
  tests rely on it instead of `t.Setenv`.
- A missing file is fine; a malformed one is an error that names the path.
