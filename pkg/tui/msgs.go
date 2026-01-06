package tui

type StateSnapshotMsg struct {
	Snapshot StateSnapshot
}

type EventLogAppendMsg struct {
	Entry EventLogEntry
}
