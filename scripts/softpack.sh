#!/bin/bash

set -euo pipefail;

declare base="$(dirname "$0")";
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

setBind() {
	declare slTemp="$(mktemp -d)";
	declare dpTemp="$(mktemp -d)";
	export TMP="$(mktemp -d)";
	trap "rm -rf ${slTemp@Q}; rm -rf ${dpTemp@Q}; rm -rf ${TMP@Q}" EXIT;

	bind="$slTemp:/usr/lib/R/site-library/,$dpTemp:/usr/lib/python3/dist-packages,$TMP:/tmp";
}

runContainer() {
	declare script="$1";
	shift;

	declare bind="";
	setBind;

	cat "$script" | if [ "${aptRepo:0:5}" = "s3://" ]; then
		singularity exec --fusemount "$(aptMount)" --bind "$bind" "$base/softpack.sif" bash "$script" "$@"
	else
		singularity exec --bind "$bind" "$base/softpack.sif" bash "$script" "$@";
	fi;
}

runShell() {
	declare bind="";
	setBind;

	if [ "${aptRepo:0:5}" = "s3://" ]; then
		singularity shell --fusemount "$(aptMount)" --bind "$bind" "$base/softpack.sif";
	else
		singularity shell --bind "$bind" "$base/softpack.sif";
	fi;
}

. "$base/commands.sh";

declare parts=( $(find "$base" -maxdepth 1 -type f -not -iname "*.sh" -not -iname "*.sif" -not -iname "*.def") );

commands "$base/global.sh" "${parts[@]}";
