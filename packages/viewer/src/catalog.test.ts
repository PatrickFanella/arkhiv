import { expect, test } from 'vitest';
import { catalogURL, representationURL, type RepresentationTransport } from './catalog.js';

test('metadata queries preserve collection scope and escape user text',()=> {
  const url = new URL(catalogURL({collection:'mixed',media:'print',query:'&collection=other',page:2}),'https://test.invalid');
  expect(url.searchParams.get('collection')).toBe('mixed');
  expect(url.searchParams.get('q')).toBe('&collection=other');
  expect(url.searchParams.get('page')).toBe('2');
});
const asset: RepresentationTransport = {media:'print',policy:'local_preserve',file:'paper.pdf',source_url:'https://archive.org/details/paper',url:'/api/catalog/v1/items/paper/representations/pdf/media?collection=mixed&action=read&file=paper.pdf'};
test('representation URLs bind source identity, exact file and viewer action',()=> {
  expect(representationURL(asset)).toBe(asset.url);
  expect(representationURL({...asset,policy:'source_stream',url:'https://archive.org/download/paper/paper.pdf'})).toBe('https://archive.org/download/paper/paper.pdf');
  for (const value of [
    {...asset,url:'//evil.invalid/api/catalog/v1/items/paper/representations/pdf/media'},
    {...asset,url:asset.url+'&collection=other'},
    {...asset,url:asset.url.replace('action=read','action=download')},
    {...asset,url:asset.url.replace('file=paper.pdf','file=other.pdf')},
    {...asset,policy:'source_stream',url:'https://archive.org/download/other/paper.pdf'},
    {...asset,source_url:'https://evil.invalid/details/paper'}
  ]) expect(()=>representationURL(value)).toThrow();
});


test('download URLs require the explicit action returned by the policy resolver',()=> {
  const download = {...asset,url:asset.url.replace('action=read','action=download')};
  expect(representationURL(download,'download')).toBe(download.url);
  expect(()=>representationURL(download)).toThrow();
  expect(()=>representationURL(asset,'download')).toThrow();
  expect(()=>representationURL({...download,url:download.url+'&action=read'},'download')).toThrow();
});
