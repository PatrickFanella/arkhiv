# Arkhiv brand files

The masters live in SUBCULT Studio: `subculture-collective/subcult-tv`, `studio/brands/arkhiv/`, revision `159489c`. Read the pack's `GUIDE.md` and `alt-text.md` there before making new artwork. Do not edit these copies; change the Studio pack, rebuild it, and copy the new files here.

## Copied byte for byte

| File | Studio source | SHA-256 |
| --- | --- | --- |
| `mark.svg` | `logos/mark.svg` | `bb09af00162e1d23a59cced52584beb8fa1c84271751742041e684633c5f710c` |
| `lockup.svg` | `logos/lockup.svg` | `2478addf4b10c82329be579c4f83e3711760ecf43be863f2216da008e292afa3` |
| `og.png` | `product/og.png` (1200×630) | `22730e3549cd4df6327d2f6fe271f2dc0dea5161840bed0b5d1704791c47fc9a` |

## Derived from the pack

| File | Made from | SHA-256 |
| --- | --- | --- |
| `favicon.ico` | `mark.svg` rendered with `rsvg-convert` at 16, 32 and 48 px, joined with `magick` | `b04269fce5019e8ef764a5bb56d6e26ac0f2b63bbef2a6ea3ae1b513f9c9bbe9` |
| `apple-touch-icon.png` | `mark.svg` rendered at 180×180 on the pack background `#f3efe4` | `efd11464f200486a091cfc59ba206ef58bba43cea7ca6af2bd82cbad79bf66ed` |
| `readme-lockup.png` | `lockup.svg` rendered at 704×256 and centred on `#f3efe4` at 832×384, so the dark ink stays readable on dark README themes | `b25d0c05410ec15522324e755abab598f3ec15568f498fc46f2da170865c4b0c` |

## Served copies

The public interface serves identical copies from `web/public/`: `favicon.svg` (same bytes as `mark.svg`), `favicon.ico`, `apple-touch-icon.png` and `og.png`. `web/public/site.json` names `/og.png` as the link preview and carries its alt text. Replace both copies together.
