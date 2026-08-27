package application

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stersh.ru/mediator/domain"
)

type fakeIndexer struct {
	name string
	rels []domain.Release
	err  error
}

func (f *fakeIndexer) Name() string { return f.name }

func (f *fakeIndexer) Search(ctx context.Context, query string, cats []int) ([]domain.Release, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rels, nil
}

func (f *fakeIndexer) RSS(ctx context.Context, cats []int, since time.Time) ([]domain.Release, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rels, nil
}

type fakeSource struct{ indexers []domain.ReleaseIndexer }

func (s *fakeSource) Active() []domain.ReleaseIndexer { return s.indexers }

func newSvc(indexers ...domain.ReleaseIndexer) *ReleaseService {
	return NewReleaseService(&fakeSource{indexers: indexers}, NewReleaseParser())
}

func TestReleaseService_Search_FilterAndSort(t *testing.T) {
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "a", rels: []domain.Release{
			{Title: "Movie.1999.1080p.WEB-DL-GRP", Seeders: 10},
			{Title: "Movie.1999.720p.WEBRip-X", Seeders: 100},
			{Title: "Movie.1999.480p.CAM-LOL", Seeders: 5},
		}},
		&fakeIndexer{name: "b", rels: []domain.Release{
			{Title: "Movie.1999.2160p.BluRay-TOP", Seeders: 3},
		}},
	}
	svc := newSvc(indexers...)

	profile := &domain.QualityProfile{
		Type:    domain.MediaTypeMovie,
		Allowed: []domain.Quality{{Kind: domain.QualityKindVideoMovie, Name: "1080p WEB-DL"}, {Kind: domain.QualityKindVideoMovie, Name: "2160p BluRay"}},
	}

	scored, err := svc.Search(context.Background(), "movie", domain.MediaTypeMovie, profile, nil)
	require.NoError(t, err)
	require.Len(t, scored, 2)

	assert.Equal(t, "2160p BluRay", scored[0].Parsed.Quality.Name)
	assert.Equal(t, "1080p WEB-DL", scored[1].Parsed.Quality.Name)
	assert.Equal(t, "Movie.1999.2160p.BluRay-TOP", scored[0].Release.Title)
}

func TestReleaseService_Search_NoProfileKeepsAll(t *testing.T) {
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "a", rels: []domain.Release{
			{Title: "Movie.480p.CAM"},
			{Title: "Movie.1080p.WEB-DL"},
		}},
	}
	svc := newSvc(indexers...)

	scored, err := svc.Search(context.Background(), "movie", domain.MediaTypeMovie, nil, nil)
	require.NoError(t, err)
	assert.Len(t, scored, 2)
}

func TestReleaseService_Search_ToleratesOneIndexerError(t *testing.T) {
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "bad", err: errors.New("down")},
		&fakeIndexer{name: "ok", rels: []domain.Release{{Title: "Movie.1080p.WEB-DL", Indexer: "ok"}}},
	}
	svc := newSvc(indexers...)

	scored, err := svc.Search(context.Background(), "movie", domain.MediaTypeMovie, nil, nil)
	require.NoError(t, err)
	require.Len(t, scored, 1)
	assert.Equal(t, "ok", scored[0].Release.Indexer)
}

func TestReleaseService_Search_AllIndexersFail(t *testing.T) {
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "a", err: errors.New("down")},
		&fakeIndexer{name: "b", err: errors.New("down")},
	}
	svc := newSvc(indexers...)

	_, err := svc.Search(context.Background(), "movie", domain.MediaTypeMovie, nil, nil)
	assert.Error(t, err)
}

