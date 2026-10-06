# Channel continuity qualification limit — October 6, 2026

Issue #108 remains open. Two independently mounted production-template bundles
joined the same generated private schedule in the native collaborative browser.
Both source elements loaded the scheduled asset and seek position, but paused
soon after requesting playback. The controls displayed “Paused · Channel
continues” rather than claiming synchronized playback. Trusted resume clicks
were also followed by pauses. Both frame documents reported visible state and
the video rectangles were within their viewports. This refines the earlier
volume report's observation of pauses while controls were outside the viewport:
viewport position alone has not explained the behavior.

The qualification recorded 67 periodic samples and media events. No sampled pair
was continuously playing. Instrumented calls to the media element's `pause()`
method did not record a call, but that does not determine the cause of a native
pause event. A main-frame trial also showed playback, buffering and pauses.
No player change was made on an unverified assumption about browser policy.
The inline preview reported `visible: false` while page documents reported
visible state; this remains an environment observation rather than proof of a
product defect or an excuse to declare the check passed.

Original dates and scheduled-replay labels remained visible. Channel elements
had no native scrub control; on-demand retained independent native seeking.
Volume/mute, rejected-autoplay retry and reviewed artwork have their own completed
reports. Unit and integration checks cover the three-second measured drift rule,
fresh rejoin and stale-response guards. They do not replace continuous actual
two-viewer playback, suspension/resume or assistive-technology evidence.

Only the operator extended the private generated schedule when its original
window exhausted; historical revisions remained retained. Viewer interactions
did not write shared schedules. Raw browser samples and events are retained
outside Git in the qualification ledger.
