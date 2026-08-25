package ending

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

const (
	endingDialogueStride         = 320
	endingDialogueBytes          = 320 * 200
	endingDialoguePortraitAnchor = 0x9017
	endingDialogueTextOffset     = 0x9514
	endingDialogueLineStep       = 19 * endingDialogueStride
	endingDialogueGlyphStep      = 16
	endingDialogueMaximumGlyphs  = 13
)

// ComposeNativeDialogueBase 建立 sub_1956B 的穩定19×5格網與下框DATO頭像。
// 0x9017是每列最右端，必須使用原版right-to-left寫入方向。
func ComposeNativeDialogueBase(background []byte, cells []fdother.RawCell, portrait dato.Frame) ([]byte, error) {
	if len(background) != endingDialogueBytes || len(cells) <= 17 {
		return nil, errors.New("ending: native dialogue base assets are invalid")
	}
	frame := append([]byte(nil), background...)
	placements, err := fdother.PlanNativeDialogueFrameGrid(endingDialogueStride, 5, 112, 19, 5)
	if err != nil {
		return nil, err
	}
	for _, placement := range placements {
		if err := cells[placement.ResourceIndex].BlitOpaqueAtOffset(frame, endingDialogueStride, placement.DestinationByte); err != nil {
			return nil, err
		}
	}
	if err := portrait.BlitRightToLeftAtOffset(frame, endingDialogueStride, endingDialoguePortraitAnchor); err != nil {
		return nil, err
	}
	return frame, nil
}

// ComposeNativeDialogueOpeningFrames 重用已證實的 sub_1974C 六段310px crop。
func ComposeNativeDialogueOpeningFrames(background, base []byte) ([][]byte, error) {
	return campaign.NativeClassListOpeningFrames(background, base)
}

// ComposeNativeDialogueProgressiveFrames 依 sub_15F84 的目的位址、色碼與
// ordinary-glyph發布順序，建立單一typed page。base須已含該句speaker frame0。
func ComposeNativeDialogueProgressiveFrames(base []byte, font *fdtxt.Font, glyphs map[string]int, pages [][]string, page int) ([][]byte, error) {
	if len(base) != endingDialogueBytes || font == nil || glyphs == nil || page < 0 || page >= len(pages) {
		return nil, errors.New("ending: native dialogue progressive assets or page are invalid")
	}
	if len(pages[page]) == 0 || len(pages[page]) > 4 {
		return nil, fmt.Errorf("ending: native dialogue page %d has %d rows", page, len(pages[page]))
	}
	frame := append([]byte(nil), base...)
	frames := make([][]byte, 0, 1+endingDialogueMaximumGlyphs*len(pages[page]))
	frames = append(frames, append([]byte(nil), frame...))
	style := fdtxt.NativeGlyphStyle{Foreground: 0xcd, Shadow: 0x4c, Background: 0x4a}
	for row, text := range pages[page] {
		runes := []rune(text)
		if len(runes) == 0 || len(runes) > endingDialogueMaximumGlyphs {
			return nil, fmt.Errorf("ending: native dialogue row %d has %d glyphs", row, len(runes))
		}
		for column, r := range runes {
			glyph, ok := glyphs[string(r)]
			if !ok || glyph < 0 || glyph >= font.GlyphCount() {
				return nil, fmt.Errorf("ending: native dialogue glyph %q is unavailable", string(r))
			}
			destination := endingDialogueTextOffset + row*endingDialogueLineStep + column*endingDialogueGlyphStep
			if err := font.BlitNativeGlyph(frame, endingDialogueStride, destination, glyph, style); err != nil {
				return nil, err
			}
			frames = append(frames, append([]byte(nil), frame...))
		}
	}
	return frames, nil
}

// ComposeNativeDialogueMouthFrame 只替換下框頭像；caller仍負責等待輸入時序。
func ComposeNativeDialogueMouthFrame(stable []byte, portrait dato.Frame) ([]byte, error) {
	if len(stable) != endingDialogueBytes {
		return nil, errors.New("ending: native dialogue mouth base is invalid")
	}
	frame := append([]byte(nil), stable...)
	if err := portrait.BlitRightToLeftAtOffset(frame, endingDialogueStride, endingDialoguePortraitAnchor); err != nil {
		return nil, err
	}
	return frame, nil
}

// ComposeNativeDialogueClosingFrames 重播 sub_2D31B 的五段crop並加入caller source restore。
func ComposeNativeDialogueClosingFrames(background, base []byte) ([][]byte, error) {
	frames, err := campaign.NativeClassListClosingFrames(background, base)
	if err != nil {
		return nil, err
	}
	frames = append(frames, append([]byte(nil), background...))
	return frames, nil
}

type NativeDialogueUtteranceFrames struct {
	Pages     [][][]byte
	MouthOpen [][]byte
}

type NativeDialogueBlockFrames struct {
	Opening    [][]byte
	Utterances []NativeDialogueUtteranceFrames
	Closing    [][]byte
}

type NativeDialoguePhase uint8

const (
	NativeDialogueOpening NativeDialoguePhase = iota
	NativeDialogueProgressive
	NativeDialogueWaiting
	NativeDialogueClosing
	NativeDialogueComplete
)

