#!/bin/sh

WORK_DIR=/tmp
BIN_DIR=/usr/local/bin

arch() {
  ARCH=$(uname -m)
  case $ARCH in
    x86_64) ARCH=amd64;;
  esac
}

os() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
}

download() {
  LATEST_RELEASE_JSON="https://api.github.com/repos/malfunkt/hyperfox/releases/latest"
  DOWNLOAD_URL=$(curl --silent -L $LATEST_RELEASE_JSON | grep browser_download_url | sed s/'^.*: "'//g | sed s/'"$'//g | grep "$OS.*$ARCH")

  echo "Downloading ${DOWNLOAD_URL}..."
  if [ -z "$DOWNLOAD_URL" ]; then
    curl --silent -L $LATEST_RELEASE_JSON;
    echo "Github API is not working right now. Please try again later.";
    exit 1
  fi;

  BASENAME=$(basename $DOWNLOAD_URL)

  wget $DOWNLOAD_URL -O $WORK_DIR/$BASENAME

  case $BASENAME in
    *.bz2)
      bzip2 -d $WORK_DIR/$BASENAME
      FILENAME=$WORK_DIR/hyperfox_${OS}_${ARCH}
      ;;
    *.gz)
      gzip -dfv $WORK_DIR/$BASENAME
      FILENAME=$WORK_DIR/hyperfox_${OS}_${ARCH}
      ;;
    *.zip)
      unzip -d $WORK_DIR -o $WORK_DIR/$BASENAME
      FILENAME=$WORK_DIR/hyperfox_${OS}_${ARCH}.exe
      ;;
    *)
      echo "Don't know how to handle downloaded file $BASENAME" && exit 1
      ;;
  esac

  if [ -z "$FILENAME" ]; then
    echo "Could not install." && exit 1
  fi;

  echo "Installing to $BIN_DIR... (it might require sudo password)"
  sudo install -v -c -m 0755 $FILENAME $BIN_DIR/hyperfox || echo "This script needs root privileges in order to install into $BIN_DIR."
  rm $FILENAME
}

arch
os
download
