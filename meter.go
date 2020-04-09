package observability

/*

Package observability provides a framework and utilties for exporting data
from hosts and processes. The basic example is that you want to take a few
fields out of /proc/vmstat and expose them over HTTP or some other RPC
scheme.

TL;DR

A "Meter" measures some observable aspect of a system. An "Origin" represents
some entity with many Meters.

Concepts

An Origin is the thing from which the data is derived. In the case of kernel
data (/proc files) the information belongs to that instance of the kernel, so
if that instance can be uniquely identified by a host name, then the host name
constitutes the identity of that Origin. On the other hand, if you have a
process on the machine, and you have some statistics about the process (such as
CPU time), then there must be other data that uniquely identifies the Origin.
PIDs are not suitable, since the process to which a PID refers changes as time
passes. Carefully choose the data that identifies an Origin, otherwise recall
of this data will be impossible.

Another aspect of the Origin is that there may be more than one thing in the
universe simultaneously exporting data from a single Origin, and there may be
some unique thing exporting data about many Origins. An example of the latter
is when you have a daemon on a machine exporting kernel /proc files, and at the
same time you have a daemon elsewhere that reads IPMI data from the same host.
This is a desirable and intended state of affairs, and works fine as long as
the two processes export disjoint sets of data for the same Origin. If there
are two entities exporting the same data for the same Origin, the result would
be undefined. An example of the latter is when you have a whole fleet of
memcached servers, and you use a central daemon that exports the stats for all
of them. The ability to instantiate multiple Origins in a single process
accommodates that.

The other core concept is the Meter. A Meter measures something, for example it
might measure the number of page faults taken by a process. A Meter is
described exactly once in any given process. The description gives the name,
explanation, and characteristics of the Meter (such as whether the Meter is
ever-increasing, or what its units are if there are applicable units). Meters
are instantiated at most once per Origin. Meters are registered along with some
function that can set them. If you have a Meter that exports the cumulative CPU
jiffies for a Linux system, then you need some function that opens and reads
/proc/stat and marshals the relevant fields into the Meter. The Origin calls
the function whenever the value of the Meter is needed. Several Meters can be
associated with a setting function, so a bunch of data from a single source can
be updated in a consistent way.
*/

import (
	"runtime"
	"time"
)

// MeterDescription describes a Meter.
type MeterDescription struct {
	// name of the meter. This is intended for humans, so it should be
	// judiciously chosen.
	name string
	// explanation explains what this meter means. This should not just
	// restate the name. Anything the user needs to know when interpreting
	// this data goes here. Refer to primary sources when possible. For
	// instance, refer to the place in the kernel where the data
	// originated.
	explanation string
	// cumulative: whether this meter describes a cumulative process (such
	// as time) or not (such as memory usage). Cumulative meters are
	// checked for wrap-around, while others are not.
	cumulative bool
	// describedAt contains the stack trace that called DescribeMeter. This
	// helps readers understand the exact meaning of the meter, so they can
	// refer to the code where it is instantiated.
	describedAt []uintptr
}

// DescOption is used to mutate the description during instantiation. TODO:
// currently there is just Cumulative option. I imagine there will also be
// units decorators (bytes, nanoseconds, whatever).
type DescOption interface {
	apply(MeterDescription) MeterDescription
}

type functorOption func(MeterDescription) MeterDescription

func (f functorOption) apply(md MeterDescription) MeterDescription {
	return f(md)
}

// Cumulative returns a DescOption that sets the cumulative field of the
// MeterDescription.
func Cumulative() DescOption {
	return functorOption(func(md MeterDescription) MeterDescription {
		md.cumulative = true
		return md
	})
}

// DescribeMeter returns a MeterDescription with the given name, explanation,
// and options.
func DescribeMeter(name, explan string, opts ...DescOption) MeterDescription {
	md := MeterDescription{
		name:        name,
		explanation: explan,
		describedAt: make([]uintptr, 1),
	}
	// Skip two frames of the call stack: one for runtime.Callers itself
	// and one more for this function.
	n := runtime.Callers(2 /* skip */, md.describedAt)
	md.describedAt = md.describedAt[:n]
	for _, opt := range opts {
		md = opt.apply(md)
	}
	return md
}

// Origin is a uniquely identifiable thing that exports meters. For example, a
// single instance of Linux running on some host, a single container, one
// process within the container. Meters are registered, along with a function
// to set them, with one or more Origins.
type Origin struct{}

// RegisterFunction registers the provided nullary functor |f| as the exclusive
// means of mutating the provided Meters. The function is expected to modify
// all of the provided meters when called, and no other context may modify
// them. The function is called exclusively by this origin. No locking is
// provided; if the function requires synchronization it must do so internally,
// for example by closing over a *sync.Mutex.
func (o *Origin) RegisterFunction(f func(), ms ...Meter) {}

type Meter interface {
	SampleAt(time.Time, uint64)
	Value() (time.Time, uint64)
	ResetAt(time.Time)
}

type setFunc func(Meter, time.Time, uint64)

// counterSet sets the reset time on a counter if it overflows.
func counterSet(m Meter, t time.Time, v uint64) {
	if _, old := m.Value(); v < old {
		m.ResetAt(t)
	}
}

// gaugeSet does nothing since setting gauges to any value is expected.
func gaugeSet(_ Meter, _ time.Time, _ uint64) {
	return
}

type scalarMeter struct {
	md MeterDescription
	v  uint64
	t  time.Time
	r  time.Time
	f  setFunc
}

func (m *scalarMeter) SampleAt(t time.Time, v uint64) {
	m.f(m, t, v)
	m.t = t
	m.v = v
}

func (m *scalarMeter) ResetAt(t time.Time) {
	m.t = t
	m.r = t
	m.v = 0
}

func (m *scalarMeter) Value() (time.Time, uint64) {
	return m.t, m.v
}

func DefineCounter(md MeterDescription) Meter {
	return &scalarMeter{
		md: md,
		r:  time.Now(),
		f:  counterSet,
	}
}
