#!/bin/sh
set -eu
: "${API_UPSTREAM:?Set API_UPSTREAM to the backend origin, e.g. http://backend:8080}"
case "$API_UPSTREAM" in
  http://*|https://*) ;;
  *) echo 'API_UPSTREAM must start with http:// or https://' >&2; exit 1 ;;
esac
origin="${API_UPSTREAM#*://}"
case "$origin" in
  ''|*@*|*/*|*\?*|*\#*|*\;*|*\ *|*\{*|*\}*) echo 'API_UPSTREAM must be an origin without a trailing slash or path' >&2; exit 1 ;;
esac
