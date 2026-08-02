#!/usr/bin/env python3
"""Export the MobileNetV4-Large frame embedding model to ONNX.

The server uses this model as a feature extractor only — the classifier head is
dropped (num_classes=0), so the graph emits a pooled embedding vector.

ImageNet normalization is baked into the graph. The Go side feeds pixels in
[0, 1] NCHW (internal/analysis/preprocessing.ImageToTensorNCHW) and must keep
doing so; the (x - mean) / std step happens inside the exported model.

Do not run this by hand — use the wrapper, which builds a virtualenv, installs
torch/timm/onnx into it, runs this script and tears the environment back down:

    ./server/install-model.sh

Running it directly works too, but only from the server/ directory and only with
torch, timm and onnx already importable.

Writes assets/models/mobilenetv4_conv_large.onnx plus a .json sidecar recording
provenance (checkpoint, input size, embedding dim, normalization, sha256), so the
committed artifact stays checkable without torch installed.
"""

import hashlib
import json
from datetime import datetime, timezone
from pathlib import Path

import onnx
import timm
import torch
from torch import nn

CHECKPOINT = "mobilenetv4_conv_large.e500_r256_in1k"
MODEL_NAME = "mobilenetv4_conv_large"
INPUT_SIZE = 256
# From the checkpoint's config.json (pretrained_cfg.mean / .std).
MEAN = [0.485, 0.456, 0.406]
STD = [0.229, 0.224, 0.225]
# Only for the legacy fallback exporter. The dynamo exporter picks its own opset;
# pinning it there forces an onnx version-converter pass that fails on this model,
# and ONNX Runtime 1.28 handles anything the exporter emits.
LEGACY_OPSET = 17
# timm sets head_hidden_size=1280 for every MobileNetV4 conv variant. Note the
# num_features=960 in the HF config.json is the pre-pool stage width, not this.
EXPECTED_DIM = 1280

OUT_DIR = Path("assets/models")


class NormalizedEmbedder(nn.Module):
    """Applies ImageNet normalization, then returns the pooled embedding."""

    def __init__(self, backbone: nn.Module):
        super().__init__()
        self.backbone = backbone
        self.register_buffer("mean", torch.tensor(MEAN).view(1, 3, 1, 1))
        self.register_buffer("std", torch.tensor(STD).view(1, 3, 1, 1))

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.backbone((x - self.mean) / self.std)


def export_model(model: nn.Module, dummy: torch.Tensor, onnx_path: Path) -> str:
    """Exports via the dynamo path, falling back to the TorchScript exporter.

    dynamo=True has been the default since torch 2.9 and takes dynamic_shapes;
    dynamic_axes only applies to the older dynamo=False path. Both are passed
    explicitly so this works the same on either side of that change.
    """
    try:
        torch.onnx.export(
            model,
            (dummy,),
            str(onnx_path),
            input_names=["input"],
            output_names=["output"],
            dynamic_shapes={"x": {0: torch.export.Dim("batch_size")}},
            dynamo=True,
            # Defaults to True on the dynamo path, which writes the weights to a
            # separate .onnx.data file. The Go loader is handed a single path and
            # the artifact is committed, so keep it self-contained.
            external_data=False,
        )
        return "dynamo"
    except Exception as err:  # noqa: BLE001 - any dynamo failure is worth retrying
        print(f"[export] dynamo export failed ({err}); retrying with the legacy exporter")

    torch.onnx.export(
        model,
        dummy,
        str(onnx_path),
        input_names=["input"],
        output_names=["output"],
        dynamic_axes={"input": {0: "batch_size"}, "output": {0: "batch_size"}},
        opset_version=LEGACY_OPSET,
        do_constant_folding=True,
        dynamo=False,
    )
    return "torchscript"


def verify_graph(onnx_path: Path) -> int:
    """Checks the graph against what the Go loader hard-codes.

    internal/analysis/detectors/onnx/model_config.go names the input and output
    nodes, and the sqlite-vec table is created with the embedding width, so a
    silent change in any of them corrupts the pipeline rather than failing it.
    """
    model = onnx.load(str(onnx_path))
    onnx.checker.check_model(model)

    external = sum(1 for t in model.graph.initializer if t.data_location == onnx.TensorProto.EXTERNAL)
    if external:
        raise SystemExit(
            f"{external} initializers point at external data — the .onnx must be "
            "self-contained, the Go loader is handed a single path"
        )

    names = [i.name for i in model.graph.input]
    if names != ["input"]:
        raise SystemExit(f"expected a single graph input named 'input', got {names}")

    names = [o.name for o in model.graph.output]
    if names != ["output"]:
        raise SystemExit(f"expected a single graph output named 'output', got {names}")

    dims = model.graph.output[0].type.tensor_type.shape.dim
    if len(dims) != 2 or dims[1].dim_value != EXPECTED_DIM:
        shape = [d.dim_param or d.dim_value for d in dims]
        raise SystemExit(f"expected output shape [batch, {EXPECTED_DIM}], got {shape}")

    if not dims[0].dim_param:
        print(f"[export] note: batch axis is fixed at {dims[0].dim_value} (the server only ever infers one frame)")

    opset = next((i.version for i in model.opset_import if i.domain in ("", "ai.onnx")), 0)
    print(f"[export] graph verified: opset {opset}, input/output names, [batch, {EXPECTED_DIM}] output")
    return opset


