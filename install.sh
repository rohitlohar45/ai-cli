#!/bin/bash

# Constants
REPO="rohitlohar45/ai-cli"
APP_NAME="ai-cli"

OS=$(uname -s)
ARCH=$(uname -m)

case $OS in
  Linux*)   OS=linux;;
  Darwin*)  OS=darwin;;
  CYGWIN*|MINGW*|MSYS*|MINGW32*|MINGW64*) OS=windows;;
  *)        echo "Unsupported OS: $OS"; exit 1;;
esac

case $ARCH in
  x86_64) ARCH=amd64;;
  *)      echo "Unsupported architecture: $ARCH"; exit 1;;
esac

DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/$APP_NAME-$OS-$ARCH"

if [ "$OS" == "windows" ]; then
  DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
  TARGET="$APP_NAME.exe"
else
  TARGET="$APP_NAME"
fi

echo "Downloading $APP_NAME for $OS/$ARCH from $DOWNLOAD_URL..."
curl -L -o $TARGET $DOWNLOAD_URL

if [ $? -ne 0 ]; then
  echo "Failed to download $APP_NAME."
  exit 1
fi

if [ "$OS" != "windows" ]; then
  chmod +x $TARGET
  sudo mv $TARGET /usr/local/bin/$APP_NAME
  echo "$APP_NAME installed successfully at /usr/local/bin/$APP_NAME"
else
  echo "$APP_NAME.exe downloaded. Please move it to a location in your PATH."
fi
