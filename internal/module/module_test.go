package module

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/afero"
	"testing"
)

func TestModule_Check(t *testing.T) {
	mod, err := NewModule(context.TODO(), afero.NewOsFs(), "go")
	if err != nil {
		t.Fatal(err)
	}

	if err := mod.Check("github.com/spf13/afero"); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(mod)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(data))
}
