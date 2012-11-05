# Hyperfox

Hyperfox is a command line utility for transparently hijacking HTTP traffic.

## Features

* Saves all the captured data and headers.
* Can modify response headers and body before arriving to the client.
* Supports streaming.
* It's also a library for making HTTP proxies with *Go*, so you can customize any
other actions at a very detailed level.

## Installation

Before installing make sure you have a [working Go environment][1] and [git][2].

```
% go install github.com/xiam/hyperfox
% hyperfox
```

## Usage example

`hyperfox` won't be of much use if the host machine has no traffic to analyze or if
the only traffic to analyze is its own.

A common usage on a LAN is putting the host machine in [forwarding mode][3], this will
allow the host to forward traffic and be used as a gateway.

```
# Linux
sysctl net.ipv4.ip_forward=1

# FreeBSD/OSX
sysctl net.inet.ip.forwarding=1
```

Then prepare the host machine to actually forward everything but the port we want to
analyze, we need all the traffic to this port to be redirected to the port `hyperfox`
is listening.

```
# Linux
iptables -A PREROUTING -t nat -i wlan0 -p tcp --destination-port 80 -j REDIRECT --to-port 9999

# FreeBSD/OSX
ipfw add fwd 127.0.0.1,9999 tcp from any to any 80 via wlan0
```

Finally, use [ARP spoofing][4] to trick other machines into *think* our host machine is
its router.

```
arpspoof -i wlan0 -t 10.0.0.123 10.0.0.1
```

The example above uses [arpspoof][5].

[1]: http://golang.org/doc/install
[2]: http://git-scm.com
[3]: http://en.wikipedia.org/wiki/IP_forwarding
[4]: http://en.wikipedia.org/wiki/ARP_spoofing
[5]: http://arpspoof.sourceforge.net/
