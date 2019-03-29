package main

import (
	"fmt"
	//"github.com/google/go-github/github"
	"github.com/meerkat/repos"
	"io/ioutil"
	"log"
	"path/filepath"
	//"sync"
	"time"

	indexing "github.com/meerkat/index"
)

func index() {
	repoPath := "/Users/behrooz/go/src/github.com/"
	files, err := ioutil.ReadDir(repoPath)
	if err != nil {
		log.Fatal(err)
	}

	fullIndexDir, err := filepath.Abs("./storage/")
	if err != nil {
		log.Fatal(err)
	}
	indexer := indexing.NewIndexer(fullIndexDir + "/master.index")

	s := time.Now()
	for _, f := range files {
		if !f.IsDir() {
			continue
		}

		start := time.Now()
		r := repos.NewRepo("github.com/"+f.Name(), repoPath+f.Name())
		if err := indexer.Index(r); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Indexed: ", f.Name(), " in ", time.Since(start))
	}

	fmt.Printf("\n🎉fully indexed everything in: %s🔥\n", time.Since(s))
}

func search() {
	fullIndexDir, err := filepath.Abs("./storage/")
	if err != nil {
		log.Fatal(err)
	}
	indexer := indexing.NewIndexer(fullIndexDir + "/master.index")
	indexer.Search(".*SHOPIFY.*")
}

func main() {
	//index()
	search()
	/*absStoragePath, err := filepath.Abs("./storage/")
	if err != nil {
		log.Fatal(err)
	}
	indexer := indexing.NewIndexer(absStoragePath + "/master.index")

	allRepos, err := repos.LoadAllGHRepos()
	fmt.Println("load ", len(allRepos), " from github, going to clone them...")
	var wg sync.WaitGroup
	for _, r := range allRepos {
		wg.Add(1)
		go func(repo *github.Repository) {

			g, err := repos.NewGithubRepo(repo, absStoragePath)
			if err != nil {
				fmt.Println("failed to clone:", err)
				return
			}

			r := repos.NewRepo(*repo.CloneURL, g.DiskPath())
			if err := indexer.Index(r); err != nil {
				log.Fatal(err)
			}

			fmt.Println("Cloned and indexed done for ", g.Name())

			wg.Done()
		}(r)
	}
	wg.Wait()*/
}
