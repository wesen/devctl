package tui

import (
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/message"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/errors"
)

func RegisterUIForwarder(bus *Bus, p *tea.Program) {
	bus.AddHandler("devctl-ui-forward", TopicUIMessages, func(msg *message.Message) error {
		defer msg.Ack()

		var env Envelope
		if err := json.Unmarshal(msg.Payload, &env); err != nil {
			return errors.Wrap(err, "unmarshal ui envelope")
		}

		switch env.Type {
		case UITypeStateSnapshot:
			var snap StateSnapshot
			if err := json.Unmarshal(env.Payload, &snap); err != nil {
				return errors.Wrap(err, "unmarshal snapshot payload")
			}
			p.Send(StateSnapshotMsg{Snapshot: snap})
		case UITypeEventAppend:
			var entry EventLogEntry
			if err := json.Unmarshal(env.Payload, &entry); err != nil {
				return errors.Wrap(err, "unmarshal event payload")
			}
			p.Send(EventLogAppendMsg{Entry: entry})
		}
		return nil
	})
}
