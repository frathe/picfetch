package worker

import (
	"encoding/binary"
	"testing"

	"golang.org/x/net/bpf"
)

// Evaluate policy on every host without installing a native filter. Each
// instruction input represents a PicFetch-owned boundary decision, no image.
func TestLinuxFilterRuntimeAndAuthority(t *testing.T) {
	for _, calls := range []linuxCalls{
		{0xc000003e, 0xd0f00, 56, 9, 10, 202, 72, 234, []uint32{0, 1, 3, 60, 231}},
		{0xc00000b7, 0x50f00, 220, 222, 226, 98, 25, 131, []uint32{63, 64, 57, 93, 94}},
	} {
		raw, err := linuxFilter(calls, 42)
		if err != nil {
			t.Fatal(err)
		}
		instructions, decoded := bpf.Disassemble(raw)
		if !decoded {
			t.Fatal("unknown BPF instructions")
		}
		vm, err := bpf.NewVM(instructions)
		if err != nil {
			t.Fatal(err)
		}
		check := func(name string, arch, call uint32, args [6]uint64, allow bool) {
			t.Helper()
			// BPF's portable VM loads network order; seccomp loads host order.
			// Encode each scalar separately to preserve the kernel's low/high
			// 32-bit layout of little-endian 64-bit syscall arguments.
			data := make([]byte, 64)
			binary.BigEndian.PutUint32(data, call)
			binary.BigEndian.PutUint32(data[4:], arch)
			for i, arg := range args {
				binary.BigEndian.PutUint32(data[16+i*8:], uint32(arg))
				binary.BigEndian.PutUint32(data[20+i*8:], uint32(arg>>32))
			}
			got, err := vm.Run(data)
			want := 0x00050001
			if allow {
				want = 0x7fff0000
			}
			if err != nil || got != want {
				t.Errorf("%s arch=%#x: decision=%#x error=%v, want %#x", name, arch, got, err, want)
			}
		}
		for _, call := range calls.ordinary {
			check("runtime", calls.architecture, call, [6]uint64{}, true)
		}
		check("unknown architecture", calls.architecture^1, calls.ordinary[0], [6]uint64{}, false)
		check("x32", calls.architecture, calls.ordinary[0]|0x40000000, [6]uint64{}, false)
		for _, flags := range []uint64{uint64(calls.threadFlags), 0, 17, uint64(calls.threadFlags) | 0x10000000, uint64(calls.threadFlags) | (1 << 32)} {
			check("clone", calls.architecture, calls.clone, [6]uint64{flags}, flags == uint64(calls.threadFlags))
		}
		for _, flags := range []uint64{0x22, 0x32, 0x02, 0x21, 0x4022, 0x22 | (1 << 32)} {
			check("mmap flags", calls.architecture, calls.mmap, [6]uint64{0, 4096, 3, flags}, flags == 0x22 || flags == 0x32)
		}
		for _, protection := range []uint64{0, 1, 3, 5, 7, 0x100000003} {
			check("mmap protection", calls.architecture, calls.mmap, [6]uint64{0, 4096, protection, 0x22}, protection == 0 || protection == 1 || protection == 3)
			check("mprotect", calls.architecture, calls.mprotect, [6]uint64{0, 4096, protection}, protection == 0 || protection == 1 || protection == 3 || protection == 5)
		}
		for _, operation := range []uint64{0, 1, 128, 129, 0x100000080} {
			check("private futex", calls.architecture, calls.futex, [6]uint64{0, operation}, operation == 128 || operation == 129)
		}
		for _, command := range []uint64{0, 1, 2, 3, 4, 5, 1030} {
			check("fcntl", calls.architecture, calls.fcntl, [6]uint64{0, command}, command >= 1 && command <= 4)
		}
		check("own thread signal", calls.architecture, calls.tgkill, [6]uint64{42, 43, 23}, true)
		check("other process signal", calls.architecture, calls.tgkill, [6]uint64{41, 43, 23}, false)
		// Common and architecture-specific file/network/process/privilege paths
		// must remain denied. Values differ between the two Linux ABIs.
		denied := []uint32{425, 426, 427, 435, 9999}
		if calls.architecture == 0xc000003e {
			denied = append(denied, 2, 41, 42, 53, 57, 58, 59, 101, 157, 257, 272, 302, 308, 319, 322)
		} else {
			denied = append(denied, 56, 117, 167, 198, 199, 203, 221, 261, 268, 279, 281)
		}
		for _, call := range denied {
			check("denied authority", calls.architecture, call, [6]uint64{}, false)
		}
	}
}
