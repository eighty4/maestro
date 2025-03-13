#!/bin/bash
set -e

if [ -z "$CHANGELOG" ]; then
    echo "CHANGELOG env var required"
    exit 1
fi

if [ -z "$TAG" ]; then
  echo "TAG env var required"
  exit 1
fi

if [ -z "$VERSION" ]; then
echo "TAG env var required"
exit 1
fi

if [ -z "$PRERELEASE" ]; then
  echo "PRERELEASE env var required"
  exit 1
fi

RELEASE_NOTES=$(./.github/workflows/changelog_get.sh cli/CHANGELOG.md "$CHANGELOG")
RELEASE_NOTES=$(echo -e "[Published on crates.io](https://crates.io/crates/maestro/$VERSION)\n\n### Release notes\n\n$RELEASE_NOTES")

_assets=$(ls "$1")

if [ "$PRERELEASE" == 'true' ]; then
    _release_type="--prerelease"
else
    _release_type="--latest"
fi

(cd "$1" && echo -e "$RELEASE_NOTES" | gh release create "$TAG" \
    --repo "eighty4/maestro" \
    --notes-file - \
    --title "Maestro v$VERSION" \
    $_release_type \
    $_assets)
