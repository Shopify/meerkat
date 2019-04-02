package main

import (
	//"runtime/pprof"
	//"context"
	"fmt"
	"github.com/google/go-github/github"
	indexing "github.com/meerkat/index"
	"github.com/meerkat/repos"
	//"golang.org/x/sync/semaphore"
	"io/ioutil"
	"log"
	//"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
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

func indexGH() {
	absStoragePath, err := filepath.Abs("./storage/")
	if err != nil {
		log.Fatal(err)
	}
	indexer := indexing.NewIndexer(absStoragePath + "/master.index")

	allRepos, err := repos.LoadAllGHRepos()
	fmt.Println("load ", len(allRepos), " from github, going to clone them...")

	var indexWG sync.WaitGroup
	var cloneWG sync.WaitGroup
	cloneQueue := make(chan *github.Repository, 256)
	indexQueue := make(chan *repos.GithubRepo, 5000)
	var totalIndexed uint64
	var totalErrors uint64

	cloneWorker := func(cloneQueue chan *github.Repository, indexQueue chan *repos.GithubRepo) {
		cloneWG.Add(1)
		defer cloneWG.Done()
		for j := range cloneQueue {
			g, err := repos.NewGithubRepo(j, absStoragePath)
			if err != nil {
				atomic.AddUint64(&totalErrors, 1)
				fmt.Printf("failed to clone repo: %s error:%s\n", *j.CloneURL, err)
				continue
			}
			indexQueue <- g
		}
	}

	indexWorker := func(cloneQueue chan *github.Repository, indexQueue chan *repos.GithubRepo) {
		indexWG.Add(1)
		defer indexWG.Done()
		for j := range indexQueue {
			r := repos.NewRepo("", j.DiskPath())
			if err := indexer.Index(r); err != nil {
				log.Fatal(err)
			}
			atomic.AddUint64(&totalIndexed, 1)
			fmt.Printf("LenIndexQueue:%d LenCloneQueue:%d Progress: %d out of %d Errors:%d repo:%s\n", len(indexQueue), len(cloneQueue), atomic.LoadUint64(&totalIndexed), len(allRepos), atomic.LoadUint64(&totalErrors), j.Name())
		}
	}

	for i := 0; i < 64; i++ {
		go cloneWorker(cloneQueue, indexQueue)
	}

	//fire clones
	for _, r := range allRepos {
		cloneQueue <- r
	}
	close(cloneQueue)
	cloneWG.Wait()
	fmt.Println("Clone done, indexing...")
	//clones done, fire indexer
	for i := 0; i < 256; i++ {
		go indexWorker(cloneQueue, indexQueue)
	}

	close(indexQueue)
	indexWG.Wait()
}

func main() {
	/*f, err := os.Create("profile.cpu")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer func() {
		pprof.StopCPUProfile()
		fmt.Println("wrote CPU profile file")
	}()*/
	//index()
	//search()
	indexGH()
}
