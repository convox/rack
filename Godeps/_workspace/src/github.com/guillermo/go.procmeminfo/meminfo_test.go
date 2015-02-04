package procmeminfo

import (
	"testing"
)

func TestUpdate(t *testing.T) {
	memInfo := &MemInfo{}
	err := memInfo.Update()
	if err != nil {
		t.Fatal(err)
	}

	validKeys := []string{"MemTotal", "MemFree", "Buffers", "Cached",
		"SwapCached", "Active", "Inactive", "Active(anon)", "Inactive(anon)",
		"Active(file)", "Inactive(file)", "Unevictable", "Mlocked", "SwapTotal",
		"SwapFree", "Dirty", "Writeback", "AnonPages", "Mapped", "Shmem", "Slab",
		"SReclaimable", "SUnreclaim", "KernelStack", "PageTables", "NFS_Unstable",
		"Bounce", "WritebackTmp", "CommitLimit", "Committed_AS", "VmallocTotal",
		"VmallocUsed", "VmallocChunk", "HardwareCorrupted", "AnonHugePages",
		"HugePages_Total", "HugePages_Free", "HugePages_Rsvd", "HugePages_Surp",
		"Hugepagesize", "DirectMap4k", "DirectMap2M"}

	for _, k := range validKeys {
		_, ok := (*memInfo)[k]
		if !ok {
			t.Error("Missing the key:", k)
		}
	}

}

func TestTotal(t *testing.T) {
	meminfo := &MemInfo{"MemTotal": 44}
	if meminfo.Total() != 44 {
		t.Error(meminfo.Total())
	}
}

func TestAvailable(t *testing.T) {
	meminfo := &MemInfo{"MemFree": 1, "Buffers": 1, "Cached": 1}
	if meminfo.Available() != 3 {
		t.Error(meminfo.Available())
	}
}

func TestUsed(t *testing.T) {
	meminfo := &MemInfo{"MemTotal": 10, "MemFree": 1, "Buffers": 1, "Cached": 1}
	if meminfo.Used() != 7 {
		t.Error(meminfo.Used())
	}
}

func TestSwap(t *testing.T) {
	meminfo := &MemInfo{"SwapTotal": 10, "SwapFree": 9}
	if meminfo.Swap() != 10 {
		t.Error(meminfo.Swap())
	}
}
