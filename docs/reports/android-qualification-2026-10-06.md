# Android qualification, October 6, 2026

The owner approved SDK/image licenses and a 12 GiB setup allowance. Sputnik now
has the checksum-verified Android command-line tools and API 35 Google APIs
x86_64 revision 9 image. Two independent Android 15 AVDs booted through KVM and
opened in T3's Device panel. SDK and allocated AVD storage measured about 7.4 GiB;
the retained verified image installer adds about 1.6 GiB. iOS is unavailable on
this Linux host without macOS/Xcode.

The hub defaulted to a macOS SDK path despite running on Linux. A compatibility
symlink points that path at the approved `~/Android/Sdk` installation. User PATH
launchers invoke the actual SDK tools with that SDK root. No T3 restart or shared
service configuration change was required. The emulator streams and CLI sessions
remain separately scoped to each serial. Loopback forwarding exposes only the
owned private preview to these devices.

## Two-device generated channel trial

Both Chrome instances opened the neutral reference interface and joined its
private generated schedule through the visible controls. The schedule uses two
20-second H.264/AAC test patterns and sine tones; no source media is involved.
Read-only Chrome debugging sampled each actual video element and the backend
clock for 180 seconds. UI actions used the T3-provided agent-device sessions.
The native desktop preview automation host had previously reported unavailable.

There were 154 samples, including 148 with both elements unpaused. Of those,
108 aligned with the same backend program and had both media elements fully
ready. Maximum simultaneous inter-device drift in that set was 1.879 seconds,
across eight distinct program starts. Samples with different assets around a
boundary are recorded separately; their raw clock difference reached 19.187
seconds and must not be called simultaneous drift within one program.

Two aligned, fully ready samples exceeded the three-second backend offset
tolerance; the largest individual offset error was 5.706 seconds. Transient
buffering and program transitions are retained in the raw ledger. This trial
therefore provides real Android decode and repeated-boundary evidence, but does
not establish uninterrupted synchronization within tolerance for the whole run.
It does not close #108 or establish source audiovisual fidelity, suspension
recovery, assistive-technology coverage or a sustained operational soak.

Raw measurements and SDK setup receipts stay outside Git in the private issue
intake workspace. Branded account/mobile controls and the approved real AVI pilot
have their own qualification work; setup and this generated trial do not prove
their completion.

## Bill Nye account checks on independent devices

Both Android Chrome instances separately completed optional private fixture OIDC
sign-in through the Bill Nye SDK 0.4.2 bundle and displayed the same account alias.
Device A retained one anonymous browser-local favorite. Device B saved and removed
a second account favorite. An identical stale add on A was safely idempotent;
a subsequent stale change to the other item produced the actual conflict message
and retained the pending removal rather than overwriting B's changes.

Reloading A preserved the pending item. Its visible `Refresh and apply these
changes` control rebased and applied the removal, then cleared the pending list.
Reloading B displayed both account favorites as removed. `Sign out all devices`
on A returned A to anonymous viewing; reloading B also returned its independent
session to anonymous viewing. Browser-local favorites remain separate from these
account changes. UI actions used each device's own T3 config/session flags.

These checks use a private fixture provider and deployment. They do not establish
live cross-product deployment isolation, provider outage/recovery, iOS behavior,
mobile exports/deletion or the other brands' independent device acceptance.
Bill Nye #10 and the corresponding product issues remain open for those gaps.
