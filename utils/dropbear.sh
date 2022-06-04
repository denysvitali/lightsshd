#!/bin/bash
set -ex
ORIGINAL_BINARY=${ORIGINAL_BINARY:-/mnt/us/lightsshd}
LISTEN_ADDR="0.0.0.0:2222"

while getopts "h?Rp:P:" opt; do
	case "$opt" in
		h)
		$ORIGINAL_BINARY -h
		exit 0
		;;
		R)
		# Create hostkeys as required
    ;;
		p)
		LISTEN_ADDR="$OPTARG"
		;;
		P)
		PID_FILE="$OPTARG"
		;;
		*)
		echo "Not recognized, $opt" >> /tmp/lightsshd_log
		;;
	esac
done

if ! echo -n "$LISTEN_ADDR" | grep ":"; then
  LISTEN_ADDR=":$LISTEN_ADDR"
fi

$ORIGINAL_BINARY \
	-L "$LISTEN_ADDR" \
	-a /mnt/us/koreader/settings/SSH/authorized_keys \
	-P "$PID_FILE" \
	-k /mnt/us/koreader/settings/SSH/lightsshd/ssh_host_ed25519_key &