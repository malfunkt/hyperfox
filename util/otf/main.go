// Copyright (c) 2012-2014 José Carlos Nieto, https://menteslibres.net/xiam
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package otf provides SSL certificates on the fly.
package otf

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"sync"
	"time"
)

const (
	certDirectory = "certs"
	rsaBits       = 2048
	pathSeparator = string(os.PathSeparator)
)

var (
	rootCACert = "../ssl/rootCA.crt"
	rootCAKey  = "../ssl/rootCA.key"
)

var (
	mu sync.Mutex
)

// SetRootCACert sets the root CA cert.
func SetRootCACert(s string) {
	rootCACert = s
}

// SetRootCAKey sets the root CA key.
func SetRootCAKey(s string) {
	rootCAKey = s
}

// CreateKeyPair creates a key pair for the given hostname on the fly.
func CreateKeyPair(hostName string) (certFile string, keyFile string, err error) {

	mu.Lock()

	defer mu.Unlock()

	h := sha1.New()
	h.Write([]byte(hostName))
	hostHash := fmt.Sprintf("%x", h.Sum(nil))

	certFile = certDirectory + pathSeparator + hostHash + ".crt"
	keyFile = certDirectory + pathSeparator + hostHash + ".key"

	// Attempt to verify certs.
	if _, err = tls.LoadX509KeyPair(certFile, keyFile); err == nil {
		// Keys already in place
		return
	}

	log.Printf("Creating SSL certificate for %s...", hostName)

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	var serialNumber *big.Int

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)

	if serialNumber, err = rand.Int(rand.Reader, serialNumberLimit); err != nil {
		return
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
			CommonName:   hostName,
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	if ip := net.ParseIP(hostName); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, hostName)
	}

	var rootCA tls.Certificate

	if rootCA, err = tls.LoadX509KeyPair(rootCACert, rootCAKey); err != nil {
		return
	}

	if rootCA.Leaf, err = x509.ParseCertificate(rootCA.Certificate[0]); err != nil {
		return
	}

	var priv *rsa.PrivateKey

	if priv, err = rsa.GenerateKey(rand.Reader, rsaBits); err != nil {
		return
	}

	var derBytes []byte

	if derBytes, err = x509.CreateCertificate(rand.Reader, &template, rootCA.Leaf, &priv.PublicKey, rootCA.PrivateKey); err != nil {
		return
	}

	if err = os.MkdirAll(certDirectory, 0755); err != nil {
		return
	}

	certOut, err := os.Create(certFile)

	if err != nil {
		return
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return
	}

	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()

	return
}
