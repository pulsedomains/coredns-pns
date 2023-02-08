// Package pns implements a plugin that returns information held in the PulseChain Name Service.
package pns

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru"
	"github.com/labstack/gommon/log"
	pns "github.com/pulsedomains/go-pns/v3"

	"github.com/miekg/dns"
)

var emptyContentHash = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

// PNS is a plugin that returns information held in the PulseChain Name Service.
type PNS struct {
	Next               plugin.Handler
	Client             *ethclient.Client
	Registry           *pns.Registry
	PlsLinkNameServers []string
	IPFSGatewayAs      []string
	IPFSGatewayAAAAs   []string
}

// IsAuthoritative checks if the PNS plugin is authoritative for a given domain
func (e PNS) IsAuthoritative(domain string) bool {
	controllerAddress, err := e.Registry.Owner(strings.TrimSuffix(domain, "."))
	if err != nil {
		return false
	}

	return controllerAddress != pns.UnknownAddress
}

// HasRecords checks if there are any records for a specific domain and name.
// This is used for wildcard eligibility
func (e PNS) HasRecords(domain string, name string) (bool, error) {
	// See if this has a contenthash record.
	resolver, err := e.getResolver(domain)
	if err != nil {
		return false, err
	}
	bytes, err := resolver.Contenthash()
	if err == nil && len(bytes) > 0 {
		return true, err
	}

	// See if this has DNS records.
	dnsResolver, err := e.getDNSResolver(strings.TrimSuffix(domain, "."))
	if err != nil {
		return false, err
	}
	return dnsResolver.HasRecords(name)
}

// Query queries a given domain/name/resource combination
func (e PNS) Query(domain string, name string, qtype uint16, do bool) ([]dns.RR, error) {
	log.Debugf("request type %d for name %s in domain %v", qtype, name, domain)

	results := make([]dns.RR, 0)

	// If the requested domain has a content hash we alter a number of the records returned
	var contentHash []byte
	hasContentHash := false
	var err error
	if qtype == dns.TypeSOA ||
		qtype == dns.TypeNS ||
		qtype == dns.TypeTXT ||
		qtype == dns.TypeA ||
		qtype == dns.TypeAAAA {
		contentHash, err = e.obtainContentHash(name, domain)
		hasContentHash = err == nil && bytes.Compare(contentHash, emptyContentHash) > 0
	}
	if hasContentHash {
		switch qtype {
		case dns.TypeSOA:
			results, err = e.handleSOA(name, domain, contentHash)
		case dns.TypeNS:
			results, err = e.handleNS(name, domain, contentHash)
		case dns.TypeTXT:
			results, err = e.handleTXT(name, domain, contentHash)
		case dns.TypeA:
			results, err = e.handleA(name, domain, contentHash)
		case dns.TypeAAAA:
			results, err = e.handleAAAA(name, domain, contentHash)
		}
	} else {
		plsDomain := strings.TrimSuffix(domain, ".")
		resolver, err := e.getDNSResolver(plsDomain)
		if err != nil {
			return results, nil
		}

		data, err := resolver.Record(name, qtype)
		if err != nil {
			return results, err
		}

		offset := 0
		for offset < len(data) {
			var result dns.RR
			result, offset, err = dns.UnpackRR(data, offset)
			if err == nil {
				results = append(results, result)
			}
		}
	}

	return results, nil
}

