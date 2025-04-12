package module

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/spf13/afero"
	"golang.org/x/mod/semver"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Module struct {
	ctx       context.Context
	fs        afero.Fs
	goBinPath string
	timeout   time.Duration
	Name      string `json:"name"`
	Hash      string `json:"hash"`
	Version   string `json:"version"`
	//Dependencies []string `json:"dependencies"`
	Versions []string `json:",omitempty"`
	Time     time.Time
}

func NewModule(ctx context.Context, afs afero.Fs, goBinPath string) (*Module, error) {
	if err := validGoBinary(goBinPath); err != nil {
		return nil, err
	}

	return &Module{
		ctx:       ctx,
		fs:        afs,
		goBinPath: goBinPath,
	}, nil
}

type ListResp struct {
	Path     string
	Version  string
	Versions []string `json:",omitempty"`
	Time     time.Time
}

func (m *Module) Check(module string) error {
	var lr ListResp

	tmpDir, err := afero.TempDir(m.fs, "", "go-list")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		_ = m.fs.RemoveAll(tmpDir)
	}()

	timeout := m.timeout
	if timeout == 0 {
		timeout = 10 * time.Second // default timeout
	}
	timeoutCtx, cancel := context.WithTimeout(m.ctx, timeout)
	defer cancel()

	// Capture user-specified version if provided
	suffix := ""
	if strings.Contains(module, "@") {
		parts := strings.SplitN(module, "@", 2)
		module = parts[0]
		suffix = parts[1]
	}

	m.Name = module
	m.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(module)))
	query := fmt.Sprintf("%s@latest", module)

	cmd := exec.CommandContext(
		timeoutCtx,
		m.goBinPath,
		"list", "-m", "-versions", "-json",
		query,
	)
	cmd.Dir = tmpDir
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go list failed: %w\n%s", err, stderr.String())
	}

	if err := json.NewDecoder(stdout).Decode(&lr); err != nil {
		return fmt.Errorf("failed to decode module list output: %w", err)
	}

	// Sort versions latest-to-oldest
	sort.Slice(lr.Versions, func(i, j int) bool {
		return semver.Compare(lr.Versions[i], lr.Versions[j]) > 0
	})

	// Use specified suffix if exists, otherwise take the latest
	version := suffix
	if version == "" && len(lr.Versions) > 0 {
		version = lr.Versions[0]
	}
	if version == "" {
		return fmt.Errorf("no versions found for module %q", module)
	}

	m.Version = version
	m.Versions = lr.Versions
	//m.Path = fmt.Sprintf("%s@%s", strings.TrimSuffix(module, "/"), version)
	m.Time = time.Now()

	return nil
}

func (m *Module) InstallModule(ctx context.Context) error {

	cmd := exec.CommandContext(ctx, "go", "install", fmt.Sprintf("%s@latest", m.Name))
	goBin := filepath.Join(fmt.Sprintf("GOBIN=%s", os.Getenv("GOPATH")), "bin", "go")
	cmd.Env = append(os.Environ(), goBin)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	// Save to SQLite DB here
	return nil
}
