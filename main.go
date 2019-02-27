package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	//_ "net/http/pprof"
	"os"
	"strings"
	"sync"
)

func countInSource(job job) {

	counts := 0
	var reader io.Reader

	if job.file {
		file, err := os.Open(job.location)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		reader = bufio.NewReader(file) // Fast read access
	} else {
		resp, err := http.Get(job.location)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()

		reader = bufio.NewReader(resp.Body)
	}

	// do I need buffered channels here?
	introspections := make(chan string)
	results := make(chan int)

	// I think we need a wait group, not sure.
	wg := new(sync.WaitGroup)

	// start up some workers that will block and wait?

	for w := 1; w <= 5; w++ {
		wg.Add(1)
		go matcher(introspections, results, wg, "Go")
	}

	// Go over a file line by line and queue up a ton of work
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			// Later I want to create a buffer of lines, not just line-by-line here ...
			introspections <- scanner.Text()
		}
		close(introspections)
	}()

	// Now collect all the results...
	// But first, make sure we close the result channel when everything was processed
	go func() {
		wg.Wait()
		close(results)
	}()

	// Add up the results from the results channel.
	for v := range results {
		counts += v
	}

	fmt.Printf("Count for %s: %d\n", job.location, counts)

	return
}

func matcher(jobs <-chan string, results chan<- int, wg *sync.WaitGroup, expression string) {

	// eventually I want to have a []string channel to work on a chunk of lines not just one line of text
	for j := range jobs {
		if strings.Contains(j, expression) {
			results <- 1
		}
	}

	// Decreasing internal counter for wait-group as soon as goroutine finishes
	wg.Done()
}

const maxWorkers = 10

type job struct {
	location string
	file     bool
}

func main() {

	//go func() {
	//	log.Println(http.ListenAndServe("localhost:6060", nil))
	//}()

	scanner := bufio.NewScanner(os.Stdin)

	jobs := make(chan job)

	// start workers
	wg := &sync.WaitGroup{}
	wg.Add(maxWorkers)
	for i := 1; i <= maxWorkers; i++ {
		go func(i int) {

			for j := range jobs {
				countInSource(j)
			}

			wg.Done()
		}(i)
	}

	for scanner.Scan() {
		location := scanner.Text()
		if location == "" {
			continue
		}
		if strings.Contains(location, "://") {
			jobs <- job{location, false}
		} else {
			jobs <- job{location, true}
		}
	}

	close(jobs)

	// wait for workers to complete
	wg.Wait()
}
