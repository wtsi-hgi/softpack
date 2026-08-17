#!/bin/bash

__args=( "$@" );

sections() {
	declare start="${1:?Start string required}";

	__handleParts "grep '^$start' ${0@Q} | cut -b'$(( ${#start} + 1 ))-' | grep -v '^$'" "sed -n -e '/^$start'\${part:-}'$/,/^--/{//!p}' ${0@Q}";
}

files() {
	declare general="${1:-}";
	shift;
	declare -a sections=( "$@" );
	declare base="$(dirname "$0")/";

	printf -v list "%s\n" "${sections[@]}";
	printf -v in "|%q" "${sections[@]}";

	__handleParts "echo -en ${list@Q}" "case \${part:-} in \"\") $(
		if [ -n "$general" ]; then
			echo "(cd ${base@Q};cat ${general@Q})";
		fi;
	);;${in:1}) cd ${base@Q};cat \$part;;esac";
}

__description() {
	sed -n '/^#/!q; p' | sed -e '1{/^#!/d}' -e 's/^# *//';
}

__flags() {
	while read line; do
		printf "%s\0%s\0%s\0" "$(sed -e 's/  +/ /g' <<< "$line" | cut -d' ' -f2)" "$(sed -e 's/  +/ /g' <<< "$line" | cut -d'#' -f1 | cut -d' ' -f3)" "$(cut -s -d'#' -f2- <<< "$line" | sed -e 's/^ *//')";
	done < <(sed -n '1,/^[^#:]/p' | grep "^: ");
}

__print_flags() {
	declare sectionFn="$1";
	declare part="${2:-}";
	declare maxLength="$(
		eval "$sectionFn" | __flags | while read -r -d '' flag && read -r -d '' && read -r -d ''; do
			printf "%s\n" "$flag";
		done | wc -L;
	)";

	if [ "$maxLength" -gt 0 ]; then
		if [ -z "$part" ]; then
			echo -e "\nGlobal Flags:";
		else
			echo -e "\nFlags:";
		fi;

		while read -r -d '' flag && read -r -d '' type && read -r -d '' desc; do
			if [ -n "$desc" -a "$flag" != "..." ]; then
				printf "  %-${maxLength}s  %s\n" "$flag" "$desc";
			fi;
		done < <(eval "$sectionFn" | __flags);
	fi;
}

__usage() {
	declare sectionFn="$1";
	declare part="${2:-}";
	declare additional=false;
	declare additionalDesc="";

	echo -n "Usage: $0 [--help] ${part:-SUBCOMMAND}";

	while read -r -d '' flag && read -r -d '' type && read -r -d '' desc; do
		flag="${flag%,*}";

		if [ "$flag" = "..." ]; then
			additional=true;
			additionalDesc="$desc";
		elif [ "${type:0:1}" = "[" ]; then
			echo -n " [$flag$(__flag_type "${type:-value}")]";
		else
			printf " %s%s" "$flag" "$(__flag_type "${type:-value}")";
		fi;
	done < <(
		if [ -n "$part" ]; then
			part="" eval "$sectionFn" | __flags;
		fi;

		eval "$sectionFn" | __flags;
	);

	if $additional; then
		echo " ARGS";
	else
		echo;
	fi;

	eval "$sectionFn" | __description;

	if [ -n "$additionalDesc" ]; then
		echo -e "\nArgs:\n$(sed -e 's/^/  /' <<< "$additionalDesc")";
	fi;
}

__help() {
	declare partsFn="$1";
	declare sectionFn="$2";
	declare maxLength="$(eval "$partsFn" | wc -L)";

	if [ "$maxLength" -eq 0 ]; then
		echo "No subcommands defined.";

		exit 127;
	fi;

	__usage "$sectionFn";
	echo -e "\nSubcommands:";

	while read part; do
		printf "  %-${maxLength}s  %s\n" "$part" "$(eval "$sectionFn" | __description)";
	done < <(eval "$partsFn");

	__print_flags "$sectionFn";
}