func TestReleaseService_Search_SeriesTargetFilters(t *testing.T) {
	ep2 := 2
	capturedCats := make([][]int, 0)
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "a", rels: []domain.Release{
			{Title: "Show.S01E02.1080p.WEB-DL-GRP", Seeders: 10},
			{Title: "Show.S01E03.1080p.WEB-DL-GRP", Seeders: 20},
			{Title: "Show.S01.Complete.1080p.WEB-DL-GRP", Seeders: 5},
			{Title: "Show.S02E02.1080p.WEB-DL-GRP", Seeders: 30},
		}},
		&catCapturingIndexer{name: "b", cats: &capturedCats},
	}
	svc := newSvc(indexers...)

	scored, err := svc.Search(context.Background(), "Show", domain.MediaTypeSeries, nil, &SeriesTarget{Season: 1, Episode: &ep2})
	require.NoError(t, err)

	titles := make([]string, 0, len(scored))
	for _, s := range scored {
		titles = append(titles, s.Release.Title)
	}
	sort.Strings(titles)
	assert.Equal(t, []string{"Show.S01.Complete.1080p.WEB-DL-GRP", "Show.S01E02.1080p.WEB-DL-GRP"}, titles)

	require.NotEmpty(t, capturedCats)
	assert.Contains(t, capturedCats[0], 5000)
}

func TestReleaseService_Search_DropsIrrelevantTitle(t *testing.T) {
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "a", rels: []domain.Release{
			{Title: "Человек-паук Новый день 2026 1080p.WEB-DL-GRP", Seeders: 10},
			{Title: "Прощай жизнь дракона Здравствуй жизнь человека E01-E12 1080p.WEB-DL-X", Seeders: 100},
		}},
	}
	svc := newSvc(indexers...)

	scored, err := svc.Search(context.Background(), "Человек-паук: Новый день", domain.MediaTypeMovie, nil, nil)
	require.NoError(t, err)
	require.Len(t, scored, 1)
	assert.Contains(t, scored[0].Release.Title, "Человек-паук")
}

func TestReleaseService_Search_KeepsCrossLanguageTitle(t *testing.T) {
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "a", rels: []domain.Release{
			{Title: "A Knight of the Seven Kingdoms S01 2026 1080p.WEB-DL-GRP", Seeders: 10},
			{Title: "Прощай жизнь дракона E01-E12 1080p.WEB-DL-X", Seeders: 5},
		}},
	}
	svc := newSvc(indexers...)

	scored, err := svc.Search(context.Background(), "A Knight of the Seven Kingdoms", domain.MediaTypeSeries, nil, &SeriesTarget{Season: 1}, "Рыцарь Семи Королевств", "A Knight of the Seven Kingdoms")
	require.NoError(t, err)
	require.Len(t, scored, 1)
	assert.Contains(t, scored[0].Release.Title, "Knight")
}

func TestReleaseService_Search_DropsCrossLanguageWithoutOriginal(t *testing.T) {
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "a", rels: []domain.Release{
			{Title: "A Knight of the Seven Kingdoms S01 2026 1080p.WEB-DL-GRP", Seeders: 10},
			{Title: "Прощай жизнь дракона E01-E12 1080p.WEB-DL-X", Seeders: 5},
		}},
	}
	svc := newSvc(indexers...)

	scored, err := svc.Search(context.Background(), "Рыцарь Семи Королевств", domain.MediaTypeSeries, nil, &SeriesTarget{Season: 1})
	require.NoError(t, err)
	assert.Empty(t, scored)
}

func TestReleaseService_Search_DropsWrongShowWithOriginal(t *testing.T) {
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "a", rels: []domain.Release{
			{Title: "The Good Doctor - S3 - rus 1080p WEBDL (LostFilm)", Seeders: 50},
			{Title: "Dr. Stone - S01E01 - 1080p WEB-DL-GRP", Seeders: 10},
		}},
	}
	svc := newSvc(indexers...)

	scored, err := svc.Search(context.Background(), "Dr. Stone", domain.MediaTypeSeries, nil, &SeriesTarget{Season: 1}, "Доктор Стоун", "Dr. Stone")
	require.NoError(t, err)
	require.Len(t, scored, 1)
	assert.Contains(t, scored[0].Release.Title, "Dr. Stone")
}

