package main

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func main() {
	// Clone to temp dir
	tempDir := "/tmp/fastapi_debug"
	os.RemoveAll(tempDir)

	fmt.Println("Cloning fastapi to", tempDir)
	_, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      "https://github.com/fastapi/fastapi",
		Progress: os.Stdout,
	})
	if err != nil {
		fmt.Printf("Clone failed: %v\n", err)
		return
	}

	repo, err := git.PlainOpen(tempDir)
	if err != nil {
		fmt.Printf("Open failed: %v\n", err)
		return
	}

	fmt.Println("Iterating Tags...")
	tIter, err := repo.Tags()
	if err != nil {
		fmt.Printf("Tags() failed: %v\n", err)
		return
	}

	count := 0
	tIter.ForEach(func(r *plumbing.Reference) error {
		count++
		name := r.Name().Short()
		hash := r.Hash().String()

		fmt.Printf("Tag: %s, Hash: %s\n", name, hash)

		// Check resolution
		tagObj, err := repo.TagObject(r.Hash())
		if err == nil {
			fmt.Printf("  -> Annotated! Target: %s\n", tagObj.Target.String())
		}
		return nil
	})
	fmt.Printf("Total Tags Found: %d\n", count)
}
