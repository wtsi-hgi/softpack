#!runContainer
# Command for performing SoftPack APT repo related tasks.

set -euo pipefail;

declare BASE=/repo/pool/main/binary-amd64/;

aptConf() {
	cat <<HEREDOC
Dir {
  ArchiveDir ".";
  CacheDir "$1";
};

Default {
  Packages::Compress ". gzip bzip2 lzma xz";
};

BinDirectory "pool/main/binary-amd64" {
  Packages "dists/resolute/main/binary-amd64/Packages";
  BinCacheDB "apt.cache";
};
HEREDOC
}

setMetadata() {
	declare file="$1";

	shift;

	declare tmpDir="$(mktemp -d)";
	trap "rm -rf ${tmpDir@Q}" EXIT;

	dpkg-deb -e "$file" "$tmpDir";

	while [ $# -gt 0 ]; do
		declare key="$1";
		declare value="$2";

		shift 2;

		if grep -q "^$key: " "$tmpDir/control"; then
			sed -i "s/^$key: .*/$key: $value/" "$tmpDir/control";
		else
			echo "$key: $value"  >> "$tmpDir/control";
		fi;
	done;

	packageControl "$tmpDir" "$file";
}

packageControl() {
	declare dir="$1";
	declare deb="$2";

	tar --owner=0 --group=0 -cf "$dir/control.tar" "$tmpDir"/* --transform 's,.*/,,';

	case "$(ar t "$deb" | grep "^control.tar" | cut -d'.' -f3)" in
	"gz")
		gzip -9 "$dir/control.tar";;
	"xz")
		xz -9 "$dir/control.tar";;
	"zst")
		zstd --rm -19 "$dir/control.tar";;
	esac;

	ar r "$deb" "$dir/control.tar."*;
}

setExecutables() {
	declare file="$1";

	declare exes=( $(dpkg -c "$file" | grep "^[^d][^ ]*x" | grep " ./(usr/local/bin/\|usr/bin/\|bin/)" | sed -e 's@.*/\([^ ]*\)\( -> .*\)\?$@\1@' | sort | uniq) );

	if [ ${#exes[@]} -eq 0 ]; then
		return;
	fi;

	setMetadata "$file" "XB-Executables" "$(
		for exe in "${exes[@]}"; do
			echo -n "$exe, ";
		done | sed -e 's/, $//';
	)";
}

downloadDeps() {
	declare downloadDir="$1";
	shift;

	declare tmpDir="$(mktemp -d)";
	trap "rm -rf ${tmpDir@Q}" EXIT;

	mkdir -p "$tmpDir/etc/apt/preferences.d" "$tmpDir/etc/apt/sources.list.d" "$tmpDir/var/lib/apt/lists/partial" "$tmpDir/var/cache/apt/archives/partial" "$tmpDir/var/lib/dpkg" "/$tmpDir/debs";
	#cp /var/lib/dpkg/status "$tmpDir/var/lib/dpkg/status";
	cp "/etc/apt/sources.list.d/ubuntu.sources" "$tmpDir/etc/apt/sources.list.d/";

	cat <<HEREDOC > "$tmpDir/etc/apt/preferences.d/99-local-priority"
Package: *
Pin: release l=SoftPack
Pin-Priority: 990

Package: *
Pin: release n=resolute
Pin-Priority: 500
HEREDOC

	cat <<HEREDOC > "$tmpDir/etc/apt/sources.list.d/apt.list"
deb [trusted=yes] http://r2u.stat.illinois.edu/ubuntu resolute main
deb [trusted=yes] https://ppa.launchpadcontent.net/marutter/rrutter4.0/ubuntu/ resolute main
deb [trusted=yes] file:///repo resolute main
HEREDOC

	OPTS=(
		-o Dir::Etc="$tmpDir/etc/apt/"
		-o Dir::Etc::sourcelist="sources.list"
		-o Dir::Etc::sourceparts="sources.list.d"
		-o Dir::State="$tmpDir/var/lib/apt/"
		-o Dir::State::status="$tmpDir/var/lib/dpkg/status"
		-o Dir::Cache="$tmpDir/var/cache/apt/"
		-o Dir::Cache::archives="$downloadDir"
		-o Dir::Cache::archives::partial="$tmpDir/var/cache/apt/archives/partial"
		-o Acquire::AllowInsecureRepositories=true
		-o Acquire::AllowDowngradeToInsecureRepositories=true
		-o APT::Get::Update::SourceListWarnings=false
	);

	apt "${OPTS[@]}" update;

	for deb; do
		apt "${OPTS[@]}" install --download-only -y --allow-downgrades --reinstall "$deb";
	done;
}

patchUCF() {
	declare ucfDeb="$1";

	declare tmpDir="$(mktemp -d)";
	trap "rm -rf ${tmpDir@Q}" EXIT;

	dpkg-deb -R "$ucfDeb" "$tmpDir";
	sed -i '2i id() { echo 0; }' "$tmpDir/usr/bin/ucf";
	sed -i '2i id() { echo 0; }' "$tmpDir/usr/bin/ucfr";
	dpkg-deb --root-owner-group -b "$tmpDir" "$ucfDeb";
}

patchSystemd() {
	declare deb="$1";

	declare tmpDir="$(mktemp -d)";
	trap "rm -rf ${tmpDir@Q}" EXIT;

	dpkg-deb -R "$deb" "$tmpDir";
	echo "#!/bin/bash" > "$tmpDir/usr/bin/systemd-sysusers";
	dpkg-deb --root-owner-group -b "$tmpDir" "$deb";
}

fixR() {
	declare file="$1";
	declare alias="$2";

	declare provides="$(dpkg-deb -f "$file" Provides)";

	if [ -n "$provides" ]; then
		provides=", $provides";
	fi;

	setMetadata "$file" \
		"Provides" "$alias (=$(dpkg-deb -f "$file" Version))$provides" \
		"XB-Alias" "$alias" \
		"XB-Softpack" "true";
}

installDebs() {
	declare debDir="$1";

	cd "$debDir";

	shopt -s nullglob;

	for file in *.deb; do
		declare l="$(basename "$file" | sed -e 's/^\(.\).*/\1/')";
		mkdir -p "$BASE/$l";

		if [ -f "$BASE/$l/$(basename "$file")" ]; then
		       continue;
		fi;

		setExecutables "$file";

		if [ "${file:0:4}" = "ucf_" ]; then
			patchUCF "$file";
		elif [ "${file:0:8}" = "systemd_" ]; then
			patchSystemd "$file";
		elif [ "${file:0:12}" = "r-base-core_" ]; then
			fixR "$file" "r";
		elif [ "${file:0:7}" = "r-cran-" -o "${file:0:7}" = "r-bioc-" ]; then
			fixR "$file" "$(sed -e 's/^r-\(bioc\|cran\)-\([^_]*\)_.*/r-\2/' -e 's/\./-/g' <<< "$file")";
		fi;

		mv -v "$file" "$BASE/$l/";
	done;

	cd - &> /dev/null;

	shopt -u nullglob;
}

# vim: set filetype=sh :
