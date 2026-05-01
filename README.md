# wkd-proxy
A WKD endpoint that proxies to an authoritative HKP server

## Rationale

Proton customers with custom domains cannot currently point their WKD DNS record to Proton's keyserver.
Their keyserver only supports `protonmail.com` and `proton.me` addresses over WKD.
This is because WKD requires per-domain TLS certificates, and it is impractical for Proton to maintain valid TLS certificates for thousands of custom domains.

A Proton customer can host this tool on their own infrastructure, behind their own TLS certificate.
It will proxy each incoming WKD request to the Proton keyserver via a standard HKPv1 request, which does not require a custom TLS certificate.

### BEWARE

This software is only intended to be used to proxy a verifying HKPv1 keyserver, such as the Proton keyserver.
You MUST NOT configure the WKD proxy to use a non-verifying keyserver.

The entire point of WKD is that it is authoritative for the domain.
Using a non-verifying keyserver could permit an attacker to inject a fake certificate.

## Configuration

The default configuration file is in `etc/wkd-proxy.conf`, and is in TOML format.
The service will listen on HTTP, and default to port 8080.
It will also listen on HTTPS if the appropriate section is uncommented.

## Testing

There is a `docker-compose.yml` file in the root of the repo.
The easiest thing to do is to incant:

```
docker compose build
docker compose up -d
```

After that, you can make standard WKD requests to localhost as follows:

```
curl 'http://localhost:8080/.well-known/openpgpkey/protonmail.com/hu/x?l=postmaster' -o postmaster.pgp
```

Like many WKD services that don't rely on static files, the "hashed local part" field is ignored in favour of the `l`.
Note however that you must specify *something* in the "hashed local part" field.
In the above example, this is the string `x`.
Passing the empty string will cause a bad request error.

## Running in production

The recommended way is to configure a web server (such as Apache or Nginx) to reverse proxy `/.well-known/openpgpkey` to `wkd-proxy`.
The web server MUST be configured to listen on `https://openpgpkey.<domain>`, and have the correct DNS record pointing to it.
It is recommended to manage TLS certificates in the reverse proxy.

Note that `wkd-proxy` will proxy *all* WKD requests to the configured keyserver, regardless of domain.
If you need to proxy multiple domains to different keyservers via the same web server, you will need to run multiple copies of `wkd-proxy`.
Your reverse proxy will need to be configured to separately forward each `/.well-known/openpgpkey/<domain>` to the correct instance of `wkd-proxy`.
