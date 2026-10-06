const $ = (selector) => document.querySelector(selector);
const escape = (value) =>
  String(value ?? "").replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );
const state = {
  collections: [],
  collection: null,
  tab: "pending",
  page: 1,
  filter: "",
  searches: [],
  items: [],
  runs: [],
  total: 0,
  generation: 0,
  catalog: { items: [], total: 0 },
  core: { target_collection: "", attachments: [] },
  media: "",
  coreError: "",
};
let noticeTimer;
let refreshTimer;
let opener;

async function api(path, options = {}) {
  const response = await fetch(`/api${path}`, {
    ...options,
    headers: { "Content-Type": "application/json", ...options.headers },
  });
  if (!response.ok) {
    let message = `Request failed (${response.status}).`;
    try {
      const data = await response.json();
      message =
        typeof data.detail === "string"
          ? data.detail
          : data.detail
              ?.map((e) => e.msg.replace(/^Value error, /, ""))
              .join(" · ") || message;
    } catch {
      /* The fallback contains the HTTP status. */
    }
    throw new Error(message);
  }
  return response.json();
}

function notify(message, error = false) {
  clearTimeout(noticeTimer);
  $("#notice").textContent = message;
  $("#notice").className = error ? "error-notice" : "";
  $("#notice").hidden = false;
  noticeTimer = setTimeout(
    () => {
      $("#notice").hidden = true;
    },
    error ? 12000 : 5000,
  );
}

function date(value) {
  return value
    ? new Date(value * 1000).toLocaleString([], {
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "2-digit",
      })
    : "Not run yet";
}
function current() {
  return state.collections.find((c) => c.id === state.collection);
}
function modal(title, content) {
  opener = document.activeElement;
  $("#modal-content").innerHTML =
    `<div class="modal-head"><h2>${escape(title)}</h2><button class="close" data-action="close" aria-label="Close dialog">×</button></div><div class="modal-body">${content}</div>`;
  if (!$("#modal").open) $("#modal").showModal();
}
function closeModal() {
  $("#modal").close();
}
$("#modal").addEventListener(
  "close",
  () => opener?.isConnected && opener.focus(),
);

async function refresh(quiet = false) {
  const generation = ++state.generation;
  const selected = state.collection;
  try {
    const collections = await api("/collections");
    if (generation !== state.generation) return;
    state.collections = collections;
    if (!state.collection && collections.length)
      state.collection = collections[0].id;
    renderSidebar();
    if (!state.collection) {
      render();
      return;
    }
    const id = state.collection;
    const [searches, runs, result] = await Promise.all([
      api(`/collections/${id}/searches`),
      api(`/collections/${id}/runs`),
      api(
        `/collections/${id}/items?status=${["pending", "accepted", "rejected"].includes(state.tab) ? state.tab : "pending"}&page=${state.page}&q=${encodeURIComponent(state.filter)}`,
      ),
    ]);
    if (
      generation !== state.generation ||
      (selected && selected !== state.collection)
    )
      return;
    Object.assign(state, {
      searches,
      runs,
      items: result.items,
      total: result.total,
    });
    if (state.tab !== "catalog" && state.page > 1 && !result.items.length) {
      state.page--;
      return refresh(quiet);
    }
    if (state.tab === "catalog") {
      try {
        const [catalog, core] = await Promise.all([
          api(`/core/catalog?page=${state.page}&q=${encodeURIComponent(state.filter)}&media=${encodeURIComponent(state.media)}`),
          api(`/collections/${id}/core`),
        ]);
        if (generation !== state.generation) return;
        Object.assign(state, { catalog, core, coreError: "" });
      } catch (error) {
        if (generation !== state.generation) return;
        state.coreError = error.message;
      }
    }
    render();
  } catch (error) {
    if (generation !== state.generation) return;
    notify(error.message, true);
    if (!quiet)
      $("#workspace").innerHTML =
        '<div class="empty"><h2>Workspace unavailable</h2><p>Could not load your collections. Check that the server is running, then retry.</p><button class="button" data-action="refresh">Retry</button></div>';
  } finally {
    $("#workspace").setAttribute("aria-busy", "false");
  }
}

