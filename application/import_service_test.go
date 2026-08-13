package application

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type mockLibraryRepo struct {
	mock.Mock
}

func (m *mockLibraryRepo) Add(l *domain.Library) error { return m.Called(l).Error(0) }
func (m *mockLibraryRepo) GetById(id domain.ID) (*domain.Library, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Library), args.Error(1)
}
func (m *mockLibraryRepo) List() ([]domain.Library, error) {
	args := m.Called()
	return args.Get(0).([]domain.Library), args.Error(1)
}
func (m *mockLibraryRepo) ListByType(t domain.MediaType) ([]domain.Library, error) {
	args := m.Called(t)
	return args.Get(0).([]domain.Library), args.Error(1)
}
func (m *mockLibraryRepo) Update(l *domain.Library) error { return m.Called(l).Error(0) }
func (m *mockLibraryRepo) Remove(id domain.ID) error      { return m.Called(id).Error(0) }

func idPtr(v uint64) *domain.ID { d := domain.ID(v); return &d }

type fakeUoW struct {
	part      domain.PartRepository
	history   domain.HistoryRepository
	ran       bool
	commitErr error
}

func (u *fakeUoW) Run(ctx context.Context, fn func(*domain.Repos) error) error {
	u.ran = true
	if err := fn(&domain.Repos{Part: u.part, History: u.history}); err != nil {
		return err
	}
	return u.commitErr
}

type fakeFS struct {
	moved    []moveRecord
	removed  []string
	moveErr  error
	failAt   int
	attempts int
	exists   map[string]bool
}

type moveRecord struct{ src, dst string }

func (f *fakeFS) Move(src, dst string) error {
	f.attempts++
	if f.moveErr != nil {
		return f.moveErr
	}
	if f.failAt > 0 && f.attempts == f.failAt {
		return assertErr("disk full")
	}
	f.moved = append(f.moved, moveRecord{src, dst})
	return nil
}
func (f *fakeFS) Exists(path string) bool {
	if f.exists != nil {
		return f.exists[path]
	}
	return true
}
func (f *fakeFS) Size(string) (int64, error) { return 0, nil }
func (f *fakeFS) Remove(path string) error {
	f.removed = append(f.removed, path)
	return nil
}
func (f *fakeFS) RemoveAll(path string) error {
	f.removed = append(f.removed, path)
	return nil
}
func (f *fakeFS) ListFiles(string) ([]string, error) { return nil, nil }

func TestImportService_Import(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	libraryRepo := new(mockLibraryRepo)
	txPart := new(mockPartRepo)
	txHistory := new(mockHistoryRepo)
	fs := &fakeFS{}

	dst := filepath.Join("/data", "movies", "My Movie", "My Movie - 1080p WEB-DL.mkv")

	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{Id: 1, Name: "My Movie", Type: domain.MediaTypeMovie, LibraryID: idPtr(2)}, nil)
	libraryRepo.On("GetById", domain.ID(2)).Return(&domain.Library{Id: 2, Path: "/data/movies"}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{{Id: 5, MediaId: 1, Monitored: true}}, nil)
	txPart.On("Update", mock.MatchedBy(func(p *domain.Part) bool {
		return p.Id == 5 && p.Path != nil && *p.Path == dst
	})).Return(nil)
	txHistory.On("Add", mock.MatchedBy(func(h *domain.History) bool {
		return h.EventType == domain.HistoryImported && h.Data == dst && h.PartId != nil && *h.PartId == 5
	})).Return(nil)

	svc := NewImportService(mediaRepo, partRepo, new(mockPartGroupRepo), libraryRepo, &fakeUoW{part: txPart, history: txHistory}, fs, NewNamingService(), NewReleaseParser())

	err := svc.Import(context.Background(), domain.QueueItem{
		MediaId: 1, ReleaseTitle: "My.Movie.1080p.WEB-DL-GRP",
	}, domain.GrabStatus{
		State:       domain.GrabCompleted,
		OutputFiles: []string{"/stage/job1.mkv"},
	})

	require.NoError(t, err)
	require.Len(t, fs.moved, 1)
	assert.Equal(t, "/stage/job1.mkv", fs.moved[0].src)
	assert.Equal(t, dst, fs.moved[0].dst)
	txPart.AssertExpectations(t)
	txHistory.AssertExpectations(t)
}

func TestImportService_NoOutputFiles(t *testing.T) {
	svc := NewImportService(new(mockMediaRepo), new(mockPartRepo), new(mockPartGroupRepo), new(mockLibraryRepo), &fakeUoW{}, &fakeFS{}, NewNamingService(), NewReleaseParser())
	err := svc.Import(context.Background(), domain.QueueItem{}, domain.GrabStatus{State: domain.GrabCompleted})
	assert.Error(t, err)
}

