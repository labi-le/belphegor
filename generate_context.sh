#!/usr/bin/env bash

set -euo pipefail

SOURCE_DIRS=()
USER_EXCLUDES=()
SUMMARY_MODE=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -e)
            if [[ -n "${2-}" ]]; then
                USER_EXCLUDES+=("$2")
                shift 2
            else
                echo "Error: -e requires an argument" >&2
                exit 1
            fi
            ;;
        -s)
            SUMMARY_MODE=true
            shift
            ;;
        *)
            SOURCE_DIRS+=("$1")
            shift
            ;;
    esac
done

if [[ ${#SOURCE_DIRS[@]} -eq 0 ]]; then
    echo "Error: no source directories specified" >&2
    exit 1
fi

get_syntax() {
    local filename="$1"
    case "$filename" in
        *.go) echo "go" ;;
        *.txt) echo "txt" ;;
        *.yaml|*.yml) echo "yaml" ;;
        *.proto) echo "protobuf" ;;
        *.nix) echo "nix" ;;
        *Makefile) echo "makefile" ;;
        *.php) echo "php" ;;
        *.xml) echo "xml" ;;
        Dockerfile) echo "dockerfile" ;;
        *.ps1) echo "powershell" ;;
        *.c|*.h) echo "c" ;;
        *.rs) echo "rs" ;;
        *) echo "text" ;;
    esac
}

print_tree() {
    local ignore_list=".git|node_modules|vendor"

    for excl in "${USER_EXCLUDES[@]}"; do
        local clean_excl="${excl%/}"
        ignore_list="${ignore_list}|${clean_excl##*/}"
    done

    echo "# Project Tree"
    echo '```text'
    if command -v tree &> /dev/null; then
        tree "${SOURCE_DIRS[@]}" -I "$ignore_list"
    else
        find "${SOURCE_DIRS[@]}" -maxdepth 3 -not -path '*/.*'
    fi
    echo '```'
    echo ""
}

print_files() {
    local find_cmd=(find "${SOURCE_DIRS[@]}")

    local is_first=true
    local base_excludes=(".git" "node_modules" "vendor" ".idea")

    find_cmd+=( \( )

    for excl in "${base_excludes[@]}"; do
        if [ "$is_first" = true ]; then is_first=false; else find_cmd+=( -o ); fi
        find_cmd+=( -name "$excl" )
    done

    for excl in "${USER_EXCLUDES[@]}"; do
        if [ "$is_first" = true ]; then is_first=false; else find_cmd+=( -o ); fi
        local clean_excl="${excl%/}"
        if [[ "$clean_excl" == *"/"* ]]; then
             find_cmd+=( -path "$clean_excl" )
        else
             find_cmd+=( -name "$clean_excl" )
        fi
    done

    find_cmd+=( \) -prune -o )

    find_cmd+=( -type f \( \
        -name "*.go" -o \
        -name "*.txt" -o \
        -name "*.yml" -o \
        -name "*.php" -o \
        -name "*.c" -o \
        -name "*.h" -o \
        -name "*.xml" -o \
        -name "*.ps1" -o \
        -name "*.rs" -o \
        -name "*.yaml" -o \
        -name "*.proto" -o \
        -name "*.nix" -o \
        -name "Dockerfile" -o \
        -name "Makefile" \
    \) -print )

    "${find_cmd[@]}" | sort | while read -r file; do
        local lang
        lang=$(get_syntax "$file")

        echo "### File: $file"
        echo "\`\`\`$lang"

        if [ "$SUMMARY_MODE" = true ] && [[ "$lang" == "go" ]]; then
             awk '
                BEGIN { depth = 0; hiding = 0 }
                {
                    line = $0
                    gsub(/\/\/.*$/, "", line)

                    n_open = gsub(/{/, "{", line)
                    n_close = gsub(/}/, "}", line)

                    is_func_start = ($0 ~ /^[[:space:]]*func/ && n_open > 0)

                    if (depth == 0 && is_func_start) {
                        hiding = 1
                        out = $0
                        sub(/[[:space:]]*{/, "", out)
                        print out
                    }
                    else if (hiding == 0) {
                        print $0
                    }
                    else if (hiding == 1) {
                        if ((depth + n_open - n_close) == 0) {
                            hiding = 0
                        }
                    }

                    depth = depth + n_open - n_close
                }
            ' "$file"

        else
            cat "$file"
        fi

        echo ""
        echo "\`\`\`"
        echo ""
    done
}

main() {
    print_tree
    print_files
}

main
