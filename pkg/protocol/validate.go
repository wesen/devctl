package protocol

import "github.com/pkg/errors"

func ValidateHandshake(h Handshake) error {
	if h.Type != FrameHandshake {
		return errors.Errorf("%s: expected handshake frame, got %q", ErrProtocolInvalidHandshake, h.Type)
	}
	if h.ProtocolVersion != ProtocolV1 {
		return errors.Errorf("%s: unsupported protocol_version %q", ErrProtocolInvalidHandshake, h.ProtocolVersion)
	}
	if h.PluginName == "" {
		return errors.Errorf("%s: missing plugin_name", ErrProtocolInvalidHandshake)
	}
	return nil
}
