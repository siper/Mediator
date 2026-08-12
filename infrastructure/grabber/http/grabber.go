package direct

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"stersh.ru/mediator/domain"
)

type Grabber struct {
	name       string
	stagingDir string
	http       *http.Client
	counter    uint64
	jobs       sync.Map
}

type job struct {
	mu       sync.Mutex
	state    domain.GrabState
	progress float64
	output   string
	err      string
}

func New(name, stagingDir string) *Grabber {
	return &Grabber{
		name:       name,
		stagingDir: stagingDir,
		http:       &http.Client{Timeout: 5 * time.Minute},
	}
}

func (g *Grabber) Name() string { return g.name }

func (g *Grabber) Supports(t domain.GrabTarget) bool {
	return t.URL != "" && !strings.HasPrefix(t.URL, "magnet:")
}

func (g *Grabber) SupportsProvider(providerName string) bool {
	return false
}

func (g *Grabber) Submit(ctx context.Context, t domain.GrabTarget) (domain.GrabHandle, error) {
	if t.URL == "" {
		return domain.GrabHandle{}, fmt.Errorf("direct grabber: empty url")
	}
	jobID := fmt.Sprintf("http-%d", atomic.AddUint64(&g.counter, 1))
	j := &job{state: domain.GrabQueued}
	g.jobs.Store(jobID, j)
	go g.download(jobID, t.URL)
	return domain.GrabHandle{GrabberName: g.name, JobID: jobID}, nil
}

func (g *Grabber) download(jobID, src string) {
	val, ok := g.jobs.Load(jobID)
	if !ok {
		return
	}
	j := val.(*job)
	j.setState(domain.GrabRunning, 0)

	dst := filepath.Join(g.stagingDir, jobID+extFromURL(src))
	req, err := http.NewRequest(http.MethodGet, src, nil)
	if err != nil {
		j.fail(err.Error())
		return
	}
	resp, err := g.http.Do(req)
	if err != nil {
		j.fail(err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		j.fail(fmt.Sprintf("HTTP %d", resp.StatusCode))
		return
	}
	if err := os.MkdirAll(g.stagingDir, 0o755); err != nil {
		j.fail(err.Error())
		return
	}
	f, err := os.Create(dst)
	if err != nil {
		j.fail(err.Error())
		return
	}
	var total int64 = -1
	if resp.ContentLength > 0 {
		total = resp.ContentLength
	}
	pr := &progressReader{
		w:           io.Writer(f),
		total:       total,
		onProgress:  j.updateProgress,
	}
	if _, err := io.Copy(pr, resp.Body); err != nil {
		f.Close()
		os.Remove(dst)
		j.fail(err.Error())
		return
	}
	if err := f.Close(); err != nil {
		os.Remove(dst)
		j.fail(err.Error())
		return
	}
	j.complete(dst)
}

type progressReader struct {
	w          io.Writer
	total      int64
	written    int64
	onProgress func(float64)
}

func (pr *progressReader) Write(p []byte) (int, error) {
	n, err := pr.w.Write(p)
	pr.written += int64(n)
	if pr.total > 0 {
		pr.onProgress(float64(pr.written) / float64(pr.total))
	}
	return n, err
}

func (g *Grabber) Status(ctx context.Context, h domain.GrabHandle) (domain.GrabStatus, error) {
	val, ok := g.jobs.Load(h.JobID)
	if !ok {
		return domain.GrabStatus{}, domain.ErrPartNotFound
	}
	j := val.(*job)
	j.mu.Lock()
	defer j.mu.Unlock()

	st := domain.GrabStatus{State: j.state, Progress: j.progress, Error: j.err}
	if j.state == domain.GrabCompleted && j.output != "" {
		st.OutputFiles = []string{j.output}
	}
	return st, nil
}

func (g *Grabber) Cancel(ctx context.Context, h domain.GrabHandle) error {
	g.jobs.Delete(h.JobID)
	return nil
}

func (j *job) setState(s domain.GrabState, p float64) {
	j.mu.Lock()
	j.state = s
	j.progress = p
	j.mu.Unlock()
}

func (j *job) updateProgress(p float64) {
	j.mu.Lock()
	if p > j.progress {
		j.progress = p
	}
	j.mu.Unlock()
}

func (j *job) fail(msg string) {
	j.mu.Lock()
	j.state = domain.GrabFailed
	j.err = msg
	j.mu.Unlock()
}

func (j *job) complete(output string) {
	j.mu.Lock()
	j.state = domain.GrabCompleted
	j.progress = 1.0
	j.output = output
	j.mu.Unlock()
}

func extFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ".bin"
	}
	ext := filepath.Ext(u.Path)
	if ext == "" {
		return ".bin"
	}
	return ext
}

var _ domain.Grabber = (*Grabber)(nil)
