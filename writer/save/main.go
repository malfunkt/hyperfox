/*
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

func Body(pr *proxy.ProxyRequest) io.WriteCloser {

	file := "archive" + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + pr.Id + ".body"

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	if fp != nil {
		fmt.Printf("ww %v\n", file)
	}

	return fp
}

func Head(pr *proxy.ProxyRequest) io.WriteCloser {

	file := "archive" + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + pr.Id + ".head"

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	if fp != nil {
		fmt.Printf("ww %v\n", file)
		fp.WriteString(fmt.Sprintf("%s %s\r\n", pr.Response.Proto, pr.Response.Status))
		pr.Response.Header.Write(fp)
		fp.WriteString("\r\n")
		fp.Close()
	}

	return nil
}

func Response(pr *proxy.ProxyRequest) io.WriteCloser {

	file := "archive" + proxy.PS + "server" + proxy.PS + pr.FileName + proxy.PS + pr.Id

	os.MkdirAll(path.Dir(file), os.ModeDir|os.FileMode(0755))

	fp, _ := os.Create(file)

	if fp != nil {
		fmt.Printf("ww %v\n", file)
		fp.WriteString(fmt.Sprintf("%s %s\r\n", pr.Response.Proto, pr.Response.Status))
		pr.Response.Header.Write(fp)
		fp.WriteString("\r\n")
		return fp
	}

	return nil
}
