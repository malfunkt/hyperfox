/*
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
	Records full request to a (binary) .client file.
*/
func Request(pr *proxy.ProxyRequest) error {

	file := "archive" + proxy.PS + "client" + proxy.PS + pr.FileName + proxy.PS + pr.Id

	fmt.Printf("ww %v\n", file)

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

func Body(pr *proxy.ProxyRequest) error {

	file := "archive" + proxy.PS + "client" + proxy.PS + pr.FileName + proxy.PS + pr.Id + ".body"

	fmt.Printf("ww %v\n", file)

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	defer fp.Close()

	if fp == nil {
		return fmt.Errorf("Could not open %s for writing.\n", file)
	}

	buf := bytes.NewBuffer(nil)
	io.Copy(io.MultiWriter(fp, buf), pr.Request.Body)
	pr.Request.Body = ioutil.NopCloser(buf)

	return nil
}

func Head(pr *proxy.ProxyRequest) error {

	file := "archive" + proxy.PS + "client" + proxy.PS + pr.FileName + proxy.PS + pr.Id + ".head"

	fmt.Printf("ww %v\n", file)

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
