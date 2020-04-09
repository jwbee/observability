package observability

import ()

var (
	// The names of these variables are consistent with the linux kernel
	// variable that exports them, but the exported name of the statistic
	// is cleaned up to be meaningful and consistent with our style.  The
	// explanations are almost verbatim from
	// http://xfs.org/index.php/Runtime_Stats with minor changes for
	// correctness and consistency.
	xfsExtentAllocxDesc = DescribeMeter(
		"/xfs/extent/extents_allocated",
		"Number of extents allocated over all XFS filesystems.",
		Cumulative())
	xfsExtentAllocbDesc = DescribeMeter(
		"/xfs/extent/blocks_allocated",
		"Number of blocks allocated over all XFS filesystems.",
		Cumulative())
	xfsDirCreateDesc = DescribeMeter(
		"/xfs/dir/created",
		"Number of times a new directory entry was created in XFS filesystems.",
		Cumulative())
	xfsReadCallsDesc = DescribeMeter(
		"/xfs/reads",
		"Number of reads of files in XFS filesystems.",
		Cumulative())
	xfsWriteCallsDesc = DescribeMeter(
		"/xfs/writes",
		"Number of writes to files in XFS filesystems.",
		Cumulative())
	xfsXPCReadBytesDesc = DescribeMeter(
		"/xfs/bytes_read",
		"Number of bytes read from files in XFS filesystems. It can be "+
			"used in conjunction with `/xfs/reads` to calculate the average "+
			"size of the read operations to files in XFS filesystems.",
		Cumulative())
	xfsXPCWriteBytesDesc = DescribeMeter(
		"/xfs/bytes_written",
		"Number of bytes written to "+
			"files in XFS filesystems. It can be used in conjunction with "+
			"`/xfs/writes` to calculate the average size of the "+
			"write operations to files in XFS filesystems.",
		Cumulative())
)
