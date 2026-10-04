import { useEffect, useState } from 'react';
import { PrintReader, type PrintHit } from '@subcult/archive-viewer/print';
import { catalogRequest } from '@subcult/archive-viewer/catalog';
import { pdfBackend } from './pdfBackend';

export default function DocumentView({url,title,path,collection}: {url:string;title:string;path:string;collection:string}) {
  const [coverage,setCoverage] = useState('');
  const params = new URLSearchParams({collection}).toString();
  useEffect(() => {
    const controller = new AbortController();
    void catalogRequest<{text_status:string}>(`${path}/print?${params}`,controller.signal).then(index=>setCoverage(index.text_status)).catch(()=>undefined);
    return ()=>controller.abort();
  },[path,params]);
  const searchable = coverage === 'page-text' || coverage === 'document-text';
  return <><p>{searchable ? coverage === 'page-text' ? 'Search covers indexed page text.' : 'Search covers document text; page mapping is unavailable.' : 'Document search is unavailable. You can browse the PDF pages.'}</p>
    <PrintReader url={url} title={title} backend={pdfBackend} search={searchable ? (query,signal) => catalogRequest<PrintHit[]>(`${path}/search?${new URLSearchParams({collection,q:query})}`,signal) : undefined} /></>;
}
