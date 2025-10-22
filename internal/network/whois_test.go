// Package network provides tests for WHOIS functionality
package network

import (
	"testing"
)

// TestGetWHOISServer tests WHOIS server selection for various TLDs
func TestGetWHOISServer(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "dev domain",
			query:    "example.dev",
			expected: "whois.nic.google:43",
		},
		{
			name:     "com domain",
			query:    "example.com",
			expected: "whois.verisign-grs.com:43",
		},
		{
			name:     "org domain",
			query:    "example.org",
			expected: "whois.pir.org:43",
		},
		{
			name:     "io domain",
			query:    "example.io",
			expected: "whois.nic.io:43",
		},
		{
			name:     "app domain (Google Registry)",
			query:    "example.app",
			expected: "whois.nic.google:43",
		},
		{
			name:     "page domain (Google Registry)",
			query:    "example.page",
			expected: "whois.nic.google:43",
		},
		{
			name:     "unknown TLD defaults to IANA",
			query:    "example.unknowntld",
			expected: "whois.iana.org:43",
		},
		{
			name:     "subdomain uses parent TLD",
			query:    "sub.example.dev",
			expected: "whois.nic.google:43",
		},
		{
			name:     "co.uk domain",
			query:    "example.co.uk",
			expected: "whois.nic.uk:43",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := client.getWHOISServer(tt.query)
			if err != nil {
				t.Errorf("getWHOISServer() error = %v", err)
				return
			}
			if server != tt.expected {
				t.Errorf("getWHOISServer() = %v, want %v", server, tt.expected)
			}
		})
	}
}

// TestParseWHOISDate tests date parsing for various formats
func TestParseWHOISDate(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name     string
		dateStr  string
		wantErr  bool
	}{
		{
			name:    "ISO 8601 format",
			dateStr: "2023-01-15T10:30:00Z",
			wantErr: false,
		},
		{
			name:    "Simple date format",
			dateStr: "2023-01-15",
			wantErr: false,
		},
		{
			name:    "Date with time",
			dateStr: "2023-01-15 10:30:00",
			wantErr: false,
		},
		{
			name:    "Date with timezone",
			dateStr: "2023-01-15 10:30:00 UTC",
			wantErr: false,
		},
		{
			name:    "Month name format",
			dateStr: "15-Jan-2023",
			wantErr: false,
		},
		{
			name:    "US format",
			dateStr: "01/15/2023",
			wantErr: false,
		},
		{
			name:    "European format",
			dateStr: "15.01.2023",
			wantErr: false,
		},
		{
			name:    "Invalid date",
			dateStr: "not-a-date",
			wantErr: true,
		},
		{
			name:    "Empty string",
			dateStr: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.parseWHOISDate(tt.dateStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWHOISDate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRemoveDuplicateStrings tests duplicate removal functionality
func TestRemoveDuplicateStrings(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "all same",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.removeDuplicateStrings(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("removeDuplicateStrings() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			
			// Check if all expected items are present (order might differ)
			expectedMap := make(map[string]bool)
			for _, item := range tt.expected {
				expectedMap[item] = true
			}
			
			for _, item := range result {
				if !expectedMap[item] {
					t.Errorf("removeDuplicateStrings() contains unexpected item: %v", item)
				}
			}
		})
	}
}