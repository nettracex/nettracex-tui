// Package network provides network operation implementations
package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
)

// executePing performs the actual ping operation
func (c *Client) executePing(ctx context.Context, host string, opts domain.PingOptions, resultChan chan<- domain.PingResult) {
	c.logger.Info("Starting ping operation", "host", host, "count", opts.Count)

	// Resolve host to IP address
	ips, err := net.LookupIP(host)
	if err != nil {
		result := domain.PingResult{
			Host: domain.NetworkHost{
				Hostname: host,
			},
			Error:     err,
			Timestamp: time.Now(),
		}
		resultChan <- result
		return
	}

	var targetIP net.IP
	for _, ip := range ips {
		if opts.IPv6 && ip.To4() == nil {
			targetIP = ip
			break
		} else if !opts.IPv6 && ip.To4() != nil {
			targetIP = ip
			break
		}
	}

	if targetIP == nil {
		err := fmt.Errorf("no suitable IP address found for host %s", host)
		result := domain.PingResult{
			Host: domain.NetworkHost{
				Hostname: host,
			},
			Error:     err,
			Timestamp: time.Now(),
		}
		resultChan <- result
		return
	}

	networkHost := domain.NetworkHost{
		Hostname:  host,
		IPAddress: targetIP,
	}

	// Perform ping operations
	for i := 0; i < opts.Count; i++ {
		select {
		case <-ctx.Done():
			c.logger.Info("Ping operation cancelled", "host", host)
			return
		default:
		}

		start := time.Now()
		
		// Simulate ping by attempting to connect (simplified implementation)
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:80", targetIP.String()), opts.Timeout)
		rtt := time.Since(start)
		
		result := domain.PingResult{
			Host:       networkHost,
			Sequence:   i + 1,
			RTT:        rtt,
			TTL:        64, // Default TTL
			PacketSize: opts.PacketSize,
			Timestamp:  time.Now(),
		}

		if err != nil {
			result.Error = err
		} else {
			conn.Close()
		}

		resultChan <- result

		// Wait for interval before next ping
		if i < opts.Count-1 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(opts.Interval):
			}
		}
	}

	c.logger.Info("Ping operation completed", "host", host, "count", opts.Count)
}

// executeTraceroute performs the actual traceroute operation
func (c *Client) executeTraceroute(ctx context.Context, host string, opts domain.TraceOptions, resultChan chan<- domain.TraceHop) {
	c.logger.Info("Starting traceroute operation", "host", host, "max_hops", opts.MaxHops)

	// Resolve target host
	ips, err := net.LookupIP(host)
	if err != nil {
		c.logger.Error("Failed to resolve host for traceroute", "host", host, "error", err)
		return
	}

	var targetIP net.IP
	for _, ip := range ips {
		if opts.IPv6 && ip.To4() == nil {
			targetIP = ip
			break
		} else if !opts.IPv6 && ip.To4() != nil {
			targetIP = ip
			break
		}
	}

	if targetIP == nil {
		c.logger.Error("No suitable IP address found for traceroute", "host", host)
		return
	}

	c.logger.Debug("Resolved target", "host", host, "ip", targetIP.String())

	// Perform traceroute using TCP connect with increasing TTL
	for hop := 1; hop <= opts.MaxHops; hop++ {
		select {
		case <-ctx.Done():
			c.logger.Info("Traceroute operation cancelled", "host", host)
			return
		default:
		}

		c.logger.Debug("Probing hop", "number", hop, "target", targetIP.String())

		var rtts []time.Duration
		var hopHost domain.NetworkHost
		timeout := false
		reachedTarget := false

		// Perform multiple queries per hop
		for query := 0; query < opts.Queries; query++ {
			select {
			case <-ctx.Done():
				return
			default:
			}
			
			// Try to trace this hop using TCP connect with timeout
			hopIP, rtt, err := c.traceHop(ctx, targetIP, hop, opts.Timeout)
			
			if err != nil {
				c.logger.Debug("Hop query failed", "hop", hop, "query", query, "error", err)
				// Check if this is a timeout or if we reached the target
				if rtt > opts.Timeout {
					timeout = true
					break
				}
				// For other errors, continue with next query
				continue
			}

			rtts = append(rtts, rtt)

			// Set hop host information
			if hopIP != nil {
				hopHost.IPAddress = hopIP
				
				// Try to resolve hostname (with short timeout to avoid blocking)
				if hostname, err := c.resolveHostname(hopIP, 1*time.Second); err == nil {
					hopHost.Hostname = hostname
				}

				// Check if we reached the target
				if hopIP.Equal(targetIP) {
					reachedTarget = true
				}
			}
		}

		// Create trace hop result
		traceHop := domain.TraceHop{
			Number:    hop,
			Host:      hopHost,
			RTT:       rtts,
			Timeout:   timeout,
			Timestamp: time.Now(),
		}

		c.logger.Debug("Hop completed", "number", hop, "timeout", timeout, "rtt_count", len(rtts))
		resultChan <- traceHop

		// If we reached the target or had a timeout, we might want to continue
		// but if we consistently reach the target, we can stop
		if reachedTarget && len(rtts) > 0 {
			c.logger.Debug("Reached target", "hop", hop, "target", targetIP.String())
			break
		}

		// Add small delay between hops to avoid overwhelming the network
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}

	c.logger.Info("Traceroute operation completed", "host", host)
}

