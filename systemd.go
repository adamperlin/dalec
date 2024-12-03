package dalec

import (
	"fmt"
	"maps"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/shell"
)

type SystemdConfiguration struct {
	// Units is a list of systemd units to include in the package.
	Units map[string]SystemdUnitConfig `yaml:"units,omitempty" json:"units,omitempty"`
	// Dropins is a list of systemd drop in files that should be included in the package
	Dropins map[string]SystemdDropinConfig `yaml:"dropins,omitempty" json:"dropins,omitempty"`
}

func (s *SystemdConfiguration) ProcessBuildArgs(lex *shell.Lex, args map[string]string) error {
	expandedUnits := make(map[string]SystemdUnitConfig, len(s.Units))
	expandedDropins := make(map[string]SystemdDropinConfig, len(s.Dropins))

	for name, unit := range s.Units {
		// mutates `unit`
		err := unit.processBuildArgs(lex, args)
		if err != nil {
			fmt.Errorf("failed to process build args for systemd unit %s: %w", name, err)
		}

		name, err = expandArgs(lex, name, args)
		if err != nil {
			return fmt.Errorf("failed to expand args for systemd unit name: %w", err)
		}

		expandedUnits[name] = unit
	}

	for name, dropin := range s.Dropins {
		// mutates `dropin`
		err := dropin.processBuildArgs(lex, args)
		if err != nil {
			fmt.Errorf("failed to process build args for systemd dropin %s: %w", name, err)
		}

		name, err = expandArgs(lex, name, args)
		if err != nil {
			return fmt.Errorf("failed to expand args for systemd dropin name: %w", err)
		}

		expandedDropins[name] = dropin
	}

	s.Units = expandedUnits
	s.Dropins = expandedDropins

	return nil
}

type SystemdUnitConfig struct {
	// Name is the name systemd unit should be copied under.
	// Nested paths are not supported. It is the user's responsibility
	// to name the service with the appropriate extension, i.e. .service, .timer, etc.
	Name string `yaml:"name,omitempty" json:"name"`

	// Enable is used to enable the systemd unit on install
	// This determines what will be written to a systemd preset file
	Enable bool `yaml:"enable,omitempty" json:"enable"`
}

func (s *SystemdUnitConfig) processBuildArgs(lex *shell.Lex, args map[string]string) error {
	if s.Name != "" {
		name, err := expandArgs(lex, s.Name, args)
		if err != nil {
			return fmt.Errorf("failed to expand args for systemd unit name: %w", err)
		}

		s.Name = name
	}

	return nil
}

func (s SystemdUnitConfig) Artifact() *ArtifactConfig {
	return &ArtifactConfig{
		SubPath: "",
		Name:    s.Name,
	}
}

func (s SystemdUnitConfig) ResolveName(name string) string {
	return s.Artifact().ResolveName(name)
}

// Splitname resolves a unit name and then gives its unit base name.
// E.g. for  `foo.socket` this would be `foo` and `socket`.
func (s SystemdUnitConfig) SplitName(name string) (string, string) {
	name = s.ResolveName(name)
	base, other, _ := strings.Cut(name, ".")
	return base, other
}

type SystemdDropinConfig struct {
	// Name is file or dir name to use for the artifact in the package.
	// If empty, the file or dir name from the produced artifact will be used.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	// Unit is the name of the systemd unit that the dropin files should be copied under.
	Unit string `yaml:"unit" json:"unit"` // the unit named foo.service maps to the directory foo.service.d
}

func (s *SystemdDropinConfig) processBuildArgs(lex *shell.Lex, args map[string]string) error {
	if s.Name != "" {
		name, err := expandArgs(lex, s.Name, args)
		if err != nil {
			return fmt.Errorf("failed to expand args for systemd dropin name: %w", err)
		}

		s.Name = name
	}

	if s.Unit != "" {
		unit, err := expandArgs(lex, s.Unit, args)
		if err != nil {
			return fmt.Errorf("failed to expand args for systemd dropin unit: %w", err)
		}
		s.Unit = unit
	}

	return nil
}

func (s SystemdDropinConfig) Artifact() *ArtifactConfig {
	return &ArtifactConfig{
		SubPath: fmt.Sprintf("%s.d", s.Unit),
		Name:    s.Name,
	}
}

func (s *SystemdConfiguration) GetUnits() map[string]SystemdUnitConfig {
	if s == nil {
		return nil
	}
	return maps.Clone(s.Units)
}

func (s *SystemdConfiguration) GetDropins() map[string]SystemdDropinConfig {
	if s == nil {
		return nil
	}
	return maps.Clone(s.Dropins)
}
