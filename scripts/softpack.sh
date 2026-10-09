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
	declare mount="container:s3fs -o logfile=/dev/null -o compat_dir -o complement_stat -o disable_noobj_cache ";

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

	addLocalRepoBind;
}

addLocalRepoBind() {
	if [ -v SOFTPACK_LOCAL_REPO ]; then
		bind+=",$SOFTPACK_LOCAL_REPO:/test-repo";
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
		singularity "$cmd" --bind "$aptRepo:/repo" "$@";
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
	mkdir "$root/r/"{build,test-repo};

	coproc SERVER { startContainer exec "$base/softpack.sif" repo-http 2>&1; };
	trap "kill $SERVER_PID; rm -rf ${TMP@Q}" EXIT;

	read -r line <&"${SERVER[0]}";
	#cat <&"${SERVER[0]}" > /dev/null &

	if [ "${line:0:14}" != "Listening on: " ]; then
		echo "Failed to start repo HTTP server: $line" >&2;

		exit 1;
	fi;

	echo "http://127.0.0.1:${line:14}" > $root/r/repo;

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

	declare bind="";

	addLocalRepoBind;

	chmod +x "$root/r/.singularity.d/env/99-etc.sh";
	singularity exec --bind "$bind" --bind "$(TMPDIR="$TMP" mktemp -d):/tmp" --writable "$root/r" bash "$1";
}

. "$base/commands.sh";

declare parts=( $(find "$base/global/" -maxdepth 1 -type f) );

if [ -v SOFTPACK_LOCAL_REPO ]; then
	parts+=( $(find "$base/building/" -maxdepth 1 -type f) );
else
	parts+=( $(find "$base/normal/" -maxdepth 1 -type f) );
fi;

commands "$base/global.sh" "${parts[@]}";
