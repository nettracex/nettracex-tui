// Package network provides mock implementations for testing
package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
)

// MockClient implements the NetworkClient interface for testing
type MockClient struct {
	mu sync.RWMutex

	// Configuration for mock behavior
	pingResponses      map[string][]domain.PingResult
	traceResponses     map[string][]domain.TraceHop
	dnsResponses       map[string]domain.DNSResult
	whoisResponses     map[string]domain.WHOISResult
	sslResponses       map[string]domain.SSLResult
	
	// Error simulation
	pingErrors         map[string]error
	traceErrors        map[string]error
	dnsErrors          map[string]error
	whoisErrors        map[string]error
	sslErrors          map[string]error
	
	// Delay simulation
	pingDelays         map[string]time.Duration
	traceDelays        map[string]time.Duration
	dnsDelays          map[string]time.Duration
	whoisDelays        map[string]time.Duration
	sslDelays          map[string]time.Duration
	
	// Call tracking
	pingCalls          []MockCall
	traceCalls         []MockCall
	dnsCalls           []MockCall
	whoisCalls         []MockCall
	sslCalls           []MockCall
	
	// Behavior flags
	simulateTimeout    bool
	simulateNetworkError bool
	callCount          int
}

// MockCall represents a recorded method call
type MockCall struct {
	Method    string
	Args      []interface{}
	Timestamp time.Time
}

// NewMockClient creates a new mock network client
func NewMockClient() *MockClient {
	return &MockClient{
		pingResponses:  make(map[string][]domain.PingResult),
		traceResponses: make(map[string][]domain.TraceHop),
		dnsResponses:   make(map[string]domain.DNSResult),
		whoisResponses: make(map[string]domain.WHOISResult),
		sslResponses:   make(map[string]domain.SSLResult),
		pingErrors:     make(map[string]error),
		traceErrors:    make(map[string]error),
		dnsErrors:      make(map[string]error),
		whoisErrors:    make(map[string]error),
		sslErrors:      make(map[string]error),
		pingDelays:     make(map[string]time.Duration),
		traceDelays:    make(map[string]time.Duration),
		dnsDelays:      make(map[string]time.Duration),
		whoisDelays:    make(map[string]time.Duration),
		sslDelays:      make(map[string]time.Duration),
	}
}

// Ping implements the NetworkClient interface with mock behavior
func (m *MockClient) Ping(ctx context.Context, host string, opts domain.PingOptions) (<-chan domain.PingResult, error) {
	m.mu.Lock()
	m.callCount++
	call := MockCall{
		Method:    "Ping",
		Args:      []interface{}{host, opts},
		Timestamp: time.Now(),
	}
	m.pingCalls = append(m.pingCalls, call)
	m.mu.Unlock()

	// Check for configured error
	if err, exists := m.pingErrors[host]; exists {
		return nil, err
	}

	// Simulate delay if configured
	if delay, exists := m.pingDelays[host]; exists {
		time.Sleep(delay)
	}

	resultChan := make(chan domain.PingResult, opts.Count)
	
	go func() {
		defer close(resultChan)
		
		// Use configured responses or generate default ones
		responses, exists := m.pingResponses[host]
		if !exists {
			responses = m.generateDefaultPingResults(host, opts)
		}
		
		for i, result := range responses {
			if i >= opts.Count {
				break
			}
			
			select {
			case <-ctx.Done():
				return
			case resultChan <- result:
				// Simulate interval between pings
				if i < len(responses)-1 {
					time.Sleep(opts.Interval)
				}
			}
		}
	}()

	return resultChan, nil
}

