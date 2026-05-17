## Tasks

## 1. internal/stack

- [ ] 1.1 Add `MarshalJSON()` method to `Entry` producing flat JSON
- [ ] 1.2 Add `ToJSON()` method to `Stack` returning `[]byte`
- [ ] 1.3 Add unit test: verify JSON output shape for a single entry

## 2. internal/cli/view.go

- [ ] 2.1 Add `--format` string flag with default `"text"`
- [ ] 2.2 Branch on format: text -> `PrintStack`, json -> print `ToJSON()`
- [ ] 2.3 Add format validation error for unknown values

## 3. E2E test

- [ ] 3.1 Add tests for `view` JSON output with `--format json`
- [ ] 3.2 Add tests for invalid format rejection
