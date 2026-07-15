package campaign

import (
	"encoding/json"
	"os"
	"testing"
)

func TestChapterThreePostBattleSpeakerControlCodes(t *testing.T) {
	type storyLine struct {
		Speaker     int    `json:"speaker"`
		SpeakerName string `json:"speaker_name"`
	}
	type storyScene struct {
		Lines []storyLine `json:"lines"`
	}
	var story struct {
		Scenes []storyScene `json:"scenes"`
	}

	data, err := os.ReadFile("../../assets/story/ch03.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &story); err != nil {
		t.Fatal(err)
	}
	if len(story.Scenes) < 2 {
		t.Fatalf("ch03 story scenes = %d, want at least 2", len(story.Scenes))
	}
	if len(story.Scenes[0].Lines) < 21 {
		t.Fatalf("ch03 pre/turn3 lines = %d, want at least 21", len(story.Scenes[0].Lines))
	}
	if len(story.Scenes[1].Lines) < 18 {
		t.Fatalf("ch03 post-battle lines = %d, want at least 18", len(story.Scenes[1].Lines))
	}
	for _, want := range []struct {
		line    int
		speaker int
	}{
		{line: 14, speaker: 77},
		{line: 15, speaker: 2},
		{line: 16, speaker: 77},
		{line: 17, speaker: 8},
		{line: 18, speaker: 2},
		{line: 19, speaker: 8},
		{line: 20, speaker: 77},
	} {
		if got := story.Scenes[0].Lines[want.line].Speaker; got != want.speaker {
			t.Errorf("ch03 turn3 #4 line%d speaker=%d, want raw portrait %d", want.line, got, want.speaker)
		}
	}

	for _, want := range []struct {
		line        int
		speaker     int
		speakerName string
	}{
		{line: 0, speaker: 77, speakerName: "約"},
		{line: 17, speaker: 6, speakerName: "萊汀"},
	} {
		got := story.Scenes[1].Lines[want.line]
		if got.Speaker != want.speaker || got.SpeakerName != want.speakerName {
			t.Errorf("ch03 scene1 line%d speaker = (%d, %q), want (%d, %q)", want.line, got.Speaker, got.SpeakerName, want.speaker, want.speakerName)
		}
	}
}