function renderSidebar() {
  $("#collections").innerHTML = state.collections.length
    ? state.collections
        .map(
          (c) =>
            `<button class="collection-link ${c.id === state.collection ? "active" : ""}" data-action="select" data-id="${c.id}" ${c.id === state.collection ? 'aria-current="page"' : ""}><span>${escape(c.name)}</span><span class="count" aria-label="${c.counts.pending} awaiting review">${c.counts.pending}</span></button>`,
        )
        .join("")
    : '<p class="side-empty">Your first collection starts here.</p>';
}

function render() {
  const collection = current();
  if (!collection) {
    $("#workspace").innerHTML =
      `<section class="welcome"><span class="eyebrow">A LITTLE ORDER IN THE ARCHIVE</span><h1>Find the pieces.<br>Build the collection.</h1><p>Turn Internet Archive searches into a collection worth keeping. Let saved searches find the material. You decide what belongs.</p><button class="button primary" data-action="new-collection">+ Create your first collection</button><div class="steps"><div><span class="step-no">01</span><strong>Follow a topic</strong>Save a search with the filters that matter.</div><div><span class="step-no">02</span><strong>Review the finds</strong>Keep the good matches. Pass on the rest.</div><div><span class="step-no">03</span><strong>Keep it growing</strong>Scheduled searches bring new items to you.</div></div><p class="footer-note">Selections stay in your private workspace. Items remain hosted on archive.org.</p></section>`;
    return;
  }
  const tabs = [
    ["pending", "Review queue"],
    ["accepted", "Collection"],
    ["rejected", "Passed"],
    ["searches", "Searches"],
    ["runs", "Activity"],
    ["catalog", "Combined catalog"],
  ];
  $("#workspace").innerHTML =
    `<section class="heading"><div><span class="eyebrow">LOCAL CURATED COLLECTION</span><h1>${escape(collection.name)}</h1><p>${escape(collection.description || "A collection built one good find at a time.")}</p></div><div class="heading-actions"><button class="button" data-action="edit-collection">Edit collection</button><button class="button primary" data-action="export">Export selection ↗</button></div></section><nav class="tabs" aria-label="Collection views">${tabs.map(([key, label]) => `<button class="tab ${state.tab === key ? "active" : ""}" data-action="tab" data-tab="${key}" ${state.tab === key ? 'aria-current="page"' : ""}>${label}${key in collection.counts ? `<span class="badge">${collection.counts[key]}</span>` : ""}</button>`).join("")}</nav><div id="view">${state.tab === "catalog" ? catalogView() : state.tab === "searches" ? searchView() : state.tab === "runs" ? runView() : itemView()}</div>`;
  document.querySelectorAll("img[data-cover]").forEach((img) =>
    img.addEventListener(
      "error",
      () => {
        img.hidden = true;
      },
      { once: true },
    ),
  );
}

function itemView() {
  const labels = {
    pending: [
      "Make room for the good finds.",
      "Accept items to add them to your collection. Pass on anything that doesn’t belong.",
    ],
    accepted: [
      "Your curated selection.",
      "The items you chose to keep, with links to their original Archive pages.",
    ],
    rejected: [
      "The ones you passed on.",
      "Decisions are remembered. Return an item to the queue if you change your mind.",
    ],
  };
  const [heading, sub] = labels[state.tab];
  let content;
  if (!state.items.length) {
    let title = "Nothing here yet.";
    let text = "Accepted items will appear here as you review your finds.";
    let action = "";
    if (state.filter) {
      title = "No items match this filter.";
      text = "Try another title or creator, or clear the filter.";
      action =
        '<button class="button" data-action="clear-filter">Clear filter</button>';
    } else if (state.tab === "pending") {
      title = state.searches.length
        ? "You’re all caught up."
        : "Give your collection a starting point.";
      text = state.searches.length
        ? "Run a saved search to discover candidates, or let its schedule bring new finds to the queue."
        : "Save an Archive search and choose your filters. Matching items will arrive here for review.";
      action = `<button class="button primary" data-action="${state.searches.length ? "show-searches" : "new-search"}">${state.searches.length ? "Open saved searches" : "+ Add a saved search"}</button>`;
    } else if (state.tab === "rejected") {
      text =
        "Items you pass on stay here and won’t be re-added to the review queue by discovery.";
    }
    content = `<div class="empty"><div class="symbol" aria-hidden="true">${state.tab === "accepted" ? "▤" : "⌕"}</div><h2>${title}</h2><p>${text}</p>${action}</div>`;
  } else
    content = `<div class="item-list">${state.items.map(itemCard).join("")}</div><div class="pagination"><span>${state.total} ${state.tab === "pending" ? "to review" : "items"} · Page ${state.page} of ${Math.ceil(state.total / 24)}</span><div><button class="button small" data-action="previous" ${state.page <= 1 ? "disabled" : ""}>← Previous</button><button class="button small" data-action="next" ${state.page * 24 >= state.total ? "disabled" : ""}>Next →</button></div></div>`;
  return `<section class="toolbar"><div><h2>${heading}</h2><p>${sub}</p></div><form id="filter-form"><input class="filter" name="filter" aria-label="Filter by title or creator" placeholder="Filter by title or creator…" value="${escape(state.filter)}"></form></section>${content}<p class="footer-note">${state.searches.length} saved ${state.searches.length === 1 ? "search" : "searches"} · Review decisions are kept across discovery runs. <button class="button small quiet" data-action="show-searches">Manage searches →</button></p>`;
}

