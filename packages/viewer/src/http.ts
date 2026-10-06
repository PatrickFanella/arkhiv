export async function readJSON<T>(response: Response): Promise<T> {
  try {
    return await response.json() as T;
  } catch (reason) {
    if (reason instanceof DOMException && (reason.name === 'AbortError' || reason.name === 'TimeoutError')) throw reason;
    throw new Error(response.status === 404
      ? 'This item is unavailable for this collection.'
      : response.ok ? 'The archive returned an unreadable response. Please try again.'
        : 'The archive is temporarily unavailable. Please try again.');
  }
}
