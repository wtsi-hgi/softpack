package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()

	configYAML := `
base-img-path:
install-dir: 
wrapper-script: 

module-path: 
artefact-store: 
db-conn: test.db
listen-addr: 
`

	path := filepath.Join(dir, "config.yaml")
	assert.NoError(t, os.WriteFile(path, []byte(configYAML), 0644)) //nolint:gosec

	cfg, err := Load(path)
	assert.NoError(t, err)

	assert.Equal(t, "8080", cfg.ListenAddr)
	assert.Equal(t, "test.db", cfg.DBConn)
}