function itemCard(item) {
  const id = escape(item.identifier);
  const url = `https://archive.org/details/${encodeURIComponent(item.identifier)}`;
  return `<article class="item"><div class="cover"><img data-cover loading="lazy" src="https://archive.org/services/img/${encodeURIComponent(item.identifier)}" alt=""></div><div><div class="item-type"><span>${escape(item.media_type || "Archive item")}</span>${item.year ? `<span>· ${escape(item.year)}</span>` : ""}</div><h3><button data-action="detail" data-id="${id}">${escape(item.title)}</button></h3><p class="creator">${escape(item.creator || "Creator not listed")}</p><p class="description">${escape(item.description || "No description provided.")}</p>${item.matches.map((m) => `<span class="match">↳ ${escape(m.name)}</span>`).join("")}${item.notes ? '<span class="match">Has a note</span>' : ""}</div><div class="item-actions">${state.tab === "pending" ? `<button class="button primary small" data-action="decide" data-id="${id}" data-status="accepted">✓ Accept</button><button class="button small reject" data-action="decide" data-id="${id}" data-status="rejected">Pass</button>` : `<button class="button small" data-action="decide" data-id="${id}" data-status="pending">Review again</button>`}<a href="${url}" target="_blank" rel="noopener noreferrer">View on Archive ↗</a></div></article>`;
}

function catalogView() {
  if (state.coreError) return `<div class="empty"><h2>Catalog unavailable</h2><p>${escape(state.coreError)}</p><button class="button" data-action="refresh">Retry</button></div>`;
  return `<section class="toolbar"><div><h2>Choose from the combined catalog.</h2><p>Attach selected editions or recordings, then review them in your queue. Access approval stays separate.</p></div><form id="filter-form"><input class="filter" name="filter" aria-label="Search combined catalog" placeholder="Search catalog…" value="${escape(state.filter)}"><select name="media" aria-label="Catalog media">${[["", "All media"], ["print", "Print"], ["audio", "Audio"], ["video", "Video"]].map(([v,l]) => `<option value="${v}" ${state.media === v ? "selected" : ""}>${l}</option>`).join("")}</select><button class="button">Search</button></form></section><form id="core-binding-form"><label for="core-target">Target core collection</label><input id="core-target" name="target_collection" required pattern="[A-Za-z0-9][A-Za-z0-9._\\-]{0,199}" maxlength="200" value="${escape(state.core.target_collection)}" placeholder="Collection identifier"><button class="button small">Save destination</button><p class="help">The manifest names this destination. An operator reviews it before changing the core.</p><p id="form-error" class="error" role="alert" hidden></p></form><div class="item-list">${state.catalog.items.map(item => `<article class="item"><div class="cover"><span>${escape([...new Set(item.representations.map(r => r.media))].join(" / "))}</span></div><div><h3>${escape(item.title)}</h3><p>${escape(item.creator)}</p><p class="description">${escape(item.description)}</p><p class="help">${item.representations.length} available representations${state.core.attachments.some(a => a.core_item_id === item.id) ? " · Attached" : ""}</p></div><div class="item-actions"><button class="button small primary" data-action="attach-core" data-id="${escape(item.id)}">Review attachment</button></div></article>`).join("") || '<div class="empty"><h2>No catalog items match.</h2><p>Try another search or media type.</p></div>'}</div><div class="pagination"><span>${state.catalog.total} items · Page ${state.page}</span><div><button class="button small" data-action="previous" ${state.page <= 1 ? "disabled" : ""}>← Previous</button><button class="button small" data-action="next" ${state.page * 24 >= state.catalog.total ? "disabled" : ""}>Next →</button></div></div><p class="footer-note">${state.core.attachments.length} retained attachments. Export includes accepted selections whose source evidence still matches.</p>`;
}

