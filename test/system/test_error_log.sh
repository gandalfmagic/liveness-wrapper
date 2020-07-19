#!/bin/sh

trap 'echo "INT SIGNAL"; exit 0' INT

i=1; while [ $i -le 10 ]; do sleep 0.01; i=$(( i + 1 )); done
>&2 echo "write a line to stderr"
i=1; while [ $i -le 10 ]; do sleep 0.01; i=$(( i + 1 )); done

exit 0
