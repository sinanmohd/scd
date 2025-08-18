package driver

import (
	"os/exec"

	"github.com/rs/zerolog/log"

	"sinanmohd.com/scid/internal/config"
	"sinanmohd.com/scid/internal/git"
)

func ExecIfChaged(paths, execLine []string, bg *git.Git) (string, string, error /* exec error */, error) {
	changed, err := bg.PathsUpdated(paths)
	if err != nil {
		return "", "", nil, err
	}
	if changed == "" {
		return "", "", nil, nil
	}

	log.Info().Msgf("Execing %v", execLine)

	if config.Config.DryRun {
		return "", changed, nil, err
	}

	output, err := exec.Command(execLine[0], execLine[1:]...).CombinedOutput()
	if err != nil {
		return string(output), changed, err, nil
	}
	return string(output), changed, nil, nil
}
