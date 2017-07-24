package git

import (
	"log"
	"os"

	"github.com/sandromello/factory/pkg/conf"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// Clone performs a PlainClone in a git server
func Clone(c *conf.Config) error {
	if c.CloneInfo.Overwrite {
		os.RemoveAll(c.CloneInfo.Path)
	}

	cloneOpts, err := c.CloneOptions()
	if err != nil {
		return err
	}

	r, err := git.PlainClone(c.CloneInfo.Path, false, cloneOpts)

	if err != nil {
		log.Fatalf("Failed cloning app: %v", err)
	}
	w, err := r.Worktree()
	if err != nil {
		log.Fatalf("Failed getting worktree: %v", err)
	}
	commitHash := plumbing.NewHash(c.CloneInfo.Commit)
	checkoutOptions := &git.CheckoutOptions{Branch: plumbing.ReferenceName(c.CloneInfo.Ref)}
	if !commitHash.IsZero() {
		checkoutOptions.Hash = commitHash
	}
	if err := w.Checkout(checkoutOptions); err != nil {
		log.Fatalf("Failed checking out: %v", err)
	}
	return nil
}
