#!/usr/bin/env bash
# This script requires bash > 4.
# You can install it on OS X via `brew install bash`.

# Config file path.
CONFIG_FILE="$HOME/.emprc"

# Associative array of configs.
declare -A config

# Keep track of config order.
declare -a order

# List apis.
function list_apis() {
  for i in "${!order[@]}"; do
    if [ "${order[$i]}" = "current" ]; then continue; fi
    if [ "${order[$i]}" = "${config[current]}" ]; then printf "* "; fi
    printf "%s \t %s\n" "${order[$i]}" "${config[${order[$i]}]}"
  done 

  return 0
}

# Add API targets.
function api_add() {
  IFS=" " read -a targets <<< "$@"
  for target in "${targets[@]}"; do
    printf "%s\n" "$target" >> $CONFIG_FILE
  done
  printf "Added api targets\n"
}

# Read config file.
while IFS='=' read -ra ADDR; do
  config["$ADDR"]=${ADDR[1]}
  order+=("$ADDR")
done < $CONFIG_FILE