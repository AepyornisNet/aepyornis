# Build Statistics

**Version:** 2026b-fix1
**Build date:** 2026-05-03T03:08:47Z

## Output Files

| File | Size | MD5 |
|------|------|-----|
| `combined-with-oceans.compress.topo.bin` | 17.8 MB | `e503b7152e427f862466903b968f5d96` |
| `combined-with-oceans.topology.compress.topo.bin` | 6.5 MB | `6e35a54bf71ce12c19175f4abfa58d37` |
| `combined-with-oceans.topology.preindex.bin` | 2.0 MB | `c185dd72d845e25474b287d7502ce040` |

## Pipeline: `compress.topo.bin` (dedup + compress on full precision)

### `deduplicatetzpb`

```
input:  timezones=444 polygons=1320 holes=754 points=8075438 bytes=96920986
output: shared_edges=3523 shared_points=4124573 inline_segs=185055 edge_ref_segs=5600 bytes=55273383
reduction: bytes=42.97%
dedup_rate: 2.94% of segments reference shared edges
```

### `compresstopotzpb`

```
input:  bytes=55273383
output: bytes=18712579
reduction: bytes=66.15%
```

## Pipeline: `topology.compress.topo.bin` (topology-aware simplify + dedup + compress)

### `reducetzpb -topology=true`

```
mode: topology
epsilon: 0.001000
dataset_before: timezones=444 polygons=1320 holes=754 points=8075438 bytes=96920979
dataset_after:  timezones=444 polygons=1320 holes=754 points=1109449 bytes=13328624
dataset_reduction: points=86.26% bytes=86.25%
topology_rings: total=2074 no_fixed=1484 one_fixed=39 multi_fixed=547 fallback=205
topology_points: input=8071955 snapped_inserted=99 fallback_points=10658 fixed_vertices=186851
topology_segments: total=188333 shared=4160(2.21%) skipped_short=184024(97.71%) cache_hits=2072 cache_misses=2088 cache_hit_rate=49.81%
topology_segment_points: input=8260276 output=1285688 reduction=84.44%
topology_segment_length_buckets: le10=184401 le25=330 le50=340 le100=456 gt100=2806
```

### `deduplicatetzpb`

```
input:  timezones=444 polygons=1320 holes=754 points=1109449 bytes=13328631
output: shared_edges=2895 shared_points=463454 inline_segs=185039 edge_ref_segs=5570 bytes=10938465
reduction: bytes=17.93%
dedup_rate: 2.92% of segments reference shared edges
```

### `compresstopotzpb`

```
input:  bytes=10938465
output: bytes=6859548
reduction: bytes=37.29%
```

## Pipeline: `topology.preindex.bin` (topology-aware simplify + tile pre-index)

### `preindextzpb`

```
input:  timezones=444 bytes=13328631
params: idxZoom=13 aggZoom=3 maxZoomLevelToKeep=10 layerDrop=2
output: total_keys=88855 bytes=2052025
```