function attachmentForm(id) {
  const item = state.catalog.items.find(item => item.id === id);
  const previous = state.core.attachments.find(a => a.core_item_id === id);
  if (!item) return;
  modal("Attach catalog item", `<h2>${escape(item.title)}</h2><p>Choose the editions or recordings that belong here. Each Archive source will enter your review queue; previous review decisions stay saved.</p><form id="core-attachment-form" data-id="${escape(id)}">${item.representations.map(r => `<label class="check"><input type="checkbox" name="representations" value="${escape(r.id)}" ${!previous || previous.evidence.representations.some(p => p.id === r.id) ? "checked" : ""}><span>${escape(r.media)} · ${escape(r.file)}<br><small>Archive source: ${escape(r.source_item_id)}</small></span></label>`).join("")}<label for="attachment-notes">Why this belongs</label><textarea id="attachment-notes" name="notes" required maxlength="4000" rows="3">${escape(previous?.notes || "")}</textarea><p class="help">Attaching an item grants no public access or media permissions.</p><p id="form-error" class="error" role="alert" hidden></p><div class="form-actions"><button type="button" class="button" data-action="close">Cancel</button><button class="button primary">Attach to review queue</button></div></form>`);
}

function searchView() {
  return `<section class="toolbar"><div><h2>Let the archive come to you.</h2><p>Each search feeds this collection’s review queue. Overlapping results appear once.</p></div><button class="button primary" data-action="new-search">+ Add search</button></section>${state.searches.length ? state.searches.map((s) => `<article class="search-card"><div class="search-head"><div><h3>${escape(s.name)}</h3><span class="state ${escape(s.active_state || "")}">${escape(s.active_state || (s.enabled ? "Scheduled" : "Paused"))}</span></div><div class="search-actions"><button class="button primary small" data-action="run" data-id="${s.id}" ${s.active_state ? "disabled" : ""}>${s.cursor ? "Continue scan" : "Find matches"}</button><button class="button small" data-action="edit-search" data-id="${s.id}" ${s.active_state ? "disabled" : ""}>Edit</button><button class="button small" data-action="pause-search" data-id="${s.id}" ${s.active_state ? "disabled" : ""}>${s.enabled ? "Pause" : "Resume"}</button></div></div><code>${escape(s.effective_query)}</code>${s.cursor && !s.active_state ? `<button class="button small" data-action="restart-scan" data-id="${s.id}">Restart scan from beginning</button><p class="help">Use this if Archive no longer accepts the saved cursor. Review decisions stay saved.</p>` : ""}<div class="search-meta"><span>Every ${s.interval_hours} ${s.interval_hours === 1 ? "hour" : "hours"}</span><span>Last run: ${date(s.last_run)}</span><span>${s.enabled ? `Next: ${date(s.next_run)}` : "Automatic discovery paused"}</span>${s.cursor ? "<span>Scan in progress · cursor saved</span>" : ""}</div>${s.last_error ? `<p class="error">${escape(s.last_error)}</p>` : ""}</article>`).join("") : '<div class="empty"><h2>Follow a topic, not a tab.</h2><p>Add a search query and filters. Arkhiv curation will check it periodically while the app is running.</p><button class="button primary" data-action="new-search">+ Add your first search</button></div>'}<p class="footer-note">Each run scans up to 300 results, saving progress after every page. Broad searches continue on their next run; completed scans start a fresh pass. Scheduling runs while Arkhiv curation is running.</p>`;
}

