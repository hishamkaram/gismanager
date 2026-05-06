// Package gismanager is a v1.x compatibility shim above the v2-style
// subpackage layout. Public symbols are re-exported as type aliases
// and function wrappers from the [convert] (conversion subsystem)
// and [publish] (publish pipeline) subpackages.
//
// Type aliases preserve full identity — errors.As, errors.Is, method
// sets, slice element types all interoperate transparently across the
// boundary. Function wrappers are pure pass-throughs with no behavior
// change.
//
// All re-exported declarations are marked Deprecated; new code should
// import the relevant subpackage directly:
//
//	import "github.com/hishamkaram/gismanager/v2/convert"
//	import "github.com/hishamkaram/gismanager/v2/publish"
//
// v2 (the next major release) drops this root-level compatibility
// shim. The v2 module path is github.com/hishamkaram/gismanager/v2;
// v1.x users staying on the v1 line track the release/v1.x branch.
//
// See ~/.claude/plans/how-can-we-improve-steady-emerson.md for the
// restructure plan and CHANGELOG.md for the per-release migration
// notes.
package gismanager
