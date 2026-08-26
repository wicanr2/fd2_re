package campaign

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

const (
	nativeCharacterIdentityCount = 32
	nativeCharacterClassCount    = 29
	nativeCharacterEXESHA256     = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
)

// NativeCatalogEntry is one editable name attached to an exact native byte
// value. It deliberately contains no portrait or sprite aliases.
type NativeCatalogEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type NativeCharacterCatalogSource struct {
	ReferenceFile string `json:"reference_file"`
	EXESHA256     string `json:"exe_sha256"`
}

// NativeCharacterCatalog names the two independently sourced byte domains
// needed to present one persistent roster record.
type NativeCharacterCatalog struct {
	SchemaVersion int                          `json:"schema_version"`
	Source        NativeCharacterCatalogSource `json:"source"`
	Identities    []NativeCatalogEntry         `json:"identities"`
	Classes       []NativeCatalogEntry         `json:"classes"`

	identityNames [nativeCharacterIdentityCount]string
	classNames    [nativeCharacterClassCount]string
}

// LoadNativeCharacterCatalog requires complete, ordered native domains. A
// missing, duplicate, reordered, or blank row fails closed.
func LoadNativeCharacterCatalog(path string) (*NativeCharacterCatalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("native character catalog: %w", err)
	}
	var catalog NativeCharacterCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, fmt.Errorf("native character catalog: %w", err)
	}
	if err := catalog.validate(); err != nil {
		return nil, err
	}
	return &catalog, nil
}

func (catalog *NativeCharacterCatalog) validate() error {
	if catalog == nil {
		return fmt.Errorf("native character catalog: missing catalog")
	}
	if catalog.SchemaVersion != 1 {
		return fmt.Errorf(
			"native character catalog: schema version=%d, want 1",
			catalog.SchemaVersion,
		)
	}
	if catalog.Source.ReferenceFile != "docs/data/fd2-reference-files.json" ||
		catalog.Source.EXESHA256 != nativeCharacterEXESHA256 {
		return fmt.Errorf(
			"native character catalog: source does not match reference FD2.EXE",
		)
	}
	if len(catalog.Identities) != nativeCharacterIdentityCount {
		return fmt.Errorf(
			"native character catalog: identities=%d, want %d",
			len(catalog.Identities), nativeCharacterIdentityCount,
		)
	}
	if len(catalog.Classes) != nativeCharacterClassCount {
		return fmt.Errorf(
			"native character catalog: classes=%d, want %d",
			len(catalog.Classes), nativeCharacterClassCount,
		)
	}
	for index, entry := range catalog.Identities {
		if entry.ID != index || entry.Name == "" {
			return fmt.Errorf(
				"native character catalog: identity row %d has id=%d name=%q",
				index, entry.ID, entry.Name,
			)
		}
		catalog.identityNames[index] = entry.Name
	}
	for index, entry := range catalog.Classes {
		if entry.ID != index || entry.Name == "" {
			return fmt.Errorf(
				"native character catalog: class row %d has id=%d name=%q",
				index, entry.ID, entry.Name,
			)
		}
		catalog.classNames[index] = entry.Name
	}
	return nil
}

// MaterializeNativePersistentPartyRecord projects only proven record fields
// into the existing party Unit. Raw +7 is the player-persistent key passed to
// 0x11019, so it is preserved as MapSelectorKey; it is not inferred to be a
// portrait, Fig, or character identity. The projection does not assign those
// fields, position, deployment state, spells, attack range, or chapter node.
func MaterializeNativePersistentPartyRecord(
	record fdsave.PersistentRecord,
	catalog *NativeCharacterCatalog,
) (battle.Unit, error) {
	if err := catalog.validate(); err != nil {
		return battle.Unit{}, err
	}
	view := record.View()
	identity := int(view.RawIdentity)
	classID := int(view.Class)
	if identity >= len(catalog.identityNames) {
		return battle.Unit{}, fmt.Errorf(
			"native persistent party: identity %d outside catalog", identity,
		)
	}
	if classID >= len(catalog.classNames) {
		return battle.Unit{}, fmt.Errorf(
			"native persistent party: class %d outside catalog", classID,
		)
	}

	unit := battle.Unit{
		Camp:                  battle.Own,
		Name:                  catalog.identityNames[identity],
		ClsName:               catalog.classNames[classID],
		ClassID:               classID,
		Lv:                    int(view.Level),
		HP:                    int(view.HP),
		MaxHP:                 int(view.MaxHP),
		MP:                    int(view.MP),
		MaxMP:                 int(view.MaxMP),
		AP:                    int(view.AP),
		DP:                    int(view.DP),
		HIT:                   int(view.HIT),
		EV:                    int(view.EV),
		MV:                    int(view.Movement),
		NativeIdentity:        identity,
		HasNativeIdentity:     true,
		NativeRecordByte8:     view.RawIdentity,
		HasNativeRecordByte8:  true,
		MapSelectorKey:        int(view.RawPresentationKey),
		HasMapSelectorKey:     true,
		NativeRecordRace:      view.Race,
		HasNativeRecordRace:   true,
		NativeRecordClass:     view.Class,
		HasNativeRecordClass:  true,
		NativeRecordWord42:    uint16(view.MaxHP),
		HasNativeRecordWord42: true,
		NativeRecordWord46:    uint16(view.MaxMP),
		HasNativeRecordWord46: true,
		NativeCommandMask:     view.CommandMask,
		NativeTransient:       view.Transient,
		NativeRecordByte5:     view.RawByte5,
		HasNativeRecordByte5:  true,
		NativeRecordByte6:     view.RawCamp,
		HasNativeRecordByte6:  true,
		NativeRecordByte34:    view.RawByte34,
		HasNativeRecordByte34: true,
		NativeRecordByte35:    view.RawByte35,
		HasNativeRecordByte35: true,
		NativeRecordByte36:    view.RawByte36,
		HasNativeRecordByte36: true,
		BaseAP:                int(view.BaseAP),
		BaseDP:                int(view.BaseDP),
		BaseHIT:               int(view.DX),
		BaseEV:                int(view.DX),
		BaseMV:                int(view.Movement),
		EquipmentBaseSet:      true,
		DX:                    int(view.DX),
		Exp:                   float64(view.Experience),
		InventorySlots:        make([]int, len(view.Inventory)),
		NativeInventoryFlags:  make([]int, len(view.Inventory)),
	}
	for index, cell := range view.Inventory {
		unit.InventorySlots[index] = int(cell.ItemID)
		unit.NativeInventoryFlags[index] = int(cell.Flags)
		if int8(cell.Flags) < 0 {
			continue
		}
		unit.Inventory = append(unit.Inventory, int(cell.ItemID))
		unit.Equipped = append(unit.Equipped, cell.Flags&0x40 != 0)
	}
	return unit, nil
}
