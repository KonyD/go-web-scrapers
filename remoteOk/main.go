package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/forPelevin/gomoji"
	"github.com/gocolly/colly"
)

// Define a struct to hold job data
type Job struct {
	Title     string   `json:"title"`
	Company   string   `json:"company"`
	Locations []string `json:"locations"`
	Salary    string   `json:"salary"`
}

func main() {
	var jobs []Job
	var mu sync.Mutex // Protect jobs slice during concurrent writes

	urls := []string{
		"https://remoteok.com/remote-javascript-jobs",
		"https://remoteok.com/remote-python-jobs",
		"https://remoteok.com/remote-go-jobs",
	}

	c := colly.NewCollector(
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: len(urls),
	})

	var wg sync.WaitGroup

	c.OnHTML("tr.job", func(e *colly.HTMLElement) {
		jobTitle := strings.TrimSpace(e.ChildText("h2[itemprop='title']"))
		company := strings.TrimSpace(e.ChildText("h3[itemprop='name']"))

		var locations []string
		salary := ""

		e.ForEach("div.location", func(_ int, el *colly.HTMLElement) {
			text := strings.TrimSpace(el.Text)
			if strings.Contains(text, "$") || strings.Contains(text, "💰") {
				text = strings.TrimSpace(gomoji.RemoveEmojis(text))
				salary = text
			} else {
				text = strings.TrimSpace(gomoji.RemoveEmojis(text))
				locations = append(locations, text)
			}
		})

		job := Job{
			Title:     jobTitle,
			Company:   company,
			Locations: locations,
			Salary:    salary,
		}

		// Append safely using mutex
		mu.Lock()
		jobs = append(jobs, job)
		mu.Unlock()

		fmt.Printf("Job: %+v\n----\n", job)
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	for _, url := range urls {
		wg.Add(1)
		if err := c.Visit(url); err != nil {
			log.Fatalf("Failed to visit %s: %v", url, err)
		}
	}

	c.OnScraped(func(_ *colly.Response) {
		wg.Done()
	})

	wg.Wait()
	c.Wait()

	// Write jobs to JSON
	file, err := os.Create("jobs.json")
	if err != nil {
		log.Fatalf("Could not create JSON file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print
	if err := encoder.Encode(jobs); err != nil {
		log.Fatalf("Could not encode JSON: %v", err)
	}

	fmt.Println("Scraped jobs saved to jobs.json")
}