func TestImportService_CustomFolder(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	libraryRepo := new(mockLibraryRepo)
	txPart := new(mockPartRepo)
	txHistory := new(mockHistoryRepo)
	fs := &fakeFS{}

	folder := "Custom Folder"
	dst := filepath.Join("/data", "movies", "Custom Folder", "My Movie - 1080p WEB-DL.mkv")

	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{
		Id: 1, Name: "My Movie", Type: domain.MediaTypeMovie, LibraryID: idPtr(2), Folder: &folder,
	}, nil)
	libraryRepo.On("GetById", domain.ID(2)).Return(&domain.Library{Id: 2, Path: "/data/movies"}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{{Id: 5, MediaId: 1, Monitored: true}}, nil)
	txPart.On("Update", mock.MatchedBy(func(p *domain.Part) bool {
		return p.Id == 5 && p.Path != nil && *p.Path == dst
	})).Return(nil)
	txHistory.On("Add", mock.MatchedBy(func(h *domain.History) bool {
		return h.EventType == domain.HistoryImported && h.Data == dst && h.PartId != nil && *h.PartId == 5
	})).Return(nil)

	svc := NewImportService(mediaRepo, partRepo, new(mockPartGroupRepo), libraryRepo, &fakeUoW{part: txPart, history: txHistory}, fs, NewNamingService(), NewReleaseParser())

	err := svc.Import(context.Background(), domain.QueueItem{
		MediaId: 1, ReleaseTitle: "My.Movie.1080p.WEB-DL-GRP",
	}, domain.GrabStatus{
		State:       domain.GrabCompleted,
		OutputFiles: []string{"/stage/job1.mkv"},
	})

	require.NoError(t, err)
	require.Len(t, fs.moved, 1)
	assert.Equal(t, "/stage/job1.mkv", fs.moved[0].src)
	assert.Equal(t, dst, fs.moved[0].dst)
	txPart.AssertExpectations(t)
	txHistory.AssertExpectations(t)
}

func TestImportService_NoLibrary(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{Id: 1, Name: "X", Type: domain.MediaTypeMovie}, nil)

	svc := NewImportService(mediaRepo, new(mockPartRepo), new(mockPartGroupRepo), new(mockLibraryRepo), &fakeUoW{}, &fakeFS{}, NewNamingService(), NewReleaseParser())
	err := svc.Import(context.Background(), domain.QueueItem{MediaId: 1}, domain.GrabStatus{
		State: domain.GrabCompleted, OutputFiles: []string{"/stage/x.mkv"},
	})
	assert.ErrorIs(t, err, domain.ErrLibraryRequired)
}

func TestImportService_NoWantedPart(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	libraryRepo := new(mockLibraryRepo)

	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{Id: 1, Name: "X", Type: domain.MediaTypeMovie, LibraryID: idPtr(2)}, nil)
	libraryRepo.On("GetById", domain.ID(2)).Return(&domain.Library{Id: 2, Path: "/data/lib"}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{{Id: 5, MediaId: 1, Monitored: true, Path: strPtr("/already.mkv")}}, nil)

	svc := NewImportService(mediaRepo, partRepo, new(mockPartGroupRepo), libraryRepo, &fakeUoW{}, &fakeFS{}, NewNamingService(), NewReleaseParser())
	err := svc.Import(context.Background(), domain.QueueItem{MediaId: 1}, domain.GrabStatus{
		State: domain.GrabCompleted, OutputFiles: []string{"/stage/x.mkv"},
	})
	assert.ErrorIs(t, err, domain.ErrPartNotFound)
}

