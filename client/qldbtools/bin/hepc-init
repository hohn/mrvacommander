#!/bin/bash

#* Utility functions
log() {
    local level="$1"
    shift
    local color_reset="\033[0m"
    local color_info="\033[1;34m"
    local color_warn="\033[1;33m"
    local color_error="\033[1;31m"

    local color
    case "$level" in
        INFO) color="$color_info" ;;
        WARN) color="$color_warn" ;;
        ERROR) color="$color_error" ;;
        *) color="$color_reset" ;;
    esac

    echo -e "${color}[$(date +"%Y-%m-%d %H:%M:%S")] [$level] $*${color_reset}" >&2
}
usage() {
    echo "Usage: $0 --db_collection_dir <directory> --starting_path <path> [-h]"
    echo
    echo "Options:"
    echo "  --db_collection_dir <directory>  Specify the database collection directory."
    echo "  --starting_path <path>           Specify the starting path."
    echo "  -h                               Show this help message."
    exit 1
}


#* Initialize and parse arguments
set -euo pipefail               # exit on error, unset var, pipefail
trap 'rm -fR /tmp/hepc.$$-*' EXIT

starting_dir=$(pwd)
db_collection_dir=""
starting_path=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --db_collection_dir)
            shift
            if [[ -z "$1" || "$1" == -* ]]; then
                echo "Error: --db_collection_dir requires a directory as an argument."
                usage
            fi
            db_collection_dir="$1"
            ;;
        --starting_path)
            shift
            if [[ -z "$1" || "$1" == -* ]]; then
                echo "Error: --starting_path requires a path as an argument."
                usage
            fi
            starting_path="$1"
            ;;
        -h)
            usage
            ;;
        *)
            echo "Error: Unknown option '$1'."
            usage
            ;;
    esac
    shift
done

# Check if required arguments were provided
if [[ -z "$db_collection_dir" ]]; then
    echo "Error: --db_collection_dir is required."
    usage
fi

if [[ -z "$starting_path" ]]; then
    echo "Error: --starting_path is required."
    usage
fi

#* Find all DBs
log INFO "searching for db.zip files"
find ${starting_path} -type f -name "db.zip" -size +0c > /tmp/hepc.$$-paths

#* Collect detailed information from the database files
# Don't assume they are unique.
log INFO "collecting information from db.zip files"
mkdir -p $db_collection_dir
cat /tmp/hepc.$$-paths | while read -r zip_path
do
    log INFO "Extracting from ${zip_path}"
    zip_dir=$(dirname ${zip_path})
    zip_file=$(basename ${zip_path})
    unzip -o -q ${zip_path} '*codeql-database.yml' -d /tmp/hepc.$$-zip 
    # The content may be LANGUAGE/codeql-database.yml

    #* For every database, create a metadata record.
    mkdir -p /tmp/hepc.$$-zip
    cd /tmp/hepc.$$-zip/*

    # Information from codeql-database.yml
    primaryLanguage=$(yq '.primaryLanguage' codeql-database.yml)
    sha=$(yq '.creationMetadata.sha' codeql-database.yml)
    cliVersion=$(yq '.creationMetadata.cliVersion' codeql-database.yml)
    creationTime=$(yq '.creationMetadata.creationTime' codeql-database.yml)
    sourceLocationPrefix=$(yq '.sourceLocationPrefix' codeql-database.yml)
    repo=${sourceLocationPrefix##*/}   # keep only last component
    # Get sourceLocationPrefix[-2]
    owner="${sourceLocationPrefix%/*}" # strip last component
    owner="${owner##*/}"               # keep only last component

    # cid for repository / db
    cid=$(echo  "${cliVersion} ${creationTime} ${primaryLanguage} ${sha}" | b2sum |\
              awk '{print substr($1, 1, 6)}')

    # Prepare the metadata record for this DB.
    new_db_fname="${owner}-${repo}-ctsj-${cid}.zip"
    result_url="http://hepc/${db_collection_dir}/${new_db_fname}"
    record='
    {
        "git_branch": "HEAD",
        "git_commit_id": "'${sha}'",
        "git_repo": "'${repo}'",
        "ingestion_datetime_utc": "'${creationTime}'",
        "result_url": "'${result_url}'",
        "tool_id": "9f2f9642-febb-4435-9204-fb50bbd43de4",
        "tool_name": "codeql-'${primaryLanguage}'",
        "tool_version": "'${cliVersion}'",
        "projname": "'${owner}/${repo}'"
    }
'
    cd "$starting_dir"
    rm -fR /tmp/hepc.$$-zip 
    echo "$record" >> $db_collection_dir/metadata.json

    #* Link original file path to collection directory for serving.  Use name including
    # the cid and field separator ctsj 
    cd ${db_collection_dir}
    [ -L ${new_db_fname} ] || ln -s ${zip_path} ${new_db_fname}

    # Interim cleanup
    rm -fR "/tmp/hepc.$$-*"    
done
