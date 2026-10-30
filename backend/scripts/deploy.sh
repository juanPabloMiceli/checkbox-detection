#!/usr/bin/env bash
set -euo pipefail

image="${1:?Pass the full ECR image URI}"
registry="${image%%/*}"
cd /home/ec2-user/checkbox-detector
compose=(docker compose -f compose.yaml -f compose.ec2.yaml)

# Authenticate using the EC2 instance role, then extract this release's settings.
aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin "$registry"
container=""
trap 'if [[ -n "$container" ]]; then docker rm "$container" >/dev/null; fi; docker logout "$registry" >/dev/null' EXIT
docker pull "$image"
container="$(docker create "$image")"
docker cp "$container:/deploy/." .
docker rm "$container"
container=""

# Keep the previous release locally for rollback.
previous=""
active="$("${compose[@]}" ps -q api)"
if [[ -n "$active" ]]; then
  previous="$(docker inspect --format '{{.Config.Image}}' "$active")"
fi

sed -i '/^BACKEND_IMAGE=/d' .env
printf 'BACKEND_IMAGE=%s\n' "$image" >> .env
"${compose[@]}" up -d --no-build --no-deps api

# Remove only older backend images; leave other services and volumes alone.
while IFS= read -r old; do
  case "$old" in
    "$registry/checkbox-detection/backend:"*|checkbox-detector:*)
      if [[ "$old" != "$image" && "$old" != "$previous" ]]; then
        docker image rm "$old"
      fi
      ;;
  esac
done < <(docker image ls --format '{{.Repository}}:{{.Tag}}')
