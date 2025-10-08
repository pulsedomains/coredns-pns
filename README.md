# CoreDNS-PNS
A CoreDNS plugin for resolving PulseChain Name Service (PNS) records and IPFS content  

---

## Introduction
CoreDNS-PNS is a plugin for [CoreDNS](https://coredns.io) that allows DNS resolution of domains stored on the **PulseChain blockchain** and enables access to **IPFS/IPNS/Skynet** content via PNS.  

This plugin was originally used to power **[PLS.TO](https://pls.to)**, a community gateway that provided zero-configuration access to PNS names and decentralized content in any standard web browser.  

The hosted **PLS.TO** service is no longer active, but with this plugin you can **self-host your own gateway** or contribute to the ecosystem.


---

## Why PLS.TO?
- **Decentralized access:** Brings IPFS/IPNS/Skynet sites into any Web2 browser  
- **Zero-configuration:** No setup required for end users or developers  
- **Transition bridge:** Helps users and projects move from centralized Web2 to decentralized Web3  
- **Privacy & security:** Enforced TLS encryption, strong isolation headers, and anonymized logs  

---

## How It Works
PLS.TO functioned as a **reverse proxy** powered by the **CoreDNS-PNS plugin**:

1. Requests for `*.pls.to` domains were rewritten internally to `*.pls` domains.  
2. The PNS resolver fetched DNS records from the PulseChain blockchain.  
3. If the PNS record contained a contenthash, the gateway returned the corresponding IPFS/IPNS/Skynet content over HTTPS.  
4. This allowed decentralized content to be loaded in any browser, just like a regular website.  

---

## CoreDNS-PNS Plugin
The **CoreDNS-PNS plugin** is the core component that enabled PLS.TO to function. It has two primary purposes:  

1. A general-purpose DNS resolver for DNS records stored on PulseChain  
2. A specialised resolver for IPFS content hashes and gateways  

Details of the first feature can also be found in the reference implementation: [EthDNS article](http://www.wealdtech.com/articles/ethdns-an-ethereum-backend-for-the-domain-name-system/).  

The second feature provides a mechanism to map DNS domains to PNS domains by removing the `.pls.to` suffix.  
- Example: `wealdtech.pls.to` → `wealdtech.pls`  
- If an **A or AAAA record** is requested: returns the IPFS gateway address  
- If a **TXT record** is requested: returns the contenthash  

This makes IPFS content directly retrievable by any web browser.  

---

## Building
- Latest build available via Docker:  

```bash
docker pull pulsedomains/coredns-pns:latest
```

- To build standalone with the plugin enabled:  
  Run the included `build-standalone.sh` script (compatible with most Unix-like systems).  

---

## Example Corefile
Below is an annotated Corefile configuration:

```coredns
# This section enables DNS lookups for all domains on PNS
. {
  rewrite stop {
    # Rewrite *.pls.to → *.pls for resolution
    name regex (.*)\.pls\.to {1}.pls
    answer name (.*)\.pls {1}.pls.to
  }
  pns {
    # Connection to a PulseChain node (local node strongly recommended)
    # Can be an IPC socket path or JSON-RPC URL
    connection /home/ethereum/.ethereum/geth.ipc

    # Nameservers serving PlsLink domains
    plslinknameservers ns1.plsdns.fyi ns2.plsdns.fyi

    # IPv4 PNS-enabled IPFS gateway
    ipfsgatewaya 176.9.154.81

    # IPv6 PNS-enabled IPFS gateway
    ipfsgatewayaaaa 2a01:4f8:160:4069::2
  }

  # Enable DNS forwarding (only for private/non-public use)
  forward . 8.8.8.8

  errors
}
```

> Note: It is also possible to run the DNS server over **TLS** or **HTTPS**.  
> See the [CoreDNS documentation](https://coredns.io/manual/tls/) for certificate setup details.  

---

## Running Standalone
Running CoreDNS standalone is simply a case of starting the binary.  
See the [CoreDNS documentation](https://coredns.io/manual/toc/) for further details.  

---

## Running with Docker
To run CoreDNS with Docker:

```bash
docker run -p 53:53/udp   --volume=/home/coredns:/etc/coredns   pulsedomains/coredns-pns:latest
```

Where `/home/coredns` is the directory containing your **Corefile** and TLS certificates.  

---

## Status of PLS.TO
The official **PLS.TO gateway** is no longer actively maintained.  

However, the **CoreDNS-PNS plugin and Docker images remain open source**. This means anyone can:  
- Host their own gateway  
- Contribute improvements to the codebase  
- Ensure the long-term resilience of PNS + IPFS access for the PulseChain community  

This transition empowers the community to take full ownership of the infrastructure — aligning with the decentralized ethos of Web3.  

---

## License
Made with ❤️ by [Pulse.Domains](https://pulse.domains)  
Open-sourced for the PulseChain community.
