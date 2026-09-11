#!/bin/bash

__args=( "$@" );

commands() {
	case "$#" in
	0)
		declare solo="";
		declare part="";

		case "${__args[0]:-}" in
		"--completions"|"--generate-roff"|"--man-page")
			;;
		*)
			__args=( "$part" "${__args[@]}" );;
		esac;

		eval "__parts() { [ -n \"\$part\" ] && echo -e ${part@Q}; }; __sections() { [ -n \"\$part\" ] && cat ${0@Q}; }";;
	1)
		declare start="${1:?Start string required}";

		eval "__parts() { grep '^$start' ${0@Q} | cut -b'$(( ${#start} + 1 ))-' | grep -v '^$'; }; __sections() { sed -n -e '/^$start'\${part:-}'$/,/^--/{//!p}' ${0@Q}; }";;
	*)
		declare general="${1:-}";
		shift;
		declare -A sections=();

		for section; do
			sections["${section##*/}"]="$section";
		done;

		printf -v list "%s\n" "${!sections[@]}";

		eval "__parts() { echo -en ${list@Q}; }; __sections() { case \${part:-} in \"\")$(
			if [ -n "$general" ]; then
				echo -n "cat ${general@Q}";
			fi;
		);;$(
			for part in "${!sections[@]}"; do
				echo -n "${part@Q})cat ${sections[$part]@Q};;";
			done;
		)esac; }";;
	esac;

	__handle_parts "${__args[@]}";
}

__description() {
	__sections | sed -n '/^#/!q; 1{/^#!/d}; s/^# *//p';
}

__flags() {
	while read line; do
		printf "%s\0%s\0%s\0" "$(sed -e 's/  \+/ /g; s/^\([^ ]*\).*/\1/' <<< "$line")" "$(sed -e 's/ \+#.*//; s/  +/ /g' <<< "$line" | cut -s -d' ' -f2)" "$(sed -n 's/.* # *//p' <<< "$line")";
	done < <(__sections | sed -n '/^#/d; /^: /!q; s/^:  *//p');
}

__flag_type() {
	declare type="$(sed -e 's/\[\]$//; s/^\[\(.*\)\]$/\1/; s/#*$//' <<< "$1")";

	if [ "$type" != "" ]; then
		printf " %s" "$type";
	fi;
}

__print_flags_usage() {
	while read -r -d '' flag && read -r -d '' type && read -r -d '' desc; do
		flag="${flag%,*}";

		if [ "$flag" = "..." -o "$flag" = "…" ]; then
			additional="1";
			additionalDesc="$desc";
		elif [ "${type: -1}" = "]" -o "$type" = "" ]; then
			echo -n " [${1+\\fI}$flag${1+\\fR}$(__flag_type "${type:-}")]";

			if [ "${type: -2}" = "[]" ]; then
				echo -n "...";
			fi;
		else
			printf " %s%s" "${1+\\fI}$flag${1+\\fR}" "$(__flag_type "$type")";
		fi;
	done < <(__flags);
}

__generate_roff() {
	declare cmd="$(basename "$0")";
	declare desc="$(__description)";
	declare additional="";
	declare additionalDesc="";

	echo ".TH ${cmd^^} 1";
	echo ".SH NAME";
	echo "$cmd";
	echo ".SH SYNOPSIS";
	echo ".B $cmd";

	if [ -v solo ]; then
		echo -n "\f";
		__print_flags_usage "";
	else
		echo -n "\fIsubcommand\fR";
		__print_flags_usage "";
		echo -n " [\fIsubcommand_flags\fR]";
	fi;

	echo " ${additional:+[\fIARGS\fR]...}";
	echo -e "${desc:+.SH DESCRIPTION\n$desc}";

	if [ ! -v solo ]; then
		echo ".SH SUBCOMMANDS";

		while read part; do
			declare desc="$(__description)";

			echo -e ".TP\n.B $part${desc+\n$desc}";
		done < <(__parts);
	fi;

	if __flags | grep -q .; then
		echo ".SH ${solo-GLOBAL }FLAGS";

		while read -r -d '' flag && read -r -d '' type && read -r -d '' desc; do
			if [ "$flag" = "..." -o "$flag" = "…" ]; then
				continue;
			fi;

			echo -en ".TP\n.B ";
			sed -e 's/,/, /g; s/#//g' <<< "$flag$(__flag_type "$type")";
			echo "$desc";
		done < <(__flags);
	fi;

	if [ -n "$additionalDesc" ]; then
		echo ".SH ADDITIONAL ARGUMENTS";
		echo "$additionalDesc";
	fi;

	if [ ! -v solo ]; then
		declare additional="";
		declare additionalDesc="";

		while read part; do
			declare desc="$(__description)";

			echo -e ".ce 1\n.SH SUBCOMMAND: $part\n.SH SYNOPSIS";
			echo -n ".B $cmd \fI$part\fR [\fIglobal_flags\fR]";
			__print_flags_usage "";
			echo " ${additional:+[\fIARGS\fR]...}";

			if [ -n "$desc" ]; then
				echo -e ".SH DESCRIPTION\n$desc";
			fi;

			if __flags | grep -q .; then
				echo ".SH FLAGS";

				while read -r -d '' flag && read -r -d '' type && read -r -d '' desc; do
					if [ "$flag" = "..." -o "$flag" = "…" ]; then
						continue;
					fi;

					echo -e ".TP\n.B ";
					sed -e 's/,/, /g; s/#//g' <<< "$flag$(__flag_type "$type")";
					echo "$desc";
				done < <(__flags);
			fi;
		done < <(__parts);

		if [ -n "$additionalDesc" ]; then
			echo ".SH ADDITIONAL ARGUMENTS";
			echo "$additionalDesc";
		fi;
	fi;
}