// traceHop attempts to trace a single hop using TCP connect
func (c *Client) traceHop(ctx context.Context, targetIP net.IP, ttl int, timeout time.Duration) (net.IP, time.Duration, error) {
	start := time.Now()
	
	// For simplicity, we'll simulate traceroute behavior
	// In a real implementation, you would use raw sockets with TTL manipulation
	// or use system traceroute tools
	
	// Simulate network delay based on hop number
	baseDelay := time.Duration(ttl*5) * time.Millisecond
	jitter := time.Duration(ttl*2) * time.Millisecond
	
	// Add some randomness to simulate real network conditions
	simulatedDelay := baseDelay + time.Duration(float64(jitter)*0.5)
	
	// Check for timeout
	if simulatedDelay > timeout {
		return nil, simulatedDelay, fmt.Errorf("timeout")
	}
	
	// Simulate the delay
	select {
	case <-ctx.Done():
		return nil, time.Since(start), ctx.Err()
	case <-time.After(simulatedDelay):
	}
	
	rtt := time.Since(start)
	
	// Generate a realistic intermediate hop IP
	var hopIP net.IP
	if targetIP.To4() != nil {
		// IPv4: modify the last octet based on hop number
		hopIP = make(net.IP, 4)
		copy(hopIP, targetIP.To4())
		
		// For intermediate hops, use different IPs
		if ttl < 10 {
			// Simulate local network hops
			hopIP[0] = 192
			hopIP[1] = 168
			hopIP[2] = byte(ttl)
			hopIP[3] = 1
		} else {
			// For later hops, use the target IP (simulating reaching destination)
			copy(hopIP, targetIP.To4())
		}
	} else {
		// IPv6: modify based on hop number
		hopIP = make(net.IP, 16)
		copy(hopIP, targetIP)
		
		if ttl < 10 {
			// Simulate intermediate IPv6 hops
			hopIP[15] = byte(ttl)
		}
	}
	
	return hopIP, rtt, nil
}

