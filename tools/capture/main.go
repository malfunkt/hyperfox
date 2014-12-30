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
	ID            uint      `json:"id" db:",omitempty,json"`
	Origin        string    `json:"origin" db:",json"`
	Method        string    `json:"method" db:",json"`
	Status        int       `json:"status" db:",json"`
	ContentType   string    `json:"content_type" db:",json"`
	ContentLength uint      `json:"content_length" db:",json"`
	Host          string    `json:"host" db:",json"`
	URL           string    `json:"url" db:",json"`
	Scheme        string    `json:"scheme" db:",json"`
	Path          string    `json:"path" db:",path"`
	Header        Header    `json:"header,omitempty" db:",json"`
	Body          []byte    `json:"body,omitempty" db:",json"`
	RequestHeader Header    `json:"request_header,omitempty" db:",json"`
	RequestBody   []byte    `json:"request_body,omitempty" db:",json"`
	DateStart     time.Time `json:"date_start" db:",json"`
	DateEnd       time.Time `json:"date_end" db:",json"`
	TimeTaken     int64     `json:"time_taken" db:",json"`
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

	io.Copy(reqbody, cwc.r.Request.Body)

	now := time.Now()

	r := Response{
		Origin:        cwc.r.Request.RemoteAddr,
		Method:        cwc.r.Request.Method,
		Status:        cwc.r.StatusCode,
		ContentType:   http.DetectContentType(cwc.Bytes()),
		ContentLength: uint(cwc.Len()),
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
