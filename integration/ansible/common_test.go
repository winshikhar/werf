package ansible_test

import (
	"github.com/flant/werf/pkg/testing/utils"
	"github.com/flant/werf/pkg/testing/utils/liveexec"
)

func werfBuild(dir string, opts liveexec.ExecCommandOptions, extraArgs ...string) error {
	return liveexec.ExecCommand(dir, werfBinPath, opts, utils.WerfBinArgs(append([]string{"build", "--stages-storage", ":local"}, extraArgs...)...)...)
}

func werfPurge(dir string, opts liveexec.ExecCommandOptions, extraArgs ...string) error {
	return liveexec.ExecCommand(dir, werfBinPath, opts, utils.WerfBinArgs(append([]string{"stages", "purge", "--stages-storage", ":local"}, extraArgs...)...)...)
}
