package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// Derive descriptors from the native public data, not from HD raster objects.
func rowLabelFrameDescriptors(d *nativeUnitStatusData, commands bool) ([]map[string]any, error) {
	fields := []map[string]any{}
	if d == nil {
		return fields, nil
	}
	if commands {
		for i, id := range d.commandIDs {
			x, y := 68+100*(i/4), 108+22*(i%4)
			fields = append(fields, map[string]any{"source": 92, "text": "-MP", "rect": []int{x, y, x + 20, y + 8}, "command_id": id})
		}
		return fields, nil
	}
	cells, err := battle.NativeItemSelectorCells(d.record, 0, -1, d.itemRows)
	if err != nil {
		return nil, err
	}
	for _, c := range cells {
		if !c.HasStatValue {
			continue
		}
		x, y := c.LabelX+68, c.LabelY+4
		fields = append(fields, map[string]any{"source": c.StatIcon, "text": map[int]string{64: "+AP", 65: "+DP", 66: "+HP", 67: "+MP"}[c.StatIcon], "rect": []int{x, y, x + 21, y + 8}, "raw_slot": c.RawSlot, "item_id": int(c.ItemID)})
	}
	return fields, nil
}

func (p *itemPanelStatusGPUProbe) drawRowLabelBank(screen *ebiten.Image) {
	index := p.bankIndex
	commands := index == 1
	name := []string{"bank-items", "bank-commands", "bank-empty"}[index]
	data := p.banks[index]
	for _, g := range []*Game{p.g, p.control} {
		d := *data
		d.itemChrome, d.itemValues, d.rowLabels = nil, nil, nil
		status, ok := d.compose(d.assets, false)
		if !ok {
			p.err = fmt.Errorf("bank native item composition unavailable")
			return
		}
		cmds, ok := d.compose(d.assets, true)
		if !ok {
			p.err = fmt.Errorf("bank native command composition unavailable")
			return
		}
		panel, phase := status, "status"
		if commands {
			panel, phase = cmds, nativePlayerStatusCommands
		}
		source := append([]byte(nil), g.nativeMapVGA...)
		steady, err := nativeChurchPanelFrame(source, panel, 0)
		if err != nil {
			p.err = err
			return
		}
		g.nativePlayerStatus = &nativePlayerStatusState{data: &d, source: source, status: status, commands: cmds, steady: steady, phase: phase}
	}
	p.g.Draw(screen)
	p.control.Draw(p.controlScreen)
	if p.g.loadErr != "" || p.control.loadErr != "" || itemPanelStatusGPUState(p.g) != itemPanelStatusGPUState(p.control) {
		p.err = fmt.Errorf("bank native control differs: %s %s", p.g.loadErr, p.control.loadErr)
		return
	}
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	presentation := &modernHDPresentation{Game: p.g}
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(p.pixels)
	digest := sha256.Sum256(p.pixels)
	state := itemPanelStatusGPUState(p.g)
	save := func(file string) error {
		picture := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(picture.Pix, p.pixels)
		f, err := os.Create(filepath.Join(p.out, file))
		if err != nil {
			return err
		}
		err = png.Encode(f, picture)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		return closeErr
	}
	// Explicit HD control for the same typed bank: suppress only the new
	// display patch, retaining its seal, native data and all other HD layers.
	owner := p.g.nativePlayerStatus.data
	prior := owner.rowLabels
	if prior == nil {
		p.err = fmt.Errorf("bank HD row owner unavailable")
		return
	}
	without := *prior
	without.items, without.commands = nil, nil
	owner.rowLabels = &without
	p.g.Draw(screen)
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(p.pixels)
	beforeDigest := sha256.Sum256(p.pixels)
	beforeFile := name + "-before.png"
	err := save(beforeFile)
	owner.rowLabels = prior
	if err != nil || state != itemPanelStatusGPUState(p.g) {
		p.err = fmt.Errorf("bank control changed state or PNG failed: %v", err)
		return
	}
	if err := p.g.applyTheme(nil); err != nil {
		p.err = err
		return
	}
	p.g.Draw(screen)
	if p.g.modernHDText != nil || p.g.modernHDOverlay != nil {
		p.err = fmt.Errorf("bank pixel mode retains HD")
		return
	}
	if err := p.g.applyTheme(p.set); err != nil {
		p.err = err
		return
	}
	p.g.Draw(screen)
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(p.pixels)
	if sha256.Sum256(p.pixels) != digest || state != itemPanelStatusGPUState(p.g) {
		p.err = fmt.Errorf("bank F2 changed pixels or state")
		return
	}
	fields, err := rowLabelFrameDescriptors(data, commands)
	if err != nil {
		p.err = err
		return
	}
	file := name + ".png"
	if err := save(file); err != nil {
		p.err = err
		return
	}
	regions, _ := nativeChurchPanelRegions(0)
	p.bankRows = append(p.bankRows, map[string]any{"phase": name, "capture": file, "before_capture": beforeFile, "before_sha256": fmt.Sprintf("%x", beforeDigest), "row_label_commands": commands, "regions": regions, "row_label_fields": fields, "row_label_record": fmt.Sprintf("%x", data.record), "row_label_command_ids": data.commandIDs, "f2_roundtrip": true, "native_control_unchanged": true, "final_sha256": fmt.Sprintf("%x", digest)})
	p.g.nativePlayerStatus, p.control.nativePlayerStatus = nil, nil
	p.bankIndex++
}

func addRowLabelFrameMetadata(row map[string]any, data *nativeUnitStatusData, commands bool) error {
	fields, err := rowLabelFrameDescriptors(data, commands)
	if err != nil {
		return err
	}
	row["row_label_fields"] = fields
	row["row_label_commands"] = commands
	if data != nil {
		row["row_label_record"] = fmt.Sprintf("%x", data.record)
		row["row_label_command_ids"] = data.commandIDs
	}
	return nil
}
