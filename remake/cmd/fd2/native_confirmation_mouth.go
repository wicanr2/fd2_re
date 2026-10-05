package main

import (
	"math/rand"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

// Only confirmed YESNO waiting owners enter sub_19953's mouth loop.
// Church and arbitrary-key messages retain their independent owners.
func (g *Game) nativeConfirmationMouthWaitingOwner() any {
	if g.nativeClassUIJob != nil {
		return nil
	}
	if g.nativeSystemEndTurnConfirm && g.nativeSystemEndTurnUI != nil {
		return g.nativeSystemEndTurnUI
	}
	if g.camp == nil || g.prepRequiredMissing != nil {
		return nil
	}
	node := g.camp.Node()
	if node == nil || node.Type != "preparation" {
		return nil
	}
	if g.prepConfirm {
		return "confirmation:" + g.camp.NodeID()
	}
	if g.nativePreparationPromptActive() {
		return "prompt:" + g.camp.NodeID()
	}
	return nil
}

func (g *Game) sampleNativeConfirmationMouthRandom() int {
	if g.nativeConfirmationMouthRNG == nil {
		// Cosmetic RNG stays independent of the native battle RNG and SAV.
		g.nativeConfirmationMouthRNG = rand.New(rand.NewSource(rand.Int63()))
	}
	return g.nativeConfirmationMouthRNG.Intn(30)
}

func (g *Game) stepNativeConfirmationMouth(owner any, advanced bool) {
	if owner == nil {
		g.nativeConfirmationMouthOwner = nil
		return
	}
	if owner != g.nativeConfirmationMouthOwner {
		state, err := dato.NewConfirmationMouthState(g.sampleNativeConfirmationMouthRandom())
		if err != nil {
			g.loadErr = err.Error()
			return
		}
		g.nativeConfirmationMouth, g.nativeConfirmationMouthOwner = state, owner
	}
	if !advanced {
		return
	}
	sample := 0
	if g.nativeConfirmationMouth.Open {
		sample = g.sampleNativeConfirmationMouthRandom()
	}
	state, err := g.nativeConfirmationMouth.Tick(sample)
	if err != nil {
		g.loadErr = err.Error()
		return
	}
	g.nativeConfirmationMouth = state
}

func (g *Game) composeNativeConfirmationMouth(frame []byte, portraits []dato.Frame) ([]byte, bool) {
	owner := g.nativeConfirmationMouthWaitingOwner()
	if owner == nil || owner != g.nativeConfirmationMouthOwner || !g.nativeConfirmationMouth.Open {
		return frame, len(frame) == 320*200
	}
	if len(portraits) <= 3 {
		return nil, false
	}
	result, err := campaign.ComposeNativeConfirmationPortrait(frame, portraits[3])
	return result, err == nil
}
