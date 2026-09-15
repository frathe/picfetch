//go:build !heicinterpreter

package worker

import "github.com/tetratelabs/wazero"

func runtimeConfig() wazero.RuntimeConfig { return wazero.NewRuntimeConfigCompiler() }
