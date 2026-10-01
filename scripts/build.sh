#!/bin/bash
# Build the softpack utility container
: --aptCache  # Enable apt-cache redirection

. commands.sh;

commands;

declare def="softpack.def";

if [ -n "${aptCache:-}" ]; then
	export TMP="$(mktemp -d)";
	trap "rm -rf ${TMP@Q}" EXIT;

	def="$(mktemp)";

	{
		head -n4 softpack.def
		cat <<-HEREDOC
		%post
		export "http_proxy=http://127.0.0.1:3142"
		#echo 'Acquire::http { Proxy "http://127.0.0.1:3142"; }' >> /etc/apt/apt.conf.d/proxy
HEREDOC
		tail -n+6 softpack.def
	}  > "$def";

fi;

sudo singularity build softpack.sif "$def"
