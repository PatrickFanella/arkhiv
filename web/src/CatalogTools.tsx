import { useEffect, useState } from 'react';
import { catalogRequest, type CatalogItem } from '@subcult/archive-viewer/catalog';
import type { Personal } from '@subcult/archive-viewer';

export function CatalogShare({url,title}: {url:string;title:string}) {
  const [fallback,setFallback]=useState(false),[status,setStatus]=useState('');
  async function share() {
    try {
      if(navigator.share) { await navigator.share({url,title});setStatus(''); }
      else { await navigator.clipboard.writeText(url);setStatus('Link copied.'); }
    } catch(reason) { if(reason instanceof DOMException && reason.name==='AbortError')return;setFallback(true);setStatus('Copy this link to share.'); }
  }
  return <div className="catalog-share"><button onClick={()=>{void share();}}>Share item</button><a href={`https://bsky.app/intent/compose?text=${encodeURIComponent(`${title} ${url}`)}`} target="_blank" rel="noreferrer">Share on Bluesky</a> <a href={`mailto:?subject=${encodeURIComponent(title)}&body=${encodeURIComponent(url)}`}>Share by email</a>{status && <p role="status">{status}</p>}{fallback && <label>Item link<input readOnly value={url} onFocus={event=>event.currentTarget.select()}/></label>}</div>;
}

export function CatalogPersonal({personal,collection,onOpen}: {personal:Personal;collection:string;onOpen:(item:CatalogItem)=>void}) {
  const [review,setReview]=useState(false),[rebase,setRebase]=useState(false),[alias,setAlias]=useState(personal.user?.alias || '');
  const [deleteReview,setDeleteReview]=useState(false),[page,setPage]=useState(1);
  const [entries,setEntries]=useState<Record<string,CatalogItem>>({});
  const [loaded,setLoaded]=useState(false),[exportError,setExportError]=useState('');
  const ids=personal.favoriteIDs;
  const currentPage=Math.min(page,Math.max(1,Math.ceil(ids.length/24)));
  const current=ids.slice((currentPage-1)*24,currentPage*24);
  const signature=JSON.stringify(current);
  useEffect(()=>{
    const controller=new AbortController();setLoaded(false);
    void Promise.all((JSON.parse(signature) as string[]).filter(id=>id.startsWith('catalog:')).map(async id=>{
      try { return [id,await catalogRequest<CatalogItem>(`/api/catalog/v1/items/${encodeURIComponent(id.slice(8))}?${new URLSearchParams({collection})}`,controller.signal)] as const; }
      catch { return null; }
    })).then(values=>{if(!controller.signal.aborted){setEntries(Object.fromEntries(values.filter(value=>value!==null)));setLoaded(true);}});
    return()=>controller.abort();
  },[signature,collection]);
  async function exportData() {
    const blob=new Blob([await personal.exportData()],{type:'application/json'}),url=URL.createObjectURL(blob);
    const link=document.createElement('a');link.href=url;link.download='archive-personal-data.json';link.click();setTimeout(()=>URL.revokeObjectURL(url),1000);
  }
  return <section className="catalog-personal" aria-label="Favorites and account"><h3>Favorites in {collection}</h3><p>{personal.user ? `Signed in as ${personal.user.alias}. Favorites sync to ${personal.deployment}.` : 'Favorites are saved in this browser.'}</p>
    {personal.error && <p role="alert">{personal.error}</p>}{exportError && <p role="alert">{exportError}</p>}
    {!personal.user && personal.enabled && <button onClick={()=>{void personal.signIn();}}>Sign in to sync favorites</button>}
    <button onClick={()=>{void exportData().catch(()=>setExportError('Personal export is unavailable. Please try again.'));}}>Export favorites and personal data</button>
    {personal.user && <><label>Display alias<input maxLength={40} value={alias} onChange={event=>setAlias(event.target.value)}/></label><button disabled={personal.busy} onClick={()=>{void personal.updateAlias(alias);}}>Save alias</button><button onClick={()=>{void personal.signOut();}}>Sign out</button><button onClick={()=>{void personal.signOut(true);}}>Sign out everywhere</button>
      <details><summary>Review browser favorites</summary><ul>{personal.localFavorites.map(id=><li key={id}>{id}</li>)}</ul></details><p>{personal.localFavorites.length} browser favorites can be imported. The browser copy is retained.</p><label><input type="checkbox" checked={review} onChange={event=>setReview(event.target.checked)}/>I reviewed this account, collection and browser favorites.</label><button disabled={!review || personal.busy} onClick={()=>{void personal.importLocal();}}>Prepare import</button>
      {personal.pending.length>0 && <><ul>{personal.pending.map(change=><li key={change.item}>{change.remove ? 'Remove' : 'Add'} {change.item}</li>)}</ul><button disabled={personal.busy} onClick={()=>{void personal.synchronize();}}>Apply reviewed changes</button><label><input type="checkbox" checked={rebase} onChange={event=>setRebase(event.target.checked)}/>Apply these changes against the latest account revision.</label><button disabled={!rebase || personal.busy} onClick={()=>{void personal.synchronize(true);}}>Refresh and apply reviewed changes</button></>}
      <details><summary>Delete account</summary><p>Deletion removes your profile, favorites and chat bodies and revokes sessions. Chat retention is 30 days, personal backups 35 days, and opaque recovery deletion records 90 days.</p><label><input type="checkbox" checked={deleteReview} onChange={event=>setDeleteReview(event.target.checked)}/>Delete my account and personal data.</label>{personal.reauthenticationRequired && <button onClick={()=>{void personal.signIn();}}>Sign in again to confirm deletion</button>}<button disabled={!deleteReview || personal.reauthenticationRequired || personal.busy} onClick={()=>{void personal.removeAccount();}}>Delete account</button></details>
    </>}
    <ul>{current.map(id=><li key={id}>{entries[id] ? <button onClick={()=>onOpen(entries[id])}>{entries[id].title}</button> : <span>{loaded ? 'Unavailable item' : 'Checking item'}: {id}</span>} <button disabled={personal.busy} onClick={()=>{void personal.toggle(id);}}>Remove favorite</button></li>)}</ul>
    {ids.length>24 && <nav aria-label="Favorite pages"><button disabled={currentPage<=1} onClick={()=>setPage(currentPage-1)}>Previous favorites</button><span>Page {currentPage}</span><button disabled={currentPage*24>=ids.length} onClick={()=>setPage(currentPage+1)}>Next favorites</button></nav>}
  </section>;
}