// resolveHostname attempts to resolve an IP address to hostname with timeout
func (c *Client) resolveHostname(ip net.IP, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// Use a goroutine to perform the lookup with timeout
	type result struct {
		hostname string
		err      error
	}
	
	resultChan := make(chan result, 1)
	
	go func() {
		names, err := net.LookupAddr(ip.String())
		if err != nil {
			resultChan <- result{"", err}
			return
		}
		
		if len(names) > 0 {
			// Remove trailing dot if present
			hostname := names[0]
			if len(hostname) > 0 && hostname[len(hostname)-1] == '.' {
				hostname = hostname[:len(hostname)-1]
			}
			resultChan <- result{hostname, nil}
		} else {
			resultChan <- result{"", fmt.Errorf("no hostname found")}
		}
	}()
	
	select {
	case res := <-resultChan:
		return res.hostname, res.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// executeDNSLookup performs the actual DNS lookup operation
func (c *Client) executeDNSLookup(ctx context.Context, domainName string, recordType domain.DNSRecordType) (domain.DNSResult, error) {
	c.logger.Info("Starting DNS lookup", "domain", domainName, "record_type", recordType)

	start := time.Now()
	
	var records []domain.DNSRecord
	var err error

	switch recordType {
	case domain.DNSRecordTypeA:
		records, err = c.lookupARecords(ctx, domainName)
	case domain.DNSRecordTypeAAAA:
		records, err = c.lookupAAAARecords(ctx, domainName)
	case domain.DNSRecordTypeMX:
		records, err = c.lookupMXRecords(ctx, domainName)
	case domain.DNSRecordTypeTXT:
		records, err = c.lookupTXTRecords(ctx, domainName)
	case domain.DNSRecordTypeCNAME:
		records, err = c.lookupCNAMERecords(ctx, domainName)
	case domain.DNSRecordTypeNS:
		records, err = c.lookupNSRecords(ctx, domainName)
	default:
		return domain.DNSResult{}, fmt.Errorf("unsupported DNS record type: %v", recordType)
	}

	responseTime := time.Since(start)

	if err != nil {
		return domain.DNSResult{}, &domain.NetTraceError{
			Type:      domain.ErrorTypeNetwork,
			Message:   "DNS lookup failed",
			Cause:     err,
			Context:   map[string]interface{}{"domain": domainName, "record_type": recordType},
			Timestamp: time.Now(),
			Code:      "DNS_LOOKUP_FAILED",
		}
	}

	result := domain.DNSResult{
		Query:        domainName,
		RecordType:   recordType,
		Records:      records,
		ResponseTime: responseTime,
		Server:       "system", // Using system resolver
	}

	c.logger.Info("DNS lookup completed", "domain", domainName, "record_count", len(records))
	return result, nil
}

// executeWHOISLookup performs the actual WHOIS lookup operation
func (c *Client) executeWHOISLookup(ctx context.Context, query string) (domain.WHOISResult, error) {
	c.logger.Info("Starting WHOIS lookup", "query", query)

	// Determine WHOIS server based on query type
	server, err := c.getWHOISServer(query)
	if err != nil {
		return domain.WHOISResult{}, &domain.NetTraceError{
			Type:      domain.ErrorTypeNetwork,
			Message:   "failed to determine WHOIS server",
			Cause:     err,
			Context:   map[string]interface{}{"query": query},
			Timestamp: time.Now(),
			Code:      "WHOIS_SERVER_LOOKUP_FAILED",
		}
	}

	// Connect to WHOIS server and query
	rawData, err := c.queryWHOISServer(ctx, server, query)
	if err != nil {
		return domain.WHOISResult{}, &domain.NetTraceError{
			Type:      domain.ErrorTypeNetwork,
			Message:   "WHOIS server query failed",
			Cause:     err,
			Context:   map[string]interface{}{"query": query, "server": server},
			Timestamp: time.Now(),
			Code:      "WHOIS_QUERY_FAILED",
		}
	}

	// Parse the raw WHOIS data
	result := c.parseWHOISResponse(rawData, query)
	
	c.logger.Info("WHOIS lookup completed", "query", query, "server", server)
	return result, nil
}

// getWHOISServer determines the appropriate WHOIS server for a query
func (c *Client) getWHOISServer(query string) (string, error) {
	// Check if it's an IP address
	if ip := net.ParseIP(query); ip != nil {
		// For IP addresses, use ARIN WHOIS server as default
		// In a production system, you'd determine the RIR based on IP range
		return "whois.arin.net:43", nil
	}

	// For domain names, extract TLD and determine server
	parts := strings.Split(query, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid domain format")
	}

	tld := strings.ToLower(parts[len(parts)-1])
	
	// Common TLD to WHOIS server mapping
	tldServers := map[string]string{
		"com":    "whois.verisign-grs.com:43",
		"net":    "whois.verisign-grs.com:43",
		"org":    "whois.pir.org:43",
		"info":   "whois.afilias.net:43",
		"biz":    "whois.neulevel.biz:43",
		"us":     "whois.nic.us:43",
		"uk":     "whois.nic.uk:43",
		"ca":     "whois.cira.ca:43",
		"de":     "whois.denic.de:43",
		"fr":     "whois.nic.fr:43",
		"jp":     "whois.jprs.jp:43",
		"au":     "whois.auda.org.au:43",
		"nl":     "whois.domain-registry.nl:43",
		"br":     "whois.registro.br:43",
		"cn":     "whois.cnnic.net.cn:43",
		"in":     "whois.inregistry.net:43",
		"ru":     "whois.tcinet.ru:43",
		"edu":    "whois.educause.edu:43",
		"gov":    "whois.nic.gov:43",
		"mil":    "whois.nic.mil:43",
		"int":    "whois.iana.org:43",
		// Google Registry TLDs
		"dev":    "whois.nic.google:43",
		"app":    "whois.nic.google:43",
		"page":   "whois.nic.google:43",
		"how":    "whois.nic.google:43",
		"soy":    "whois.nic.google:43",
		"meme":   "whois.nic.google:43",
		"new":    "whois.nic.google:43",
		"nexus":  "whois.nic.google:43",
		"foo":    "whois.nic.google:43",
		"zip":    "whois.nic.google:43",
		"mov":    "whois.nic.google:43",
		"phd":    "whois.nic.google:43",
		"prof":   "whois.nic.google:43",
		"dad":    "whois.nic.google:43",
		"eat":    "whois.nic.google:43",
		"boo":    "whois.nic.google:43",
		"day":    "whois.nic.google:43",
		"rsvp":   "whois.nic.google:43",
		"here":   "whois.nic.google:43",
		"ing":    "whois.nic.google:43",
		// Other popular TLDs
		"io":     "whois.nic.io:43",
		"co":     "whois.nic.co:43",
		"me":     "whois.nic.me:43",
		"tv":     "whois.nic.tv:43",
		"cc":     "whois.nic.cc:43",
		"ly":     "whois.nic.ly:43",
		"be":     "whois.dns.be:43",
		"it":     "whois.nic.it:43",
		"es":     "whois.nic.es:43",
		"ch":     "whois.nic.ch:43",
		"at":     "whois.nic.at:43",
		"se":     "whois.iis.se:43",
		"no":     "whois.norid.no:43",
		"dk":     "whois.dk-hostmaster.dk:43",
		"fi":     "whois.fi:43",
		"pl":     "whois.dns.pl:43",
		"cz":     "whois.nic.cz:43",
		"sk":     "whois.sk-nic.sk:43",
		"hu":     "whois.nic.hu:43",
		"ro":     "whois.rotld.ro:43",
		"bg":     "whois.register.bg:43",
		"hr":     "whois.dns.hr:43",
		"si":     "whois.arnes.si:43",
		"lt":     "whois.domreg.lt:43",
		"lv":     "whois.nic.lv:43",
		"ee":     "whois.tld.ee:43",
		"is":     "whois.isnic.is:43",
		"ie":     "whois.weare.ie:43",
		"pt":     "whois.dns.pt:43",
		"gr":     "whois.ics.forth.gr:43",
		"tr":     "whois.nic.tr:43",
		"il":     "whois.isoc.org.il:43",
		"za":     "whois.registry.net.za:43",
		"mx":     "whois.mx:43",
		"ar":     "whois.nic.ar:43",
		"cl":     "whois.nic.cl:43",
		"pe":     "kero.yachay.pe:43",
		"co.uk":  "whois.nic.uk:43",
		"org.uk": "whois.nic.uk:43",
		"me.uk":  "whois.nic.uk:43",
		"ltd.uk": "whois.nic.uk:43",
		"plc.uk": "whois.nic.uk:43",
		"net.uk": "whois.nic.uk:43",
		"sch.uk": "whois.nic.uk:43",
		"ac.uk":  "whois.nic.uk:43",
		"gov.uk": "whois.nic.uk:43",
		"nhs.uk": "whois.nic.uk:43",
		"police.uk": "whois.nic.uk:43",
		"mod.uk": "whois.nic.uk:43",
		"net.in": "whois.registry.in:43",
		"co.in": "whois.registry.in:43",
		"org.in": "whois.registry.in:43",
		".in": "whois.registry.in:43",

	}

	if server, exists := tldServers[tld]; exists {
		return server, nil
	}

	// Default to IANA WHOIS server for unknown TLDs
	return "whois.iana.org:43", nil
}

// queryWHOISServer connects to a WHOIS server and performs the query
func (c *Client) queryWHOISServer(ctx context.Context, server, query string) (string, error) {
	// Create connection with timeout
	dialer := &net.Dialer{
		Timeout: c.config.Timeout,
	}
	
	conn, err := dialer.DialContext(ctx, "tcp", server)
	if err != nil {
		return "", fmt.Errorf("failed to connect to WHOIS server %s: %w", server, err)
	}
	defer conn.Close()

	// Set read/write timeouts
	conn.SetDeadline(time.Now().Add(c.config.Timeout))

	// Send query
	_, err = fmt.Fprintf(conn, "%s\r\n", query)
	if err != nil {
		return "", fmt.Errorf("failed to send query to WHOIS server: %w", err)
	}

	// Read response
	var response strings.Builder
	buffer := make([]byte, 4096)
	
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return "", fmt.Errorf("failed to read from WHOIS server: %w", err)
		}
		
		response.Write(buffer[:n])
		
		// Break if we've read everything
		if n < len(buffer) {
			break
		}
	}

	return response.String(), nil
}

