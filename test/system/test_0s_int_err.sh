#!/bin/sh

trap '>&2 echo "INT SIGNAL"; exit 131' INT

i=1; while [ $i -le 5 ]; do sleep 0.01; i=$(( i + 1 )); done

exit 0
