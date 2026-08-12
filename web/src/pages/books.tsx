import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api/client";
import type { Media } from "@/lib/api/types";
import { MediaType } from "@/lib/api/types";

export function BooksPage() {
  const { t } = useTranslation();
  const { data, isLoading, error } = useQuery({
    queryKey: ["media", "by-type", MediaType.Book],
    queryFn: () => api.get<Media[]>(`/api/media?type=${MediaType.Book}&limit=100`),
  });

  return (
    <div className="mx-auto max-w-7xl px-8 py-8">
      <header className="mb-8">
        <h1 className="text-2xl font-semibold tracking-tight">{t("pages.books.title")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("pages.books.description")}
        </p>
      </header>

      {isLoading && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
          {Array.from({ length: 12 }).map((_, i) => (
            <div key={i} className="aspect-[2/3] animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      )}

      {error && (
        <p className="text-sm text-destructive">{t("common.failedToLoad")}: {(error as Error).message}</p>
      )}

      {data && data.length === 0 && (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-24 text-center">
          <p className="text-sm text-muted-foreground">
            {t("empty.libraryBook")}
          </p>
        </div>
      )}

      {data && data.length > 0 && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
          {data.map((m) => (
            <MediaCard key={m.Id} media={m} />
          ))}
        </div>
      )}
    </div>
  );
}

export function MediaCard({ media }: { media: Media }) {
  const { t } = useTranslation();
  return (
    <Link to={`/books/${media.Id}`} className="group space-y-2 transition-transform hover:scale-[1.02]">
      <div className="aspect-[2/3] overflow-hidden rounded-lg border border-border bg-muted">
        {media.Cover ? (
          <img src={media.Cover} alt={media.Name} className="h-full w-full object-cover" />
        ) : (
          <div className="flex h-full items-center justify-center text-xs text-muted-foreground">
            {t("common.noCover")}
          </div>
        )}
      </div>
      <p className="line-clamp-1 text-sm font-medium">{media.Name}</p>
    </Link>
  );
}
