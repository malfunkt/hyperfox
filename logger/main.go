/*
	Written by José Carlos Nieto <xiam@menteslibres.org>
	License MIT
*/

package logger

import (
	"github.com/xiam/hyperfox/proxy"
	"log"
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
