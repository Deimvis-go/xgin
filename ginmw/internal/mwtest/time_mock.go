// Package mwtest exposes hooks used by middleware tests to control time.
package mwtest

import "time"

// TimeAfter is a swappable indirection over [time.After]. Tests replace it
// to drive timers deterministically.
var TimeAfter = time.After
