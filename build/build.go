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
type Package struct {
	Name        string
	Version     string
	Interpreter bool
}

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
func Build(baseImage, tempDir, installDir, wrapperScript, aptSrc string, pkgs []Package) (*Artefacts, error) {
	if len(pkgs) == 0 {
		return nil, ErrNoPackages
	}

	if tempDir == "" {
		tempDir = installDir
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

func runCommands(root, baseImage, installDir, sqfs, httpURL, wrapperScript string, pkgs []Package) (*Artefacts, error) {
	if err := extractImage(root, baseImage); err != nil {
		return nil, err
	}

	if err := setAptRepo(root, httpURL); err != nil {
		return nil, err
	}

	a, err := installPackages(root, pkgs)
	if err != nil {
		return nil, err
	}

	if err := makeSquashFS(root, sqfs); err != nil {
		return nil, err
	}

	if err := buildContainer(sqfs, installDir); err != nil {
		return nil, err
	}

	if err := addWrappers(installDir, wrapperScript, a.Exes); err != nil {
		return nil, err
	}

	return a, nil
}

func extractImage(root, baseImage string) error {
	if err := exec.Command( //nolint:noctx
		"singularity",
		"build",
		"--sandbox", root,
		baseImage,
	).Run(); err != nil {
		return fmt.Errorf("error extracting base image: %w", err)
	}

	return nil
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

func installPackages(root string, pkgs []Package) (*Artefacts, error) {
	packages := make([]string, len(pkgs))

	for n, pkg := range pkgs {
		if pkg.Version != "" {
			packages[n] = pkg.Name + "=" + pkg.Version
		} else {
			packages[n] = pkg.Name
		}
	}

	var log strings.Builder

	if err := cmp.Or(
		aptWithLog(&log, root, "update"),
		aptWithLog(&log, root, append(
			[]string{"-y", "-o", "DPkg::Options::=--force-not-root", "install"},
			packages...,
		)...),
		os.RemoveAll(filepath.Join(root, "var", "cache", "apt")),
		os.RemoveAll(filepath.Join(root, "var", "lib", "apt")),
	); err != nil {
		return &Artefacts{Log: log.String()}, err
	}

	return getArtefacts(root, pkgs, log.String())
}

func aptWithLog(log *strings.Builder, root string, args ...string) error {
	cmd := exec.Command( //nolint:noctx,gosec
		"singularity",
		append([]string{
			"exec", "--writable", "--no-home", root, "apt",
		}, args...)...,
	)
	cmd.Stdout = log
	cmd.Stderr = log

	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")

	return cmd.Run()
}

// Artefacts contains the contains the list of executables exposed by the built
// container; the list of Packages, including installed versions; and the build
// log.
type Artefacts struct {
	Exes     []string
	Packages []Package
	Log      string
}

func getArtefacts(root string, pkgs []Package, log string) (*Artefacts, error) {
	f, err := os.Open(filepath.Join(root, "var", "lib", "dpkg", "status"))
	if err != nil {
		return nil, err
	}

	installed, err := control.ParseBinaryIndex(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	ps, executables := getExecutablesAndConcretise(pkgs, installed)

	return &Artefacts{
		Exes:     executables,
		Packages: ps,
		Log:      log,
	}, nil
}

func getExecutablesAndConcretise(pkgs []Package, installed []control.BinaryIndex) ([]Package, []string) {
	pkgs = addInterpreters(pkgs)
	exes := map[string]struct{}{}

	for _, deb := range installed {
		idx := slices.IndexFunc(pkgs, func(v Package) bool { return v.Name == deb.Package })
		if idx < 0 {
			continue
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

func addInterpreters(pkgs []Package) []Package { //nolint:gocognit,gocyclo,cyclop
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
		pkgs = append(pkgs, Package{Name: "python", Interpreter: true})
	}

	if hasRLib && !hasR {
		pkgs = append(pkgs, Package{Name: "r", Interpreter: true})
	}

	return pkgs
}

func makeSquashFS(root, sqfs string) error {
	return exec.Command( //nolint:noctx
		"mksquashfs",
		root,
		sqfs,
		"-all-root",
	).Run()
}

func buildContainer(sqfs, installDir string) error {
	sif := SingularityPath(installDir)

	return cmp.Or(
		exec.Command("singularity", "sif", "new", sif).Run(), //nolint:noctx,gosec
		exec.Command( //nolint:noctx,gosec
			"singularity",
			"sif",
			"add",
			"--datatype", "4",
			"--parttype", "1",
			"--partfs", "1",
			"--partarch", arch[runtime.GOARCH],
			SingularityPath(installDir), sqfs,
		).Run(),
		exec.Command("singularity", "sif", "setprim", "1", sif).Run(), //nolint:noctx,gosec
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
