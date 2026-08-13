package application

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"stersh.ru/mediator/domain"
)

type ImportService struct {
	mediaRepo   domain.MediaRepository
	partRepo    domain.PartRepository
	groupRepo   domain.PartGroupRepository
	libraryRepo domain.LibraryRepository
	uow         domain.UnitOfWork
	fs          domain.FileService
	naming      *NamingService
	parser      *ReleaseParser
}

func NewImportService(
	mediaRepo domain.MediaRepository,
	partRepo domain.PartRepository,
	groupRepo domain.PartGroupRepository,
	libraryRepo domain.LibraryRepository,
	uow domain.UnitOfWork,
	fs domain.FileService,
	naming *NamingService,
	parser *ReleaseParser,
) *ImportService {
	return &ImportService{
		mediaRepo:   mediaRepo,
		partRepo:    partRepo,
		groupRepo:   groupRepo,
		libraryRepo: libraryRepo,
		uow:         uow,
		fs:          fs,
		naming:      naming,
		parser:      parser,
	}
}

func (s *ImportService) Import(ctx context.Context, item domain.QueueItem, status domain.GrabStatus) error {
	slog.Debug("import service: import called", "media_id", item.MediaId, "job_id", item.JobID, "output_files", status.OutputFiles)
	if len(status.OutputFiles) == 0 {
		slog.Warn("import service: no output files", "media_id", item.MediaId, "job_id", item.JobID)
		return fmt.Errorf("import: no output files")
	}

	media, err := s.mediaRepo.GetById(item.MediaId)
	if err != nil {
		slog.Warn("import service: media not found", "media_id", item.MediaId, "job_id", item.JobID, "err", err)
		return err
	}
	if media.LibraryID == nil {
		slog.Warn("import service: media has no library", "media_id", item.MediaId, "job_id", item.JobID)
		return domain.ErrLibraryRequired
	}
	library, err := s.libraryRepo.GetById(*media.LibraryID)
	if err != nil {
		slog.Warn("import service: library not found", "media_id", item.MediaId, "job_id", item.JobID, "library_id", *media.LibraryID, "err", err)
		return err
	}
	parts, err := s.partRepo.GetByMediaId(item.MediaId)
	if err != nil {
		slog.Warn("import service: parts fetch failed", "media_id", item.MediaId, "job_id", item.JobID, "err", err)
		return err
	}

	libPath := filepath.Clean(library.Path)
	switch media.Type {
	case domain.MediaTypeSeries:
		return s.importSeries(ctx, item, status, media, parts, libPath, library.Settings["season_format"])
	case domain.MediaTypeMusicAlbum:
		return s.importAlbum(ctx, item, status, media, parts, libPath)
	default:
		return s.importSingle(ctx, item, status, media, parts, libPath)
	}
}

func (s *ImportService) importSingle(ctx context.Context, item domain.QueueItem, status domain.GrabStatus, media *domain.Media, parts []domain.Part, libPath string) error {
	target := wantedPart(parts)
	if target == nil {
		slog.Warn("import service: no wanted part found", "media_id", item.MediaId, "job_id", item.JobID, "parts_count", len(parts))
		return domain.ErrPartNotFound
	}

	src := status.OutputFiles[0]
	parsed, _ := s.parser.Parse(item.ReleaseTitle, media.Type)
	folder := mediaFolder(media)
	dst := s.naming.Build(libPath, folder, media.Name, parsed.Quality.Name, filepath.Ext(src))
	slog.Debug("import service: moving file", "media_id", item.MediaId, "job_id", item.JobID, "src", src, "dst", dst)

	now := time.Now().UTC()
	if err := s.commitImport(ctx, item, []stagedImport{{part: target, src: src, dst: dst}}, now); err != nil {
		return err
	}
	slog.Info("import service: import completed", "media_id", item.MediaId, "job_id", item.JobID, "dst", dst)
	return nil
}