func TestReleaseService_Search_DropsShortTitleSubsetFalsePositive(t *testing.T) {
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "a", rels: []domain.Release{
			{Title: "The Navigator A Mediaeval Odyssey 1988 VO Blu-Ray Remux 1080p - RUSSIAN", Seeders: 50},
			{Title: "The.Odyssey.2026.1080p.BluRay.REMUX-GROUP", Seeders: 10},
		}},
	}
	svc := newSvc(indexers...)

	scored, err := svc.Search(context.Background(), "The Odyssey", domain.MediaTypeMovie, nil, nil, "Одиссей", "The Odyssey")
	require.NoError(t, err)
	require.Len(t, scored, 1)
	assert.Contains(t, scored[0].Release.Title, "The.Odyssey.2026")
}

func TestReleaseService_Search_QueriesAllTitles(t *testing.T) {
	var queries []string
	ix := &queryCapturingIndexer{
		name:    "a",
		queries: &queries,
		relsByQuery: map[string][]domain.Release{
			"Доктор Стоун S01": {{Title: "Доктор Стоун S01 1080p WEB-DL-RUS", MagnetURI: "magnet:?xt=urn:btih:aaa", Seeders: 20}},
			"Dr. Stone S01":    {{Title: "Dr. Stone S01 1080p WEB-DL-GRP", MagnetURI: "magnet:?xt=urn:btih:bbb", Seeders: 10}},
		},
	}
	svc := newSvc(ix)

	scored, err := svc.Search(context.Background(), "Dr. Stone", domain.MediaTypeSeries, nil, &SeriesTarget{Season: 1}, "Доктор Стоун", "Dr. Stone")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Доктор Стоун", "Доктор Стоун S01", "Dr. Stone", "Dr. Stone S01"}, queries)
	require.Len(t, scored, 2)
	titles := []string{scored[0].Release.Title, scored[1].Release.Title}
	assert.Contains(t, titles, "Доктор Стоун S01 1080p WEB-DL-RUS")
	assert.Contains(t, titles, "Dr. Stone S01 1080p WEB-DL-GRP")
}

func TestReleaseService_Search_DedupesDuplicateIdentity(t *testing.T) {
	same := domain.Release{Title: "Movie.1080p.WEB-DL-GRP", MagnetURI: "magnet:?xt=urn:btih:abc", Seeders: 5}
	better := domain.Release{Title: "Movie.1080p.WEB-DL-GRP", MagnetURI: "magnet:?xt=urn:btih:abc", Seeders: 40}
	ix := &queryCapturingIndexer{
		name: "a",
		relsByQuery: map[string][]domain.Release{
			"Movie WEB": {same},
			"Movie":     {better},
		},
	}
	svc := newSvc(ix)

	scored, err := svc.Search(context.Background(), "Movie", domain.MediaTypeMovie, nil, nil, "Movie WEB", "Movie")
	require.NoError(t, err)
	require.Len(t, scored, 1)
	assert.Equal(t, 40, scored[0].Release.Seeders)
}

func TestMediaMatchTitles(t *testing.T) {
	assert.Equal(t, []string{"Интерстеллар", "Interstellar"}, MediaMatchTitles(domain.Media{
		Name:         "Интерстеллар",
		OriginalName: "Interstellar",
	}))
	assert.Equal(t, []string{"Interstellar"}, MediaMatchTitles(domain.Media{
		Name:         "Interstellar",
		OriginalName: "Interstellar",
	}))
	assert.Equal(t, []string{"Interstellar"}, MediaMatchTitles(domain.Media{
		Name:         "Interstellar",
		OriginalName: "interstellar",
	}))
}

func TestReleaseService_Search_MovieCategories(t *testing.T) {
	capturedCats := make([][]int, 0)
	indexers := []domain.ReleaseIndexer{
		&catCapturingIndexer{name: "b", cats: &capturedCats},
	}
	svc := newSvc(indexers...)

	_, _ = svc.Search(context.Background(), "movie", domain.MediaTypeMovie, nil, nil)
	require.NotEmpty(t, capturedCats)
	assert.Contains(t, capturedCats[0], 2000)
}

