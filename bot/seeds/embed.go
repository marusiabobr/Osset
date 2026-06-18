package seeds

import "embed"

//go:embed topics.json lexicon_stub.json levels/*.json
var Files embed.FS
