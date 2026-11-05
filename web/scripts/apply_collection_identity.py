"""Apply a named collection identity after building a retained web frontend.

The default Arkhiv identity and engine/profile data are unchanged. This command
touches HTML identity tags and the separate studio credit, never collection data.
"""
import argparse
import json
from pathlib import Path
from html.parser import HTMLParser


class Tags(HTMLParser):
    def __init__(self, text):
        super().__init__()
        self.tags = []
        self.feed(text.split('</head>')[0])

    def handle_starttag(self, tag, attrs):
        if tag in ('meta', 'link'):
            self.tags.append((tag, dict(attrs), self.get_starttag_text()))


def apply(root, identity):
    data = json.loads(identity.read_text())
    assert data['slug'] == 'anarchist-archive', 'Select the named Anarchist collection export'
    keys = {attrs.get('property') or attrs.get('name') for tag, attrs, _ in
            Tags('<head>' + ''.join(data['tags']) + '</head>').tags if tag == 'meta'}
    path = root / 'index.html'
    text = path.read_text()
    for tag, attrs, raw in Tags(text).tags:
        if tag == 'meta' and (attrs.get('property') or attrs.get('name')) in keys:
            text = text.replace(raw, '')
        elif tag == 'link' and attrs.get('rel') in ('icon', 'apple-touch-icon', 'canonical'):
            text = text.replace(raw, '')
    assert 'class="subcult-identity"' not in text, 'Identity already applied; rebuild the source first'
    text = text.replace('</head>', '\n'.join(data['tags']) +
                        '<link rel="stylesheet" href="/brand/anarchist-archive/credit.css"></head>')
    text = text.replace('</body>', data['credit'] + '</body>')
    path.write_text(text)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--site-root', type=Path, required=True)
    parser.add_argument('--identity', type=Path, required=True)
    args = parser.parse_args()
    apply(args.site_root, args.identity)