// Traceroute implements the NetworkClient interface with mock behavior
func (m *MockClient) Traceroute(ctx context.Context, host string, opts domain.TraceOptions) (<-chan domain.TraceHop, error) {
	m.mu.Lock()
	m.callCount++
	call := MockCall{
		Method:    "Traceroute",
		Args:      []interface{}{host, opts},
		Timestamp: time.Now(),
	}
	m.traceCalls = append(m.traceCalls, call)
	m.mu.Unlock()

	// Check for configured error
	if err, exists := m.traceErrors[host]; exists {
		return nil, err
	}

	// Simulate delay if configured
	if delay, exists := m.traceDelays[host]; exists {
		time.Sleep(delay)
	}

	resultChan := make(chan domain.TraceHop, opts.MaxHops)
	
	go func() {
		defer close(resultChan)
		
		// Use configured responses or generate default ones
		responses, exists := m.traceResponses[host]
		if !exists {
			responses = m.generateDefaultTraceResults(host, opts)
		}
		
		for i, hop := range responses {
			if i >= opts.MaxHops {
				break
			}
			
			select {
			case <-ctx.Done():
				return
			case resultChan <- hop:
				// Simulate delay between hops
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	return resultChan, nil
}

// DNSLookup implements the NetworkClient interface with mock behavior
func (m *MockClient) DNSLookup(ctx context.Context, domainName string, recordType domain.DNSRecordType) (domain.DNSResult, error) {
	m.mu.Lock()
	m.callCount++
	call := MockCall{
		Method:    "DNSLookup",
		Args:      []interface{}{domainName, recordType},
		Timestamp: time.Now(),
	}
	m.dnsCalls = append(m.dnsCalls, call)
	m.mu.Unlock()

	key := fmt.Sprintf("%s:%d", domainName, recordType)
	
	// Check for configured error
	if err, exists := m.dnsErrors[key]; exists {
		return domain.DNSResult{}, err
	}

	// Simulate delay if configured
	if delay, exists := m.dnsDelays[key]; exists {
		time.Sleep(delay)
	}

	// Use configured response or generate default one
	if result, exists := m.dnsResponses[key]; exists {
		return result, nil
	}

	return m.generateDefaultDNSResult(domainName, recordType), nil
}

// WHOISLookup implements the NetworkClient interface with mock behavior
func (m *MockClient) WHOISLookup(ctx context.Context, query string) (domain.WHOISResult, error) {
	m.mu.Lock()
	m.callCount++
	call := MockCall{
		Method:    "WHOISLookup",
		Args:      []interface{}{query},
		Timestamp: time.Now(),
	}
	m.whoisCalls = append(m.whoisCalls, call)
	m.mu.Unlock()

	// Check for configured error
	if err, exists := m.whoisErrors[query]; exists {
		return domain.WHOISResult{}, err
	}

	// Simulate delay if configured
	if delay, exists := m.whoisDelays[query]; exists {
		time.Sleep(delay)
	}

	// Use configured response or generate default one
	if result, exists := m.whoisResponses[query]; exists {
		return result, nil
	}

	return m.generateDefaultWHOISResult(query), nil
}

// SSLCheck implements the NetworkClient interface with mock behavior
func (m *MockClient) SSLCheck(ctx context.Context, host string, port int) (domain.SSLResult, error) {
	m.mu.Lock()
	m.callCount++
	call := MockCall{
		Method:    "SSLCheck",
		Args:      []interface{}{host, port},
		Timestamp: time.Now(),
	}
	m.sslCalls = append(m.sslCalls, call)
	m.mu.Unlock()

	key := fmt.Sprintf("%s:%d", host, port)
	
	// Check for configured error
	if err, exists := m.sslErrors[key]; exists {
		return domain.SSLResult{}, err
	}

	// Simulate delay if configured
	if delay, exists := m.sslDelays[key]; exists {
		time.Sleep(delay)
	}

	// Use configured response or generate default one
	if result, exists := m.sslResponses[key]; exists {
		return result, nil
	}

	return m.generateDefaultSSLResult(host, port), nil
}

// Configuration methods for setting up mock behavior

// SetPingResponse configures a mock ping response for a specific host
func (m *MockClient) SetPingResponse(host string, results []domain.PingResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingResponses[host] = results
}

// SetPingError configures a mock ping error for a specific host
func (m *MockClient) SetPingError(host string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingErrors[host] = err
}

// SetPingDelay configures a mock ping delay for a specific host
func (m *MockClient) SetPingDelay(host string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingDelays[host] = delay
}

// SetTraceResponse configures a mock traceroute response for a specific host
func (m *MockClient) SetTraceResponse(host string, hops []domain.TraceHop) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.traceResponses[host] = hops
}

// SetTraceError configures a mock traceroute error for a specific host
func (m *MockClient) SetTraceError(host string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.traceErrors[host] = err
}

// SetDNSResponse configures a mock DNS response for a specific domain and record type
func (m *MockClient) SetDNSResponse(domainName string, recordType domain.DNSRecordType, result domain.DNSResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%d", domainName, recordType)
	m.dnsResponses[key] = result
}

// SetDNSError configures a mock DNS error for a specific domain and record type
func (m *MockClient) SetDNSError(domainName string, recordType domain.DNSRecordType, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%d", domainName, recordType)
	m.dnsErrors[key] = err
}

// SetWHOISResponse configures a mock WHOIS response for a specific query
func (m *MockClient) SetWHOISResponse(query string, result domain.WHOISResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.whoisResponses[query] = result
}

// SetWHOISError configures a mock WHOIS error for a specific query
func (m *MockClient) SetWHOISError(query string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.whoisErrors[query] = err
}

// SetSSLResponse configures a mock SSL response for a specific host and port
func (m *MockClient) SetSSLResponse(host string, port int, result domain.SSLResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%d", host, port)
	m.sslResponses[key] = result
}

// SetSSLError configures a mock SSL error for a specific host and port
func (m *MockClient) SetSSLError(host string, port int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%d", host, port)
	m.sslErrors[key] = err
}

// Inspection methods for testing

// GetCallCount returns the total number of method calls made
func (m *MockClient) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetPingCalls returns all recorded ping calls
func (m *MockClient) GetPingCalls() []MockCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]MockCall(nil), m.pingCalls...)
}

