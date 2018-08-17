package configutil

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/blend/go-sdk/assert"
	"github.com/blend/go-sdk/env"
	"github.com/blend/go-sdk/exception"
	"github.com/blend/go-sdk/uuid"
)

type config struct {
	Environment string `json:"env" yaml:"env" env:"SERVICE_ENV"`
	Other       string `json:"other" yaml:"other" env:"OTHER"`
}

func TestPathsWithDefaults(t *testing.T) {
	assert := assert.New(t)

	assert.Len(PathsWithDefaults(), 6)
	assert.Equal("/var/secrets/config.yml", PathsWithDefaults()[0])
	assert.Equal("./_config/config.json", PathsWithDefaults()[5])
	assert.Equal("foo.yml", PathsWithDefaults("foo.yml")[6])
}

func TestPaths(t *testing.T) {
	assert := assert.New(t)
	defer env.Restore()
	env.Env().Delete(EnvVarConfigPath)

	assert.Empty(Paths())
	assert.Equal([]string{"testdata/foo.yml"}, Paths("testdata/foo.yml"))
	assert.Equal([]string{"testdata/foo.yml"}, Paths("testdata/foo.yml"))
}

func TestPathEnv(t *testing.T) {
	assert := assert.New(t)
	defer env.Restore()
	env.Env().Set(EnvVarConfigPath, "testdata/config.yml,testdata/alt.yml")

	assert.Equal([]string{"testdata/config.yml", "testdata/alt.yml"}, Paths())
	assert.Equal([]string{"testdata/config.yml", "testdata/alt.yml", "foo.yml"}, Paths("foo.yml"))
	assert.Equal([]string{"testdata/config.yml", "testdata/alt.yml", "foo.yml", "bar.yml"}, Paths("foo.yml", "bar.yml"))
}

func TestDeserializeInvalid(t *testing.T) {
	assert := assert.New(t)

	err := Deserialize(".???", nil, nil)
	assert.NotNil(err)
	assert.True(IsInvalidConfigExtension(err))
}

func TestDeserializeYAML(t *testing.T) {
	assert := assert.New(t)

	var cfg config
	assert.Nil(Deserialize(ExtensionYAML, bytes.NewBuffer([]byte("env: test\nother: foo")), &cfg))
	assert.Equal("test", cfg.Environment)
	assert.Equal("foo", cfg.Other)
}

func TestDeserializeYML(t *testing.T) {
	assert := assert.New(t)

	var cfg config
	assert.Nil(Deserialize(ExtensionYML, bytes.NewBuffer([]byte("env: test\nother: foo")), &cfg))
	assert.Equal("test", cfg.Environment)
	assert.Equal("foo", cfg.Other)
}

func TestDeserializeAddsPrefix(t *testing.T) {
	assert := assert.New(t)

	var cfg config
	assert.Nil(Deserialize("yml", bytes.NewBuffer([]byte("env: test\nother: foo")), &cfg))
	assert.Equal("test", cfg.Environment)
	assert.Equal("foo", cfg.Other)
}

func TestDeserializeJSON(t *testing.T) {
	assert := assert.New(t)

	var cfg config
	assert.Nil(Deserialize(ExtensionJSON, bytes.NewBuffer([]byte(`{"env": "test", "other": "foo"}`)), &cfg))
	assert.Equal("test", cfg.Environment)
	assert.Equal("foo", cfg.Other)
}

func TestReadFromReader(t *testing.T) {
	assert := assert.New(t)
	defer env.Restore()

	env.Env().Set(env.VarServiceEnv, "dev")
	var cfg config
	err := ReadFromReader(&cfg, bytes.NewBuffer([]byte("env: test\nother: foo")), ExtensionYAML)
	assert.Nil(err)
	assert.Equal("dev", cfg.Environment)
}

func TestTryReadFromPathYAML(t *testing.T) {
	assert := assert.New(t)

	var cfg config
	err := TryReadFromPaths(&cfg, "testdata/config.yaml")
	assert.Nil(err)
	assert.Equal("test_yaml", cfg.Environment)
	assert.Equal("foo", cfg.Other)
}

func TestTryReadFromPathYML(t *testing.T) {
	assert := assert.New(t)

	var cfg config
	err := TryReadFromPaths(&cfg, "testdata/config.yml")
	assert.Nil(err)
	assert.Equal("test_yml", cfg.Environment)
	assert.Equal("foo", cfg.Other)
}

func TestTryReadFromPathJSON(t *testing.T) {
	assert := assert.New(t)

	var cfg config
	err := TryReadFromPaths(&cfg, "testdata/config.json")
	assert.Nil(err)
	assert.Equal("test_json", cfg.Environment)
	assert.Equal("moo", cfg.Other)
}

func TestReadPathUnset(t *testing.T) {
	assert := assert.New(t)
	defer env.Restore()

	env.Env().Set(env.VarServiceEnv, "dev")
	var cfg config
	err := TryReadFromPaths(&cfg, "")
	assert.True(IsNotExist(err))
	assert.NotEqual("dev", cfg.Environment)
}

func TestReadPathNotFound(t *testing.T) {
	assert := assert.New(t)

	var cfg config
	err := TryReadFromPaths(&cfg, filepath.Join("testdata", uuid.V4().String()))
	assert.True(IsNotExist(err))
}

func TestIsUnset(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsConfigPathUnset(exception.New(ErrConfigPathUnset)))
	assert.False(IsConfigPathUnset(exception.New(uuid.V4().String())))
}

func TestIsIgnored(t *testing.T) {
	assert := assert.New(t)
	assert.True(IsIgnored(nil))
	assert.True(IsIgnored(exception.New(ErrConfigPathUnset)))
	assert.True(IsIgnored(exception.New(ErrInvalidConfigExtension)))
}