func TestReleaseService_Search_BookCategories(t *testing.T) {
	capturedCats := make([][]int, 0)
	indexers := []domain.ReleaseIndexer{
		&catCapturingIndexer{name: "b", cats: &capturedCats},
	}
	svc := newSvc(indexers...)

	_, _ = svc.Search(context.Background(), "book", domain.MediaTypeBook, nil, nil)
	require.NotEmpty(t, capturedCats)
	assert.Contains(t, capturedCats[0], 7000)
	assert.Contains(t, capturedCats[0], 7020)
	assert.Contains(t, capturedCats[0], 7030)
}

type catCapturingIndexer struct {
	name string
	cats *[][]int
}

func (c *catCapturingIndexer) Name() string { return c.name }

func (c *catCapturingIndexer) Search(ctx context.Context, query string, cats []int) ([]domain.Release, error) {
	*c.cats = append(*c.cats, cats)
	return nil, nil
}

func (c *catCapturingIndexer) RSS(ctx context.Context, cats []int, since time.Time) ([]domain.Release, error) {
	return nil, nil
}

type queryCapturingIndexer struct {
	mu          sync.Mutex
	name        string
	queries     *[]string
	relsByQuery map[string][]domain.Release
}

func (c *queryCapturingIndexer) Name() string { return c.name }

func (c *queryCapturingIndexer) Search(ctx context.Context, query string, cats []int) ([]domain.Release, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.queries != nil {
		*c.queries = append(*c.queries, query)
	}
	if c.relsByQuery != nil {
		return c.relsByQuery[query], nil
	}
	return nil, nil
}

func (c *queryCapturingIndexer) RSS(ctx context.Context, cats []int, since time.Time) ([]domain.Release, error) {
	return nil, nil
}

func TestMatchesTarget_MultiSeasonAndComplete(t *testing.T) {
	ep3 := 3
	intVal := func(v int) *int { return &v }

	tests := []struct {
		name   string
		parsed domain.ParsedRelease
		target SeriesTarget
		want   bool
	}{
		{
			name:   "multi-season covers target season",
			parsed: domain.ParsedRelease{Season: intVal(1), Seasons: []int{1, 2, 3, 4, 5}},
			target: SeriesTarget{Season: 3, Episode: &ep3},
			want:   true,
		},
		{
			name:   "multi-season misses target season",
			parsed: domain.ParsedRelease{Season: intVal(1), Seasons: []int{1, 2}},
			target: SeriesTarget{Season: 3, Episode: &ep3},
			want:   false,
		},
		{
			name:   "complete series covers any episode",
			parsed: domain.ParsedRelease{Complete: true},
			target: SeriesTarget{Season: 5, Episode: &ep3},
			want:   true,
		},
		{
			name:   "season pack covers specific episode",
			parsed: domain.ParsedRelease{Season: intVal(1), Seasons: []int{1}},
			target: SeriesTarget{Season: 1, Episode: &ep3},
			want:   true,
		},
		{
			name:   "single episode exact match",
			parsed: domain.ParsedRelease{Season: intVal(1), Seasons: []int{1}, Episodes: []int{3}},
			target: SeriesTarget{Season: 1, Episode: &ep3},
			want:   true,
		},
		{
			name:   "single episode different number",
			parsed: domain.ParsedRelease{Season: intVal(1), Seasons: []int{1}, Episodes: []int{5}},
			target: SeriesTarget{Season: 1, Episode: &ep3},
			want:   false,
		},
		{
			name:   "season-only target accepts any episode of season",
			parsed: domain.ParsedRelease{Season: intVal(2), Seasons: []int{2}, Episodes: []int{7}},
			target: SeriesTarget{Season: 2},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchesTarget(tt.parsed, tt.target))
		})
	}
}

func TestReleaseService_Search_PrefersCompleteSeriesPack(t *testing.T) {
	indexers := []domain.ReleaseIndexer{
		&fakeIndexer{name: "a", rels: []domain.Release{
			{Title: "Show - S1E1-20 - 1080p WEB-DL-GRP", Seeders: 80},
			{Title: "Show - S1-3E1-70 - 1080p WEB-DL-GRP", Seeders: 5},
			{Title: "Show - S01E01 - 1080p WEB-DL-GRP", Seeders: 200},
		}},
	}
	svc := newSvc(indexers...)

	scored, err := svc.Search(context.Background(), "Show", domain.MediaTypeSeries, nil, &SeriesTarget{Season: 1})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(scored), 2)
	assert.Contains(t, scored[0].Release.Title, "S1-3E1-70")
	assert.Contains(t, scored[1].Release.Title, "S1E1-20")
}

