#!/bin/sh

function w() {
sleep 0.1
}
trap "w" EXIT

sleep 0.2
exit 10
