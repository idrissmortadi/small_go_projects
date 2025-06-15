package main

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkMandelbrot(b *testing.B) {
	start := time.Now()
	for i := 0; i < b.N; i++ {
		main()
	}
	elapsed := time.Since(start)
	avg := elapsed / time.Duration(b.N)
	// Report the average in nanoseconds per iteration.
	b.ReportMetric(float64(avg.Nanoseconds()), "ns/op")
	fmt.Printf("Average time: %v\n", avg)
}
