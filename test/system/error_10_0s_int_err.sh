#!/bin/sh

trap 'echo "INT SIGNAL"; sleep 0.1; exit 131' INT
trap 'echo "TERM SIGNAL"; sleep 0.1; exit 131' TERM

i=0; while [ $i -le 9 ]; do i=$(( i + 1 )); sleep 0.01; echo ${i}0ms; done

>&2 echo "write a line to stderr"
exit 10
