package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

func TestGrabMissingJob_Movie(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	qualityRepo := new(mockQualityRepo)
	groupRepo := new(mockPartGroupRepo)

	profileID := domain.ID(10)
	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{
		Id: 1, Name: "My Movie", Type: domain.MediaTypeMovie, QualityProfileID: &profileID,
	}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeMovie,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoMovie, Name: "1080p WEB-DL"}},
	}, nil)
	partRepo.On("GetWanted", 1, 100).Return([]domain.Part{{Id: 50, MediaId: 1, Monitored: true}}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)
	queueRepo.On("Add", mock.MatchedBy(func(q *domain.QueueItem) bool {
		return q.MediaId == 1 && q.GrabberName == "torrent"
	})).Return(nil)
	historyRepo.On("Add", mock.Anything).Return(nil)

	indexer := &fakeIndexer{name: "ix", rels: []domain.Release{
		{Title: "My.Movie.1080p.WEB-DL-GRP", MagnetURI: "magnet:?x", Seeders: 20},
	}}
	releaseSvc := NewReleaseService(&fakeSource{indexers: []domain.ReleaseIndexer{indexer}}, NewReleaseParser())

	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))

	mediaSvc := NewMediaService(mediaRepo, nil, nil, nil)
	partSvc := NewPartService(partRepo, mediaRepo, nil)

	job := NewGrabMissingJob(grabSvc, partSvc, mediaSvc, releaseSvc, qualityRepo, groupRepo)

	require.NoError(t, job(context.Background()))

	queueRepo.AssertExpectations(t)
	historyRepo.AssertExpectations(t)
}

func TestGrabMissingJob_MovieNoProfileSkips(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	qualityRepo := new(mockQualityRepo)
	groupRepo := new(mockPartGroupRepo)

	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{
		Id: 1, Name: "My Movie", Type: domain.MediaTypeMovie,
	}, nil)
	partRepo.On("GetWanted", 1, 100).Return([]domain.Part{{Id: 50, MediaId: 1, Monitored: true}}, nil)

	indexer := &fakeIndexer{name: "ix", rels: []domain.Release{{Title: "My.Movie.1080p.WEB-DL-GRP", MagnetURI: "magnet:?x"}}}
	releaseSvc := NewReleaseService(&fakeSource{indexers: []domain.ReleaseIndexer{indexer}}, NewReleaseParser())
	grabSvc := NewGrabService(nil, queueRepo, newGrabUoW(queueRepo, historyRepo))
	mediaSvc := NewMediaService(mediaRepo, nil, nil, nil)
	partSvc := NewPartService(partRepo, mediaRepo, nil)

	job := NewGrabMissingJob(grabSvc, partSvc, mediaSvc, releaseSvc, qualityRepo, groupRepo)

	require.NoError(t, job(context.Background()))
	queueRepo.AssertNotCalled(t, "Add")
}

func TestGrabMissingJob_SeriesPrefersSeasonPack(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	qualityRepo := new(mockQualityRepo)
	groupRepo := new(mockPartGroupRepo)

	profileID := domain.ID(10)
	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{
		Id: 1, Name: "My Show", Type: domain.MediaTypeSeries, QualityProfileID: &profileID,
	}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeSeries,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoSeries, Name: "1080p WEB-DL"}},
	}, nil)
	seasonID := domain.ID(100)
	ep5, ep6 := 5, 6
	partRepo.On("GetWanted", 1, 100).Return([]domain.Part{
		{Id: 5, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep5, Monitored: true},
		{Id: 6, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep6, Monitored: true},
	}, nil)
	groupRepo.On("GetByMediaID", domain.ID(1)).Return([]domain.PartGroup{{Id: 100, Order: 1, MediaId: 1}}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)

	var capturedPartIds []domain.ID
	queueRepo.On("Add", mock.MatchedBy(func(q *domain.QueueItem) bool {
		capturedPartIds = q.PartIds
		return q.MediaId == 1
	})).Return(nil)
	historyRepo.On("Add", mock.Anything).Return(nil)

	indexer := &fakeIndexer{name: "ix", rels: []domain.Release{
		{Title: "My.Show.S01E05.1080p.WEB-DL-X", MagnetURI: "magnet:?e5", Seeders: 50},
		{Title: "My.Show.S01.Complete.1080p.WEB-DL-GRP", MagnetURI: "magnet:?pack", Seeders: 5},
	}}
	releaseSvc := NewReleaseService(&fakeSource{indexers: []domain.ReleaseIndexer{indexer}}, NewReleaseParser())

	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))
	mediaSvc := NewMediaService(mediaRepo, nil, nil, nil)
	partSvc := NewPartService(partRepo, mediaRepo, nil)

	job := NewGrabMissingJob(grabSvc, partSvc, mediaSvc, releaseSvc, qualityRepo, groupRepo)

	require.NoError(t, job(context.Background()))

	require.Len(t, capturedPartIds, 2, "season pack should target both wanted episodes")
	assert.Contains(t, capturedPartIds, domain.ID(5))
	assert.Contains(t, capturedPartIds, domain.ID(6))
}

