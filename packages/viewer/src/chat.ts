import { useCallback, useEffect, useRef, useState } from 'react';
import { PersonalError, personalRequest, type Personal } from './personal.js';
import type { Session } from './model.js';

export type ChatProgram = { revision: string; start: string; episode: string; original_date: string };
export type ChatMessage = ChatProgram & { seq: number; id: string; user_id: string; alias: string; body: string; collection: string; created_at: string };
export type ChatRoom = { collection: string; current: ChatProgram | null; read: boolean; post: boolean; retention_days: number; poll_seconds: number; moderation_contact: string };
export type ChatReport = { id: string; message_id: string; reason: string; body: string; collection: string; resolved: boolean; created_at: string };
const reasonMessage = (reason: unknown) => reason instanceof Error ? reason.message : 'Chat unavailable.';

export function useChannelChat(collection: string, session: Session | null, paused: boolean, personal: Personal) {
  const [room, setRoom] = useState<ChatRoom | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [reports, setReports] = useState<ChatReport[]>([]);
  const [blocked, setBlocked] = useState<string[]>([]);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [before, setBefore] = useState(0);
  const [oldest, setOldest] = useState(0);
  const [enabled, setEnabled] = useState(false);
  const generation = useRef(0);
  const actionInFlight = useRef(false);
  const retry = useRef<{ body: string; key: string; context: ChatProgram } | null>(null);
  const refresh = useCallback(async () => {
    const token = generation.current;
    const context = await personalRequest<ChatRoom>(`/api/chat/v1/rooms/${encodeURIComponent(collection)}`);
    if (generation.current !== token) return;
    setEnabled(true); setRoom(context);
    if (!context.read) { setMessages([]); return; }
    const page = await personalRequest<{ messages: ChatMessage[]; before: number }>(`/api/chat/v1/rooms/${encodeURIComponent(collection)}/messages${before ? `?before=${before}` : ''}`);
    if (generation.current !== token) return;
    setMessages(page.messages); setOldest(page.before); setError('');
  }, [collection, before]);
  useEffect(() => {
    generation.current++; retry.current = null;
    setReports([]); setBlocked([]);
    const token = generation.current;
    if (personal.user) void personalRequest<{ blocked: string[] }>('/api/chat/v1/blocks').then(result => { if (generation.current === token) setBlocked(result.blocked); }).catch(reason => { if (generation.current === token) setError(reasonMessage(reason)); });
    let stopped = false, inFlight = false, failures = 0;
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      if (stopped || inFlight) return; inFlight = true;
      try { await refresh(); failures = 0; }
      catch (reason) { if (!stopped) { if (reason instanceof PersonalError && reason.status === 404) { setEnabled(false); setRoom(null); setMessages([]); } else setError(reasonMessage(reason)); failures++; } }
      finally { inFlight = false; if (!stopped) timer = setTimeout(() => { void poll(); }, Math.min(60_000, 10_000 * 2 ** Math.min(failures, 3))); }
    };
    void poll(); return () => { stopped = true; generation.current++; clearTimeout(timer); };
  }, [refresh, personal.user?.id]);
  const canPost = !!personal.user && !!room?.post && session?.mode === 'channel' && !paused && session.episode.id === room.current?.episode && session.programKey === `${room.current?.revision}:${room.current?.start}`;
  const action = async (callback: () => Promise<void>) => { if (actionInFlight.current) return false; actionInFlight.current = true; setBusy(true); try { await callback(); setError(''); return true; } catch (reason) { setError(reasonMessage(reason)); return false; } finally { actionInFlight.current = false; setBusy(false); } };
  const send = async (body: string) => {
    if (!canPost || !room?.current) { setError('Join the current channel and sign in before posting.'); return false; }
    let sent = false;
    await action(async () => {
      const context = room.current!;
      const previous = retry.current;
      const change = previous && previous.body === body && previous.context.revision === context.revision && previous.context.start === context.start ? previous : { body, key: crypto.randomUUID(), context };
      retry.current = change;
      await personalRequest(`/api/chat/v1/rooms/${encodeURIComponent(collection)}/messages`, personal.csrf, 'POST', { body, idempotency_key: change.key, ...change.context, mode: session!.mode, paused });
      retry.current = null; sent = true; setBefore(0); await refresh();
    });
    return sent;
  };
  const report = (id: string, reason: string) => action(async () => { await personalRequest(`/api/chat/v1/messages/${encodeURIComponent(id)}/reports`, personal.csrf, 'POST', { reason }); });
  const remove = (id: string) => action(async () => { await personalRequest(`/api/chat/v1/messages/${encodeURIComponent(id)}`, personal.csrf, 'DELETE'); await refresh(); });
  const block = (id: string, remove = false) => action(async () => { await personalRequest(`/api/chat/v1/blocks/${encodeURIComponent(id)}`, personal.csrf, remove ? 'DELETE' : 'PUT'); const result = await personalRequest<{ blocked: string[] }>('/api/chat/v1/blocks'); setBlocked(result.blocked); await refresh(); });
  const reviewReports = () => action(async () => { const result = await personalRequest<{ reports: ChatReport[] }>('/api/chat/v1/moderation'); setReports(result.reports); });
  const moderate = (actionName: string, target: string, reason: string, hours = 0) => action(async () => { await personalRequest('/api/chat/v1/moderation', personal.csrf, 'POST', { action: actionName, target, reason, hours }); await refresh(); });
  return { room, messages, reports, blocked, enabled, error, busy, canPost, history: before > 0, hasOlder: messages.length === 50, older: () => { if (oldest) setBefore(oldest); }, current: () => setBefore(0), refresh, send, report, remove, block, reviewReports, moderate };
}
export type ChannelChat = ReturnType<typeof useChannelChat>;
