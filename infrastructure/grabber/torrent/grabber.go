package torrent

import (
	"context"
	"path/filepath"
	"strings"
	"unicode"

	"stersh.ru/mediator/domain"
)

type Grabber struct {
	name     string
	runner   domain.DownloadRunner
	fs       domain.FileService
	category string
}

func New(name string, runner domain.DownloadRunner, fs domain.FileService, category string) *Grabber {
	return &Grabber{name: name, runner: runner, fs: fs, category: category}
}

func (g *Grabber) Name() string { return g.name }

func (g *Grabber) Supports(t domain.GrabTarget) bool {
	return t.Release != nil && (t.Release.MagnetURI != "" || t.Release.DownloadURL != "")
}

func (g *Grabber) SupportsProvider(providerName string) bool {
	return false
}

func (g *Grabber) Submit(ctx context.Context, t domain.GrabTarget) (domain.GrabHandle, error) {
	task, err := g.runner.Add(context.Background(), *t.Release, g.category)
	if err != nil {
		return domain.GrabHandle{}, err
	}
	return domain.GrabHandle{GrabberName: g.name, JobID: task.ID, Name: t.Release.Title}, nil
}

func (g *Grabber) Status(ctx context.Context, h domain.GrabHandle) (domain.GrabStatus, error) {
	task, err := g.runner.Get(ctx, h.JobID)
	if err != nil {
		if h.Name != "" {
			tasks, listErr := g.runner.List(ctx)
			if listErr == nil {
				if best := bestTokenMatch(h.Name, tasks); best != nil {
					st := g.taskToStatus(best)
					st.ResolvedID = best.ID
					return st, nil
				}
			}
		}
		return domain.GrabStatus{}, err
	}
	return g.taskToStatus(task), nil
}

func (g *Grabber) Cancel(ctx context.Context, h domain.GrabHandle) error {
	return g.runner.Remove(ctx, h.JobID, true)
}

func (g *Grabber) taskToStatus(t *domain.DownloadTask) domain.GrabStatus {
	st := domain.GrabStatus{Progress: t.Progress}
	switch t.Status {
	case domain.DownloadCompleted:
		st.State = domain.GrabCompleted
		if t.OutputPath != "" {
			st.OutputFiles = g.outputFiles(t.OutputPath)
		}
	case domain.DownloadFailed:
		st.State = domain.GrabFailed
	case domain.DownloadQueued:
		st.State = domain.GrabQueued
	default:
		st.State = domain.GrabRunning
	}
	return st
}

func (g *Grabber) outputFiles(outputPath string) []string {
	if g.fs == nil {
		return []string{outputPath}
	}
	files, err := g.fs.ListFiles(outputPath)
	if err != nil || len(files) == 0 {
		return []string{outputPath}
	}
	video := make([]string, 0, len(files))
	for _, f := range files {
		if isVideoFile(f) {
			video = append(video, f)
		}
	}
	if len(video) == 0 {
		return files
	}
	return video
}

func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mkv", ".mp4", ".avi", ".ts", ".m4v", ".mov", ".wmv", ".flv", ".webm", ".mpg", ".mpeg":
		return true
	}
	return false
}

func tokenize(s string) map[string]struct{} {
	out := make(map[string]struct{})
	var cur strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
		} else {
			if cur.Len() >= 2 {
				out[strings.ToLower(cur.String())] = struct{}{}
			}
			cur.Reset()
		}
	}
	if cur.Len() >= 2 {
		out[strings.ToLower(cur.String())] = struct{}{}
	}
	return out
}

func matchReleaseName(a, b string) bool {
	ta := tokenize(a)
	tb := tokenize(b)
	if len(ta) == 0 || len(tb) == 0 {
		return false
	}
	common := 0
	for w := range ta {
		if _, ok := tb[w]; ok {
			common++
		}
	}
	union := len(ta) + len(tb) - common
	if union == 0 {
		return false
	}
	return float64(common)/float64(union) >= 0.4 && common >= 2
}

func bestTokenMatch(releaseTitle string, tasks []domain.DownloadTask) *domain.DownloadTask {
	releaseTokens := tokenize(releaseTitle)
	if len(releaseTokens) == 0 {
		return nil
	}
	var best *domain.DownloadTask
	bestScore := 0
	for i := range tasks {
		taskTokens := tokenize(tasks[i].Name)
		common := 0
		for w := range releaseTokens {
			if _, ok := taskTokens[w]; ok {
				common++
			}
		}
		if common > bestScore {
			bestScore = common
			best = &tasks[i]
		}
	}
	if bestScore < 1 {
		return nil
	}
	return best
}

var _ domain.Grabber = (*Grabber)(nil)
