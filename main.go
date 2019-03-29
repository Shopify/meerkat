package main

import (
	"fmt"
	"github.com/meerkat/repos"
	"io/ioutil"
	"log"
	"path/filepath"
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
	//search()
	//repos.NewGithubRepo()
	repos.CloneToDisk()
}
