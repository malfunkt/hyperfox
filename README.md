# Hyperfox

Hyperfox is a security tool for transparently hijacking/proxying HTTP and HTTPs traffic.

HTTPs could be hijacked/proxied if, and only if, the client application accepts bogus
certificates.

Hyperfox could be used as a tool for auditing a wide range of applications, including
mobile apps.

## Features

* Saves all the traffic between client and server.
* Can modify server responses before arriving to the client.
* Can modify client requests before sending them to the destination server.
* Supports SSL.
* Supports streaming.

## Installation

Before installing, make sure you have a [working Go environment][1] and [git][2].

```sh
% go get github.com/xiam/hyperfox
% hyperfox -h
```

## Usage example

Run `hyperfox`, it will start in HTTP mode listening at `0.0.0.0:9999` by default.

```sh
% hyperfox
```

If you want to analyze HTTPs instead of HTTP, use the `-s` flag and provide appropriate
[cert.pem](https://github.com/xiam/hyperfox/raw/master/ssl/cert.pem) and
[key.pem](https://github.com/xiam/hyperfox/raw/master/ssl/key.pem) files.

```sh
% hyperfox -s -c ssl/cert.pem -k ssl/key.pem
```

`hyperfox` won't be of much use if the host machine has no traffic to analyze or if
the only traffic to analyze is its own.

A common usage on a LAN is putting the host machine in [forwarding mode][3], this will
allow the host to forward traffic and be used as a gateway.

```sh
# Linux
sysctl -w net.ipv4.ip_forward=1

# FreeBSD/OSX
sysctl -w net.inet.ip.forwarding=1
```

Then prepare the host machine to actually forward everything but the port we want to
analyze (80 in this example), we need all the traffic on that port to be redirected to
the port `hyperfox` is listening.

```sh
# Linux (HTTP)
iptables -A PREROUTING -t nat -i wlan0 -p tcp --destination-port 80 -j REDIRECT --to-port 9999

# FreeBSD/OSX (HTTP)
ipfw add fwd 127.0.0.1,9999 tcp from any to any 80 via wlan0
```

Finally, use [ARP spoofing][4] to trick other machines into *think* our host machine is
their router.

```sh
arpspoof -i wlan0 -t 10.0.0.123 10.0.0.1
```

The example above uses [arpspoof][5].

[1]: http://golang.org/doc/install
[2]: http://git-scm.com
[3]: http://en.wikipedia.org/wiki/IP_forwarding
[4]: http://en.wikipedia.org/wiki/ARP_spoofing
[5]: http://arpspoof.sourceforge.net/
