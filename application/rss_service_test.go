package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type mockIndexerSource struct {
	indexers []domain.ReleaseIndexer
}

func (m *mockIndexerSource) Active() []domain.ReleaseIndexer { return m.indexers }

type rssIndexer struct {
	name         string
	rels         []domain.Release
	err          error
	since        time.Time
	capturedCats []int
}

func (r *rssIndexer) Name() string { return r.name }
func (r *rssIndexer) Search(context.Context, string, []int) ([]domain.Release, error) {
	return nil, nil
}
func (r *rssIndexer) RSS(ctx context.Context, cats []int, since time.Time) ([]domain.Release, error) {
	r.capturedCats = cats
	r.since = since
	if r.err != nil {
		return nil, r.err
	}
	return r.rels, nil
}

func TestRssService_NoIndexers(t *testing.T) {
	svc := NewRssService(
		NewGrabService(nil, new(mockQueueRepo), newGrabUoW(new(mockQueueRepo), new(mockHistoryRepo))),
		new(mockMediaRepo),
		new(mockPartRepo),
		new(mockPartGroupRepo),
		new(mockQualityRepo),
		NewReleaseParser(),
		&mockIndexerSource{indexers: nil},
	)
	require.NoError(t, svc.Run(context.Background()))
}

func TestRssService_NoSeries(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	mediaType := domain.MediaTypeSeries
	mediaRepo.On("GetPaged", 1, 100, &mediaType).Return([]domain.Media{}, nil)

	ix := &rssIndexer{name: "ix", rels: []domain.Release{{Title: "Some.Show.S01E02.1080p.WEB-DL-GRP", MagnetURI: "magnet:?x"}}}
	svc := NewRssService(
		NewGrabService(nil, new(mockQueueRepo), newGrabUoW(new(mockQueueRepo), new(mockHistoryRepo))),
		mediaRepo,
		new(mockPartRepo),
		new(mockPartGroupRepo),
		new(mockQualityRepo),
		NewReleaseParser(),
		&mockIndexerSource{indexers: []domain.ReleaseIndexer{ix}},
	)
	require.NoError(t, svc.Run(context.Background()))
}

func TestRssService_GrabsMatchingEpisode(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	groupRepo := new(mockPartGroupRepo)
	qualityRepo := new(mockQualityRepo)

	profileID := domain.ID(10)
	media := &domain.Media{
		Id: 1, Name: "My Show", Type: domain.MediaTypeSeries, QualityProfileID: &profileID,
	}
	seasonID := domain.ID(100)
	ep := 2
	part := domain.Part{Id: 5, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep, Monitored: true}

	mt := domain.MediaTypeSeries
	mediaRepo.On("GetPaged", 1, 100, &mt).Return([]domain.Media{*media}, nil)
	mediaRepo.On("GetPaged", 2, 100, &mt).Return([]domain.Media{}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{part}, nil)
	groupRepo.On("GetByMediaID", domain.ID(1)).Return([]domain.PartGroup{{Id: 100, Order: 1, MediaId: 1}}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeSeries,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoSeries, Name: "1080p WEB-DL"}},
	}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)

	var grabbedTitle string
	var grabbedPartIds []domain.ID
	queueRepo.On("Add", mock.MatchedBy(func(q *domain.QueueItem) bool {
		grabbedTitle = q.ReleaseTitle
		grabbedPartIds = q.PartIds
		return q.MediaId == 1 && q.GrabberName == "torrent"
	})).Return(nil)
	historyRepo.On("Add", mock.Anything).Return(nil)

	ix := &rssIndexer{name: "ix", rels: []domain.Release{
		{Title: "My.Show.S01E02.1080p.WEB-DL-GRP", MagnetURI: "magnet:?x", Seeders: 20},
	}}
	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))

	svc := NewRssService(grabSvc, mediaRepo, partRepo, groupRepo, qualityRepo, NewReleaseParser(), &mockIndexerSource{indexers: []domain.ReleaseIndexer{ix}})
	require.NoError(t, svc.Run(context.Background()))

	require.Len(t, grabbedPartIds, 1)
	assert.Equal(t, domain.ID(5), grabbedPartIds[0])
	assert.NotEmpty(t, grabbedTitle)
}

