#!/usr/bin/env python3
"""
Refresh test fixtures from upstream meteofrance.com.

Auth flow:
  1. GET / → server sets mfsession cookie (ROT13-encoded JWT)
  2. ROT13-decode the cookie value → Bearer token
  3. Use token in Authorization header for all subsequent requests

Files updated:
  racine.html        — root HTML page (contains embedded JSON config)
  pays007.svg        — SVG map for France (PAYS007 / METROPOLE)
  geography.json     — GeoJSON region boundaries
  multiforecast.json — forecast data for all cities
"""

import codecs
import json
import sys
import urllib.parse
import urllib.request
from html.parser import HTMLParser
from pathlib import Path

UPSTREAM = "https://meteofrance.com"
API_UPSTREAM = "https://rpcache-aa.meteofrance.com/internet2018client/2.0"
UA = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:133.0) Gecko/20100101 Firefox/133.0"
OUT_DIR = Path(__file__).parent


def rot13(s: str) -> str:
    return codecs.encode(s, "rot_13")


def fetch(url: str, token: str | None = None) -> tuple[bytes, str | None]:
    """GET url, return (body, new_mfsession_cookie_or_None)."""
    req = urllib.request.Request(url, headers={"User-Agent": UA})
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    with urllib.request.urlopen(req) as resp:
        body = resp.read()
        # extract mfsession from Set-Cookie headers
        new_cookie = None
        for header, value in resp.headers.items():
            if header.lower() == "set-cookie" and "mfsession=" in value:
                cookie_val = value.split("mfsession=", 1)[1].split(";", 1)[0]
                new_cookie = cookie_val
                break
        return body, new_cookie


class DrupalSettingsParser(HTMLParser):
    """Extract the JSON blob from <script data-drupal-selector="drupal-settings-json">."""

    def __init__(self):
        super().__init__()
        self._capture = False
        self.data = ""

    def handle_starttag(self, tag, attrs):
        if tag == "script" and ("data-drupal-selector", "drupal-settings-json") in attrs:
            self._capture = True

    def handle_data(self, data):
        if self._capture:
            self.data += data

    def handle_endtag(self, tag):
        if tag == "script" and self._capture:
            self._capture = False


def parse_drupal_settings(html: bytes) -> dict:
    parser = DrupalSettingsParser()
    parser.feed(html.decode("utf-8", errors="replace"))
    if not parser.data.strip():
        raise ValueError("drupal-settings-json script tag not found in HTML")
    return json.loads(parser.data)


def main():
    print("Fetching root page and extracting auth token...")
    body, mfsession = fetch(f"{UPSTREAM}/")

    if not mfsession:
        print("ERROR: no mfsession cookie in response", file=sys.stderr)
        sys.exit(1)

    token = rot13(mfsession)
    print(f"Token acquired ({len(token)} chars)")

    (OUT_DIR / "racine.html").write_bytes(body)
    print("Saved racine.html")

    settings = parse_drupal_settings(body)

    map_info = settings["mf_map_layers_v2"]
    path_assets = map_info["path_assets"]           # e.g. "METROPOLE"
    id_technique = map_info["field_id_technique"]   # e.g. "PAYS007"
    svg_name = id_technique.lower()                 # e.g. "pays007"

    children_poi = settings["mf_map_layers_v2_children_poi"]
    insee_list = ",".join(p["insee"] for p in children_poi)

    print(f"path_assets={path_assets}  id_technique={id_technique}  svg={svg_name}.svg")
    print(f"INSEE codes: {len(children_poi)} cities")

    print("Fetching SVG map...")
    svg_url = f"{UPSTREAM}/modules/custom/mf_map_layers_v2/maps/desktop/{path_assets}/{svg_name}.svg"
    svg_body, _ = fetch(svg_url, token)
    (OUT_DIR / f"{svg_name}.svg").write_bytes(svg_body)
    print(f"Saved {svg_name}.svg")

    print("Fetching geography GeoJSON...")
    geo_id = id_technique.lower()
    geo_url = f"{UPSTREAM}/modules/custom/mf_map_layers_v2/maps/desktop/{path_assets}/geo_json/{geo_id}-aggrege.json"
    geo_body, _ = fetch(geo_url, token)
    (OUT_DIR / "geography.json").write_bytes(geo_body)
    print("Saved geography.json")

    print("Fetching multiforecast data...")
    params = urllib.parse.urlencode({
        "bbox": "",
        "begin_time": "",
        "end_time": "",
        "time": "",
        "instants": "morning,afternoon,evening,night",
        "liste_id": insee_list,
    })
    forecast_url = f"{API_UPSTREAM}/multiforecast?{params}"
    forecast_body, _ = fetch(forecast_url, token)
    (OUT_DIR / "multiforecast.json").write_bytes(forecast_body)
    print("Saved multiforecast.json")

    print("\nDone. Files updated:")
    for f in ["racine.html", f"{svg_name}.svg", "geography.json", "multiforecast.json"]:
        p = OUT_DIR / f
        print(f"  {p.name:25s}  {p.stat().st_size:>10,} bytes")


if __name__ == "__main__":
    main()