func (s *ImportService) importSeries(ctx context.Context, item domain.QueueItem, status domain.GrabStatus, media *domain.Media, parts []domain.Part, libPath, seasonFormat string) error {
	groups, err := s.groupRepo.GetByMediaID(media.Id)
	if err != nil {
		slog.Warn("import service: groups fetch failed", "media_id", item.MediaId, "err", err)
		return err
	}
	groupOrder := make(map[domain.ID]int, len(groups))
	for _, g := range groups {
		groupOrder[g.Id] = g.Order
	}
	elibigle := eligibleSet(item.PartIds)

	var stagedFiles []stagedImport
	consumed := make(map[domain.ID]bool)
	for _, src := range status.OutputFiles {
		fileParsed, _ := s.parser.Parse(filepath.Base(src), domain.MediaTypeSeries)
		target := matchEpisodePart(parts, groupOrder, fileParsed, elibigle, consumed)
		if target == nil {
			slog.Debug("import service: no matching episode part for file", "media_id", item.MediaId, "src", src)
			continue
		}
		relParsed, _ := s.parser.Parse(item.ReleaseTitle, domain.MediaTypeSeries)
		folder := mediaFolder(media)
		dst := s.naming.BuildEpisode(libPath, folder, media.Name, *fileParsed.Season, fileParsed.Episodes[0], target.Name, seasonFormat, relParsed.Quality.Name, filepath.Ext(src))
		slog.Debug("import service: staging episode move", "media_id", item.MediaId, "src", src, "dst", dst, "part_id", target.Id)
		consumed[target.Id] = true
		stagedFiles = append(stagedFiles, stagedImport{part: target, src: src, dst: dst})
	}

	if len(stagedFiles) == 0 {
		slog.Warn("import service: no episode files matched", "media_id", item.MediaId, "job_id", item.JobID)
		return domain.ErrPartNotFound
	}

	now := time.Now().UTC()
	if err := s.commitImport(ctx, item, stagedFiles, now); err != nil {
		return err
	}
	slog.Info("import service: series import completed", "media_id", item.MediaId, "job_id", item.JobID, "episodes", len(stagedFiles))
	return nil
}

func (s *ImportService) importAlbum(ctx context.Context, item domain.QueueItem, status domain.GrabStatus, media *domain.Media, parts []domain.Part, libPath string) error {
	eligible := eligibleSet(item.PartIds)
	wanted := wantedAlbumParts(parts, eligible)
	if len(wanted) == 0 {
		slog.Warn("import service: no wanted album tracks", "media_id", item.MediaId, "job_id", item.JobID)
		return domain.ErrPartNotFound
	}

	files := append([]string(nil), status.OutputFiles...)
	sort.Strings(files)

	assigned := matchAlbumFiles(files, wanted)
	artist, album := albumArtistTitle(media)
	relParsed, _ := s.parser.Parse(item.ReleaseTitle, domain.MediaTypeMusicAlbum)

	var stagedFiles []stagedImport
	for i, src := range files {
		target := assigned[i]
		if target == nil {
			slog.Debug("import service: no matching album track for file", "media_id", item.MediaId, "src", src)
			continue
		}
		trackNo := 0
		if target.GroupOrder != nil {
			trackNo = *target.GroupOrder
		}
		title := ""
		if target.Name != nil {
			title = *target.Name
		}
		dst := s.naming.BuildAlbum(libPath, artist, album, trackNo, title, relParsed.Quality.Name, filepath.Ext(src))
		slog.Debug("import service: staging album move", "media_id", item.MediaId, "src", src, "dst", dst, "part_id", target.Id)
		stagedFiles = append(stagedFiles, stagedImport{part: target, src: src, dst: dst})
	}

	if len(stagedFiles) == 0 {
		slog.Warn("import service: no album files matched", "media_id", item.MediaId, "job_id", item.JobID)
		return domain.ErrPartNotFound
	}

	now := time.Now().UTC()
	if err := s.commitImport(ctx, item, stagedFiles, now); err != nil {
		return err
	}
	slog.Info("import service: album import completed", "media_id", item.MediaId, "job_id", item.JobID, "tracks", len(stagedFiles))
	return nil
}

type stagedImport struct {
	part *domain.Part
	src  string
	dst  string
}

type fsMove struct {
	src string
	dst string
}

