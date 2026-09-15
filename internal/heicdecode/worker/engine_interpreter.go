//go:build heicinterpreter

package worker

import "github.com/tetratelabs/wazero"

// The interpreter variant permits native performance/signing qualification
// without granting generated-code authority. It uses the same finite limits.
func runtimeConfig() wazero.RuntimeConfig { return wazero.NewRuntimeConfigInterpreter() }
