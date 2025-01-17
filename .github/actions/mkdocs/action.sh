#!/bin/bash

set -e

REQUIREMENTS="$GITHUB_WORKSPACE/docs/requirements.txt"

if [ -f "$REQUIREMENTS" ]; then
    pip install -r "$REQUIREMENTS"
fi

if [ -n "$GITHUB_TOKEN" ]; then
    REMOTE_REPO="https://x-access-token:$GITHUB_TOKEN@github.com/$GITHUB_REPOSITORY.git"
fi

git config --global user.name "$GITHUB_ACTOR"
git config --global user.email "$GITHUB_ACTOR@users.noreply.github.com"

mkdocs build --config-file "$GITHUB_WORKSPACE/mkdocs.yml"

git clone --branch=gh-pages --single-branch --depth=1 "$REMOTE_REPO" gh-pages
cd gh-pages

# remove current content in branch gh-pages
git rm -r .
# copy new doc.
cp -r ../site/* .
# commit changes
git add .
git commit -m "deploy GitHub Pages"
git push --force --quiet "$REMOTE_REPO" gh-pages > /dev/null 2>&1
