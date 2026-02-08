# depstat with containerd

This guide shows how to run `depstat` against containerd's dual-module repository and reproduce common dependency analyses.

## Prerequisites

- A containerd checkout (main branch)
- `depstat` installed (`go install github.com/kubernetes-sigs/depstat@latest`)
- `jq` (for JSON inspection)

## Paths

```bash
export CONTAINERD_DIR=<your-containerd-checkout>   # e.g. ~/go/src/github.com/containerd/containerd
```

## Main modules

```bash
CONTAINERD_MODS="github.com/containerd/containerd/v2,github.com/containerd/containerd/api"
```

## Stats

```bash
depstat stats -m "$CONTAINERD_MODS" --json --dir "$CONTAINERD_DIR" | jq
```

## List (split test-only)

```bash
depstat list -m "$CONTAINERD_MODS" --split-test-only --dir "$CONTAINERD_DIR"
```

## Graph

```bash
depstat graph -m "$CONTAINERD_MODS" --dot --show-edge-types --dir "$CONTAINERD_DIR" | dot -Tsvg -o containerd-graph.svg
```

Topology JSON:

```bash
depstat graph -m "$CONTAINERD_MODS" --json --dir "$CONTAINERD_DIR" | jq
```

## Cycles

```bash
depstat cycles -m "$CONTAINERD_MODS" --summary --dir "$CONTAINERD_DIR"
```

## Archived dependencies

```bash
depstat archived -m "$CONTAINERD_MODS" --github-token-path /tmp/gh-token --json --dir "$CONTAINERD_DIR" | jq
```

## Diff between refs

```bash
depstat diff v2.1.0 v2.2.1 -m "$CONTAINERD_MODS" --json --dir "$CONTAINERD_DIR" | jq
```

## Why (trace a dependency)

```bash
depstat why github.com/Microsoft/hcsshim -m "$CONTAINERD_MODS" --dot --dir "$CONTAINERD_DIR" | dot -Tsvg -o containerd-why-hcsshim.svg
```