func TestGrabMissingJob_SeriesDedupSkipsActiveGrab(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	qualityRepo := new(mockQualityRepo)
	groupRepo := new(mockPartGroupRepo)

	profileID := domain.ID(10)
	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{
		Id: 1, Name: "My Show", Type: domain.MediaTypeSeries, QualityProfileID: &profileID,
	}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{Id: 10, Type: domain.MediaTypeSeries}, nil)
	seasonID := domain.ID(100)
	ep5 := 5
	partRepo.On("GetWanted", 1, 100).Return([]domain.Part{
		{Id: 5, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep5, Monitored: true},
	}, nil)
	groupRepo.On("GetByMediaID", domain.ID(1)).Return([]domain.PartGroup{{Id: 100, Order: 1, MediaId: 1}}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{{MediaId: 1, PartIds: []domain.ID{5}, State: domain.GrabRunning}}, nil)

	releaseSvc := NewReleaseService(&fakeSource{}, NewReleaseParser())
	grabSvc := NewGrabService(nil, queueRepo, newGrabUoW(queueRepo, historyRepo))
	mediaSvc := NewMediaService(mediaRepo, nil, nil, nil)
	partSvc := NewPartService(partRepo, mediaRepo, nil)

	job := NewGrabMissingJob(grabSvc, partSvc, mediaSvc, releaseSvc, qualityRepo, groupRepo)

	require.NoError(t, job(context.Background()))
	queueRepo.AssertNotCalled(t, "Add")
}

func TestHasActiveGrab(t *testing.T) {
	queueRepo := new(mockQueueRepo)
	queueRepo.On("ListActive").Return([]domain.QueueItem{
		{MediaId: 1, PartIds: []domain.ID{5, 6}},
		{MediaId: 2, PartIds: nil},
	}, nil)
	grabSvc := NewGrabService(nil, queueRepo, newGrabUoW(queueRepo, new(mockHistoryRepo)))

	assert.True(t, grabSvc.HasActiveGrab(1, 5), "part 5 of media 1 is active")
	assert.False(t, grabSvc.HasActiveGrab(1, 7), "part 7 of media 1 is not active")
	assert.True(t, grabSvc.HasActiveGrab(2, 99), "media 2 grab with empty PartIds covers any part")
	assert.False(t, grabSvc.HasActiveGrab(3, 1), "no grab for media 3")
}

