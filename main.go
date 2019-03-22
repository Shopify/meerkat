package main

import (
	"fmt"
	"github.com/meerkat/index"
	"github.com/meerkat/repos"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"
)

func main() {
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

	fmt.Printf("\n🎉fully indexed everything🔥\n")
}
