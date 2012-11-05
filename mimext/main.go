/*
	Written by José Carlos Nieto <xiam@menteslibres.org>
	License MIT
*/

package mimext

import (
	"io/ioutil"
	"mime"
	"os"
	"regexp"
	"strings"
)

const Source = "/etc/mime.types"

var extensions = make(map[string][]string)

/*
	Returns the first extension found in the extensions map
	for the given MIME string.
*/
func Ext(mimeString string) string {
	mimeType, _, _ := mime.ParseMediaType(mimeString)
	if len(extensions[mimeType]) > 0 {
		return extensions[mimeType][0]
	}
	return "dat"
}

/*
	Populates the extensions map with contents from Source.
*/
func init() {
	re, _ := regexp.Compile(`[\t\s]+`)

	_, err := os.Stat(Source)
	if err == nil {
		text, _ := ioutil.ReadFile(Source)
		lines := strings.Split(string(text), "\n")

		for _, line := range lines {
			if len(line) > 1 {
				if line[0:1] != "#" {
					line = strings.Trim(re.ReplaceAllString(line, " "), " ")

					def := strings.Split(line, " ")

					if len(def) > 1 {
						extensions[def[0]] = def[1:]
					}
				}
			}
		}

	}

}
