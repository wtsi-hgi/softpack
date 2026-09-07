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

runContainer() {
	declare script="$1";
	shift;

	declare slTemp="$(mktemp -d)";
	export TMP="$(mktemp -d)";
	trap "rm -rf ${slTemp@Q}; rm -rf ${TMP@Q}" EXIT;

	cat "$script" | if [ "${aptRepo:0:5}" = "s3://" ]; then
		singularity exec --fusemount "$(aptMount)" --bind "$slTemp:/usr/lib/R/site-library/,$TMP:/tmp" "$base/softpack.sif" bash "$script" "$@"
	else
		singularity exec --bind "$aptRepo:/repo,$slTemp:/usr/lib/R/site-library/,$TMP:/tmp" "$base/softpack.sif" bash "$script" "$@";
	fi;
}

runShell() {
	declare slTemp="$(mktemp -d)";
	export TMP="$(mktemp -d)";
	trap "rm -rf ${slTemp@Q}; rm -rf ${TMP@Q}" EXIT;

	if [ "${aptRepo:0:5}" = "s3://" ]; then
		singularity shell --fusemount "$(aptMount)" --bind "$slTemp:/usr/lib/R/site-library/,$TMP:/tmp" "$base/softpack.sif";
	else
		singularity shell --bind "$aptRepo:/repo,$slTemp:/usr/lib/R/site-library/,$TMP:/tmp" "$base/softpack.sif";
	fi;
}

. "$base/commands.sh";

declare parts=( $(find "$base" -maxdepth 1 -type f -not -iname "*.sh" -not -iname "*.sif" -not -iname "*.def") );

commands "$base/global.sh" "${parts[@]}";
