// Package musiccatalog validates the separated FM／MT-32 OGG bundle.
package musiccatalog

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var expectedTracks = [...]int{1, 3, 4, 6, 8, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}

type Source struct {
	File   string `json:"file"`
	Size   int    `json:"size"`
	MD5    string `json:"md5"`
	SHA256 string `json:"sha256"`
}

type Loop struct {
	Mode           string `json:"mode"`
	AcceptedCounts []int  `json:"accepted_counts"`
	SeamEvidence   string `json:"seam_evidence"`
	EvidenceLevel  string `json:"evidence_level"`
}

type Profile struct {
	RenderPipeline   string `json:"render_pipeline"`
	ProvenanceStatus string `json:"provenance_status"`
	RightsNote       string `json:"rights_note"`
}

type Render struct {
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	SHA256     string `json:"sha256"`
	Codec      string `json:"codec"`
	Channels   int    `json:"channels"`
	SampleRate int    `json:"sample_rate"`
	PCMSamples uint64 `json:"pcm_samples"`
	DurationMS uint64 `json:"duration_ms"`
}

type Track struct {
	TrackID       string            `json:"track_id"`
	ResourceIndex int               `json:"resource_index"`
	Renders       map[string]Render `json:"renders"`
}

type Catalog struct {
	Schema        string             `json:"schema"`
	SchemaVersion int                `json:"schema_version"`
	Source        Source             `json:"source"`
	Loop          Loop               `json:"loop"`
	Profiles      map[string]Profile `json:"profiles"`
	Tracks        []Track            `json:"tracks"`
	byID          map[string]Track
	root          string
}

func Load(assetsRoot string) (*Catalog, error) {
	raw, err := os.ReadFile(filepath.Join(assetsRoot, "music_catalog.json"))
	if err != nil {
		return nil, fmt.Errorf("music catalog: %w", err)
	}
	var catalog Catalog
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return nil, fmt.Errorf("music catalog decode: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("music catalog trailing data")
	}
	if catalog.Schema != "fd2_music_catalog" || catalog.SchemaVersion != 1 {
		return nil, fmt.Errorf("music catalog schema %q version %d", catalog.Schema, catalog.SchemaVersion)
	}
	if catalog.Source != (Source{File: "FDMUS.DAT", Size: 80367, MD5: "4dfa214125edcc4658acbba2e1201a28", SHA256: "4105ebde543fe1c497e852728f6bc333bda80edeb7fb3671e487504bee74e998"}) {
		return nil, fmt.Errorf("music catalog FDMUS identity mismatch")
	}
	if catalog.Loop.Mode != "whole_file_runtime_repeat" || len(catalog.Loop.AcceptedCounts) != 2 || catalog.Loop.AcceptedCounts[0] != 0 || catalog.Loop.AcceptedCounts[1] != 1 || catalog.Loop.SeamEvidence != "unknown" || catalog.Loop.EvidenceLevel != "strong_inference_e1" {
		return nil, fmt.Errorf("music catalog loop contract mismatch")
	}
	if len(catalog.Profiles) != 2 {
		return nil, fmt.Errorf("music catalog profiles=%d, want 2", len(catalog.Profiles))
	}
	for _, profile := range []string{"fm", "mt32"} {
		meta, ok := catalog.Profiles[profile]
		if !ok || meta.RenderPipeline == "" || meta.ProvenanceStatus != "incomplete_legacy_render" || meta.RightsNote == "" {
			return nil, fmt.Errorf("music catalog profile %q invalid", profile)
		}
	}
	if len(catalog.Tracks) != len(expectedTracks) {
		return nil, fmt.Errorf("music catalog tracks=%d, want %d", len(catalog.Tracks), len(expectedTracks))
	}
	catalog.byID = make(map[string]Track, len(catalog.Tracks))
	catalog.root = assetsRoot
	for position, resource := range expectedTracks {
		track := catalog.Tracks[position]
		expectedID := fmt.Sprintf("FDMUS_%03d", resource)
		if track.TrackID != expectedID || track.ResourceIndex != resource {
			return nil, fmt.Errorf("music catalog track %d identity mismatch", position)
		}
		if _, duplicate := catalog.byID[track.TrackID]; duplicate {
			return nil, fmt.Errorf("music catalog duplicate track %q", track.TrackID)
		}
		if len(track.Renders) != 2 {
			return nil, fmt.Errorf("music catalog track %q renders=%d", track.TrackID, len(track.Renders))
		}
		for _, profile := range []string{"fm", "mt32"} {
			render, ok := track.Renders[profile]
			if !ok {
				return nil, fmt.Errorf("music catalog track %q lacks %s", track.TrackID, profile)
			}
			expectedPath := filepath.ToSlash(filepath.Join("music_"+profile, expectedID+".ogg"))
			if render.Path != expectedPath || filepath.IsAbs(render.Path) || strings.Contains(render.Path, "..") {
				return nil, fmt.Errorf("music catalog track %q profile %s path %q", track.TrackID, profile, render.Path)
			}
			if err := validateRender(filepath.Join(assetsRoot, filepath.FromSlash(render.Path)), render); err != nil {
				return nil, fmt.Errorf("music catalog track %q profile %s: %w", track.TrackID, profile, err)
			}
		}
		catalog.byID[track.TrackID] = track
	}
	return &catalog, nil
}