func (s *ImportService) commitImport(ctx context.Context, item domain.QueueItem, files []stagedImport, now time.Time) error {
	var moved []fsMove
	for _, f := range files {
		if err := s.fs.Move(f.src, f.dst); err != nil {
			slog.Warn("import service: move failed", "media_id", item.MediaId, "job_id", item.JobID, "src", f.src, "dst", f.dst, "err", err)
			s.rollbackMoves(item.MediaId, item.JobID, moved)
			return fmt.Errorf("import move failed: %w", err)
		}
		moved = append(moved, fsMove{src: f.src, dst: f.dst})
	}
	err := s.uow.Run(ctx, func(repos *domain.Repos) error {
		for _, f := range files {
			updated := *f.part
			path := f.dst
			updated.Path = &path
			if err := repos.Part.Update(&updated); err != nil {
				slog.Warn("import service: part update failed", "media_id", item.MediaId, "job_id", item.JobID, "part_id", f.part.Id, "err", err)
				return err
			}
			pid := f.part.Id
			if err := repos.History.Add(&domain.History{
				MediaId:      item.MediaId,
				PartId:       &pid,
				EventType:    domain.HistoryImported,
				ReleaseTitle: item.ReleaseTitle,
				Data:         f.dst,
				CreatedAt:    now,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.rollbackMoves(item.MediaId, item.JobID, moved)
		slog.Warn("import service: tx failed", "media_id", item.MediaId, "job_id", item.JobID, "err", err)
		return err
	}
	return nil
}

func (s *ImportService) rollbackMoves(mediaId domain.ID, jobID string, moves []fsMove) {
	for i := len(moves) - 1; i >= 0; i-- {
		m := moves[i]
		if err := s.fs.Move(m.dst, m.src); err != nil {
			slog.Error("import service: rollback move failed", "media_id", mediaId, "job_id", jobID, "src", m.dst, "dst", m.src, "err", err)
		}
	}
}

func wantedPart(parts []domain.Part) *domain.Part {
	for i := range parts {
		if parts[i].Monitored && parts[i].Path == nil {
			return &parts[i]
		}
	}
	return nil
}

func eligibleSet(partIds []domain.ID) map[domain.ID]bool {
	if len(partIds) == 0 {
		return nil
	}
	out := make(map[domain.ID]bool, len(partIds))
	for _, id := range partIds {
		out[id] = true
	}
	return out
}

func matchEpisodePart(parts []domain.Part, groupOrder map[domain.ID]int, parsed domain.ParsedRelease, eligible map[domain.ID]bool, consumed map[domain.ID]bool) *domain.Part {
	if parsed.Season == nil || len(parsed.Episodes) == 0 {
		return nil
	}
	season := *parsed.Season
	episode := parsed.Episodes[0]
	for i := range parts {
		p := &parts[i]
		if consumed[p.Id] {
			continue
		}
		if p.Path != nil {
			continue
		}
		if eligible != nil && !eligible[p.Id] {
			continue
		}
		if p.GroupId == nil {
			continue
		}
		gord, ok := groupOrder[*p.GroupId]
		if !ok || gord != season {
			continue
		}
		if p.GroupOrder == nil || *p.GroupOrder != episode {
			continue
		}
		return p
	}
	return nil
}

func mediaFolder(media *domain.Media) string {
	if media.Folder != nil && *media.Folder != "" {
		return *media.Folder
	}
	return media.Name
}

func albumArtistTitle(media *domain.Media) (artist, album string) {
	name := media.Name
	if media.Folder != nil && *media.Folder != "" {
		artist = *media.Folder
		if i := strings.Index(name, " - "); i >= 0 {
			album = strings.TrimSpace(name[i+3:])
		} else {
			album = name
		}
		return artist, album
	}
	if i := strings.Index(name, " - "); i >= 0 {
		return strings.TrimSpace(name[:i]), strings.TrimSpace(name[i+3:])
	}
	return "", name
}

func wantedAlbumParts(parts []domain.Part, eligible map[domain.ID]bool) []*domain.Part {
	var out []*domain.Part
	for i := range parts {
		p := &parts[i]
		if !p.Monitored || p.Path != nil {
			continue
		}
		if eligible != nil && !eligible[p.Id] {
			continue
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		oi, oj := 0, 0
		if out[i].GroupOrder != nil {
			oi = *out[i].GroupOrder
		}
		if out[j].GroupOrder != nil {
			oj = *out[j].GroupOrder
		}
		if oi != oj {
			return oi < oj
		}
		return out[i].Id < out[j].Id
	})
	return out
}

var reLeadingTrack = regexp.MustCompile(`(?i)^(\d{1,3})[\s._-]+`)

func extractTrackNumber(filename string) int {
	base := filepath.Base(filename)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	m := reLeadingTrack.FindStringSubmatch(base)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return n
}

func matchAlbumFiles(files []string, wanted []*domain.Part) []*domain.Part {
	assigned := make([]*domain.Part, len(files))
	consumed := make(map[domain.ID]bool)

	for i, src := range files {
		n := extractTrackNumber(src)
		if n == 0 {
			continue
		}
		for _, p := range wanted {
			if consumed[p.Id] {
				continue
			}
			if p.GroupOrder != nil && *p.GroupOrder == n {
				assigned[i] = p
				consumed[p.Id] = true
				break
			}
		}
	}

	for i, src := range files {
		if assigned[i] != nil {
			continue
		}
		base := strings.ToLower(filepath.Base(src))
		for _, p := range wanted {
			if consumed[p.Id] || p.Name == nil || *p.Name == "" {
				continue
			}
			if strings.Contains(base, strings.ToLower(*p.Name)) {
				assigned[i] = p
				consumed[p.Id] = true
				break
			}
		}
	}

	for i := range files {
		if assigned[i] != nil {
			continue
		}
		for _, p := range wanted {
			if consumed[p.Id] {
				continue
			}
			assigned[i] = p
			consumed[p.Id] = true
			break
		}
	}
	return assigned
}