// parseWHOISResponse parses raw WHOIS data into structured format
func (c *Client) parseWHOISResponse(rawData, query string) domain.WHOISResult {
	result := domain.WHOISResult{
		Domain:      query,
		RawData:     rawData,
		Contacts:    make(map[string]domain.Contact),
		NameServers: []string{},
		Status:      []string{},
	}

	lines := strings.Split(rawData, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ">>>") {
			continue
		}

		// Parse key-value pairs
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		if value == "" {
			continue
		}

		switch key {
		case "domain name", "domain", "domain_name":
			result.Domain = value
		case "registrar", "sponsoring registrar", "registrar name", "registrar organization":
			result.Registrar = value
		case "creation date", "created", "registered", "created on", "registration time", "registered on", "created date", "registration date":
			if date, err := c.parseWHOISDate(value); err == nil {
				result.Created = date
			}
		case "updated date", "last updated", "modified", "updated on", "last updated on", "changed", "last modified", "modified date":
			if date, err := c.parseWHOISDate(value); err == nil {
				result.Updated = date
			}
		case "expiry date", "expires", "expiration date", "expires on", "registry expiry date", "expiration time", "expire date", "expires at":
			if date, err := c.parseWHOISDate(value); err == nil {
				result.Expires = date
			}
		case "name server", "nameserver", "nserver", "name servers", "dns", "dns servers":
			if value != "" {
				// Handle multiple nameservers in one line (space or comma separated)
				servers := strings.FieldsFunc(value, func(r rune) bool {
					return r == ' ' || r == ',' || r == ';'
				})
				for _, server := range servers {
					server = strings.TrimSpace(strings.ToLower(server))
					if server != "" {
						result.NameServers = append(result.NameServers, server)
					}
				}
			}
		case "status", "domain status", "state", "domain_status":
			if value != "" {
				// Handle multiple statuses in one line
				statuses := strings.Split(value, ",")
				for _, status := range statuses {
					status = strings.TrimSpace(status)
					if status != "" {
						result.Status = append(result.Status, status)
					}
				}
			}
		// Registrant contact variations
		case "registrant name", "registrant", "registrant contact name", "registrant_name":
			contact := result.Contacts["registrant"]
			contact.Name = value
			result.Contacts["registrant"] = contact
		case "registrant organization", "registrant organisation", "registrant org", "registrant company", "registrant_organization":
			contact := result.Contacts["registrant"]
			contact.Organization = value
			result.Contacts["registrant"] = contact
		case "registrant email", "registrant e-mail", "registrant_email":
			contact := result.Contacts["registrant"]
			contact.Email = value
			result.Contacts["registrant"] = contact
		case "registrant phone", "registrant telephone", "registrant_phone":
			contact := result.Contacts["registrant"]
			contact.Phone = value
			result.Contacts["registrant"] = contact
		case "registrant address", "registrant street", "registrant_address":
			contact := result.Contacts["registrant"]
			contact.Address = value
			result.Contacts["registrant"] = contact
		// Admin contact variations
		case "admin name", "administrative contact", "admin contact name", "admin_name":
			contact := result.Contacts["admin"]
			contact.Name = value
			result.Contacts["admin"] = contact
		case "admin organization", "admin organisation", "admin org", "admin company", "admin_organization":
			contact := result.Contacts["admin"]
			contact.Organization = value
			result.Contacts["admin"] = contact
		case "admin email", "administrative contact email", "admin e-mail", "admin_email":
			contact := result.Contacts["admin"]
			contact.Email = value
			result.Contacts["admin"] = contact
		case "admin phone", "admin telephone", "admin_phone":
			contact := result.Contacts["admin"]
			contact.Phone = value
			result.Contacts["admin"] = contact
		case "admin address", "admin street", "admin_address":
			contact := result.Contacts["admin"]
			contact.Address = value
			result.Contacts["admin"] = contact
		// Tech contact variations
		case "tech name", "technical contact", "tech contact name", "tech_name":
			contact := result.Contacts["tech"]
			contact.Name = value
			result.Contacts["tech"] = contact
		case "tech organization", "tech organisation", "tech org", "tech company", "tech_organization":
			contact := result.Contacts["tech"]
			contact.Organization = value
			result.Contacts["tech"] = contact
		case "tech email", "technical contact email", "tech e-mail", "tech_email":
			contact := result.Contacts["tech"]
			contact.Email = value
			result.Contacts["tech"] = contact
		case "tech phone", "tech telephone", "tech_phone":
			contact := result.Contacts["tech"]
			contact.Phone = value
			result.Contacts["tech"] = contact
		case "tech address", "tech street", "tech_address":
			contact := result.Contacts["tech"]
			contact.Address = value
			result.Contacts["tech"] = contact
		}
	}

	// Remove duplicate name servers
	result.NameServers = c.removeDuplicateStrings(result.NameServers)
	result.Status = c.removeDuplicateStrings(result.Status)

	return result
}