func TestImportService_SeriesMultiEpisode(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	groupRepo := new(mockPartGroupRepo)
	libraryRepo := new(mockLibraryRepo)
	txPart := new(mockPartRepo)
	txHistory := new(mockHistoryRepo)
	fs := &fakeFS{}

	grpID := domain.ID(10)
	ep1, ep2 := 1, 2
	mediaRepo.On("GetById", domain.ID(7)).Return(&domain.Media{Id: 7, Name: "Show", Type: domain.MediaTypeSeries, LibraryID: idPtr(3)}, nil)
	libraryRepo.On("GetById", domain.ID(3)).Return(&domain.Library{Id: 3, Path: "/data/series"}, nil)
	groupRepo.On("GetByMediaID", domain.ID(7)).Return([]domain.PartGroup{{Id: grpID, Order: 1, MediaId: 7}}, nil)
	partRepo.On("GetByMediaId", domain.ID(7)).Return([]domain.Part{
		{Id: 100, GroupId: &grpID, GroupOrder: &ep1, MediaId: 7, Monitored: true},
		{Id: 101, GroupId: &grpID, GroupOrder: &ep2, MediaId: 7, Monitored: true},
	}, nil)

	dst1 := filepath.Join("/data", "series", "Show", "Season 1", "Show S01E01.mkv")
	dst2 := filepath.Join("/data", "series", "Show", "Season 1", "Show S01E02.mkv")
	txPart.On("Update", mock.MatchedBy(func(p *domain.Part) bool {
		return p.Id == 100 && p.Path != nil && *p.Path == dst1
	})).Return(nil)
	txPart.On("Update", mock.MatchedBy(func(p *domain.Part) bool {
		return p.Id == 101 && p.Path != nil && *p.Path == dst2
	})).Return(nil)
	txHistory.On("Add", mock.MatchedBy(func(h *domain.History) bool {
		return h.EventType == domain.HistoryImported && h.PartId != nil && *h.PartId == 100
	})).Return(nil)
	txHistory.On("Add", mock.MatchedBy(func(h *domain.History) bool {
		return h.EventType == domain.HistoryImported && h.PartId != nil && *h.PartId == 101
	})).Return(nil)

	svc := NewImportService(mediaRepo, partRepo, groupRepo, libraryRepo, &fakeUoW{part: txPart, history: txHistory}, fs, NewNamingService(), NewReleaseParser())

	err := svc.Import(context.Background(), domain.QueueItem{
		MediaId: 7, PartIds: []domain.ID{100, 101}, ReleaseTitle: "Show.S01.WEB-DL-GRP",
	}, domain.GrabStatus{
		State:       domain.GrabCompleted,
		OutputFiles: []string{"/stage/Show.S01E01.mkv", "/stage/Show.S01E02.mkv"},
	})

	require.NoError(t, err)
	require.Len(t, fs.moved, 2)
	assert.Equal(t, "/stage/Show.S01E01.mkv", fs.moved[0].src)
	assert.Equal(t, dst1, fs.moved[0].dst)
	assert.Equal(t, "/stage/Show.S01E02.mkv", fs.moved[1].src)
	assert.Equal(t, dst2, fs.moved[1].dst)
	txPart.AssertExpectations(t)
	txHistory.AssertExpectations(t)
}

func TestImportService_AlbumMultiTrack(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	libraryRepo := new(mockLibraryRepo)
	txPart := new(mockPartRepo)
	txHistory := new(mockHistoryRepo)
	fs := &fakeFS{}

	t1, t2 := 1, 2
	n1, n2 := "Bohemian Rhapsody", "You're My Best Friend"
	mediaRepo.On("GetById", domain.ID(8)).Return(&domain.Media{
		Id: 8, Name: "Queen - A Night at the Opera", Type: domain.MediaTypeMusicAlbum, LibraryID: idPtr(4),
	}, nil)
	libraryRepo.On("GetById", domain.ID(4)).Return(&domain.Library{Id: 4, Path: "/data/music"}, nil)
	partRepo.On("GetByMediaId", domain.ID(8)).Return([]domain.Part{
		{Id: 200, GroupOrder: &t1, Name: &n1, MediaId: 8, Monitored: true},
		{Id: 201, GroupOrder: &t2, Name: &n2, MediaId: 8, Monitored: true},
	}, nil)

	dst1 := filepath.Join("/data", "music", "Queen", "A Night at the Opera", "01 - Bohemian Rhapsody - flac.flac")
	dst2 := filepath.Join("/data", "music", "Queen", "A Night at the Opera", "02 - You're My Best Friend - flac.flac")
	txPart.On("Update", mock.MatchedBy(func(p *domain.Part) bool {
		return p.Id == 200 && p.Path != nil && *p.Path == dst1
	})).Return(nil)
	txPart.On("Update", mock.MatchedBy(func(p *domain.Part) bool {
		return p.Id == 201 && p.Path != nil && *p.Path == dst2
	})).Return(nil)
	txHistory.On("Add", mock.MatchedBy(func(h *domain.History) bool {
		return h.EventType == domain.HistoryImported && h.PartId != nil && *h.PartId == 200
	})).Return(nil)
	txHistory.On("Add", mock.MatchedBy(func(h *domain.History) bool {
		return h.EventType == domain.HistoryImported && h.PartId != nil && *h.PartId == 201
	})).Return(nil)

	svc := NewImportService(mediaRepo, partRepo, new(mockPartGroupRepo), libraryRepo, &fakeUoW{part: txPart, history: txHistory}, fs, NewNamingService(), NewReleaseParser())

	err := svc.Import(context.Background(), domain.QueueItem{
		MediaId: 8, PartIds: []domain.ID{200, 201}, ReleaseTitle: "Queen.A.Night.at.the.Opera.FLAC",
	}, domain.GrabStatus{
		State:       domain.GrabCompleted,
		OutputFiles: []string{"/stage/02 - You're My Best Friend.flac", "/stage/01 - Bohemian Rhapsody.flac"},
	})

	require.NoError(t, err)
	require.Len(t, fs.moved, 2)
	assert.Equal(t, "/stage/01 - Bohemian Rhapsody.flac", fs.moved[0].src)
	assert.Equal(t, dst1, fs.moved[0].dst)
	assert.Equal(t, "/stage/02 - You're My Best Friend.flac", fs.moved[1].src)
	assert.Equal(t, dst2, fs.moved[1].dst)
	txPart.AssertExpectations(t)
	txHistory.AssertExpectations(t)
}