__flag_completion() {
	while read -r -d '' flag && read -r -d '' type && read -r -d ''; do
		if [ "$flag" = "..." -o "$flag" = "…" ]; then
			hasExtra=true;
		else
			declare t=" ";

			if [ "${type: -2}" = "[]" ]; then
				type="${type:0:-2}";
				t="a";
			elif [ "${type:0:1}" = "[" -a "${type: -1}" = "]" ]; then
				type="${type:1:-1}";
			fi;

			if [ -z "$type" ]; then
				t+="!";
			elif [ "${type: -1}" != "#" ]; then
				t+="-f";
			fi;

			echo -n "[${flag@Q}]=${t@Q} ";
		fi;
	done < <(__flags);
}

__restrict_completion() {
	cat <<HEREDOC
$1	declare onValue="$([ -v solo ] && echo "false" || echo "true")";
$1	declare isAny="false";

$1	for arg in "\${COMP_WORDS[@]:1:\$(( \$COMP_CWORD - 1 ))}"; do
$1		declare flag="\${opts[\$arg]}";

$1		if \$onValue || [ -z "\${flag:-}" ]; then
$1			onValue=false;

$1			continue;
$1		fi;

$1		if [ "\${flag:0:1}" = ' ' ]; then
$1			unset opts[\$arg];
$1		fi;

$1		if [ "\${flag:1}" = '!' ]; then
$1			continue;
$1		fi;

$1		onValue=true;
$1		isAny="\${flag:1}";
$1	done;

$1	if \$onValue; then
$1		opts=();
$1		hasExtra="\$isAny";
$1	fi;
HEREDOC
}

