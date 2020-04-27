# Hyperfox

[![Build Status](https://travis-ci.org/malfunkt/hyperfox.svg?branch=master)](https://travis-ci.org/malfunkt/hyperfox)

[Hyperfox][1] is a security auditing tool that proxies and records HTTP and
HTTPS traffic between two points.

## Installation

You can install the latest version of hyperfox to `/usr/local/bin` with the
following command (requires admin privileges):

```sh
curl -sL 'https://raw.githubusercontent.com/malfunkt/hyperfox/master/install.sh' | sh
```

If you'd rather not accept free candy from this van you can also grab a release
from our [releases page](https://github.com/malfunkt/hyperfox/releases) and
install it manually.

### Building `hyperfox` from source

In order to build `hyperfox` from source you'll need Go and a C compiler:

Use `go install` to build and install `hyperfox`:

```
go install github.com/malfunkt/hyperfox
```

## How does it work?

Hyperfox creates a transparent HTTP proxy server and binds it to port 1080/TCP
on localhost (`-addr 127.0.0.1 -http 1080`). The proxy server reads plaintext
HTTP requests and redirects them to the target destination (the `Host` header
is used to identify the destination), when the target destination replies,
Hyperfox intercepts the response and forwards it to the original client.

All HTTP communications between origin and destination are intercepted by
Hyperfox and recorded on a SQLite database that is created automatically.
Everytime Hyperfox starts, a new database is created (e.g.:
`hyperfox-00123.db`). You can change this behaviour by explicitly providing a
database name (e.g.: `-db traffic-log.db`).

### Usage

Launch Hyperfox with default configuration:

```
hyperfox
```

use `cURL` to request any HTTP page, the `-x` parameter tells cURL to use
hyperfox as proxy:

```
curl -x http://127.0.0.1:1080 example.com
```

you should be able to see a log for the page you requested in Hyperfox's output:

```
...
127.0.0.1:44254 - - [11/Apr/2020:19:19:48 -0500] "GET http://example.com/ HTTP/1.1" 200 -1
```

### User interface (`-ui`)

![hyperfox-ui](https://user-images.githubusercontent.com/385670/79090465-6e7eb300-7d0f-11ea-8fc6-df1e6da8a12e.png)

Use the `-ui` parameter to enable Hyperfox UI wich will open in a new browser
window:

```
hyperfox -db records.db -ui
```

The above command creates a web server that binds to `127.0.0.1:1984`. If you'd
like to change the bind address or port use the `-ui-addr` switch:

```
hyperfox -db records.db -ui -ui-addr 127.0.0.1:3000
```

Changing the UI server address is specially useful when Hyperfox is running on
a remote or headless host and you'd like to see the UI from another host.

Enabling the UI also enables a minimal REST API (at `127.0.0.1:4891`) that is
consumed by the front-end application.

Please note that Hyperfox's REST API is only protected by a randomly generated
key that changes everytime Hyperfox starts, depending on your use case this
might not be adecuate.

#### Run Hyperfox UI on your mobile device

When the `-ui-addr`parameter is different from `127.0.0.1` Hyperfox will
output a QR code to make it easier to connect from mobile devices:

```
hyperfox -db records.db -ui -ui-addr 192.168.1.23:1984
```

### SSL/TLS mode (`-ca-cert` & `-ca-key`)

SSL/TLS connections are secure end to end and protected from eavesdropping.
Hyperfox won't be able to see anything happening between a client and a secure
destination. This is only valid as long as the chain of trust remains
untouched.

Let's suppose that the client trusts a root CA certificate that is known by
Hyperfox, if that happens Hyperfox will be able to issue certificates that are
going to be trusted by the client.

Examples of such bogus root CA files be found here:

* [Hyperfox CA cert](https://raw.githubusercontent.com/malfunkt/hyperfox/master/ca/rootCA.crt)
* [Hyperfox CA key](https://raw.githubusercontent.com/malfunkt/hyperfox/master/ca/rootCA.key)

you can also [generate your own root CA certificate and
key](https://www.ibm.com/support/knowledgecenter/SSZQDR/com.ibm.rba.doc/LD_rootkeyandcert.html).

There are a [number of ways to install root CA
certificates](https://www.bounca.org/tutorials/install_root_certificate.html),
depending on your operating system.

This QR code might come in handy when installing Hyperfox's root CA on a mobile
device:

![Hyperfox root CA certificate](https://chart.googleapis.com/chart?cht=qr&choe=UTF-8&chs=220x220&chl=https://static.hyperfox.org/rootCA.crt)

Use the `-ca-cert` and `-ca-key` flags to provide Hyperfox with the root CA
certificate and key you'd like to use:

```
hyperfox -ca-cert rootCA.crt -ca-key rootCA.key
```

the above command creates a special server and binds it to `127.0.0.1:10443`,
this server waits for a SSL/TLS connection to arrive. When a new SSL/TLS
connection hits in, Hyperfox uses the
[SNI](https://en.wikipedia.org/wiki/Server_Name_Indication) extension to
identify the destination nameserver and to create a SSL/TLS certificate for it,
this certificate is signed with the providede root CA key.

#### TLS interception example

Launch Hyperfox with appropriate TLS parameters and `-http 443` (port 443
requires admin privileges).

```
sudo hyperfox -ca-cert ./ca/rootCA.crt -ca-key ./ca/rootCA.key -https 443
```

Use cURL to build a HTTPs request to example.com: the `-resolve` option tells
cURL to skip DNS verification and use `127.0.0.1` as if it were the legitimate
address for `example.com`, while the `-k` parameter tells cURL to accept any
TLS certificate.

```
curl -k -resolve example.com:443:127.0.0.1 https://example.com
```

you should be able to see a log for the page you requested in Hyperfox's output:

```
127.0.0.1:36398 - - [11/Apr/2020:19:36:56 -0500] "GET https://example.com/ HTTP/2.0" 200 -1
```

## Usage examples

### Via `/etc/hosts` on localhost

Add the host you'd like to inspect to your `/etc/hosts` file:

```
example.com 127.0.0.1
```

Run Hyperfox with the options you'd like, just remember that you should use
ports 80 for HTTP and 443 for HTTPS, and that requires admin privileges. In
addition to `-http` and `-https` use the `-dns` parameter with a valid DNS
resolver:

```
sudo hyperfox -ui -http 80 -dns 8.8.8.8
```

that will make Hyperfox skip the OS DNS resolver and use an alternative one
(remember that example.com points to 127.0.1).

Now use cURL and try to go to the destination:

```
curl http://example.com
```

Hyperfox will capture the request and print it to its output:

```
127.0.0.1:41766 - - [11/Apr/2020:19:43:30 -0500] "GET http://example.com/ HTTP/1.1" 200 -1
```

### Via ARP Spoofing on a LAN

See [MITM attack with Hyperfox and arpfox](https://xiam.dev/mitm-attack-with-hyperfox-and-arpfox/).

## Hacking

Choose an [issue][2], fix it and send a pull request.

## License

> Copyright (c) 2012-today José Nieto, https://xiam.io
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
[2]: https://github.com/malfunkt/hyperfox/issues
