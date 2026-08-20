package contract_test

import (
	"context"
	"os"
	"testing"

	"github.com/gombit-dev/gombit/contract"
)

func TestMain(m *testing.M) {
	// Install is sync.Once — install once for the whole package so shuffled
	// test order cannot leave RequestID unset.
	contract.Install(contract.InstallOptions{
		RequestID: func(context.Context) string { return "req-test-1" },
	})
	os.Exit(m.Run())
}
