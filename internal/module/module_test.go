package module

import (
	"context"
	"github.com/spf13/afero"
	"testing"
)

func TestModule_Check(t *testing.T) {
	mod, err := NewModule(context.TODO(), afero.NewOsFs(), "go")
	if err != nil {
		t.Fatal(err)
	}

	if err := mod.FetchModuleInfo("github.com/spf13/afero"); err != nil {
		t.Fatal(err)
	}

	if err := mod.SaveToFile("module_data.json"); err != nil {
		t.Fatal(err)
	}
}

func TestModule_Check_Latest(t *testing.T) {
	mod, err := NewModule(context.TODO(), afero.NewOsFs(), "go")
	if err != nil {
		t.Fatal(err)
	}

	if err := mod.FetchModuleInfo("github.com/spf13/afero@latest"); err != nil {
		t.Fatal(err)
	}

	if err := mod.SaveToFile("module_data_latest.json"); err != nil {
		t.Fatal(err)
	}
}
