package gismanager

import (
	"errors"
	"testing"

	"github.com/hishamkaram/gismanager/internal/errs"
)

// TestErrAliasChain_v1Compat confirms the v1.x compat shim resolves
// public API symbols (gismanager.GISError, gismanager.ErrConvertFailed,
// the package-private newGISError) to the SAME underlying types and
// values living in [internal/errs]. The exhaustive *GISError mechanic
// tests live in internal/errs/errs_test.go.
//
// This single test guards against a future refactor accidentally
// breaking the alias chain — e.g. introducing a separate type at root
// instead of `type GISError = errs.GISError`, which would break
// errors.As across the alias boundary for external callers.
func TestErrAliasChain_v1Compat(t *testing.T) {
	cause := errors.New("underlying")
	gerr := newGISError("OpenSource", "/tmp/foo.shp", ErrUnsupportedFormat, cause)

	// Sentinel match works against both the root alias and the
	// underlying internal/errs sentinel — they're the same instance.
	if !errors.Is(gerr, ErrUnsupportedFormat) {
		t.Errorf("errors.Is(gerr, ErrUnsupportedFormat) = false; want true (root alias)")
	}
	if !errors.Is(gerr, errs.ErrUnsupportedFormat) {
		t.Errorf("errors.Is(gerr, errs.ErrUnsupportedFormat) = false; want true (alias points at same underlying value)")
	}

	// Type alias means errors.As works for both the root-spelled type
	// and the internal-spelled type.
	var rootShape *GISError
	if !errors.As(gerr, &rootShape) {
		t.Fatalf("errors.As to root-spelled *GISError failed")
	}
	var internalShape *errs.GISError
	if !errors.As(gerr, &internalShape) {
		t.Fatalf("errors.As to *errs.GISError failed (alias should permit both spellings)")
	}
	if rootShape != internalShape {
		t.Errorf("alias spellings should resolve to the same pointer; got root=%p internal=%p",
			rootShape, internalShape)
	}
}
