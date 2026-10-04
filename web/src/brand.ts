export type Brand = {
  interface?: 'video' | 'catalog';
  title: string; subtitle: string; collection: string; kicker: string;
  headline: string; description: string; logo?: string;
  colors: { background: string; surface: string; ink: string; muted: string; accent: string; border: string };
};
export function parseBrand(input: unknown): Brand {
  if (!input || typeof input !== 'object') throw new Error('Site configuration is missing.');
  const value = input as Brand;
  if (value.interface !== undefined && value.interface !== 'video' && value.interface !== 'catalog') throw new Error('Invalid site interface.');
  for (const key of ['title', 'subtitle', 'collection', 'kicker', 'headline', 'description'] as const) {
    if (typeof value[key] !== 'string' || !value[key].trim() || value[key].length > 1000) throw new Error(`Invalid site ${key}.`);
  }
  if (!/^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$/.test(value.collection)) throw new Error('Invalid site collection.');
  for (const key of ['background', 'surface', 'ink', 'muted', 'accent', 'border'] as const) {
    if (!/^#[0-9a-fA-F]{6}$/.test(value.colors?.[key] || '')) throw new Error(`Invalid site color ${key}.`);
  }
  if (value.logo && (!/^\/[A-Za-z0-9_./-]+$/.test(value.logo) || value.logo.startsWith('//') || value.logo.includes('..'))) throw new Error('Logo must be a local asset path.');
  return value;
}
