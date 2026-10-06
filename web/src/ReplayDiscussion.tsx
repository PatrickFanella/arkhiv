import { useState } from 'react';
import type { ChannelChat, Personal } from '@subcult/archive-viewer';

export default function ReplayDiscussion({chat,personal,media}: {chat:ChannelChat;personal:Personal;media:'video'|'audio'}) {
  const [body,setBody]=useState(''),[reason,setReason]=useState('');
  const moderator=media==='audio' ? personal.user?.audio_moderator : personal.user?.moderator;
  if(!chat.enabled)return null;
  return <section aria-label={`${media} replay discussion`}>
    <h4>{media==='audio' ? 'Audio station' : 'Video channel'} discussion</h4>
    <p>Scheduled replay · {chat.room?.current?.original_date || 'Original date unknown'} · {chat.room?.current?.revision || 'No current program'}</p>
    {!chat.canPost&&<p>Read only. Sign in and join the current program with playback running to post.</p>}
    {!chat.room?.read&&<p>Discussion is disabled.</p>}
    {chat.error&&<p role="alert">{chat.error}</p>}
    <ol>{chat.messages.map(message=><li key={message.id}>
      <p><strong>{message.alias}</strong> · <time>{message.created_at}</time> · Original date {message.original_date || 'unknown'}</p>
      <p>{message.body}</p><small>Replay {message.revision} · {message.start}</small>
      {personal.user&&(message.user_id===personal.user.id ? <button disabled={chat.busy} onClick={()=>{void chat.remove(message.id);}}>Delete my message</button> : <>
        <button disabled={chat.busy||!reason.trim()} onClick={()=>{void chat.report(message.id,reason);}}>Report message</button>
        <button disabled={chat.busy} onClick={()=>{void chat.block(message.user_id);}}>Block viewer</button>
      </>)}
    </li>)}</ol>
    {chat.hasOlder&&<button onClick={chat.older}>Earlier discussion</button>}{chat.history&&<button onClick={chat.current}>Current discussion</button>}
    <form onSubmit={event=>{event.preventDefault();void chat.send(body).then(sent=>{if(sent)setBody('');});}}>
      <label>{media==='audio' ? 'Audio station message' : 'Video channel message'}<textarea value={body} maxLength={1000} disabled={!chat.canPost||chat.busy} onChange={event=>setBody(event.target.value)}/></label>
      <button disabled={!chat.canPost||chat.busy||!body.trim()}>Send message</button>
    </form>
    {personal.user&&<label>Report or moderation reason<input value={reason} maxLength={4096} onChange={event=>setReason(event.target.value)}/></label>}
    {chat.blocked.map(id=><p key={id}>Blocked viewer <button disabled={chat.busy} onClick={()=>{void chat.block(id,true);}}>Unblock</button></p>)}
    {moderator&&<><button disabled={chat.busy} onClick={()=>{void chat.reviewReports();}}>Review {media} reports</button>{chat.reports.map(report=><article key={report.id}>
      <p>{report.reason}</p><p>{report.body}</p>
      <button disabled={chat.busy||!reason.trim()} onClick={()=>{void chat.moderate('hide',report.message_id,reason);}}>Withdraw message</button>
      <button disabled={chat.busy||!reason.trim()} onClick={()=>{void chat.moderate(report.resolved?'reopen':'resolve',report.id,reason);}}>{report.resolved?'Reopen':'Resolve'} report</button>
    </article>)}</>}
    <p>Messages are retained for {chat.room?.retention_days || 30} days. {chat.room?.moderation_contact}</p>
  </section>;
}