function runView() {
  return `<section class="toolbar"><div><h2>A record of every search.</h2><p>Recent discovery runs, new finds, and anything that needs attention.</p></div></section>${state.runs.length ? state.runs.map((r) => `<article class="run"><div><strong>${escape(r.search_name)}</strong><time>${date(r.started_at || r.created_at)}</time></div><div><span class="state ${escape(r.state)}">${escape(r.state)}</span></div><div>${r.added} new · ${r.scanned} scanned</div><div>${r.state === "completed" ? (r.continued ? "More results to scan" : "Scan complete") : r.state === "failed" ? "Progress saved" : "Discovery in progress"}</div>${r.error ? `<p class="error">${escape(r.error)}</p>` : ""}</article>`).join("") : '<div class="empty"><h2>No discovery runs yet.</h2><p>Your run history will appear here after a saved search checks Archive for matches.</p><button class="button" data-action="show-searches">Open saved searches</button></div>'}`;
}

function collectionForm(edit = false) {
  const c = edit ? current() : {};
  modal(
    edit ? "Edit collection" : "Start a collection",
    `<p>A topic, a theme, a thread worth following. Give it a home in your private workspace.</p><form id="collection-form" data-edit="${edit}"><label for="collection-name">Collection name</label><input id="collection-name" name="name" required maxlength="120" placeholder="e.g. Independent print culture" value="${escape(c.name || "")}"><label for="collection-description">Description</label><textarea id="collection-description" name="description" rows="3" maxlength="4000" placeholder="What belongs in this collection?">${escape(c.description || "")}</textarea><p id="form-error" class="error" role="alert" hidden></p><div class="form-actions"><button type="button" class="button" data-action="close">Cancel</button><button class="button primary">${edit ? "Save changes" : "Create collection"}</button></div></form>`,
  );
}

function searchForm(id) {
  const s = state.searches.find((s) => s.id === Number(id)) || {
    name: "",
    query: "",
    interval_hours: 24,
    enabled: true,
  };
  modal(
    s.id ? "Edit saved search" : "Follow a new search",
    `<p>Use Archive’s search syntax, then narrow the results. New matches arrive for review, never automatic acceptance.</p><form id="search-form" data-id="${s.id || ""}"><label for="search-name">Search name</label><input id="search-name" name="name" required maxlength="120" value="${escape(s.name)}" placeholder="e.g. Little magazines"><label for="search-query">Archive search query</label><textarea id="search-query" name="query" required maxlength="2000" rows="2" placeholder='e.g. subject:"little magazines"'>${escape(s.query)}</textarea><p class="help">Use words, phrases, or fields such as title:, subject:, and creator:. <a href="https://archive.org/advancedsearch.php" target="_blank" rel="noopener noreferrer">Archive search help ↗</a></p><div class="form-row"><div><label for="media-type">Media type</label><select id="media-type" name="media_type">${[
      ["", "All media"],
      ["texts", "Texts"],
      ["movies", "Video"],
      ["audio", "Audio"],
      ["image", "Images"],
      ["software", "Software"],
      ["data", "Data"],
      ["etree", "Live music"],
    ]
      .map(
        ([v, l]) =>
          `<option value="${v}" ${s.media_type === v ? "selected" : ""}>${l}</option>`,
      )
      .join(
        "",
      )}</select></div><div><label for="source-collection">Archive source collection</label><input id="source-collection" name="source_collection" maxlength="200" value="${escape(s.source_collection || "")}" placeholder="Optional identifier"></div></div><div class="form-row"><div><label for="year-from">From year</label><input id="year-from" name="year_from" type="number" min="1" max="9999" value="${s.year_from || ""}" placeholder="Any year"></div><div><label for="year-to">To year</label><input id="year-to" name="year_to" type="number" min="1" max="9999" value="${s.year_to || ""}" placeholder="Any year"></div></div><label for="interval">Check every</label><select id="interval" name="interval_hours">${[1, 6, 12, 24, 48, 168, 720].map((h) => `<option value="${h}" ${s.interval_hours === h ? "selected" : ""}>${h === 24 ? "Day" : h === 168 ? "Week" : `${h} hours`}</option>`).join("")}</select><label class="check"><input name="enabled" type="checkbox" ${s.enabled ? "checked" : ""}> Enable scheduled discovery</label><p class="help">${s.id ? "Changing the query or filters restarts the scan. Your review decisions stay saved." : "Enabled searches run after saving, then follow this schedule."}</p><p id="form-error" class="error" role="alert" hidden></p><div class="form-actions"><button type="button" class="button" data-action="close">Cancel</button><button class="button primary">Save search</button></div></form>`,
  );
}