func TestImportService_AlbumPositionalFallback(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	libraryRepo := new(mockLibraryRepo)
	txPart := new(mockPartRepo)
	txHistory := new(mockHistoryRepo)
	fs := &fakeFS{}

	t1, t2 := 1, 2
	n1, n2 := "Track One", "Track Two"
	mediaRepo.On("GetById", domain.ID(9)).Return(&domain.Media{
		Id: 9, Name: "Artist - Album", Type: domain.MediaTypeMusicAlbum, LibraryID: idPtr(4),
	}, nil)
	libraryRepo.On("GetById", domain.ID(4)).Return(&domain.Library{Id: 4, Path: "/data/music"}, nil)
	partRepo.On("GetByMediaId", domain.ID(9)).Return([]domain.Part{
		{Id: 300, GroupOrder: &t1, Name: &n1, MediaId: 9, Monitored: true},
		{Id: 301, GroupOrder: &t2, Name: &n2, MediaId: 9, Monitored: true},
	}, nil)

	dst1 := filepath.Join("/data", "music", "Artist", "Album", "01 - Track One.flac")
	dst2 := filepath.Join("/data", "music", "Artist", "Album", "02 - Track Two.flac")
	txPart.On("Update", mock.MatchedBy(func(p *domain.Part) bool {
		return p.Id == 300 && p.Path != nil && *p.Path == dst1
	})).Return(nil)
	txPart.On("Update", mock.MatchedBy(func(p *domain.Part) bool {
		return p.Id == 301 && p.Path != nil && *p.Path == dst2
	})).Return(nil)
	txHistory.On("Add", mock.Anything).Return(nil)

	svc := NewImportService(mediaRepo, partRepo, new(mockPartGroupRepo), libraryRepo, &fakeUoW{part: txPart, history: txHistory}, fs, NewNamingService(), NewReleaseParser())

	err := svc.Import(context.Background(), domain.QueueItem{
		MediaId: 9, ReleaseTitle: "Artist.Album",
	}, domain.GrabStatus{
		State:       domain.GrabCompleted,
		OutputFiles: []string{"/stage/aaa.flac", "/stage/bbb.flac"},
	})

	require.NoError(t, err)
	require.Len(t, fs.moved, 2)
	assert.Equal(t, dst1, fs.moved[0].dst)
	assert.Equal(t, dst2, fs.moved[1].dst)
}

func TestImportService_MoveFailure(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	libraryRepo := new(mockLibraryRepo)
	txPart := new(mockPartRepo)
	txHistory := new(mockHistoryRepo)

	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{Id: 1, Name: "X", Type: domain.MediaTypeMovie, LibraryID: idPtr(2)}, nil)
	libraryRepo.On("GetById", domain.ID(2)).Return(&domain.Library{Id: 2, Path: "/data/lib"}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{{Id: 5, MediaId: 1, Monitored: true}}, nil)
	txPart.On("Update", mock.Anything).Return(nil)
	txHistory.On("Add", mock.Anything).Return(nil)

	uow := &fakeUoW{part: txPart, history: txHistory}
	fs := &fakeFS{moveErr: assertErr("disk full")}
	svc := NewImportService(mediaRepo, partRepo, new(mockPartGroupRepo), libraryRepo, uow, fs, NewNamingService(), NewReleaseParser())

	err := svc.Import(context.Background(), domain.QueueItem{MediaId: 1}, domain.GrabStatus{
		State: domain.GrabCompleted, OutputFiles: []string{"/stage/x.mkv"},
	})
	assert.Error(t, err)
	assert.False(t, uow.ran)
	assert.Empty(t, fs.moved)
}

