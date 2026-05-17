## Tasks

## 1. internal/stack

- [x] 1.1 Add `MarshalJSON()` method to `Entry` producing flat JSON
- [x] 1.2 Add `ToJSON()` method to `Stack` returning `[]byte`
- [x] 1.3 Add unit test: verify JSON output shape for a single entry

## 2. internal/cli/view.go

- [x] 2.1 Add `--format` string flag with default `"text"`
- [x] 2.2 Branch on format: text -> `PrintStack`, json -> print `ToJSON()`
- [x] 2.3 Add format validation error for unknown values

## 3. E2E test

- [x] 3.1 Add tests for `view` JSON output with `--format json`
- [x] 3.2 Add tests for invalid format rejection
