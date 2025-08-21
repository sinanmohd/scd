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
	changedPaths     []string
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

	g := Git{
		LocalPath: localPath,
		repo:      repo,
		NewHash:   &newHash,
		OldHash:   &oldHash,
	}
	err = g.changedPathsSet()
	if err != nil {
		return nil, err
	}

	return &g, nil
}

func (g *Git) changedPathsSet() error {
	coOld, err := g.repo.CommitObject(*g.OldHash)
	if err != nil {
		return err
	}
	treeOld, err := coOld.Tree()
	if err != nil {
		return err
	}

	coNew, err := g.repo.CommitObject(*g.NewHash)
	if err != nil {
		return err
	}
	treeNew, err := coNew.Tree()
	if err != nil {
		return err
	}

	changes, err := treeOld.Diff(treeNew)
	if err != nil {
		return err
	}

	for _, change := range changes {
		if change.From.Name != "" {
			g.changedPaths = append(g.changedPaths, change.From.Name)
		}
		if change.To.Name != "" {
			g.changedPaths = append(g.changedPaths, change.To.Name)
		}
	}

	return err
}

func (g *Git) HeadMoved() bool {
	if config.Config.DryRun {
		return true
	} else if g.OldHash == nil {
		return true
	}

	return *g.NewHash != *g.OldHash
}

func (g *Git) ArePathsChanged(prefixPaths []string) string {
	for _, changedPath := range g.changedPaths {
		for _, prefixPath := range prefixPaths {
			if strings.HasPrefix(changedPath, prefixPath) {
				return changedPath
			}
		}
	}

	return ""
}
