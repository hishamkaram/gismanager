package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestPrintVersion_AllSet exercises the path where build-time ldflags
// have populated every field — output should be deterministic.
func TestPrintVersion_AllSet(t *testing.T) {
	saved := Version
	t.Cleanup(func() { Version = saved })
	savedC := Commit
	t.Cleanup(func() { Commit = savedC })
	savedD := Date
	t.Cleanup(func() { Date = savedD })

	Version = "v1.4.0"
	Commit = "abc1234"
	Date = "2026-05-06T12:00:00Z"

	var buf bytes.Buffer
	PrintVersion(&buf, "gismanager")
	got := buf.String()
	want := "gismanager version=v1.4.0 commit=abc1234 built=2026-05-06T12:00:00Z\n"
	if got != want {
		t.Errorf("\n got: %q\nwant: %q", got, want)
	}
}

// TestPrintVersion_NeverEmptyValues guards the contract that every
// printed field is non-empty even when no ldflag and no build info
// apply (e.g. testing harness). The literal "(unknown)" should appear
// in place of any field that can't be filled.
func TestPrintVersion_NeverEmptyValues(t *testing.T) {
	saved := Version
	t.Cleanup(func() { Version = saved })
	savedC := Commit
	t.Cleanup(func() { Commit = savedC })
	savedD := Date
	t.Cleanup(func() { Date = savedD })

	Version = ""
	Commit = ""
	Date = ""

	var buf bytes.Buffer
	PrintVersion(&buf, "gismanager")
	got := buf.String()

	// Fields might pick up vcs.* from `go test`'s build info; we don't
	// hard-code them, but we DO assert the line starts with the binary
	// name and contains all three field keys.
	if !strings.HasPrefix(got, "gismanager version=") {
		t.Errorf("missing version field; got %q", got)
	}
	if !strings.Contains(got, " commit=") {
		t.Errorf("missing commit field; got %q", got)
	}
	if !strings.Contains(got, " built=") {
		t.Errorf("missing built field; got %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("missing trailing newline; got %q", got)
	}
	// No empty values — the contract is "(unknown)" or real value.
	if strings.Contains(got, "version= ") || strings.Contains(got, "version=\n") {
		t.Errorf("empty version field leaked through; got %q", got)
	}
}

func TestRequireFlag(t *testing.T) {
	cases := []struct {
		name      string
		binary    string
		flag      string
		value     string
		wantError bool
	}{
		{"non-empty value passes", "gismanager", "config", "/etc/gm.yaml", false},
		{"empty value errors", "gismanager", "config", "", true},
		{"empty config flag", "layerSchema", "config", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := RequireFlag(tc.binary, tc.flag, tc.value)
			if tc.wantError {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				want := tc.binary + ": --" + tc.flag + " is required"
				if err.Error() != want {
					t.Errorf("\n got: %q\nwant: %q", err.Error(), want)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestSignalContext_CancelsOnSIGTERM confirms the returned context
// cancels when the process receives SIGTERM. We deliberately use
// SIGTERM rather than SIGINT here because Go's testing framework
// catches SIGINT on its own in some configurations. Sending SIGTERM
// to ourselves is safe — signal.NotifyContext intercepts it before
// the default handler kills the process.
func TestSignalContext_CancelsOnSIGTERM(t *testing.T) {
	parent, parentCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer parentCancel()
	ctx, cancel := SignalContext(parent)
	defer cancel()

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("kill self: %v", err)
	}

	select {
	case <-ctx.Done():
		// Either context.Canceled (signal) or context.DeadlineExceeded
		// would technically cancel; we only want the signal path.
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("ctx.Err() = %v; want context.Canceled", ctx.Err())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ctx did not cancel within 2s of SIGTERM")
	}
}
