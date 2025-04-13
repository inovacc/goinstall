package database

import (
	"context"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"path/filepath"
	"testing"
)

func TestNewDatabase(t *testing.T) {
	afs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(afs, "", "database")
	if err != nil {
		t.Fatal(err)
	}

	viper.Set("installPath", filepath.Join(tmpDir, "modules.db"))

	db, err := NewDatabase(context.TODO(), afs)
	if err != nil {
		t.Fatal(err)
	}

	if db == nil {
		t.Fatal("db is nil")
	}
}
