/*
Package procmeminfo provides an interface for /proc/meminfo

	import "github.com/guillermo/go.procmeminfo"

	meminfo := &procmeminfo.MemInfo{}
	meminfo.Update()

Once the info was updated you can access like a normal map[string]float64

	v := (*meminfo)["MemTotal"]  // 1809379328 (1766972 * 1024)

It also implement some handy methods, like:

	meminfo.Total() // (*meminfo)["MemTotal"]
	meminfo.Free()  // MemFree + Buffers + Cached
	meminfo.Used()  // Total - Free


Return all the values in units, so while you get this from cat /proc/meminfo

	MemTotal:        1766972 kB
	MemFree:          115752 kB
	Buffers:            3172 kB
	Cached:           182552 kB
	SwapCached:        83572 kB
	Active:          1055284 kB
	Inactive:         382872 kB
	Active(anon):     932712 kB
	Inactive(anon):   329508 kB
	Active(file):     122572 kB
	Inactive(file):    53364 kB
	Unevictable:       10640 kB
	Mlocked:           10640 kB
	SwapTotal:       1808668 kB
	SwapFree:        1205672 kB
	Dirty:               100 kB
	Writeback:             0 kB
	AnonPages:       1214740 kB
	Mapped:           115636 kB
	Shmem:              4840 kB
	Slab:              77412 kB
	SReclaimable:      34344 kB
	SUnreclaim:        43068 kB
	KernelStack:        4328 kB
	PageTables:        39428 kB
	NFS_Unstable:          0 kB
	Bounce:                0 kB
	WritebackTmp:          0 kB
	CommitLimit:     2692152 kB
	Committed_AS:    5448372 kB
	VmallocTotal:   34359738367 kB
	VmallocUsed:      106636 kB
	VmallocChunk:   34359618556 kB
	HardwareCorrupted:     0 kB
	AnonHugePages:         0 kB
	HugePages_Total:       0
	HugePages_Free:        0
	HugePages_Rsvd:        0
	HugePages_Surp:        0
	Hugepagesize:       2048 kB
	DirectMap4k:      216236 kB
	DirectMap2M:     1593344 kB


All the kB values are multiply by 1024

This is the info extracted from the man page of proc:


       /proc/meminfo
              This file reports statistics about memory usage on the system.  It is used by free(1) to report the amount of free and used memory (both physical and swap) on the system as well as the shared memory and buffers used by the kernel.  Each line of the file consists of a parameter name, followed by a colon, the value of
              the  parameter, and an option unit of measurement (e.g., "kB").  The list below describes the parameter names and the format specifier required to read the field value.  Except as noted below, all of the fields have been present since at least Linux 2.6.0.  Some fileds are displayed only if the kernel was configured
              with various options; those dependencies are noted in the list.

              MemTotal %lu
                     Total usable RAM (i.e. physical RAM minus a few reserved bits and the kernel binary code).

              MemFree %lu
                     The sum of LowFree+HighFree.

              Buffers %lu
                     Relatively temporary storage for raw disk blocks that shouldn't get tremendously large (20MB or so).

              Cached %lu
                     In-memory cache for files read from the disk (the page cache).  Doesn't include SwapCached.

              SwapCached %lu
                     Memory that once was swapped out, is swapped back in but still also is in the swap file.  (If memory pressure is high, these pages don't need to be swapped out again because they are already in the swap file.  This saves I/O.)

              Active %lu
                     Memory that has been used more recently and usually not reclaimed unless absolutely necessary.

              Inactive %lu
                     Memory which has been less recently used.  It is more eligible to be reclaimed for other purposes.

              Active(anon) %lu (since Linux 2.6.28)
                     [To be documented.]

              Inactive(anon) %lu (since Linux 2.6.28)
                     [To be documented.]

              Active(file) %lu (since Linux 2.6.28)
                     [To be documented.]

              Inactive(file) %lu (since Linux 2.6.28)
                     [To be documented.]

              Unevictable %lu (since Linux 2.6.28)
                     (From Linux 2.6.28 to 2.6.30, CONFIG_UNEVICTABLE_LRU was required.)  [To be documented.]

              Mlocked %lu (since Linux 2.6.28)
                     (From Linux 2.6.28 to 2.6.30, CONFIG_UNEVICTABLE_LRU was required.)  [To be documented.]

              HighTotal %lu
                     (Starting with Linux 2.6.19, CONFIG_HIGHMEM is required.)  Total amount of highmem.  Highmem is all memory above ~860MB of physical memory.  Highmem areas are for use by user-space programs, or for the page cache.  The kernel must use tricks to access this memory, making it slower to access than lowmem.

              HighFree %lu
                     (Starting with Linux 2.6.19, CONFIG_HIGHMEM is required.)  Amount of free highmem.

              LowTotal %lu
                     (Starting with Linux 2.6.19, CONFIG_HIGHMEM is required.)  Total amount of lowmem.  Lowmem is memory which can be used for everything that highmem can be used for, but it is also available for the kernel's use for its own data structures.  Among many other things, it is where everything  from  Slab  is  allo‐
                     cated.  Bad things happen when you're out of lowmem.

              LowFree %lu
                     (Starting with Linux 2.6.19, CONFIG_HIGHMEM is required.)  Amount of free lowmem.

              MmapCopy %lu (since Linux 2.6.29)
                     (CONFIG_MMU is required.)  [To be documented.]

              SwapTotal %lu
                     Total amount of swap space available.

              SwapFree %lu
                     Amount of swap space that is currently unused.

              Dirty %lu
                     Memory which is waiting to get written back to the disk.

              Writeback %lu
                     Memory which is actively being written back to the disk.

              AnonPages %lu (since Linux 2.6.18)
                     Non-file backed pages mapped into user-space page tables.

              Mapped %lu
                     Files which have been mmaped, such as libraries.

              Shmem %lu (since Linux 2.6.32)
                     [To be documented.]

              Slab %lu
                     In-kernel data structures cache.

              SReclaimable %lu (since Linux 2.6.19)
                     Part of Slab, that might be reclaimed, such as caches.

              SUnreclaim %lu (since Linux 2.6.19)
                     Part of Slab, that cannot be reclaimed on memory pressure.

              KernelStack %lu (since Linux 2.6.32)
                     Amount of memory allocated to kernel stacks.

              PageTables %lu (since Linux 2.6.18)
                     Amount of memory dedicated to the lowest level of page tables.

              Quicklists %lu (since Linux 2.6.27)
                     (CONFIG_QUICKLIST is required.)  [To be documented.]

              NFS_Unstable %lu (since Linux 2.6.18)
                     NFS pages sent to the server, but not yet committed to stable storage.

              Bounce %lu (since Linux 2.6.18)
                     Memory used for block device "bounce buffers".

              WritebackTmp %lu (since Linux 2.6.26)
                     Memory used by FUSE for temporary writeback buffers.

              CommitLimit %lu (since Linux 2.6.10)
                     Based  on the overcommit ratio ('vm.overcommit_ratio'), this is the total amount of  memory currently available to be allocated on the system.  This limit is adhered to only if strict overcommit accounting is enabled (mode 2 in /proc/sys/vm/overcommit_ratio).  The CommitLimit is calculated using the following
                     formula:

                         CommitLimit = (overcommit_ratio * Physical RAM) + Swap

                     For example, on a system with 1GB of physical RAM and 7GB of swap with a overcommit_ratio of 30, this formula yields a CommitLimit of 7.3GB.  For more details, see the memory overcommit documentation in the kernel source file Documentation/vm/overcommit-accounting.

              Committed_AS %lu
                     The amount of memory presently allocated on the system.  The committed memory is a sum of all of the memory which has been allocated by processes, even if it has not been "used" by them as of yet.  A process which allocates 1GB of memory (using malloc(3) or similar), but touches only 300MB of that memory will
                     show  up  as  using  only 300MB of memory even if it has the address space allocated for the entire 1GB.  This 1GB is memory which has been "committed" to by the VM and can be used at any time by the allocating application.  With strict overcommit enabled on the system (mode 2 /proc/sys/vm/overcommit_memory),
                     allocations which would exceed the CommitLimit (detailed above) will not be permitted.  This is useful if one needs to guarantee that processes will not fail due to lack of memory once that memory has been successfully allocated.

              VmallocTotal %lu
                     Total size of vmalloc memory area.

              VmallocUsed %lu
                     Amount of vmalloc area which is used.

              VmallocChunk %lu
                     Largest contiguous block of vmalloc area which is free.

              HardwareCorrupted %lu (since Linux 2.6.32)
                     (CONFIG_MEMORY_FAILURE is required.)  [To be documented.]

              AnonHugePages %lu (since Linux 2.6.38)
                     (CONFIG_TRANSPARENT_HUGEPAGE is required.)  Non-file backed huge pages mapped into user-space page tables.

              HugePages_Total %lu
                     (CONFIG_HUGETLB_PAGE is required.)  The size of the pool of huge pages.

              HugePages_Free %lu
                     (CONFIG_HUGETLB_PAGE is required.)  The number of huge pages in the pool that are not yet allocated.

              HugePages_Rsvd %lu (since Linux 2.6.17)
                     (CONFIG_HUGETLB_PAGE is required.)  This is the number of huge pages for which a commitment to allocate from the pool has been made, but no allocation has yet been made.  These reserved huge pages guarantee that an application will be able to allocate a huge page from the pool of huge pages at fault time.

              HugePages_Surp %lu (since Linux 2.6.24)
                     (CONFIG_HUGETLB_PAGE is required.)  This is the number of huge pages in the pool above the value in /proc/sys/vm/nr_hugepages.  The maximum number of surplus huge pages is controlled by /proc/sys/vm/nr_overcommit_hugepages.

              Hugepagesize %lu
                     (CONFIG_HUGETLB_PAGE is required.)  The size of huge pages.
*/
package procmeminfo

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MemInfo is a map[string]uint64 with all the values found in /proc/meminfo
type MemInfo map[string]uint64

