package search

import "github.com/bshafiee/codesearch/regexp"

type FileSearcher interface {
	// searches a regex inside filepath and writes results in resChan, returns number of hits
	Search(filePath string, grep *regexp.Grep, resChan chan *FileSearchResult) int
}

type SearchEngine interface {
	Search(*Query)
}

type Query struct {
	TermPatten    string
	FilePattern   string
	CaseSensitive bool
	BruteForce    bool
}
