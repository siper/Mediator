import { StrictMode, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "sonner";
import { I18nextProvider } from "@/i18n";
import { ThemeProvider } from "@/components/theme-provider";
import { ErrorBoundary } from "@/components/error-boundary";
import { AppShell } from "@/components/app-shell";
import { AuthGuard, AdminGuard } from "@/components/auth-guard";
import { AuthStatusGate } from "@/components/auth-status-gate";
import { useAuthStore } from "@/store/auth";
import { setAuthFailureHandler } from "@/lib/api/client";
import { authStatusQueryOptions } from "@/lib/auth-status";
import { LoginPage } from "@/pages/auth/login";
import { RegisterPage } from "@/pages/auth/register";
import { MoviesPage } from "@/pages/movies";
import { MovieDetailPage } from "@/pages/movie-detail";
import { SeriesPage } from "@/pages/series";
import { SeriesDetailPage } from "@/pages/series-detail";
import { BooksPage } from "@/pages/books";
import { BookDetailPage } from "@/pages/book-detail";
import { MusicPage } from "@/pages/music";
import { MusicDetailPage } from "@/pages/music-detail";
import { WantedPage } from "@/pages/wanted";
import { QueuePage } from "@/pages/queue";
import { HistoryPage } from "@/pages/history";
import { RequestsPage } from "@/pages/requests";
import { SettingsPage } from "@/pages/settings";
import { LibrariesSection } from "@/components/settings/libraries-section";
import { IndexersSection } from "@/components/settings/indexers-section";
import { ClientsSection } from "@/components/settings/clients-section";
import { ProfilesSection } from "@/components/settings/profiles-section";
import { SourcesSection } from "@/components/settings/sources-section";
import { ProxiesSection } from "@/components/settings/proxies-section";
import { TasksSection } from "@/components/settings/tasks-section";
import { RequestsSection } from "@/components/settings/requests-section";
import { UsersSection } from "@/components/settings/users-section";
import { GeneralSection } from "@/components/settings/general-section";
import "@/index.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 10_000, retry: 1 },
  },
});

void queryClient.prefetchQuery(authStatusQueryOptions);

setAuthFailureHandler(() => useAuthStore.getState().setUser(null));

function AppRoutes() {
  const checkAuth = useAuthStore((s) => s.checkAuth);
  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route element={<AuthGuard><AppShell /></AuthGuard>}>
        <Route index element={<Navigate to="/movies" replace />} />
        <Route path="/movies" element={<MoviesPage />} />
        <Route path="/movies/:id" element={<MovieDetailPage />} />
        <Route path="/series" element={<SeriesPage />} />
        <Route path="/series/:id" element={<SeriesDetailPage />} />
        <Route path="/books" element={<BooksPage />} />
        <Route path="/books/:id" element={<BookDetailPage />} />
        <Route path="/music" element={<MusicPage />} />
        <Route path="/music/:id" element={<MusicDetailPage />} />
         <Route path="/wanted" element={<WantedPage />} />
         <Route path="/queue" element={<QueuePage />} />
         <Route path="/history" element={<HistoryPage />} />
         <Route path="/requests" element={<RequestsPage />} />
        <Route path="/settings" element={<SettingsPage />}>
          <Route index element={<Navigate to="/settings/general" replace />} />
          <Route path="general" element={<GeneralSection />} />
        </Route>
        <Route element={<AdminGuard />}>
          <Route path="/admin" element={<SettingsPage admin />}>
            <Route index element={<Navigate to="/admin/libraries" replace />} />
            <Route path="libraries" element={<LibrariesSection />} />
            <Route path="indexers" element={<IndexersSection />} />
            <Route path="clients" element={<ClientsSection />} />
            <Route path="profiles" element={<ProfilesSection />} />
            <Route path="sources" element={<SourcesSection />} />
            <Route path="proxies" element={<ProxiesSection />} />
            <Route path="tasks" element={<TasksSection />} />
            <Route path="requests" element={<RequestsSection />} />
            <Route path="users" element={<UsersSection />} />
          </Route>
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/movies" replace />} />
    </Routes>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <I18nextProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <ErrorBoundary>
            <BrowserRouter>
              <AuthStatusGate>
                <AppRoutes />
              </AuthStatusGate>
            </BrowserRouter>
          </ErrorBoundary>
          <Toaster theme="dark" position="bottom-right" />
        </QueryClientProvider>
      </ThemeProvider>
    </I18nextProvider>
  </StrictMode>
);
