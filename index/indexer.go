package indexing

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/codesearch/index"
	"github.com/google/codesearch/regexp"
	"github.com/karrick/godirwalk"
	"github.com/meerkat/repos"
	"github.com/pkg/errors"

	"os"
)

const filePatternsNotToIndex = `/.git/`

type FileSearchResult struct {
	filePath string
	lineNo   int
	line     string
}

type Indexer interface {
	Index(r repos.Repo) error
	Search(regex string) ([]*FileSearchResult, error)
}

type indexer struct {
	masterIndexpath string
	mergeMutex      sync.Mutex
	matchMutex      sync.Mutex
}

func NewIndexer(masterIndexFilePath string) Indexer {
	return &indexer{
		masterIndexpath: masterIndexFilePath,
	}
}

func (i *indexer) Search(reQuery string) ([]*FileSearchResult, error) {
	g := regexp.Grep{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	g.L = false //"list matching files only")
	g.C = false //"print match counts only")
	g.N = true  //"n", false, "show line numbers")
	g.H = false //"omit file names"

	pat := "(?m)" + reQuery
	iFlag := false                                              //case insensitive
	fFlag := "^/Users/behrooz/go/src/github.com/Shopify/ivm/.*" //file pattern
	bruteFlag := false                                          //brute force - search all files in index
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
	resChan := make(chan *FileSearchResult, 999999999)
	var wg = sync.WaitGroup{}
	for _, fileID := range post {
		wg.Add(1)
		go func(fid uint32) {
			name := ix.Name(fid)
			i.file(&g, name, resChan)
			wg.Done()
		}(fileID)
	}
	go func() {
		wg.Wait()
		close(resChan)
	}()

	for r := range resChan {
		fmt.Println(r.filePath, ":", r.lineNo, "", r.line)
	}

	return nil, nil
}

func (i *indexer) file(g *regexp.Grep, name string, resChan chan *FileSearchResult) error {
	f, err := os.Open(name)
	if err != nil {
		return errors.Wrap(err, "file open failed")
	}
	defer f.Close()
	return i.resultProcessor(g, f, name, resChan)
}

func (i *indexer) resultProcessor(g *regexp.Grep, r io.Reader, name string, resChan chan *FileSearchResult) error {
	var nl = []byte{'\n'}
	var (
		buf       = make([]byte, 1<<20)
		lineno    = 1
		beginText = true
		endText   = false
	)
	for {
		n, err := io.ReadFull(r, buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		end := len(buf)
		if err == nil {
			i := bytes.LastIndex(buf, nl)
			if i >= 0 {
				end = i + 1
			}
		} else {
			endText = true
		}
		chunkStart := 0
		for chunkStart < end {
			i.matchMutex.Lock()
			m1 := g.Regexp.Match(buf[chunkStart:end], beginText, endText) + chunkStart //not concurrent-safe
			i.matchMutex.Unlock()
			beginText = false
			if m1 < chunkStart {
				break
			}
			g.Match = true
			lineStart := bytes.LastIndex(buf[chunkStart:m1], nl) + 1 + chunkStart
			lineEnd := m1 + 1
			if lineEnd > end {
				lineEnd = end
			}
			lineno += countNL(buf[chunkStart:lineStart])
			line := buf[lineStart:lineEnd]

			resChan <- &FileSearchResult{
				filePath: name,
				line:     string(line),
				lineNo:   lineno,
			}

			lineno++

			chunkStart = lineEnd
		}
		if err == nil {
			lineno += countNL(buf[chunkStart:end])
		}
		n = copy(buf, buf[end:])
		buf = buf[:n]
		if len(buf) == 0 && err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				fmt.Fprintf(g.Stderr, "%s: %v\n", name, err)
			}
			break
		}
	}

	return nil
}

func countNL(b []byte) int {
	n := 0
	for {
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			break
		}
		n++
		b = b[i+1:]
	}
	return n
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
			if !de.IsRegular() || strings.Contains(osPathname, filePatternsNotToIndex) {
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