function detail(id) {
  const item = state.items.find((i) => i.identifier === id);
  if (!item) return;
  modal(
    "A closer look",
    `<span class="eyebrow">${escape(item.media_type)} · ${escape(item.year || "Year not listed")}</span><h2>${escape(item.title)}</h2><p class="detail-meta">${escape(item.creator || "Creator not listed")}<br>Subjects: ${escape(item.subjects || "Not listed")}<br>Source collections: ${escape(item.source_collections || "Not listed")}</p><p class="detail-description">${escape(item.description || "No description provided.")}</p><a class="button small" href="https://archive.org/details/${encodeURIComponent(id)}" target="_blank" rel="noopener noreferrer">Open original item ↗</a><form id="note-form" data-id="${escape(id)}" data-status="${item.status}"><label for="item-notes">Your curation note</label><textarea id="item-notes" name="notes" maxlength="4000" rows="3" placeholder="Why this belongs, what to revisit…">${escape(item.notes)}</textarea><p id="form-error" class="error" role="alert" hidden></p><div class="form-actions"><button type="button" class="button" data-action="close">Close</button><button class="button primary">Save note</button></div></form>`,
  );
}

function exportDialog() {
  const c = current();
  modal(
    "Export your selection",
    `<p>${c.counts.accepted} accepted items in <strong>${escape(c.name)}</strong>. Pending and passed items are excluded.</p><div class="exports"><a class="button" href="/api/collections/${c.id}/manifest" download>↓ Core migration manifest <span>— retained source evidence; separate access approval</span></a><a class="button" href="/api/collections/${c.id}/export?format=json" download>↓ JSON manifest <span>— metadata and curation notes</span></a><a class="button" href="/api/collections/${c.id}/export?format=csv" download>↓ CSV list <span>— spreadsheet-ready selection</span></a><a class="button" href="/api/collections/${c.id}/export?format=inquiry" download>↓ Archive destination inquiry</a></div><p>These are local selections. Creating a formal Archive collection requires Archive’s help. Its Simple Lists API can associate existing items once a collection and list permissions are provisioned.</p><p class="help">The inquiry is a draft for you to review and send. Arkhiv curation does not send messages or change Archive items.</p>`,
  );
}

document.addEventListener("click", async (event) => {
  const button = event.target.closest("[data-action]");
  if (!button || button.disabled) return;
  const { action, id, status, tab } = button.dataset;
  try {
    if (action === "close") return closeModal();
    if (action === "new-collection") return collectionForm();
    if (action === "edit-collection") return collectionForm(true);
    if (action === "new-search") return searchForm();
    if (action === "edit-search") return searchForm(id);
    if (action === "detail") return detail(id);
    if (action === "attach-core") return attachmentForm(id);
    if (action === "export") return exportDialog();
    if (action === "select") {
      state.collection = Number(id);
      state.page = 1;
      state.filter = "";
    }
    if (action === "tab" || action === "show-searches") {
      state.tab = tab || "searches";
      state.page = 1;
      state.filter = "";
    }
    if (action === "next") state.page++;
    if (action === "previous") state.page--;
    if (action === "clear-filter") {
      state.filter = "";
      state.page = 1;
    }
    if (action === "decide") {
      button.disabled = true;
      await api(
        `/collections/${state.collection}/items/${encodeURIComponent(id)}`,
        { method: "PATCH", body: JSON.stringify({ status }) },
      );
      notify(
        status === "accepted"
          ? "Added to your collection."
          : status === "rejected"
            ? "Passed. This decision will be remembered."
            : "Returned to the review queue.",
      );
    }
    if (action === "run") {
      button.disabled = true;
      await api(`/searches/${id}/run`, { method: "POST", body: "{}" });
      notify("Discovery queued. New matches will appear in the review queue.");
    }
    if (action === "restart-scan") {
      button.disabled = true;
      await api(`/searches/${id}/restart`, { method: "POST", body: "{}" });
      notify("Scan restarted. Your review decisions stay saved.");
    }
    if (action === "pause-search") {
      const search = state.searches.find((s) => s.id === Number(id));
      button.disabled = true;
      await api(`/searches/${id}`, {
        method: "PUT",
        body: JSON.stringify({ ...search, enabled: !search.enabled }),
      });
      notify(
        search.enabled
          ? "Scheduled discovery paused."
          : "Scheduled discovery resumed.",
      );
    }
    await refresh();
  } catch (error) {
    button.disabled = false;
    notify(error.message, true);
  }
});

