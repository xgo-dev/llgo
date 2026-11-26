# Demo-local post-sync hook, sourced by _demo/common/make-artifact.sh right
# after the common scaffold and glue sync (the pipeline's working directory
# is the demo root and $MAIN points at the main component).
#
# The pipeline writes the shared 4 MB partition table into every project;
# this firmware needs the esp-sr model partition on top, which only exists
# in this demo - so the table is replaced here, after the sync. The layout
# is the esp-sr S31 CI one (16 MB flash: factory 8000K, model 7900K) with
# the standard NVS/PHY sectors ahead of the app.
cat > "$MAIN/../partitions.csv" <<'CSV'
# Name,   Type, SubType, Offset,  Size, Flags
nvs,      data, nvs,     0x9000,  0x6000,
phy_init, data, phy,     0xf000,  0x1000,
factory,  app,  factory, 0x10000, 8000K,
model,    data, spiffs,  ,        7900K,
CSV
