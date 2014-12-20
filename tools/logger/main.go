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

// Package logger provides standard logger.
package logger

import (
	"fmt"
	"strings"
	"time"

	"github.com/xiam/hyperfox/proxy"
)

func chunk(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

// Stdout struct implements proxy.Logger
type Stdout struct {
}

// Log prints a standard log string to the system.
func (s Stdout) Log(pr *proxy.ProxiedRequest) error {

	line := []string{
		chunk(pr.Request.RemoteAddr),
		chunk(""),
		chunk(""),
		chunk("[" + time.Now().Format("02/Jan/2006:15:04:05 -0700") + "]"),
		chunk("\"" + fmt.Sprintf("%s %s %s", pr.Request.Method, pr.Request.URL, pr.Request.Proto) + "\""),
		chunk(fmt.Sprintf("%d", pr.Response.StatusCode)),
		chunk(fmt.Sprintf("%d", pr.Response.ContentLength)),
	}

	fmt.Println(strings.Join(line, " "))

	return nil
}
