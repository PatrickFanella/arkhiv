# Shared archive viewer

`@subcult/archive-viewer` provides the typed video API, playback/schedule model,
viewer hook and composed player/library/channel components. React is supplied by
the consumer. The package includes no brand assets, global CSS or page layout.

```tsx
import { useViewer, Screen, ChannelPanel, Library } from '@subcult/archive-viewer';

function Archive() {
  const viewer = useViewer('democracy_now_vid');
  return <main>
    <Screen viewer={viewer}><p>Select an episode.</p></Screen>
    <ChannelPanel viewer={viewer} />
    <Library viewer={viewer} />
  </main>;
}
```

Consumers own page composition and CSS for the documented classes already used
by the video apps. `Player` uses native video controls, source attribution,
caption controls and explicit loading/failure messages. A keyed player owns one
asset; channel clock updates do not reload it. `useViewer` exposes listener-local
state and commands independently of page layout. Set
`useViewer(collection, { scrollTarget: false })` for consumer-owned navigation,
or supply a custom selector. The default retains the existing `#screen` scroll
behavior and honors reduced motion.

The root entry supports video API version `1`. `Compatibility` verifies the exact
collection for standalone consumers. Video channel/playback requests name the
selected collection; typed print/audio entries have their own contracts. Consumers
own mixed media page composition. Acquisition/publication policies stay on the
server.

Build with the pinned root pnpm version using `pnpm build:packages`. Packages
ship compiled ESM and declarations; consumers need no sibling source checkout.
The legacy `scripts/export_viewer.py` remains a compatibility path until every
consumer adopts released packages. Edit these sources; web compatibility barrels
and legacy exported copies are not separate editing authorities.

GPL-3.0-only software. Media and identity assets retain their separate owners.

The unreleased `0.2.0` print entry exports `PrintReader` from
`@subcult/archive-viewer/print`. Consumers supply its PDF backend and worker
assets; importing the root entry does not load a PDF runtime. See the engine's
[print module contract](../../docs/reference/print-module.md) for indexing and
access requirements.

The audio entry exports `AudioPlayer`, reviewed set bounds and station clock
helpers from `@subcult/archive-viewer/audio`. The engine supplies a verified
continuous recording URL; the consumer owns station polling and navigation.
Source credits, warnings and track order remain in the recording contract.

The catalog entry exports `useCatalog`, scoped metadata query types and reviewed
representation URL validation from `@subcult/archive-viewer/catalog`. Consumers
own collection filters, item detail, navigation and media composition. The hook
hides superseded results immediately and cancels their requests. The neutral
reference frontend composes all four entries; a single-media brand can import
only the entries it uses.
