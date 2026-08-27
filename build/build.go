package build

import (
	"bufio"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"

	"github.com/wtsi-hgi/softpack/db"
	"pault.ag/go/debian/control"
)

const (
	singularitySQFS = "singularity.sqfs"
	singularitySIF  = "singularity.sif"
)

// Package represents a desired, top-level package to be installed in a
// container.
//
// The Version is optional, and Interpreter should only be set by this package
// to indicate an installed intepreter that wasn't explicitly chosen.
// type Package struct {
// 	Name        string
// 	Version     string
// 	Interpreter bool
// }

// Build constructs a singularity container, installing it into the desired
// location and creating wrapper symlinks to make running of executables inside
// the container simple.
//
// The returned artefacts contains the list of packages, and their installed
// versions, a list of exposed executables, and the build log.
//
// The supplied baseImage should be a docker://image, /path/to/image/dir/,
// /path/to/image.sif, or any other singularity source except a definition file.
//
// The tempDir will be the directory used to build the container; it will
// default to the install directory if not specified.
//
// The installDir will be where the final singularity.sif file is placed and
// where the wrapper symlinks will be created.
//
// The wrapperScript will be the target of the created utility symlinks; one
// made for each export executable.
//
// The aptSrc param can be an S3 location, an HTTP location, or an on-disk
// location.
//
// Pkgs is the list of desired packages to be installed inside the container.
//
// The returned artefacts will contain the list of packages with the
// as-installed versions specified, the list of exported executables, and the
// build log.
func Build(baseImage, tempDir, installDir, wrapperScript, aptSrc string, pkgs []db.Package) (*Artefacts, error) {
	if len(pkgs) == 0 {
		return nil, ErrNoPackages
	}

	if tempDir == "" {
		tempDir = installDir
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil { //nolint:mnd
		return nil, err
	}

	root, err := os.MkdirTemp(tempDir, "")
	if err != nil {
		return nil, err
	}

	sqfs := filepath.Join(tempDir, singularitySQFS)

	defer cleanup(root, sqfs)

	l, err := startServer(aptSrc)
	if err != nil {
		return nil, err
	}

	return runCommands(filepath.Join(root, "root"), baseImage, installDir, sqfs, l.Addr().String(), wrapperScript, pkgs)
}

func cleanup(root, sqfs string) {
	if err := os.Remove(sqfs); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Error("error removing temp sqfs file", "sqfs", sqfs, "err", err)
	}

	if err := os.RemoveAll(root); err != nil {
		slog.Error("error removing temp root", "root", root, "err", err)
	}
}

func runCommands( //nolint:funlen
	root, baseImage, installDir, sqfs, httpURL, wrapperScript string, pkgs []db.Package) (*Artefacts, error) {
	a := &Artefacts{}

	var log strings.Builder

	defer func() {
		a.Log = log.String()
	}()

	if err := extractImage(&log, root, baseImage); err != nil {
		return a, err
	}

	if err := setAptRepo(root, httpURL); err != nil {
		return a, err
	}

	executables, pkgs, err := installPackages(&log, root, pkgs)
	if err != nil {
		return a, err
	}

	a.Exes = executables
	a.Packages = pkgs

	if err := makeSquashFS(&log, root, sqfs); err != nil {
		return a, err
	}

	if err := buildContainer(&log, sqfs, installDir); err != nil {
		return a, err
	}

	if err := addWrappers(installDir, wrapperScript, a.Exes); err != nil {
		return a, err
	}

	return a, nil
}

func extractImage(log *strings.Builder, root, baseImage string) error {
	return runWithLog(log, exec.Command( //nolint:noctx
		"singularity",
		"build",
		"--sandbox", root,
		baseImage,
	))
}

func setAptRepo(root, httpURL string) error {
	return cmp.Or(
		os.Remove(filepath.Join(root, "etc", "apt", "sources.list.d", "ubuntu.sources")),
		os.WriteFile(
			filepath.Join(root, "etc", "apt", "sources.list.d", "ubuntu.list"),
			fmt.Appendf(nil, "deb [trusted=yes] http://%s resolute main", httpURL),
			0600, //nolint:mnd
		),
	)
}

func installPackages(log *strings.Builder, root string, pkgs []db.Package) ([]string, []db.Package, error) {
	packages := make([]string, len(pkgs))

	for n, pkg := range pkgs {
		if pkg.Version != "" {
			packages[n] = pkg.Name + "=" + pkg.Version
		} else {
			packages[n] = pkg.Name
		}
	}

	if err := cmp.Or(
		aptWithLog(log, root, "update"),
		aptWithLog(log, root, append(
			[]string{
				"-y",
				"-o", "DPkg::Options::=--force-not-root",
				"--allow-downgrades", "--allow-change-held-packages",
				"--allow-remove-essential", "--no-strict-pinning",
				"install"},
			packages...,
		)...),
		os.RemoveAll(filepath.Join(root, "var", "cache", "apt")),
		os.RemoveAll(filepath.Join(root, "var", "lib", "apt")),
	); err != nil {
		return nil, nil, err
	}

	return getArtefacts(root, pkgs)
}

