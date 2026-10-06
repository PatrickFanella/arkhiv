// Shared across hook remounts; isolated by storage key and browser page.
export class PersonalBrowserStorage {
  private memory = new Map<string, unknown>();
  private failed = new Set<string>();
  constructor(private storage: () => Pick<Storage, 'getItem' | 'setItem'>) {}
  read<T>(key: string, fallback: T): T {
    if (this.failed.has(key) && this.memory.has(key)) return this.memory.get(key) as T;
    try {
      return JSON.parse(this.storage().getItem(key) || 'null') as T ?? this.memory.get(key) as T ?? fallback;
    } catch {
      this.failed.add(key);
      return this.memory.get(key) as T ?? fallback;
    }
  }
  write(key: string, value: unknown) {
    this.memory.set(key, value);
    try {
      this.storage().setItem(key, JSON.stringify(value));
      this.failed.delete(key);
    } catch { this.failed.add(key); }
  }
  changed(key: string) { this.memory.delete(key); this.failed.delete(key); }
  temporary(key: string) { return this.failed.has(key); }
}
