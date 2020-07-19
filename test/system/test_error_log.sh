#!/bin/sh

i=1; while [ $i -le 5 ]; do sleep 0.01; i=$(( i + 1 )); done
>&2 echo "write a line to stderr"
i=1; while [ $i -le 5 ]; do sleep 0.01; i=$(( i + 1 )); done

exit 0
