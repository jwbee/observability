package observability

import (
	"bufio"
	"bytes"
)

// lineFunc associates some name with a function. The function is run whenever
// the name is encountered. This is intended to be used with kernel /proc files
// or similar files that are structured like word 123\nbob 456\n. The function
// is called with the space-separated fields that follow the name. All
// whitespace is stripped from the fields. NB the argument to the function
// points into temporary scratch space that might later be clobbered. The
// function is responsible for parsing it or copying it as needed.
type lineFunc struct {
	name []byte
	f    func(fields [][]byte)
}

// BufferScanner encapsulates a reader, caller-provided buffer, lineFunc
// callbacks, and scratch space for the fields.
type BufferScanner struct {
	lineReader *bytes.Reader
	lineBuf    []byte
	lineFuncs  []lineFunc
	fields     [][]byte
}

// NewBufferScanner creates a BufferScanner from the given buffer and slice of
// lineFuncs. For best performance the buffer should be capacious enough to
// hold all of the input that can be expected on a single line.  Note that the
// lineFuncs must be ordered in the same order that their names appear in the
// input. If the file contains "alice 1\nbob 2\n" then the function for "alice"
// must immediately precede the one for "bob". If they are reversed, only the
// "bob" function would be called. It is acceptable to have input lines without
// corresponding functions ("alice 123\ngeorge 456\nbob 42\n") but it's
// unacceptable to have functions without corresponding input lines; every
// function must have a corresponding line in the input.
//
// The requirements for lineFuncs are compatible with the way the Linux kernel
// produces stats in proc files. It's not really congruent with the way that,
// say, memcached emits stats in an undefined order.
func NewBufferScanner(lineBuf []byte, lineFuncs []lineFunc) *BufferScanner {
	bs := &BufferScanner{
		lineReader: bytes.NewReader(nil),
		lineBuf:    lineBuf[:cap(lineBuf)],
		lineFuncs:  lineFuncs,
	}
	return bs
}

// naiveAtoi converts the text representation of an unsigned decimal number to
// a uint64. Use this only for ASCII text which is guaranteed to be in range
// and which consists strictly of ASCII 0-9. Use strconv for all other
// purposes. naiveAtoi is intended for use with kernel /proc files which are
// known to be produced with printf %ull.
func naiveAtoi(b []byte) uint64 {
	rv := uint64(0)
	for _, c := range b {
		rv *= 10
		rv += uint64(c - '0')
	}
	return rv
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

// asciiByteFields is ripped off from the standard bytes package, with slight
// modifications to remove UTF-8 handling. This is superior to bytes.Fields in
// that it does not allocate the return value. It is superior to
// bufio.ScanWords because ScanWords handles Unicode but we don't need that.
func asciiByteFields(s []byte, a [][]byte) [][]byte {
	fieldStart := 0
	i := 0
	// Skip spaces in the front of the input.
	for i < len(s) && asciiSpace[s[i]] != 0 {
		i++
	}
	fieldStart = i
	for i < len(s) {
		if asciiSpace[s[i]] == 0 {
			i++
			continue
		}
		a = append(a, s[fieldStart:i:i])
		i++
		// Skip spaces in between fields.
		for i < len(s) && asciiSpace[s[i]] != 0 {
			i++
		}
		fieldStart = i
	}
	if fieldStart < len(s) { // Last field might end at EOF.
		a = append(a, s[fieldStart:len(s):len(s)])
	}
	return a
}

// Fields returns a slice containing the space-separated ASCII things on the
// line.
func (bs *BufferScanner) Fields(line []byte) [][]byte {
	bs.fields = asciiByteFields(line, bs.fields[0:0])
	return bs.fields
}

// Scan reads all of the lines in the given byte buffer, calling the
// corresponding functions for the first field of each line.
func (bs *BufferScanner) Scan(b []byte) {
	bs.lineReader.Reset(b)
	scanner := bufio.NewScanner(bs.lineReader)
	scanner.Buffer(bs.lineBuf, cap(bs.lineBuf))
	for _, f := range bs.lineFuncs {
		for scanner.Scan() {
			fields := bs.Fields(scanner.Bytes())
			if len(fields) > 1 {
				if bytes.Equal(f.name, fields[0]) {
					f.f(fields[1:])
					break
				}
			}
		}
	}
}