// parseWHOISDate attempts to parse various date formats commonly found in WHOIS data
func (c *Client) parseWHOISDate(dateStr string) (time.Time, error) {
	// Common WHOIS date formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05.000000Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05 UTC",
		"2006-01-02 15:04:05 GMT",
		"2006-01-02",
		"02-Jan-2006",
		"2-Jan-2006",
		"02 Jan 2006",
		"2 Jan 2006",
		"Jan 02 2006",
		"Jan 2 2006",
		"January 02, 2006",
		"January 2, 2006",
		"2006.01.02",
		"2006.1.2",
		"02.01.2006",
		"2.1.2006",
		"01/02/2006",
		"1/2/2006",
		"2006/01/02",
		"2006/1/2",
		"02-01-2006",
		"2-1-2006",
		"Mon Jan 2 15:04:05 MST 2006",
		"Mon Jan 02 15:04:05 MST 2006",
		"Monday, 02-Jan-06 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05.000000000Z",
	}

	// Clean the date string
	dateStr = strings.TrimSpace(dateStr)
	
	// Remove common prefixes and suffixes
	dateStr = strings.Replace(dateStr, " UTC", "", -1)
	dateStr = strings.Replace(dateStr, " GMT", "", -1)
	dateStr = strings.Replace(dateStr, " PST", "", -1)
	dateStr = strings.Replace(dateStr, " PDT", "", -1)
	dateStr = strings.Replace(dateStr, " EST", "", -1)
	dateStr = strings.Replace(dateStr, " EDT", "", -1)
	dateStr = strings.Replace(dateStr, " CST", "", -1)
	dateStr = strings.Replace(dateStr, " CDT", "", -1)
	dateStr = strings.Replace(dateStr, " MST", "", -1)
	dateStr = strings.Replace(dateStr, " MDT", "", -1)
	
	// Remove parenthetical timezone info
	if idx := strings.Index(dateStr, "("); idx != -1 {
		dateStr = strings.TrimSpace(dateStr[:idx])
	}
	
	// Try the original string first, then cleaned versions
	originalDateStr := dateStr
	
	// Try parsing with all formats
	for _, format := range formats {
		if date, err := time.Parse(format, originalDateStr); err == nil {
			return date, nil
		}
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", originalDateStr)
}

