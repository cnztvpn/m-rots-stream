package stream

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	lowe "github.com/m-rots/bernard"
	"golang.org/x/time/rate"
)

type fetch struct {
	auth    lowe.Authenticator
	baseURL string
	client  *http.Client
	limiter *rate.Limiter
}

func NewFetch(auth lowe.Authenticator) fetch {
	const baseURL string = "https://www.googleapis.com/drive/v3"

	return fetch{
		auth:    auth,
		baseURL: baseURL,
		limiter: rate.NewLimiter(10, 1),
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				// An empty map for TLSNextProto disables HTTP/2.
				// Unfortunately, memory use and stability get hammered when using HTTP/2.
				TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			},
			Timeout: time.Minute * 5,
		},
	}
}

func (f fetch) Range(ctx context.Context, rw io.Writer, ID string, start uint64, end uint64) error {
	err := f.limiter.Wait(ctx)
	if err != nil {
		return err
	}

	token, _, err := f.auth.AccessToken()
	if err != nil {
		return err
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", f.baseURL+"/files/"+ID+"?alt=media", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	res, err := f.client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != 206 {
		switch res.StatusCode {
		case 403:
			return ErrRateLimit
		default:
			return errors.New("weird status code")
		}
	}

	buf := streamingBufPool.Get().([]byte)
	defer streamingBufPool.Put(buf)

	_, err = io.CopyBuffer(rw, res.Body, buf)
	return err
}

var (
	ErrRateLimit = errors.New("stream: rate limit")
)

var streamingBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}
