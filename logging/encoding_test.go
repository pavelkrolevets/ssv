package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

/*
type Encoder interface {
	ObjectEncoder

	// Clone copies the encoder, ensuring that adding fields to the copy doesn't
	// affect the original.
	Clone() Encoder

	// EncodeEntry encodes an entry and fields, along with any accumulated
	// context, into a byte buffer and returns it. Any fields that are empty,
	// including fields on the `Entry` type, should be omitted.
	EncodeEntry(Entry, []Field) (*buffer.Buffer, error)
}
*/

func TestTest(t *testing.T) {

	config := zap.NewDevelopmentConfig()
	config.Encoding = customEncoder

	logger, err := config.Build()
	require.NoError(t, err)
	logger = logger.Named("name1").Named("name2")

	logger.Info("msg", zap.String("key", "small value"))
	logger.Info("msg", zap.String("key", "value is longer than allowed"))

	// TEST OUTPUT:
	// {"L":"INFO","T":"2023-03-01T12:16:08.841+0200","C":"logging/context_test.go:64","M":"msg","key":"small value"}
	// {"L":"ERROR","T":"2023-03-01T12:16:08.842+0200","C":"logging/context_test.go:65","M":"msg","key":"value is longer than allowed","error":"log line longer than 20"}

	t.Fail() // fail test to print logging
}
