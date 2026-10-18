package link

import (
	"errors"
	"io"

	"github.com/hujm2023/go-sms-protocol/codec"
)

// connReader 包装 io.Reader 实现 codec.ConnReader 接口。
// MyCodec.Decode 使用 DecodeBlocked（仅依赖 io.ReadFull），
// 因此 Peek/Discard/Size 为占位实现，不会被调用。
type connReader struct {
	r io.Reader
}

var _ codec.ConnReader = (*connReader)(nil)

func newConnReader(r io.Reader) *connReader {
	return &connReader{r: r}
}

func (c *connReader) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func (c *connReader) Peek(n int) ([]byte, error) {
	return nil, errors.New("Peek not supported on blocking connReader")
}

func (c *connReader) Discard(n int) (int, error) {
	return 0, errors.New("Discard not supported on blocking connReader")
}

func (c *connReader) Size() int {
	return 0
}