def verify_numerics(model: nn.Module, onnx_path: Path) -> None:
    """Runs both models on the same input and compares the embeddings.

    This is what actually proves the export is faithful — that the baked-in
    normalization and the NCHW layout survived. A graph that loads but computes
    something slightly different would silently degrade every similarity result
    instead of failing.
    """
    import numpy as np
    import onnxruntime as ort

    generator = torch.Generator().manual_seed(0)
    sample = torch.rand(1, 3, INPUT_SIZE, INPUT_SIZE, generator=generator)

    with torch.no_grad():
        expected = model(sample).numpy()

    session = ort.InferenceSession(str(onnx_path), providers=["CPUExecutionProvider"])
    actual = session.run(["output"], {"input": sample.numpy()})[0]

    if actual.shape != expected.shape:
        raise SystemExit(f"ONNX output shape {actual.shape} != torch {expected.shape}")

    max_diff = float(np.abs(actual - expected).max())
    cosine = float(
        np.dot(actual.ravel(), expected.ravel())
        / (np.linalg.norm(actual) * np.linalg.norm(expected))
    )
    print(f"[export] numerics: max|diff| = {max_diff:.2e}, cosine = {cosine:.8f}")
    if max_diff > 1e-3 or cosine < 0.9999:
        raise SystemExit("exported model does not match the PyTorch model")


def main() -> None:
    if not OUT_DIR.is_dir():
        raise SystemExit(f"{OUT_DIR} not found — run this from the server/ directory")

    print(f"[export] loading {CHECKPOINT}")
    backbone = timm.create_model(CHECKPOINT, pretrained=True, num_classes=0)
    model = NormalizedEmbedder(backbone).eval()

    dummy = torch.zeros(1, 3, INPUT_SIZE, INPUT_SIZE)
    with torch.no_grad():
        out = model(dummy)

    print(f"[export] embedding shape: {tuple(out.shape)}")
    if tuple(out.shape) != (1, EXPECTED_DIM):
        raise SystemExit(
            f"unexpected embedding shape {tuple(out.shape)}, expected (1, {EXPECTED_DIM}). "
            "The sqlite-vec frame_vectors table is created with this dimension — "
            "changing it is a schema migration, not just a data reset. Aborting."
        )

    onnx_path = OUT_DIR / f"{MODEL_NAME}.onnx"
    # A previous run may have left weights in a sidecar file; a stale one would be
    # picked up over the weights embedded in the new export.
    external_data = onnx_path.with_suffix(".onnx.data")
    if external_data.exists():
        print(f"[export] removing stale {external_data}")
        external_data.unlink()

    print(f"[export] writing {onnx_path}")
    exporter = export_model(model, dummy, onnx_path)
    opset = verify_graph(onnx_path)
    verify_numerics(model, onnx_path)

    digest = hashlib.sha256(onnx_path.read_bytes()).hexdigest()
    sidecar = {
        "model_name": MODEL_NAME,
        "checkpoint": f"timm/{CHECKPOINT}",
        "input_size": INPUT_SIZE,
        "input_layout": "NCHW",
        "input_range": "[0,1] — ImageNet normalization is applied inside the graph",
        "embedding_dim": EXPECTED_DIM,
        "normalization": {"mean": MEAN, "std": STD},
        "opset": opset,
        "exporter": exporter,
        "torch_version": torch.__version__,
        "timm_version": timm.__version__,
        "sha256": digest,
        "exported_at": datetime.now(timezone.utc).isoformat(timespec="seconds"),
    }
    sidecar_path = OUT_DIR / f"{MODEL_NAME}.json"
    sidecar_path.write_text(json.dumps(sidecar, indent=2) + "\n")

    size_mb = onnx_path.stat().st_size / 1024 / 1024
    print(f"[export] done: {onnx_path} ({size_mb:.1f} MB), sha256 {digest[:16]}…")
    print(f"[export] sidecar: {sidecar_path}")


if __name__ == "__main__":
    main()
