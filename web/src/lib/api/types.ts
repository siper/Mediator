export const MediaType = {
  Book: 0,
  Series: 1,
  Movie: 2,
  MusicAlbum: 3,
} as const;

export type MediaType = (typeof MediaType)[keyof typeof MediaType];

export const MEDIA_TYPE_SLUG: Record<MediaType, string> = {
  [MediaType.Book]: "books",
  [MediaType.Series]: "series",
  [MediaType.Movie]: "movies",
  [MediaType.MusicAlbum]: "music",
};

export const MEDIA_TYPE_KEY: Record<MediaType, string> = {
  [MediaType.Book]: "book",
  [MediaType.Series]: "series",
  [MediaType.Movie]: "movie",
  [MediaType.MusicAlbum]: "music",
};

export const MEDIA_TYPE_BADGE: Record<number, string> = {
  [MediaType.Movie]: "border-transparent bg-red-500/10 text-red-500",
  [MediaType.Series]: "border-transparent bg-blue-500/10 text-blue-500",
  [MediaType.Book]: "border-transparent bg-amber-500/10 text-amber-500",
  [MediaType.MusicAlbum]: "border-transparent bg-emerald-500/10 text-emerald-500",
};

export const typeOptions: { value: string; type: MediaType }[] = [
  { value: String(MediaType.Movie), type: MediaType.Movie },
  { value: String(MediaType.Series), type: MediaType.Series },
  { value: String(MediaType.MusicAlbum), type: MediaType.MusicAlbum },
];

export interface Media {
  Id: number;
  Name: string;
  OriginalName: string;
  Folder: string | null;
  Cover: string | null;
  Type: MediaType;
  LibraryID: number | null;
  ProviderID: string;
  ExternalID: string;
  Status: string;
  LastModified: string;
  QualityProfileID: number | null;
}

export interface Library {
  Id: number;
  Name: string;
  Path: string;
  Type: MediaType;
  Settings: Record<string, string>;
}

export interface Part {
  Id: number;
  Name: string | null;
  GroupOrder: number | null;
  GroupId: number | null;
  MediaId: number;
  Path: string | null;
  Monitored: boolean;
}

export interface WantedItem {
  Part: Part;
  Media: Media;
  Season: number | null;
}

export interface PartGroup {
  Id: number;
  Name: string;
  Order: number;
  MediaId: number;
}

export interface QueueItem {
  Id: number;
  MediaId: number;
  PartIds: number[] | null;
  GrabberName: string;
  JobId: string;
  DownloadId: string | null;
  ReleaseTitle: string;
  State: string;
  Progress: number;
  AddedAt: string;
}

export interface QueueListItem extends QueueItem {
  Media: Media;
  PartLabel?: string | null;
}

export interface History {
  Id: number;
  MediaId: number;
  PartId: number | null;
  EventType: string;
  ReleaseTitle: string;
  Data: string;
  CreatedAt: string;
}

export interface HistoryListItem extends History {
  Media: Media;
  PartLabel?: string | null;
}

export type MediaRequestStatus = "pending" | "approved" | "rejected" | "canceled";

export interface Setting {
  Key: string;
  Value: string;
}

export interface PublicSettings {
  requests_enabled?: string;
}

export interface MediaRequest {
  Id: number;
  UserId: number;
  Provider: string;
  ExternalID: string;
  Title: string;
  Cover: string;
  Type: MediaType;
  LibraryID: number;
  QualityProfileID: number | null;
  Folder: string | null;
  Status: MediaRequestStatus;
  CreatedAt: string;
  ApprovedAt: string | null;
  ApprovedBy: number | null;
  RejectedAt: string | null;
  RejectedBy: number | null;
  CanceledAt: string | null;
  Notes: string | null;
  MediaID: number | null;
}

export interface SearchResult {
  ProviderName: string;
  ExternalID: string;
  Title: string;
  Overview: string;
  CoverURL: string;
  MediaType: MediaType;
  Year: number | null;
}

export interface SearchPage {
  Results: SearchResult[];
  Page: number;
  Limit: number;
  HasMore: boolean;
}

