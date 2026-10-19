// Package engine implements the platform-independent text expansion matching
// engine. It must not import any operating-system specific package.
package engine

// matcher.go holds the suffix-matching logic. The implementation lives in
// index.go so that exact and regex matching share one Index.
