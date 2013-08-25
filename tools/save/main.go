/*
	Hyperfox - Man In The Middle Proxy for HTTP(s).

	Default Writers for the Hyperfox tool.

	Written by Carlos Reventlov <carlos@reventlov.com>
	License MIT
*/

package save

import (
	"errors"
	"fmt"
	"github.com/xiam/hyperfox/proxy"
	"io"
	"os"
	"path"
)

var (
	ErrCantWriteFile = errors.New(`Can't open file "%s" for writing.`)
)

// Returns a WriteCloser that can be user to write the server's
// response body to a .body file.
func Body(pr *proxy.ProxyRequest) (io.WriteCloser, error) {
	var err error

	file := proxy.Workdir + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + pr.Id + ".body"

	if pr.Response.ContentLength != 0 {
		os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

		fp, _ := os.Create(file)

		if fp == nil {
			err = fmt.Errorf(ErrCantWriteFile.Error(), file)
		}

		// Don't close io.WriterCloser yet.
		return fp, err
	}

	return nil, err
}

// Writes the server response headers to a .head file.
func Head(pr *proxy.ProxyRequest) (io.WriteCloser, error) {
	var err error

	file := proxy.Workdir + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + pr.Id + ".head"

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	if fp == nil {
		err = fmt.Errorf(ErrCantWriteFile.Error(), file)
	} else {
		defer fp.Close()
		fp.WriteString(fmt.Sprintf("%s %s\r\n", pr.Response.Proto, pr.Response.Status))
		pr.Response.Header.Write(fp)
		fp.WriteString("\r\n")
	}

	return nil, err
}

// Writes the full server response to disk.
func Response(pr *proxy.ProxyRequest) (io.WriteCloser, error) {
	var err error

	file := proxy.Workdir + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + pr.Id

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	if fp == nil {
		err = fmt.Errorf(ErrCantWriteFile.Error(), file)
	} else {
		// Don't close io.WriterCloser yet.
		fp.WriteString(fmt.Sprintf("%s %s\r\n", pr.Response.Proto, pr.Response.Status))
		pr.Response.Header.Write(fp)
		fp.WriteString("\r\n")
		return fp, nil
	}

	return nil, err
}