export interface ParsedRelease {
  Quality: Quality;
  Source: string;
  Resolution: string;
  Codec: string;
  Year: number | null;
  Season: number | null;
  Episodes: number[];
  Group: string;
  IsRepack: boolean;
}

export interface ScoredRelease {
  Release: {
    Title: string;
    DownloadURL: string;
    MagnetURI: string;
    InfoHash: string;
    Size: number;
    Seeders: number;
    Indexer: string;
    PublishDate: string;
  };
  Parsed: ParsedRelease;
}

export interface Indexer {
  Id: number;
  Name: string;
  Type: string;
  Settings: Record<string, string>;
  Enabled: boolean;
}

export interface DownloadClient {
  Id: number;
  Name: string;
  Type: string;
  Settings: Record<string, string>;
  Enabled: boolean;
}

export interface Quality {
  Kind: string;
  Name: string;
}

export const QUALITY_KIND = {
  VideoMovie: "video_movie",
  VideoSeries: "video_series",
  Book: "book",
  Audio: "audio",
} as const;

export const VIDEO_QUALITIES_MOVIE: Quality[] = [
  "CAM", "TS", "TC", "SCR",
  "DVD",
  "480p", "480p WEB-DL", "480p BluRay",
  "720p", "720p HDTV", "720p WEBRip", "720p WEB-DL", "720p BluRay",
  "1080p", "1080p HDTV", "1080p WEBRip", "1080p WEB-DL", "1080p BluRay",
  "2160p", "2160p WEBRip", "2160p WEB-DL", "2160p BluRay",
].map((name) => ({ Kind: QUALITY_KIND.VideoMovie, Name: name }));

export const VIDEO_QUALITIES_SERIES: Quality[] = [
  "720p", "720p HDTV", "720p WEBRip", "720p WEB-DL", "720p BluRay",
  "1080p", "1080p HDTV", "1080p WEBRip", "1080p WEB-DL", "1080p BluRay",
  "2160p", "2160p WEBRip", "2160p WEB-DL", "2160p BluRay",
].map((name) => ({ Kind: QUALITY_KIND.VideoSeries, Name: name }));

export const AUDIO_QUALITIES: Quality[] = [
  "mp3", "aac", "ogg", "opus", "alac", "flac", "wav",
].map((name) => ({ Kind: QUALITY_KIND.Audio, Name: name }));

export const BOOK_QUALITIES: Quality[] = [
  "txt", "pdf", "djvu", "fb2", "epub", "mobi", "azw3", "docx",
].map((name) => ({ Kind: QUALITY_KIND.Book, Name: name }));

export function qualitiesForType(type: MediaType): Quality[] {
  if (type === MediaType.Movie) return VIDEO_QUALITIES_MOVIE;
  if (type === MediaType.Series) return VIDEO_QUALITIES_SERIES;
  if (type === MediaType.MusicAlbum) return AUDIO_QUALITIES;
  return [];
}

export interface QualityProfile {
  Id: number;
  Name: string;
  Type: MediaType;
  Allowed: Quality[];
  Cutoff: Quality;
}

export interface Task {
  Name: string;
  Interval: string;
  Enabled: boolean;
  LastRun: string | null;
  NextRun: string | null;
  LastStatus: "idle" | "running" | "ok" | "error";
  LastError: string;
  Settings?: Record<string, string>;
  Data?: string;
}

export interface Source {
  Id: number;
  Type: string;
  Name: string;
  Settings: Record<string, string>;
  Enabled: boolean;
  ProxyID: number | null;
}

export interface Proxy {
  Id: number;
  Name: string;
  Type: string;
  Endpoint: string;
  Enabled: boolean;
}

export const AUTHOR_TODAY_NAME = "author_today";

export type UserRole = "admin" | "user";

export interface User {
  Id: number;
  Name: string;
  Email: string;
  Role: UserRole;
  CreatedAt: string;
}

export interface AuthStatus {
  LoginEnabled: boolean;
  RegistrationEnabled: boolean;
  OIDCEnabled: boolean;
}

export interface AuthResponse {
  User: User;
  AccessToken: string;
}
