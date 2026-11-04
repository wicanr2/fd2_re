package battle

import (
	"encoding/binary"
	"fmt"
)

// NativeItemRelocationRoute closes the post-confirm type-23 item branch.
// 0x2218a reuses command 23's MP cost, retains the item, and relocates only
// the first target through the shared 0x22253 presentation.
type NativeItemRelocationRoute struct {
	ItemType         byte
	RowWord          uint16
	CommandID        int
	RequiredIdentity byte
	RequiredMaxMP    uint16
	Presentation     uint32
	ConsumesSource   bool
}

type NativeItemRelocationResult struct {
	TargetUnit       int
	FromX, FromY     byte
	DestinationX     byte
	DestinationY     byte
	RemainingMP      uint16
	AccumulatorDelta int
}

func NativeItemRelocationRouteForType(itemType byte, rowWord uint16) (NativeItemRelocationRoute, bool) {
	if itemType != 23 {
		return NativeItemRelocationRoute{}, false
	}
	return NativeItemRelocationRoute{
		ItemType: itemType, RowWord: rowWord, CommandID: 23,
		RequiredIdentity: 24, RequiredMaxMP: 20, Presentation: 0x22253,
		ConsumesSource: false,
	}, true
}

// ApplyNativeItemRelocation preserves the mutation portion after 0x1bbdc has
// confirmed the type-23 destination. Selector mode-6 terrain/occupancy checks
// and the 27-present indexed renderer remain caller-owned.
func ApplyNativeItemRelocation(
	records []byte,
	actorUnit int,
	targets []byte,
	destinationX, destinationY byte,
	route NativeItemRelocationRoute,
	commandBook []NativeCommandRecord,
) (NativeItemRelocationResult, error) {
	verified, ok := NativeItemRelocationRouteForType(route.ItemType, route.RowWord)
	if !ok || verified != route {
		return NativeItemRelocationResult{}, fmt.Errorf("native item relocation route is invalid: %#v", route)
	}
	actorBase := actorUnit * nativeRecordSize
	if actorUnit < 0 || actorBase < 0 || actorBase+nativeRecordSize > len(records) {
		return NativeItemRelocationResult{}, recordBoundsError(actorUnit)
	}
	actor := records[actorBase : actorBase+nativeRecordSize]
	if actor[0x08] != route.RequiredIdentity || binary.LittleEndian.Uint16(actor[0x46:]) < route.RequiredMaxMP {
		return NativeItemRelocationResult{}, fmt.Errorf("native item relocation identity/max MP gate failed")
	}
	return applyNativeCommand23Relocation(records, actorUnit, targets, destinationX, destinationY, commandBook, false)
}

// ApplyNativeCommand23Relocation is the spell dispatch to sub_2218A.
// Item-only identity/max-MP guards belong to the item caller, not this handler.
// The spell caller has already checked current MP; preserve that atomic gate.
func ApplyNativeCommand23Relocation(records []byte, actorUnit int, targets []byte, destinationX, destinationY byte, commandBook []NativeCommandRecord) (NativeItemRelocationResult, error) {
	return applyNativeCommand23Relocation(records, actorUnit, targets, destinationX, destinationY, commandBook, true)
}

func applyNativeCommand23Relocation(records []byte, actorUnit int, targets []byte, destinationX, destinationY byte, commandBook []NativeCommandRecord, requireMP bool) (NativeItemRelocationResult, error) {
	if len(targets) == 0 {
		return NativeItemRelocationResult{}, fmt.Errorf("native item relocation has no target")
	}
	targetUnit := int(targets[0])
	actorBase := actorUnit * nativeRecordSize
	targetBase := targetUnit * nativeRecordSize
	if actorUnit < 0 || actorBase < 0 || actorBase+nativeRecordSize > len(records) {
		return NativeItemRelocationResult{}, recordBoundsError(actorUnit)
	}
	if targetBase < 0 || targetBase+nativeRecordSize > len(records) {
		return NativeItemRelocationResult{}, recordBoundsError(targetUnit)
	}
	if len(commandBook) != NativeCommandRecordCount || commandBook[23].ID != 23 {
		return NativeItemRelocationResult{}, fmt.Errorf("native command record23 is unavailable")
	}
	actor := records[actorBase : actorBase+nativeRecordSize]
	cost := commandBook[23].MPCost
	if cost < 0 || cost > 0xff {
		return NativeItemRelocationResult{}, fmt.Errorf("native command23 MP cost=%d is invalid", cost)
	}
	if requireMP && int(binary.LittleEndian.Uint16(actor[0x44:])) < cost {
		return NativeItemRelocationResult{}, fmt.Errorf("native command23 insufficient MP")
	}

	target := records[targetBase : targetBase+nativeRecordSize]
	remainingMP := binary.LittleEndian.Uint16(actor[0x44:]) - uint16(cost)
	effectiveLevel := int(target[0x21])
	if classID := target[0x20]; classID > 8 && classID < 25 {
		effectiveLevel += 30
	}
	result := NativeItemRelocationResult{
		TargetUnit: targetUnit, FromX: target[0], FromY: target[1],
		DestinationX: destinationX, DestinationY: destinationY,
		RemainingMP: remainingMP, AccumulatorDelta: effectiveLevel * 10,
	}
	binary.LittleEndian.PutUint16(actor[0x44:], remainingMP)
	target[0], target[1] = destinationX, destinationY
	return result, nil
}
