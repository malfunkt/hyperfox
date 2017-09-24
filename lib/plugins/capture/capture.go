package capture

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type Header struct {
	http.Header
}

type Response struct {
	ID            uint64    `json:"id" db:"id,omitempty"`
	Origin        string    `json:"origin" db:"origin"`
	Method        string    `json:"method" db:"method"`
	Status        int       `json:"status" db:"status"`
	ContentType   string    `json:"content_type" db:"content_type"`
	ContentLength uint64    `json:"content_length" db:"content_length"`
	Host          string    `json:"host" db:"host"`
	URL           string    `json:"url" db:"url"`
	Scheme        string    `json:"scheme" db:"scheme"`
	Path          string    `json:"path" db:"path"`
	Header        Header    `json:"header,omitempty" db:"header"`
	Body          []byte    `json:"body,omitempty" db:"body"`
	RequestHeader Header    `json:"request_header,omitempty" db:"request_header"`
	RequestBody   []byte    `json:"request_body,omitempty" db:"request_body"`
	DateStart     time.Time `json:"date_start" db:"date_start"`
	DateEnd       time.Time `json:"date_end" db:"date_end"`
	TimeTaken     int64     `json:"time_taken" db:"time_taken"`
}

func (h Header) MarshalDB() (interface{}, error) {
	return json.Marshal(h.Header)
}

func (h *Header) UnmarshalDB(data interface{}) error {
	if s, ok := data.(string); ok {
		return json.Unmarshal([]byte(s), &h.Header)
	}
	return nil
}

type CaptureWriteCloser struct {
	r *http.Response
	c chan Response
	s time.Time
	bytes.Buffer
}

func (cwc *CaptureWriteCloser) Close() error {
	reqbody := bytes.NewBuffer(nil)

	if cwc.r.Request.Body != nil {
		io.Copy(reqbody, cwc.r.Request.Body)
		cwc.r.Request.Body.Close()
	}

	now := time.Now()
	r := Response{
		Origin:        cwc.r.Request.RemoteAddr,
		Method:        cwc.r.Request.Method,
		Status:        cwc.r.StatusCode,
		ContentType:   http.DetectContentType(cwc.Bytes()),
		ContentLength: uint64(cwc.Len()),
		Host:          cwc.r.Request.URL.Host,
		URL:           cwc.r.Request.URL.String(),
		Scheme:        cwc.r.Request.URL.Scheme,
		Path:          cwc.r.Request.URL.Path,
		Header:        Header{cwc.r.Header},
		Body:          cwc.Bytes(),
		RequestHeader: Header{cwc.r.Request.Header},
		RequestBody:   reqbody.Bytes(),
		DateStart:     cwc.s,
		DateEnd:       now,
		TimeTaken:     now.UnixNano() - cwc.s.UnixNano(),
	}

	cwc.c <- r

	return nil
}

type Capture struct {
	c chan Response
}

func New(c chan Response) *Capture {
	return &Capture{c: c}
}

func (c *Capture) NewWriteCloser(res *http.Response) (io.WriteCloser, error) {
	return &CaptureWriteCloser{r: res, c: c.c, s: time.Now()}, nil
}
