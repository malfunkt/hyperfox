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
	peekLength    = 1024 * 1024 * 10
	minWordLength = 3
	space         = []byte{' '}
)

func peek(in io.Reader) []byte {
	dest := make([]byte, peekLength)

	gz, err := gzip.NewReader(in)
	if err == nil {
		_, _ = gz.Read(dest)
	} else {
		_, _ = in.Read(dest)
	}

	return bytes.TrimSpace(dest)
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
