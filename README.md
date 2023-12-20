# nameserver

Work in progress personal Cloudflare replacement SSL/DNS manager built with [caddy](https://github.com/caddyserver/caddy) and [HTMJ](https://github.com/acheong08/HTMJ).

## Build
`go build .`

### Dependencies

- Caddy
- A domain name? (Maybe 2 domain names. I don't know)

## Setup

- Caddy should be running at `127.0.0.1:2019`
- Create 2 A records pointing to your DNS server (e.g. ns1.yourdomain.com, ns2.yourdomain.com)
- Configure your nameserver for a domain to be the A records set previously
- Run the nameserver

## Usage
```
Usage of nameserver
  -debug
    	Debug mode
  -dns-addr string
    	DNS listen address (default ":5553")
  -http-addr string
    	HTTP listen address (default ":8080")
  -public-ip string
    	Public IP address (default "127.0.0.1")
```

DNS address should be run on `:53` except for during debugging

## Work in progress
- Rate limiting and OWASP firewall
