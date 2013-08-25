/*
	Hyperfox - Man In The Middle Proxy for HTTP(s).

	Default client payload loaders for the Hyperfox tool.

	Written by Carlos Reventlov <carlos@reventlov.com>
	License MIT
*/

// This package provides default directors for the Hyperfox tool. The main
// purpose of these default directors is to check if special named payload
// files exists, if they do, they replace the client request.
package inject

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
// If found, the file contents will replace the original client's request
// headers.
//
// This file should be put in the request's working directory.
func Head(pr *proxy.ProxyRequest) error {

	file := proxy.Workdir + proxy.PS + "client" + proxy.PS + pr.FileName + proxy.PS + fmt.Sprintf("%s-head.payload", pr.Request.Method)

	_, err := os.Stat(file)

	if err == nil {
		fp, _ := os.Open(file)
		defer fp.Close()

		content, _ := ioutil.ReadAll(fp)

		lines := strings.Split(string(content), "\n")

		for _, line := range lines {
			hline := strings.SplitN(line, ":", 2)
			if len(hline) > 1 {
				pr.Request.Header.Set(strings.Trim(hline[0], " \r\n"), strings.Trim(hline[1], " \r\n"))
			}
		}

	}

	return nil
}

// Checks for a ${HTTP_METHOD}-body.payload file.
//
// If found, the file contents will replace the original client's request body.
//
// This file should be put in the request's working directory.
func Body(pr *proxy.ProxyRequest) error {

	file := proxy.Workdir + proxy.PS + "client" + proxy.PS + pr.FileName + proxy.PS + fmt.Sprintf("%s-body.payload", pr.Request.Method)

	stat, err := os.Stat(file)

	if err == nil {
		fp, _ := os.Open(file)

		pr.Request.ContentLength = stat.Size()
		pr.Request.Header.Set("Content-Length", strconv.Itoa(int(pr.Request.ContentLength)))
		pr.Request.Body.Close()

		pr.Request.Body = fp
	}

	return nil
}
