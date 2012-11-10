/*
	Hyperfox

	Written by José Carlos Nieto <xiam@menteslibres.org>
	License MIT
*/

package save

import (
	"github.com/xiam/hyperfox/proxy"
	"io"
	"fmt"
	"path"
	"os"
)

/*
	Returns a WriteCloser that can be user to write the server's
	response body to a .body file.
*/
func Body(pr *proxy.ProxyRequest) io.WriteCloser {

	file := proxy.Workdir + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + pr.Id + ".body"

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	if fp == nil {
		fmt.Errorf(fmt.Sprintf("Could not open file %s for writing.", file))
	}

	return fp
}

/*
	Writes the server response headers to a .head file.
*/
func Head(pr *proxy.ProxyRequest) io.WriteCloser {

	file := "archive" + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + pr.Id + ".head"

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	if fp == nil {
		fmt.Errorf(fmt.Sprintf("Could not open file %s for writing.", file))
	} else {
		fp.WriteString(fmt.Sprintf("%s %s\r\n", pr.Response.Proto, pr.Response.Status))
		pr.Response.Header.Write(fp)
		fp.WriteString("\r\n")
		fp.Close()
	}

	return nil
}

/*
	Writes the full server response to disk.
*/
func Response(pr *proxy.ProxyRequest) io.WriteCloser {

	file := "archive" + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + pr.Id

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	if fp == nil {
		fmt.Errorf(fmt.Sprintf("Could not open file %s for writing.", file))
	} else {
		fp.WriteString(fmt.Sprintf("%s %s\r\n", pr.Response.Proto, pr.Response.Status))
		pr.Response.Header.Write(fp)
		fp.WriteString("\r\n")
		return fp
	}

	return nil
}
