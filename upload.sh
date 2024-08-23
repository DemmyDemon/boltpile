#!/usr/bin/env bash
FILE=$1
if [ -z "$FILE" ]; then 
    echo "Specify a file"
    exit 1
fi
PILE=84ed8bd7-f8a1-4e4b-bc4d-85868208dae5
TOKEN=d5e1c56e-c8db-4b2a-adbe-60167ddee431
DATA=$(/usr/bin/env curl -s -X POST --oauth2-bearer "$TOKEN" -H "Content-Type: multipart/form-data" -F "data=@$FILE" http://localhost:1995/$PILE/)
ENTRY=$(jq -r '.entry' <<< "$DATA")
if [ "$ENTRY" == "null" ]; then
    echo $DATA
    exit 1
fi
echo http://localhost:1995/$PILE/$ENTRY
/usr/bin/env curl -s --oauth2-bearer "$TOKEN" http://localhost:1995/$PILE/$ENTRY