func (p NativeDialoguePhase) String() string {
	switch p {
	case NativeDialogueOpening:
		return "opening"
	case NativeDialogueProgressive:
		return "progressive"
	case NativeDialogueWaiting:
		return "waiting"
	case NativeDialogueClosing:
		return "closing"
	case NativeDialogueComplete:
		return "complete"
	default:
		return "unknown"
	}
}

// NativeDialoguePlayback 只管理已完整預建的indexed frames；輸入映射與時鐘由caller擁有。
type NativeDialoguePlayback struct {
	Blocks                 []NativeDialogueBlockFrames
	Block, Utterance, Page int
	Frame                  int
	Phase                  NativeDialoguePhase
}

func NewNativeDialoguePlayback(blocks []NativeDialogueBlockFrames) (*NativeDialoguePlayback, error) {
	if len(blocks) == 0 {
		return nil, errors.New("ending: native dialogue playback has no blocks")
	}
	for blockIndex, block := range blocks {
		if len(block.Opening) != 6 || len(block.Closing) != 6 || len(block.Utterances) == 0 {
			return nil, fmt.Errorf("ending: native dialogue block %d has incomplete lifecycle frames", blockIndex)
		}
		for utteranceIndex, utterance := range block.Utterances {
			if len(utterance.Pages) == 0 || len(utterance.MouthOpen) != len(utterance.Pages) {
				return nil, fmt.Errorf("ending: native dialogue block %d utterance %d has incomplete pages", blockIndex, utteranceIndex)
			}
			for page, frames := range utterance.Pages {
				if len(frames) == 0 || len(utterance.MouthOpen[page]) != endingDialogueBytes {
					return nil, fmt.Errorf("ending: native dialogue block %d utterance %d page %d is incomplete", blockIndex, utteranceIndex, page)
				}
				for _, frame := range frames {
					if len(frame) != endingDialogueBytes {
						return nil, fmt.Errorf("ending: native dialogue block %d utterance %d page %d has invalid frame", blockIndex, utteranceIndex, page)
					}
				}
			}
		}
		for _, frame := range append(append([][]byte(nil), block.Opening...), block.Closing...) {
			if len(frame) != endingDialogueBytes {
				return nil, fmt.Errorf("ending: native dialogue block %d has invalid opening or closing frame", blockIndex)
			}
		}
	}
	return &NativeDialoguePlayback{Blocks: blocks, Phase: NativeDialogueOpening}, nil
}

func (p *NativeDialoguePlayback) CurrentFrame(mouthOpen bool) []byte {
	if p == nil || p.Phase == NativeDialogueComplete || p.Block < 0 || p.Block >= len(p.Blocks) {
		return nil
	}
	block := &p.Blocks[p.Block]
	switch p.Phase {
	case NativeDialogueOpening:
		return block.Opening[p.Frame]
	case NativeDialogueProgressive:
		return block.Utterances[p.Utterance].Pages[p.Page][p.Frame]
	case NativeDialogueWaiting:
		if mouthOpen {
			return block.Utterances[p.Utterance].MouthOpen[p.Page]
		}
		frames := block.Utterances[p.Utterance].Pages[p.Page]
		return frames[len(frames)-1]
	case NativeDialogueClosing:
		return block.Closing[p.Frame]
	default:
		return nil
	}
}

func (p *NativeDialoguePlayback) Waiting() bool {
	return p != nil && p.Phase == NativeDialogueWaiting
}

func (p *NativeDialoguePlayback) Done() bool {
	return p == nil || p.Phase == NativeDialogueComplete
}

// Step 發布下一張非互動畫面；等待輸入時保持穩定頁。
func (p *NativeDialoguePlayback) Step() {
	if p == nil || p.Done() || p.Waiting() {
		return
	}
	block := &p.Blocks[p.Block]
	switch p.Phase {
	case NativeDialogueOpening:
		if p.Frame+1 < len(block.Opening) {
			p.Frame++
			return
		}
		p.Phase, p.Frame = NativeDialogueProgressive, 0
	case NativeDialogueProgressive:
		frames := block.Utterances[p.Utterance].Pages[p.Page]
		if p.Frame+1 < len(frames) {
			p.Frame++
			return
		}
		p.Phase, p.Frame = NativeDialogueWaiting, 0
	case NativeDialogueClosing:
		if p.Frame+1 < len(block.Closing) {
			p.Frame++
			return
		}
		p.Block++
		p.Utterance, p.Page, p.Frame = 0, 0, 0
		if p.Block >= len(p.Blocks) {
			p.Phase = NativeDialogueComplete
		} else {
			p.Phase = NativeDialogueOpening
		}
	}
}

// Confirm 只在完整頁等待期接受輸入，依序換頁、換句或開始收框。
func (p *NativeDialoguePlayback) Confirm() bool {
	if !p.Waiting() {
		return false
	}
	block := &p.Blocks[p.Block]
	utterance := &block.Utterances[p.Utterance]
	if p.Page+1 < len(utterance.Pages) {
		p.Page++
		p.Frame, p.Phase = 0, NativeDialogueProgressive
		return true
	}
	if p.Utterance+1 < len(block.Utterances) {
		p.Utterance++
		p.Page, p.Frame, p.Phase = 0, 0, NativeDialogueProgressive
		return true
	}
	p.Frame, p.Phase = 0, NativeDialogueClosing
	return true
}
