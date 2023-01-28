package pns

import (
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/ethereum/go-ethereum/ethclient"
	pns "github.com/pulsedomains/go-pns/v3"
)

func init() {
	caddy.RegisterPlugin("pns", caddy.Plugin{
		ServerType: "dns",
		Action:     setupPNS,
	})
}

func setupPNS(c *caddy.Controller) error {
	connection, plsLinkNameServers, ipfsGatewayAs, ipfsGatewayAAAAs, err := pnsParse(c)
	if err != nil {
		return plugin.Error("pns", err)
	}

	client, err := ethclient.Dial(connection)
	if err != nil {
		return plugin.Error("pns", err)
	}

	// Obtain the registry contract
	registry, err := pns.NewRegistry(client)
	if err != nil {
		return plugin.Error("pns", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return PNS{
			Next:               next,
			Client:             client,
			PlsLinkNameServers: plsLinkNameServers,
			Registry:           registry,
			IPFSGatewayAs:      ipfsGatewayAs,
			IPFSGatewayAAAAs:   ipfsGatewayAAAAs,
		}
	})

	return nil
}

func pnsParse(c *caddy.Controller) (string, []string, []string, []string, error) {
	var connection string
	plsLinkNameServers := make([]string, 0)
	ipfsGatewayAs := make([]string, 0)
	ipfsGatewayAAAAs := make([]string, 0)

	c.Next()
	for c.NextBlock() {
		switch strings.ToLower(c.Val()) {
		case "connection":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return "", nil, nil, nil, c.Errf("invalid connection; no value")
			}
			if len(args) > 1 {
				return "", nil, nil, nil, c.Errf("invalid connection; multiple values")
			}
			connection = args[0]
		case "plslinknameservers":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return "", nil, nil, nil, c.Errf("invalid plslinknameservers; no value")
			}
			plsLinkNameServers = make([]string, len(args))
			copy(plsLinkNameServers, args)
		case "ipfsgatewaya":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return "", nil, nil, nil, c.Errf("invalid IPFS gateway A; no value")
			}
			ipfsGatewayAs = make([]string, len(args))
			copy(ipfsGatewayAs, args)
		case "ipfsgatewayaaaa":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return "", nil, nil, nil, c.Errf("invalid IPFS gateway AAAA; no value")
			}
			ipfsGatewayAAAAs = make([]string, len(args))
			copy(ipfsGatewayAAAAs, args)
		default:
			return "", nil, nil, nil, c.Errf("unknown value %v", c.Val())
		}
	}
	if connection == "" {
		return "", nil, nil, nil, c.Errf("no connection")
	}
	if len(plsLinkNameServers) == 0 {
		return "", nil, nil, nil, c.Errf("no plslinknameservers")
	}
	for i := range plsLinkNameServers {
		if !strings.HasSuffix(plsLinkNameServers[i], ".") {
			plsLinkNameServers[i] = plsLinkNameServers[i] + "."
		}
	}
	return connection, plsLinkNameServers, ipfsGatewayAs, ipfsGatewayAAAAs, nil
}
