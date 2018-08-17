package configutil

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/blend/go-sdk/env"
	"github.com/blend/go-sdk/exception"
	"github.com/blend/go-sdk/yaml"
)

const (
	// EnvVarConfigPath is the env var for configs.
	EnvVarConfigPath = "CONFIG_PATH"
	// ExtensionJSON is a file extension.
	ExtensionJSON = ".json"
	// ExtensionYAML is a file extension.
	ExtensionYAML = ".yaml"
	// ExtensionYML is a file extension.
	ExtensionYML = ".yml"
)

var (
	// DefaultPaths are default path locations.
	DefaultPaths = []string{
		"/var/secrets/config.yml",
		"/var/secrets/config.yaml",
		"/var/secrets/config.json",
		"./_config/config.yml",
		"./_config/config.yaml",
		"./_config/config.json",
	}
)

// Read reads a config from a default path (or inferred path from the environment).
func Read(ref Any, paths ...string) error {
	return TryReadFromPaths(ref, PathsWithDefaults(paths...)...)
}

// TryReadFromPaths tries to read the config from a list of given paths, reading from the first file that exists.
func TryReadFromPaths(ref Any, paths ...string) error {
	if len(paths) == 0 {
		return exception.New(ErrConfigPathUnset)
	}

	// for each of the paths
	// if the path doesn't exist, continue, read the path that is found.
	for _, path := range paths {
		if path == "" {
			continue
		}

		f, err := os.Open(path)
		if IsNotExist(err) {
			continue
		}
		if err != nil {
			return exception.New(err)
		}
		defer f.Close()

		return ReadFromReader(ref, f, filepath.Ext(path))
	}
	return exception.New(os.ErrNotExist).WithMessagef("no provided paths exist")
}

// ReadFromReader reads a config from a given reader.
func ReadFromReader(ref Any, r io.Reader, ext string) error {
	if err := Deserialize(ext, r, ref); err != nil {
		return err
	}
	return env.Env().ReadInto(ref)
}

// PathsWithDefaults returns the default paths and additional optional paths.
func PathsWithDefaults(paths ...string) []string {
	return Paths(append(DefaultPaths, paths...)...)
}

// Paths returns config paths.
// It is coalesced from a csv read from the env var 'CONFIG_PATH'
// and defaults passed into the function.
func Paths(defaults ...string) (output []string) {
	if env.Env().Has(EnvVarConfigPath) {
		output = append(output, env.Env().CSV(EnvVarConfigPath)...)
	}
	output = append(output, defaults...)
	return
}

// Deserialize deserializes a config.
func Deserialize(ext string, r io.Reader, ref Any) error {
	// make sure the extension starts with a "."
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// based off the extension, use the appropriate deserializer
	switch strings.ToLower(ext) {
	case ExtensionJSON:
		return exception.New(json.NewDecoder(r).Decode(ref))
	case ExtensionYAML, ExtensionYML:
		return exception.New(yaml.NewDecoder(r).Decode(ref))
	default: // return an error if we're passed a weird extension
		return exception.New(ErrInvalidConfigExtension).WithMessagef("extension: %s", ext)
	}
}
