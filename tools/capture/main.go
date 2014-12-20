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
	ID         uint      `json:"id" db:",omitempty,json"`
	RemoteAddr string    `json:"remote_addr" db:",json"`
	Method     string    `json:"method" db:",json"`
	Status     int       `json:"status" db:",json"`
	Host       string    `json:"host" db:",json"`
	URL        string    `json:"url" db:",json"`
	Header     Header    `json:"header" db:",json"`
	Body       []byte    `json:"body" db:",json"`
	Date       time.Time `json:"date" db:",json"`
}

func (h Header) MarshalDB() (interface{}, error) {
	return json.Marshal(h.Header)
}

func (h *Header) UnmarshalDB(data interface{}) error {
	if b, ok := data.([]byte); ok {
		return json.Unmarshal(b, &h.Header)
	}
	return nil
}

type CaptureWriteCloser struct {
	r *http.Response
	c chan Response
	bytes.Buffer
}

func (cwc *CaptureWriteCloser) Close() error {

	r := Response{
		RemoteAddr: cwc.r.Request.RemoteAddr,
		Method:     cwc.r.Request.Method,
		Status:     cwc.r.StatusCode,
		Host:       cwc.r.Request.URL.Host,
		URL:        cwc.r.Request.URL.String(),
		Header:     Header{cwc.r.Request.Header},
		Body:       cwc.Bytes(),
		Date:       time.Now(),
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
	return &CaptureWriteCloser{r: res, c: c.c}, nil
}
