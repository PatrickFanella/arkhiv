import { useCallback, useState } from 'react';
import { AudioPlayer, useAudioStation, useAudioChannelChat } from '@subcult/archive-viewer/audio';
import { ChannelPanel, Screen, useViewer, usePersonal, useChannelChat } from '@subcult/archive-viewer';
import ReplayDiscussion from './ReplayDiscussion';

function AudioChannel({collection}: {collection:string}) {
  const station = useAudioStation(collection);
  const personal = usePersonal(collection);
  const [playbackState,setPlaybackState] = useState('loading');
  const playbackChanged = useCallback((state:string)=>setPlaybackState(state),[]);
  const chat = useAudioChannelChat(collection,station.session,playbackState!=='synchronized',personal);
  return <section aria-label="Shared audio station"><h3>Shared audio station</h3><p>{station.station?.reason || (station.station?.state === 'playing' ? 'A scheduled recording is playing.' : 'No current program.')}</p>
    {station.error && <p role="alert">{station.error}</p>}
    <button disabled={station.station?.state !== 'playing'} onClick={()=>{void station.join();}}>Tune in</button>
    {station.session && <><button onClick={station.leave}>Leave station</button><AudioPlayer {...station.session} autoPlay onPlaybackState={playbackChanged} onRejoin={()=>{setPlaybackState('loading');void station.rejoin();}} onEnded={()=>{setPlaybackState('loading');void station.join();}}/></>}
    <ReplayDiscussion chat={chat} personal={personal} media="audio"/>
  </section>;
}
function VideoChannel({collection}: {collection:string}) {
  const viewer = useViewer(collection);
  const personal=usePersonal(collection);
  const chat=useChannelChat(collection,viewer.session,viewer.paused,personal);
  return <section aria-label="Shared video replay"><h3>Shared video replay</h3><Screen viewer={viewer}><p>Join the scheduled replay for this collection.</p></Screen><ChannelPanel viewer={viewer}/><ReplayDiscussion chat={chat} personal={personal} media="video"/></section>;
}
export default function CollectionChannels({collection,media}: {collection:string;media:string[]}) {
  return <div className="collection-channels">{media.includes('audio') && <AudioChannel key={`audio:${collection}`} collection={collection}/>}{media.includes('video') && <VideoChannel key={`video:${collection}`} collection={collection}/>}</div>;
}
