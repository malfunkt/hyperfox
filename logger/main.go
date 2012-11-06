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
	"net/http"
	"os"
)

/*
	A very simple response logger that writes to *os.File.
*/
func Simple(fp *os.File) proxy.Logger {
	self := log.New(fp, "", 0)
	fn := func(res *http.Response) error {
		self.Printf("%s %s %s %s %s %d\n", res.Request.RemoteAddr, res.Request.Method, res.Request.URL, res.Proto, res.Status, res.ContentLength)
		return nil
	}
	return fn
}

/*
	Records full request to a (binary) .client file.
*/
func Request(res *http.Response) error {

	file := proxy.ClientFile(res)

	fmt.Printf("--> %s\n", file)

	proxy.Workdir(path.Dir(file))

	fp, _ := os.Create(file)

	if fp == nil {
		return fmt.Errorf("Could not open %s for writing.\n", file)
	}

	defer fp.Close()

	fp.WriteString(fmt.Sprintf("%s %s %s\r\n", res.Request.Method, res.Request.RequestURI, res.Request.Proto))

	res.Request.Header.Write(fp)

	fp.WriteString("\r\n");

	io.Copy(fp, res.Request.Body)

	res.Request.Body.Close()

	return nil
}
