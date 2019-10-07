package capture

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type Header struct {
	http.Header `json:",inline"`
}

type ResponseMeta struct {
	ID            uint64    `json:"id" db:"id,omitempty"`
	Origin        string    `json:"origin" db:"origin"`
	Method        string    `json:"method" db:"method"`
	Status        int       `json:"status" db:"status"`
	ContentType   string    `json:"content_type" db:"content_type"`
	ContentLength uint64    `json:"content_length" db:"content_length"`
	Host          string    `json:"host" db:"host"`
	URL           string    `json:"url" db:"url"`
	Path          string    `json:"path" db:"path"`
	Scheme        string    `json:"scheme" db:"scheme"`
	DateStart     time.Time `json:"date_start" db:"date_start"`
	DateEnd       time.Time `json:"date_end" db:"date_end"`
	TimeTaken     int64     `json:"time_taken" db:"time_taken"`
}

type Response struct {
	ResponseMeta `json:",inline" db:",inline"`

	RequestHeader Header `json:"request_header,omitempty" db:"request_header"`
	RequestBody   []byte `json:"request_body,omitempty" db:"request_body"`

	Header Header `json:"header,omitempty" db:"header"`
	Body   []byte `json:"body,omitempty" db:"body"`
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
	res  *http.Response
	resp chan *Response
	t    time.Time
	bytes.Buffer
}

func (cwc *CaptureWriteCloser) Close() error {
	reqbody := bytes.NewBuffer(nil)

	if cwc.res.Request.Body != nil {
		defer cwc.res.Request.Body.Close()
		_, err := io.Copy(reqbody, cwc.res.Request.Body)
		if err != nil {
			return err
		}
	}

	now := time.Now()

	resp := &Response{
		ResponseMeta: ResponseMeta{
			Origin:        cwc.res.Request.RemoteAddr,
			Method:        cwc.res.Request.Method,
			Status:        cwc.res.StatusCode,
			ContentType:   http.DetectContentType(cwc.Bytes()),
			ContentLength: uint64(cwc.Len()),
			Host:          cwc.res.Request.URL.Host,
			URL:           cwc.res.Request.URL.String(),
			Scheme:        cwc.res.Request.URL.Scheme,
			Path:          cwc.res.Request.URL.Path,
			DateStart:     cwc.t,
			DateEnd:       now,
			TimeTaken:     now.UnixNano() - cwc.t.UnixNano(),
		},
		Header:        Header{cwc.res.Header},
		Body:          cwc.Bytes(),
		RequestHeader: Header{cwc.res.Request.Header},
		RequestBody:   reqbody.Bytes(),
	}

	cwc.resp <- resp

	return nil
}

type Capture struct {
	resp chan *Response
}

func New(resp chan *Response) *Capture {
	return &Capture{resp: resp}
}

func (c *Capture) NewWriteCloser(res *http.Response) (io.WriteCloser, error) {
	return &CaptureWriteCloser{
		res:  res,
		resp: c.resp,
		t:    time.Now(),
	}, nil
}
