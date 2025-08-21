package git

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"lukechampine.com/blake3"
	"sinanmohd.com/scid/internal/config"
)

type Git struct {
	LocalPath        string
	repo             *git.Repository
	NewHash, OldHash *plumbing.Hash
	// https://github.com/go-git/go-git/issues/773
	mu sync.Mutex
}

func New(repoUrl, branchName string) (*Git, error) {
	sum256 := blake3.Sum256([]byte(repoUrl + branchName))
	localPath := fmt.Sprintf("%x", sum256)

	_, err := os.Stat(localPath)
	if os.IsNotExist(err) {
		repo, err := git.PlainClone(localPath, &git.CloneOptions{
			URL:           repoUrl,
			SingleBranch:  true,
			ReferenceName: plumbing.ReferenceName(branchName),
			Progress:      os.Stdout,
		})
		if err != nil {
			return nil, err
		}

		headRef, err := repo.Head()
		if err != nil {
			return nil, err
		}
		newHash := headRef.Hash()

		return &Git{
			LocalPath: localPath,
			repo:      repo,
			NewHash:   &newHash,
			OldHash:   nil,
		}, nil
	} else if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(localPath)
	if err != nil {
		return nil, err
	}
	headRef, err := repo.Head()
	if err != nil {
		return nil, err
	}
	oldHash := headRef.Hash()

	workTree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	err = workTree.Pull(&git.PullOptions{
		SingleBranch: true,
	})
	if err == git.NoErrAlreadyUpToDate {
		return &Git{
			LocalPath: localPath,
			repo:      repo,
			NewHash:   &oldHash,
			OldHash:   &oldHash,
		}, nil
	} else if err != nil {
		return nil, err
	}

	headRef, err = repo.Head()
	if err != nil {
		return nil, err
	}
	newHash := headRef.Hash()

	return &Git{
		LocalPath: localPath,
		repo:      repo,
		NewHash:   &newHash,
		OldHash:   &oldHash,
	}, nil
}

func (bg *Git) HeadMoved() bool {
	if config.Config.DryRun {
		return true
	} else if bg.OldHash == nil {
		return true
	}

	return *bg.NewHash != *bg.OldHash
}

func (bg *Git) PathsUpdated(prefixPaths []string) (string, error) {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	if config.Config.DryRun {
		return "/", nil
	} else if bg.OldHash == nil {
		return "/", nil
	}

	coOld, err := bg.repo.CommitObject(*bg.OldHash)
	if err != nil {
		return "", err
	}
	treeOld, err := coOld.Tree()
	if err != nil {
		return "", err
	}

	coNew, err := bg.repo.CommitObject(*bg.NewHash)
	if err != nil {
		return "", err
	}
	treeNew, err := coNew.Tree()
	if err != nil {
		return "", err
	}

	changes, err := treeOld.Diff(treeNew)
	if err != nil {
		return "", err
	}

	for _, change := range changes {
		for _, path := range prefixPaths {
			if strings.HasPrefix(change.From.Name, path) {
				return change.From.Name, nil
			} else if strings.HasPrefix(change.To.Name, path) {
				return change.To.Name, nil
			}
		}
	}

	return "", nil
}
