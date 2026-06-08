package config

import (
	"os"
	"path/filepath"
)

// GlobalFileName is the default global config file name under the user's home.
const GlobalFileName = ".opencommit.yaml"

// ProjectFileName is the per-project config file looked up in the working dir.
const ProjectFileName = ".opencommit.yaml"

// GlobalPath resolves the global config file path with this precedence:
//
//	explicit (flag --config) > OCO_CONFIG_PATH env > ~/.opencommit.yaml
//
// explicit may be empty to skip the flag layer.
func GlobalPath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if env := os.Getenv("OCO_CONFIG_PATH"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, GlobalFileName), nil
}

// ProjectPath returns the project-local config path (working dir +
// .opencommit.yaml), or "" if it does not exist.
func ProjectPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	p := filepath.Join(wd, ProjectFileName)
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

// Options controls how Resolve layers configuration sources.
type Options struct {
	// ConfigPath is the explicit global config path (from --config). Empty
	// falls back to OCO_CONFIG_PATH then ~/.opencommit.yaml.
	ConfigPath string
	// Flags are per-run flag overrides as key->value (highest precedence).
	Flags map[string]string
	// Env, when non-nil, is used instead of the process environment. Lets
	// tests inject env vars deterministically.
	Env func(string) (string, bool)
}

// Resolve builds the effective Config by layering, lowest to highest priority:
//
//	default < global file < project file < env vars < flags
//
// It returns the resolved Config and the global path it read (useful for
// subsequent saves).
func Resolve(opts Options) (Config, string, error) {
	globalPath, err := GlobalPath(opts.ConfigPath)
	if err != nil {
		return Config{}, "", err
	}

	cfg, err := Load(globalPath)
	if err != nil {
		return cfg, globalPath, err
	}

	if pp := ProjectPath(); pp != "" && pp != globalPath {
		if err := merge(&cfg, pp); err != nil {
			return cfg, globalPath, err
		}
	}

	// Overlay the active profile (resolved from file, then OCO_ACTIVE_PROFILE,
	// then --active-profile flag) onto the top-level provider fields, before the
	// per-key env/flag layers so an explicit OCO_MODEL etc. still wins.
	applyActiveProfile(&cfg, opts)

	if err := applyEnv(&cfg, opts.Env); err != nil {
		return cfg, globalPath, err
	}

	if err := applyFlags(&cfg, opts.Flags); err != nil {
		return cfg, globalPath, err
	}

	return cfg, globalPath, nil
}

// applyActiveProfile selects the effective profile name (file < env < flag) and
// overlays its fields onto cfg. Unknown names are ignored.
func applyActiveProfile(cfg *Config, opts Options) {
	fileActive := cfg.ActiveProfile
	name := fileActive
	lookup := opts.Env
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if v, ok := lookup("OCO_ACTIVE_PROFILE"); ok {
		name = v
	}
	if v, ok := opts.Flags["active_profile"]; ok {
		name = v
	}
	// Prefer the requested name; if it is unknown, fall back to the file's
	// active profile so an env/flag typo doesn't silently drop credentials.
	for _, candidate := range []string{name, fileActive} {
		if candidate == "" {
			continue
		}
		if p, ok := cfg.Profiles[candidate]; ok {
			cfg.ActiveProfile = candidate
			ApplyProfile(cfg, p)
			return
		}
	}
}

// applyEnv overlays OCO_-prefixed env vars onto cfg using the key registry.
// lookup defaults to os.LookupEnv when nil.
func applyEnv(cfg *Config, lookup func(string) (string, bool)) error {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	for _, k := range registry {
		// active_profile via env is handled (leniently) by applyActiveProfile,
		// which already ran; re-applying here would reject unknown names.
		if k.Key == "active_profile" {
			continue
		}
		if v, ok := lookup(k.Env); ok {
			if err := k.Set(cfg, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyFlags overlays flag overrides (key->value) onto cfg via the registry.
func applyFlags(cfg *Config, flags map[string]string) error {
	for key, v := range flags {
		if err := Set(cfg, key, v); err != nil {
			return err
		}
	}
	return nil
}
