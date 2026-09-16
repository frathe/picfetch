package worker

import "golang.org/x/net/bpf"

// linuxCalls keeps kernel numbers supplied by the native build separate from
// the portable policy evaluator. Arguments are 64-bit little-endian fields.
type linuxCalls struct {
	architecture, threadFlags                   uint32
	clone, mmap, mprotect, futex, fcntl, tgkill uint32
	ordinary                                    []uint32
}

func linuxFilter(calls linuxCalls, pid uint32) ([]bpf.RawInstruction, error) {
	const deny = 0x00050001 // SECCOMP_RET_ERRNO | EPERM
	const allow = 0x7fff0000
	instructions := []bpf.Instruction{
		bpf.LoadAbsolute{Off: 4, Size: 4},
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: calls.architecture, SkipTrue: 1},
		bpf.RetConstant{Val: deny},
		bpf.LoadAbsolute{Off: 0, Size: 4},
		bpf.JumpIf{Cond: bpf.JumpGreaterOrEqual, Val: 0x40000000, SkipFalse: 1},
		bpf.RetConstant{Val: deny},
	}
	for _, call := range calls.ordinary {
		instructions = append(instructions,
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: call, SkipFalse: 1},
			bpf.RetConstant{Val: allow})
	}
	// Each matched rule terminates. Its argument checks cannot fall through to
	// another syscall rule with an argument still loaded in the accumulator.
	rule := func(call uint32, conditions ...[]bpf.Instruction) {
		var body []bpf.Instruction
		for _, condition := range conditions {
			body = append(body, condition...)
		}
		body = append(body, bpf.RetConstant{Val: allow})
		instructions = append(instructions, bpf.JumpIf{Cond: bpf.JumpEqual, Val: call, SkipFalse: uint8(len(body))})
		instructions = append(instructions, body...)
	}
	// Restrict scalar arguments, including their high word. Pointer contents
	// cannot be inspected by seccomp and are not treated as an authority check.
	argument := func(index uint32, values ...uint32) []bpf.Instruction {
		check := []bpf.Instruction{
			bpf.LoadAbsolute{Off: 20 + index*8, Size: 4},
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: 0, SkipTrue: 1},
			bpf.RetConstant{Val: deny},
			bpf.LoadAbsolute{Off: 16 + index*8, Size: 4},
		}
		for i, value := range values {
			check = append(check, bpf.JumpIf{Cond: bpf.JumpEqual, Val: value, SkipTrue: uint8(len(values) - i)})
		}
		return append(check, bpf.RetConstant{Val: deny})
	}
	// Go's exact CLONE_THREAD form permits runtime threads, never a new process
	// or namespace. clone3 and all unlisted process-launch calls remain denied.
	rule(calls.clone, argument(0, calls.threadFlags))
	// Only private anonymous mappings; huge-page attempts fail and wazero falls
	// back to normal pages. Compiled code transitions RW -> RX via mprotect.
	rule(calls.mmap, argument(2, 0, 1, 3), argument(3, 0x22, 0x32))
	rule(calls.mprotect, argument(2, 0, 1, 3, 5))
	rule(calls.futex, argument(1, 128, 129))
	rule(calls.fcntl, argument(1, 1, 2, 3, 4))
	rule(calls.tgkill, argument(0, pid))
	instructions = append(instructions, bpf.RetConstant{Val: deny})
	return bpf.Assemble(instructions)
}
