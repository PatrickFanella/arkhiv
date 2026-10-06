import { describe,it,expect } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { AudioPlayer, audioPlaybackState, audioURL, setBounds, radioOffset, type AudioAsset } from './audio.js';
const asset: AudioAsset = {policy:'local_preserve',url:'/api/catalog/v1/items/concert/representations/recording/audio/media?collection=audio-fixture',source_url:'https://archive.org/details/source',snapshot_sha256:'a'.repeat(64),sha256:'b'.repeat(64),duration:20,recording:{id:'recording',concert_id:'concert',artist:'Fixture',date:'2026-10-03',venue:'Fixture venue',location:'',title:'Synthetic recording',kind:'concert',source:'Synthetic tone',taper:'Fixture generator',transferer:'',tracks:[{position:0,title:'One',start:0,duration:10},{position:1,title:'Two',start:10,duration:10}],sets:[],duration:20,warnings:['Set boundaries unreviewed'],synthetic:true}};
describe('continuous audio contracts',()=>{
 it('retains recording/credit controls without downloading or public relay',()=>{
  const html=renderToStaticMarkup(<AudioPlayer asset={asset}/>);
  expect(html).toContain('Recording position');expect(html).toContain('Fixture generator');expect(html).toContain('Set boundaries unreviewed');expect(html).not.toContain('download=');
 });
 it('rejects unreviewed sets, external assets and malformed station clocks',()=>{
  expect(()=>setBounds(asset.recording,{title:'Inferred set',first:0,last:1,reviewed:false})).toThrow();
  expect(()=>audioURL({...asset,url:'https://example.org/private.wav'})).toThrow();
  expect(()=>audioURL({...asset,url:asset.url+'&collection=foreign'})).toThrow();
  expect(()=>radioOffset(NaN,1000,0,20,2000)).toThrow();
  expect(radioOffset(5,1000,100,20,2000)).toBeCloseTo(6.1);
  expect(setBounds(asset.recording,{title:'Reviewed second track',first:1,last:1,reviewed:true})).toEqual([10,20]);
 });
});

it('scheduled replay has no seek slider or enabled track jumps and retains original dates',()=>{
 const html=renderToStaticMarkup(<AudioPlayer asset={asset} station={{offset:5,receivedAt:1000,latency:0,duration:20,programKey:'fixture'}} onRejoin={()=>undefined}/>);
 expect(html).not.toContain('Recording position');expect(html).toContain('Resume at current program');expect(html).toContain('Rejoin current program');expect(html).toContain('2026-10-03');expect(html).toMatch(/button disabled=""/);expect(html).toContain('Mute');
});
it('audio synchronization requires ready active playback near the station clock',()=>{
 const station={offset:5,receivedAt:1000,latency:0,duration:20,programKey:'fixture'},media={paused:false,ended:false,seeking:false,readyState:4,currentTime:6};
 expect(audioPlaybackState(media,station,2000)).toBe('synchronized');
 expect(audioPlaybackState({...media,currentTime:0},station,2000)).toBe('joining');
 expect(audioPlaybackState({...media,seeking:true},station,2000)).toBe('joining');
 expect(audioPlaybackState({...media,paused:true},station,2000)).toBe('paused');
 expect(audioPlaybackState({...media,readyState:2},station,2000)).toBe('buffering');
});
