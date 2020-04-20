package capture

import (
	"bytes"
	"compress/gzip"
	"io"
	"regexp"
)

var (
	reUnsafeChars   = regexp.MustCompile(`[^0-9a-zA-Z\s\.]`)
	reRepeatedBlank = regexp.MustCompile(`\s+`)
)

var (
	peekLength    = int64(1024 * 1024 * 10)
	minWordLength = 3
	space         = []byte{' '}
)

func peek(in io.Reader) []byte {
	out := bytes.NewBuffer(nil)
	tee := io.TeeReader(in, out)

	gz, err := gzip.NewReader(tee)
	if err == nil {
		dst := bytes.NewBuffer(nil)
		if _, err := io.CopyN(dst, gz, peekLength); err == nil {
			return bytes.TrimSpace(dst.Bytes())
		}
	}

	io.CopyN(out, in, peekLength)
	return bytes.TrimSpace(out.Bytes())
}

func extractKeywords(in ...io.Reader) []byte {
	keywords := []byte{}
	for i := range in {
		keywords = append(keywords, peek(in[i])...)
		keywords = append(keywords, ' ')
	}
	keywords = bytes.ToLower(keywords)
	keywords = reUnsafeChars.ReplaceAll(keywords, space)
	keywords = reRepeatedBlank.ReplaceAll(keywords, space)
	words := bytes.Split(keywords, space)
	keywords = []byte{}
	for i := range words {
		if len(words[i]) >= minWordLength {
			keywords = append(keywords, words[i]...)
			keywords = append(keywords, ' ')
		}
	}
	return keywords
}
