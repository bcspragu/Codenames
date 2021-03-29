#/bin/bash

if ! systemctl is-active --user --quiet docker.service; then
  echo "Docker isn't running"
  exit 1
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

nsenter \
  -U --preserve-credentials -n -m --wd="$DIR" \
  -t "$(cat $XDG_RUNTIME_DIR/docker.pid)" \
  go run github.com/bcspragu/Codenames/cmd/codenames-server