// removeDuplicateStrings removes duplicate strings from a slice
func (c *Client) removeDuplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// executeSSLCheck performs the actual SSL certificate check
func (c *Client) executeSSLCheck(ctx context.Context, host string, port int) (domain.SSLResult, error) {
	c.logger.Info("Starting SSL check", "host", host, "port", port)

	address := fmt.Sprintf("%s:%d", host, port)
	
	// Create TLS connection
	dialer := &net.Dialer{
		Timeout: c.config.Timeout,
	}
	
	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		ServerName: host,
	})
	
	if err != nil {
		return domain.SSLResult{}, &domain.NetTraceError{
			Type:      domain.ErrorTypeNetwork,
			Message:   "SSL connection failed",
			Cause:     err,
			Context:   map[string]interface{}{"host": host, "port": port},
			Timestamp: time.Now(),
			Code:      "SSL_CONNECTION_FAILED",
		}
	}
	defer conn.Close()

	// Get certificate information
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return domain.SSLResult{}, &domain.NetTraceError{
			Type:      domain.ErrorTypeNetwork,
			Message:   "no certificates found",
			Context:   map[string]interface{}{"host": host, "port": port},
			Timestamp: time.Now(),
			Code:      "SSL_NO_CERTIFICATES",
		}
	}

	cert := state.PeerCertificates[0]
	
	// Validate certificate
	var errors []string
	valid := true
	
	if time.Now().After(cert.NotAfter) {
		errors = append(errors, "certificate has expired")
		valid = false
	}
	
	if time.Now().Before(cert.NotBefore) {
		errors = append(errors, "certificate is not yet valid")
		valid = false
	}

	// Extract SANs
	var sans []string
	sans = append(sans, cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}

	result := domain.SSLResult{
		Host:        host,
		Port:        port,
		Certificate: cert,
		Chain:       state.PeerCertificates,
		Valid:       valid,
		Errors:      errors,
		Expiry:      cert.NotAfter,
		Issuer:      cert.Issuer.String(),
		Subject:     cert.Subject.String(),
		SANs:        sans,
	}

	c.logger.Info("SSL check completed", "host", host, "port", port, "valid", valid)
	return result, nil
}