func TestImportService_TxFailureRollsBackMove(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	libraryRepo := new(mockLibraryRepo)
	txPart := new(mockPartRepo)
	txHistory := new(mockHistoryRepo)
	fs := &fakeFS{}

	src := "/stage/job1.mkv"
	dst := filepath.Join("/data", "movies", "My Movie", "My Movie - 1080p WEB-DL.mkv")

	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{Id: 1, Name: "My Movie", Type: domain.MediaTypeMovie, LibraryID: idPtr(2)}, nil)
	libraryRepo.On("GetById", domain.ID(2)).Return(&domain.Library{Id: 2, Path: "/data/movies"}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{{Id: 5, MediaId: 1, Monitored: true}}, nil)
	txPart.On("Update", mock.Anything).Return(nil)
	txHistory.On("Add", mock.Anything).Return(nil)

	uow := &fakeUoW{part: txPart, history: txHistory, commitErr: assertErr("commit failed")}
	svc := NewImportService(mediaRepo, partRepo, new(mockPartGroupRepo), libraryRepo, uow, fs, NewNamingService(), NewReleaseParser())

	err := svc.Import(context.Background(), domain.QueueItem{
		MediaId: 1, ReleaseTitle: "My.Movie.1080p.WEB-DL-GRP",
	}, domain.GrabStatus{
		State:       domain.GrabCompleted,
		OutputFiles: []string{src},
	})

	require.Error(t, err)
	require.Len(t, fs.moved, 2)
	assert.Equal(t, src, fs.moved[0].src)
	assert.Equal(t, dst, fs.moved[0].dst)
	assert.Equal(t, dst, fs.moved[1].src)
	assert.Equal(t, src, fs.moved[1].dst)
}

func TestImportService_SeriesMoveFailureRollsBackPrior(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	groupRepo := new(mockPartGroupRepo)
	libraryRepo := new(mockLibraryRepo)
	txPart := new(mockPartRepo)
	txHistory := new(mockHistoryRepo)
	fs := &fakeFS{failAt: 2}

	grpID := domain.ID(10)
	ep1, ep2 := 1, 2
	mediaRepo.On("GetById", domain.ID(7)).Return(&domain.Media{Id: 7, Name: "Show", Type: domain.MediaTypeSeries, LibraryID: idPtr(3)}, nil)
	libraryRepo.On("GetById", domain.ID(3)).Return(&domain.Library{Id: 3, Path: "/data/series"}, nil)
	groupRepo.On("GetByMediaID", domain.ID(7)).Return([]domain.PartGroup{{Id: grpID, Order: 1, MediaId: 7}}, nil)
	partRepo.On("GetByMediaId", domain.ID(7)).Return([]domain.Part{
		{Id: 100, GroupId: &grpID, GroupOrder: &ep1, MediaId: 7, Monitored: true},
		{Id: 101, GroupId: &grpID, GroupOrder: &ep2, MediaId: 7, Monitored: true},
	}, nil)
	txPart.On("Update", mock.Anything).Return(nil)
	txHistory.On("Add", mock.Anything).Return(nil)

	dst1 := filepath.Join("/data", "series", "Show", "Season 1", "Show S01E01.mkv")
	src1 := "/stage/Show.S01E01.mkv"

	svc := NewImportService(mediaRepo, partRepo, groupRepo, libraryRepo, &fakeUoW{part: txPart, history: txHistory}, fs, NewNamingService(), NewReleaseParser())

	err := svc.Import(context.Background(), domain.QueueItem{
		MediaId: 7, PartIds: []domain.ID{100, 101}, ReleaseTitle: "Show.S01.WEB-DL-GRP",
	}, domain.GrabStatus{
		State:       domain.GrabCompleted,
		OutputFiles: []string{src1, "/stage/Show.S01E02.mkv"},
	})

	require.Error(t, err)
	require.Len(t, fs.moved, 2)
	assert.Equal(t, src1, fs.moved[0].src)
	assert.Equal(t, dst1, fs.moved[0].dst)
	assert.Equal(t, dst1, fs.moved[1].src)
	assert.Equal(t, src1, fs.moved[1].dst)
}

type errorAssert struct{ msg string }

func (e errorAssert) Error() string { return e.msg }

func strPtr(s string) *string { return &s }
func assertErr(msg string) error {
	return errorAssert{msg: msg}
}