// GetTraceCalls returns all recorded traceroute calls
func (m *MockClient) GetTraceCalls() []MockCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]MockCall(nil), m.traceCalls...)
}

// GetDNSCalls returns all recorded DNS calls
func (m *MockClient) GetDNSCalls() []MockCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]MockCall(nil), m.dnsCalls...)
}

// GetWHOISCalls returns all recorded WHOIS calls
func (m *MockClient) GetWHOISCalls() []MockCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]MockCall(nil), m.whoisCalls...)
}

// GetSSLCalls returns all recorded SSL calls
func (m *MockClient) GetSSLCalls() []MockCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]MockCall(nil), m.sslCalls...)
}

// Reset clears all recorded calls and configured responses
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.pingResponses = make(map[string][]domain.PingResult)
	m.traceResponses = make(map[string][]domain.TraceHop)
	m.dnsResponses = make(map[string]domain.DNSResult)
	m.whoisResponses = make(map[string]domain.WHOISResult)
	m.sslResponses = make(map[string]domain.SSLResult)
	
	m.pingErrors = make(map[string]error)
	m.traceErrors = make(map[string]error)
	m.dnsErrors = make(map[string]error)
	m.whoisErrors = make(map[string]error)
	m.sslErrors = make(map[string]error)
	
	m.pingDelays = make(map[string]time.Duration)
	m.traceDelays = make(map[string]time.Duration)
	m.dnsDelays = make(map[string]time.Duration)
	m.whoisDelays = make(map[string]time.Duration)
	m.sslDelays = make(map[string]time.Duration)
	
	m.pingCalls = nil
	m.traceCalls = nil
	m.dnsCalls = nil
	m.whoisCalls = nil
	m.sslCalls = nil
	
	m.callCount = 0
}

// Default result generation methods

// generateDefaultPingResults generates realistic ping results for testing
func (m *MockClient) generateDefaultPingResults(host string, opts domain.PingOptions) []domain.PingResult {
	var results []domain.PingResult
	
	// Parse or generate IP address
	ip := net.ParseIP(host)
	if ip == nil {
		ip = net.IPv4(192, 168, 1, 1) // Default test IP
	}
	
	networkHost := domain.NetworkHost{
		Hostname:  host,
		IPAddress: ip,
	}
	
	for i := 0; i < opts.Count; i++ {
		// Simulate realistic RTT with some variation
		baseRTT := 20 * time.Millisecond
		variation := time.Duration(i*2) * time.Millisecond
		rtt := baseRTT + variation
		
		result := domain.PingResult{
			Host:       networkHost,
			Sequence:   i + 1,
			RTT:        rtt,
			TTL:        64,
			PacketSize: opts.PacketSize,
			Timestamp:  time.Now(),
		}
		
		// Simulate occasional packet loss (5% chance)
		if m.simulateNetworkError && i%20 == 0 {
			result.Error = fmt.Errorf("request timeout")
		}
		
		results = append(results, result)
	}
	
	return results
}

// generateDefaultTraceResults generates realistic traceroute results for testing
func (m *MockClient) generateDefaultTraceResults(host string, opts domain.TraceOptions) []domain.TraceHop {
	var hops []domain.TraceHop
	
	// Generate a realistic number of hops (typically 8-15)
	numHops := 10
	if opts.MaxHops < numHops {
		numHops = opts.MaxHops
	}
	
	for i := 1; i <= numHops; i++ {
		// Generate hop IP address
		hopIP := net.IPv4(10, byte(i), 1, 1)
		
		// Generate RTTs for multiple queries
		var rtts []time.Duration
		for j := 0; j < opts.Queries; j++ {
			baseRTT := time.Duration(i*5) * time.Millisecond
			variation := time.Duration(j*2) * time.Millisecond
			rtts = append(rtts, baseRTT+variation)
		}
		
		hop := domain.TraceHop{
			Number: i,
			Host: domain.NetworkHost{
				Hostname:  fmt.Sprintf("hop-%d.example.com", i),
				IPAddress: hopIP,
			},
			RTT:       rtts,
			Timeout:   false,
			Timestamp: time.Now(),
		}
		
		// Simulate occasional timeout (10% chance)
		if m.simulateTimeout && i%10 == 0 {
			hop.Timeout = true
			hop.RTT = nil
		}
		
		hops = append(hops, hop)
	}
	
	return hops
}

