/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_box

import (
	"os"

	b64 "encoding/base64"

	"github.com/macaroni-os/anise/pkg/box"
	"github.com/macaroni-os/anise/pkg/config"
	. "github.com/macaroni-os/anise/pkg/logger"

	"github.com/spf13/cobra"
)

func NewBoxExecCommand(cfg *config.AniseConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "exec [OPTIONS]",
		Short: "Execute a binary in a box",
		Args:  cobra.OnlyValidArgs,
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
			err := b.Run()
			if err != nil {
				Fatal(err)
			}
		},
	}
	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	ans.Flags().String("rootfs", path, "Rootfs path")
	ans.Flags().Bool("stdin", false, "Attach to stdin")
	ans.Flags().Bool("stdout", true, "Attach to stdout")
	ans.Flags().Bool("stderr", true, "Attach to stderr")
	ans.Flags().Bool("decode", false, "Base64 decode")
	ans.Flags().StringArrayP("env", "e", []string{}, "Environment settings")
	ans.Flags().StringArrayP("mount", "m", []string{}, "List of paths to bind-mount from the host")

	ans.Flags().String("entrypoint", "/bin/sh", "Entrypoint command (/bin/sh)")

	return ans
}
