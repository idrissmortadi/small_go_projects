package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
)

// Read sites.txt line-by-line
// For each URL:
//	Make a GET request with a timeout (use context.WithTimeout)
//	Parse the HTML title (use golang.org/x/net/html or regex if you want speed over accuracy)
//	Send the result (URL + Title or error) to a channel
// # Use goroutines to handle multiple URLs concurrently
// Collect results from the channel and write them to results.csv

func fillSitesChan(siteChannel chan string) {
	file, err := os.Open("sites.txt")
	if err != nil {
		log.Fatalf("failed to open sites.txt: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("adding site to channel: %s", line)
		siteChannel <- line
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	close(siteChannel)
}

func getSiteContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body for %s: %w", url, err)
	}

	re := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("no <title> tag found for %s", url)
	}
	return matches[1], nil
}

func processSite(siteChannel chan string, resultsChannel chan string, wg *sync.WaitGroup) {
	for url := range siteChannel {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			getTitle, err := getSiteContent(url)
			if err != nil {
				log.Printf("error getting title for %s: %v", url, err)
			}
			result := url + ", " + getTitle
			log.Printf("result for %s: %s", url, result)
			resultsChannel <- result
		}(url)
	}
}

func printResults(resultsChannel chan string) {
	file, err := os.Create("titles.txt")
	if err != nil {
		log.Fatalf("failed to create titles.txt: %v", err)
	}
	defer file.Close()
	for result := range resultsChannel {
		log.Println("result:", result)
		_, err := file.WriteString(result + "\n")
		if err != nil {
			log.Printf("failed to write result to file: %v", err)
		}
	}
}

func main() {
	log.Println("starting site title fetcher...")
	siteChannel := make(chan string)
	resultsChannel := make(chan string)
	var wg sync.WaitGroup

	go func() {
		fillSitesChan(siteChannel)
	}()
	go printResults(resultsChannel)
	processSite(siteChannel, resultsChannel, &wg)
	wg.Wait()
	close(resultsChannel)
}
