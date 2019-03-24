package indexing

import (
	"io"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/google/codesearch/index"
	"github.com/google/codesearch/regexp"
	"github.com/karrick/godirwalk"
	"github.com/meerkat/repos"
	"github.com/pkg/errors"

	"os"
)

type Indexer interface {
	Index(r repos.Repo) error
	Search(regex string) ([]string, error)
}

type indexer struct {
	masterIndexpath string
	mergeMutex      sync.Mutex
}

func NewIndexer(masterIndexFilePath string) Indexer {
	return &indexer{
		masterIndexpath: masterIndexFilePath,
	}
}

func (i *indexer) Search(reQuery string) ([]string, error) {
	g := regexp.Grep{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	g.AddFlags()

	pat := "(?m)" + reQuery
	iFlag := false     //case insensitive
	fFlag := ""        //file pattern
	bruteFlag := false //brute force - search all files in index
	if iFlag {
		pat = "(?i)" + pat
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return nil, errors.Wrap(err, "search failed, failed to compile input regex, must be valid re2")
	}
	g.Regexp = re
	var fre *regexp.Regexp
	if fFlag != "" {
		fre, err = regexp.Compile(fFlag)
		if err != nil {
			log.Fatal(err)
		}
	}
	q := index.RegexpQuery(re.Syntax)

	ix := index.Open(i.masterIndexpath)
	ix.Verbose = false
	var post []uint32
	if bruteFlag {
		post = ix.PostingQuery(&index.Query{Op: index.QAll})
	} else {
		post = ix.PostingQuery(q)
	}

	log.Printf("post query identified %d possible files\n", len(post))

	if fre != nil {
		fnames := make([]uint32, 0, len(post))

		for _, fileid := range post {
			name := ix.Name(fileid)
			if fre.MatchString(name, true, true) < 0 {
				continue
			}
			fnames = append(fnames, fileid)
		}

		log.Printf("filename regexp matched %d files\n", len(fnames))

		post = fnames
	}

	for _, fileid := range post {
		name := ix.Name(fileid)
		g.File(name)
	}

	return nil, nil
}

//Index is concurrent safe
func (i *indexer) Index(r repos.Repo) error {
	if r.DiskPath() == "" {
		return errors.Errorf("indexing %s failed since it has empty diskPath\n", r.Name())
	}

	indexFullpath := r.DiskPath() + "/" + r.Name() + ".index" + strconv.Itoa((int)(time.Now().Unix()))
	ixWriter := index.Create(indexFullpath)
	defer func() {
		if err := os.Remove(indexFullpath); err != nil {
			log.Println(errors.Wrapf(err, "failed to remove temp index:%s\n", indexFullpath))
		}
	}()

	ixWriter.AddPaths([]string{r.DiskPath()})
	if err := godirwalk.Walk(r.DiskPath(), &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if !de.IsRegular() {
				return nil
			}
			ixWriter.AddFile(osPathname)
			return nil
		},
		Unsorted: true, //set true for faster yet non-deterministic enumeration
	}); err != nil {
		return errors.Wrapf(err, "failed to walk on repo path of:%s\n", r.Name())
	}
	ixWriter.Flush()

	i.mergeMutex.Lock()
	defer i.mergeMutex.Unlock()

	if _, err := os.Stat(i.masterIndexpath); err == nil {
		//if master DOES exists
		index.Merge(indexFullpath+"@", i.masterIndexpath, indexFullpath)
	} else {
		index.Merge(indexFullpath+"@", indexFullpath, indexFullpath)
	}
	if err := os.Rename(indexFullpath+"@", i.masterIndexpath); err != nil {
		return errors.Wrapf(err, "failed to rename %s to %s \n", indexFullpath+"@", i.masterIndexpath)
	}

	return nil
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