// Update s with current values, usign the pid stored in the Stat
func (m *MemInfo) Update() error {
	var err error

	path := filepath.Join("/proc/meminfo")
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()

		n := strings.Index(text, ":")
		if n == -1 {
			continue
		}

		key := text[:n] // metric
		data := strings.Split(strings.Trim(text[(n+1):], " "), " ")
		if len(data) == 1 {
			value, err := strconv.ParseUint(data[0], 10, 64)
			if err != nil {
				continue
			}
			(*m)[key] = value
		} else if len(data) == 2 {
			if data[1] == "kB" {
				value, err := strconv.ParseUint(data[0], 10, 64)
				if err != nil {
					continue
				}
				(*m)[key] = value * 1024
			}
		}

	}
	return nil

}

// Total return the size of the memory in bytes.
// It is an alias of (*m)["MemInfo"]
func (m *MemInfo) Total() uint64 {
	return (*m)["MemTotal"]
}

// Available return the available memory following this formula:
//
//	Available = Free + Buffers + Cached
func (m *MemInfo) Available() uint64 {
	d := *m
	return d["MemFree"] + d["Buffers"] + d["Cached"]
}

// Used is a generic way of reporting used memory. It follows the next formula:
//
//	Used = Total - Available
func (m *MemInfo) Used() uint64 {
	return m.Total() - m.Available()
}

// Swap returns the % of swap used
func (m *MemInfo) Swap() int {
	total := (*m)["SwapTotal"]
	free := (*m)["SwapFree"]
	if total == 0 {
		return 0
	}
	return int((100 * (total - free)) / total)
}
