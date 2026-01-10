/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"

	b64 "encoding/base64"

	"github.com/macaroni-os/anise/pkg/box"
	config "github.com/macaroni-os/anise/pkg/config"
	. "github.com/macaroni-os/anise/pkg/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newExecCommand(cfg *config.AniseConfig) *cobra.Command {
	var execCmd = &cobra.Command{
		Use:   "exec --rootfs /path [command]",
		Short: "Execute a command in the rootfs context",
		Long:  `Uses unshare technique and pivot root to execute a command inside a folder containing a valid rootfs`,
		PreRun: func(cmd *cobra.Command, args []string) {
		},
		// If you change this, look at pkg/box/exec that runs this command and adapt
		Run: func(cmd *cobra.Command, args []string) {

			stdin, _ := cmd.Flags().GetBool("stdin")
			stdout, _ := cmd.Flags().GetBool("stdout")
			stderr, _ := cmd.Flags().GetBool("stderr")
			rootfs, _ := cmd.Flags().GetString("rootfs")
			base, _ := cmd.Flags().GetBool("decode")

			entrypoint, _ := cmd.Flags().GetString("entrypoint")
			envs, _ := cmd.Flags().GetStringArray("env")
			mounts, _ := cmd.Flags().GetStringArray("mount")

			if base {
				var ss []string
				for _, a := range args {
					sDec, _ := b64.StdEncoding.DecodeString(a)
					ss = append(ss, string(sDec))
				}
				//If the command to run is complex,using base64 to avoid bad input

				args = ss
			}
			Info("Executing", args, "in", rootfs)

			b := box.NewBox(entrypoint, args, mounts, envs, rootfs,
				stdin, stdout, stderr, cfg)
			err := b.Exec()
			if err != nil {
				Fatal(errors.Wrap(err, fmt.Sprintf("entrypoint: %s rootfs: %s", entrypoint, rootfs)))
			}
		},
	}

	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	execCmd.Hidden = true
	execCmd.Flags().String("rootfs", path, "Rootfs path")
	execCmd.Flags().Bool("stdin", false, "Attach to stdin")
	execCmd.Flags().Bool("stdout", false, "Attach to stdout")
	execCmd.Flags().Bool("stderr", false, "Attach to stderr")
	execCmd.Flags().Bool("decode", false, "Base64 decode")

	execCmd.Flags().StringArrayP("env", "e", []string{}, "Environment settings")
	execCmd.Flags().StringArrayP("mount", "m", []string{}, "List of paths to bind-mount from the host")

	execCmd.Flags().String("entrypoint", "/bin/sh", "Entrypoint command (/bin/sh)")

	return execCmd
}
