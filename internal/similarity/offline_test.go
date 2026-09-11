package similarity

import (
	"encoding/binary"
	"testing"

	"golang.org/x/net/bpf"
)

// Evaluate seccomp decisions without installing a filter or accessing the network.
func TestLinuxNetworkFilter(t *testing.T) {
	for _, abi := range []struct {
		name                     string
		arch, socket, pair       uint32
		read, open, clone, futex uint32
	}{
		{"amd64", 0xc000003e, 41, 53, 0, 257, 56, 202},
		{"arm64", 0xc00000b7, 198, 199, 63, 56, 220, 98},
	} {
		t.Run(abi.name, func(t *testing.T) {
			raw, err := linuxNetworkFilter(abi.arch, abi.socket, abi.pair, 425)
			if err != nil {
				t.Fatal(err)
			}
			instructions, allDecoded := bpf.Disassemble(raw)
			if !allDecoded {
				t.Fatal("filter contains unknown instructions")
			}
			vm, err := bpf.NewVM(instructions)
			if err != nil {
				t.Fatal(err)
			}
			for _, check := range []struct {
				name       string
				arch, call uint32
				allow      bool
			}{
				{"socket", abi.arch, abi.socket, false},
				{"socketpair", abi.arch, abi.pair, false},
				{"io_uring_setup", abi.arch, 425, false},
				{"read", abi.arch, abi.read, true},
				{"openat", abi.arch, abi.open, true},
				{"clone", abi.arch, abi.clone, true},
				{"futex", abi.arch, abi.futex, true},
				{"different_arch", abi.arch ^ 1, abi.read, false},
				{"32_bit_arch", abi.arch &^ 0x80000000, abi.read, false},
				{"x32_socket", abi.arch, abi.socket | 0x40000000, false},
				{"invalid_syscall", abi.arch, 0xffffffff, false},
			} {
				t.Run(check.name, func(t *testing.T) {
					// The VM reads network order; these bytes represent the same
					// scalar fields the seccomp kernel reads in host byte order.
					data := make([]byte, 64)
					binary.BigEndian.PutUint32(data, check.call)
					binary.BigEndian.PutUint32(data[4:], check.arch)
					got, err := vm.Run(data)
					want := 0x00050001 // SECCOMP_RET_ERRNO | EPERM
					if check.allow {
						want = 0x7fff0000 // SECCOMP_RET_ALLOW
					}
					if err != nil || got != want {
						t.Fatalf("seccomp decision = %#x, %v; want %#x", got, err, want)
					}
				})
			}
		})
	}
}
