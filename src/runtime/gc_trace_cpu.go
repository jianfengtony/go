// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && (amd64 || arm64)

package runtime

import (
	"runtime/internal/atomic"
	"runtime/internal/syscall"
)

// gcWorkerTraceEntry is a single trace record for a GC mark worker wake-up.
type gcWorkerTraceEntry struct {
	tid uint64 // OS thread ID (gp.m.procid)
	cpu uint32 // Physical CPU number
}

// gcWorkerTraceBuffer holds trace records for one GC cycle.
// Protected by atomic operations; only written by gcBgMarkWorker,
// only read by gcMarkTermination (which runs under STW).
const gcWorkerTraceMax = 1024

var (
	gcWorkerTraceBuf      [gcWorkerTraceMax]gcWorkerTraceEntry
	gcWorkerTraceIndex    uint32 // next write index, accessed atomically
	gcWorkerTraceOverflow uint32 // set to 1 if buffer wrapped, accessed atomically
)

// getPhysicalCPU returns the OS-assigned CPU number of the calling thread.
// Returns 0 on error.
func getPhysicalCPU() uint32 {
	cpu, _ := syscall.Getcpu()
	return cpu
}

// recordGCWorkerTrace records the current worker's TID and CPU assignment.
// Called from gcBgMarkWorker on every wake-up. Safe during GC mark phase.
//
//go:nosplit
func recordGCWorkerTrace(tid uint64) {
	idx := atomic.Xadd(&gcWorkerTraceIndex, 1) - 1
	if idx >= gcWorkerTraceMax {
		atomic.Store(&gcWorkerTraceOverflow, 1)
		idx = idx % gcWorkerTraceMax
	}
	gcWorkerTraceBuf[idx].tid = tid
	gcWorkerTraceBuf[idx].cpu = getPhysicalCPU()
}

// printGCWorkerTrace prints all recorded trace entries in a single line and resets the buffer.
// Called from gcMarkTermination at the start of mark termination (under STW).
// Format (gctrace style): gc <num> @<time>s workers=<count>: <tid>-><cpu> <tid>-><cpu> ...
func printGCWorkerTrace() {
	count := atomic.Load(&gcWorkerTraceIndex)
	overflow := atomic.Load(&gcWorkerTraceOverflow) != 0

	if count == 0 {
		return
	}

	print(" workers=", count)
	if overflow {
		print("(overflow)")
	}
	print(": ")

	printEntries := count
	if printEntries > gcWorkerTraceMax {
		printEntries = gcWorkerTraceMax
	}

	for i := uint32(0); i < printEntries; i++ {
		if i > 0 {
			print(" ")
		}
		entry := gcWorkerTraceBuf[i]
		print(entry.tid, "->", entry.cpu)
	}
	print("\n")

	// Reset for next cycle
	atomic.Store(&gcWorkerTraceIndex, 0)
	atomic.Store(&gcWorkerTraceOverflow, 0)
}
