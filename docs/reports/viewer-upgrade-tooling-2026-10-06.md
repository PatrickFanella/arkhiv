# Independent viewer upgrade qualification, October 6, 2026

The current six-consumer registry now uses each product's SDK-only receipt and separate retained core inputs. It preserves the previous registration and dated adoption evidence. The upgrade/proposal tools accept the independent viewer receipt without inventing another engine release. Viewer checks bind the SDK tarball checksum, compatible core version, required API/media/modules and exact retained core image pin before mutation.

Five isolated regression tests passed in the `vhness` profile on Sputnik. They cover lane selection, SDK-only changes, repeated no-op preparation, incompatible metadata/paths/core contracts, rollback after failed or core-mutating checks, and idempotent proposal delivery with only the three SDK files staged. Existing combined/core proposal tests remain part of the full required gate.

The downloaded SDK 0.4.2 artifact matched the published SHA-256. The real preparation command returned `already applied` without file changes or PR creation in all six clean, independently qualified consumer checkouts. Video consumers use their merged SDK 0.4.2 branches; Dedhed and both print consumers use their ready SDK adoption branches. This validates current registry paths and retained core 0.3.0 pins. It does not claim that those ready PRs merged or that native backends changed.

Hosted delivery for this new lane remains owner-waived while Kvant is down. A future new SDK release still needs its independently verified artifact, consumer gates, reviewed proposal and product qualification. No duplicate PR or live deployment was created for an already applied release.

The required `make verify && make verify-curation` command then passed in the same isolated `vhness` profile. Results included Go/race checks with disposable PostgreSQL integration, 45 SDK tests, packed ESM checks, 56 Python tests with the two separate real personal-restore cases skipped in the ordinary suite, four reference frontend tests, builds and all 19 frozen-uv curation tests plus lint/format/JavaScript checks. The separately qualified personal restore evidence is unchanged. Hosted CI was not run.
