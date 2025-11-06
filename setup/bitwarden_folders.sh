#!/usr/bin/env bash

# Debug levels:
# - BWENV_DEBUG=0 or unset: No debug output
# - BWENV_DEBUG=1: Show steps only (no secrets)
# - BWENV_DEBUG=2: Show steps and secrets (full debug)
# Backward compatibility: DEBUG_BW=true equals BWENV_DEBUG=2

_bwenv_log() {
  local level="$1"
  shift
  local debug_level="${BWENV_DEBUG:-0}"
  
  # Backward compatibility
  if [ "${DEBUG_BW:-false}" = "true" ]; then
    debug_level=2
  fi
  
  if [ "$debug_level" -ge "$level" ]; then
    echo "$@" >&2
  fi
}

_bwenv_log_step() {
  _bwenv_log 1 "$@"
}

_bwenv_log_secret() {
  _bwenv_log 2 "$@"
}

# --- Helper functions for logging ---
_bwenv_log_divider() {
  local level="$1"
  local width="${2:-50}"
  _bwenv_log "$level" "$(printf '─%.0s' $(seq 1 $width))"
}

_bwenv_log_header() {
  local level="$1"
  shift
  _bwenv_log_divider "$level" 50
  _bwenv_log "$level" "  $*"
  _bwenv_log_divider "$level" 50
}

_bwenv_log_section() {
  local level="$1"
  shift
  _bwenv_log "$level" ""
  _bwenv_log "$level" "┌─ $*"
}

_bwenv_log_subsection() {
  local level="$1"
  shift  
  _bwenv_log "$level" "│"
  _bwenv_log "$level" "├─ $*"
}

_bwenv_log_item() {
  local level="$1"
  shift
  _bwenv_log "$level" "│  $*"
}

_bwenv_log_end_section() {
  local level="$1"
  shift
  _bwenv_log "$level" "└─ $*"
  _bwenv_log "$level" ""
}

# --- Activate Bitwarden CLI integration ---
use_bitwarden_folders() {
  _bwenv_log_header 1 "🔧 Bitwarden Integration Setup"
  
  _bwenv_log_section 1 "Checking Dependencies"
  
  if ! command -v bw >/dev/null 2>&1; then
    echo "❌ Bitwarden CLI is not installed." >&2
    _bwenv_log_item 1 "Install it from: https://bitwarden.com/help/cli/"
    return 1
  fi
  
  _bwenv_log_item 1 "✅ Bitwarden CLI found"

  if ! command -v jq >/dev/null 2>&1; then
    echo "❌ jq is not installed." >&2
    _bwenv_log_item 1 "Install it with: sudo apt install jq (Ubuntu/Debian) or brew install jq (macOS)"
    return 1
  fi
  
  _bwenv_log_item 1 "✅ jq found"
  
  _bwenv_log_subsection 1 "Validating Session"

  if [ -z "$BW_SESSION" ]; then
    echo "⚠️  BW_SESSION is not defined." >&2
    _bwenv_log_item 1 "Run: export BW_SESSION=\$(bw unlock --raw)"
    return 1
  fi
  
  _bwenv_log_item 1 "✅ BW_SESSION is set"
  
  # Test if session is valid
  if ! bw list folders --session "$BW_SESSION" >/dev/null 2>&1; then
    echo "❌ Invalid or expired BW_SESSION." >&2
    _bwenv_log_item 1 "Run: export BW_SESSION=\$(bw unlock --raw)"
    return 1
  fi
  
  _bwenv_log_end_section 1 "✅ Session validated successfully"
}

# --- Load variables from Bitwarden folder ---
load_bitwarden_folder_vars() {
  local folder_name="$1"
  local vars_loaded=0
  local items_processed=0

  if [ -z "$folder_name" ]; then
    echo "❌ Folder name not provided." >&2
    return 1
  fi

  _bwenv_log_header 1 "📦 Loading Environment Variables from Bitwarden"
  _bwenv_log_section 1 "Locating Folder: \"$folder_name\""

  # Find folder ID by name
  local folder_id
  folder_id=$(bw list folders --session "$BW_SESSION" 2>/dev/null | jq -r ".[] | select(.name==\"$folder_name\") | .id")

  if [ -z "$folder_id" ] || [ "$folder_id" = "null" ]; then
    echo "❌ Folder \"$folder_name\" not found in Bitwarden." >&2
    _bwenv_log_item 1 "Available folders:"
    bw list folders --session "$BW_SESSION" 2>/dev/null | jq -r '.[].name' | sed 's/^/│     ◦ /' >&2
    return 1
  fi

  _bwenv_log_end_section 1 "✅ Found folder: \"$folder_name\" (id: ${folder_id:0:8}...)"

  # List all items from folder
  local items_json
  items_json=$(bw list items --folderid "$folder_id" --session "$BW_SESSION" 2>/dev/null)
  
  if [ -z "$items_json" ] || [ "$items_json" = "null" ] || [ "$items_json" = "[]" ]; then
    _bwenv_log_section 1 "No Items Found"
    _bwenv_log_end_section 1 "⚠️  No items found in folder \"$folder_name\""
    return 0
  fi

  _bwenv_log_section 1 "Processing Items from Folder"

  # Iterate through items
  while IFS= read -r item; do
    if [ -z "$item" ] || [ "$item" = "null" ]; then
      continue
    fi
    
    local item_name
    item_name=$(echo "$item" | jq -r '.name // "unnamed"')
    items_processed=$((items_processed + 1))
    
    _bwenv_log_subsection 1 "🔹 Processing item: $item_name"

    # Process fields
    local fields_count=0
    while IFS= read -r field; do
      if [ -z "$field" ] || [ "$field" = "null" ]; then
        continue
      fi
      
      local key val
      key=$(echo "$field" | jq -r '.name // ""')
      val=$(echo "$field" | jq -r '.value // ""')

      if [ -n "$key" ] && [ "$key" != "null" ]; then
        # Show secret value only at debug level 2, otherwise show [HIDDEN]
        local debug_level="${BWENV_DEBUG:-0}"
        if [ "${DEBUG_BW:-false}" = "true" ]; then
          debug_level=2
        fi
        
        if [ "$debug_level" -ge 2 ]; then
          _bwenv_log_secret "│     ↳ Exporting: $key = $val"
        else
          _bwenv_log_step "│     ↳ Exporting: $key = [HIDDEN]"
        fi
        
        export "$key"="$val"
        vars_loaded=$((vars_loaded + 1))
        fields_count=$((fields_count + 1))
      fi
    done < <(echo "$item" | jq -c '.fields[]? // empty')
    
    if [ $fields_count -eq 0 ]; then
      _bwenv_log_item 1 "⚠️  No custom fields found in item: $item_name"
    fi

  done < <(echo "$items_json" | jq -c '.[] // empty')
  
  _bwenv_log_end_section 1 "✅ Completed processing all items"
  
  _bwenv_log_header 1 "📊 Summary Report"
  _bwenv_log_item 1 "📁 Folder: \"$folder_name\""
  _bwenv_log_item 1 "📦 Items processed: $items_processed"
  _bwenv_log_item 1 "🔧 Variables exported: $vars_loaded"
  
  if [ $vars_loaded -eq 0 ]; then
    _bwenv_log_item 1 ""
    _bwenv_log_item 1 "⚠️  No environment variables were exported."
    _bwenv_log_item 1 "   Make sure your Bitwarden items have custom fields with names and values."
  fi
  
  _bwenv_log_divider 1 50
}