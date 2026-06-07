package main

import (
	"fmt"

	"github.com/siriscent7/tenantguard/limiter"
)

func main() {
	b := limiter.NewBucket(3, 1) // 3 tokens, refill 1/sec
	for i := 0; i < 5; i++ {
		fmt.Printf("Request %d allowed: %v\n", i, b.Allow())
	}
}
