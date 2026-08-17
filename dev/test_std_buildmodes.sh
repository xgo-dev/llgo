#!/bin/bash

set -euo pipefail

if [[ $# -eq 0 ]]; then
	echo "usage: $0 ./test/std/package [...]" >&2
	exit 2
fi

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_pkgs=("$@")
import_paths=()
stems=()
groups=()
max_group=0
for test_pkg in "${test_pkgs[@]}"; do
	read -r import_path package_dir < <(go list -tags=llgo -f '{{.ImportPath}} {{.Dir}}' "${test_pkg}")
	case "${import_path}" in
		github.com/xgo-dev/llgo/test/std/*) ;;
		*)
			echo "not a test/std package: ${test_pkg}" >&2
			exit 2
			;;
	esac
	stem="$(basename "${package_dir}").test"
	group=0
	for i in "${!stems[@]}"; do
		if [[ "${stems[$i]}" == "${stem}" ]]; then
			group=$((group + 1))
		fi
	done
	import_paths+=("${import_path}")
	stems+=("${stem}")
	groups+=("${group}")
	if (( group > max_group )); then
		max_group="${group}"
	fi
done

# Keep the unique package in its own compiler invocation. Its generic use of
# weak.Pointer can otherwise assign shared instantiations to libunique.test and
# leave another output with unresolved weak.Pointer methods.
for i in "${!import_paths[@]}"; do
	if [[ "${import_paths[$i]}" == "github.com/xgo-dev/llgo/test/std/unique" ]]; then
		max_group=$((max_group + 1))
		groups[$i]="${max_group}"
	fi
done

llgo_cmd="${LLGO:-llgo}"
if [[ "${llgo_cmd}" == */* && "${llgo_cmd}" != /* ]]; then
	llgo_cmd="$(cd "$(dirname "${llgo_cmd}")" && pwd)/$(basename "${llgo_cmd}")"
fi
runner_source="${root_dir}/dev/test_std_buildmodes/runner.c"

runtime_libs=(-lpthread -lm -lresolv)
if [[ "$(uname -s)" == Darwin ]]; then
	runtime_libs+=(-framework CoreFoundation -framework Security)
fi
for dependency in bdw-gc libuv libffi; do
  if pkg-config --exists "${dependency}"; then
    while IFS= read -r flag; do
      [[ -n "${flag}" ]] && runtime_libs+=("${flag}")
    done < <(pkg-config --libs "${dependency}" | xargs -n1)
  fi
done
if [[ "$(uname -s)" != Darwin ]] && pkg-config --exists libunwind; then
  while IFS= read -r flag; do
    [[ -n "${flag}" ]] && runtime_libs+=("${flag}")
  done < <(pkg-config --libs libunwind | xargs -n1)
fi

work_dir="$(mktemp -d "${root_dir}/.std-buildmodes.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT

for mode in c-shared c-archive; do
	for ((group = 0; group <= max_group; group++)); do
		group_imports=()
		for i in "${!import_paths[@]}"; do
			if [[ "${groups[$i]}" -eq "${group}" ]]; then
				group_imports+=("${import_paths[$i]}")
			fi
		done
		if [[ "${#group_imports[@]}" -eq 0 ]]; then
			continue
		fi
		echo "==> ${mode}: compile ${#group_imports[@]} test package(s)"
		(
			cd "${work_dir}"
			"${llgo_cmd}" test -c -buildmode="${mode}" "${group_imports[@]}"
		)

		for i in "${!import_paths[@]}"; do
			if [[ "${groups[$i]}" -ne "${group}" ]]; then
				continue
			fi
			import_path="${import_paths[$i]}"
			stem="${stems[$i]}"
			test_main_pkg="${import_path}.test"
			runner_base="${work_dir}/runner-${i}"
			echo "==> ${test_pkgs[$i]}: run ${mode}"
			clang -x c -c "${runner_source}" \
				"-DGO_TEST_PACKAGE=\"${test_main_pkg}\"" \
				-o "${runner_base}.o"

			if [[ "${mode}" == c-shared ]]; then
				if [[ "$(uname -s)" == Darwin ]]; then
					library="${work_dir}/lib${stem}.dylib"
				else
					library="${work_dir}/lib${stem}.so"
				fi
				clang++ "${runner_base}.o" -o "${runner_base}" \
					-L"${work_dir}" "-l${stem}" "${runtime_libs[@]}"
				LD_LIBRARY_PATH="${work_dir}${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}" \
					DYLD_LIBRARY_PATH="${work_dir}${DYLD_LIBRARY_PATH:+:${DYLD_LIBRARY_PATH}}" \
					"${runner_base}"
			else
				library="${work_dir}/lib${stem}.a"
				clang++ "${runner_base}.o" -o "${runner_base}" \
					"${library}" "${runtime_libs[@]}"
				"${runner_base}"
			fi

			if [[ ! -s "${library}" ]]; then
				echo "missing ${mode} library: ${library}" >&2
				exit 1
			fi
			for artifact in "${runner_base}" "${runner_base}.o" "${library}" "${work_dir}/lib${stem}.h"; do
				if [[ -e "${artifact}" ]]; then
					unlink "${artifact}"
				fi
			done
		done
	done
done
