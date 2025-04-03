#!/bin/bash
set -e

# Check if version tag is provided
if [ $# -lt 1 ]; then
    echo "Usage: $0 <version_tag>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

VERSION=$1
echo "Preparing release for version $VERSION"

# Ensure we're on the main branch and everything is up to date
git checkout main
git pull origin main

# Ensure we have a clean workspace
if [[ -n $(git status --porcelain) ]]; then
    echo "Error: Working directory is not clean. Please commit or stash changes."
    exit 1
fi

# Build the application
echo "Building Lil version $VERSION..."
go build -ldflags "-s -w -X main.version=$VERSION -X main.buildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" -o lil .

# Create a release tag
echo "Creating release tag $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

# Create release tarball for homebrew
echo "Creating release tarball..."
mkdir -p dist
TARBALL="dist/lil-${VERSION#v}.tar.gz"
git archive --format=tar.gz --prefix="lil-${VERSION#v}/" -o "$TARBALL" "$VERSION"

# Calculate the SHA256 checksum
SHA256=$(shasum -a 256 "$TARBALL" | cut -d ' ' -f 1)
echo "SHA256 checksum: $SHA256"

# Update the homebrew formula (local update only, GitHub Actions will handle the rest)
echo "Updating local Homebrew formula..."
sed -i '' "s/url \".*\"/url \"https:\/\/github.com\/pzurek\/lil\/archive\/refs\/tags\/$VERSION.tar.gz\"/" formula/lil.rb
sed -i '' "s/sha256 \".*\"/sha256 \"$SHA256\"/" formula/lil.rb

echo "Homebrew formula updated locally."
echo ""
echo "Next steps:"
echo "1. Create a GitHub release:"
echo "   - Go to: https://github.com/pzurek/lil/releases/new?tag=$VERSION"
echo "   - Upload the built binary as an asset to the release"
echo "   - Publish the release"
echo ""
echo "2. GitHub Actions will automatically:"
echo "   - Update the Homebrew formula in the main repository"
echo "   - Update the Homebrew tap repository (if it exists)"
echo ""
echo "3. If this is your first release, you'll need to create a Homebrew tap repository:"
echo "   - Create a new GitHub repository named 'homebrew-lil'"
echo "   - Create a Personal Access Token with 'repo' scope"
echo "   - Add it to your GitHub repository secrets as GH_PAT"
echo ""
echo "4. Users can install Lil with:"
echo "   brew tap pzurek/lil"
echo "   brew install lil" 