package tui

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func NewEnvelope(typ string, payload any) (Envelope, error) {
	if typ == "" {
		return Envelope{}, errors.New("empty envelope type")
	}
	if payload == nil {
		return Envelope{Type: typ}, nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, errors.Wrap(err, "marshal envelope payload")
	}
	return Envelope{Type: typ, Payload: b}, nil
}

func (e Envelope) MarshalJSONBytes() ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, errors.Wrap(err, "marshal envelope")
	}
	return b, nil
}
