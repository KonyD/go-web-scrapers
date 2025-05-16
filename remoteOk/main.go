package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gocolly/colly"
)

func main() {
	file, err := os.Create("jobs.csv")
	if err != nil {
		log.Fatalf("Could not create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Job Title", "Company", "Location", "Salary"})

	// Create a new asynchronous collector with parallelism limit
	c := colly.NewCollector(
		colly.Async(true),
	)

	c.Limit((&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 3,
	}))

	var wg sync.WaitGroup // WaitGroup to track when all URLs are fully scraped

	c.OnHTML("tr.job", func(e *colly.HTMLElement) {
		jobTitle := strings.TrimSpace(e.DOM.Find("h2[itemprop='title']").Text())
		company := strings.TrimSpace(e.DOM.Find("h3[itemprop='name']").Text())
		locations := e.DOM.Find("div.location")
		location := locations.Eq(0).Text()
		salary := locations.Eq(1).Text()

		err := writer.Write([]string{jobTitle, company, location, salary})
		if err != nil {
			log.Printf("Could not write record to CSV: %v", err)
		}

		fmt.Printf("Job Title: %s\nCompany: %s\nLocation: %s\nSalary: %s\n----\n", jobTitle, company, location, salary)
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	urls := []string{
		"https://remoteok.com/remote-javascript-jobs",
		"https://remoteok.com/remote-python-jobs",
		"https://remoteok.com/remote-go-jobs",
	}

	// Add a counter for each URL visit, then visit asynchronously
	for _, url := range urls {
		wg.Add(1)
		if err := c.Visit(url); err != nil {
			log.Fatalf("Failed to visit page: %v", err)
		}
	}

	// Called once each page scraping finishes, signals WaitGroup to decrement
	c.OnScraped(func(r *colly.Response) {
		wg.Done()
	})

	// Wait until all pages have finished scraping
	wg.Wait()

	// Wait for all asynchronous requests to complete before exiting
	c.Wait()
}
