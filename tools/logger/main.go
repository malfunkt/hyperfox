/*
	Hyperfox

	Written by José Carlos Nieto <xiam@menteslibres.org>
	License MIT
*/

package logger

import (
	"github.com/xiam/hyperfox/proxy"
	"log"
	"path"
	"fmt"
	"io"
	"bytes"
	"io/ioutil"
	"os"
)

/*
	A very simple request logger that writes to *os.File.
*/
func Client(fp *os.File) proxy.Director {
	self := log.New(fp, "-> ", 0)

	fn := func(pr *proxy.ProxyRequest) error {
		self.Printf(
			"%s %s: %s %s %s %db\n",
			pr.Request.RemoteAddr,
			pr.Request.Host,
			pr.Request.Method,
			pr.Request.RequestURI,
			pr.Request.Proto,
			pr.Request.ContentLength,
		)
		return nil
	}

	return fn
}

/*
	A very simple response logger that writes to *os.File.
*/
func Server(fp *os.File) proxy.Logger {
	self := log.New(fp, "<- ", 0)

	fn := func(pr *proxy.ProxyRequest) error {
		self.Printf(
			"%s %s: %s %s %s %db %d\n",
			pr.Request.RemoteAddr,
			pr.Request.Host,
			pr.Request.Method,
			pr.Request.RequestURI,
			pr.Request.Proto,
			pr.Response.ContentLength,
			pr.Response.StatusCode,
		)
		return nil
	}

	return fn
}

/*
	Records full request to a (binary) file.
*/
func Request(pr *proxy.ProxyRequest) error {

	file := proxy.Workdir + proxy.PS + "client" + proxy.PS + pr.FileName + proxy.PS + pr.Id

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	defer fp.Close()

	if fp == nil {
		return fmt.Errorf("Could not open %s for writing.\n", file)
	}

	fp.WriteString(fmt.Sprintf("%s %s %s\r\n", pr.Request.Method, pr.Request.RequestURI, pr.Request.Proto))

	pr.Request.Header.Write(fp)

	fp.WriteString("\r\n")

	buf := bytes.NewBuffer(nil)
	io.Copy(io.MultiWriter(fp, buf), pr.Request.Body)
	pr.Request.Body = ioutil.NopCloser(buf)

	return nil
}

/*
	Records client's request body to a .body file.
*/
func Body(pr *proxy.ProxyRequest) error {

	file := proxy.Workdir + proxy.PS + "client" + proxy.PS + pr.FileName + proxy.PS + pr.Id + ".body"

	if pr.Request.ContentLength != 0 {

		os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

		fp, _ := os.Create(file)

		defer fp.Close()

		if fp == nil {
			return fmt.Errorf("Could not open %s for writing.\n", file)
		}

		buf := bytes.NewBuffer(nil)
		io.Copy(io.MultiWriter(fp, buf), pr.Request.Body)
		pr.Request.Body = ioutil.NopCloser(buf)
	}

	return nil
}

/*
	Records client's request headers to a wire formatted .head file.
*/
func Head(pr *proxy.ProxyRequest) error {

	file := proxy.Workdir + proxy.PS + "client" + proxy.PS + pr.FileName + proxy.PS + pr.Id + ".head"

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	defer fp.Close()

	if fp == nil {
		return fmt.Errorf("Could not open %s for writing.\n", file)
	}

	fp.WriteString(fmt.Sprintf("%s %s %s\r\n", pr.Request.Method, pr.Request.RequestURI, pr.Request.Proto))

	pr.Request.Header.Write(fp)

	fp.WriteString("\r\n")

	return nil
}
