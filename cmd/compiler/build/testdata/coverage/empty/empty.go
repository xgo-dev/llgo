package empty

// A declaration-only source produces an empty standalone coverage metadata
// file. It must not make another package's -coverpkg report fail with EOF.
type Item struct{}