__completions() {
	declare hasV="$(compgen -V var -W "a" "" &> /dev/null && echo true || echo false)";
	declare fn="__completions";
	declare hasExtra=false;

	while declare -F "$fn" > /dev/null; do
		fn="__do_completion_$RANDOM";
	done;

	echo -en "$fn() {\n\tdeclare -A opts=( ";

	if [ -v solo ]; then
		__flag_completion;
	fi;

	echo -en ");\n\tdeclare hasExtra=\"";

	if $hasExtra; then
		echo -n "-f";
	fi;

	echo -e "\";";
	if [ -v solo ]; then
		__restrict_completion "";
		echo;
	else
		echo -en "\n\tif [ \$COMP_CWORD -eq 1 ]; then\n\t\topts=( ";

		while read part; do
			echo -n "[${part@Q}]='' ";
		done < <(__parts);

		echo -e ");\n\telse";

		if [ "$(__flags | wc -c)" -gt 0 ]; then
			echo -en "\t\topts=( ";

			__flag_completion;

			echo -e ");";
		fi;

		declare primaryHasExtra="$hasExtra";

		if $hasExtra; then
			echo -e "\t\thasExtra=\"-f\";";
		fi;

		echo -e "\t\tcase \"\${COMP_WORDS[1]}\" in";

		while read part; do
			echo -en "\t\t${part@Q})\n\t\t\topts+=( ";

			__flag_completion;

			echo -n ");";

			if $hasExtra && ! $primaryHasExtra; then
				echo -en "\n\t\t\thasExtra=\"-f\";";
			fi;

			echo ";";
		done < <(__parts);

		echo -e "\t\tesac;\n";

		__restrict_completion "	";

		echo -e "\tfi;\n";
	fi;

	cat <<HEREDOC
	if [ -n "\$hasExtra" -o \${#opts[@]} -gt 0 ]; then
		$($hasV || echo -n "read -d '\\n' -a COMPREPLY < <(")compgen$($hasV && echo -n " -V COMPREPLY" || true) \${opts[@]+ -W "\${!opts[*]}"} \${hasExtra:---}\${hasExtra:+ --} "\${COMP_WORDS[\$COMP_CWORD]}"$($hasV || echo -n ")");
		readarray -t COMPREPLY < <(printf '%s\\n' "\${COMPREPLY[@]}" | LC_ALL=C sort);
	fi;
}

complete -F ${fn@Q} ${0@Q};
HEREDOC
}

__print_flags() {
	declare part="${1:-}";
	declare maxLength="$(
		while read -r -d '' flag && read -r -d '' && read -r -d '' desc; do
			if [ -n "$desc" -a "$flag" != "..." -a "$flag" != "…" ]; then
				printf "%s\n" "$flag";
			fi;
		done < <(__flags) | wc -L;
	)";

	if [ "$maxLength" -gt 0 ]; then
		if [ -z "$part" ]; then
			echo -e "\nGlobal Flags:";
		else
			echo -e "\nFlags:";
		fi;

		while read -r -d '' flag && read -r -d '' type && read -r -d '' desc; do
			if [ -n "$desc" -a "$flag" != "..." -a "$flag" != "…" ]; then
				printf "  %-${maxLength}s" "$flag";

				if [ -n "$desc" ]; then
					echo -n " ";
					eval "printf ' %s' $desc";
					echo;
				fi;
			fi;
		done < <(__flags);
	fi;
}

__usage() {
	declare part="${1:-}";
	declare additional="";
	declare additionalDesc="";

	echo -n "Usage: $0 [--help]${solo- ${part:-SUBCOMMAND}}";

	if [ -n "$part" ]; then
		part="" __print_flags_usage;
	fi;

	__print_flags_usage;

	echo "${additional:+ ARGS}";

	__description;

	if [ -n "$additionalDesc" ]; then
		echo -e "\nArgs:\n$(sed -e 's/^/  /' <<< "$additionalDesc")";
	fi;
}

__help() {
	declare maxLength="$(__parts | wc -L)";

	if [ "$maxLength" -eq 0 ]; then
		echo "No subcommands defined.";

		exit 127;
	fi;

	__usage;
	echo -e "\nSubcommands:";

	while read part; do
		declare desc="$(__description)";
		printf "  %-${maxLength}s${desc:+  }%s\n" "$part" "$desc";
	done < <(__parts);

	__print_flags;
}

__section_help() {
	__usage "$part";
	__print_flags "$part";
	__print_flags;
}

__bind_flags() {
	for flag in "${!setFlags[@]}"; do
		echo "declare -g $(tr -d '-' <<< "$flag")=${setFlags[$flag]@Q}";
	done;

	for flag in "${!arrays[@]}"; do
		echo "declare -g -a $(tr -d '-' <<< "$flag")=${arrays[$flag]} )";
	done;
}

