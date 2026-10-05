export type VideoFilters = { dateFrom?: string; dateTo?: string; subject?: string; language?: string; creator?: string; durationMin?: string; durationMax?: string };
export function videoCatalogURL(collection: string, query: string, year: string, page: number, filters: VideoFilters = {}): string {
  const params = new URLSearchParams({ collection, q: query, year, page: String(page) });
  for (const [key, value] of Object.entries({ date_from: filters.dateFrom, date_to: filters.dateTo, subject: filters.subject, language: filters.language, creator: filters.creator, duration_min: filters.durationMin, duration_max: filters.durationMax })) {
    if (value) params.set(key, value);
  }
  return `/api/episodes?${params}`;
}
