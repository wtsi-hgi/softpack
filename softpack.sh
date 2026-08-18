#!/bin/bash

set -euo pipefail;

declare aptRepo="$(grep "^aptrepo:" "${SOFTPACK_CONFIG:-$HOME/.softpack/config.yaml}" | head -n1 | cut -d':' -f2- | sed -e 's/^ *//')";

aptMount() {
	declare host="$(echo "$aptRepo" | cut -d'/' -f3)";
	declare path="/$(echo "$aptRepo" | cut -d'/' -f4-)";

	declare mount="container:s3fs -o logfile=/dev/null"

	if [ -f ~/.aws/config ]; then
		declare endpoint_url="$(cat ~/.aws/config | grep "^endpoint_url" | head -n1 | sed -e 's/.*= *//')";

		if [ -n "$endpoint_url" ]; then
			mount+=" -o url=$endpoint_url";
		fi;
	fi;

	mount+=" $host";

	if  [ -n "$path" ]; then
		if [ "${path:0:1}" != "/" ]; then
			path="/$path";
		fi;

		if [ "${path:-1}" != "/" ]; then
			path+="/";
		fi;

		mount+=":$path";
	fi;

	mount+=" /repo";

	echo "$mount";
}

runContainer() {
	declare script="$1";
	shift;

	cat "$script" | if [ "${aptRepo:0:5}" = "s3://" ]; then
		singularity shell --fusemount "$(aptMount)" softpack.sif -- "$@";
	else
		singularity shell --bind "$aptRepo:/repo" softpack.sif -- "$@";
	fi;
}

runShell() {
	if [ "${aptRepo:0:5}" = "s3://" ]; then
		singularity shell --fusemount "$(aptMount)" softpack.sif;
	else
		singularity shell --bind "$aptRepo:/repo" softpack.sif;
	fi;
}

. commands.sh;

declare parts=( $(ls -I "*.sh" -I "*.sif" -I "*.def") );

files global.sh "${parts[@]}";
