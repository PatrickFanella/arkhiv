# Arkhiv public SEO frontend deployment

Verified main pushes and manual main dispatches build and deploy only the static
`arkhiv-seo-web` frontend on Almaz. Its producer is `web/`: the locked workspace
build, `seo:render public/site.json https://arkhiv.subcult.tv/`, then
`web/Dockerfile.static`. The reference-image publisher and platform/core deployer
retain their independent targets and release procedures.

The SEO job waits for the existing application/database verification and checks
the owned curation gate before activation. PRs build the actual prerendered image
and check nginx, the catalog headline, canonical URL, large preview card, selected
icons and preview image. The image also checks its own `/healthz` endpoint.

## Almaz registration and first adoption

The original SEO container is standalone. Qualify adoption with its retained
immutable image before allowing CI activation. `deploy/seo-compose.yml` records
the existing user, external projects network, read-only filesystem, bounded
tmpfs, capability restrictions, restart policy, memory/swap, CPU, process limit
and logging policy. It publishes no host port and mounts no application data.

Keep the original inspect, image ID, public file hashes and related-service
identities in a private, timestamped recovery directory. Render the merged
Compose template with `ARKHIV_SEO_IMAGE` set to that immutable image ID. First run
an isolated fixture using its own internal network and unique container name;
compare resources, tmpfs, user, command, entrypoint and served bytes with the live
container. Preserve the selected public assets and any existing proxy behavior.

The host registration uses these operator-owned paths:

- Profile: `/home/onnwee/.config/fleet-deploy/arkhiv-seo.json`.
- Resolved baseline: `/home/onnwee/.config/fleet-deploy/arkhiv-seo-compose.json`.
- Receipts and lock: `/home/onnwee/.local/state/fleet-deploy/arkhiv-seo/`.
- Compose project/service: `arkhiv-seo-ci` / `web`.
- Container: `arkhiv-seo-web`.

After the source checks pass and the change merges, hold the app's `deploy.lock`,
confirm the original container/image still match the saved observation, stop it,
rename it to the retained name recorded in the recovery directory, and start the
reviewed baseline with the same immutable image. Compare the complete
runtime guard and public bytes before releasing the lock. Keep the original
container and image. If adoption fails, remove only the newly created managed
container, restore the retained container's original name, and start it again.

An unadopted standalone container fails the receiver's Compose ownership check
before activation. If CI arrives first, finish the reviewed adoption and retry
the current main job. Do not weaken its guard or select the platform deployer.

## Credentials and subsequent deployments

The dedicated `HOMELAB_DEPLOY_KEY` and `HOMELAB_KNOWN_HOSTS` secrets belong to this
frontend. Its SSH key is source-restricted and forces the reviewed fleet receiver
with this one profile. The platform's existing `ALMAZ_DEPLOY_KEY` stays separate.
Key custody belongs to the homelab operator; the initial rotation deadline is
2027-02-03. Private key values and raw host diagnostics stay out of source,
workflow artifacts and public reports.

The receiver permits only `web`, requires the current repository main, validates
source/revision labels, and retains one accepted image per commit. Retries query
and reuse that image instead of rebuilding a different image under the same
commit. Every activation checks runtime configuration and complete public HTML,
icons and preview bytes, with immutable-image rollback on failure. The CI job
independently compares public HTML and the favicon after activation.

Use the exact rollback argv from the relevant private receipt only after checking
for newer deployments and host changes. Keep the referenced image and baseline.
Core/API/database, curation workers, media acquisition, source-streaming policy,
consumer pins, public routes and full playback qualification retain their own
deployment and acceptance gates.
