#!/usr/bin/env bash

# --- Activate Bitwarden CLI integration ---
use_bitwarden_folders() {
  if ! command -v bw >/dev/null 2>&1; then
    echo "❌ Bitwarden CLI is not installed." >&2
    exit 1
  fi

  if [ -z "$BW_SESSION" ]; then
    echo "⚠️  BW_SESSION is not defined. Run: export BW_SESSION=\$(bw unlock --raw)" >&2
    exit 1
  fi
}

# --- Load variables from Bitwarden folder ---
load_bitwarden_folder_vars() {
  local folder_name="$1"
  local debug="${DEBUG_BW:-false}"

  if [ -z "$folder_name" ]; then
    echo "⚠️  Folder name not provided."
    return 1
  fi

  # Find folder ID by name
  local folder_id
  folder_id=$(bw list folders --session "$BW_SESSION" | jq -r ".[] | select(.name==\"$folder_name\") | .id")

  if [ -z "$folder_id" ]; then
    echo "⚠️  Folder \"$folder_name\" not found in Bitwarden."
    return 1
  fi

  [ "$debug" = true ] && echo "📁 Using Bitwarden folder: \"$folder_name\" (id: $folder_id)"

  # List all items from folder
  local items_json
  items_json=$(bw list items --folderid "$folder_id" --session "$BW_SESSION")

  # Iterate through items
  while read -r item; do
    item_name=$(echo "$item" | jq -r '.name')
    [ "$debug" = true ] && echo "🔹 Found item: $item_name"

    while read -r field; do
      key=$(echo "$field" | jq -r '.name')
      val=$(echo "$field" | jq -r '.value')

      [ "$debug" = true ] && echo "   ↳ Field: $key = $val"

      export "$key"="$val"
    done < <(echo "$item" | jq -c '.fields[]?')

  done < <(echo "$items_json" | jq -c '.[]')
}