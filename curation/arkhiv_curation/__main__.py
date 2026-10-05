import argparse

import uvicorn

from .app import create_app

parser = argparse.ArgumentParser(description="Arkhiv private curation workbench")
parser.add_argument("--host", default="127.0.0.1")
parser.add_argument("--port", type=int, default=18764)
parser.add_argument(
    "--no-discovery", action="store_true", help="Disable discovery in an isolated preview"
)
args = parser.parse_args()
uvicorn.run(create_app(start_worker=not args.no_discovery), host=args.host, port=args.port)
