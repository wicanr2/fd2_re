package battle

import (
	"fmt"
)

// NativeItemCommandDamageRoute closes item types 20/21/24. The item row word
// is reused as a 0x4e516 command ID by 0x1c75e; 20/24 use the ten-frame
// 0x1cd17 presentation while 21 reaches 0x1cac7 through 0x2111a. None removes
// the source slot.
type NativeItemCommandDamageRoute struct {
	ItemType       byte
	CommandID      int
	Presentation   uint32
	ConsumesSource bool
}

func NativeItemCommandDamageRouteForType(itemType byte, rowWord uint16) (NativeItemCommandDamageRoute, bool) {
	if int(rowWord) >= NativeCommandRecordCount {
		return NativeItemCommandDamageRoute{}, false
	}
	var presentation uint32
	switch itemType {
	case 20, 24:
		presentation = 0x1cd17
	case 21:
		presentation = 0x1cac7
	default:
		return NativeItemCommandDamageRoute{}, false
	}
	return NativeItemCommandDamageRoute{
		ItemType: itemType, CommandID: int(rowWord), Presentation: presentation,
		ConsumesSource: false,
	}, true
}

// ApplyNativeItemCommandDamage preserves the shared post-presentation target
// loop. It intentionally performs no 0x1ca89 MP debit and no inventory
// removal: neither call exists in these dispatcher branches.
func ApplyNativeItemCommandDamage(
	targets []*Unit,
	route NativeItemCommandDamageRoute,
	commandBook []NativeCommandRecord,
	resistances map[int]int,
	rngState uint16,
) ([]NativeCommandDamage, uint16, error) {
	verified, ok := NativeItemCommandDamageRouteForType(route.ItemType, uint16(route.CommandID))
	if !ok || verified != route {
		return nil, rngState, fmt.Errorf("native item command-damage route is invalid: %#v", route)
	}
	if route.CommandID >= len(commandBook) || commandBook[route.CommandID].ID != route.CommandID {
		return nil, rngState, fmt.Errorf("native command record %d is unavailable", route.CommandID)
	}
	record := commandBook[route.CommandID]
	if record.Damage < 0 || record.Hit < 0 || record.Hit > 100 {
		return nil, rngState, fmt.Errorf("native command record %d has invalid damage/hit", route.CommandID)
	}
	for i, target := range targets {
		if target == nil {
			return nil, rngState, fmt.Errorf("nil native item target %d", i)
		}
		resistance, exists := resistances[nativeCommandResistanceClass(target)]
		if !exists || resistance < 0 || resistance > 10 {
			return nil, rngState, fmt.Errorf("native resistance class %d is unavailable", nativeCommandResistanceClass(target))
		}
	}

	results := make([]NativeCommandDamage, 0, len(targets))
	for _, target := range targets {
		result, nextRNG, err := ApplyNativeCommandDamage(
			target, record.Damage, record.Hit, resistances[nativeCommandResistanceClass(target)], rngState,
		)
		if err != nil {
			return nil, rngState, err
		}
		rngState = nextRNG
		results = append(results, result)
	}
	return results, rngState, nil
}
