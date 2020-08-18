#!/bin/sh

trap 'echo "INT SIGNAL"; sleep 0.01; exit 0' INT
trap 'echo "TERM SIGNAL"; sleep 0.01; exit 0' TERM
trap 'i=0; while [ $i -le 9 ]; do i=$(( i + 1 )); sleep 0.01; echo EXIT ${i}0ms; done' EXIT

i=0; while [ $i -le 9 ]; do i=$(( i + 1 )); sleep 0.01; echo ${i}0ms; done

exit 0
