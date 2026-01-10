/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"

	config "github.com/macaroni-os/anise/pkg/config"
	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"
	. "github.com/macaroni-os/anise/pkg/logger"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

func newMigrateLuetCommand(cfg *config.AniseConfig) *cobra.Command {
	var ans = &cobra.Command{
		Hidden: false,
		Use:    "migrate-luet",
		Short:  "Migrate old /var/cache/luet directory to /var/cache/anise/",
		Long:   `Migrate luet database to anise database.`,
		Run: func(cmd *cobra.Command, args []string) {

			oldCachedir := "/var/cache/luet"
			newCachedir := "/var/cache/anise"
			if !fileHelper.Exists(oldCachedir) {
				fmt.Println("Directory " + oldCachedir + " not present. Nothing to do.")
				return
			}

			dir2rm := true
			if !fileHelper.Exists(newCachedir) {
				err := os.Rename(oldCachedir, newCachedir)
				if err != nil {
					Fatal(fmt.Sprintf(
						"error on rename directory %s to %s: %s",
						oldCachedir, newCachedir, err.Error()))
				}
				dir2rm = false
			}

			oldDbpath := "/var/cache/anise/luet.db"
			newDbpath := "/var/cache/anise/anise.db"
			if fileHelper.Exists(newDbpath) {
				fmt.Println("Database " + newDbpath + " already present." +
					" Remove it if you want force the migration.")
				return
			}

			if dir2rm {
				oldDbpath = "/var/cache/luet/luet.db"
			}

			if fileHelper.Exists(oldDbpath) {
				err := os.Rename(oldDbpath, newDbpath)
				if err != nil {
					Fatal(fmt.Sprintf(
						"error on rename database %s to %s: %s",
						oldDbpath, newDbpath, err.Error()))
				}
			} else {
				fmt.Println("Database " + oldDbpath + " not present." +
					" Nothing to do.")
				return
			}

			if dir2rm {
				err := os.RemoveAll(oldCachedir)
				if err != nil {
					Fatal(fmt.Sprintf(
						"error on remove directory %s: %s",
						oldCachedir, err.Error()))
				}
			}

			InfoC(fmt.Sprintf(":confetti_ball:%s",
				Bold(Blue("All done."))))

		},
	}

	return ans
}
