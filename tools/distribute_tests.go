package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <worker-index> <total-workers>\n", os.Args[0])
		os.Exit(1)
	}

	workerIndex, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid worker index: %v\n", err)
		os.Exit(1)
	}

	totalWorkers, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid total workers: %v\n", err)
		os.Exit(1)
	}

	if workerIndex < 0 || workerIndex >= totalWorkers {
		fmt.Fprintf(os.Stderr, "Worker index must be between 0 and %d\n", totalWorkers-1)
		os.Exit(1)
	}

	// Read all tests from stdin
	var tests []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		tests = append(tests, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading tests: %v\n", err)
		os.Exit(1)
	}

	// Distribute tests across workers
	workerTests := distributeTests(tests, workerIndex, totalWorkers)

	// Output tests for this worker
	for _, test := range workerTests {
		fmt.Println(test)
	}
}

// distributeTests divides tests evenly across workers
func distributeTests(tests []string, workerIndex, totalWorkers int) []string {
	totalTests := len(tests)
	testsPerWorker := totalTests / totalWorkers
	remainder := totalTests % totalWorkers

	// Calculate start and end indices for this worker
	start := workerIndex * testsPerWorker
	if workerIndex < remainder {
		// Workers with index < remainder get one extra test
		start += workerIndex
		testsPerWorker++
	} else {
		// Adjust for the extra tests distributed to earlier workers
		start += remainder
	}

	end := start + testsPerWorker
	if end > totalTests {
		end = totalTests
	}

	return tests[start:end]
}