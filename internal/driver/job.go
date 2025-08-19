package driver

import (
	"fmt"
	"sync"

	"sinanmohd.com/scid/internal/config"
	"sinanmohd.com/scid/internal/git"
	"sinanmohd.com/scid/internal/slack"

	"github.com/rs/zerolog/log"
)

func JobRunIfChaged(job config.JobConfig, g *git.Git) error {
	output, changedPath, execErr, err := ExecIfChaged(job.WatchPaths, job.ExecLine, g)
	if err != nil {
		return err
	} else if changedPath == "" {
		return nil
	}

	var color string
	if job.SlackColor == "" {
		color = "#000000"
	} else {
		color = job.SlackColor
	}

	if execErr != nil {
		extraText := fmt.Sprintf("watch path %s changed\n%s: %s", changedPath, execErr.Error(), output)
		err = slack.SendMesg(g, color, job.Name, false, extraText)
	} else {
		extraText := fmt.Sprintf("watch path %s changed\n%s", changedPath, output)
		err = slack.SendMesg(g, color, job.Name, true, extraText)
	}
	if err != nil {
		return err
	}

	return nil
}

func JobRunIfChagedWrapped(job config.JobConfig, bg *git.Git, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		err := JobRunIfChaged(job, bg)
		if err != nil {
			log.Error().Err(err).Msgf("Running Job %s", job.Name)
		}

		wg.Done()
	}()
}

func JobsRunIfChaged(g *git.Git) error {
	var jobWg sync.WaitGroup
	for _, job := range config.Config.Jobs {
		JobRunIfChagedWrapped(job, g, &jobWg)
	}
	jobWg.Wait()

	return nil
}
