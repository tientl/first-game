//go:build !windows

package main

// processMemory is not measured off Windows.
func processMemory() (workingSet, private uint64) { return 0, 0 }
