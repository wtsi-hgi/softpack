#!runContainer
# Command for performing SoftPack APT repo related tasks.

set -euo pipefail;

declare BASE=/repo/pool/main/binary-amd64/;
declare maintainer="hgi <hgi@sanger.ac.uk>";

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

	dpkg-deb -e "$file" "$tmpDir";

	while [ $# -gt 0 ]; do
		declare key="$1";
		declare value="$2";

		shift 2;

		if [ -z "$value" ]; then
			sed -i "/^$key: /d" "$tmpDir/control";
		elif grep -q "^$key: " "$tmpDir/control"; then
			sed -i "s/^$key: .*/$key: $value/" "$tmpDir/control";
		else
			echo "$key: $value" >> "$tmpDir/control";
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

	declare exes=( $(dpkg -c "$file" | grep "^[^d][^ ]*x" | grep " ./\(usr/local/bin/\|usr/bin/\|bin/\)" | sed -e 's@.*/\([^ ]*\)\( -> .*\)\?$@\1@' | sort | uniq) );

	if [ "$(dpkg-deb -f "$file" Package)" = "python3" ]; then
		declare version="$(dpkg-deb -f "$file" Version)";

		exes+=( "python$(cut -d'.' -f1 <<< "$version")" "python$(cut -d'.' -f1-2 <<< "$version")" );
	fi;

	if [ ${#exes[@]} -eq 0 ]; then
		return;
	fi;

	setMetadata "$file" "XB-Executables" "$(
		for exe in "${exes[@]}"; do
			echo -n "$exe, ";
		done | sed -e 's/, $//';
	)";
}

aptOpts() {
	declare tmpDir="$1";
	declare downloadDir="${2:-$tmpDir}";

	mkdir -p "$tmpDir/etc/apt/preferences.d" "$tmpDir/etc/apt/sources.list.d" "$tmpDir/var/lib/apt/lists/partial" "$tmpDir/var/cache/apt/archives/partial" "$tmpDir/var/lib/dpkg" "/$tmpDir/debs";

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
deb [trusted=yes] https://ppa.launchpadcontent.net/deadsnakes/ppa/ubuntu/ resolute main
deb [trusted=yes] file:///repo resolute main
HEREDOC

	if [ -v aptDir ]; then
		echo "deb [trusted=yes] file://$aptDir resolute main" >> "$tmpDir/etc/apt/sources.list.d/apt.list";
	fi;

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
}

downloadDeps() {
	declare downloadDir="$1";
	shift;

	declare tmpDir="$(mktemp -d)";
	declare cacheDir="$(mktemp -d)/";
	declare aptDir="$(mktemp -d)/";

	aptConf "$cacheDir" > "$cacheDir/apt.conf";

	(
		cd "$aptDir";
		mkdir -p "dists/resolute/main/binary-amd64";
		mkdir -p "pool/main/binary-amd64";
		cp "$downloadDir"/* "./pool/main/binary-amd64/";
		cat > release.conf <<HEREDOC
APT::FTPArchive::Release::Origin "HGI";
APT::FTPArchive::Release::Label "SoftPack";
APT::FTPArchive::Release::Suite "resolute";
APT::FTPArchive::Release::Codename "resolute";
APT::FTPArchive::Release::Architectures "amd64";
APT::FTPArchive::Release::Components "main";
APT::FTPArchive::Release::Description "SoftPack APT Repository";
HEREDOC

		apt-ftparchive generate "$cacheDir"/apt.conf;
		apt-ftparchive -c release.conf release dists/resolute > dists/resolute/Release;
	)

	declare -a packages=();

	while read pkg && read ver && read; do
		packages+=( "$pkg=$ver" );
	done < <(grep-dctrl -s Package,Version . -n "$aptDir/dists/resolute/main/binary-amd64/Packages");

	declare -a OPTS;
	aptOpts "$tmpDir" "$downloadDir";

	apt "${OPTS[@]}" update;
	apt "${OPTS[@]}" install --download-only -y --allow-downgrades --allow-change-held-packages --allow-remove-essential --no-strict-pinning --reinstall "${packages[@]}";
}

patchUCF() {
	declare ucfDeb="$1";

	declare tmpDir="$(mktemp -d)";

	dpkg-deb -R "$ucfDeb" "$tmpDir";
	sed -i '2i id() { echo 0; }' "$tmpDir/usr/bin/ucf";
	sed -i '2i id() { echo 0; }' "$tmpDir/usr/bin/ucfr";
	dpkg-deb --root-owner-group -b "$tmpDir" "$ucfDeb";
}

patchSystemd() {
	declare deb="$1";

	declare tmpDir="$(mktemp -d)";

	dpkg-deb -R "$deb" "$tmpDir";
	echo "#!/bin/bash" > "$tmpDir/usr/bin/systemd-sysusers";
	dpkg-deb --root-owner-group -b "$tmpDir" "$deb";
}

addRSymlinkIfOpt() {
	declare deb="$1";

	declare tmpDir="$(mktemp -d)";

	dpkg-deb -R "$deb" "$tmpDir";

	if [ ! -d "$tmpDir/opt" ]; then
		return;
	fi;

	mkdir -p "$tmpDir/usr/local/bin";
	ln -s "/opt/R/$(basename "$tmpDir/opt/R/"*)/bin/R" "$tmpDir/usr/local/bin/R";
	ln -s "/opt/R/$(basename "$tmpDir/opt/R/"*)/bin/Rscript" "$tmpDir/usr/local/bin/Rscript";

	dpkg-deb --root-owner-group -b "$tmpDir" "$deb";
}

fixR() {
	declare file="$1";
	declare alias="$2";

	setMetadata "$file" \
		"XB-Alias" "$alias" \
		"XB-Softpack" "true";
}

addPythonSymlinkIfMissing() {
	declare deb="$1";

	declare tmpDir="$(mktemp -d)";

	dpkg-deb -R "$deb" "$tmpDir";

	declare -a exes=( "$tmpDir/usr/bin/python"* );

	if [ -e "$tmpDir/usr/bin/python" -o ! -v exes ]; then
		return;
	fi;

	mkdir -p "$tmpDir/usr/bin";
	ln -s "/usr/bin/$(basename "$exes")" "$tmpDir/usr/bin/python";

	dpkg-deb --root-owner-group -b "$tmpDir" "$deb";
}

fixPythonVersioning() {
	declare file="$1";

	addPythonSymlinkIfMissing "$file";

	if [ -z "$(grep "^python3.[0-9]\+_" <<< "$file")" ]; then
		return 0;
	fi;

	setMetadata "$file" \
		"Package" "python3" \
		"XB-Alias" "python" \
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

		if [ "${file:0:4}" = "ucf_" ]; then
			patchUCF "$file";
		elif [ "${file:0:8}" = "systemd_" ]; then
			patchSystemd "$file";
		elif [ "${file:0:8}" = "python3." ]; then
			fixPythonVersioning "$file";
		elif [ "${file:0:12}" = "r-base-core_" ]; then
			addRSymlinkIfOpt "$file";
			fixR "$file" "r";
		elif [ "${file:0:7}" = "r-cran-" -o "${file:0:7}" = "r-bioc-" -o "${file:0:7}" = "r-misc-" ]; then
			fixR "$file" "$(sed -e 's/^r-\(bioc\|cran\|misc\)-\([^_]*\)_.*/r-\2/' -e 's/\./-/g' <<< "$file")";
		fi;

		setExecutables "$file";

		mv -v "$file" "$BASE/$l/";
	done;

	cd - &> /dev/null;

	shopt -u nullglob;
}

# vim: set filetype=bash :
