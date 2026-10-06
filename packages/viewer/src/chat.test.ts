import { expect,it } from 'vitest';
import { chatContextMatches, type ChatContext, type ChatRoom } from './chat.js';

it('audio posting requires its exact joined representation and current program',()=>{
 const room:ChatRoom={collection:'mixed',media:'audio',read:true,post:true,retention_days:30,poll_seconds:10,moderation_contact:'operator',current:{media:'audio',revision:'audio-1',start:'2026-10-06T00:00:00Z',item:'recording',representation:'continuous',episode:'',original_date:'1977-05-08'}};
 const session:ChatContext={mode:'channel',programKey:'audio-1:2026-10-06T00:00:00Z',item:'recording',representation:'continuous'};
 expect(chatContextMatches(room,session,false,'audio')).toBe(true);
 expect(chatContextMatches(room,session,true,'audio')).toBe(false);
 expect(chatContextMatches(room,{...session,mode:'ondemand'},false,'audio')).toBe(false);
 expect(chatContextMatches(room,{...session,representation:'foreign'},false,'audio')).toBe(false);
 expect(chatContextMatches(room,{...session,item:'foreign'},false,'audio')).toBe(false);
 expect(chatContextMatches(room,{...session,programKey:'audio-0:2026-10-06T00:00:00Z'},false,'audio')).toBe(false);
 expect(chatContextMatches({...room,post:false},session,false,'audio')).toBe(false);
 expect(chatContextMatches(room,session,false,'video')).toBe(false);
});
it('existing video rooms remain compatible without new media fields',()=>{
 const room:ChatRoom={collection:'legacy',read:true,post:true,retention_days:30,poll_seconds:10,moderation_contact:'operator',current:{revision:'video-1',start:'start',episode:'episode',original_date:'unknown'}};
 expect(chatContextMatches(room,{mode:'channel',programKey:'video-1:start',episode:'episode'},false,'video')).toBe(true);
 expect(chatContextMatches(room,{mode:'channel',programKey:'video-1:start',item:'episode',representation:'audio'},false,'audio')).toBe(false);
});
