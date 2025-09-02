package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Contact struct {
	Auths Auths `yaml:"auths"`
}

type Auths struct {
	Spear *SpearAuth `yaml:"spear,omitempty"`
	HTTP  *HTTPAuth  `yaml:"http,omitempty"`
}

type SpearAuth struct {
	Fingerprint string `yaml:"fingerprint"`
}

type HTTPAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type SpearConfig struct {
	Contacts   map[string]Contact  `yaml:"contacts"`
	Identities map[string]Identity `yaml:"identities"`
}

type Identity struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

func LoadConfig(path string) (*SpearConfig, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal the YAML data
	var config SpearConfig
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &config, nil
}
