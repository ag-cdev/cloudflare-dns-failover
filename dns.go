package main

import (
	"context"
	"fmt"
	"sync"

	cf "github.com/cloudflare/cloudflare-go"
)

type DNSState struct {
	ActiveIP string
	ID       string
	Mutex    sync.RWMutex
}

func (s *DNSState) UpdateActiveIP(newIP string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.ActiveIP = newIP
}

func (s *DNSState) GetActiveIP() string {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	return s.ActiveIP
}

func (s *DNSState) GetID() string {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	return s.ID
}

func fetchARecords(api *cf.API, ctx context.Context, r Record) ([]cf.DNSRecord, error) {
	records, _, err := api.ListDNSRecords(ctx, cf.ZoneIdentifier(r.ZoneID), cf.ListDNSRecordsParams{Name: r.Domain, Type: "A"})
	return records, err
}

func updateARecord(ctx context.Context, api *cf.API, r Record, s *DNSState, ip string) error {
	updateRecord := cf.UpdateDNSRecordParams{
		Type:    "A",
		Name:    r.Domain,
		Content: ip,
		ID:      s.GetID(),
		TTL:     1,
		Proxied: &r.Proxied,
	}

	_, err := api.UpdateDNSRecord(ctx, cf.ZoneIdentifier(r.ZoneID), updateRecord)
	return err
}

func manageDNS(ctx context.Context, api *cf.API, r Record, s *DNSState, responsiveIP string, logCh chan<- logEntry) {
	activeIP := s.GetActiveIP()
	if activeIP == responsiveIP {
		sendLogEntry(logCh, r.Domain, fmt.Sprintf("%s: already set to %s", r.Domain, responsiveIP))
		return
	}

	err := updateARecord(ctx, api, r, s, responsiveIP)
	if err != nil {
		sendLogEntry(logCh, r.Domain, fmt.Sprintf("%s: DNS update failed: %v", r.Domain, err))
		return
	}

	s.UpdateActiveIP(responsiveIP)
	sendLogEntry(logCh, r.Domain, fmt.Sprintf("%s: updated to %s (previous: %s)", r.Domain, responsiveIP, activeIP))
}
