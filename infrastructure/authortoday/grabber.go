package authortoday

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"stersh.ru/mediator/domain"
)

type Grabber struct {
	client     *APIClient
	mediaRepo  domain.MediaRepository
	stagingDir string
	counter    uint64
	jobs       sync.Map
}

func NewGrabber(token string, mediaRepo domain.MediaRepository, stagingDir string) *Grabber {
	return &Grabber{
		client:     NewAPIClient(func() string { return token }),
		mediaRepo:  mediaRepo,
		stagingDir: stagingDir,
	}
}

func (g *Grabber) Name() string { return Name }

func (g *Grabber) Supports(t domain.GrabTarget) bool {
	return t.ProviderName == Name
}

func (g *Grabber) SupportsProvider(providerName string) bool {
	return providerName == Name
}

type atJob struct {
	mu       sync.Mutex
	id       string
	state    domain.GrabState
	progress float64
	output   string
	err      string
}

func (g *Grabber) Submit(ctx context.Context, t domain.GrabTarget) (domain.GrabHandle, error) {
	slog.Debug("author.today grabber: submit", "media_id", t.MediaID, "provider_name", t.ProviderName)
	media, err := g.mediaRepo.GetById(t.MediaID)
	if err != nil {
		slog.Warn("author.today grabber: failed to get media", "media_id", t.MediaID, "err", err)
		return domain.GrabHandle{}, err
	}
	slog.Debug("author.today grabber: media fetched", "media_id", media.Id, "provider_id", media.ProviderID, "external_id", media.ExternalID, "name", media.Name)
	if media.ProviderID != Name {
		slog.Warn("author.today grabber: media provider mismatch", "media_id", media.Id, "provider_id", media.ProviderID)
		return domain.GrabHandle{}, fmt.Errorf("author.today grabber: media provider is %q", media.ProviderID)
	}
	if !g.client.HasToken() {
		slog.Warn("author.today grabber: no token configured")
		return domain.GrabHandle{}, fmt.Errorf("author.today: %w", domain.ErrNoToken)
	}

	jobID := fmt.Sprintf("at-%d", atomic.AddUint64(&g.counter, 1))
	j := &atJob{id: jobID, state: domain.GrabQueued}
	g.jobs.Store(jobID, j)
	slog.Info("author.today grabber: job submitted", "job_id", jobID, "external_id", media.ExternalID, "media_id", media.Id, "media_name", media.Name)
	go g.run(context.Background(), jobID, media.ExternalID, media.Name)
	return domain.GrabHandle{GrabberName: g.Name(), JobID: jobID}, nil
}