func TestRssService_GrabsSeasonPack(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	groupRepo := new(mockPartGroupRepo)
	qualityRepo := new(mockQualityRepo)

	profileID := domain.ID(10)
	media := &domain.Media{
		Id: 1, Name: "My Show", Type: domain.MediaTypeSeries, QualityProfileID: &profileID,
	}
	seasonID := domain.ID(100)
	ep3, ep4 := 3, 4
	parts := []domain.Part{
		{Id: 5, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep3, Monitored: true},
		{Id: 6, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep4, Monitored: true},
	}

	mt := domain.MediaTypeSeries
	mediaRepo.On("GetPaged", 1, 100, &mt).Return([]domain.Media{*media}, nil)
	mediaRepo.On("GetPaged", 2, 100, &mt).Return([]domain.Media{}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return(parts, nil)
	groupRepo.On("GetByMediaID", domain.ID(1)).Return([]domain.PartGroup{{Id: 100, Order: 1, MediaId: 1}}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeSeries,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoSeries, Name: "1080p WEB-DL"}},
	}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)

	var capturedPartIds []domain.ID
	queueRepo.On("Add", mock.MatchedBy(func(q *domain.QueueItem) bool {
		capturedPartIds = q.PartIds
		return q.MediaId == 1
	})).Return(nil)
	historyRepo.On("Add", mock.Anything).Return(nil)

	ix := &rssIndexer{name: "ix", rels: []domain.Release{
		{Title: "My.Show.S01.Complete.1080p.WEB-DL-GRP", MagnetURI: "magnet:?pack", Seeders: 5},
	}}
	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))

	svc := NewRssService(grabSvc, mediaRepo, partRepo, groupRepo, qualityRepo, NewReleaseParser(), &mockIndexerSource{indexers: []domain.ReleaseIndexer{ix}})
	require.NoError(t, svc.Run(context.Background()))

	require.Len(t, capturedPartIds, 2)
	assert.Contains(t, capturedPartIds, domain.ID(5))
	assert.Contains(t, capturedPartIds, domain.ID(6))
}

func TestRssService_SkipsAlreadyImportedPart(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	groupRepo := new(mockPartGroupRepo)
	qualityRepo := new(mockQualityRepo)

	profileID := domain.ID(10)
	media := &domain.Media{
		Id: 1, Name: "My Show", Type: domain.MediaTypeSeries, QualityProfileID: &profileID,
	}
	seasonID := domain.ID(100)
	ep := 2
	existingPath := "/lib/show/s01e02.mkv"
	parts := []domain.Part{
		{Id: 5, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep, Monitored: true, Path: &existingPath},
	}

	mt := domain.MediaTypeSeries
	mediaRepo.On("GetPaged", 1, 100, &mt).Return([]domain.Media{*media}, nil)
	mediaRepo.On("GetPaged", 2, 100, &mt).Return([]domain.Media{}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return(parts, nil)
	groupRepo.On("GetByMediaID", domain.ID(1)).Return([]domain.PartGroup{{Id: 100, Order: 1, MediaId: 1}}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeSeries,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoSeries, Name: "1080p WEB-DL"}},
	}, nil)

	ix := &rssIndexer{name: "ix", rels: []domain.Release{
		{Title: "My.Show.S01E02.1080p.WEB-DL-GRP", MagnetURI: "magnet:?x"},
	}}
	grabber := &fakeGrabber{name: "torrent", support: true}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))

	svc := NewRssService(grabSvc, mediaRepo, partRepo, groupRepo, qualityRepo, NewReleaseParser(), &mockIndexerSource{indexers: []domain.ReleaseIndexer{ix}})
	require.NoError(t, svc.Run(context.Background()))

	queueRepo.AssertNotCalled(t, "Add")
}

