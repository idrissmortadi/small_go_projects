package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

// Helper to create a temporary sites.txt file
func createTempSitesFile(urls []string) (func(), error) {
	content := strings.Join(urls, "\n")
	err := os.WriteFile("temp_sites.txt", []byte(content), 0644)
	cleanup := func() {
		os.Remove("temp_sites.txt")
	}
	return cleanup, err
}

func TestFillSitesChan(t *testing.T) {
	urls := []string{"http://example.com", "http://test.com"}
	cleanup, err := createTempSitesFile(urls)
	if err != nil {
		t.Fatalf("failed to create temp sites.txt: %v", err)
	}
	defer cleanup()

	ch := make(chan string, len(urls))
	go fillSitesChan(ch)
	var got []string
	for url := range ch {
		got = append(got, url)
	}
	if len(got) != len(urls) {
		t.Errorf("expected %d urls, got %d", len(urls), len(got))
	}
}

func TestGetSiteContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><title>TestTitle</title></html>"))
	}))
	defer ts.Close()

	title, err := getSiteContent(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "TestTitle" {
		t.Errorf("expected 'TestTitle', got '%s'", title)
	}
}

func TestProcessSite(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><title>UnitTestTitle</title></html>"))
	}))
	defer ts.Close()

	siteChannel := make(chan string, 1)
	resultsChannel := make(chan string, 1)
	var wg sync.WaitGroup

	siteChannel <- ts.URL
	close(siteChannel)

	processSite(siteChannel, resultsChannel, &wg)
	wg.Wait()
	close(resultsChannel)

	var got []string
	for result := range resultsChannel {
		got = append(got, result)
	}
	if len(got) != 1 || !strings.Contains(got[0], "UnitTestTitle") {
		t.Errorf("expected result to contain 'UnitTestTitle', got %v", got)
	}
}

func TestProcessSiteAndPrintResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><title>IntegrationTestTitle</title></html>"))
	}))
	defer ts.Close()

	siteChannel := make(chan string, 1)
	resultsChannel := make(chan string, 1)
	var wg sync.WaitGroup

	// Don't need wg.Add(1) here since processSite will add to the WaitGroup internally
	siteChannel <- ts.URL
	close(siteChannel)

	// Run processSite without a goroutine since it spawns its own goroutines internally
	processSite(siteChannel, resultsChannel, &wg)

	// Wait for all goroutines spawned by processSite to finish
	go func() {
		wg.Wait()
		close(resultsChannel)
	}()

	var got []string
	for result := range resultsChannel {
		got = append(got, result)
	}

	if len(got) != 1 || !strings.Contains(got[0], "IntegrationTestTitle") {
		t.Errorf("expected result to contain 'IntegrationTestTitle', got %v", got)
	}
}

func TestPrintResultsToFile(t *testing.T) {
	resultsChannel := make(chan string, 2)
	resultsChannel <- "http://example.com, ExampleTitle"
	resultsChannel <- "http://test.com, TestTitle"
	close(resultsChannel)

	// Remove the file if it exists
	os.Remove("titles.txt")

	printResults(resultsChannel)

	data, err := os.ReadFile("titles.txt")
	if err != nil {
		t.Fatalf("failed to read titles.txt: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "ExampleTitle") || !strings.Contains(content, "TestTitle") {
		t.Errorf("titles.txt does not contain expected titles, got: %s", content)
	}

	// Cleanup
	os.Remove("titles.txt")
}
