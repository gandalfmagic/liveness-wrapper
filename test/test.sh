#!/bin/sh

function w() {
sleep 0.3
}
trap "w" EXIT

sleep 0.1
exit 0