func (c *Catalog) Resolve(profile, trackID string) (string, error) {
	if c == nil || (profile != "fm" && profile != "mt32") {
		return "", fmt.Errorf("music profile %q unavailable", profile)
	}
	track, ok := c.byID[trackID]
	if !ok {
		return "", fmt.Errorf("music track %q unavailable", trackID)
	}
	return filepath.Join(c.root, filepath.FromSlash(track.Renders[profile].Path)), nil
}

func validateRender(path string, expected Render) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	if int64(len(raw)) != expected.Bytes || hex.EncodeToString(sum[:]) != expected.SHA256 {
		return fmt.Errorf("identity mismatch")
	}
	channels, rate, samples, err := inspectVorbis(raw)
	if err != nil {
		return err
	}
	duration := (samples*1000 + uint64(rate)/2) / uint64(rate)
	if expected.Codec != "vorbis" || channels != expected.Channels || rate != expected.SampleRate || samples != expected.PCMSamples || duration != expected.DurationMS || channels != 2 {
		return fmt.Errorf("Vorbis geometry mismatch")
	}
	return nil
}

func inspectVorbis(raw []byte) (int, int, uint64, error) {
	offset := 0
	packet := make([]byte, 0, 64)
	var identification []byte
	var lastGranule uint64
	for offset < len(raw) {
		if offset+27 > len(raw) || string(raw[offset:offset+4]) != "OggS" || raw[offset+4] != 0 {
			return 0, 0, 0, fmt.Errorf("invalid Ogg page at %d", offset)
		}
		segments := int(raw[offset+26])
		headerEnd := offset + 27 + segments
		if headerEnd > len(raw) {
			return 0, 0, 0, fmt.Errorf("truncated Ogg segment table")
		}
		bodyLength := 0
		for _, length := range raw[offset+27 : headerEnd] {
			bodyLength += int(length)
		}
		bodyEnd := headerEnd + bodyLength
		if bodyEnd > len(raw) {
			return 0, 0, 0, fmt.Errorf("truncated Ogg body")
		}
		granule := binary.LittleEndian.Uint64(raw[offset+6 : offset+14])
		if granule != ^uint64(0) {
			lastGranule = granule
		}
		cursor := headerEnd
		for _, lengthByte := range raw[offset+27 : headerEnd] {
			length := int(lengthByte)
			packet = append(packet, raw[cursor:cursor+length]...)
			cursor += length
			if length < 255 {
				if identification == nil {
					identification = append([]byte(nil), packet...)
				}
				packet = packet[:0]
			}
		}
		offset = bodyEnd
	}
	if offset != len(raw) || len(packet) != 0 || len(identification) < 16 || string(identification[:7]) != "\x01vorbis" || lastGranule == 0 {
		return 0, 0, 0, fmt.Errorf("invalid Vorbis stream")
	}
	if binary.LittleEndian.Uint32(identification[7:11]) != 0 {
		return 0, 0, 0, fmt.Errorf("unsupported Vorbis version")
	}
	return int(identification[11]), int(binary.LittleEndian.Uint32(identification[12:16])), lastGranule, nil
}
