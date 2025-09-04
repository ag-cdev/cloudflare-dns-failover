package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	cf "github.com/cloudflare/cloudflare-go"
)

// createAPIClient stays the same
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
			return http.ErrUseLastResponse
		},
	}
}

// isResponsive can handle HTTP(S) or raw TCP checks
func isResponsive(client *http.Client, protocol string, ip string, port int) (bool, error) {
	// Wrap IPv6 in brackets for URL or TCP target
	host := ip
	if net.ParseIP(ip) != nil && strings.Contains(ip, ":") {
		host = "[" + ip + "]"
	}

	switch protocol {
	case "http", "https":
		url := fmt.Sprintf("%s://%s:%d", protocol, host, port)
		resp, err := client.Get(url)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()
		return true, nil

	case "tcp":
		target := fmt.Sprintf("%s:%d", host, port)
		conn, err := net.DialTimeout("tcp", target, 3*time.Second)
		if err != nil {
			return false, err
		}
		conn.Close()
		return true, nil

	default:
		return false, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// getResponsiveIP now takes protocol + port
func getResponsiveIP(httpClient *http.Client, r Record, protocol string, port int, logCh chan<- logEntry) (string, error) {
	var responsiveIP string

	for _, ip := range r.IPs {
		ok, err := isResponsive(httpClient, protocol, ip, port)
		if err != nil {
			sendLogEntry(logCh, r.Domain, fmt.Sprintf("%s: Error checking %s://%s:%d: %v", r.Domain, protocol, ip, port, err))
			continue
		}
		if !ok {
			continue
		}
		responsiveIP = ip
		break
	}

	if responsiveIP == "" {
		return "", fmt.Errorf("%s: No responsive IPs found", r.Domain)
	}
	return responsiveIP, nil
}
