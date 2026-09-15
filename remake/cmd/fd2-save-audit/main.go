// fd2-save-audit prints the proven current-runtime projection of a user-provided
// FD2.SAV. It never rewrites the input and deliberately leaves raw identities,
// classes, item flags and item IDs uninterpreted.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

type auditedRecord struct {
	Index    int                         `json:"index"`
	RuntimeX *int                        `json:"runtime_x,omitempty"`
	RuntimeY *int                        `json:"runtime_y,omitempty"`
	View     fdsave.PersistentRecordView `json:"view"`
}

type auditResult struct {
	InputSHA256 string                      `json:"input_sha256"`
	Header      fdsave.CurrentRuntimeHeader `json:"header"`
	Persistent  []auditedRecord             `json:"persistent"`
	Runtime     []auditedRecord             `json:"runtime"`
	Slots       []auditedSlot               `json:"slots"`
}

type auditedSlot struct {
	Slot     int                     `json:"slot"`
	Verified fdsave.VerifiedMetadata `json:"verified"`
	Records  []auditedRecord         `json:"records"`
}

func audit(path string) (auditResult, error) {
	stored, err := os.ReadFile(path)
	if err != nil {
		return auditResult{}, err
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		return auditResult{}, err
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		return auditResult{}, err
	}
	digest := sha256.Sum256(stored)
	result := auditResult{
		InputSHA256: hex.EncodeToString(digest[:]),
		Header:      snapshot.Header,
		Persistent:  make([]auditedRecord, int(snapshot.Header.PersistentCount)),
		Runtime:     make([]auditedRecord, len(snapshot.RuntimeRecords)),
	}
	for index := range result.Persistent {
		result.Persistent[index] = auditedRecord{Index: index, View: snapshot.PersistentRecords[index].View()}
	}
	for index, record := range snapshot.RuntimeRecords {
		x, y := int(record.Raw[0]), int(record.Raw[1])
		result.Runtime[index] = auditedRecord{
			Index: index, RuntimeX: &x, RuntimeY: &y, View: record.View(),
		}
	}
	for slot := 0; slot < fdsave.SlotCount; slot++ {
		chapter, slotErr := fdsave.InspectChapterSlot(plain, slot)
		if errors.Is(slotErr, fdsave.ErrEmptyChapterSlot) {
			continue
		}
		if slotErr != nil {
			return auditResult{}, slotErr
		}
		records := chapter.ActiveRecords()
		audited := auditedSlot{
			Slot: slot, Verified: chapter.Verified,
			Records: make([]auditedRecord, len(records)),
		}
		for index, record := range records {
			audited.Records[index] = auditedRecord{Index: index, View: record.View()}
		}
		result.Slots = append(result.Slots, audited)
	}
	return result, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "用法：fd2-save-audit FD2.SAV")
		os.Exit(2)
	}
	result, err := audit(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