__flag_type() {
	declare type="$(sed -e 's/^\[\(.*\)\]$/\1/' <<< "$1")";

	if [ "$type" != "boolean" ]; then
		printf " %s" "$type";
	fi;
}

__section_help() {
	declare sectionFn="$1";
	declare part="$2";

	__usage "$sectionFn" "$part";
	__print_flags "$sectionFn" "$part";
	__print_flags "$sectionFn";
}

__handleParts() {
	declare partsFn="$1";
	declare sectionFn="$2";

	set -- "${__args[@]}";

	if [ "${1:-}" = "--help" ]; then
		__help "$partsFn" "$sectionFn";

		exit 0;
	fi;

	if [ -z "${1:-}" ]; then
		{
			echo -e "Error: Subcommand required\n";
			__help "$partsFn" "$sectionFn";

			exit 127;
		} >&2;
	fi;

	if [ -z "$(eval "$partsFn" | grep "^$1$")" ]; then
		{
			echo -e "Error: Unknown subcommand $1\n";
			__help "$partsFn" "$sectionFn";

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
		elif [ "${type:0:1}" = "[" -a "${type: -1}" = "]" ]; then
			flags[$flag]="${type:1:-1}";
		else
			required[$flag]=true;
			flags[$flag]="$type";
		fi;
	done < <(
		part="" eval "$sectionFn" | __flags;
		eval "$sectionFn" | __flags;
	);

	while [ $# -gt 0 ]; do
		declare flag="$1";
		shift;

		if [ -v aliases[$flag] ]; then
			flag="${aliases[$flag]}";
		fi;

		if [ "$flag" = "--help" ]; then
			__section_help "$sectionFn" "$part";

			exit 0;
		elif [ ! -v flags[$flag] ]; then
			if $hasAdditional; then
				args+=( "$flag" );

				continue;
			else
				{
					echo -e "Error: Unknown flag: $flag\n";
					__section_help "$sectionFn" "$part";

					exit 2;
				} >&2;
			fi;
		fi;

		if [ "${flags[$flag]}" = "boolean" ]; then
			setFlags[$flag]="true";
		elif [ $# -eq 0 ]; then
			{
				echo -e "Error: Flag requires value: $flag\n";
				__section_help "$sectionFn" "$part";

				exit 2;
			} >&2;
		elif [ "${flags[$flag]}" = "number" -a -z "$(grep "^[+-]\?[0-9]*\(\.[0-9]\+\)\?$" <<< "$1")" -o "${flags[$flag]}" = "integer" -a -z "$(grep "^[+-]\?[0-9]\+$" <<< "$1")" ]; then
			{
				echo -e "Error: Invalid flag value: $flag "$1"\n";
				__section_help "$sectionFn" "$part";

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
				__section_help "$sectionFn" "$part";

				exit 2;
			} >&2;
		fi;
	done;

	mapfile -d '' CMD < <(
		(
			eval "$sectionFn" | head -n1;
			part="" eval "$sectionFn" | head -n1;
		) | {
			grep "^#!" | head -n1 | cut -b 3- || echo -n "$BASH";
		} | xargs printf '%s\0';
	);

	if declare -F "${CMD[0]:-}" > /dev/null; then
		"${CMD[@]}" <(
			part="" eval "$sectionFn";

			for flag in "${!setFlags[@]}"; do
				echo "declare $(tr -d '-' <<< "$flag")=${setFlags[$flag]@Q}";
			done;

			eval "$sectionFn";
		) "${args[@]}";

		exit $?;
	else
		exec -a "$part" "${CMD[@]}" <(
			part="" eval "$sectionFn";

			for flag in "${!setFlags[@]}"; do
				echo "declare $(tr -d '-' <<< "$flag")=${setFlags[$flag]@Q}";
			done;

			eval "$sectionFn";
		) "${args[@]}";
	fi;
}