func TestGrabMissingService_ProcessMedia_Series(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	qualityRepo := new(mockQualityRepo)
	groupRepo := new(mockPartGroupRepo)

	profileID := domain.ID(10)
	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{
		Id: 1, Name: "My Show", Type: domain.MediaTypeSeries, QualityProfileID: &profileID,
	}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeSeries,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoSeries, Name: "1080p WEB-DL"}},
	}, nil)

	seasonID := domain.ID(100)
	ep5, ep6 := 5, 6
	episodePath := "/existing/episode.mp4"
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{
		{Id: 5, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep5, Monitored: true, Path: nil},
		{Id: 6, MediaId: 1, GroupId: &seasonID, GroupOrder: &ep6, Monitored: true, Path: nil},
		{Id: 7, MediaId: 1, GroupId: &seasonID, GroupOrder: ptrInt(7), Monitored: false, Path: nil},
		{Id: 8, MediaId: 1, GroupId: &seasonID, GroupOrder: ptrInt(8), Monitored: true, Path: &episodePath},
	}, nil)
	groupRepo.On("GetByMediaID", domain.ID(1)).Return([]domain.PartGroup{{Id: 100, Order: 1, MediaId: 1}}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)

	var capturedPartIds []domain.ID
	queueRepo.On("Add", mock.MatchedBy(func(q *domain.QueueItem) bool {
		capturedPartIds = q.PartIds
		return q.MediaId == 1
	})).Return(nil)
	historyRepo.On("Add", mock.Anything).Return(nil)

	indexer := &fakeIndexer{name: "ix", rels: []domain.Release{
		{Title: "My.Show.S01.Complete.1080p.WEB-DL-GRP", MagnetURI: "magnet:?pack", Seeders: 5},
	}}
	releaseSvc := NewReleaseService(&fakeSource{indexers: []domain.ReleaseIndexer{indexer}}, NewReleaseParser())

	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))

	mediaSvc := NewMediaService(mediaRepo, nil, nil, nil)
	partSvc := NewPartService(partRepo, mediaRepo, nil)
	svc := NewGrabMissingService(grabSvc, partSvc, mediaSvc, releaseSvc, qualityRepo, groupRepo)

	count, err := svc.ProcessMedia(context.Background(), domain.ID(1))
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should grab one season pack")
	require.Len(t, capturedPartIds, 2, "season pack should target both wanted episodes (5 and 6)")
	assert.Contains(t, capturedPartIds, domain.ID(5))
	assert.Contains(t, capturedPartIds, domain.ID(6))
	assert.NotContains(t, capturedPartIds, domain.ID(7), "unmonitored part 7 should not be grabbed")
	assert.NotContains(t, capturedPartIds, domain.ID(8), "already imported part 8 should not be grabbed")

	queueRepo.AssertExpectations(t)
	partRepo.AssertExpectations(t)
	groupRepo.AssertExpectations(t)
}

func TestGrabMissingService_ProcessMedia_Album(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	qualityRepo := new(mockQualityRepo)
	groupRepo := new(mockPartGroupRepo)

	profileID := domain.ID(10)
	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{
		Id: 1, Name: "Queen - A Night at the Opera", Type: domain.MediaTypeMusicAlbum, QualityProfileID: &profileID,
	}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeMusicAlbum,
		Allowed: []domain.Quality{{Kind: domain.QualityKindAudio, Name: "flac"}},
	}, nil)

	t1, t2 := 1, 2
	imported := "/existing/track.flac"
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{
		{Id: 5, MediaId: 1, GroupOrder: &t1, Monitored: true, Path: nil},
		{Id: 6, MediaId: 1, GroupOrder: &t2, Monitored: true, Path: nil},
		{Id: 7, MediaId: 1, GroupOrder: ptrInt(3), Monitored: false, Path: nil},
		{Id: 8, MediaId: 1, GroupOrder: ptrInt(4), Monitored: true, Path: &imported},
	}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)

	var capturedPartIds []domain.ID
	queueRepo.On("Add", mock.MatchedBy(func(q *domain.QueueItem) bool {
		capturedPartIds = q.PartIds
		return q.MediaId == 1
	})).Return(nil)
	historyRepo.On("Add", mock.Anything).Return(nil)

	indexer := &fakeIndexer{name: "ix", rels: []domain.Release{
		{Title: "Queen.A.Night.at.the.Opera.FLAC", MagnetURI: "magnet:?album", Seeders: 10},
	}}
	releaseSvc := NewReleaseService(&fakeSource{indexers: []domain.ReleaseIndexer{indexer}}, NewReleaseParser())

	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))

	mediaSvc := NewMediaService(mediaRepo, nil, nil, nil)
	partSvc := NewPartService(partRepo, mediaRepo, nil)
	svc := NewGrabMissingService(grabSvc, partSvc, mediaSvc, releaseSvc, qualityRepo, groupRepo)

	count, err := svc.ProcessMedia(context.Background(), domain.ID(1))
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should submit one album grab")
	require.Len(t, capturedPartIds, 2, "album grab should target all wanted tracks")
	assert.Contains(t, capturedPartIds, domain.ID(5))
	assert.Contains(t, capturedPartIds, domain.ID(6))
	assert.NotContains(t, capturedPartIds, domain.ID(7))
	assert.NotContains(t, capturedPartIds, domain.ID(8))

	queueRepo.AssertExpectations(t)
}

