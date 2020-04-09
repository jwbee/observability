package observability

import (
	"reflect"
	"strconv"
	"testing"
	"unsafe"
)

var (
	val = []byte("8475589")
	pow = [20]uint64{
		10000000000000000000,
		1000000000000000000,
		100000000000000000,
		10000000000000000,
		1000000000000000,
		100000000000000,
		10000000000000,
		1000000000000,
		100000000000,
		10000000000,
		1000000000,
		100000000,
		10000000,
		1000000,
		100000,
		10000,
		1000,
		100,
		10,
		1,
	}
)

// These benchmarks are here to show whether we can beat strconv.Atoi, which is
// true under certain conditions. The big cost of Atoi is the allocation of a
// string when we are given []byte. The naive approach is the fastest.  Neither
// the table nor the unrolled table can beat it, though they are all three much
// faster than Atoi and none of them allocate.  On EC2::
//
// Intel(R) Xeon(R) Platinum 8175M CPU @ 2.50GHz
// BenchmarkAtoiBytes              50000000                26.8 ns/op             8 B/op          1 allocs/op
// BenchmarkNaive                  200000000                7.12 ns/op            0 B/op          0 allocs/op
// BenchmarkPowTable               200000000                9.57 ns/op            0 B/op          0 allocs/op
// BenchmarkPowTableUnrolled4      100000000               10.3 ns/op             0 B/op          0 allocs/op
// BenchmarkAtoiEvil               100000000               12.1 ns/op             0 B/op          0 allocs/op
//
// With shorter inputs, such as "0", the difference is more pronounced.

func BenchmarkAtoiBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		strconv.Atoi(string(val))
	}
}

func BenchmarkNaive(b *testing.B) {
	for i := 0; i < b.N; i++ {
		naiveAtoi(val)
	}
}

func powTableAtoi(b []byte) uint64 {
	o := len(pow) - len(b)
	rv := uint64(0)
	for _, c := range b {
		rv += pow[o] * uint64(c-'0')
		o++
	}
	return rv
}

func BenchmarkPowTable(b *testing.B) {
	for i := 0; i < b.N; i++ {
		powTableAtoi(val)
	}
}

func powTableUnrolled4(b []byte) uint64 {
	o := len(pow) - len(b)
	rv := uint64(0)
	i := 0
	for ; i+4 <= len(b); i += 4 {
		ch := b[i : i+4]
		p := pow[o : o+4]
		r1 := p[0] * uint64(ch[0]-'0')
		r2 := p[1] * uint64(ch[1]-'0')
		r3 := p[2] * uint64(ch[2]-'0')
		r4 := p[3] * uint64(ch[3]-'0')
		rv += r1 + r2 + r3 + r4
		o += 4
	}
	for _, c := range b[i:] {
		rv += pow[o] * uint64(c-'0')
		o++
	}
	return rv
}

func BenchmarkPowTableUnrolled4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		powTableUnrolled4(val)
	}
}

func EvilAtoi(in []byte) (int, error) {
	header := reflect.StringHeader{
		Data: (uintptr)(unsafe.Pointer(&in[0])),
		Len:  len(in),
	}
	return strconv.Atoi(*(*string)(unsafe.Pointer(&header)))
}

func BenchmarkAtoiEvil(b *testing.B) {
	for i := 0; i < b.N; i++ {
		EvilAtoi(val)
	}
}

func TestScanner(t *testing.T) {
	buf := make([]byte, 0, 512)
	lf := []lineFunc{
		{
			name: []byte("foo"),
			f: func(fields [][]byte) {
				t.Log(string(fields[0]))
			},
		},
	}
	in := []byte("foo 123\nbar 456\n")
	bs := NewBufferScanner(buf, lf)
	bs.Scan(in)
	bs.Scan(in)
}

// BenchmarkScanner checks the time required to scan a trivial input with no
// functions. This takes about 11ns.
func BenchmarkScanner(b *testing.B) {
	buf := make([]byte, 0, 512)
	lf := []lineFunc{}
	in := []byte("foo 123\nbar 456\n")
	bs := NewBufferScanner(buf, lf)
	for i := 0; i < b.N; i++ {
		bs.Scan(in)
	}
}

var xfsLiteral = `extent_alloc 2850797 1422306569 2208846 750744525
abt 0 0 0 0
blk_map 4013747586 986817971 310235996 4126710 164562762 1017458437 0
bmbt 0 0 0 0
dir 17411309 155380624 155285147 119241322
trans 0 3129811657 1969
ig 163038204 160993353 482 2044851 0 2008515 1783442
log 664378396 1060789568 2 665530550 665521925
push_ail 3134332303 0 24612870 3615919 0 126217 16647 2765770 0 25742
xstrat 626433 0
rw 1344496242 2324555337
attr 864146844 5624 16406 27978
icluster 2314776 819411 2701964
vnodes 36336 0 0 0 156574971 156574971 156574971 0
buf 1423557008 1549457 1422027046 1309954 38676 1529963 0 1590113 29137
abtb2 5079763 38135146 455605 450823 149 147 18549 12399 2368 3132 197 190 346 337 184916757
abtc2 9533096 73949789 4659713 4655173 393 391 4873 937 2431 2651 486 477 879 868 1090495113
bmbt2 2086454 15066211 740201 719110 2 0 4198 768 3348 4196 92 11 94 11 8735550
ibt2 615194355 1456409582 12439 10932 0 0 2850810 36928 543 22 8 0 8 0 1582374
fibt2 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
rmapbt 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
refcntbt 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
qm 0 0 0 0 0 0 0 0
xpc 5588678254592 20036891491898 18802600680845
debug 0
`

// BenchmarkScannerXfs tries to ballpark the minimum time needed to parse XFS
// stats, including atoi but not including setting the metrics.  We need this
// to run in much less than 1ms, if we intend to collect this data several
// times per second. This runs in about 3.5μs on EC2.
func BenchmarkScannerXfs(b *testing.B) {
	buf := make([]byte, 0, 4096)
	f := func(fields [][]byte) {
		for _, field := range fields {
			naiveAtoi(field)
		}
	}
	lf := []lineFunc{
		{name: []byte("extent_alloc"), f: f},
		{name: []byte("blk_map"), f: f},
		{name: []byte("dir"), f: f},
		{name: []byte("trans"), f: f},
		{name: []byte("ig"), f: f},
		{name: []byte("log"), f: f},
		{name: []byte("push_ail"), f: f},
		{name: []byte("xstrat"), f: f},
		{name: []byte("rw"), f: f},
		{name: []byte("attr"), f: f},
		{name: []byte("icluster"), f: f},
		{name: []byte("vnodes"), f: f},
		{name: []byte("buf"), f: f},
		{name: []byte("abtb2"), f: f},
		{name: []byte("abtc2"), f: f},
		{name: []byte("bmbt2"), f: f},
		{name: []byte("ibt2"), f: f},
		{name: []byte("xpc"), f: f},
	}
	in := []byte(xfsLiteral)
	bs := NewBufferScanner(buf, lf)
	for i := 0; i < b.N; i++ {
		bs.Scan(in)
	}
}
