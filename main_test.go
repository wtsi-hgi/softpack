package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wtsi-hgi/softpack/config"
	"github.com/wtsi-hgi/softpack/internal/apt"
	"gopkg.in/yaml.v3"
)

func TestCommands(t *testing.T) {
	tmpDir := t.TempDir()

	appExe := filepath.Join(tmpDir, "softpack")

	err := exec.Command("go", "build", "-o", tmpDir).Run()
	require.NoError(t, err)

	out, err := exec.Command(appExe, "server").CombinedOutput()
	assert.Error(t, err)
	assert.Contains(t, string(out), `Error: required flag(s) "config" not set`)

	conf := filepath.Join(t.TempDir(), "config.yaml")
	f, err := os.Create(conf)
	require.NoError(t, err)

	root := apt.CreateTestAptRepo(t, apt.ExamplePackages())
	moduleBase := t.TempDir()
	installBase := t.TempDir()
	artefactBase := t.TempDir()
	dbConn := filepath.Join(t.TempDir(), "db")

	err = yaml.NewEncoder(f).Encode(&config.Config{
		BaseImgPath:   apt.BuildBase,
		ModulePath:    moduleBase,
		TempDir:       "",
		InstallDir:    installBase,
		WrapperScript: "a-wrapper-script",
		AptSrc:        root,
		AptIndexSrc:   filepath.Join(root, "dists", "resolute", "main", "binary-"+runtime.GOARCH, "Packages"),
		ArtefactStore: artefactBase,
		DBConn:        dbConn,
		Driver:        "sqlite3",
		ListenAddr:    ":1025",
	})
	assert.NoError(t, err)

	err = f.Close()
	assert.NoError(t, err)

	cmd := exec.Command(appExe, "server", "--config", conf)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	assert.NoError(t, err)
	assert.NotNil(t, cmd.Process)

	var resp *http.Response

	u, err := user.Current()
	assert.NoError(t, err)

	for wait := 1; wait < 5; wait++ {
		body, err := json.Marshal(u.Username) //nolint:govet
		require.NoError(t, err)

		resp, err = http.Post("http://localhost:1025/groups", "application/json", bytes.NewReader(body))
		if err != nil {
			time.Sleep(time.Duration(wait*2) * time.Second)

			continue
		}

		break
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	var body []string

	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)

	assert.Contains(t, body, "ubuntu")

	err = cmd.Process.Signal(syscall.SIGINT)
	require.NoError(t, err)

	err = cmd.Wait()
	require.NoError(t, err)
}
