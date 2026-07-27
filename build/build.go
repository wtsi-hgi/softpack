package build

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Package struct {
	Name    string
	Version string
}

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

	sqfs := filepath.Join(tempDir, "singularity.sqfs")

	defer cleanup(root, sqfs)

	l, err := startServer(aptSrc)
	if err != nil {
		return nil, err
	}

	defer l.Close()

	return runCommands(root, baseImage, installDir, sqfs, l.Addr().String(), wrapperScript, pkgs)
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

	if err := installPackages(root, pkgs); err != nil {
		return nil, err
	}

	a, err := getArtefacts(root)
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
	return exec.Command(
		"singularity",
		"build",
		"--sandbox", root,
		baseImage,
	).Run()
}

func setAptRepo(root, httpURL string) error {
	return os.WriteFile(
		filepath.Join(root, "etc", "apt", "sources.list.d", "ubuntu.sources"),
		fmt.Appendf(nil, "deb [trusted=yes] http://%s/ main", httpURL),
		0644,
	)
}

const buildInstructions = "apt update && " +
	"apt -y -o DPkg::Options::=--force-not-root %[1]s && " +
	"for pkg in %[2]s; do " +
	`dpkg-query -W -f="$pkg@"'${Version}\n' "${pkg#'!'}";` +
	"done | sort > /pkgs && " +
	"for pkg in %[2]s; do " +
	`dpkg-query -W -f='${XB-Executables}\n' "${pkg#'!'}";` +
	"done | tr -d ' ' | tr ',' '\n' | sort | uniq > /exes &&" +
	"apt-get clean && rm -rf /var/lib/apt/lists && mkdir /var/lib/apt/lists;"

func installPackages(root string, pkgs []Package) error {
	return exec.Command(
		"singularity",
		"exec",
		"--writable",
		"--no-home", root,
		"bash", "-c", fmt.Sprintf(
			buildInstructions,
			packageListWithVersions(pkgs),
			packageListWithInterpreters(pkgs),
		),
	).Run()
}

func packageListWithVersions(pkgs []Package) string {
	list := make([]string, len(pkgs))

	for n, pkg := range pkgs {
		if pkg.Version != "" {
			list[n] = pkg.Name + "==" + pkg.Version
		} else {
			list[n] = pkg.Name
		}
	}

	return strings.Join(list, " ")
}

func packageListWithInterpreters(pkgs []Package) string {
	list := make([]string, len(pkgs))
	extra := make(map[string]bool)

	for n, pkg := range pkgs {
		list[n] = pkg.Name

		switch pkg.Name {
		case "r":
			extra["r"] = true
		case "python":
			extra["python"] = true
		}

		if strings.HasPrefix(pkg.Name, "r-") && !extra["r"] {
			extra["r"] = false
		} else if strings.HasPrefix(pkg.Name, "py-") && !extra["python"] {
			extra["python"] = false
		}
	}

	for pkg, exists := range extra {
		if !exists {
			list = append(list, "!"+pkg)
		}
	}

	return strings.Join(list, " ")
}

type Artefacts struct {
	Exes, Packages []string
}

func getArtefacts(root string) (*Artefacts, error) {
	exesPath := filepath.Join(root, "exes")
	pkgsPath := filepath.Join(root, "pkgs")

	exes, err := os.ReadFile(exesPath)
	if err != nil {
		return nil, err
	}

	pkgs, err := os.ReadFile(pkgsPath)
	if err != nil {
		return nil, err
	}

	if err := cmp.Or(os.Remove(exesPath), os.Remove(pkgsPath)); err != nil {
		return nil, err
	}

	return &Artefacts{
		Exes:     strings.Split(string(exes), "\n"),
		Packages: strings.Split(string(pkgs), "\n"),
	}, nil
}

func makeSquashFS(root, sqfs string) error {
	return exec.Command(
		"mksquashfs",
		root,
		sqfs,
		"-all-root",
	).Run()
}

func buildContainer(sqfs, installDir string) error {
	return exec.Command(
		"singularity",
		"build",
		filepath.Join(installDir, "singularity.sif"), sqfs,
	).Run()
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
