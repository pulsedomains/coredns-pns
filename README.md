# coredns-pns

CoreDNS-PNS is a CoreDNS plugin that resolves DNS information over PNS.  It has two primary purposes:

  1. A general-purpose DNS resolver for DNS records stored on the PulseChain blockchain
  2. A specialised resolver for IPFS content hashes and gatways

Details of the first feature can be found at http://www.wealdtech.com/articles/ethdns-an-ethereum-backend-for-the-domain-name-system/

The second feature provides a mechanism to map DNS domains to PNS domains by removing the relevant suffix, for example the DNS domain `wealdtech.pls.to` maps to the PNS domain `wealdtech.pls`, and returning information for IPFS gateways (if an A or AAAA record is requested) as well as IPFS and content hashes (if a TXT record is requested).  The result of this is that IPFS content can be obtained from any web browser by simply 

# Building

The latest build is always available as a docker repository at `pulsedomains/coredns-pns`.  If you want to build a standalone copy of CoreDNS with this plugin enabled run the `build-standalone.sh` script, which should work on most unix-like systems.

# Corefile

The plugin has a number of configuration options.  An example annotated Corefile is shown below:

```
# This section enables DNS lookups for all domains on PNS
. {
  rewrite stop {
    # This rewrites any requests for *.pls.to domains to *.pls internally
    # prior to being processed by the main PNS resolver.
    name regex (.*)\.pls\.to {1}.pls
    answer name (.*)\.pls {1}.pls.to
  }
  pns {
    # connection is the connection to an Ethereum node.  It is *highly*
    # recommended that a local node is used, as remote connections can
    # cause DNS requests to time out.
    # This can be either a path to an IPC socket or a URL to a JSON-RPC
    # endpoint.
    connection /home/ethereum/.ethereum/geth.ipc

    # plslinknameservers are the names of the nameservers that serve
    # PlsLink domains.  This will usually be the name of this server,
    # plus potentially one or more others.
    plslinknameservers ns1.plsdns.fyi ns2.plsdns.fyi

    # ipfsgatewaya is the address of an PNS-enabled IPFS gateway.
    # This value is returned when a request for an A record of an Ethlink
    # domain is received and the domain has a contenthash record in PNS but
    # no A record.  Multiple values can be supplied, separated by a space,
    # in which case all records will be returned.
    ipfsgatewaya 176.9.154.81

    # ipfsgatewayaaaa is the address of an PNS-enabled IPFS gateway.
    # This value is returned when a request for an AAAA record of an Ethlink
    # domain is received and the domain has a contenthash record in PNS but
    # no A record.  Multiple values can be supplied, separated by a space,
    # in which case all records will be returned.
    ipfsgatewayaaaa 2a01:4f8:160:4069::2
  }

  # This enables DNS forwarding.  It should only be enabled if this DNS server
  # is not exposed to the internet, otherwise it becomes an open DNS server and
  # will be flooded with attack packets.
  forward . 8.8.8.8

  errors
}
```

It is also possible to run the DNS server over TLS or over HTTPS; details on how to set up certificates the can be found in the CoreDNS documentation.

# Running standalone

Running CoreDNS standalone is simply a case of starting the binary.  See the CoreDNS documentation for further information.

# Running with Docker

Running CoreDNS with Docker requires running the image created in the `Building` section.  A sample command-line might be:

    docker run -p 53:53/udp --volume=/home/coredns:/etc/coredns pulsedomains/coredns-pns:latest

where `/home/coredns` is the directory on the server that contains the Corefile and certificates.
