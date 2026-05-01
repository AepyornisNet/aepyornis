package tzfdist

import _ "embed"

//go:embed combined-with-oceans.compress.topo.bin
var CompressTopoData []byte

//go:embed combined-with-oceans.topology.compress.topo.bin
var TopologyCompressTopoData []byte

//go:embed combined-with-oceans.topology.preindex.bin
var PreindexData []byte
