# Build Statistics

**Version:** 2026b
**Build date:** 2026-04-30T00:32:39Z

## Output Files

| File | Size | MD5 |
|------|------|-----|
| `combined-with-oceans.compress.topo.bin` | 17.0 MB | `41ac408c2fe46c95a88d0f6d1152a587` |
| `combined-with-oceans.topology.compress.topo.bin` | 5.7 MB | `2df2c7a684cb8bd9e24df197dd72a66f` |
| `combined-with-oceans.topology.preindex.bin` | 2.0 MB | `fb2921014ceb23bb9365a8a3b68a1ac8` |

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
output: bytes=17838872
reduction: bytes=67.73%
```

## Pipeline: `topology.compress.topo.bin` (topology-aware simplify + dedup + compress)

### `reducetzpb -topology=true`

```
mode: topology
epsilon: 0.001000
dataset_before: timezones=444 polygons=1320 holes=754 points=8075438 bytes=96920979
dataset_after:  timezones=444 polygons=1320 holes=754 points=1109455 bytes=13328696
dataset_reduction: points=86.26% bytes=86.25%
topology_rings: total=2074 no_fixed=1484 one_fixed=39 multi_fixed=547 fallback=205
topology_points: input=8071955 snapped_inserted=99 fallback_points=10658 fixed_vertices=186851
topology_segments: total=188333 shared=4160(2.21%) skipped_short=184024(97.71%) cache_hits=2072 cache_misses=2088 cache_hit_rate=49.81%
topology_segment_points: input=8260276 output=1285694 reduction=84.44%
topology_segment_length_buckets: le10=184401 le25=330 le50=340 le100=456 gt100=2806
```

### `deduplicatetzpb`

```
input:  timezones=444 polygons=1320 holes=754 points=1109455 bytes=13328703
output: shared_edges=2895 shared_points=463457 inline_segs=185039 edge_ref_segs=5570 bytes=10938501
reduction: bytes=17.93%
dedup_rate: 2.92% of segments reference shared edges
```

### `compresstopotzpb`

```
input:  bytes=10938501
output: bytes=5987766
reduction: bytes=45.26%
```

## Pipeline: `topology.preindex.bin` (topology-aware simplify + tile pre-index)

### `preindextzpb`

```
input:  timezones=444 bytes=13328703
params: idxZoom=13 aggZoom=3 maxZoomLevelToKeep=10 layerDrop=2
output: total_keys=88855 bytes=2052025
```

