/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	. "github.com/geaaru/luet/pkg/logger"
	solver "github.com/geaaru/luet/pkg/v2/solver"
)

func (m *ArtifactsManager) Upgrade(opts *InstallOpts, targetRootfs string) error {

	//mapRepos := make(map[string]*wagon.WagonRepository, 0)
	//errs := []error{}

	m.Setup()

	err := m.ShowReposRevision()
	if err != nil {
		return err
	}

	Info(":thinking:Computing upgrade, please hang tight... :zzz:")

	Spinner(3)

	solverOpts := &solver.SolverOpts{
		IgnoreConflicts: false,
		NoDeps:          opts.NoDeps,
	}

	s := solver.NewSolverImplementation("solverv2", m.Config, solverOpts)
	(*s).SetDatabase(m.Database)
	pkgs2Remove, pkgs2Update, pkgs2Install, err := (*s).Upgrade()
	SpinnerStop()
	if err != nil {
		return err
	}

	if len(pkgs2Remove.Artifacts) == 0 &&
		len(pkgs2Update.Artifacts) == 0 &&
		len(pkgs2Install.Artifacts) == 0 {
		// POST: No new updates.
		InfoC(":smiling_face_with_sunglasses:No packages to updates. The system is updated.")
	} else {

		/*
			fmt.Println("REMOVE ", len(pkgs2Remove.Artifacts))
			fmt.Println("UPDATE ", len(pkgs2Update.Artifacts))
			fmt.Println("INSTALL ", len(pkgs2Install.Artifacts))
		*/
	}

	// Cleanup solver and memory
	s = nil

	return nil
}
