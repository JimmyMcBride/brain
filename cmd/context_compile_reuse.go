package cmd

import "brain/internal/session"

func latestMatchingPacketRecord(records []session.PacketRecord, fingerprint string) *session.PacketRecord {
	if fingerprint == "" {
		return nil
	}
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Fingerprint == fingerprint {
			return &records[i]
		}
	}
	return nil
}

func latestTaskPacketRecord(records []session.PacketRecord, taskText string) *session.PacketRecord {
	if taskText == "" {
		return nil
	}
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].TaskText == taskText {
			return &records[i]
		}
	}
	return nil
}
