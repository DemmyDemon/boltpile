#!/usr/bin/env bash
FILE=$1
if [ -z "$FILE" ]; then 
    echo "Specify a file"
    exit 1
fi
PILE=84ed8bd7-f8a1-4e4b-bc4d-85868208dae5
DATA=$(/usr/bin/env curl -s -X POST -H "Content-Type: multipart/form-data" -F "data=@$FILE" http://localhost:1995/$PILE/)
ENTRY=$(jq -r '.entry' <<< "$DATA")
echo http://localhost:1995/$PILE/$ENTRY
/usr/bin/env curl -s http://localhost:1995/$PILE/$ENTRY