$("#new-collection").addEventListener("click", () => collectionForm());
document.addEventListener("submit", async (event) => {
  const form = event.target;
  if (
    !["collection-form", "search-form", "note-form", "filter-form", "core-binding-form", "core-attachment-form"].includes(
      form.id,
    )
  )
    return;
  event.preventDefault();
  const data = Object.fromEntries(new FormData(form));
  if (form.id === "filter-form") {
    state.filter = data.filter;
    if (state.tab === "catalog") state.media = data.media || "";
    state.page = 1;
    return refresh();
  }
  const submit = form.querySelector('button:not([type="button"])');
  submit.disabled = true;
  const errorBox = form.querySelector("#form-error");
  errorBox.hidden = true;
  try {
    let addSearch = false;
    if (form.id === "core-binding-form") {
      await api(`/collections/${state.collection}/core`, { method: "PUT", body: JSON.stringify(data) });
      notify("Manifest destination saved.");
    } else if (form.id === "core-attachment-form") {
      await api(`/collections/${state.collection}/core/attachments`, { method: "POST", body: JSON.stringify({item_id: form.dataset.id, representations: new FormData(form).getAll("representations"), notes: data.notes}) });
      state.tab = "pending";
      state.page = 1;
      state.filter = "";
      notify("Attached. Review each source item in your queue before exporting.");
    } else if (form.id === "collection-form") {
      const edit = form.dataset.edit === "true";
      const c = await api(
        edit ? `/collections/${state.collection}` : "/collections",
        { method: edit ? "PUT" : "POST", body: JSON.stringify(data) },
      );
      state.collection = c.id;
      if (!edit) {
        state.tab = "searches";
        state.page = 1;
        state.filter = "";
        addSearch = true;
      }
      notify(
        edit
          ? "Collection updated."
          : "Collection created. Add a search to start finding items.",
      );
    } else if (form.id === "search-form") {
      const id = form.dataset.id;
      for (const key of ["year_from", "year_to"])
        data[key] = data[key] ? Number(data[key]) : null;
      data.interval_hours = Number(data.interval_hours);
      data.enabled = form.elements.enabled.checked;
      await api(
        id ? `/searches/${id}` : `/collections/${state.collection}/searches`,
        { method: id ? "PUT" : "POST", body: JSON.stringify(data) },
      );
      state.tab = "searches";
      notify("Search saved.");
    } else if (form.id === "note-form") {
      await api(
        `/collections/${state.collection}/items/${encodeURIComponent(form.dataset.id)}`,
        {
          method: "PATCH",
          body: JSON.stringify({
            status: form.dataset.status,
            notes: data.notes,
          }),
        },
      );
      notify("Curation note saved.");
    }
    closeModal();
    await refresh();
    if (addSearch) searchForm();
  } catch (error) {
    errorBox.textContent = error.message;
    errorBox.hidden = false;
    submit.disabled = false;
  }
});

async function poll() {
  // Keep dialogs, filters, and keyboard navigation stable while the user is working.
  const focused = document.activeElement;
  if (
    !document.hidden &&
    !$("#modal").open &&
    (!focused || focused === document.body || focused.id === "main")
  )
    await refresh(true);
  refreshTimer = setTimeout(poll, 6000);
}
window.addEventListener("pagehide", () => clearTimeout(refreshTimer));
await refresh();
refreshTimer = setTimeout(poll, 6000);
