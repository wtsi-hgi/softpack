#!/bin/bash

__args=( "$@" );

sections() {
	declare start="${1:?Start string required}";

	declare partsFn="grep '^$start' ${0@Q} | cut -b'$(( ${#start} + 1 ))-' | grep -v '^$'";
	declare sectionFn="sed -n -e '/^$start'\${part:-}'$/,/^--/{//!p}' ${0@Q}";

	__handle_parts;
}

files() {
	declare general="${1:-}";
	shift;
	declare base="$(dirname "$0")/";
	declare -A sections=();

	for section; do
		sections["${section##*/}"]="$section";
	done;

	printf -v list "%s\n" "${!sections[@]}";

	declare partsFn="echo -en ${list@Q}";
	declare sectionFn="(cd ${base@Q};case \${part:-} in \"\")$(
		if [ -n "$general" ]; then
			echo -n "cat ${general@Q}";
		fi;
	);;$(
		for part in "${!sections[@]}"; do
			echo -n "${part@Q})cat ${sections[$part]@Q};;";
		done;
	)esac)";

	__handle_parts;
}

__parts() {
	eval "$partsFn";
}

__sections() {
	eval "$sectionFn";
}

__description() {
	__sections | sed -n '/^#/!q; p' | sed -e '1{/^#!/d}' -e 's/^# *//';
}

__flags() {
	while read line; do
		printf "%s\0%s\0%s\0" "$(sed -e 's/  +/ /g' <<< "$line" | cut -d' ' -f2)" "$(sed -e 's/  +/ /g' <<< "$line" | cut -d'#' -f1 | cut -d' ' -f3)" "$(cut -s -d'#' -f2- <<< "$line" | sed -e 's/^ *//')";
	done < <(__sections | sed -n '1,/^[^#:]/p' | grep "^: ");
}

__print_flags() {
	declare part="${1:-}";
	declare maxLength="$(
		__flags | while read -r -d '' flag && read -r -d '' && read -r -d ''; do
			printf "%s\n" "$flag";
		done | wc -L;
	)";

	if [ "$maxLength" -gt 0 ]; then
		if [ -z "$part" ]; then
			echo -e "\nGlobal Flags:";
		else
			echo -e "\nFlags:";
		fi;

		__flags | while read -r -d '' flag && read -r -d '' type && read -r -d '' desc; do
			if [ -n "$desc" -a "$flag" != "..." -a "$flag" != "…" ]; then
				printf "  %-${maxLength}s${desc:+  }%s\n" "$flag" "$desc";
			fi;
		done;
	fi;
}

__usage() {
	declare part="${1:-}";
	declare additional=false;
	declare additionalDesc="";

	echo -n "Usage: $0 [--help] ${part:-SUBCOMMAND}";

	while read -r -d '' flag && read -r -d '' type && read -r -d '' desc; do
		flag="${flag%,*}";

		if [ "$flag" = "..." -o "$flag" = "…" ]; then
			additional=true;
			additionalDesc="$desc";
		elif [ "${type:0:1}" = "[" -o "$type" = "boolean" ]; then
			echo -n " [$flag$(__flag_type "${type:-value}")]";
		else
			printf " %s%s" "$flag" "$(__flag_type "${type:-value}")";
		fi;
	done < <(
		if [ -n "$part" ]; then
			part="" __flags;
		fi;

		__flags;
	);

	if $additional; then
		echo " ARGS";
	else
		echo;
	fi;

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

__flag_type() {
	declare type="$(sed -e 's/^\[\(.*\)\]$/\1/' <<< "$1")";

	if [ "$type" != "boolean" ]; then
		printf " %s" "$type";
	fi;
}

__section_help() {
	__usage "$part";
	__print_flags "$part";
	__print_flags;
}

__script() {
	part="" __sections;

	for flag in "${!setFlags[@]}"; do
		echo "declare $(tr -d '-' <<< "$flag")=${setFlags[$flag]@Q}";
	done;

	__sections;
}

__handle_parts() {
	set -- "${__args[@]}";

	if [ "${1:-}" = "--help" ]; then
		__help;

		exit 0;
	fi;

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

		if [ "$flag" = "..." ]; then
			hasAdditional=true;
		elif [ "$type" = "boolean" ]; then
			setFlags[$flag]="false";
			flags[$flag]="$type";
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

		if [ "${flags[$flag]}" = "boolean" ]; then
			setFlags[$flag]="true";
		elif [ $# -eq 0 ]; then
			{
				echo -e "Error: Flag requires value: $flag\n";
				__section_help;

				exit 2;
			} >&2;
		elif [ "${flags[$flag]}" = "number" -a -z "$(grep "^[+-]\?[0-9]*\(\.[0-9]\+\)\?$" <<< "$1")" -o "${flags[$flag]}" = "integer" -a -z "$(grep "^[+-]\?[0-9]\+$" <<< "$1")" ]; then
			{
				echo -e "Error: Invalid flag value: $flag "$1"\n";
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

	mapfile -d '' CMD < <(
		{
			__sections | head -n1;
			part="" __sections | head -n1;
			echo -n "#!$BASH";
		} | grep "^#!" | head -n1 | cut -b 3- | xargs printf '%s\0';
	);

	if declare -F "${CMD[0]:-}" > /dev/null; then
		"${CMD[@]}" <(__script) "${args[@]}";

		exit $?;
	fi;

	exec -a "$part" "${CMD[@]}" <(__script) "${args[@]}";
}
