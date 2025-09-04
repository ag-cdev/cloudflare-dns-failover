package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

func main() {
	// Use a distinct name for the logger to avoid shadowing the log package
	logger := log.New(customLogWriter{}, "", 0)

	cfg, err := parseConfig()
	if err != nil {
		logger.Panic(err)
	}

	ctx := context.Background()
	httpClient := createHTTPClient()
	api, err := createAPIClient(cfg.APIKey)
	if err != nil {
		logger.Panic(err)
	}

	// Fetch initial DNS states
	dnsStates := make(map[string]*DNSState)
	for _, record := range cfg.Records {
		activeRecords, err := fetchDNSRecords(api, ctx, record)
		if err != nil {
			logger.Panicf("Error fetching DNS record for %v: %v", record.Domain, err)
		}
		if len(activeRecords) == 0 {
			logger.Panicf("No DNS records found for %v", record.Domain)
		}
		for _, rec := range activeRecords {
			if rec.Type != "A" && rec.Type != "AAAA" {
				continue // skip other types like CNAME, TXT, etc.
			}

			key := fmt.Sprintf("%s|%s", record.Domain, rec.Type) // unique key per type
			dnsStates[key] = &DNSState{
				ActiveIP:   rec.Content,
				ID:         rec.ID,
				RecordType: rec.Type,
			}
		}
	}

	lastLogMsgs := make(map[string]string)

	for {
		logCh := make(chan logEntry, len(cfg.Records))
		var wg sync.WaitGroup // reset each loop

		// Spawn workers
		for _, r := range cfg.Records {
			wg.Add(1)
			go func(r Record) {
				defer wg.Done()
				// Step 1: Get the responsive IP from your logic
				ip, err := getResponsiveIP(httpClient, r, r.Protocol, r.Port, logCh)
				if err != nil {
					sendLogEntry(logCh, r.Domain, fmt.Sprintf("Error getting responsive IP: %v", err))
					return
				}
				// Step 2: Detect type from the IP string
				recType := detectRecordType(ip)
				// Step 3: Build the composite key and fetch the right DNSState
				key := fmt.Sprintf("%s|%s", r.Domain, recType)
				state, ok := dnsStates[key]
				if !ok {
					sendLogEntry(logCh, r.Domain, fmt.Sprintf("No DNS state found for %s", key))
					return
				}
				// Step 4: Pass along to your update logic
				manageDNS(ctx, api, r, state, ip, logCh)

			}(r)
		}

		// Log reader (single goroutine per loop)
		go func() {
			for entry := range logCh {
				if lastLogMsgs[entry.domain] != entry.msg {
					logger.Printf("[%s]: %s\n", entry.timestamp.Format("2006-01-02 15:04:05"), entry.msg)
					lastLogMsgs[entry.domain] = entry.msg
				}
			}
		}()

		// Close logCh after all workers finish
		wg.Wait()
		close(logCh)

		time.Sleep(time.Duration(cfg.CheckInterval) * time.Second)
	}
}
