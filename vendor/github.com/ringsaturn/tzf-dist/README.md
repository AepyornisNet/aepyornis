# tzf-dist

Binary data distribution for [tzf](https://github.com/ringsaturn/tzf) and
[tzf-rs](https://github.com/ringsaturn/tzf-rs).

If you have any questions or suggestions, please open an issue in the
[upstream tzf repository](https://github.com/ringsaturn/tzf).

## Files

| File                                              | Format                    | Size   | Description                                                              |
| ------------------------------------------------- | ------------------------- | ------ | ------------------------------------------------------------------------ |
| `combined-with-oceans.compress.topo.bin`          | `CompressedTopoTimezones` | ~17MB  | Full precision: shared-edge dedup + polyline compression                 |
| `combined-with-oceans.topology.compress.topo.bin` | `CompressedTopoTimezones` | ~5.4MB | Lite: topology-aware simplify + shared-edge dedup + polyline compression |
| `combined-with-oceans.reduce.preindex.bin`        | `PreindexTimezones`       | ~2MB   | Tile pre-index for FuzzyFinder                                           |

You can view the file in
[`tzf-bin-viewer`](https://ringsaturn.github.io/tzf-bin-viewer/) once you
download it.

The underlying data structures are defined in the [tzinfo.proto](https://github.com/ringsaturn/tzf/blob/main/pb/tzf/v1/tzinfo.proto).

## Branch structure

- **`main`** — source code, workflows, and `embed.go` template; never tagged
- **`data`** — latest generated binaries as an orphan commit (force-pushed on
  each update); the tip commit is what gets tagged for releases

Historical data is accessible via tags — each tag points to a single orphan
commit on `data` that contains the binary files for that
timezone-boundary-builder version.

## Usage (Go module)

Import a tagged version (tags point to `data` branch commits containing the
embedded files):

```go
import tzfdist "github.com/ringsaturn/tzf-dist"

// tzfdist.CompressTopoData        — full precision CompressedTopoTimezones
// tzfdist.TopologyCompressTopoData — lite CompressedTopoTimezones
// tzfdist.PreindexData             — tile pre-index
```

## Usage (Rust crate)

```toml
[dependencies]
tzf-dist = "..."
```

If you need full data precision, use the `full` feature flag with git based
dependency(full data is not available on crates.io due to size constraints):

```toml
[dependencies]
tzf-dist = { git = "https://github.com/ringsaturn/tzf-dist", tag = "...", features = ["full"], default-features = false}
```

## Releases

Binary files are attached to each GitHub Release as assets, built from the
corresponding
[timezone-boundary-builder](https://github.com/evansiroky/timezone-boundary-builder)
release.

Tags are created manually from the `data` branch tip after automated data
generation. Creating a GitHub Release from a tag triggers asset upload via the
release workflow.

## License

Code is licensed under the MIT License. See [LICENSE](LICENSE) for details.

Data is licensed under ODbL. See [DATA_LICENSE](DATA_LICENSE) for details. Same
with the
[timezone-boundary-builder](https://github.com/evansiroky/timezone-boundary-builder).
