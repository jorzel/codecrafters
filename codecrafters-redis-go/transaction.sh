#!/bin/bash

# Send SET command and WAIT right after
redis-cli --verbose <<EOF
MULTI
INCR a
INCR b
GET b
EXEC
EOF