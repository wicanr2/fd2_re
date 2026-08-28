package main

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type nativeAttackResource struct {
	id        int
	images    []*ebiten.Image
	delays    []int
	positions [][2]int
}

// ensureNativeAttackPresentation supplements the checked-in export subset
// from the player's separated FIGANI pack. All resources are decoded and
// scheduled before any Game map is changed, so a malformed pair cannot leave
// a half-published presentation.
func (g *Game) ensureNativeAttackPresentation(atkGroup, defGroup int) error {
	if g.nativeAttackPresentationAvailable(atkGroup, defGroup) {
		return nil
	}
	if g == nil || len(g.nativeUIPalette) < 256 {
		return errors.New("native FIGANI palette unavailable")
	}
	animationRoot := separatedAssetPath("animations")

	ids := []int{figaniIndex(atkGroup) + 1, figaniIndex(defGroup)}
	resources := make([]nativeAttackResource, 0, len(ids))
	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		animation, err := figani.LoadSeparatedResource(animationRoot, id)
		if err != nil {
			return fmt.Errorf("native FIGANI resource %d: %w", id, err)
		}
		images, positions, err := nativeFIGANIImages(animation, g.nativeUIPalette)
		if err != nil {
			return fmt.Errorf("native FIGANI resource %d images: %w", id, err)
		}
		delays := make([]int, len(animation.Frames))
		for i := range animation.Frames {
			delays[i] = animation.Frames[i].Delay
		}
		if _, err := figani.NewDisplayScheduler(delays, battleFPT()); err != nil {
			return fmt.Errorf("native FIGANI resource %d schedule: %w", id, err)
		}
		resources = append(resources, nativeAttackResource{
			id: id, images: images, delays: delays, positions: positions,
		})
	}

	if g.figani == nil {
		g.figani = make(map[int][]*ebiten.Image)
	}
	if g.figaniDelays == nil {
		g.figaniDelays = make(map[int][]int)
	}
	if g.figMeta == nil {
		g.figMeta = make(map[int][][2]int)
	}
	for _, resource := range resources {
		g.figani[resource.id] = resource.images
		g.figaniDelays[resource.id] = resource.delays
		g.figMeta[resource.id] = resource.positions
	}
	if !g.nativeAttackPresentationAvailable(atkGroup, defGroup) {
		return errors.New("native FIGANI presentation remained incomplete")
	}
	return nil
}