__handle_parts() {
	case "${1:-}" in
	"--help")
		__help;

		exit 0;;
	"--completions")
		__completions;

		exit 0;;
	"--generate-roff")
		__generate_roff;

		exit 0;;
	"--man-page")
		man -l <(__generate_roff);

		exit 0;;
	esac;

	if [ -z "${1:-}" ]; then
		{
			echo -e "Error: Subcommand required\n";
			__help;

			exit 127;
		} >&2;
	fi;

	if [ -z "$(__parts | grep "^$1$")" ]; then
		{
			echo -e "Error: Unknown subcommand $1\n";
			__help;

			exit 127;
		} >&2;
	fi;

	declare part="$1";
	shift;
	declare -A flags=();
	declare -A required=();
	declare -A arrays=();
	declare -A setFlags=();
	declare -A aliases=();
	declare -a args=();
	declare hasAdditional=false;

	while read -r -d '' flag && read -r -d '' type && read -r -d ''; do
		{
			read flag;

			while read alias; do
				aliases[$alias]="$flag";
			done;
		} < <(tr ',' '\n' <<< "$flag");

		if [ "$flag" = "..." -o "$flag" = "…" ]; then
			hasAdditional=true;
		elif [ "$type" = "" ]; then
			setFlags[$flag]="false";
			flags[$flag]="$type";
		elif [ "${type: -2}" = "[]" ]; then
			arrays[$flag]="(";
			flags[$flag]="${type:0:-2}";
		elif [ "${type:0:1}" = "[" -a "${type: -1}" = "]" ]; then
			flags[$flag]="${type:1:-1}";
		else
			required[$flag]=true;
			flags[$flag]="$type";
		fi;
	done < <(
		part="" __flags;
		__flags;
	);

	while [ $# -gt 0 ]; do
		declare flag="$1";
		shift;

		if [ -v aliases[$flag] ]; then
			flag="${aliases[$flag]}";
		fi;

		if [ "$flag" = "--help" ]; then
			__section_help;

			exit 0;
		elif [ ! -v flags[$flag] ]; then
			if [ "${flag:0:1}" = "-" -a "${flag:0:2}" != "--" ]; then
				declare allBinary=true;

				while IFS= read -r -n 1 f; do
					f="-$f";

					if [ -v aliases["$f"] ]; then
						f="${aliases[$f]}";
					fi;

					if [ "${flags["$f"]-!}" != "" ]; then
						allBinary=false;

						break;
					fi;
				done < <(echo -n "${flag:1}");

				if $allBinary; then
					while IFS= read -r -n 1 f; do
						declare flag="-$f";

						if [ -v aliases[$flag] ]; then
							flag="${aliases[$flag]}";
						fi;

						if [ -v arrays["$flag"] ]; then
							arrays["$flag"]="${arrays[$flag]} true";
						else
							setFlags[$flag]="true";
						fi;
					done < <(echo -n "${flag:1}");

					continue;
				fi;
			fi;

			if $hasAdditional; then
				args+=( "$flag" );

				continue;
			else
				{
					echo -e "Error: Unknown flag: $flag\n";
					__section_help;

					exit 2;
				} >&2;
			fi;
		fi;

		if [ "${flags[$flag]}" = "" ]; then
			if [ -v arrays["$flag"] ]; then
				arrays["$flag"]="${arrays[$flag]} true";
			else
				setFlags[$flag]="true";
			fi;

			continue;
		elif [ $# -eq 0 ]; then
			{
				echo -e "Error: Flag requires value: $flag\n";
				__section_help;

				exit 2;
			} >&2;
		elif [ "${flags[$flag]: -2}" = "##" -a -z "$(grep "^[+-]\?[0-9]*\(\.[0-9]\+\)\?$" <<< "$1")" -o "${flags[$flag]: -1}" = "#" -a "${flags[$flag]: -2}" != "##" -a -z "$(grep "^[+-]\?[0-9]\+$" <<< "$1")" ]; then
			{
				echo -e "Error: Invalid flag value: $flag "$1"\n";
				__section_help;

				exit 2;
			} >&2;
		elif [ -v arrays["$flag"] ]; then
			arrays["$flag"]="${arrays[$flag]} ${1@Q}";
		elif [ -v setFlags["$flag"] ]; then
			{
				echo -e "Error: Flag already set: $flag\n";
				__section_help;

				exit 2;
			} >&2;
		else
			setFlags[$flag]="$1";
		fi;

		shift;
	done;

	for flag in "${!required[@]}"; do
		if [ ! -v setFlags[$flag] ]; then
			{
				echo -e "Error: Required flag not set: $flag\n";
				__section_help;

				exit 2;
			} >&2;
		fi;
	done;

	if [ -v solo ]; then
		eval "$(__bind_flags)";

		return 0;
	fi;

	declare script="$(mktemp --tmpdir=/dev/shm 2> /dev/null || mktemp)";

	{
		__bind_flags;
		part="" __sections;
		__sections;
	} > "$script";

	exec {fd}< "$script";
	rm -f "$script";

	mapfile -d '' CMD < <(
		{
			__sections | head -n1;
			part="" __sections | head -n1;
			echo -n "#!$BASH";
		} | grep "^#!" | head -n1 | cut -b 3- | xargs printf '%s\0';
	);

	if declare -F "${CMD[0]:-}" > /dev/null; then
		"${CMD[@]}" /proc/self/fd/$fd "${args[@]}";

		exit $?;
	fi;

	exec "${CMD[@]}" /proc/self/fd/$fd "${args[@]}";
}
