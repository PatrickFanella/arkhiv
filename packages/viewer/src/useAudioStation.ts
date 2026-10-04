import { useCallback, useEffect, useRef, useState } from 'react';
import { request } from './model.js';
import { audioURL, type AudioAsset } from './audioModel.js';

export type AudioStation = {
 state: string;
 reason?: string;
 server_time: string;
 revision?: string;
 offset?: number;
 program?: {item_id: string; representation_id: string; start: string; duration: number};
 next?: {item_id: string; representation_id: string; start: string; duration: number};
 playback?: AudioAsset;
};
export type StationSession = {asset: AudioAsset; station: {offset: number; receivedAt: number; latency: number; duration: number; programKey: string}};

// Polling reads schedule state; joining/rejoining changes only this listener.
export function useAudioStation(collection: string, enabled = true) {
 const [snapshot,setSnapshot] = useState<{collection: string; value?: AudioStation; error?: string}>({collection});
 const [listener,setListener] = useState<{collection: string; session: StationSession} | null>(null);
 const controller = useRef<AbortController | null>(null);
 const listenerController = useRef<AbortController | null>(null);
 const generation = useRef(0);
 useEffect(() => {
  if (!enabled) return;
  const current = new AbortController(); controller.current = current;
  let timer: ReturnType<typeof setTimeout>;
  async function poll() {
   try {
    const value = await request<AudioStation>(`/api/audio/channel?collection=${encodeURIComponent(collection)}`,current.signal);
    if (!current.signal.aborted) setSnapshot({collection,value});
   } catch {
    if (!current.signal.aborted) setSnapshot({collection,error:'Station status is unavailable. Please try again.'});
   }
   if (!current.signal.aborted) timer=setTimeout(() => {void poll();},5000);
  }
  void poll();
  return () => {current.abort();clearTimeout(timer);};
 },[collection,enabled]);
 useEffect(() => () => {listenerController.current?.abort();},[collection]);
 const join = useCallback(async () => {
  listenerController.current?.abort();
  const current = new AbortController(); listenerController.current=current;
  const sentAt=performance.now();
  try {
   const value=await request<AudioStation>(`/api/audio/channel?collection=${encodeURIComponent(collection)}`,current.signal);
   if (current.signal.aborted) return;
   setSnapshot({collection,value});
   if (value.state !== 'playing' || !value.playback || !value.program || !value.revision || value.offset === undefined || !Number.isFinite(value.offset) || value.offset < 0 || !Number.isFinite(value.program.duration) || value.program.duration <= 0 || value.offset >= value.program.duration) throw new Error('Station unavailable');
   audioURL(value.playback);
   const receivedAt=performance.now();
   setListener({collection,session:{asset:value.playback,station:{offset:value.offset,receivedAt:Date.now(),latency:(receivedAt-sentAt)/2,duration:value.program.duration,programKey:`${value.revision}:${value.program.item_id}:${value.program.representation_id}:${value.program.start}:${++generation.current}`}}});
  } catch {
   if (!current.signal.aborted) {setListener(null);setSnapshot({collection,error:'The station could not be joined. Please try again.'});}
  }
 },[collection]);
 const leave = useCallback(() => {listenerController.current?.abort();setListener(null);},[]);
 return {station:enabled && snapshot.collection === collection ? snapshot.value : undefined,error:snapshot.collection === collection ? snapshot.error || '' : '',session:listener?.collection === collection ? listener.session : null,join,rejoin:join,leave};
}
