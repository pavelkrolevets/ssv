package logging

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const (
	maxLogLength  = 20
	customEncoder = "custom-encoder"
)

// TODO: we should implement this change after getting rid of logex the use of the .Build() function all of the place.
// The problem with Build is that it allows different logging encoding so the program can be using different loggers
// with different encoding and it is a mess

type LengthErrorEncoder struct {
	// both encoders are actually the same encoder
	zapcore.Encoder                 // only here for method inheritance
	childEncoder    zapcore.Encoder // only used to wrap the EncodeEntry method
}

func init() {
	err := zap.RegisterEncoder(customEncoder, func(config zapcore.EncoderConfig) (zapcore.Encoder, error) {
		enc := zapcore.NewJSONEncoder(config)
		return LengthErrorEncoder{enc, enc}, nil
	})
	if err != nil {
		panic(err)
	}
}

func (l LengthErrorEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	for i := range fields {
		if len(fields[i].String) > maxLogLength {
			entry.Level = zap.ErrorLevel
			fields = append(fields, zap.Error(fmt.Errorf("log line longer than %v", maxLogLength)))
		}
	}
	return l.childEncoder.EncodeEntry(entry, fields)
}
