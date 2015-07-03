package lineio

import (
	"io"

	"github.com/prattmic/lesser/sortedmap"
)

type LineReader struct {
	src io.ReaderAt

	// offsetCache remembers the offset of various lines in src.
	// At minimum, line 1 must be prepopulated.
	offsetCache sortedmap.Map
}

// scanForLine reads from curOffset (which is on curLine), looking for line,
// returning the offset of line.
func (l *LineReader) scanForLine(line, curLine, curOffset int64) (offset int64, err error) {
	for {
		buf := make([]byte, 128)

		n, err := l.src.ReadAt(buf, curOffset)
		// Keep looking as long as *something* is returned
		if n == 0 && err != nil {
			return 0, err
		}

		buf = buf[:n]

		for i, b := range buf {
			if b != '\n' {
				continue
			}

			offset := curOffset + int64(i) + 1
			curLine += 1

			l.offsetCache.Insert(curLine, offset)

			if curLine == line {
				return offset, nil
			}
		}

		curOffset += int64(len(buf))
	}
}

// findLine returns the offset of start of line.
func (l *LineReader) findLine(line int64) (offset int64, err error) {
	nearest, offset, err := l.offsetCache.NearestLessEqual(line)
	if err != nil {
		return 0, err
	}

	// Is this the line we want?
	if nearest == line {
		return offset, nil
	}

	return l.scanForLine(line, nearest, offset)
}

// findLineRange returns the offset of the first and last bytes in line.
// end = -1 if EOF is encountered before the end of line.
func (l *LineReader) findLineRange(line int64) (start, end int64, err error) {
	start, err = l.findLine(line)
	if err != nil {
		return 0, 0, err
	}

	end, err = l.findLine(line + 1)
	// EOF means there is no next line.
	if err == io.EOF {
		return start, -1, nil
	}
	if err != nil {
		return 0, 0, err
	}

	// The caller expects end to be the last character in the line,
	// but findLine returns the start of the next line.
	end -= 1

	return start, end, nil
}

// ReadLine reads up to len(p) bytes from line number line from the source.
// It returns the numbers of bytes written and any error encountered.
// If n < len(p), err is set to a non-nil value explaining why.
// See io.ReaderAt for full description of return values.
func (l *LineReader) ReadLine(p []byte, line int64) (n int, err error) {
	start, end, err := l.findLineRange(line)
	if err != nil {
		return 0, err
	}

	var shrunk bool
	if end >= 0 {
		// Only read one line worth of data.
		size := end - start
		if size < int64(len(p)) {
			p = p[:size]
			shrunk = true
		}
	}

	n, err = l.src.ReadAt(p, start)
	// We used less than len(p), we must return EOF.
	if err == nil && shrunk {
		err = io.EOF
	}

	return n, err
}

func NewLineReader(src io.ReaderAt) LineReader {
	l := LineReader{
		src:         src,
		offsetCache: sortedmap.NewMap(),
	}

	// Line 1 starts at the beginning of the file!
	l.offsetCache.Insert(1, 0)

	return l
}
