package systems

import (
	"fmt"
	"io"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

// AudioSystem handles all audio playback
type AudioSystem struct {
	audioContext *audio.Context
	bgmPlayer    *audio.Player
	bgmStream    io.ReadSeeker
	volume       float64
	sampleRate   int
}

// NewAudioSystem creates a new audio system
func NewAudioSystem() *AudioSystem {
	sampleRate := 44100
	return &AudioSystem{
		audioContext: audio.NewContext(sampleRate),
		volume:       1.0, // Default volume
		sampleRate:   sampleRate,
	}
}

// PlayBGM starts playing background music
func (s *AudioSystem) PlayBGM(path string) error {
	// Stop any currently playing BGM
	if s.bgmPlayer != nil {
		s.bgmPlayer.Close()
		s.bgmPlayer = nil
	}
	if s.bgmStream != nil {
		if closer, ok := s.bgmStream.(io.Closer); ok {
			closer.Close()
		}
		s.bgmStream = nil
	}

	// Open the audio file
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open audio file: %v", err)
	}

	var stream io.ReadSeeker

	// Determine file type and create appropriate stream
	if len(path) > 4 {
		switch path[len(path)-4:] {
		case ".mp3":
			stream, err = mp3.DecodeWithSampleRate(s.sampleRate, file)
		case ".ogg":
			stream, err = vorbis.DecodeWithSampleRate(s.sampleRate, file)
		default:
			file.Close()
			return fmt.Errorf("unsupported audio format: %s", path)
		}
		if err != nil {
			file.Close()
			return fmt.Errorf("failed to decode audio file: %v", err)
		}
	}

	s.bgmStream = stream
	player, err := s.audioContext.NewPlayer(stream)
	if err != nil {
		file.Close()
		if closer, ok := stream.(io.Closer); ok {
			closer.Close()
		}
		return fmt.Errorf("failed to create audio player: %v", err)
	}

	s.bgmPlayer = player
	s.bgmPlayer.SetVolume(s.volume)
	s.bgmPlayer.Play()
	return nil
}

// StopBGM stops the background music
func (s *AudioSystem) StopBGM() {
	if s.bgmPlayer != nil {
		s.bgmPlayer.Close()
		s.bgmPlayer = nil
	}
	if s.bgmStream != nil {
		if closer, ok := s.bgmStream.(io.Closer); ok {
			closer.Close()
		}
		s.bgmStream = nil
	}
}

// ResumeBGM resumes the background music
func (s *AudioSystem) ResumeBGM() {
	if s.bgmPlayer != nil {
		s.bgmPlayer.Play()
	}
}

// IsBGMPlaying returns whether background music is currently playing
func (s *AudioSystem) IsBGMPlaying() bool {
	return s.bgmPlayer != nil && s.bgmPlayer.IsPlaying()
}

// SetVolume sets the volume for background music (0.0 to 1.0)
func (s *AudioSystem) SetVolume(volume float64) {
	s.volume = volume
	if s.bgmPlayer != nil {
		s.bgmPlayer.SetVolume(volume)
	}
}

// GetVolume returns the current volume setting
func (s *AudioSystem) GetVolume() float64 {
	return s.volume
}

func (s *AudioSystem) Close() {
	s.StopBGM()
}
