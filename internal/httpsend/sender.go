package httpsend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// defaultHTTPClient 默认 HTTP 客户端，设置超时
var defaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Send 发送 HTTP 请求并返回响应体
func Send(req *SubmitRequest) ([]byte, error) {
	bodyReader := bytes.NewReader(req.Body)

	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create http request failed: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := defaultHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http do failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %w", err)
	}

	return respBody, nil
}
