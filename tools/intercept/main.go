/*
	Hyperfox - Man In The Middle Proxy for HTTP(s).

	Default server payload loaders for the Hyperfox tool.

	Written by Carlos Reventlov <carlos@reventlov.com>
	License MIT
*/

package intercept

import (
	"fmt"
	"github.com/xiam/hyperfox/proxy"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// Checks for a wire formatted ${HTTP_METHOD}-head.payload file.
//
// If found, the file contents will replace the original server's response
// headers.
//
// This file should be put in the response's working directory.
func Head(pr *proxy.ProxyRequest) error {

	file := proxy.Workdir + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + fmt.Sprintf("%s-head.payload", pr.Request.Method)

	_, err := os.Stat(file)

	if err == nil {
		fp, _ := os.Open(file)

		if fp != nil {

			defer fp.Close()

			content, _ := ioutil.ReadAll(fp)

			lines := strings.Split(string(content), "\n")

			for _, line := range lines {
				hline := strings.SplitN(line, ":", 2)
				if len(hline) > 1 {
					pr.Response.Header.Set(strings.Trim(hline[0], " \r\n"), strings.Trim(hline[1], " \r\n"))
				}
			}
		}

	}

	return nil
}

// Checks for a ${HTTP_METHOD}-body.payload file.
//
// If found, the file contents will replace the original server's response
// body.
//
// This file should be put in the response's working directory.
func Body(pr *proxy.ProxyRequest) error {

	file := proxy.Workdir + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + fmt.Sprintf("%s-body.payload", pr.Request.Method)

	stat, err := os.Stat(file)

	if err == nil {

		fp, _ := os.Open(file)

		if fp != nil {

			defer fp.Close()

			pr.Response.ContentLength = stat.Size()
			pr.Response.Header.Set("Content-Length", strconv.Itoa(int(pr.Response.ContentLength)))
			pr.Response.Body.Close()

			pr.Response.Body = fp
		}
	}

	return nil
}
