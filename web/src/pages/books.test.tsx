import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { I18nextTestProvider } from "@/i18n";
import { MediaCard } from "@/pages/books";
import type { Media } from "@/lib/api/types";
import { MediaType } from "@/lib/api/types";

const baseMedia: Media = {
  Id: 1,
  Name: "Test Movie",
  OriginalName: "",
  Folder: null,
  Cover: null,
  Type: MediaType.Movie,
  LibraryID: null,
  ProviderID: "tmdb",
  ExternalID: "123",
  Status: "ok",
  LastModified: "",
  QualityProfileID: null,
};

describe("MediaCard", () => {
  it("renders media name", () => {
    render(
      <MemoryRouter>
        <I18nextTestProvider>
          <MediaCard media={baseMedia} />
        </I18nextTestProvider>
      </MemoryRouter>,
    );
    expect(screen.getByText("Test Movie")).toBeInTheDocument();
  });

  it("renders cover image when present", () => {
    const media = { ...baseMedia, Cover: "/covers/test.jpg" };
    render(
      <MemoryRouter>
        <I18nextTestProvider>
          <MediaCard media={media} />
        </I18nextTestProvider>
      </MemoryRouter>,
    );
    const img = screen.getByAltText("Test Movie");
    expect(img).toHaveAttribute("src", "/covers/test.jpg");
  });

  it("renders placeholder when cover is null", () => {
    render(
      <MemoryRouter>
        <I18nextTestProvider>
          <MediaCard media={baseMedia} />
        </I18nextTestProvider>
      </MemoryRouter>,
    );
    expect(screen.getByText("No cover")).toBeInTheDocument();
  });

  it("links to the correct detail page", () => {
    render(
      <MemoryRouter>
        <I18nextTestProvider>
          <MediaCard media={baseMedia} />
        </I18nextTestProvider>
      </MemoryRouter>,
    );
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/books/1");
  });
});
