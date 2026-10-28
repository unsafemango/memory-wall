package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/unsafemango/memory-wall/internal/storage"
	"github.com/unsafemango/memory-wall/internal/store"
)

// VideoProcessor makes a thumbnail for each uploaded video and records its
// duration and dimensions, using ffmpeg and ffprobe as subprocesses.
type VideoProcessor struct {
	storage *storage.Storage
	media   *store.MediaStore
}

func NewVideoProcessor(s *storage.Storage, media *store.MediaStore) *VideoProcessor {
	return &VideoProcessor{storage: s, media: media}
}

func (p *VideoProcessor) Handle(ctx context.Context, payload []byte) error {
	var input struct {
		SubmissionID string `json:"submission_id"`
	}
	if err := json.Unmarshal(payload, &input); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	key, err := p.media.StorageKeyFor(ctx, input.SubmissionID)
	if err != nil {
		return fmt.Errorf("find media: %w", err)
	}

	dir, err := os.MkdirTemp("", "video-job-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	videoPath := filepath.Join(dir, "input"+filepath.Ext(key))
	if err := p.download(ctx, key, videoPath); err != nil {
		return fmt.Errorf("download video: %w", err)
	}

	info, err := probe(ctx, videoPath)
	if err != nil {
		return fmt.Errorf("probe video: %w", err)
	}

	thumbPath := filepath.Join(dir, "thumb.jpg")
	if err := generateThumbnail(ctx, videoPath, thumbPath, info.DurationSeconds); err != nil {
		return fmt.Errorf("make thumbnail: %w", err)
	}

	thumb, err := os.Open(thumbPath)
	if err != nil {
		return err
	}
	defer thumb.Close()

	info.ThumbnailKey = strings.TrimSuffix(key, filepath.Ext(key)) + "-thumb.jpg"
	if err := p.storage.Upload(ctx, info.ThumbnailKey, "image/jpeg", thumb); err != nil {
		return fmt.Errorf("upload thumbnail: %w", err)
	}

	return p.media.SetVideoInfo(ctx, input.SubmissionID, info)
}

func (p *VideoProcessor) download(ctx context.Context, key, path string) error {
	body, err := p.storage.Download(ctx, key)
	if err != nil {
		return err
	}
	defer body.Close()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, body)
	return err
}

// probe reads the first video stream's size and the file's duration.
func probe(ctx context.Context, path string) (store.VideoInfo, error) {
	out, err := run(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height:format=duration",
		"-of", "json",
		path,
	)
	if err != nil {
		return store.VideoInfo{}, err
	}

	var parsed struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return store.VideoInfo{}, err
	}
	if len(parsed.Streams) == 0 {
		return store.VideoInfo{}, fmt.Errorf("no video stream found")
	}

	duration, _ := strconv.ParseFloat(parsed.Format.Duration, 64)
	return store.VideoInfo{
		Width:           parsed.Streams[0].Width,
		Height:          parsed.Streams[0].Height,
		DurationSeconds: duration,
	}, nil
}

// generateThumbnail grabs one frame, one second in (or halfway through very
// short clips), scaled to 640px wide.
func generateThumbnail(ctx context.Context, inputPath, thumbnailPath string, duration float64) error {
	at := 1.0
	if duration > 0 && duration < 2 {
		at = duration / 2
	}
	_, err := run(ctx, "ffmpeg",
		"-y",
		"-ss", strconv.FormatFloat(at, 'f', 2, 64),
		"-i", inputPath,
		"-frames:v", "1",
		"-vf", "scale=640:-2",
		thumbnailPath,
	)
	return err
}

func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
