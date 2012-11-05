/*
	Written by José Carlos Nieto <xiam@menteslibres.org>
	License MIT
*/

package inject

import (
	"fmt"
	"github.com/xiam/hyperfox/proxy"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

/*
	Checks for a wire formatted .head.payload file.

	If the file exists, the actual response headers
	are replaced with the contents of the file.
*/
func Head(res *http.Response) error {

	head := proxy.ArchiveFile(res) + ".head.payload"

	_, err := os.Stat(head)

	if err == nil {
		fp, _ := os.Open(head)

		content, _ := ioutil.ReadAll(fp)

		lines := strings.Split(string(content), "\n")

		for _, line := range lines {
			hline := strings.SplitN(line, ":", 2)
			if len(hline) > 1 {
				res.Header.Set(strings.Trim(hline[0], " \r\n"), strings.Trim(hline[1], " \r\n"))
			}
		}

		fp.Close()
	}

	return nil
}

/*
	Checks for a .payload file.

	If the file exists, the actual response body
	is replaced with the content of the file.
*/
func Body(res *http.Response) error {

	file := proxy.ArchiveFile(res) + ".payload"

	stat, err := os.Stat(file)

	if err == nil {
		fp, _ := os.Open(file)

		res.Header.Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
		res.Body.Close()

		res.Body = fp
	}

	return nil
}
