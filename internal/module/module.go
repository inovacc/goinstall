package module

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const dummyModuleName = "dummy"

type Module struct {
	ctx          context.Context
	fs           afero.Fs
	goBinPath    string
	timeout      time.Duration
	Time         time.Time    `json:"time"`
	Name         string       `json:"name"`
	Hash         string       `json:"hash"`
	Version      string       `json:"version"`
	Versions     []string     `json:"versions"`
	Dependencies []Dependency `json:"dependencies"`
}

type Dependency struct {
	Name         string       `json:"name"`
	Hash         string       `json:"hash"`
	Version      string       `json:"version"`
	Versions     []string     `json:"versions"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

type ListResp struct {
	Time     time.Time `json:"time"`
	Path     string    `json:"path"`
	Version  string    `json:"version"`
	Versions []string  `json:"versions,omitempty"`
}

func NewModule(ctx context.Context, afs afero.Fs, goBinPath string) (*Module, error) {
	if err := validGoBinary(goBinPath); err != nil {
		return nil, err
	}
	return &Module{
		ctx:          ctx,
		fs:           afs,
		goBinPath:    goBinPath,
		Dependencies: make([]Dependency, 0),
	}, nil
}

func (m *Module) Check(module string) error {
	tmpDir, err := afero.TempDir(m.fs, "", "go-list")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func(fs afero.Fs, path string) {
		cobra.CheckErr(fs.RemoveAll(path))
	}(m.fs, tmpDir)

	ctx, cancel := context.WithTimeout(m.ctx, m.getTimeout())
	defer cancel()

	module, version := splitModuleVersion(module)
	m.Name = module

	// Get versions from upstream
	lr, err := m.fetchModuleVersions(ctx, tmpDir, module)
	if err != nil {
		return err
	}

	m.Versions = lr.Versions
	m.Version = pickVersion(version, lr.Versions)
	m.Time = time.Now()
	m.Hash = sha256Hash(fmt.Sprintf("%s@%s", module, version))

	// Setup dummy mod
	if err := m.setupTempModule(ctx, tmpDir); err != nil {
		return err
	}

	// Install target module in dummy
	if err := m.getModule(ctx, tmpDir, fmt.Sprintf("%s@%s", module, m.Version)); err != nil {
		return err
	}

	// Extract dependencies
	m.Dependencies, err = m.extractDependencies(ctx, tmpDir, module)
	return err
}

func (m *Module) dependency(module string) (*Dependency, error) {
	tmpDir, err := afero.TempDir(m.fs, "", "go-list")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func(fs afero.Fs, path string) {
		cobra.CheckErr(fs.RemoveAll(path))
	}(m.fs, tmpDir)

	ctx, cancel := context.WithTimeout(m.ctx, m.getTimeout())
	defer cancel()

	name, suffix := splitModuleVersion(module)

	lr, err := m.fetchModuleVersions(ctx, tmpDir, name)
	if err != nil {
		return nil, err
	}

	version := pickVersion(suffix, lr.Versions)
	return &Dependency{
		Name:     name,
		Hash:     sha256Hash(fmt.Sprintf("%s@%s", name, version)),
		Version:  version,
		Versions: lr.Versions,
	}, nil
}

func (m *Module) InstallModule(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "install", fmt.Sprintf("%s@latest", m.Name))
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOBIN=%s/bin", os.Getenv("GOPATH")))
	return cmd.Run()
}

func (m *Module) fetchModuleVersions(ctx context.Context, dir, module string) (*ListResp, error) {
	cmd := exec.CommandContext(ctx, m.goBinPath, "list", "-m", "-versions", "-json", fmt.Sprintf("%s@latest", module))
	cmd.Dir = dir

	var lr ListResp
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}
	if err := json.NewDecoder(&out).Decode(&lr); err != nil {
		return nil, fmt.Errorf("decoding list response failed: %w", err)
	}

	sort.Slice(lr.Versions, func(i, j int) bool {
		return semver.Compare(lr.Versions[i], lr.Versions[j]) > 0
	})
	return &lr, nil
}

func (m *Module) setupTempModule(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, m.goBinPath, "mod", "init", dummyModuleName)
	cmd.Dir = dir
	return cmd.Run()
}

func (m *Module) getModule(ctx context.Context, dir, moduleWithVersion string) error {
	cmd := exec.CommandContext(ctx, m.goBinPath, "get", moduleWithVersion)
	cmd.Dir = dir
	return cmd.Run()
}

func (m *Module) extractDependencies(ctx context.Context, dir, self string) ([]Dependency, error) {
	cmd := exec.CommandContext(ctx, m.goBinPath, "list", "-m", "all")
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list -m all failed: %w", err)
	}

	var deps []Dependency
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] == dummyModuleName || fields[0] == self {
			continue
		}
		dep, err := m.dependency(fields[0])
		if err == nil {
			deps = append(deps, *dep)
		}
	}
	return deps, nil
}

func (m *Module) getTimeout() time.Duration {
	if m.timeout == 0 {
		return 10 * time.Second
	}
	return m.timeout
}

func splitModuleVersion(full string) (string, string) {
	parts := strings.SplitN(full, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return full, ""
}

func sha256Hash(input string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(input)))
}

func pickVersion(preferred string, versions []string) string {
	if preferred != "" {
		return preferred
	}
	if len(versions) > 0 {
		return versions[0]
	}
	return ""
}
