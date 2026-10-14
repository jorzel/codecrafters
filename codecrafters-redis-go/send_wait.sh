#!/bin/bash

# Send SET command and WAIT right after
redis-cli <<EOF
SET ba 786
WAIT 5 2000
EOF