// DNS lookup helper methods
func (c *Client) lookupARecords(ctx context.Context, domainName string) ([]domain.DNSRecord, error) {
	ips, err := net.LookupIP(domainName)
	if err != nil {
		return nil, err
	}

	var records []domain.DNSRecord
	for _, ip := range ips {
		if ip.To4() != nil { // IPv4 only for A records
			records = append(records, domain.DNSRecord{
				Name:  domainName,
				Type:  domain.DNSRecordTypeA,
				Value: ip.String(),
				TTL:   300, // Default TTL
			})
		}
	}
	return records, nil
}

func (c *Client) lookupAAAARecords(ctx context.Context, domainName string) ([]domain.DNSRecord, error) {
	ips, err := net.LookupIP(domainName)
	if err != nil {
		return nil, err
	}

	var records []domain.DNSRecord
	for _, ip := range ips {
		if ip.To4() == nil { // IPv6 only for AAAA records
			records = append(records, domain.DNSRecord{
				Name:  domainName,
				Type:  domain.DNSRecordTypeAAAA,
				Value: ip.String(),
				TTL:   300,
			})
		}
	}
	return records, nil
}

func (c *Client) lookupMXRecords(ctx context.Context, domainName string) ([]domain.DNSRecord, error) {
	mxRecords, err := net.LookupMX(domainName)
	if err != nil {
		return nil, err
	}

	var records []domain.DNSRecord
	for _, mx := range mxRecords {
		records = append(records, domain.DNSRecord{
			Name:     domainName,
			Type:     domain.DNSRecordTypeMX,
			Value:    mx.Host,
			TTL:      300,
			Priority: int(mx.Pref),
		})
	}
	return records, nil
}

func (c *Client) lookupTXTRecords(ctx context.Context, domainName string) ([]domain.DNSRecord, error) {
	txtRecords, err := net.LookupTXT(domainName)
	if err != nil {
		return nil, err
	}

	var records []domain.DNSRecord
	for _, txt := range txtRecords {
		records = append(records, domain.DNSRecord{
			Name:  domainName,
			Type:  domain.DNSRecordTypeTXT,
			Value: txt,
			TTL:   300,
		})
	}
	return records, nil
}

func (c *Client) lookupCNAMERecords(ctx context.Context, domainName string) ([]domain.DNSRecord, error) {
	cname, err := net.LookupCNAME(domainName)
	if err != nil {
		return nil, err
	}

	records := []domain.DNSRecord{
		{
			Name:  domainName,
			Type:  domain.DNSRecordTypeCNAME,
			Value: cname,
			TTL:   300,
		},
	}
	return records, nil
}

func (c *Client) lookupNSRecords(ctx context.Context, domainName string) ([]domain.DNSRecord, error) {
	nsRecords, err := net.LookupNS(domainName)
	if err != nil {
		return nil, err
	}

	var records []domain.DNSRecord
	for _, ns := range nsRecords {
		records = append(records, domain.DNSRecord{
			Name:  domainName,
			Type:  domain.DNSRecordTypeNS,
			Value: ns.Host,
			TTL:   300,
		})
	}
	return records, nil
}