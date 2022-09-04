/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	helpers "github.com/geaaru/luet/cmd/helpers"
	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/subsets"
	installer "github.com/geaaru/luet/pkg/v2/installer"

	"github.com/spf13/cobra"
)

func newUninstallCommand(config *cfg.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "uninstall <pkg> <pkg2> ...",
		Short: "Uninstall a package or a list of packages",
		Long: `
Remove one or more package and his dependencies recursively

	$ luet uninstall cat/foo1 ... cat/foo2

Remove one or more packages without dependencies

	$ luet uninstall cat/foo1 ... --nodeps

Remove one or more packages and skip errors

	$ luet uninstall cat/foo1 ... --force

Remove one or more packages without ask confirm

	$ luet uninstall cat/foo1 ... --yes

Remove one or more packages without ask confirm and skip execution
of the finalizers.

	$ luet uninstall cat/foo1 ... --yes --skip-finalizers
`,
		Aliases: []string{"rm", "un"},
		Run: func(cmd *cobra.Command, args []string) {
			toRemove := []*pkg.DefaultPackage{}
			for _, a := range args {

				pack, err := helpers.ParsePackageStr(a)
				if err != nil {
					Fatal("Invalid package string ", a, ": ", err.Error())
				}
				toRemove = append(toRemove, pack)
			}

			force := config.Viper.GetBool("force")
			nodeps, _ := cmd.Flags().GetBool("nodeps")
			yes := config.Viper.GetBool("yes")
			keepProtected, _ := cmd.Flags().GetBool("keep-protected-files")
			preserveSystem, _ := cmd.Flags().GetBool("preserve-system-essentials")
			skipFinalizers, _ := cmd.Flags().GetBool("skip-finalizers")
			finalizerEnvs, _ := cmd.Flags().GetStringArray("finalizer-env")

			config.ConfigProtectSkip = !keepProtected

			// Load config protect configs
			installer.LoadConfigProtectConfs(config)
			// Load subsets defintions
			subsets.LoadSubsetsDefintions(config)
			// Load subsets config
			subsets.LoadSubsetsConfig(config)

			// Load finalizer runtime environments
			err := util.SetCliFinalizerEnvs(finalizerEnvs)
			if err != nil {
				Fatal(err.Error())
			}

			aManager := installer.NewArtifactsManager(config)
			defer aManager.Close()

			opts := &installer.UninstallOpts{
				Force:                       force,
				NoDeps:                      nodeps,
				PreserveSystemEssentialData: preserveSystem,
				Ask:                         !yes,
				SkipFinalizers:              skipFinalizers,
			}

			if err := aManager.Uninstall(opts, config.GetSystem().Rootfs,
				toRemove...,
			); err != nil {
				Fatal("Error: " + err.Error())
			}
		},
	}

	flags := ans.Flags()

	flags.Bool("nodeps", false, "Don't consider package dependencies (harmful! overrides checkconflicts and full!)")
	flags.Bool("force", false, "Force uninstall")
	flags.BoolP("yes", "y", false, "Don't ask questions")
	flags.BoolP("keep-protected-files", "k", false, "Keep package protected files around")
	flags.Bool("preserve-system-essentials", true, "Preserve system luet files")
	ans.Flags().StringArray("finalizer-env", []string{},
		"Set finalizer environment in the format key=value.")
	ans.Flags().Bool("skip-finalizers", false,
		"Skip the execution of the finalizers.")

	config.Viper.BindPFlag("nodeps", flags.Lookup("nodeps"))
	config.Viper.BindPFlag("force", flags.Lookup("force"))
	config.Viper.BindPFlag("yes", flags.Lookup("yes"))

	return ans
}
