# Hyperfox

[![Build Status](https://travis-ci.org/malfunkt/hyperfox.svg?branch=master)](https://travis-ci.org/malfunkt/hyperfox)

[Hyperfox][1] is a security auditing tool that proxies and records HTTP traffic
between two points.

## Installation

You can install the latest version of hyperfox to `/usr/local/bin` with the
following command (requires admin privileges):

```sh
curl -sL 'https://raw.githubusercontent.com/malfunkt/hyperfox/master/install.sh' | sh
```

You can also grab a release from our [releases
page](https://github.com/malfunkt/hyperfox/releases) and install it manually.

## How does it work?

Hyperfox creates a transparent HTTP proxy server and binds it to port 1080/TCP
on localhost (`--addr 127.0.0.1 --http 1080`). The proxy server reads plaintext
HTTP requests and redirects them to the target destination (the `Host` header
is used to identify the destination), when the target destination replies,
Hyperfox intercepts the response and forwards it to the original client.

All HTTP communications between origin and destination are intercepted by
Hyperfox and recorded on a SQLite database that is created automatically.
Everytime Hyperfox starts, a new database is created (e.g.:
`hyperfox-00123.db`). You can change this behaviour by explicitly providing a
database name (e.g.: `--db traffic-log.db`).

### User interface (`--ui`)

Use the `--ui` parameter to enable Hyperfox UI. If you're running on a system
with a GUI and a web browser, Hyperfox will attempt to open a browser window to
display its UI:

```
hyperfox --db records.db --ui
```

The above command creates a web server that binds to 127.0.0.1:1984. If you'd
like to change the bind address or port use the `--ui-addr` switch:

```
hyperfox --db records.db --ui --ui-addr 127.0.0.1:3000
```

Changing the UI server address is specially useful when Hyperfox is running on
a remote or headless host and you'd like to see the UI from another host.

Enabling the UI also enables a minimal REST API (at `127.0.0.1:4891`) that is
consumed by the front-end application.

Please note that Hyperfox's REST API is only protected by a randomly generated
key that changes everytime Hyperfox starts, this might not be adecuate
depending on your use case.

#### Run Hyperfox UI on your mobile device

When the `--ui-addr`parameter is different from `127.0.0.1` Hyperfox will
output a QR code to make it easier to connect from mobile devices:

```
hyperfox --db records.db --ui --ui-addr 192.168.1.23:1984
```

### SSL/TLS mode (`--ca-cert` & `--ca-key`)

SSL/TLS connections are secure end to end and protected from eavesdropping.
Hyperfox won't be able to see anything happening between a client and a secure
destination. This is only valid as long as the chain of trust remains
untouched.

Let's suppose the client trusts a root CA controlled by Hyperfox, in this case
Hyperfox will be able to be part of the chain of trust by issuing certificates
signed by the bogus CA.

Examples of such bogus root CA can be found here:

* [Hyperfox CA cert](https://raw.githubusercontent.com/malfunkt/hyperfox/master/ca/rootCA.crt)
* [Hyperfox CA key](https://raw.githubusercontent.com/malfunkt/hyperfox/master/ca/rootCA.key)

Use the `--ca-cert` and `--ca-key` flags to provide Hyperfox with the root CA
certificate and key you'd like to use:

```
hyperfox --ca-cert rootCA.crt --ca-key rootCA.key
```

the above command creates a special server and binds it to `127.0.0.1:10443`,
this server waits for a SSL/TLS connection to arrive. When a new SSL/TLS hits
in, Hyperfox uses the
[SNI](https://en.wikipedia.org/wiki/Server_Name_Indication) extension to
identify the destination nameserver and to create a SSL/TLS certificate for it,
this certificate is signed with the root CA key.

## Usage

### Via `/etc/hosts`

```
example.org 127.0.0.1
```

```
hyperfox --http 80
```

### Via `arpfox` (ARPSpoofing)

## Hacking

Choose an [issue][3], fix it and send a pull request.

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
[2]: https://en.wikipedia.org/wiki/Man-in-the-middle_attack
[3]: https://github.com/malfunkt/hyperfox/issues
