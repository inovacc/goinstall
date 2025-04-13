package monitor

import (
	"github.com/inovacc/goinstall/internal/database"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var afs afero.Fs

func Monitor(cmd *cobra.Command, args []string) error {
	afs = afero.NewOsFs()
	db, err := database.NewDatabase(cmd.Context(), afs)
	if err != nil {
		return err
	}
	defer func(db *database.Database) {
		cobra.CheckErr(db.Close())
	}(db)

	return nil
}

func moduleMonitor(db *database.Database) {

}
