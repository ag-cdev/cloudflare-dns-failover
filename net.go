package main

import (
	"fmt"
	"net/http"
	"time"

	cf "github.com/cloudflare/cloudflare-go"
)

func createAPIClient(apiKey string) (*cf.API, error) {
	api, err := cf.NewWithAPIToken(apiKey)
	if err != nil {
		return api, fmt.Errorf("Error creating Cloudflare client: %v", err)
	}

	return api, nil
}

func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Do not follow redirects
		},
	}
}

func isHTTPResponsive(client *http.Client, ip string) (bool, error) {
	resp, err := client.Get("http://" + ip)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()
	return true, nil
}

func getResponsiveIP(httpClient *http.Client, r Record, logCh chan<- logEntry) (string, error) {
	var responsiveIP string

	for _, ip := range r.IPs {
		ipResponsive, err := isHTTPResponsive(httpClient, ip)
		if err != nil {
			sendLogEntry(logCh, r.Domain, fmt.Sprintf("%s: Error checking response for IP %s: %v", r.Domain, ip, err))
			continue
		}

		if !ipResponsive {
			continue
		}

		responsiveIP = ip
		break // Exit the loop since we found a responsive IP
	}

	if responsiveIP == "" {
		return responsiveIP, fmt.Errorf("%s: No responsive IPs found", r.Domain)
	}

	return responsiveIP, nil
}
