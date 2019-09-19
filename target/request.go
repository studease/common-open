package target

import (
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"

	basecfg "github.com/studease/common/utils/config"
)

// Request generates a new http request with the given arguments
func Request(cfg *basecfg.URL, rawquery string) (*http.Response, error) {
	var (
		srv    *basecfg.Server
		client http.Client
	)

	u, err := Parse(cfg.Path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(cfg.Method, u, nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = url.QueryEscape(rawquery)

	res, err := client.Do(req)
	if err != nil {
		if srv != nil {
			atomic.AddInt32(&srv.Failures, 1)
		}

		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("%s", res.Status)
	}

	return res, nil
}
