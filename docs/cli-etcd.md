# depstat with etcd

This guide shows how to run `depstat` against the etcd repository. It focuses on multi-module usage, excluding tooling, and common analysis commands (stats, list, graph, cycles, archived, diff, why).

## Prerequisites

- An etcd checkout (main branch)
- `depstat` installed (`go install github.com/kubernetes-sigs/depstat@latest`)
- `jq` (for JSON inspection)

## Paths

```bash
export ETCD_DIR=<your-etcd-checkout>   # e.g. ~/go/src/github.com/etcd-io/etcd
```

## Main modules (etcd is multi-module)

Build the module list once and reuse it:

```bash
ETCD_MODS="go.etcd.io/etcd/v3,go.etcd.io/etcd/api/v3,go.etcd.io/etcd/client/pkg/v3,\
go.etcd.io/etcd/client/v3,go.etcd.io/etcd/etcdctl/v3,go.etcd.io/etcd/etcdutl/v3,\
go.etcd.io/etcd/pkg/v3,go.etcd.io/etcd/server/v3,go.etcd.io/etcd/tests/v3,\
go.etcd.io/etcd/tools/v3,go.etcd.io/etcd/tools/rw-heatmaps/v3,go.etcd.io/etcd/tools/testgrid-analysis/v3"
```

> Tip: If you want core-only analysis (excluding tooling), keep a second list without the tools modules and use `--exclude-modules "go.etcd.io/etcd/tools/*"`.

## Stats

```bash
depstat stats -m "$ETCD_MODS" --json --dir "$ETCD_DIR" | jq
```

Core-only stats:

```bash
depstat stats -m "$ETCD_MODS" --exclude-modules "go.etcd.io/etcd/tools/*" --json --dir "$ETCD_DIR" | jq
```

## List (split test-only)

```bash
depstat list -m "$ETCD_MODS" --split-test-only --dir "$ETCD_DIR"
```

## Graph

```bash
depstat graph -m "$ETCD_MODS" --dot --show-edge-types --dir "$ETCD_DIR" | dot -Tsvg -o etcd-graph.svg
```

Topology JSON:

```bash
depstat graph -m "$ETCD_MODS" --json --dir "$ETCD_DIR" | jq
```

## Cycles

```bash
depstat cycles -m "$ETCD_MODS" --summary --dir "$ETCD_DIR"
```

Limit cycle length:

```bash
depstat cycles -m "$ETCD_MODS" --max-length 2 --summary --json --dir "$ETCD_DIR" | jq
```

## Archived dependencies

```bash
depstat archived -m "$ETCD_MODS" --github-token-path /tmp/gh-token --json --dir "$ETCD_DIR" | jq
```

## Diff between refs

```bash
depstat diff v3.5.0 v3.6.0 -m "$ETCD_MODS" --json --dir "$ETCD_DIR" | jq
```

## Why (trace a dependency)

```bash
depstat why google.golang.org/grpc -m "$ETCD_MODS" --dot --dir "$ETCD_DIR" | dot -Tsvg -o etcd-why-grpc.svg
```