func (e PNS) handleSOA(name string, domain string, contentHash []byte) ([]dns.RR, error) {
	results := make([]dns.RR, 0)
	if len(e.PlsLinkNameServers) > 0 {
		// Create a synthetic SOA record
		now := time.Now()
		ser := ((now.Hour()*3600 + now.Minute()) * 100) / 86400
		dateStr := fmt.Sprintf("%04d%02d%02d%02d", now.Year(), now.Month(), now.Day(), ser)
		result, err := dns.NewRR(fmt.Sprintf("%s 10800 IN SOA %s hostmaster.%s %s 3600 600 1209600 300", e.PlsLinkNameServers[0], name, name, dateStr))
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (e PNS) handleNS(name string, domain string, contentHash []byte) ([]dns.RR, error) {
	results := make([]dns.RR, 0)
	for _, nameserver := range e.PlsLinkNameServers {
		result, err := dns.NewRR(fmt.Sprintf("%s 3600 IN NS %s", domain, nameserver))
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}

func (e PNS) handleTXT(name string, domain string, contentHash []byte) ([]dns.RR, error) {
	results := make([]dns.RR, 0)
	txtRRSet, err := e.obtainTXTRRSet(name, domain)
	if err == nil && len(txtRRSet) != 0 {
		// We have a TXT rrset; use it
		offset := 0
		for offset < len(txtRRSet) {
			var result dns.RR
			result, offset, err = dns.UnpackRR(txtRRSet, offset)
			if err == nil {
				results = append(results, result)
			}
		}
	}

	if isRealOnChainDomain(name, domain) {
		plsDomain := strings.TrimSuffix(domain, ".")
		resolver, err := e.getResolver(plsDomain)
		if err != nil {
			log.Warnf("error obtaining resolver for %s: %v", plsDomain, err)
			return results, nil
		}

		address, err := resolver.Address()
		if err != nil {
			if err.Error() != "abi: unmarshalling empty output" {
				return results, err
			}
			return results, nil
		}

		if address != pns.UnknownAddress {
			result, err := dns.NewRR(fmt.Sprintf("%s 3600 IN TXT \"a=%s\"", name, address.Hex()))
			if err != nil {
				return results, err
			}
			results = append(results, result)
		}

		result, err := dns.NewRR(fmt.Sprintf("%s 3600 IN TXT \"contenthash=0x%x\"", name, contentHash))
		if err != nil {
			return results, err
		}
		results = append(results, result)

		// Also provide dnslink for compatibility with older IPFS gateways
		contentHashStr, err := pns.ContenthashToString(contentHash)
		if err != nil {
			return results, err
		}
		result, err = dns.NewRR(fmt.Sprintf("%s 3600 IN TXT \"dnslink=%s\"", name, contentHashStr))
		if err != nil {
			return results, nil
		}
		results = append(results, result)
	} else if isRealOnChainDomain(strings.TrimPrefix(name, "_dnslink."), domain) {
		// This is a request to _dnslink.<domain>, return the DNS link record.
		contentHashStr, err := pns.ContenthashToString(contentHash)
		if err != nil {
			return results, err
		}
		result, err := dns.NewRR(fmt.Sprintf("%s 3600 IN TXT \"dnslink=%s\"", name, contentHashStr))
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}

func (e PNS) handleA(name string, domain string, contentHash []byte) ([]dns.RR, error) {
	results := make([]dns.RR, 0)

	aRRSet, err := e.obtainARRSet(name, domain)
	if err == nil && len(aRRSet) != 0 {
		// We have an A rrset; use it
		offset := 0
		for offset < len(aRRSet) {
			var result dns.RR
			result, offset, err = dns.UnpackRR(aRRSet, offset)
			if err == nil {
				results = append(results, result)
			}
		}
	} else {
		// We have a content hash but no A record; use the default
		for i := range e.IPFSGatewayAs {
			result, err := dns.NewRR(fmt.Sprintf("%s 3600 IN A %s", name, e.IPFSGatewayAs[i]))
			if err != nil {
				return results, err
			}
			results = append(results, result)
		}
	}

	return results, nil
}

func (e PNS) handleAAAA(name string, domain string, contentHash []byte) ([]dns.RR, error) {
	results := make([]dns.RR, 0)

	aaaaRRSet, err := e.obtainAAAARRSet(name, domain)
	if err == nil && len(aaaaRRSet) != 0 {
		// We have an AAAA rrset; use it
		offset := 0
		for offset < len(aaaaRRSet) {
			var result dns.RR
			result, offset, err = dns.UnpackRR(aaaaRRSet, offset)
			if err == nil {
				results = append(results, result)
			}
		}
	} else {
		// We have a content hash but no AAAA record; use the default
		for i := range e.IPFSGatewayAAAAs {
			result, err := dns.NewRR(fmt.Sprintf("%s 3600 IN AAAA %s", name, e.IPFSGatewayAAAAs[i]))
			if err != nil {
				log.Warnf("error creating %s AAAA RR: %v", name, err)
			}
			results = append(results, result)
		}
	}
	return results, nil
}

// ServeDNS implements the plugin.Handler interface.
func (e PNS) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	a := new(dns.Msg)
	a.SetReply(r)
	a.Compress = true
	a.Authoritative = true
	var result Result
	a.Answer, a.Ns, a.Extra, result = Lookup(e, state)
	switch result {
	case Success:
		state.SizeAndDo(a)
		w.WriteMsg(a)
		return dns.RcodeSuccess, nil
	case NoData:
		if e.Next == nil {
			state.SizeAndDo(a)
			w.WriteMsg(a)
			return dns.RcodeSuccess, nil
		}
		return plugin.NextOrFailure(e.Name(), e.Next, ctx, w, r)
	case NameError:
		a.Rcode = dns.RcodeNameError
	case ServerFailure:
		return dns.RcodeServerFailure, nil
	}
	// Unknown result...
	return dns.RcodeServerFailure, nil

}

func (e PNS) obtainARRSet(name string, domain string) ([]byte, error) {
	plsDomain := strings.TrimSuffix(domain, ".")
	resolver, err := e.getDNSResolver(plsDomain)
	if err != nil {
		return []byte{}, nil
	}

	return resolver.Record(name, dns.TypeA)
}

func (e PNS) obtainAAAARRSet(name string, domain string) ([]byte, error) {
	plsDomain := strings.TrimSuffix(domain, ".")
	resolver, err := e.getDNSResolver(plsDomain)
	if err != nil {
		return []byte{}, nil
	}

	return resolver.Record(name, dns.TypeAAAA)
}

func (e PNS) obtainContentHash(name string, domain string) ([]byte, error) {
	plsDomain := strings.TrimSuffix(domain, ".")
	resolver, err := e.getResolver(plsDomain)
	if err != nil {
		return []byte{}, nil
	}

	return resolver.Contenthash()
}

func (e PNS) obtainTXTRRSet(name string, domain string) ([]byte, error) {
	plsDomain := strings.TrimSuffix(domain, ".")
	resolver, err := e.getDNSResolver(plsDomain)
	if err != nil {
		return []byte{}, nil
	}

	return resolver.Record(name, dns.TypeTXT)
}

// Name implements the Handler interface.
func (e PNS) Name() string { return "pns" }

// isRealOnChainDomain will return true if the name requested
// is also the domain, which implies the entry has an on-chain
// presence
func isRealOnChainDomain(name string, domain string) bool {
	return name == domain
}

var resolverCache *lru.Cache
var dnsResolverCache *lru.Cache

func init() {
	resolverCache, _ = lru.New(16)
	dnsResolverCache, _ = lru.New(16)
}

func (e *PNS) getDNSResolver(domain string) (*pns.DNSResolver, error) {
	log.Infof("getDNSResolver %s", domain)
	if !dnsResolverCache.Contains(domain) {
		resolver, err := pns.NewDNSResolver(e.Client, domain)
		if err == nil {
			dnsResolverCache.Add(domain, resolver)
		} else {
			log.Infof("getDNSResolver err %s", err.Error())
			if err.Error() == "no contract code at given address" ||
				strings.HasSuffix(err.Error(), " is not a DNS resolver contract") {
				dnsResolverCache.Add(domain, nil)
			}
		}
	}
	resolver, _ := dnsResolverCache.Get(domain)
	if resolver == nil {
		return nil, errors.New("no resolver")
	}
	return resolver.(*pns.DNSResolver), nil
}

func (e *PNS) newDNSResolver(domain string) (*pns.DNSResolver, error) {
	log.Infof("newDNSResolver %s", domain)
	// Obtain the resolver address for this domain
	resolver, err := e.Registry.ResolverAddress(domain)
	if err != nil {
		return nil, err
	}
	return pns.NewDNSResolverAt(e.Client, domain, resolver)
}

func (e *PNS) getResolver(domain string) (*pns.Resolver, error) {
	log.Infof("getResolver %s", domain)
	if !resolverCache.Contains(domain) {
		resolver, err := e.newResolver(domain)
		if err == nil {
			resolverCache.Add(domain, resolver)
		} else {
			log.Infof("getResolver err %s", err.Error())
			if err.Error() == "no contract code at given address" ||
				strings.HasSuffix(err.Error(), " is not a resolver contract") {
				resolverCache.Add(domain, nil)
			}
		}
	}
	resolver, _ := resolverCache.Get(domain)
	if resolver == nil {
		return nil, errors.New("no resolver")
	}
	return resolver.(*pns.Resolver), nil
}

func (e *PNS) newResolver(domain string) (*pns.Resolver, error) {
	log.Infof("newResolver %s", domain)
	// Obtain the resolver address for this domain
	resolver, err := e.Registry.ResolverAddress(domain)
	if err != nil {
		return nil, err
	}
	return pns.NewResolverAt(e.Client, domain, resolver)
}

// Ready returns true if we're ready to serve DNS records i.e. our chain is synced
func (e PNS) Ready() bool {
	progress, err := e.Client.SyncProgress(context.Background())
	log.Infof("Ready %s", err.Error())
	if err != nil {
		return false
	}
	return progress == nil
}
