package search

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/bshafiee/codesearch/regexp"
)

func file(g *regexp.Grep, name string) {
	f, err := os.Open(name)
	if err != nil {
		fmt.Fprintf(g.Stderr, "%s\n", err)
		return
	}
	defer f.Close()
	g.Reader(f, name)
}

func resultProcessor(g *regexp.Grep, r io.Reader, name string) {
	var nl = []byte{'\n'}
	var (
		buf        = make([]byte, 1<<20)
		needLineno = g.N
		lineno     = 1
		count      = 0
		prefix     = ""
		beginText  = true
		endText    = false
	)
	if !g.H {
		prefix = name + ":"
	}
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
			m1 := g.Regexp.Match(buf[chunkStart:end], beginText, endText) + chunkStart
			beginText = false
			if m1 < chunkStart {
				break
			}
			g.Match = true
			if g.L {
				fmt.Fprintf(g.Stdout, "%s\n", name)
				return
			}
			lineStart := bytes.LastIndex(buf[chunkStart:m1], nl) + 1 + chunkStart
			lineEnd := m1 + 1
			if lineEnd > end {
				lineEnd = end
			}
			if needLineno {
				lineno += countNL(buf[chunkStart:lineStart])
			}
			line := buf[lineStart:lineEnd]
			nl := ""
			if len(line) == 0 || line[len(line)-1] != '\n' {
				nl = "\n"
			}
			switch {
			case g.C:
				count++
			case g.N:
				fmt.Fprintf(g.Stdout, "%s%d:%s%s", prefix, lineno, line, nl)
			default:
				fmt.Fprintf(g.Stdout, "%s%s%s", prefix, line, nl)
			}
			if needLineno {
				lineno++
			}
			chunkStart = lineEnd
		}
		if needLineno && err == nil {
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
	if g.C && count > 0 {
		fmt.Fprintf(g.Stdout, "%s: %d\n", name, count)
	}
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