func aptWithLog(log *strings.Builder, root string, args ...string) error {
	cmd := exec.Command( //nolint:gosec
		"singularity",
		append([]string{
			"exec", "--writable", "--no-home", root, "apt",
		}, args...)...,
	)

	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")

	return runWithLog(log, cmd)
}

func runWithLog(log *strings.Builder, cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}
	cmd.Stdout = log
	cmd.Stderr = log

	return cmd.Run()
}

// Artefacts contains the contains the list of executables exposed by the built
// container; the list of Packages, including installed versions; and the build
// log.
type Artefacts struct {
	Exes     []string
	Packages []db.Package
	Log      string
}

func getArtefacts(root string, pkgs []db.Package) ([]string, []db.Package, error) {
	f, err := os.Open(filepath.Join(root, "var", "lib", "dpkg", "status"))
	if err != nil {
		return nil, nil, err
	}

	installed, err := control.ParseBinaryIndex(bufio.NewReader(f))
	if err != nil {
		return nil, nil, err
	}

	ps, executables := getExecutablesAndConcretise(pkgs, installed)

	return executables, ps, nil
}

func getExecutablesAndConcretise(pkgs []db.Package, installed []control.BinaryIndex) ([]db.Package, []string) {
	pkgs = addInterpreters(pkgs)
	exes := map[string]struct{}{}

	for _, deb := range installed {
		idx := slices.IndexFunc(pkgs, func(v db.Package) bool {
			return v.Name == deb.Package
		})
		if idx < 0 {
			continue
		}

		if alias, ok := deb.Values["XB-Alias"]; ok {
			pkgs[idx].Name = alias
		}

		pkgs[idx].Version = deb.Version.Version

		if pkgExes, ok := deb.Values["XB-Executables"]; ok {
			for exe := range strings.SplitSeq(pkgExes, ", ") {
				exes[exe] = struct{}{}
			}
		}
	}

	executables := slices.Collect(maps.Keys(exes))
	slices.Sort(executables)

	return pkgs, executables
}

func addInterpreters(pkgs []db.Package) []db.Package { //nolint:gocognit,gocyclo,cyclop
	var hasPy, hasPython, hasRLib, hasR bool

	for _, pkg := range pkgs {
		if pkg.Name == "r" { //nolint:gocritic,nestif
			hasR = true
		} else if pkg.Name == "python" { //nolint:goconst
			hasPython = true
		} else if strings.HasPrefix(pkg.Name, "r-") {
			hasRLib = true
		} else if strings.HasPrefix(pkg.Name, "py-") {
			hasPy = true
		} else {
			continue
		}

		if hasPy && hasPython && hasR && hasRLib {
			break
		}
	}

	if hasPy && !hasPython {
		pkgs = append(pkgs, db.Package{Name: "python3", Interpreter: true})
	}

	if hasRLib && !hasR {
		pkgs = append(pkgs, db.Package{Name: "r-base-core", Interpreter: true})
	}

	return pkgs
}

func makeSquashFS(log *strings.Builder, root, sqfs string) error {
	return runWithLog(log, exec.Command( //nolint:noctx
		"mksquashfs",
		root,
		sqfs,
		"-all-root",
	))
}

func buildContainer(log *strings.Builder, sqfs, installDir string) error {
	sif := SingularityPath(installDir)

	return cmp.Or(
		runWithLog(log, exec.Command("singularity", "sif", "new", sif)), //nolint:noctx
		runWithLog(log, exec.Command( //nolint:noctx,gosec
			"singularity",
			"sif",
			"add",
			"--datatype", "4",
			"--parttype", "1",
			"--partfs", "1",
			"--partarch", arch[runtime.GOARCH],
			SingularityPath(installDir), sqfs,
		)),
		runWithLog(log, exec.Command("singularity", "sif", "setprim", "1", sif)), //nolint:noctx
	)
}

var arch = map[string]string{
	"386":      "1",
	"amd64":    "2",
	"arm":      "3",
	"arm64":    "4",
	"ppc64":    "5",
	"ppc64le":  "6",
	"mips":     "7",
	"mipsle":   "8",
	"mips64":   "9",
	"mips64le": "10",
	"s390x":    "11",
	"riscv64":  "12",
}

// SingularityPath appends the singularity filename to the given path.
func SingularityPath(installDir string) string {
	return filepath.Join(installDir, singularitySIF)
}

func addWrappers(installDir, wrapperScript string, exes []string) error {
	for _, exe := range exes {
		if err := os.Symlink(wrapperScript, filepath.Join(installDir, exe)); err != nil {
			return err
		}
	}

	return nil
}

var ErrNoPackages = errors.New("no packages specified")