// generateDefaultDNSResult generates realistic DNS results for testing
func (m *MockClient) generateDefaultDNSResult(domainName string, recordType domain.DNSRecordType) domain.DNSResult {
	var records []domain.DNSRecord
	
	switch recordType {
	case domain.DNSRecordTypeA:
		records = []domain.DNSRecord{
			{
				Name:  domainName,
				Type:  domain.DNSRecordTypeA,
				Value: "192.168.1.100",
				TTL:   300,
			},
			{
				Name:  domainName,
				Type:  domain.DNSRecordTypeA,
				Value: "192.168.1.101",
				TTL:   300,
			},
		}
	case domain.DNSRecordTypeAAAA:
		records = []domain.DNSRecord{
			{
				Name:  domainName,
				Type:  domain.DNSRecordTypeAAAA,
				Value: "2001:db8::1",
				TTL:   300,
			},
		}
	case domain.DNSRecordTypeMX:
		records = []domain.DNSRecord{
			{
				Name:     domainName,
				Type:     domain.DNSRecordTypeMX,
				Value:    "mail.example.com",
				TTL:      300,
				Priority: 10,
			},
			{
				Name:     domainName,
				Type:     domain.DNSRecordTypeMX,
				Value:    "mail2.example.com",
				TTL:      300,
				Priority: 20,
			},
		}
	case domain.DNSRecordTypeTXT:
		records = []domain.DNSRecord{
			{
				Name:  domainName,
				Type:  domain.DNSRecordTypeTXT,
				Value: "v=spf1 include:_spf.example.com ~all",
				TTL:   300,
			},
		}
	case domain.DNSRecordTypeCNAME:
		records = []domain.DNSRecord{
			{
				Name:  domainName,
				Type:  domain.DNSRecordTypeCNAME,
				Value: "canonical.example.com",
				TTL:   300,
			},
		}
	case domain.DNSRecordTypeNS:
		records = []domain.DNSRecord{
			{
				Name:  domainName,
				Type:  domain.DNSRecordTypeNS,
				Value: "ns1.example.com",
				TTL:   86400,
			},
			{
				Name:  domainName,
				Type:  domain.DNSRecordTypeNS,
				Value: "ns2.example.com",
				TTL:   86400,
			},
		}
	}
	
	return domain.DNSResult{
		Query:        domainName,
		RecordType:   recordType,
		Records:      records,
		ResponseTime: 50 * time.Millisecond,
		Server:       "mock-dns-server",
	}
}

// generateDefaultWHOISResult generates realistic WHOIS results for testing
func (m *MockClient) generateDefaultWHOISResult(query string) domain.WHOISResult {
	return domain.WHOISResult{
		Domain:      query,
		Registrar:   "Mock Registrar Inc.",
		Created:     time.Now().AddDate(-2, 0, 0),
		Updated:     time.Now().AddDate(0, -3, 0),
		Expires:     time.Now().AddDate(1, 0, 0),
		NameServers: []string{"ns1.mockregistrar.com", "ns2.mockregistrar.com"},
		Contacts: map[string]domain.Contact{
			"registrant": {
				Name:         "John Doe",
				Organization: "Example Corporation",
				Email:        "admin@example.com",
				Phone:        "+1.5551234567",
				Address:      "123 Main St, Anytown, ST 12345, US",
			},
			"admin": {
				Name:         "Jane Smith",
				Organization: "Example Corporation",
				Email:        "admin@example.com",
				Phone:        "+1.5551234567",
				Address:      "123 Main St, Anytown, ST 12345, US",
			},
		},
		Status:  []string{"clientTransferProhibited", "clientUpdateProhibited"},
		RawData: fmt.Sprintf("Domain Name: %s\nRegistrar: Mock Registrar Inc.\nCreation Date: %s\n", query, time.Now().AddDate(-2, 0, 0).Format("2006-01-02")),
	}
}

// generateDefaultSSLResult generates realistic SSL results for testing
func (m *MockClient) generateDefaultSSLResult(host string, port int) domain.SSLResult {
	// Create a mock certificate (in real implementation, this would be a real x509.Certificate)
	expiry := time.Now().AddDate(0, 6, 0) // Expires in 6 months
	
	return domain.SSLResult{
		Host:        host,
		Port:        port,
		Certificate: nil, // Would be a real certificate in production
		Chain:       nil, // Would be certificate chain in production
		Valid:       true,
		Errors:      []string{},
		Expiry:      expiry,
		Issuer:      "CN=Mock CA,O=Mock Certificate Authority,C=US",
		Subject:     fmt.Sprintf("CN=%s,O=Mock Organization,C=US", host),
		SANs:        []string{host, fmt.Sprintf("www.%s", host)},
	}
}

// Behavior configuration methods

// SetSimulateTimeout enables or disables timeout simulation
func (m *MockClient) SetSimulateTimeout(simulate bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateTimeout = simulate
}

// SetSimulateNetworkError enables or disables network error simulation
func (m *MockClient) SetSimulateNetworkError(simulate bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateNetworkError = simulate
}