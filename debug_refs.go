package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func main() {
	path := "/workspaces/playwithantigravity/temp_repro/json-server"
	repo, err := git.PlainOpen(path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("--- Branches ---")
	iter, _ := repo.Branches()
	iter.ForEach(func(r *plumbing.Reference) error {
		fmt.Println(r.Name(), r.Hash())
		return nil
	})

	fmt.Println("--- Tags ---")
	tIter, _ := repo.Tags()
	tIter.ForEach(func(r *plumbing.Reference) error {
		fmt.Println(r.Name(), r.Hash())
		return nil
	})
    
    fmt.Println("--- HEAD ---")
    h, _ := repo.Head()
    fmt.Println(h.Name(), h.Hash())
}
