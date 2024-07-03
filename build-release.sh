#!/bin/bash
set -eo pipefail

if [ $# -ne 1 ]
  then
    echo "Usage: ./build-release.sh <version>"
    echo ""
    echo "Example: ./build-release.sh v1.0.4"
    echo ""
    exit 0
fi

version=$1

mkdir build/$version

TARGETS="linux/arm64 linux/arm linux/amd64 darwin/arm64 darwin/amd64 windows/amd64 windows/arm64"

for target in $TARGETS
do
    platform=$(echo $target | cut -d/ -f1)
    arch=$(echo $target | cut -d/ -f2)

    if [ $platform = "windows" ]
      then
        extension=".exe"
      else
        extension=""
    fi

    echo Building for $platform/$arch...
    GOOS=$platform GOARCH=$arch go build -o build/$version/hashy-$platform-$arch$extension -ldflags="-X main.version=$version" ./
done

echo Tagging $version...
git tag -a $version -m "Release $version"

echo Pushing tag...
git push origin $version
git push