func TestReleaseService_Search_FindsUnscopedCompletePack(t *testing.T) {
	ix := &queryCapturingIndexer{
		name: "a",
		relsByQuery: map[string][]domain.Release{
			"Show S01": {{Title: "Show - S1E1-20 - 1080p WEB-DL-GRP", Seeders: 80}},
			"Show":     {{Title: "Show - S1-3E1-70 - 1080p WEB-DL-GRP", Seeders: 5}},
		},
	}
	svc := newSvc(ix)

	scored, err := svc.Search(context.Background(), "Show", domain.MediaTypeSeries, nil, &SeriesTarget{Season: 1})
	require.NoError(t, err)
	require.Len(t, scored, 2)
	assert.Contains(t, scored[0].Release.Title, "S1-3E1-70")
	assert.Contains(t, scored[1].Release.Title, "S1E1-20")
}

func TestPreferRelease_SeriesCoverage(t *testing.T) {
	q1080 := domain.Quality{Kind: domain.QualityKindVideoSeries, Name: "1080p WEB-DL"}
	q720 := domain.Quality{Kind: domain.QualityKindVideoSeries, Name: "720p WEB-DL"}

	full := ScoredRelease{
		Release: domain.Release{Title: "S1-3E1-70", Seeders: 5},
		Parsed:  domain.ParsedRelease{Quality: q1080, Season: intVal(1), Seasons: []int{1, 2, 3}, Episodes: epRange(1, 70)},
	}
	season := ScoredRelease{
		Release: domain.Release{Title: "S1E1-20", Seeders: 80},
		Parsed:  domain.ParsedRelease{Quality: q1080, Season: intVal(1), Seasons: []int{1}, Episodes: epRange(1, 20)},
	}
	assert.True(t, preferRelease(full, season, true))
	assert.False(t, preferRelease(season, full, true))

	full720 := ScoredRelease{
		Release: domain.Release{Title: "S1-3E1-70 720p", Seeders: 5},
		Parsed:  domain.ParsedRelease{Quality: q720, Season: intVal(1), Seasons: []int{1, 2, 3}, Episodes: epRange(1, 70)},
	}
	assert.True(t, preferRelease(full720, season, true))

	fullBetter := ScoredRelease{
		Release: domain.Release{Title: "S1-3E1-70 1080p", Seeders: 1},
		Parsed:  domain.ParsedRelease{Quality: q1080, Season: intVal(1), Seasons: []int{1, 2, 3}, Episodes: epRange(1, 70)},
	}
	fullWorseQ := ScoredRelease{
		Release: domain.Release{Title: "S1-3E1-70 720p more seeders", Seeders: 50},
		Parsed:  domain.ParsedRelease{Quality: q720, Season: intVal(1), Seasons: []int{1, 2, 3}, Episodes: epRange(1, 70)},
	}
	assert.True(t, preferRelease(fullBetter, fullWorseQ, true))

	seasonPack := ScoredRelease{
		Release: domain.Release{Title: "S01 complete", Seeders: 1},
		Parsed:  domain.ParsedRelease{Quality: q1080, Season: intVal(1), Seasons: []int{1}},
	}
	partial := ScoredRelease{
		Release: domain.Release{Title: "S01E01-10", Seeders: 40},
		Parsed:  domain.ParsedRelease{Quality: q1080, Season: intVal(1), Seasons: []int{1}, Episodes: epRange(1, 10)},
	}
	assert.True(t, preferRelease(seasonPack, partial, true))

	complete := ScoredRelease{
		Release: domain.Release{Title: "Complete series", Seeders: 1},
		Parsed:  domain.ParsedRelease{Quality: q1080, Complete: true},
	}
	assert.True(t, preferRelease(complete, full, true))
}

func intVal(v int) *int { return &v }
