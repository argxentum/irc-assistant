package marshaling

import (
	"assistant/pkg/log"
	"bytes"
	"encoding/json"
	"io"
)

func Unmarshal[T any](body io.ReadCloser, destination *T) error {
	logger := log.Logger()

	buf, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	if err := json.NewDecoder(bytes.NewReader(buf)).Decode(destination); err != nil {
		logger.Warningf(&marshalingLabeler{response: string(buf)}, "error decoding json response, %s", err)
		return err
	}

	return nil
}

type marshalingLabeler struct {
	response string
}

func (m *marshalingLabeler) Labels() map[string]string {
	return map[string]string{
		"response": m.response,
	}
}