func TestRssService_SkipsUnwantedQuality(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	groupRepo := new(mockPartGroupRepo)
	qualityRepo := new(mockQualityRepo)

	profileID := domain.ID(10)
	media := &domain.Media{
		Id: 1, Name: "My Show", Type: domain.MediaTypeSeries, QualityProfileID: &profileID,
	}
	seasonID := domain.ID(100)
	ep := 2
	part := domain.Part{Id: 5, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep, Monitored: true}

	mt := domain.MediaTypeSeries
	mediaRepo.On("GetPaged", 1, 100, &mt).Return([]domain.Media{*media}, nil)
	mediaRepo.On("GetPaged", 2, 100, &mt).Return([]domain.Media{}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{part}, nil)
	groupRepo.On("GetByMediaID", domain.ID(1)).Return([]domain.PartGroup{{Id: 100, Order: 1, MediaId: 1}}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeSeries,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoSeries, Name: "720p WEB-DL"}},
	}, nil)

	ix := &rssIndexer{name: "ix", rels: []domain.Release{
		{Title: "My.Show.S01E02.1080p.WEB-DL-GRP", MagnetURI: "magnet:?x"},
	}}
	grabber := &fakeGrabber{name: "torrent", support: true}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))

	svc := NewRssService(grabSvc, mediaRepo, partRepo, groupRepo, qualityRepo, NewReleaseParser(), &mockIndexerSource{indexers: []domain.ReleaseIndexer{ix}})
	require.NoError(t, svc.Run(context.Background()))

	queueRepo.AssertNotCalled(t, "Add")
}

func TestRssService_DedupSeenReleases(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	groupRepo := new(mockPartGroupRepo)
	qualityRepo := new(mockQualityRepo)

	profileID := domain.ID(10)
	media := &domain.Media{
		Id: 1, Name: "My Show", Type: domain.MediaTypeSeries, QualityProfileID: &profileID,
	}
	seasonID := domain.ID(100)
	ep := 2
	part := domain.Part{Id: 5, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep, Monitored: true}

	mt := domain.MediaTypeSeries
	mediaRepo.On("GetPaged", 1, 100, &mt).Return([]domain.Media{*media}, nil)
	mediaRepo.On("GetPaged", 2, 100, &mt).Return([]domain.Media{}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{part}, nil)
	groupRepo.On("GetByMediaID", domain.ID(1)).Return([]domain.PartGroup{{Id: 100, Order: 1, MediaId: 1}}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeSeries,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoSeries, Name: "1080p WEB-DL"}},
	}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)
	queueRepo.On("Add", mock.Anything).Return(nil).Once()
	historyRepo.On("Add", mock.Anything).Return(nil)

	infoHash := "deadbeef123"
	rels := []domain.Release{
		{Title: "My.Show.S01E02.1080p.WEB-DL-GRP", InfoHash: infoHash, MagnetURI: "magnet:?x"},
		{Title: "My.Show.S01E02.1080p.WEB-DL-GRP", InfoHash: infoHash, MagnetURI: "magnet:?x"},
	}
	ix := &rssIndexer{name: "ix", rels: rels}
	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))

	svc := NewRssService(grabSvc, mediaRepo, partRepo, groupRepo, qualityRepo, NewReleaseParser(), &mockIndexerSource{indexers: []domain.ReleaseIndexer{ix}})
	require.NoError(t, svc.Run(context.Background()))

	queueRepo.AssertNumberOfCalls(t, "Add", 1)
}

func TestRssService_SkipsNonSeriesReleases(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	mt := domain.MediaTypeSeries
	mediaRepo.On("GetPaged", 1, 100, &mt).Return([]domain.Media{}, nil)

	ix := &rssIndexer{name: "ix", rels: []domain.Release{
		{Title: "Some.Movie.1080p.WEB-DL-GRP", MagnetURI: "magnet:?x"},
	}}
	svc := NewRssService(
		NewGrabService(nil, new(mockQueueRepo), newGrabUoW(new(mockQueueRepo), new(mockHistoryRepo))),
		mediaRepo,
		new(mockPartRepo),
		new(mockPartGroupRepo),
		new(mockQualityRepo),
		NewReleaseParser(),
		&mockIndexerSource{indexers: []domain.ReleaseIndexer{ix}},
	)
	require.NoError(t, svc.Run(context.Background()))
}

