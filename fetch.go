package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	lowe "github.com/m-rots/bernard"
)

type fetch struct {
	auth    lowe.Authenticator
	baseURL string
	client  *http.Client
}

func (f *fetch) Range(ctx context.Context, rw io.Writer, ID string, start uint64, end uint64) error {
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
