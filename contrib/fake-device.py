#!/usr/bin/env python3

import argparse
import os
import socket
import sys

import requests


def main():
    parser = argparse.ArgumentParser(
        description="Fake device for interacting with satellite server."
    )
    parser.add_argument(
        "--port",
        type=int,
        help="Port of device-gateway. default=%(default)d",
        default=8443,
    )
    parser.add_argument(
        "--http-method",
        type=str,
        help="default=%(default)s",
        default="GET",
    )
    parser.add_argument(
        "-d",
        "--device-dir",
        type=str,
        required=True,
    )
    parser.add_argument(
        "resource",
        type=str,
    )

    args = parser.parse_args()

    tls_args = {
        "cert": (
            os.path.join(args.device_dir, "client.pem"),
            os.path.join(args.device_dir, "pkey.pem"),
        ),
        "verify": os.path.join(args.device_dir, "root.crt"),
    }
    with open(os.path.join(args.device_dir, "dghostname"), "r") as f:
        hostname = f.read().strip()

    if args.resource[0] == "/":
        args.resource = args.resource[1:]
    url = f"https://{hostname}:{args.port}/{args.resource}"

    try:
        if args.http_method == "GET":
            r = requests.get(url, **tls_args)
        else:
            sys.exit(f"Unsupported HTTP method: {args.http_method}")
    except requests.exceptions.RequestException as e:
        sys.exit(f"Error connecting to {url}: {e}")

    print(f"< HTTP {r.status_code}", file=sys.stderr)
    print(r.text)

if __name__ == "__main__":
    main()