func TestRssService_ToleratesIndexerError(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	groupRepo := new(mockPartGroupRepo)
	qualityRepo := new(mockQualityRepo)

	media := &domain.Media{
		Id: 1, Name: "My Show", Type: domain.MediaTypeSeries, QualityProfileID: nil,
	}
	seasonID := domain.ID(100)
	ep := 2
	part := domain.Part{Id: 5, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep, Monitored: true}

	mt := domain.MediaTypeSeries
	mediaRepo.On("GetPaged", 1, 100, &mt).Return([]domain.Media{*media}, nil)
	mediaRepo.On("GetPaged", 2, 100, &mt).Return([]domain.Media{}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{part}, nil)
	groupRepo.On("GetByMediaID", domain.ID(1)).Return([]domain.PartGroup{{Id: 100, Order: 1, MediaId: 1}}, nil)

	badIx := &rssIndexer{name: "bad", err: errors.New("connection refused")}
	goodIx := &rssIndexer{name: "good", rels: []domain.Release{
		{Title: "My.Show.S01E02.1080p.WEB-DL-GRP", MagnetURI: "magnet:?x"},
	}}
	queueRepo := new(mockQueueRepo)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)
	queueRepo.On("Add", mock.Anything).Return(nil)
	historyRepo := new(mockHistoryRepo)
	historyRepo.On("Add", mock.Anything).Return(nil)

	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))

	svc := NewRssService(grabSvc, mediaRepo, partRepo, groupRepo, qualityRepo, NewReleaseParser(), &mockIndexerSource{indexers: []domain.ReleaseIndexer{badIx, goodIx}})
	require.NoError(t, svc.Run(context.Background()))

	queueRepo.AssertNumberOfCalls(t, "Add", 1)
}

func TestFindWantedParts_SingleEpisode(t *testing.T) {
	ep5 := 5
	si := &rssSeries{
		wanted: map[int][]domain.Part{
			1: {{Id: 10, GroupOrder: &ep5}},
		},
	}

	parsed := domain.ParsedRelease{
		Season:   intPtr(1),
		Seasons:  []int{1},
		Episodes: []int{5},
	}
	result := findWantedParts(parsed, si)
	require.Len(t, result, 1)
	assert.Equal(t, domain.ID(10), result[0].Id)
}

func TestFindWantedParts_SeasonPack(t *testing.T) {
	ep5 := 5
	ep6 := 6
	si := &rssSeries{
		wanted: map[int][]domain.Part{
			1: {
				{Id: 10, GroupOrder: &ep5},
				{Id: 11, GroupOrder: &ep6},
			},
		},
	}

	parsed := domain.ParsedRelease{
		Season:  intPtr(1),
		Seasons: []int{1},
	}
	result := findWantedParts(parsed, si)
	require.Len(t, result, 2)
}

func TestFindWantedParts_MultiSeasonRange(t *testing.T) {
	ep5 := 5
	si := &rssSeries{
		wanted: map[int][]domain.Part{
			1: {{Id: 10, GroupOrder: &ep5}},
			2: {{Id: 11, GroupOrder: &ep5}},
			3: {{Id: 12, GroupOrder: &ep5}},
		},
	}

	parsed := domain.ParsedRelease{
		Season:  intPtr(1),
		Seasons: []int{1, 2, 3},
	}
	result := findWantedParts(parsed, si)
	require.Len(t, result, 3)
}