func (g *Grabber) run(ctx context.Context, jobID, externalID, mediaName string) {
	slog.Debug("author.today grabber: job started", "job_id", jobID, "external_id", externalID, "media_name", mediaName)
	j := g.job(jobID)
	if j == nil {
		slog.Warn("author.today grabber: job not found", "job_id", jobID)
		return
	}
	j.setState(domain.GrabRunning, 0.1)

	workID, err := parseWorkID(externalID)
	if err != nil {
		slog.Warn("author.today grabber: parse work id failed", "job_id", jobID, "external_id", externalID, "err", err)
		j.fail(err.Error())
		return
	}
	slog.Debug("author.today grabber: work id parsed", "job_id", jobID, "work_id", workID)

	w, err := g.client.WorkDetails(ctx, workID)
	if err != nil {
		slog.Warn("author.today grabber: work details failed", "job_id", jobID, "work_id", workID, "err", err)
		j.fail("work-details: " + err.Error())
		return
	}
	slog.Debug("author.today grabber: work details fetched", "job_id", jobID, "work_id", workID, "title", w.Title, "is_purchased", w.IsPurchased, "allow_downloads", w.AllowDownloads, "chapters_count", len(w.Chapters))

	var ids []int64
	for _, ch := range w.Chapters {
		slog.Debug("author.today grabber: chapter info", "job_id", jobID, "chapter_id", ch.ID, "title", ch.Title, "is_available", ch.IsAvailable, "sort_order", ch.SortOrder, "text_length", ch.TextLength)
		if ch.IsAvailable {
			ids = append(ids, ch.ID)
		}
	}
	slog.Debug("author.today grabber: chapter availability", "job_id", jobID, "total_chapters", len(w.Chapters), "available_chapters", len(ids))
	if len(ids) == 0 {
		slog.Warn("author.today grabber: no accessible chapters", "job_id", jobID, "work_id", workID, "is_purchased", w.IsPurchased, "allow_downloads", w.AllowDownloads)
		j.fail("no accessible chapters (book may not be purchased or is unavailable)")
		return
	}
	j.setState(domain.GrabRunning, 0.4)

	chapters := make([]Chapter, 0, len(ids))
	for _, ch := range w.Chapters {
		if !ch.IsAvailable {
			continue
		}
		plain, secret, err := g.client.ChapterText(ctx, workID, ch.ID)
		if err != nil {
			slog.Warn("author.today grabber: chapter text failed", "job_id", jobID, "chapter_id", ch.ID, "title", ch.Title, "err", err)
			continue
		}
		if plain == "" || secret == "" {
			slog.Debug("author.today grabber: empty chapter text or secret", "job_id", jobID, "chapter_id", ch.ID, "title", ch.Title, "text_len", len(plain), "secret_len", len(secret))
			continue
		}
		decoded := DecodeText(secret, plain)
		slog.Debug("author.today grabber: chapter decrypted", "job_id", jobID, "chapter_id", ch.ID, "title", ch.Title, "secret_len", len(secret), "text_len", len(plain), "decoded_len", len(decoded))
		chapters = append(chapters, Chapter{
			Title: ch.Title,
			HTML:  decoded,
		})
	}
	slog.Debug("author.today grabber: chapters decrypted", "job_id", jobID, "decrypted_count", len(chapters))
	if len(chapters) == 0 {
		slog.Warn("author.today grabber: no decrypted chapters", "job_id", jobID, "work_id", workID)
		j.fail("no decrypted chapters")
		return
	}

	book := Book{
		Title:      w.Title,
		Annotation: w.Annotation,
		Authors:    []string{w.AuthorFIO},
		Chapters:   chapters,
	}
	if w.SeriesTitle != "" && w.SeriesOrder > 0 {
		book.Series = &Series{Title: w.SeriesTitle, Number: w.SeriesOrder + 1}
	}

	if err := os.MkdirAll(g.stagingDir, 0o755); err != nil {
		slog.Warn("author.today grabber: mkdir failed", "job_id", jobID, "staging_dir", g.stagingDir, "err", err)
		j.fail(err.Error())
		return
	}
	dst := filepath.Join(g.stagingDir, jobID+".fb2")
	if err := os.WriteFile(dst, []byte(BuildFB2(book)), 0o644); err != nil {
		slog.Warn("author.today grabber: write file failed", "job_id", jobID, "dst", dst, "err", err)
		j.fail(err.Error())
		return
	}
	slog.Info("author.today grabber: job completed", "job_id", jobID, "dst", dst)
	j.complete(dst)
}

func (g *Grabber) Status(ctx context.Context, h domain.GrabHandle) (domain.GrabStatus, error) {
	j := g.job(h.JobID)
	if j == nil {
		slog.Warn("author.today grabber: status requested for unknown job", "job_id", h.JobID)
		return domain.GrabStatus{}, domain.ErrPartNotFound
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	st := domain.GrabStatus{State: j.state, Progress: j.progress, Error: j.err}
	if j.state == domain.GrabCompleted && j.output != "" {
		st.OutputFiles = []string{j.output}
	}
	slog.Debug("author.today grabber: status requested", "job_id", h.JobID, "state", st.State, "progress", st.Progress, "error", st.Error)
	return st, nil
}

func (g *Grabber) Cancel(ctx context.Context, h domain.GrabHandle) error {
	if j, ok := g.jobs.LoadAndDelete(h.JobID); ok {
		if aj, ok := j.(*atJob); ok && aj.output != "" {
			os.Remove(aj.output)
		}
	}
	return nil
}

func (g *Grabber) job(jobID string) *atJob {
	val, ok := g.jobs.Load(jobID)
	if !ok {
		return nil
	}
	return val.(*atJob)
}

func (j *atJob) setState(s domain.GrabState, p float64) {
	j.mu.Lock()
	j.state = s
	j.progress = p
	j.mu.Unlock()
}

func (j *atJob) fail(msg string) {
	slog.Warn("author.today grabber: job failed", "job_id", j.id, "error", msg)
	j.mu.Lock()
	j.state = domain.GrabFailed
	j.err = msg
	j.mu.Unlock()
}

func (j *atJob) complete(output string) {
	slog.Info("author.today grabber: job completed", "job_id", j.id, "output", output)
	j.mu.Lock()
	j.state = domain.GrabCompleted
	j.progress = 1.0
	j.output = output
	j.mu.Unlock()
}

func parseWorkID(externalID string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(externalID, "%d", &id)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid author.today work id %q", externalID)
	}
	return id, nil
}

var _ domain.Grabber = (*Grabber)(nil)
