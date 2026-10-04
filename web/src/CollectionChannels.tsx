import { AudioPlayer, useAudioStation } from '@subcult/archive-viewer/audio';
import { ChannelPanel, Screen, useViewer } from '@subcult/archive-viewer';

function AudioChannel({collection}: {collection:string}) {
  const station = useAudioStation(collection);
  return <section aria-label="Shared audio station"><h3>Shared audio station</h3><p>{station.station?.reason || (station.station?.state === 'playing' ? 'A scheduled recording is playing.' : 'No current program.')}</p>
    {station.error && <p role="alert">{station.error}</p>}
    <button disabled={station.station?.state !== 'playing'} onClick={()=>{void station.join();}}>Tune in</button>
    {station.session && <><button onClick={station.leave}>Leave station</button><AudioPlayer {...station.session} autoPlay onRejoin={()=>{void station.rejoin();}} onEnded={()=>{void station.join();}}/></>}
  </section>;
}
function VideoChannel({collection}: {collection:string}) {
  const viewer = useViewer(collection);
  return <section aria-label="Shared video replay"><h3>Shared video replay</h3><Screen viewer={viewer}><p>Join the scheduled replay for this collection.</p></Screen><ChannelPanel viewer={viewer}/></section>;
}
export default function CollectionChannels({collection,media}: {collection:string;media:string[]}) {
  return <div className="collection-channels">{media.includes('audio') && <AudioChannel key={`audio:${collection}`} collection={collection}/>}{media.includes('video') && <VideoChannel key={`video:${collection}`} collection={collection}/>}</div>;
}
