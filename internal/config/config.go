// Package config implements INI-style configuration matching the Python
// configparser semantics used by the original stack-pr (SPEC.md §7).
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/victorhsb/branchless-pr/internal/git"
)

// Default path inside the repo for the config file.
const DefaultFilename = ".stack-pr.cfg"

// Config holds INI-style section/key/value state.
type Config struct {
	sections map[string]map[string]string // section -> key -> value (all lowercased)
	path     string
}

// FilePath returns the effective config file path.
// 1. $STACKPR_CONFIG if set.
// 2. <repo-root>/.stack-pr.cfg otherwise.
func FilePath() (string, error) {
	if p := os.Getenv("STACKPR_CONFIG"); p != "" {
		return p, nil
	}
	root, err := git.RepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, DefaultFilename), nil
}

// Load reads the config file at path, or returns an empty Config if it does
// not exist. Returns an error only for I/O failures.
func Load(path string) (*Config, error) {
	c := &Config{
		sections: make(map[string]map[string]string),
		path:     path,
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var section string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			if c.sections[section] == nil {
				c.sections[section] = make(map[string]string)
			}
			continue
		}
		if section == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(k))
		value := strings.TrimSpace(v)
		c.sections[section][key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return c, nil
}

// Save writes the current config back to the file path.
func (c *Config) Save() error {
	if c.path == "" {
		return fmt.Errorf("config: no path set")
	}
	f, err := os.Create(c.path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, section := range orderedKeysNested(c.sections) {
		fmt.Fprintf(w, "[%s]\n", section)
		for _, key := range orderedKeys(c.sections[section]) {
			fmt.Fprintf(w, "%s = %s\n", key, c.sections[section][key])
		}
		fmt.Fprintln(w)
	}
	return w.Flush()
}

// Get returns the raw string value, or empty string if missing.
func (c *Config) Get(section, key string) string {
	if s, ok := c.sections[strings.ToLower(section)]; ok {
		if v, ok := s[strings.ToLower(key)]; ok {
			return v
		}
	}
	return ""
}

// GetBool parses the value as a boolean matching Python configparser rules:
// true/yes/on/1 -> true; false/no/off/0 -> false. Case-insensitive.
func (c *Config) GetBool(section, key string) (bool, error) {
	raw := c.Get(section, key)
	if raw == "" {
		return false, nil
	}
	return parseBool(raw)
}

// Set stores a key/value in a section, creating the section if needed.
func (c *Config) Set(section, key, value string) {
	s := strings.ToLower(section)
	k := strings.ToLower(key)
	if c.sections[s] == nil {
		c.sections[s] = make(map[string]string)
	}
	c.sections[s][k] = value
}

// Has reports whether a key exists in the given section.
func (c *Config) Has(section, key string) bool {
	if s, ok := c.sections[strings.ToLower(section)]; ok {
		_, ok := s[strings.ToLower(key)]
		return ok
	}
	return false
}

// ParseConfigArg validates the `config` command argument form section.key=value.
func ParseConfigArg(arg string) (section, key, value string, err error) {
	rest, value, ok := strings.Cut(arg, "=")
	if !ok {
		return "", "", "", fmt.Errorf("invalid config format: expected <section>.<key>=<value>")
	}
	path := strings.SplitN(rest, ".", 2)
	if len(path) != 2 {
		return "", "", "", fmt.Errorf("invalid config format: expected <section>.<key>=<value>")
	}
	section = strings.TrimSpace(path[0])
	key = strings.TrimSpace(path[1])
	value = strings.TrimSpace(value)
	if section == "" || key == "" {
		return "", "", "", fmt.Errorf("invalid config format: expected <section>.<key>=<value>")
	}
	return section, key, value, nil
}

// Defaults returns a Config pre-populated with all documented defaults.
func Defaults() *Config {
	c := &Config{sections: make(map[string]map[string]string)}
	c.Set("common", "verbose", "false")
	c.Set("common", "hyperlinks", "true")
	c.Set("common", "draft", "false")
	c.Set("common", "keep_body", "false")
	c.Set("common", "stash", "false")
	c.Set("common", "show_tips", "true")
	c.Set("repo", "remote", "origin")
	c.Set("repo", "target", "main")
	c.Set("repo", "reviewer", "")
	c.Set("repo", "branch_name_template", "$USERNAME/stack")
	c.Set("comments", "ignore_authors", "")
	c.Set("land", "style", "bottom-only")
	return c
}

// Merge copies all keys from defaults that are not already present in c.
func (c *Config) Merge(defaults *Config) {
	for s, kv := range defaults.sections {
		for k, v := range kv {
			if !c.Has(s, k) {
				c.Set(s, k, v)
			}
		}
	}
}

func parseBool(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	trues := map[string]struct{}{"1": {}, "yes": {}, "true": {}, "on": {}}
	falses := map[string]struct{}{"0": {}, "no": {}, "false": {}, "off": {}}
	if _, ok := trues[s]; ok {
		return true, nil
	}
	if _, ok := falses[s]; ok {
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean value %q", s)
}

func orderedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Preserve insertion order is not feasible with a plain map;
	// sort for deterministic output.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func orderedKeysNested(m map[string]map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
