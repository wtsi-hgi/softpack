#!/bin/bash

set -euo pipefail;

declare base="$(realpath "$(dirname "$0")")";
declare aptRepo="$(grep "^aptrepo:" "${SOFTPACK_CONFIG:-$HOME/.softpack/config.yaml}" | head -n1 | cut -d':' -f2- | sed -e 's/^ *//')";
export TMP="$(mktemp -d)";
trap "rm -rf ${TMP@Q}" EXIT;

aptMount() {
	declare host="$(echo "$aptRepo" | cut -d'/' -f3)";
	declare path="/$(echo "$aptRepo" | cut -d'/' -f4-)";

	#declare mount="container:s3fs -d -o curldbg -o dbglevel=info";
	declare mount="container:s3fs -o logfile=/dev/null";

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
	declare slTemp="$(TMPDIR="$TMP" mktemp -d)";
	declare dpTemp="$(TMPDIR="$TMP" mktemp -d)";

	bind="$slTemp:/usr/lib/R/site-library/,$dpTemp:/usr/lib/python3/dist-packages,$TMP:/tmp";

	if [ -v SOFTPACK_LOCAL_REPO ]; then
		bind+=",$SOFTPACK_LOCAL_REPO:/repo-local";
	fi;
}

runContainer() {
	declare script="$1";
	shift;

	declare bind="";

	setBind;

	startContainer exec --bind "$bind" "$base/softpack.sif" bash "$script" "$@"
}

startContainer() {
	declare cmd="$1";
	shift;

	if [ "${aptRepo:0:5}" = "s3://" ]; then
		singularity "$cmd" --fusemount "$(aptMount)" "$@"
	else
		singularity exec "$@";
	fi;
}

runShell() {
	declare bind="";
	setBind;

	startContainer shell --bind "$bind" "$base/softpack.sif" bash "$script" "$@"
}

buildContainer() {
	declare root="$(TMPDIR="$TMP" mktemp -d)";
	declare container="$(TMPDIR="$TMP" mktemp -d)";

	singularity build --sandbox "$root/r" docker://ubuntu:latest;
	singularity exec --bind "$root/r/:/r" "$base/softpack.sif" cp /usr/local/bin/s3fs /r/usr/local/bin/;
	mkdir "$root/r/"{repo,build};

	if [ "${aptRepo:0:5}" = "s3://" ]; then
		singularity exec --writable "$root/r" bash -c "export DEBIAN_FRONTEND=noninteractive; apt update && apt -y -o DPkg::Options::=--force-not-root install --no-install-recommends ca-certificates s3fs libcurl4-openssl-dev libfuse-dev libxml2-dev libssl-dev";
		singularity exec --bind "$root/r/:/build" "$base/softpack.sif" bash -c "cp /usr/local/bin/s3fs /build/usr/local/bin/";
	fi;

	mv "$root/r/etc/"passwd{,.new};
	mv "$root/r/etc/"group{,.new};
	ln -s "/etc/passwd.new" "$root/r/etc/passwd";
	ln -s "/etc/group.new" "$root/r/etc/group";

	mkdir -p "$root/r/.singularity.d/env";
	cat > "$root/r/.singularity.d/env/99-etc.sh" <<-HEREDOC
	rm /etc/{passwd,group}
	mv /etc/passwd{.new,}
	mv /etc/group{.new,}
HEREDOC

	chmod +x "$root/r/.singularity.d/env/99-etc.sh";
	startContainer exec --bind "$(TMPDIR="$TMP" mktemp -d):/tmp" --writable "$root/r" bash "$1";
}

. "$base/commands.sh";

declare parts=( $(find "$base" -maxdepth 1 -type f -not -iname "*.sh" -not -iname "*.sif" -not -iname "*.def" -not -iname "builder" -not -iname "update-cran" -not -iname "commit") );

if [ -v SOFTPACK_LOCAL_REPO ]; then
	parts+=( "commit" );
else
	parts+=( "builder" "update-cran" );
fi;

commands "$base/global.sh" "${parts[@]}";
