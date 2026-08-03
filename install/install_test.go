package install

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/build"
	"github.com/wtsi-hgi/softpack/internal/apt"
)

func TestInstall(t *testing.T) {
	apt.SkipIfBadEnvironment(t)

	root := apt.CreateTestAptRepo(t, apt.ExamplePackages())
	moduleBase := t.TempDir()
	installBase := t.TempDir()
	artefactBase := t.TempDir()

	arts, err := Install(apt.BuildBase, moduleBase, "", installBase, "a-wrapper-script", artefactBase, "groups/myGroup", "myEnv", "1", root, "My Environment", []build.Package{
		{Name: "r-lib"},
		{Name: "abc", Version: "1"},
	})
	assert.NoError(t, err)

	assert.Equal(t, arts.Exes, []string{"R", "Rscript", "abc"})
	assert.Equal(t, arts.Packages, []build.Package{
		{Name: "r-lib", Version: "1.1"},
		{Name: "abc", Version: "1"},
		{Name: "r", Version: "4.4.0", Interpreter: true},
	})

	assert.FileExists(t, filepath.Join(moduleBase, "groups", "myGroup", "myEnv", "1"))
	assert.FileExists(t, filepath.Join(installBase, "groups", "myGroup", "myEnv", "1-scripts", "singularity.sif"))
	assert.FileExists(t, filepath.Join(installBase, "groups", "myGroup", "myEnv", "1-scripts", "abc"))
	assert.FileExists(t, filepath.Join(artefactBase, "groups", "myGroup", "myEnv", "1", "singularity.sif"))
	assert.FileExists(t, filepath.Join(artefactBase, "groups", "myGroup", "myEnv", "1", "build.log"))
	assert.FileExists(t, filepath.Join(artefactBase, "groups", "myGroup", "myEnv", "1", "module"))
}
