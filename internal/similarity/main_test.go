package similarity

import (
	"fmt"
	"os"
	"testing"

	"github.com/frathe/picfetch/internal/heic"
)

func TestMain(m *testing.M) {
	if heic.WorkerMain() {
		return
	}
	if os.Getenv(workerEnvironment) == "1" && os.Getenv("PICFETCH_TEST_HEIC_CAPTURE") == "1" {
		if err := heicCaptureWorker(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if WorkerMain() {
		return
	}
	if mode := os.Getenv("PICFETCH_TEST_ANALYSIS_PROTOCOL"); mode != "" {
		// Protocol helpers must not start testing.M's alarm or write its
		// status lines to stdout. The parent context owns their lifetime.
		if err := analysisProtocolHelperProcess(mode); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}
	os.Exit(m.Run())
}
