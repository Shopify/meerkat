package indexing

import (
	"github.com/google/codesearch/index"
	"github.com/karrick/godirwalk"
	"github.com/meerkat/repos"
	"github.com/pkg/errors"
	"io"
	"log"
	"strconv"
	"sync"
	"time"

	"os"
)

type Indexer interface {
	Index(r repos.Repo) error
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