func TestGrabMissingService_ProcessMedia_Book(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	qualityRepo := new(mockQualityRepo)
	groupRepo := new(mockPartGroupRepo)

	profileID := domain.ID(10)
	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{
		Id: 1, Name: "Dune", Type: domain.MediaTypeBook, ProviderID: "openlibrary", QualityProfileID: &profileID,
	}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeBook,
		Allowed: []domain.Quality{{Kind: domain.QualityKindBook, Name: "epub"}},
	}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{
		{Id: 50, MediaId: 1, Monitored: true},
	}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)
	queueRepo.On("Add", mock.MatchedBy(func(q *domain.QueueItem) bool {
		return q.MediaId == 1 && len(q.PartIds) == 1 && q.PartIds[0] == 50
	})).Return(nil)
	historyRepo.On("Add", mock.Anything).Return(nil)

	indexer := &fakeIndexer{name: "ix", rels: []domain.Release{
		{Title: "Dune.epub", MagnetURI: "magnet:?book", Seeders: 5},
	}}
	releaseSvc := NewReleaseService(&fakeSource{indexers: []domain.ReleaseIndexer{indexer}}, NewReleaseParser())
	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))
	mediaSvc := NewMediaService(mediaRepo, nil, nil, nil)
	partSvc := NewPartService(partRepo, mediaRepo, nil)
	svc := NewGrabMissingService(grabSvc, partSvc, mediaSvc, releaseSvc, qualityRepo, groupRepo)

	count, err := svc.ProcessMedia(context.Background(), domain.ID(1))
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	queueRepo.AssertExpectations(t)
}

func TestGrabMissingService_ProcessMedia_AuthorTodayFallsBackToTorrent(t *testing.T) {
	mediaRepo := new(mockMediaRepo)
	partRepo := new(mockPartRepo)
	queueRepo := new(mockQueueRepo)
	historyRepo := new(mockHistoryRepo)
	qualityRepo := new(mockQualityRepo)
	groupRepo := new(mockPartGroupRepo)

	profileID := domain.ID(10)
	mediaRepo.On("GetById", domain.ID(1)).Return(&domain.Media{
		Id: 1, Name: "Some Book", Type: domain.MediaTypeBook,
		ProviderID: string(domain.SourceAuthorToday), QualityProfileID: &profileID,
	}, nil)
	qualityRepo.On("GetById", profileID).Return(&domain.QualityProfile{
		Id: 10, Type: domain.MediaTypeBook,
		Allowed: []domain.Quality{{Kind: domain.QualityKindBook, Name: "epub"}},
	}, nil)
	partRepo.On("GetByMediaId", domain.ID(1)).Return([]domain.Part{
		{Id: 50, MediaId: 1, Monitored: true},
	}, nil)
	queueRepo.On("ListActive").Return([]domain.QueueItem{}, nil)
	queueRepo.On("Add", mock.MatchedBy(func(q *domain.QueueItem) bool {
		return q.MediaId == 1 && q.GrabberName == "torrent"
	})).Return(nil)
	historyRepo.On("Add", mock.Anything).Return(nil)

	indexer := &fakeIndexer{name: "ix", rels: []domain.Release{
		{Title: "Some.Book.epub", MagnetURI: "magnet:?book", Seeders: 3},
	}}
	releaseSvc := NewReleaseService(&fakeSource{indexers: []domain.ReleaseIndexer{indexer}}, NewReleaseParser())
	grabber := &fakeGrabber{name: "torrent", support: true, handle: domain.GrabHandle{GrabberName: "torrent", JobID: "j1"}}
	grabSvc := NewGrabService([]domain.Grabber{grabber}, queueRepo, newGrabUoW(queueRepo, historyRepo))
	mediaSvc := NewMediaService(mediaRepo, nil, nil, nil)
	partSvc := NewPartService(partRepo, mediaRepo, nil)
	svc := NewGrabMissingService(grabSvc, partSvc, mediaSvc, releaseSvc, qualityRepo, groupRepo)

	count, err := svc.ProcessMedia(context.Background(), domain.ID(1))
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	queueRepo.AssertExpectations(t)
}

func ptrInt(v int) *int { return &v }
