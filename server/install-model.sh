#!/usr/bin/env bash
#
# Exports the MobileNetV4-Large frame embedding model to
# assets/models/mobilenetv4_conv_large.onnx, which the server requires at startup.
#
# Creates a throwaway virtualenv, installs torch/timm/onnx into it, runs
# scripts/export_mobilenetv4_onnx.py, then deactivates. Nothing is installed
# system-wide or into ~/.local.
#
#   ./install-model.sh            # export if the model is missing
#   ./install-model.sh --force    # re-export over an existing model
#   ./install-model.sh --clean    # delete the virtualenv when finished
#
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="${ROOT_DIR}/.venv-export"
MODEL_PATH="${ROOT_DIR}/assets/models/mobilenetv4_conv_large.onnx"
EXPORT_SCRIPT="${ROOT_DIR}/scripts/export_mobilenetv4_onnx.py"

FORCE=0
CLEAN=0
for arg in "$@"; do
  case "${arg}" in
    --force) FORCE=1 ;;
    --clean) CLEAN=1 ;;
    -h|--help) sed -n '2,12p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "[install-model] Unknown option: ${arg}" >&2; exit 1 ;;
  esac
done

if [[ -f "${MODEL_PATH}" && "${FORCE}" -eq 0 ]]; then
  echo "[install-model] Already exported: ${MODEL_PATH}"
  echo "[install-model] Re-export with: $0 --force"
  exit 0
fi

if [[ ! -f "${EXPORT_SCRIPT}" ]]; then
  echo "[install-model] Missing ${EXPORT_SCRIPT}" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "[install-model] python3 not found. Install Python 3.10+ and re-run." >&2
  exit 1
fi

CPU_INDEX="https://download.pytorch.org/whl/cpu"

create_venv() {
  echo "[install-model] Creating virtualenv at ${VENV_DIR} ..."
  # python3-venv ships separately on Debian/Ubuntu, a common first failure.
  if ! python3 -m venv "${VENV_DIR}"; then
    # venv creates the directory before it fails, so clear the half-built tree.
    rm -rf "${VENV_DIR}"
    PY_MINOR="$(python3 -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')"
    cat >&2 <<EOF

[install-model] Could not create the virtualenv (see the error above).
[install-model] On Debian/Ubuntu the venv module is a separate package:

    sudo apt install python${PY_MINOR}-venv

[install-model] Alternatively, export in a container without touching this machine:

    docker run --rm -v "\$PWD/server:/w" -w /w python:3.12-slim bash -c \\
      "pip install -q --index-url ${CPU_INDEX} torch torchvision && \\
       pip install -q timm onnx && python scripts/export_mobilenetv4_onnx.py"

EOF
    exit 1
  fi
}

install_deps() {
  local py="${VENV_DIR}/bin/python"
  echo "[install-model] Installing dependencies (a few hundred MB, first run only) ..."
  "${py}" -m pip install --quiet --upgrade pip

  # torch and torchvision must come from the SAME index in ONE resolve. torchvision
  # ships a compiled extension linked against a specific torch build: installing it
  # from PyPI next to a +cpu torch yields matching version numbers but incompatible
  # binaries, and every torchvision op fails to register at import time.
  # The CPU index also avoids ~2.5 GB of CUDA libraries this export never uses.
  "${py}" -m pip install --quiet --index-url "${CPU_INDEX}" torch torchvision

  # timm depends on torchvision, so pin the pair to exactly what was just installed
  # (+cpu local versions included) — otherwise pip may swap in the PyPI build.
  local constraints="${VENV_DIR}/export-constraints.txt"
  "${py}" - "${constraints}" <<'PY'
import sys, torch, torchvision
with open(sys.argv[1], "w") as fh:
    fh.write(f"torch=={torch.__version__}\ntorchvision=={torchvision.__version__}\n")
PY
  # onnxscript backs the torch.export-based exporter, the default since torch 2.9.
  # Without it torch.onnx.export falls back to the deprecated TorchScript path.
  "${py}" -m pip install --quiet --constraint "${constraints}" timm onnx onnxscript onnxruntime
}

# Catches the torch/torchvision build mismatch above, plus a half-finished install.
env_healthy() {
  [[ -x "${VENV_DIR}/bin/python" ]] || return 1
  "${VENV_DIR}/bin/python" - >/dev/null 2>&1 <<'PY'
import torch, torchvision, timm, onnx, onnxscript, onnxruntime  # noqa: F401
torch.ops.torchvision.nms  # unresolved unless the two builds match
PY
}

if [[ ! -x "${VENV_DIR}/bin/python" ]]; then
  create_venv
  install_deps
fi

if ! env_healthy; then
  echo "[install-model] Existing virtualenv is inconsistent — rebuilding it from scratch."
  rm -rf "${VENV_DIR}"
  create_venv
  install_deps
  if ! env_healthy; then
    echo "[install-model] The rebuilt virtualenv still fails its import check. Re-run with --clean and open an issue." >&2
    exit 1
  fi
fi

# shellcheck disable=SC1091
source "${VENV_DIR}/bin/activate"
# Deactivate on any exit path, including failure.
trap 'deactivate 2>/dev/null || true' EXIT

echo "[install-model] Exporting model ..."
cd "${ROOT_DIR}"
python "${EXPORT_SCRIPT}"

deactivate
trap - EXIT

if [[ "${CLEAN}" -eq 1 ]]; then
  echo "[install-model] Removing ${VENV_DIR}"
  rm -rf "${VENV_DIR}"
else
  echo "[install-model] Virtualenv kept at ${VENV_DIR} ($(du -sh "${VENV_DIR}" | cut -f1)) — remove it with: rm -rf ${VENV_DIR}"
fi

echo "[install-model] Done: ${MODEL_PATH}"
