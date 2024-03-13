package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

func main() {
	// Strip log timestamp to catch repeated messages in main loop
	log := log.New(customLogWriter{}, "", 0)

	cfg, err := parseConfig()
	if err != nil {
		log.Panic(err)
	}

	ctx := context.Background()
	httpClient := createHTTPClient()
	api, err := createAPIClient(cfg.APIKey)
	if err != nil {
		log.Panic(err)
	}

	// Fetch initial DNS states to be kept updated without additional API calls
	dnsStates := make(map[string]*DNSState)
	for _, record := range cfg.Records {
		activeRecord, err := fetchARecords(api, ctx, record)
		if err != nil {
			log.Panicf("Error fetching DNS record for %v: %v", record.Domain, err)
			return
		}

		if len(activeRecord) == 0 {
			log.Panicf("No DNS records found for %v: %v", record.Domain, err)
			return
		}

		// Pointer to struct to reflect state across funcs, goroutines, etc.
		dnsStates[record.Domain] = &DNSState{
			ActiveIP: activeRecord[0].Content,
			ID:       activeRecord[0].ID,
		}
	}

	lastLogMsgs := make(map[string]string)
	var wg sync.WaitGroup

	for {
		logCh := make(chan logEntry, len(cfg.Records))

		for _, r := range cfg.Records {
			wg.Add(1)

			go func(httpClient *http.Client, r Record, s *DNSState, logCh chan logEntry) {
				defer wg.Done()

				ip, err := getResponsiveIP(httpClient, r, logCh)
				if err != nil {
					sendLogEntry(logCh, r.Domain, fmt.Sprintf("%s: Error getting responsive IP: %v", r.Domain, err))
					return
				}

				manageDNS(ctx, api, httpClient, r, s, ip, logCh)
			}(httpClient, r, dnsStates[r.Domain], logCh)
		}

		// Log without consecutive repetitions since we're running in a loop
		go func() {
			for entry := range logCh {
				if lastLogMsgs[entry.domain] != entry.msg {
					log.Printf("[%s]: %s\n", entry.timestamp.Format("2006-01-02 15:04:05"), entry.msg)
					lastLogMsgs[entry.domain] = entry.msg
				}
			}
		}()

		go func() {
			wg.Wait()
			close(logCh)
		}()

		time.Sleep(time.Duration(cfg.CheckInterval) * time.Second)
	}
}
