/* Demo-local glue for the SPI probe.
 *
 * The lpcore demo needs a demo_glue.c for LP-program accessors and sleep
 * bindings; the SPI surface needs no C helpers beyond its package glue
 * (spi_glue.c, synced from pkg/esp by host/make-artifact.sh), and the shared
 * app_main.c scaffold already owns the memory-ledger hook. This file stays
 * in the component's SRCS list only to keep the family pattern - an
 * otherwise empty translation unit.
 */
typedef int llgo_spi_demo_no_extra_glue;