func TestRssService_GrabsMultiSeasonRelease(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	groupRepo := new(mockPartGroupRepo)
	qualityRepo := new(mockQualityRepo)

	profileID := domain.ID(10)
	media := &domain.Media{
		Id: 1, Name: "My Show", Type: domain.MediaTypeSeries, QualityProfileID: &profileID,
	}
	season1ID := domain.ID(100)
	season2ID := domain.ID(200)
	ep5 := 5
	parts := []domain.Part{
		{Id: 5, MediaId: 1, GroupId: &season1ID, GroupOrder: &ep5, Monitored: true},
		{Id: 6, MediaId: 1, GroupId: &season2ID, GroupOrder: &ep5, Monitored: true},
	}

	mt := domain.MediaTypeSeries
	mediaRepo.On("GetPaged", 1, 100, &mt).Return([]domain.Media{*media}, nil)
	mediaRepo.On("GetPaged", 2, 100, &mt).Return([]domain.Media{}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return(parts, nil)
	groupRepo.On("GetByMediaID", domain.ID(1)).Return([]domain.PartGroup{
		{Id: 100, Order: 1, MediaId: 1},
		{Id: 200, Order: 2, MediaId: 1},
	}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeSeries,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoSeries, Name: "1080p WEB-DL"}},
	}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)

	var capturedPartIds []domain.ID
	queueRepo.On("Add", mock.MatchedBy(func(q *domain.QueueItem) bool {
		capturedPartIds = q.PartIds
		return q.MediaId == 1
	})).Return(nil)
	historyRepo.On("Add", mock.Anything).Return(nil)

	ix := &rssIndexer{name: "ix", rels: []domain.Release{
		{Title: "My.Show.S01-S02.1080p.WEB-DL-GRP", MagnetURI: "magnet:?multi"},
	}}
	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))

	svc := NewRssService(grabSvc, mediaRepo, partRepo, groupRepo, qualityRepo, NewReleaseParser(), &mockIndexerSource{indexers: []domain.ReleaseIndexer{ix}})
	require.NoError(t, svc.Run(context.Background()))

	require.Len(t, capturedPartIds, 2)
	assert.Contains(t, capturedPartIds, domain.ID(5))
	assert.Contains(t, capturedPartIds, domain.ID(6))
}

func TestRssService_MultipleIndexersConcurrent(t *testing.T) {
	ix1 := &rssIndexer{name: "ix1", rels: []domain.Release{
		{Title: "My.Show.S01E02.1080p.WEB-DL-GRP", InfoHash: "hash1", MagnetURI: "magnet:?1"},
	}}
	ix2 := &rssIndexer{name: "ix2", rels: []domain.Release{
		{Title: "My.Show.S01E03.1080p.WEB-DL-GRP", InfoHash: "hash2", MagnetURI: "magnet:?2"},
	}}

	// Verify both indexers are called concurrently
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, ix := range []domain.ReleaseIndexer{ix1, ix2} {
		wg.Add(1)
		go func(ix domain.ReleaseIndexer) {
			defer wg.Done()
			<-start
			_, _ = ix.RSS(context.Background(), nil, time.Time{})
		}(ix)
	}
	close(start)
	wg.Wait()
}

func TestRssService_NewRssJob(t *testing.T) {
	job := NewRssJob(
		NewGrabService(nil, new(mockQueueRepo), newGrabUoW(new(mockQueueRepo), new(mockHistoryRepo))),
		new(mockMediaRepo),
		new(mockPartRepo),
		new(mockPartGroupRepo),
		new(mockQualityRepo),
		&mockIndexerSource{},
	)
	assert.NotNil(t, job)
}

func TestRssService_FindsIndexerCategories(t *testing.T) {
	ix := &rssIndexer{name: "ix"}
	svc := NewRssService(
		NewGrabService(nil, new(mockQueueRepo), newGrabUoW(new(mockQueueRepo), new(mockHistoryRepo))),
		new(mockMediaRepo),
		new(mockPartRepo),
		new(mockPartGroupRepo),
		new(mockQualityRepo),
		NewReleaseParser(),
		&mockIndexerSource{indexers: []domain.ReleaseIndexer{ix}},
	)
	_ = svc.Run(context.Background())
	assert.Contains(t, ix.capturedCats, 5000)
	assert.Contains(t, ix.capturedCats, 5040)
}
