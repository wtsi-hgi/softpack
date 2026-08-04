package module

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/build"
)

func TestWriteModuleFile(t *testing.T) {
	var sb strings.Builder

	installBase := "/software/modules/HGI/softpack"

	assert.NoError(t, writeModuleFile(&sb, installBase, "groups/myGroup", "myEnv", "1.2", "My Environment",
		[]string{"xxhsum", "xxh32sum", "xxh64sum", "xxh128sum", "R", "Rscript", "python"},
		[]build.Package{
			{Name: "xxhash", Version: "0.8.1"},
			{Name: "r-seurat", Version: "4"},
			{Name: "py-anndata", Version: "3.14"},
		},
	))

	assert.Equal(t, sb.String(), `#%Module

proc ModulesHelp { } {
	puts stderr "My Environment"
	puts stderr ""
	puts stderr "The following executables are added to your PATH:"
	puts stderr "  - xxhsum"
	puts stderr "  - xxh32sum"
	puts stderr "  - xxh64sum"
	puts stderr "  - xxh128sum"
	puts stderr "  - R"
	puts stderr "  - Rscript"
	puts stderr "  - python"
}

module-whatis "Name: myEnv"
module-whatis "Version: 1.2"
module-whatis "Packages: xxhash@0.8.1, r-seurat@4, py-anndata@3.14"

prepend-path PATH "/software/modules/HGI/softpack/groups/myGroup/myEnv/1.2-scripts"
`)
}
