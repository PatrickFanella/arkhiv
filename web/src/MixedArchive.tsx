import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import { useCatalog, catalogRequest, representationPath, representationURL, type CatalogCollection, type CatalogItem, type CatalogMedia, type CatalogRepresentation } from '@subcult/archive-viewer/catalog';
import { AudioPlayer, type AudioAsset } from '@subcult/archive-viewer/audio';
import { Player, playbackURL, type Playback } from '@subcult/archive-viewer';
import type { Brand } from './brand';
const DocumentView = lazy(()=>import('./DocumentView'));
const CollectionChannels = lazy(()=>import('./CollectionChannels'));

type Selection = {item:CatalogItem;representation:CatalogRepresentation;collection:string};
type Transport = {media:CatalogMedia;policy:string;url:string;source_url:string;file:string};
function SelectedItem({selection,onClose}: {selection:Selection;onClose:()=>void}) {
  const {item,representation,collection} = selection;
  const path = representationPath(item,representation);
  const [asset,setAsset] = useState<Transport>();
  const [audio,setAudio] = useState<AudioAsset>();
  const [legacy,setLegacy] = useState<Playback>();
  const [paused,setPaused] = useState(true);
  const [setIndex,setSetIndex] = useState(-1);
  const [error,setError] = useState('');
  const section = useRef<HTMLElement>(null);
  useEffect(()=>{section.current?.scrollIntoView({block:'start'});section.current?.focus();},[]);
  useEffect(()=> {
    const controller = new AbortController();
    const params = new URLSearchParams({collection});
    void (async()=> {
      if (representation.media === 'print' && !/\.pdf$/i.test(representation.file)) throw new Error('This reader supports PDF files. Open the source page for this format.');
      // Continuous recordings have a separate derivative receipt and resolver.
      if (representation.media === 'audio') {
        const full = await fetch(`${path}/audio/resolve?${params}`,{signal:controller.signal,cache:'no-store'});
        if (full.ok) { const value = await full.json() as AudioAsset; if(!controller.signal.aborted)setAudio(value); return; }
        if (full.status !== 404) throw new Error('The recording is temporarily unavailable.');
        if (representation.extension.audio?.recording) throw new Error('The continuous recording is unavailable. Open the source page to check its original files.');
      }
      if (representation.legacy_episode_id) {
        const response = await catalogRequest<Playback>(`/api/playback/${encodeURIComponent(representation.legacy_episode_id)}?${params}`,controller.signal);
        playbackURL(response);
        if(!controller.signal.aborted){setLegacy(response);setAsset({...response,media:'video',file:representation.file});} return;
      }
      const resolved = await catalogRequest<Transport>(`${path}/resolve?${params}`,controller.signal);
      if (resolved.media !== representation.media) throw new Error('The media format has changed.');
      representationURL(resolved);
      if(!controller.signal.aborted)setAsset(resolved);
    })().catch(reason=> {if(!controller.signal.aborted)setError(reason instanceof Error ? reason.message : 'This item is unavailable.');});
    return ()=>controller.abort();
  },[path,collection,representation]);
  const source = `https://archive.org/details/${encodeURIComponent(representation.source_item_id)}`;
  return <section ref={section} tabIndex={-1} className="catalog-selection" aria-label="Selected archive item">
    <button onClick={onClose}>Close item</button><p className="eyebrow">{representation.media} · {collection}</p><h2>{item.title}</h2>
    <p>{[item.creator,item.date].filter(Boolean).join(' · ')}</p><p>{item.description}</p>
    {!error && !asset && !audio && <p role="status">Checking access…</p>}{error && <p role="alert">{error}</p>}
    {audio && <><label>Recording section<select value={setIndex} onChange={event=>setSetIndex(Number(event.target.value))}><option value={-1}>Full recording</option>{audio.recording.sets.map((set,index)=>set.reviewed && <option key={index} value={index}>{set.title}</option>)}</select></label><AudioPlayer asset={audio} set={setIndex>=0 ? audio.recording.sets[setIndex] : undefined}/></>}
    {asset?.media === 'print' && <Suspense fallback={<p role="status">Opening PDF reader…</p>}><DocumentView url={asset.url} title={item.title} path={path} collection={collection}/></Suspense>}
    {legacy && <Player key={legacy.url} session={{mode:'ondemand',collection,episode:{id:representation.legacy_episode_id!,title:item.title,date:item.date,description:item.description,synthetic:false},playback:legacy,offset:0,receivedAt:performance.now(),latency:0,generation:1}} paused={paused} onPaused={setPaused}/>}
    {asset?.media === 'video' && !legacy && <video key={asset.url} controls playsInline preload="metadata" src={asset.url} aria-label={item.title} onError={()=>setError('The video source is unavailable. Try another format or open the source page.')} />}
    {asset?.media === 'audio' && <audio key={asset.url} controls preload="metadata" src={asset.url} aria-label={item.title} onError={()=>setError('The audio source is unavailable. Try another format or open the source page.')} />}
    {!audio && representation.extension.audio && <dl>{Object.entries(representation.extension.audio.credits || {}).map(([label,value])=><div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>}
    <p><a href={source} target="_blank" rel="noreferrer">Source item at Internet Archive</a> · {representation.file}</p>
  </section>;
}
export default function MixedArchive({brand}: {brand:Brand}) {
  const [collections,setCollections] = useState<CatalogCollection[]>([]);
  const [collectionError,setCollectionError] = useState('');
  const [collection,setCollection] = useState('');
  const [media,setMedia] = useState<CatalogMedia | ''>('');
  const [query,setQuery] = useState('');
  const [page,setPage] = useState(1);
  const [selection,setSelection] = useState<Selection>();
  const catalog = useCatalog({collection,media,query,page});
  useEffect(()=> {
    document.title = brand.title;
    for (const [key,color] of Object.entries(brand.colors)) document.documentElement.style.setProperty(`--${key}`,color);
  },[brand]);
  useEffect(()=> {
    const controller = new AbortController();
    void catalogRequest<CatalogCollection[]>('/api/catalog/v1/collections',controller.signal).then(value=> {if(!controller.signal.aborted)setCollections(value);}).catch(reason=>{if(!controller.signal.aborted)setCollectionError(reason.message);});
    return ()=>controller.abort();
  },[]);
  function changeFilter(update:()=>void) {update();setPage(1);setSelection(undefined);}
  return <main className="mixed-archive"><a className="skip-link" href="#catalog">Skip to archive catalog</a>
    <header><a className="wordmark" href="/">{brand.logo && <img src={brand.logo} alt=""/>}{brand.title}<span>{brand.subtitle}</span></a><a href="#catalog">Browse collections</a></header>
    <section className="catalog-intro"><p className="eyebrow">{brand.kicker}</p><h1>{brand.headline}</h1><p>{brand.description}</p></section>
    <section id="catalog" aria-label="Archive catalog">
      <div className="catalog-filters"><label>Collection<select value={collection} onChange={event=>changeFilter(()=>setCollection(event.target.value))}><option value="">All collections</option>{collections.map(c=><option key={c.id} value={c.id}>{c.title} ({c.items})</option>)}</select></label>
        <label>Media<select value={media} onChange={event=>changeFilter(()=>setMedia(event.target.value as CatalogMedia | ''))}><option value="">All media</option><option value="print">Print</option><option value="audio">Audio</option><option value="video">Video</option></select></label>
        <label>Search titles and descriptions<input type="search" value={query} maxLength={200} onChange={event=>changeFilter(()=>setQuery(event.target.value))}/></label></div>
      {collectionError && <p role="alert">Collection names are unavailable: {collectionError}</p>}
      {catalog.pending && <p role="status">Loading catalog…</p>}{catalog.error && <p role="alert">{catalog.error} <button onClick={catalog.retry}>Try again</button></p>}
      {catalog.page && <><p role="status">{catalog.page.total} items · Metadata search</p>{catalog.page.items.length === 0 && <p>No items match these filters.</p>}
        <div className="catalog-items">{catalog.page.items.map(item=><article key={item.id}><p className="eyebrow">{item.representations.map(rep=>rep.media).filter((value,index,array)=>array.indexOf(value)===index).join(' / ')}</p><h2>{item.title}</h2><p>{[item.creator,item.date].filter(Boolean).join(' · ')}</p><p className="catalog-description">{item.description}</p>
          {item.representations.flatMap(rep=>rep.collections.map(member=><button key={`${rep.id}:${member}`} onClick={()=>setSelection({item,representation:rep,collection:member})}>{rep.media === 'print' ? 'Read' : rep.media === 'audio' ? 'Listen' : 'Watch'}{rep.collections.length>1 ? ` · ${member}` : ''}</button>))}</article>)}</div>
        <nav className="catalog-pages" aria-label="Catalog pages"><button disabled={page<=1} onClick={()=>{setPage(value=>value-1);setSelection(undefined);}}>Previous</button><span>Page {page}</span><button disabled={page*24>=catalog.page.total} onClick={()=>{setPage(value=>value+1);setSelection(undefined);}}>Next</button></nav></>}
    </section>
    {collection && <details><summary>Scheduled programs for this collection</summary><Suspense fallback={<p role="status">Opening channel controls…</p>}><CollectionChannels collection={collection} media={collections.find(c=>c.id===collection)?.media || []}/></Suspense></details>}
    {selection && <SelectedItem key={`${selection.item.id}:${selection.representation.id}:${selection.collection}`} selection={selection} onClose={()=>setSelection(undefined)}/>}
    <footer><span>{brand.title}</span><span>Source metadata retained · Access reviewed per collection</span></footer>
  </main>;
}
