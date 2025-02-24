package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/tsdb/prometheus/models"
)

type MockDoer struct {
	Req *http.Request
}

func (doer *MockDoer) Do(req *http.Request) (*http.Response, error) {
	doer.Req = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
	}, nil
}

func TestClient(t *testing.T) {
	t.Run("QueryResource", func(t *testing.T) {
		doer := &MockDoer{}
		// The method here does not really matter for resource calls
		client := NewClient(doer, http.MethodGet, "http://localhost:9090")

		t.Run("sends correct POST request", func(t *testing.T) {
			req := &backend.CallResourceRequest{
				PluginContext: backend.PluginContext{},
				Path:          "/api/v1/series",
				Method:        http.MethodPost,
				URL:           "/api/v1/series",
				Body:          []byte("match%5B%5D: ALERTS\nstart: 1655271408\nend: 1655293008"),
			}
			res, err := client.QueryResource(context.Background(), req)
			defer func() {
				if res != nil && res.Body != nil {
					if err := res.Body.Close(); err != nil {
						fmt.Println("Error", "err", err)
					}
				}
			}()
			require.NoError(t, err)
			require.NotNil(t, doer.Req)
			require.Equal(t, http.MethodPost, doer.Req.Method)
			body, err := io.ReadAll(doer.Req.Body)
			require.NoError(t, err)
			require.Equal(t, []byte("match%5B%5D: ALERTS\nstart: 1655271408\nend: 1655293008"), body)
			require.Equal(t, "http://localhost:9090/api/v1/series", doer.Req.URL.String())
		})

		t.Run("unGzip response data", func(t *testing.T) {
			rawData := []byte("match%5B%5D: ALERTS\nstart: 1655271408\nend: 1655293008")
			buffer := bytes.NewBuffer(make([]byte, 0, 1024))
			gzipW := gzip.NewWriter(buffer)
			_, err := gzipW.Write(rawData)
			assert.Nil(t, err)
			err = gzipW.Close()
			assert.Nil(t, err)

			req := &backend.CallResourceRequest{
				PluginContext: backend.PluginContext{},
				Path:          "/api/v1/series",
				Method:        http.MethodPost,
				URL:           "/api/v1/series",
				Body:          buffer.Bytes(),
			}
			res, err := client.QueryResource(context.Background(), req)
			defer func() {
				if res != nil && res.Body != nil {
					if err := res.Body.Close(); err != nil {
						fmt.Println("Error", "err", err)
					}
				}
			}()
			require.NoError(t, err)
			require.NotNil(t, doer.Req)
			require.Equal(t, http.MethodPost, doer.Req.Method)
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, []byte("match%5B%5D: ALERTS\nstart: 1655271408\nend: 1655293008"), body)
			require.Equal(t, "http://localhost:9090/api/v1/series", doer.Req.URL.String())
		})

		t.Run("sends correct GET request", func(t *testing.T) {
			req := &backend.CallResourceRequest{
				PluginContext: backend.PluginContext{},
				Path:          "/api/v1/series",
				Method:        http.MethodGet,
				URL:           "api/v1/series?match%5B%5D=ALERTS&start=1655272558&end=1655294158",
			}
			res, err := client.QueryResource(context.Background(), req)
			defer func() {
				if res != nil && res.Body != nil {
					if err := res.Body.Close(); err != nil {
						fmt.Println("Error", "err", err)
					}
				}
			}()
			require.NoError(t, err)
			require.NotNil(t, doer.Req)
			require.Equal(t, http.MethodGet, doer.Req.Method)
			body, err := io.ReadAll(doer.Req.Body)
			require.NoError(t, err)
			require.Equal(t, []byte{}, body)
			require.Equal(t, "http://localhost:9090/api/v1/series?match%5B%5D=ALERTS&start=1655272558&end=1655294158", doer.Req.URL.String())
		})
	})

	t.Run("QueryRange", func(t *testing.T) {
		doer := &MockDoer{}

		t.Run("sends correct POST query", func(t *testing.T) {
			client := NewClient(doer, http.MethodPost, "http://localhost:9090")
			req := &models.Query{
				Expr:       "rate(ALERTS{job=\"test\" [$__rate_interval]})",
				Start:      time.Unix(0, 0),
				End:        time.Unix(1234, 0),
				RangeQuery: true,
				Step:       1 * time.Second,
			}
			res, err := client.QueryRange(context.Background(), req)
			defer func() {
				if res != nil && res.Body != nil {
					if err := res.Body.Close(); err != nil {
						fmt.Println("Error", "err", err)
					}
				}
			}()
			require.NoError(t, err)
			require.NotNil(t, doer.Req)
			require.Equal(t, http.MethodPost, doer.Req.Method)
			require.Equal(t, "application/x-www-form-urlencoded", doer.Req.Header.Get("Content-Type"))
			body, err := io.ReadAll(doer.Req.Body)
			require.NoError(t, err)
			require.Equal(t, []byte("end=1234&query=rate%28ALERTS%7Bjob%3D%22test%22+%5B%24__rate_interval%5D%7D%29&start=0&step=1"), body)
			require.Equal(t, "http://localhost:9090/api/v1/query_range", doer.Req.URL.String())
		})

		t.Run("sends correct GET query", func(t *testing.T) {
			client := NewClient(doer, http.MethodGet, "http://localhost:9090")
			req := &models.Query{
				Expr:       "rate(ALERTS{job=\"test\" [$__rate_interval]})",
				Start:      time.Unix(0, 0),
				End:        time.Unix(1234, 0),
				RangeQuery: true,
				Step:       1 * time.Second,
			}
			res, err := client.QueryRange(context.Background(), req)
			defer func() {
				if res != nil && res.Body != nil {
					if err := res.Body.Close(); err != nil {
						fmt.Println("Error", "err", err)
					}
				}
			}()
			require.NoError(t, err)
			require.NotNil(t, doer.Req)
			require.Equal(t, http.MethodGet, doer.Req.Method)
			body, err := io.ReadAll(doer.Req.Body)
			require.NoError(t, err)
			require.Equal(t, []byte{}, body)
			require.Equal(t, "http://localhost:9090/api/v1/query_range?end=1234&query=rate%28ALERTS%7Bjob%3D%22test%22+%5B%24__rate_interval%5D%7D%29&start=0&step=1", doer.Req.URL.String())
		})
	})
}
